package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/ferux/btcount/internal/btcontext"
	"github.com/ferux/btcount/internal/btcount"

	"go.uber.org/zap"
)

// NewHistoryStore creates new history store.
func NewHistoryStore() HistoryStore { return HistoryStore{} }

// HistoryStore implements btcount.HistoryStorage interface
// for postgres database based on pgx driver.
type HistoryStore struct{}

const historyColumns = `"datetime"` +
	`, "amount"`

// Save implements btcount.HistoryStorage interface.
func (HistoryStore) Save(ctx context.Context, db btcount.Database, stat btcount.HistoryStat) (err error) {
	const query = `INSERT INTO btcount.history_stats (` + historyColumns + `) VALUES (` +
		`  $1` +
		`, $2` +
		`)`

	err = db.Exec(ctx, query,
		stat.Datetime,
		stat.Amount,
	)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}

// SaveMany implements btcount.HistoryStorage interface.
func (hs HistoryStore) SaveMany(ctx context.Context, db btcount.Database, stats []btcount.HistoryStat) (err error) {
	if len(stats) == 0 {
		return nil
	}

	const query = `INSERT INTO btcount.history_stats (` + historyColumns + `) VALUES (` +
		`  $1` +
		`, $2` +
		`)`

	batch := newBatch()
	for i := range stats {
		batch.Queue(query,
			stats[i].Datetime,
			stats[i].Amount,
		)
	}

	results := db.SendBatch(ctx, batch)
	defer func() {
		errclose := results.Close()
		if errclose == nil {
			return
		}

		if err == nil {
			err = errclose
		} else {
			btcontext.
				Logger(ctx).
				Error("unable to close results", zap.Error(errclose))
		}
	}()

	for i := 0; i < len(stats); i++ {
		err = results.Exec()
		if err != nil {
			return fmt.Errorf("inserting %v: %w", stats[i], err)
		}
	}

	return nil
}

// Load implements btcount.HistoryStorage interface.
func (HistoryStore) Load(ctx context.Context, db btcount.Database, query btcount.TimerangeQuery) (hs []btcount.HistoryStat, err error) {
	const q = `SELECT ` + historyColumns +
		` FROM btcount.history_stats` +
		` WHERE "datetime" > $1 AND "datetime" <= $2` +
		` ORDER BY "datetime" ASC;`

	var rows btcount.DBRows
	rows, err = db.Query(ctx, q, query.Since, query.Till)
	if err != nil {
		return nil, fmt.Errorf("querying: %w", err)
	}
	defer func() {
		errclose := rows.Close()
		if errclose == nil {
			return
		}

		if err == nil {
			err = errclose
		} else {
			btcontext.
				Logger(ctx).
				Error("unable to close rows", zap.Error(errclose))
		}
	}()

	for rows.Next() {
		var stat btcount.HistoryStat
		err = rows.Scan(
			&stat.Datetime,
			&stat.Amount,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		hs = append(hs, stat)
	}

	return hs, nil
}

// LoadLastStat implements btcount.HistoryStorage interface.
func (HistoryStore) LoadLastStat(ctx context.Context, db btcount.Database, ts time.Time) (h btcount.HistoryStat, err error) {
	const query = `SELECT ` + historyColumns +
		` FROM btcount.history_stats` +
		` WHERE "datetime" <= $1` +
		` ORDER BY "datetime" DESC LIMIT 1;`

	err = db.QueryRow(ctx, query, ts).Scan(
		&h.Datetime,
		&h.Amount,
	)
	if err != nil {
		return h, fmt.Errorf("scanning row: %w", err)
	}

	return h, nil
}
