package btcount

import (
	"testing"
	"time"
)

func TestCollectTransactionsIntoStats(t *testing.T) {
	now := time.Date(2010, 1, 2, 3, 4, 5, 0, time.UTC) // 2010-01-02 03:04:05.000

	var tt = []struct {
		name       string
		in         []Transaction
		initialSum Decimal
		exp        []HistoryStat
	}{{
		name:       "empty transactions",
		initialSum: DecimalFromFloat(0.0),
		in:         nil,
		exp:        []HistoryStat{},
	}, {
		name:       "single transaction",
		initialSum: DecimalFromFloat(10.0),
		in:         []Transaction{{Amount: DecimalFromFloat(1.0), Datetime: now}},
		exp: []HistoryStat{{
			Datetime: now.Truncate(time.Hour).Add(time.Hour),
			Amount:   DecimalFromFloat(11.0),
		}},
	}, {
		name:       "multiple transactions in hour",
		initialSum: DecimalFromFloat(11.0),
		in: []Transaction{
			{Amount: DecimalFromFloat(0.5), Datetime: now},
			{Amount: DecimalFromFloat(1.2), Datetime: now.Add(time.Minute * 2)},
		},
		exp: []HistoryStat{{
			Datetime: now.Truncate(time.Hour).Add(time.Hour),
			Amount:   DecimalFromFloat(12.7),
		}},
	}, {
		name:       "multiple transactions in hours",
		initialSum: DecimalFromFloat(12.0),
		in: []Transaction{
			// Current hour. (+1.7) [13.7]
			{Amount: DecimalFromFloat(0.4), Datetime: now},
			{Amount: DecimalFromFloat(1.2), Datetime: now.Add(time.Minute * 2)},
			{Amount: DecimalFromFloat(0.1), Datetime: now.Add(time.Minute * 4)},
			// Next hour. (+0.5) [14.2]
			{Amount: DecimalFromFloat(0.5), Datetime: now.Truncate(time.Hour).Add(time.Hour)},
			// After 2 hours. (+2.8) [17.0]
			{Amount: DecimalFromFloat(0.7), Datetime: now.Add(time.Hour * 2)},
			{Amount: DecimalFromFloat(2.1), Datetime: now.Add(time.Hour*2 + time.Minute*5)},
		},
		exp: []HistoryStat{{
			Datetime: now.Truncate(time.Hour).Add(time.Hour),
			Amount:   DecimalFromFloat(13.7),
		}, {
			Datetime: now.Truncate(time.Hour).Add(time.Hour * 2),
			Amount:   DecimalFromFloat(14.2),
		}, {
			Datetime: now.Truncate(time.Hour).Add(time.Hour * 3),
			Amount:   DecimalFromFloat(17.0),
		}},
	}}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := CollectTransactionsIntoStats(tc.in, tc.initialSum)

			// Do not use reflect.DeepEqual here because it does not work.
			if len(got) != len(tc.exp) {
				t.Errorf("length not equal\nexp: %v\ngot: %v", tc.exp, got)
				return
			}
			for i := range got {
				if !got[i].Amount.Equal(tc.exp[i].Amount) {
					t.Errorf("values not equal\nexp: %v\ngot: %v", tc.exp, got)

					return
				}
			}
		})
	}
}
