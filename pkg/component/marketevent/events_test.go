package marketevent_test

import (
	"iter"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talav/civil"
	"github.com/talav/country"

	"github.com/torhx/torhx/pkg/component/marketevent"
)

// The dates of the schedule's events, and the dates the schedule covers.
const (
	dividendDay = "2021-02-05"
	busyDay     = "2021-02-10"
	auctionDay  = "2021-03-01"
	firstDay    = "2021-01-01"
	lastDay     = "2021-12-31"
)

// schedule holds the events the read tests are about, and the Events of
// them, which cover 2021.
type schedule struct {
	// events reads every event of the schedule.
	events marketevent.Events

	// aaplDividend goes ex on dividendDay.
	aaplDividend marketevent.Dividend

	// caCPI is Canada's CPI release of busyDay.
	caCPI marketevent.EconomicEvent

	// usCPI is the US CPI release of busyDay.
	usCPI marketevent.EconomicEvent

	// ecbDecision is the euro area's rate decision of busyDay.
	ecbDecision marketevent.EconomicEvent

	// aaplEarnings is AAPL's report of busyDay.
	aaplEarnings marketevent.Earnings

	// aaplBuyback is AAPL's buyback of busyDay, given after its report.
	aaplBuyback buyback

	// nvdaEarnings is NVDA's report of busyDay, after the market closes.
	nvdaEarnings marketevent.Earnings

	// usAuction is the US Treasury auction of auctionDay.
	usAuction auction

	// usGDP is the US GDP release of auctionDay, given after the auction.
	usGDP marketevent.EconomicEvent
}

// mustSchedule returns the schedule, whose events are given to NewEvents out
// of order.
func mustSchedule(t *testing.T) schedule {
	t.Helper()

	economic := func(region marketevent.Region, date, kind, detail string) marketevent.EconomicEvent {
		e, err := marketevent.NewEconomicEvent(region, mustDateTiming(t, date), kind, detail)
		require.NoError(t, err)

		return e
	}

	earnings := func(figi string, timing marketevent.Timing) marketevent.Earnings {
		e, err := marketevent.NewEarnings(mustFIGI(t, figi), timing)
		require.NoError(t, err)

		return e
	}

	us := mustCountryRegion(t, country.US)

	afterMarket, err := marketevent.NewAfterMarketTiming(mustDate(t, busyDay), mustLocation(t, "America/New_York"))
	require.NoError(t, err)

	dividend, err := marketevent.NewDividend(mustFIGI(t, aaplFIGI), mustDateTiming(t, dividendDay), 0.205)
	require.NoError(t, err)

	repurchase, err := marketevent.NewCompanyEvent(mustFIGI(t, aaplFIGI), mustDateTiming(t, busyDay), "buyback")
	require.NoError(t, err)

	s := schedule{
		aaplDividend: dividend,
		caCPI:        economic(mustCountryRegion(t, country.CA), busyDay, "CPI", ""),
		usCPI:        economic(us, busyDay, "CPI", ""),
		ecbDecision:  economic(mustCurrencyRegion(t, "EUR"), busyDay, "rate decision", ""),
		aaplEarnings: earnings(aaplFIGI, mustDateTiming(t, busyDay)),
		aaplBuyback:  buyback{CompanyEvent: repurchase, shares: 1e6},
		nvdaEarnings: earnings(nvdaFIGI, afterMarket),
		usAuction:    auction{EconomicEvent: economic(us, auctionDay, "Treasury auction", ""), term: "10-Year"},
		usGDP:        economic(us, auctionDay, "GDP", "advance"),
	}

	given := []marketevent.Event{
		s.nvdaEarnings, s.usAuction, s.aaplEarnings, s.ecbDecision, s.usCPI,
		s.aaplBuyback, s.usGDP, s.aaplDividend, s.caCPI,
	}

	s.events, err = marketevent.NewEvents(given, mustDate(t, firstDay), mustDate(t, lastDay))
	require.NoError(t, err)

	return s
}

// all returns every event of the schedule in the order Events keeps: by
// date, then regions before stocks and countries before currency zones, each
// by its code, then as given.
func (s schedule) all() marketevent.Selection {
	return marketevent.Selection{
		s.aaplDividend,
		s.caCPI, s.usCPI, s.ecbDecision, s.aaplEarnings, s.aaplBuyback, s.nvdaEarnings,
		s.usAuction, s.usGDP,
	}
}

// assertSelection asserts that got holds the events of want, in order. A
// read that finds nothing may return an empty Selection or a nil one.
func assertSelection(t *testing.T, want, got marketevent.Selection) {
	t.Helper()

	if len(want) == 0 {
		assert.Empty(t, got)

		return
	}

	assert.Equal(t, want, got)
}

// subjectless is a program's type that replaces Subject with one that
// returns the zero value.
type subjectless struct{ marketevent.EconomicEvent }

