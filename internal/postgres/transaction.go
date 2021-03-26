package postgres

import (
	"context"
	"fmt"

	"github.com/ferux/btcount/internal/btcontext"
	"github.com/ferux/btcount/internal/btcount"

	"go.uber.org/zap"
)

// NewTransactionStore creates new transaction store.
func NewTransactionStore() TransactionStore { return TransactionStore{} }

type TransactionStore struct{}

const transactionColumns = `"datetime"` +
	`, "amount"`

func (TransactionStore) Save(ctx context.Context, db btcount.Database, transaction btcount.Transaction) (err error) {
	const query = `INSERT INTO btcount.transactions (` + transactionColumns + `) VALUES (` +
		`  $1` +
		`, $2` +
		`)`

	err = db.Exec(ctx, query,
		transaction.Datetime,
		transaction.Amount,
	)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}

func (TransactionStore) Load(ctx context.Context, db btcount.Database, params btcount.TimerangeQuery) (ts []btcount.Transaction, err error) {
	const query = `SELECT ` + transactionColumns +
		`  FROM btcount.transactions` +
		`  WHERE "datetime" BETWEEN $1 AND $2`

	var rows btcount.DBRows
	rows, err = db.Query(ctx, query, params.Since, params.Till)
	if err != nil {
		return nil, fmt.Errorf("querying rows: %w", err)
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
				Error("unable to close rows", zap.Error(err))
		}
	}()

	for rows.Next() {
		var t btcount.Transaction
		err = rows.Scan(
			&t.Datetime,
			&t.Amount,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		ts = append(ts, t)
	}

	return ts, nil
}
