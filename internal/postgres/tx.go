package postgres

import (
	"context"
	"fmt"

	"github.com/ferux/btcount/internal/btcount"
	"github.com/jackc/pgx/v4"
)

func Begin(ctx context.Context, db btcount.Database) (tx btcount.Database, err error) {
	pgxdb, ok := db.(*DB)
	if !ok {
		return nil, fmt.Errorf("%w: %T", btcount.ErrUnexpectedType, tx)
	}

	return pgxdb.Begin(ctx)
}

// Commit commits the transaction
func Commit(ctx context.Context, tx btcount.Database) (err error) {
	txpg, ok := tx.(*Tx)
	if !ok {
		return fmt.Errorf("%w: %T", btcount.ErrUnexpectedType, tx)
	}

	return txpg.Commit(ctx)
}

// Rollback rollsback the transaction.
func Rollback(ctx context.Context, tx btcount.Database) (err error) {
	txpg, ok := tx.(*Tx)
	if !ok {
		return fmt.Errorf("%w: %T", btcount.ErrUnexpectedType, tx)
	}

	return txpg.Rollback(ctx)
}

type Tx struct {
	tx pgx.Tx
}

func (tx *Tx) Commit(ctx context.Context) (err error) {
	return tx.tx.Commit(ctx)
}

func (tx *Tx) Rollback(ctx context.Context) (err error) {
	return tx.tx.Rollback(ctx)
}

// Exec implements btcount.Database interface.
func (tx *Tx) Exec(ctx context.Context, query string, args ...interface{}) (err error) {
	_, err = tx.tx.Exec(ctx, query, args...)

	return err
}

// QueryRow implementa btcount.Database interface.
func (tx *Tx) QueryRow(ctx context.Context, query string, args ...interface{}) btcount.DBRow {
	return &dbrow{Row: tx.tx.QueryRow(ctx, query, args...)}
}

// Query implements btcount.Database interface.
func (tx *Tx) Query(ctx context.Context, query string, args ...interface{}) (rows btcount.DBRows, err error) {
	var pgrows pgx.Rows
	pgrows, err = tx.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return &dbrows{Rows: pgrows}, nil
}

// SendBatch implements btcount.Database interface.
func (tx *Tx) SendBatch(ctx context.Context, b btcount.Batch) btcount.BatchResults {
	batch := b.(*pgx.Batch)
	results := tx.tx.SendBatch(ctx, batch)

	return &batchresults{BatchResults: results}
}

// Close implements btcount.Database interface.
func (tx *Tx) Close() error {
	return nil
}