// Subject returns the zero subject.
func (subjectless) Subject() marketevent.Subject { return marketevent.Subject{} }

// kindless is a program's type that replaces Kind with one that returns no
// kind.
type kindless struct{ marketevent.EconomicEvent }

// Kind returns no kind.
func (kindless) Kind() string { return "" }

// detailed is any event that has a detail.
type detailed interface {
	marketevent.Event

	// Detail returns what narrows the event's kind.
	Detail() string
}

func TestNewEventsKeepsTheSlice(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)
	given := []marketevent.Event{s.usGDP, s.aaplDividend, s.usGDP}

	events, err := marketevent.NewEvents(given, mustDate(t, firstDay), mustDate(t, lastDay))
	require.NoError(t, err)

	want := marketevent.Selection{s.aaplDividend, s.usGDP, s.usGDP}

	assert.Equal(t, want, marketevent.Selection(given))
	assert.Equal(t, want, events.Between(mustDate(t, firstDay), mustDate(t, lastDay)))
}

func TestNewEventsRejects(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)
	first := mustDate(t, firstDay)
	last := mustDate(t, lastDay)
	busy := mustDate(t, busyDay)

	tests := []struct {
		name    string
		events  []marketevent.Event
		from    civil.Date
		to      civil.Date
		wantErr string
	}{
		{name: "from after to", from: last, to: first, wantErr: "are not a range"},
		{name: "zero from", from: civil.Date{}, to: last, wantErr: "are not a range"},
		{name: "zero to", from: first, to: civil.Date{}, wantErr: "are not a range"},
		{name: "nil event", events: []marketevent.Event{nil}, from: first, to: last, wantErr: "event 0 is missing"},
		{
			name:    "zero timing",
			events:  []marketevent.Event{s.usCPI, auction{}},
			from:    first,
			to:      last,
			wantErr: "event 1 has no timing",
		},
		{
			name:    "zero subject",
			events:  []marketevent.Event{subjectless{EconomicEvent: s.usCPI}},
			from:    first,
			to:      last,
			wantErr: "event 0 has no subject",
		},
		{
			name:    "empty kind",
			events:  []marketevent.Event{kindless{EconomicEvent: s.usCPI}},
			from:    first,
			to:      last,
			wantErr: "event 0 has no kind",
		},
		{
			name:    "dated before the coverage",
			events:  []marketevent.Event{s.usCPI},
			from:    busy.AddDays(1),
			to:      last,
			wantErr: "event 0 is dated 2021-02-10, outside 2021-02-11 to 2021-12-31",
		},
		{
			name:    "dated after the coverage",
			events:  []marketevent.Event{s.usCPI},
			from:    first,
			to:      busy.AddDays(-1),
			wantErr: "event 0 is dated 2021-02-10, outside 2021-01-01 to 2021-02-09",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := marketevent.NewEvents(tt.events, tt.from, tt.to)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestEventsCoverage(t *testing.T) {
	t.Parallel()

	from, to := mustSchedule(t).events.Coverage()

	assert.Equal(t, mustDate(t, firstDay), from)
	assert.Equal(t, mustDate(t, lastDay), to)
}

func TestZeroEvents(t *testing.T) {
	t.Parallel()

	var events marketevent.Events

	from, to := events.Coverage()
	first := mustDate(t, firstDay)
	last := mustDate(t, lastDay)
	us := mustRegionSubject(t, mustCountryRegion(t, country.US))

	assert.False(t, from.IsValid())
	assert.False(t, to.IsValid())
	assert.False(t, events.Has(first))
	assert.Empty(t, events.Find(first))
	assert.Empty(t, events.Between(first, last))
	assert.Empty(t, events.For(us).Between(first, last))
}

func TestEventsFindAndHas(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)

	tests := []struct {
		name  string
		date  string
		kinds []string
		want  marketevent.Selection
	}{
		{
			name: "every kind, by subject",
			date: busyDay,
			want: marketevent.Selection{s.caCPI, s.usCPI, s.ecbDecision, s.aaplEarnings, s.aaplBuyback, s.nvdaEarnings},
		},
		{name: "one kind", date: busyDay, kinds: []string{"CPI"}, want: marketevent.Selection{s.caCPI, s.usCPI}},
		{
			name:  "any of several kinds",
			date:  busyDay,
			kinds: []string{"earnings", "CPI"},
			want:  marketevent.Selection{s.caCPI, s.usCPI, s.aaplEarnings, s.nvdaEarnings},
		},
		{name: "a kind the date lacks", date: busyDay, kinds: []string{"FOMC"}},
		{name: "a date with one event", date: dividendDay, want: marketevent.Selection{s.aaplDividend}},
		{name: "a date without events", date: "2021-02-06"},
		{name: "a date outside the coverage", date: "2022-02-10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			date := mustDate(t, tt.date)

			assertSelection(t, tt.want, s.events.Find(date, tt.kinds...))
			assert.Equal(t, len(tt.want) > 0, s.events.Has(date, tt.kinds...))
		})
	}
}

