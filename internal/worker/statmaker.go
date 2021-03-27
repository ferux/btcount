package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ferux/btcount/internal/btcount"

	"go.uber.org/zap"
)

type StatMakerWorkerConfig struct {
	TStore btcount.TransactionStorage
	HStore btcount.HistoryStatStorage

	DB         btcount.Database
	RetryDelay time.Duration
}

// RunStatMakerWorker fetchs all transactions for the previous hours
// that was not saved to the stat table.
func RunStatMakerWorker(ctx context.Context, cfg StatMakerWorkerConfig, log *zap.Logger) {
	log = log.With(zap.String("worker", "stat_maker_worker"))

	var (
		hstore     = cfg.HStore
		tstore     = cfg.TStore
		db         = cfg.DB
		retrydelay = cfg.RetryDelay

		num  int
		till time.Time
		err  error
	)

	for {
		num, err = syncstats(ctx, hstore, tstore, db, time.Now().Truncate(time.Hour).Add(-time.Hour))
		if err != nil {
			log.Error("unable to handle first tick", zap.Error(err))

			select {
			case <-ctx.Done():
				return
			case <-time.After(retrydelay):
				continue
			}
		}

		log.Debug("handled first tick", zap.Int("inserted_stats", num))

		break
	}

	timer := time.NewTimer(untilNextHour())

	for {
		select {
		case till = <-timer.C:
			till = till.Truncate(time.Hour).Add(-time.Hour)
		case <-ctx.Done():
			log.Info("context canceled")

			return
		}

		log.Debug("handle tick for stats", zap.Time("till", till))

		num, err = syncstats(ctx, hstore, tstore, db, till)
		if err != nil {
			log.Error("unable to handle tick", zap.Error(err))

			timer.Reset(retrydelay)
			continue
		}

		log.Debug("handled tick", zap.Int("inserted_stats", num))

		timer.Reset(untilNextHour())
	}
}

// unitlNextHour calculates duration before the next hour.
func untilNextHour() time.Duration {
	return time.Until(time.Now().Add(time.Hour).Truncate(time.Hour))
}

func syncstats(ctx context.Context, hstore btcount.HistoryStatStorage, tstore btcount.TransactionStorage, db btcount.Database, till time.Time) (amount int, err error) {
	var hstat btcount.HistoryStat
	hstat, err = hstore.LoadLastStat(ctx, db, till)
	if err != nil && !errors.Is(err, btcount.ErrNotFound) {
		return 0, fmt.Errorf("loading last history stat: %w", err)
	}

	if !hstat.Datetime.Before(till) {
		return 0, nil
	}

	var ts []btcount.Transaction
	ts, err = tstore.Load(ctx, db, btcount.NewTimeRangeQuery(hstat.Datetime, till))
	if err != nil {
		return 0, fmt.Errorf("loading transactions: %w", err)
	}

	if len(ts) == 0 {
		return 0, nil
	}

	stats := btcount.CollectTransactionsIntoStats(ts, hstat.Amount)
	err = hstore.SaveMany(ctx, db, stats)
	if err != nil {
		return 0, fmt.Errorf("saving many stats: %w", err)
	}

	return len(stats), nil
}
