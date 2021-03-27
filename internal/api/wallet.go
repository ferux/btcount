package api

import (
	"context"
	"fmt"
	"time"

	"github.com/ferux/btcount/internal/btcontext"
	"github.com/ferux/btcount/internal/btcount"
	"github.com/ferux/btcount/internal/cache"
	"go.uber.org/zap"
)

// WalletAPI provides methods for interacting with wallet.
type WalletAPI interface {
	// CreateTransaction saves transaction to the storage. It also check
	// the amount is not less that 0 and the datetime is not empty.
	CreateTransaction(ctx context.Context, t btcount.Transaction) (err error)
	// FetchBalanceByHour loads balance of the wallet by the provided
	// time range.
	FetchBalanceByHour(ctx context.Context, from, till time.Time) (ts []btcount.HistoryStat, err error)
}

// NopWalletAPI implements WalletAPI and stands for No-op entity.
type NopWalletAPI struct{}

// CreateTransaction implements WalletAPI.
func (NopWalletAPI) CreateTransaction(context.Context, btcount.Transaction) error { return nil }

// FetchBalanceByHour implements WalletAPI.
func (NopWalletAPI) FetchBalanceByHour(context.Context, time.Time, time.Time) ([]btcount.HistoryStat, error) {
	return nil, nil
}

// NewWalletAPI creates a new wallet api.
func NewWalletAPI(db btcount.Database, hstore btcount.HistoryStatStorage, tstore btcount.TransactionStorage, statCollector *cache.CurrentHourStatCollector) WalletAPI {
	return walletAPI{
		db:            db,
		tstore:        tstore,
		hstore:        hstore,
		statCollector: statCollector,
	}
}

type walletAPI struct {
	db     btcount.Database
	tstore btcount.TransactionStorage
	hstore btcount.HistoryStatStorage

	statCollector *cache.CurrentHourStatCollector
}

// CreateTransaction implements WalletAPI interface.
func (api walletAPI) CreateTransaction(ctx context.Context, transaction btcount.Transaction) (err error) {
	if transaction.Amount.LessThan(btcount.DecimalFromFloat(0)) {
		return fmt.Errorf("%w: %s", btcount.ErrNegativeValue, "amount")
	}

	if transaction.Datetime.IsZero() {
		return fmt.Errorf("%w: %s", btcount.ErrInvalidParameter, "datetime")
	}

	transaction.Datetime = transaction.Datetime.UTC()

	btcontext.Logger(ctx).Debug("saving", zap.Any("transaction", transaction))

	err = api.tstore.Save(ctx, api.db, transaction)
	if err != nil {
		return fmt.Errorf("saving transaction to the storage: %w", err)
	}

	if api.statCollector != nil {
		api.statCollector.Collect(transaction)
	}

	return nil
}

func (api walletAPI) FetchBalanceByHour(ctx context.Context, since time.Time, till time.Time) (stats []btcount.HistoryStat, err error) {
	log := btcontext.Logger(ctx)
	// get the start of the hour.
	since = since.Truncate(time.Hour)
	// round up till.
	tillHour := till.Truncate(time.Hour)
	if tillHour.Before(till) {
		till = tillHour.Add(time.Hour)
	} else {
		till = tillHour
	}
	// calculate left time. This is needed because it's possible to collect
	// history stats until the current hour. The left transactions might
	// be cached, if cache is initied, or should be loaded from the database
	// and prepared.
	log.Debug("time ranges",
		zap.Time("since", since),
		zap.Time("till", till),
	)

	stats, err = api.hstore.Load(ctx, api.db, btcount.TimerangeQuery{
		Since: since,
		Till:  till,
	})
	if err != nil {
		return nil, fmt.Errorf("loading history stats: %w", err)
	}

	if len(stats) == 0 {
		log.Debug("no stats loaded, using slow method")
		return api.loadBalanceSlow(ctx, since, till)
	}

	log.Debug("loaded history stats", zap.Int("len", len(stats)))
	lastStat := stats[len(stats)-1]
	// it means lastStat datetime is equal or after till and there's
	// nothing left to do.
	if !lastStat.Datetime.Before(till) {
		return stats, nil
	}

	hourStart := time.Now().Truncate(time.Hour)
	if lastStat.Datetime.Before(hourStart) {
		var ts []btcount.Transaction
		ts, err = api.tstore.Load(ctx, api.db, btcount.TimerangeQuery{
			Since: till,
			Till:  hourStart,
		})
		if err != nil {
			return nil, fmt.Errorf("loading transactions: %w", err)
		}

		newstats := btcount.CollectTransactionsIntoStats(ts, lastStat.Amount)
		if len(newstats) > 0 {
			stats = append(stats, newstats...)
			lastStat = stats[len(stats)-1]
		}
	}

	if api.statCollector != nil {
		log.Debug("loading leftovers from cache")

		return append(stats, api.statCollector.GetStat()), nil
	}

	var ts []btcount.Transaction
	ts, err = api.tstore.Load(ctx, api.db, btcount.TimerangeQuery{
		Since: hourStart,
		Till:  till,
	})
	if err != nil {
		return nil, fmt.Errorf("loading transactions for last hour: %w", err)
	}

	newstats := btcount.CollectTransactionsIntoStats(ts, lastStat.Amount)
	return append(stats, newstats...), nil
}

