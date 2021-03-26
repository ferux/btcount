package btcount

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// Transaction is a single transaction that stores the amount of coins
// that has been sent and the time of it.
type Transaction struct {
	Amount   Decimal   `json:"amount"`
	Datetime time.Time `json:"datetime"`
}

// Decimal is a wrapper around shopsptring/decimal value.
type Decimal struct{ decimal.Decimal }

// Add adds another decimal and returns new value.
func (d Decimal) Add(other Decimal) Decimal {
	return Decimal{d.Decimal.Add(other.Decimal)}
}

// Equal checks whether two decimals equals or not.
func (d Decimal) Equal(other Decimal) (equal bool) {
	return d.Decimal.Equal(other.Decimal)
}

// LessThan check whether the origin value is less than other value.
func (d Decimal) LessThan(other Decimal) (less bool) {
	return d.Decimal.LessThan(other.Decimal)
}

// DecimalFromFloat creates new decimal from float64.
func DecimalFromFloat(v float64) Decimal {
	return Decimal{decimal.NewFromFloat(v)}
}

// CollectTransactionsIntoStats iterates over each transaction and
// makes history stat partitioned by each hour.
func CollectTransactionsIntoStats(origin []Transaction, initialSum Decimal) (stats []HistoryStat) {
	if len(origin) == 0 {
		return []HistoryStat{}
	}

	if len(origin) == 1 {
		return []HistoryStat{{
			Datetime: origin[0].Datetime.Truncate(time.Hour).Add(time.Hour),
			Amount:   origin[0].Amount.Add(initialSum),
		}}
	}

	// Do not sort origin slice to avoid changing the origin data.
	ts := make([]Transaction, len(origin))
	copy(ts, origin)

	sort.Slice(ts, func(i, j int) bool {
		return ts[i].Datetime.Before(ts[j].Datetime)
	})

	endHour := ts[0].Datetime.Truncate(time.Hour).Add(time.Hour)
	sum := ts[0].Amount.Add(initialSum)

	var addLast bool
	for i := 1; i < len(ts); i++ {
		current := ts[i]
		if current.Datetime.Before(endHour) {
			sum = sum.Add(current.Amount)
			addLast = true

			continue
		}

		stat := HistoryStat{
			Datetime: endHour,
			Amount:   sum,
		}
		stats = append(stats, stat)

		sum = sum.Add(current.Amount)
		endHour = current.Datetime.Truncate(time.Hour).Add(time.Hour)
		addLast = false
	}

	if addLast {
		stat := HistoryStat{
			Datetime: endHour,
			Amount:   sum,
		}

		stats = append(stats, stat)
	}

	return stats
}
