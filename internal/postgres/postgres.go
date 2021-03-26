package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ferux/btcount/internal/btcount"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Config defines params for the database connection.
type Config struct {
	MaxConns int32
	MinConns int32
}

// Open connection to the database.
func Open(ctx context.Context, dsn string, cfg Config) (db *DB, err error) {
	var pgcfg *pgxpool.Config
	pgcfg, err = pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing config; %w", err)
	}

	pgcfg.MaxConns = cfg.MaxConns
	pgcfg.MinConns = cfg.MinConns

	var pool *pgxpool.Pool
	pool, err = pgxpool.ConnectConfig(ctx, pgcfg)
	if err != nil {
		return nil, fmt.Errorf("connection: %w", err)
	}

	return &DB{pool: pool}, nil
}

type DB struct {
	pool *pgxpool.Pool
}

// Exec implements btcount.Database interface.
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (err error) {
	_, err = db.pool.Exec(ctx, query, args...)

	return err
}

// QueryRow implementa btcount.Database interface.
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) btcount.DBRow {
	return &dbrow{Row: db.pool.QueryRow(ctx, query, args...)}
}

// Query implements btcount.Database interface.
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (rows btcount.DBRows, err error) {
	var pgrows pgx.Rows
	pgrows, err = db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return &dbrows{Rows: pgrows}, nil
}

// SendBatch implements btcount.Database interface.
func (db *DB) SendBatch(ctx context.Context, b btcount.Batch) btcount.BatchResults {
	batch := b.(*pgx.Batch)
	results := db.pool.SendBatch(ctx, batch)

	return &batchresults{BatchResults: results}
}

// Close implements btcount.Database interface.
func (db *DB) Close() error {
	db.pool.Close()

	return nil
}

func (db *DB) Begin(ctx context.Context) (tx btcount.Database, err error) {
	pgxtx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction")
	}

	return &Tx{tx: pgxtx}, nil
}

type dbrows struct{ pgx.Rows }

// Close implements btcount.DBRows interface.
func (r *dbrows) Close() error {
	r.Rows.Close()

	return nil
}

func newBatch() btcount.Batch {
	return &pgx.Batch{}
}

type batchresults struct{ pgx.BatchResults }

// Exec implements btcount.BatchResults interface.
func (br *batchresults) Exec() (err error) {
	_, err = br.BatchResults.Exec()

	return err
}

// Query implements btcount.BatchResults interface.
func (br *batchresults) Query() (btcount.DBRows, error) {
	rows, err := br.BatchResults.Query()
	if err != nil {
		return nil, err
	}

	return &dbrows{rows}, nil
}

// QueryRow implements btcount.BatchResults interface.
func (br *batchresults) QueryRow() btcount.DBRow {
	return br.BatchResults.QueryRow()
}

type dbrow struct{ pgx.Row }

// Scan implements btcount.DBRow interface.
func (dbrow *dbrow) Scan(dest ...interface{}) (err error) {
	err = dbrow.Row.Scan(dest...)

	return mapError(err)
}

func mapError(err error) (mapped error) {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return btcount.ErrNotFound
	}

	return nil
}