func TestEventsBetween(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)
	all := s.all()

	tests := []struct {
		name string
		from string
		to   string
		want marketevent.Selection
	}{
		{name: "the coverage", from: firstDay, to: lastDay, want: all},
		{name: "wider than the coverage", from: "2020-01-01", to: "2022-12-31", want: all},
		{name: "both ends included", from: dividendDay, to: busyDay, want: all[:7]},
		{name: "one date", from: auctionDay, to: auctionDay, want: marketevent.Selection{s.usAuction, s.usGDP}},
		{name: "dates without events", from: "2021-04-01", to: "2021-04-30"},
		{name: "from after to", from: lastDay, to: firstDay},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertSelection(t, tt.want, s.events.Between(mustDate(t, tt.from), mustDate(t, tt.to)))
		})
	}
}

func TestEventsFor(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)
	us := mustRegionSubject(t, mustCountryRegion(t, country.US))
	canada := mustRegionSubject(t, mustCountryRegion(t, country.CA))
	japan := mustRegionSubject(t, mustCountryRegion(t, country.JP))
	euroArea := mustRegionSubject(t, mustCurrencyRegion(t, "EUR"))
	aapl := mustStockSubject(t, aaplFIGI)

	tests := []struct {
		name   string
		reader marketevent.Reader
		want   marketevent.Selection
	}{
		{name: "a stock", reader: s.events.For(aapl), want: marketevent.Selection{s.aaplDividend, s.aaplEarnings, s.aaplBuyback}},
		{name: "a country", reader: s.events.For(us), want: marketevent.Selection{s.usCPI, s.usAuction, s.usGDP}},
		{name: "a currency zone", reader: s.events.For(euroArea), want: marketevent.Selection{s.ecbDecision}},
		{name: "a subject without events", reader: s.events.For(japan)},
		{name: "the zero subject", reader: s.events.For(marketevent.Subject{})},
		{name: "the same subject twice", reader: s.events.For(us).For(us), want: marketevent.Selection{s.usCPI, s.usAuction, s.usGDP}},
		{name: "two subjects", reader: s.events.For(us).For(canada)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertSelection(t, tt.want, tt.reader.Between(mustDate(t, firstDay), mustDate(t, lastDay)))
		})
	}
}

func TestNarrowedReadsByDate(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)
	busy := mustDate(t, busyDay)
	aapl := s.events.For(mustStockSubject(t, aaplFIGI))
	nvda := s.events.For(mustStockSubject(t, nvdaFIGI))

	assert.True(t, nvda.Has(busy, "earnings"))
	assert.False(t, nvda.Has(busy, "dividend"))
	assert.False(t, nvda.Has(mustDate(t, dividendDay)))
	assertSelection(t, marketevent.Selection{s.aaplEarnings}, aapl.Find(busy, "earnings"))
	assertSelection(t, marketevent.Selection{s.aaplEarnings, s.aaplBuyback}, aapl.Find(busy))

	// Narrowing leaves the Events it narrows as it is.
	assertSelection(t, s.all(), s.events.Between(mustDate(t, firstDay), mustDate(t, lastDay)))
}

func TestSelectionOf(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)
	all := s.all()

	assert.Equal(t, []marketevent.Dividend{s.aaplDividend}, slices.Collect(all.Of[marketevent.Dividend]()))
	assert.Equal(t, []marketevent.Earnings{s.aaplEarnings, s.nvdaEarnings}, slices.Collect(all.Of[marketevent.Earnings]()))

	// A concrete type keeps the events of exactly that type, so a program's
	// type that embeds EconomicEvent is left out, and so is its pointer.
	assert.Equal(t,
		[]marketevent.EconomicEvent{s.caCPI, s.usCPI, s.ecbDecision, s.usGDP},
		slices.Collect(all.Of[marketevent.EconomicEvent]()),
	)
	assert.Equal(t, []auction{s.usAuction}, slices.Collect(all.Of[auction]()))
	assert.Empty(t, slices.Collect(all.Of[*auction]()))

	// An interface keeps every event that implements it.
	assert.Equal(t,
		[]detailed{s.caCPI, s.usCPI, s.ecbDecision, s.usAuction, s.usGDP},
		slices.Collect(all.Of[detailed]()),
	)
}

func TestSelectionOfStopsWithItsCaller(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)

	next, stop := iter.Pull(s.all().Of[marketevent.Earnings]())
	first, ok := next()

	stop()

	require.True(t, ok)
	assert.Equal(t, s.aaplEarnings, first)
}
