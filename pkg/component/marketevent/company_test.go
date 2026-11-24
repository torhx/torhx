package marketevent_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talav/civil"
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

// assertExDate checks the date and zone exposed through the Event interface.
func assertExDate(t *testing.T, want civil.LocationDate, timing marketevent.Timing) {
	t.Helper()

	assert.True(t, want.Equal(timing.LocationDate()))
	_, exact := timing.Instant()
	assert.False(t, exact)
	_, flagged := timing.Flag()
	assert.False(t, flagged)
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
	exDate := mustLocationDate(t, "2021-02-05", newYork)

	dividend, err := marketevent.NewDividend(aapl, exDate, 0.205)
	require.NoError(t, err)

	assert.Equal(t, aapl, dividend.Stock())
	assert.Equal(t, mustStockSubject(t, aaplFIGI), dividend.Subject())
	assertExDate(t, exDate, dividend.Timing())
	assert.Equal(t, "dividend", dividend.Kind())
	assert.InDelta(t, 0.205, dividend.Amount(), 0)
}

func TestNewDividendRejects(t *testing.T) {
	t.Parallel()

	aapl := mustFIGI(t, aaplFIGI)
	day := mustLocationDate(t, "2021-02-05", newYork)

	tests := []struct {
		name    string
		stock   identifier.FIGI
		exDate  civil.LocationDate
		amount  float64
		wantErr string
	}{
		{name: "zero stock", stock: identifier.FIGI{}, exDate: day, amount: 0.205, wantErr: "stock is missing"},
		{name: "missing ex-date", stock: aapl, amount: 0.205, wantErr: "ex-date is missing"},
		{name: "zero amount", stock: aapl, exDate: day, amount: 0, wantErr: "not positive and finite"},
		{name: "negative amount", stock: aapl, exDate: day, amount: -0.205, wantErr: "not positive and finite"},
		{name: "amount that is not a number", stock: aapl, exDate: day, amount: math.NaN(), wantErr: "not positive and finite"},
		{name: "infinite amount", stock: aapl, exDate: day, amount: math.Inf(1), wantErr: "not positive and finite"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := marketevent.NewDividend(tt.stock, tt.exDate, tt.amount)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestNewSplit(t *testing.T) {
	t.Parallel()

	aapl := mustFIGI(t, aaplFIGI)
	exDate := mustLocationDate(t, "2020-08-31", newYork)

	split, err := marketevent.NewSplit(aapl, exDate, 0.25)
	require.NoError(t, err)

	assert.Equal(t, aapl, split.Stock())
	assert.Equal(t, mustStockSubject(t, aaplFIGI), split.Subject())
	assertExDate(t, exDate, split.Timing())
	assert.Equal(t, "split", split.Kind())
	assert.InDelta(t, 0.25, split.Factor(), 0)
}

func TestNewSplitRejects(t *testing.T) {
	t.Parallel()

	aapl := mustFIGI(t, aaplFIGI)
	day := mustLocationDate(t, "2020-08-31", newYork)

	tests := []struct {
		name    string
		stock   identifier.FIGI
		exDate  civil.LocationDate
		factor  float64
		wantErr string
	}{
		{name: "zero stock", stock: identifier.FIGI{}, exDate: day, factor: 0.25, wantErr: "stock is missing"},
		{name: "missing ex-date", stock: aapl, factor: 0.25, wantErr: "ex-date is missing"},
		{name: "zero factor", stock: aapl, exDate: day, factor: 0, wantErr: "not positive and finite"},
		{name: "negative factor", stock: aapl, exDate: day, factor: -0.25, wantErr: "not positive and finite"},
		{name: "factor that is not a number", stock: aapl, exDate: day, factor: math.NaN(), wantErr: "not positive and finite"},
		{name: "infinite factor", stock: aapl, exDate: day, factor: math.Inf(1), wantErr: "not positive and finite"},
		{name: "factor of one", stock: aapl, exDate: day, factor: 1, wantErr: "factor 1 changes no price"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := marketevent.NewSplit(tt.stock, tt.exDate, tt.factor)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
