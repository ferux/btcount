package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"

	// Import pgx driver to the sql driver list.
	_ "github.com/jackc/pgx/v4/stdlib"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	dsn := os.Getenv("DATABASE_DSN")

	db, err := opendb(ctx, dsn)
	exitOnError(err)

	err = makeschema(ctx, db)
	exitOnError(err)

	migrations := getmigrations()

	err = migrate(ctx, db, migrations)
	exitOnError(err)
}

func exitOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func opendb(ctx context.Context, dsn string) (db *sql.DB, err error) {
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("dialing db: %w", err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return db, nil
}

type migration struct {
	name string
	sql  string
}

func migrate(ctx context.Context, db *sql.DB, migrations []migration) (err error) {
	var tx *sql.Tx
	tx, err = db.Begin()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	defer func() {
		var errtx error
		if err != nil {
			log.Println("rolling back changes")
			errtx = tx.Rollback()
		} else {
			log.Println("commiting changes")
			errtx = tx.Commit()
		}

		if errtx != nil {
			if err == nil {
				err = errtx
			} else {
				log.Printf("finishing transaction: %v", err)
			}
		}
	}()

	var applied bool
	for _, m := range migrations {
		applied, err = checkApplied(ctx, tx, m.name)
		if err != nil {
			return fmt.Errorf("checking migration %s: %w", m.name, err)
		}

		if applied {
			continue
		}

		log.Printf("migration %s not applied", m.name)

		err = applyMigration(ctx, tx, m)
		if err != nil {
			return fmt.Errorf("applying migration %s: %w", m.name, err)
		}
	}

	return nil
}

func getmigrations() []migration {
	return []migration{{
		name: "0001_init",
		sql: `
		CREATE SCHEMA IF NOT EXISTS btcount;
		CREATE TABLE IF NOT EXISTS btcount.transactions (` +
			`  "id" BIGSERIAL PRIMARY KEY` +
			`, "datetime" TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now()` +
			`, "amount" FLOAT8 NOT NULL` +
			`);
		CREATE TABLE IF NOT EXISTS btcount.history_stats (` +
			`  "id" BIGSERIAL PRIMARY KEY` +
			`, "datetime" TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now()` +
			`, "amount" FLOAT8 NOT NULL` +
			`);`,
	}}
}

func checkApplied(ctx context.Context, tx *sql.Tx, name string) (applied bool, err error) {
	const query = `SELECT COUNT(1) FROM public.migrations WHERE "name" = $1`

	var count int
	err = tx.QueryRowContext(ctx, query, name).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return count == 1, nil
}

func applyMigration(ctx context.Context, tx *sql.Tx, m migration) (err error) {
	_, err = tx.ExecContext(ctx, m.sql)

	if err != nil {
		return fmt.Errorf("making changes: %w", err)
	}

	const query = `INSERT INTO public.migrations (name) VALUES ($1)`

	_, err = tx.ExecContext(ctx, query, m.name)
	if err != nil {
		return fmt.Errorf("inserting migration name to migrations: %w", err)
	}

	return nil
}

func makeschema(ctx context.Context, db *sql.DB) (err error) {
	const query = `CREATE TABLE IF NOT EXISTS public.migrations ("name" TEXT PRIMARY KEY);`

	_, err = db.ExecContext(ctx, query)
	return err
}
