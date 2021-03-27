package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ferux/btcount/internal/btcount"
	"go.uber.org/zap"
)

type CurrentHourStatCollector struct {
	lastStat btcount.HistoryStat
	mu       sync.RWMutex
}

// Collect appends new transaction to the history stat.
func (c *CurrentHourStatCollector) Collect(t btcount.Transaction) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastStat.Amount = c.lastStat.Amount.Add(t.Amount)
	c.lastStat.Datetime = t.Datetime
}

// Adjust changes the value of the amount.
func (c *CurrentHourStatCollector) Adjust(amount btcount.Decimal) {
	c.lastStat.Amount = amount
	c.lastStat.Datetime = time.Now()
}

// GetStat returns the currently collected stat.
func (c *CurrentHourStatCollector) GetStat() btcount.HistoryStat {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lastStat
}

type HistoryStatParams struct {
	HStore btcount.HistoryStatStorage
	TStore btcount.TransactionStorage

	DB btcount.Database
}

func InitHistoryStatCollector(ctx context.Context, params HistoryStatParams, log *zap.Logger) (c *CurrentHourStatCollector, err error) {
	hourStart := time.Now()
	lastStat, err := params.HStore.LoadLastStat(ctx, params.DB, hourStart)
	if err != nil {
		return nil, fmt.Errorf("loading last stat: %w", err)
	}

	var ts []btcount.Transaction
	ts, err = params.TStore.Load(ctx, params.DB, btcount.TimerangeQuery{Since: lastStat.Datetime, Till: time.Now()})
	if err != nil {
		return nil, fmt.Errorf("loading transactions: %w", err)
	}

	stats := btcount.CollectTransactionsIntoStats(ts, lastStat.Amount)
	if len(stats) == 0 {
		return &CurrentHourStatCollector{lastStat: btcount.HistoryStat{}}, nil
	}

	stat := stats[len(stats)-1]
	log.Debug("created cache", zap.Any("stat", stat))
	return &CurrentHourStatCollector{lastStat: stat}, nil
}
