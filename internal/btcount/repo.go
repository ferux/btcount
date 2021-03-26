package btcount

import (
	"context"
	"time"
)

// Database is an interface to interacting with database.
type Database interface {
	Exec(ctx context.Context, query string, args ...interface{}) error
	QueryRow(ctx context.Context, query string, args ...interface{}) DBRow
	Query(ctx context.Context, query string, args ...interface{}) (DBRows, error)
	SendBatch(ctx context.Context, b Batch) BatchResults

	Close() error
}

type Beginner interface {
	Begin(ctx context.Context) (Database, error)
}

// Batch allows to execute multiple methods.
type Batch interface {
	Queue(query string, args ...interface{})
	Len() int
}

// BatchResults is the results of executing batch.
type BatchResults interface {
	Exec() error
	Query() (DBRows, error)
	QueryRow() DBRow
	Close() error
}

// DBRow is an abstraction over single database row.
type DBRow interface {
	Scan(dest ...interface{}) error
}

// DBRows is an iterator over rows.
type DBRows interface {
	DBRow

	Next() bool
	Err() error
	Close() error
}

// TransactionStorage provides API for iteracting with transaction storage.
type TransactionStorage interface {
	// Save the transaction to the storage.
	Save(ctx context.Context, db Database, transaction Transaction) (err error)
	// Load transactions by provided query.
	Load(ctx context.Context, db Database, query TimerangeQuery) (ts []Transaction, err error)
}

// TimerangeQuery filters output by provided bounds.
type TimerangeQuery struct {
	Since time.Time
	Till  time.Time
}

// NewTimeRangeQuery creates a new query for selecting records by time
// interval.
func NewTimeRangeQuery(since time.Time, till time.Time) TimerangeQuery {
	return TimerangeQuery{
		Since: since,
		Till:  till,
	}
}

// HistoryStatStorage provides API for interacting with storage.
type HistoryStatStorage interface {
	// Save a single history stat to the database.
	Save(ctx context.Context, db Database, stat HistoryStat) (err error)
	// SaveMany saves many history stats to the database.
	SaveMany(ctx context.Context, db Database, stats []HistoryStat) (err error)
	// Load history stats from the database, ordered by datetime in
	// ascending order.
	Load(ctx context.Context, db Database, query TimerangeQuery) (hss []HistoryStat, err error)
	// LoadLastStat loads last saved stat prior to provided ts.
	LoadLastStat(ctx context.Context, db Database, ts time.Time) (h HistoryStat, err error)
}

// HistoryStat stores amount
type HistoryStat struct {
	Datetime time.Time
	Amount   Decimal
}
