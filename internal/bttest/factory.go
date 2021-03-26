package bttest

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ferux/btcount/internal/btcount"
)

func NewTransaction() btcount.Transaction {
	return btcount.Transaction{
		Amount:   btcount.DecimalFromFloat(rand.Float64()),
		Datetime: time.Now(),
	}
}

func MustInsertTransaction(t *testing.T, transaction btcount.Transaction) {
	ctx := GetContext()
	db := GetDB()
	store := GetTStore()

	err := store.Save(ctx, db, transaction)
	must(t, err)
}

func MustInsertHistoryStat(t *testing.T, stat btcount.HistoryStat) {
	ctx := GetContext()
	db := GetDB()
	store := GetHStore()

	err := store.Save(ctx, db, stat)
	must(t, err)
}
