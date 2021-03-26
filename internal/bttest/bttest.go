package bttest

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/ferux/btcount/internal/api"
	"github.com/ferux/btcount/internal/btcount"
	"github.com/ferux/btcount/internal/postgres"
)

var db btcount.Database
var tx btcount.Database
var hstore btcount.HistoryStatStorage
var tstore btcount.TransactionStorage
var wAPI api.WalletAPI
var ctx context.Context
var cancel context.CancelFunc

func GetDB() btcount.Database               { return tx }
func GetHStore() btcount.HistoryStatStorage { return hstore }
func GetTStore() btcount.TransactionStorage { return tstore }
func GetWalletAPI() api.WalletAPI           { return wAPI }
func GetContext() context.Context           { return ctx }

// Prepare setupts the environment.
func Prepare() error {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Fatal("getting database dsn")
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*10)

	var err error
	db, err = postgres.Open(ctx, dsn, postgres.Config{
		MaxConns: 2,
		MinConns: 1,
	})
	if err != nil {
		return fmt.Errorf("opening database")
	}

	tx, err = postgres.Begin(ctx, db)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	hstore = postgres.NewHistoryStore()
	tstore = postgres.NewTransactionStore()
	wAPI = api.NewWalletAPI(db, hstore, tstore)

	return nil
}

// Finish rollsback the transaction and cancels the context..
func Finish() (err error) {
	err = postgres.Rollback(ctx, tx)
	cancel()

	return err
}

func must(t testing.TB, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