func (api walletAPI) loadBalanceSlow(ctx context.Context, since, till time.Time) (stats []btcount.HistoryStat, err error) {
	now := time.Now().Truncate(time.Hour)

	var ts []btcount.Transaction
	ts, err = api.tstore.Load(ctx, api.db, btcount.NewTimeRangeQuery(since, till))
	if err != nil {
		return nil, fmt.Errorf("loading transactions: %w", err)
	}

	stats = btcount.CollectTransactionsIntoStats(ts, btcount.DecimalFromFloat(0))

	if till.Before(now) {
		return stats, nil
	}

	lastStat := stats[len(stats)-1]
	if lastStat.Datetime.After(now) {
		api.statCollector.Adjust(lastStat.Amount)
	}

	return stats, nil
}

// FetchBalanceByHour implements WalletAPI interface.
func (api walletAPI) FetchBalanceByHour_old(ctx context.Context, since time.Time, till time.Time) (stats []btcount.HistoryStat, err error) {
	log := btcontext.Logger(ctx)
	// get the start of the hour.
	since = since.Truncate(time.Hour)
	// till should be rounded up but since truncate performs round down,
	// check if till later than start of the hour and add addirional
	// hour then.
	now := time.Now().Truncate(time.Hour)
	if till.After(now) {
		till = now
	}
	// tillHour := till.Truncate(time.Hour)
	// now := time.Now().Truncate(time.Hour)
	// if till.After(tillHour) {
	// 	till = tillHour.Add(time.Hour).Truncate(time.Hour)
	// }

	historyStatsQuery := btcount.TimerangeQuery{
		Since: since,
		Till:  till,
	}

	log.Debug("loaded history stats", zap.Any("bouds", historyStatsQuery))
	stats, err = api.hstore.Load(ctx, api.db, historyStatsQuery)
	if err != nil {
		return nil, fmt.Errorf("loading history stats: %w", err)
	}

	log.Debug("loaded history stats", zap.Int("len", len(stats)))

	var startHour time.Time
	initialBalance := btcount.DecimalFromFloat(0.0)
	if len(stats) == 0 {
		startHour = since
	} else {
		stat := stats[len(stats)-1]
		initialBalance = stat.Amount
		startHour = stat.Datetime
	}

	if !till.After(startHour) {
		return stats, nil
	}

	if api.statCollector != nil {
		stats = append(stats, api.statCollector.GetStat())
	} else {
		var ts []btcount.Transaction
		ts, err = api.tstore.Load(ctx, api.db, btcount.TimerangeQuery{
			Since: startHour,
			Till:  till,
		})
		if err != nil {
			return nil, fmt.Errorf("loading transactions: %w", err)
		}

		log.Debug("loading additional raw transactions",
			zap.Int("len", len(ts)),
		)

		leftStats := btcount.CollectTransactionsIntoStats(ts, initialBalance)
		if len(leftStats) > 0 {
			stats = append(stats, leftStats...)
		}

	}
	return stats, nil
}
