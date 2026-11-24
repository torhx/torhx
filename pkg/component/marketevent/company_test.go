package marketevent_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/torhx/identifier"

	"github.com/torhx/torhx/pkg/component/marketevent"
)

// timings holds one timing of each form on one date in New York.
type timings struct {
	// date is the date without a flag.
	date marketevent.Timing

	// beforeMarket is the date before the market opens.
	beforeMarket marketevent.Timing

	// afterMarket is the date after the market closes.
	afterMarket marketevent.Timing

	// exact is 16:05 on the date.
	exact marketevent.Timing
}

// mustTimings returns one timing of each form on 2021-02-05 in New York.
func mustTimings(t *testing.T) timings {
	t.Helper()

	day := mustLocationDate(t, "2021-02-05", newYork)

	beforeMarket, err := marketevent.NewBeforeMarketTiming(day)
	require.NoError(t, err)

	afterMarket, err := marketevent.NewAfterMarketTiming(day)
	require.NoError(t, err)

	exact, err := marketevent.NewExactTiming(time.Date(2021, time.February, 5, 16, 5, 0, 0, day.Location()), day)
	require.NoError(t, err)

	return timings{
		date:         mustDateTiming(t, "2021-02-05"),
		beforeMarket: beforeMarket,
		afterMarket:  afterMarket,
		exact:        exact,
	}
}

func TestNewEarnings(t *testing.T) {
	t.Parallel()

	aapl := mustFIGI(t, aaplFIGI)
	on := mustTimings(t)

	tests := []struct {
		name   string
		timing marketevent.Timing
	}{
		{name: "date", timing: on.date},
		{name: "before market", timing: on.beforeMarket},
		{name: "after market", timing: on.afterMarket},
		{name: "exact", timing: on.exact},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			report, err := marketevent.NewEarnings(aapl, tt.timing)
			require.NoError(t, err)

			assert.Equal(t, aapl, report.Stock())
			assert.Equal(t, mustStockSubject(t, aaplFIGI), report.Subject())
			assert.Equal(t, tt.timing, report.Timing())
			assert.Equal(t, "earnings", report.Kind())
		})
	}
}

func TestNewEarningsRejects(t *testing.T) {
	t.Parallel()

	_, err := marketevent.NewEarnings(identifier.FIGI{}, mustDateTiming(t, "2021-02-05"))
	require.ErrorContains(t, err, "stock is missing")

	_, err = marketevent.NewEarnings(mustFIGI(t, aaplFIGI), marketevent.Timing{})
	require.ErrorContains(t, err, "timing is missing")
}

func TestNewDividend(t *testing.T) {
	t.Parallel()

	aapl := mustFIGI(t, aaplFIGI)
	exDate := mustDateTiming(t, "2021-02-05")

	dividend, err := marketevent.NewDividend(aapl, exDate, 0.205)
	require.NoError(t, err)

	assert.Equal(t, aapl, dividend.Stock())
	assert.Equal(t, mustStockSubject(t, aaplFIGI), dividend.Subject())
	assert.Equal(t, exDate, dividend.Timing())
	assert.Equal(t, "dividend", dividend.Kind())
	assert.InDelta(t, 0.205, dividend.Amount(), 0)
}

func TestNewDividendRejects(t *testing.T) {
	t.Parallel()

	aapl := mustFIGI(t, aaplFIGI)
	on := mustTimings(t)

	tests := []struct {
		name    string
		stock   identifier.FIGI
		timing  marketevent.Timing
		amount  float64
		wantErr string
	}{
		{name: "zero stock", stock: identifier.FIGI{}, timing: on.date, amount: 0.205, wantErr: "stock is missing"},
		{name: "zero timing", stock: aapl, timing: marketevent.Timing{}, amount: 0.205, wantErr: "timing is missing"},
		{name: "exact timing", stock: aapl, timing: on.exact, amount: 0.205, wantErr: "ex-date is an exact time"},
		{name: "before market", stock: aapl, timing: on.beforeMarket, amount: 0.205, wantErr: "ex-date has the flag BeforeMarket"},
		{name: "after market", stock: aapl, timing: on.afterMarket, amount: 0.205, wantErr: "ex-date has the flag AfterMarket"},
		{name: "zero amount", stock: aapl, timing: on.date, amount: 0, wantErr: "not positive and finite"},
		{name: "negative amount", stock: aapl, timing: on.date, amount: -0.205, wantErr: "not positive and finite"},
		{name: "amount that is not a number", stock: aapl, timing: on.date, amount: math.NaN(), wantErr: "not positive and finite"},
		{name: "infinite amount", stock: aapl, timing: on.date, amount: math.Inf(1), wantErr: "not positive and finite"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := marketevent.NewDividend(tt.stock, tt.timing, tt.amount)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestNewSplit(t *testing.T) {
	t.Parallel()

	aapl := mustFIGI(t, aaplFIGI)
	exDate := mustDateTiming(t, "2020-08-31")

	split, err := marketevent.NewSplit(aapl, exDate, 0.25)
	require.NoError(t, err)

	assert.Equal(t, aapl, split.Stock())
	assert.Equal(t, mustStockSubject(t, aaplFIGI), split.Subject())
	assert.Equal(t, exDate, split.Timing())
	assert.Equal(t, "split", split.Kind())
	assert.InDelta(t, 0.25, split.Factor(), 0)
}

func TestNewSplitRejects(t *testing.T) {
	t.Parallel()

	aapl := mustFIGI(t, aaplFIGI)
	on := mustTimings(t)

	tests := []struct {
		name    string
		stock   identifier.FIGI
		timing  marketevent.Timing
		factor  float64
		wantErr string
	}{
		{name: "zero stock", stock: identifier.FIGI{}, timing: on.date, factor: 0.25, wantErr: "stock is missing"},
		{name: "zero timing", stock: aapl, timing: marketevent.Timing{}, factor: 0.25, wantErr: "timing is missing"},
		{name: "exact timing", stock: aapl, timing: on.exact, factor: 0.25, wantErr: "ex-date is an exact time"},
		{name: "before market", stock: aapl, timing: on.beforeMarket, factor: 0.25, wantErr: "ex-date has the flag BeforeMarket"},
		{name: "after market", stock: aapl, timing: on.afterMarket, factor: 0.25, wantErr: "ex-date has the flag AfterMarket"},
		{name: "zero factor", stock: aapl, timing: on.date, factor: 0, wantErr: "not positive and finite"},
		{name: "negative factor", stock: aapl, timing: on.date, factor: -0.25, wantErr: "not positive and finite"},
		{name: "factor that is not a number", stock: aapl, timing: on.date, factor: math.NaN(), wantErr: "not positive and finite"},
		{name: "infinite factor", stock: aapl, timing: on.date, factor: math.Inf(1), wantErr: "not positive and finite"},
		{name: "factor of one", stock: aapl, timing: on.date, factor: 1, wantErr: "factor 1 changes no price"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := marketevent.NewSplit(tt.stock, tt.timing, tt.factor)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
