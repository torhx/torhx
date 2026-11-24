package marketevent_test

import (
	"iter"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talav/civil"
	"github.com/talav/country"

	"github.com/torhx/torhx/pkg/component/calendar"
	"github.com/torhx/torhx/pkg/component/marketevent"
)

// The dates of the schedule's events, and the dates the schedule covers.
const (
	dividendDay = "2021-02-05"
	busyDay     = "2021-02-10"
	auctionDay  = "2021-03-01"
	year2021    = "[2021-01-01,2022-01-01)"
)

// mustDateRange returns the range of dates written in s as
// [YYYY-MM-DD,YYYY-MM-DD), whose start is included and whose end is not.
func mustDateRange(t *testing.T, s string) civil.DateRange {
	t.Helper()

	r, err := civil.NewDateRangeFromString(s)
	require.NoError(t, err)

	return r
}

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

	economic := func(region marketevent.Region, date, kind string) marketevent.EconomicEvent {
		e, err := marketevent.NewEconomicEvent(region, mustDateTiming(t, date), kind)
		require.NoError(t, err)

		return e
	}

	earnings := func(figi string, timing marketevent.Timing) marketevent.Earnings {
		e, err := marketevent.NewEarnings(mustFIGI(t, figi), timing)
		require.NoError(t, err)

		return e
	}

	us := mustCountryRegion(t, country.US)

	afterMarket, err := marketevent.NewAfterMarketTiming(mustLocationDate(t, busyDay, newYork))
	require.NoError(t, err)

	dividend, err := marketevent.NewDividend(mustFIGI(t, aaplFIGI), mustLocationDate(t, dividendDay, newYork), 0.205)
	require.NoError(t, err)

	repurchase, err := marketevent.NewCompanyEvent(mustFIGI(t, aaplFIGI), mustDateTiming(t, busyDay), "buyback")
	require.NoError(t, err)

	s := schedule{
		aaplDividend: dividend,
		caCPI:        economic(mustCountryRegion(t, country.CA), busyDay, "CPI"),
		usCPI:        economic(us, busyDay, "CPI"),
		ecbDecision:  economic(mustCurrencyRegion(t, "EUR"), busyDay, "rate decision"),
		aaplEarnings: earnings(aaplFIGI, mustDateTiming(t, busyDay)),
		aaplBuyback:  buyback{CompanyEvent: repurchase, shares: 1e6},
		nvdaEarnings: earnings(nvdaFIGI, afterMarket),
		usAuction:    auction{EconomicEvent: economic(us, auctionDay, "Treasury auction"), term: "10-Year"},
		usGDP:        economic(us, auctionDay, "GDP"),
	}

	given := []marketevent.Event{
		s.nvdaEarnings, s.usAuction, s.aaplEarnings, s.ecbDecision, s.usCPI,
		s.aaplBuyback, s.usGDP, s.aaplDividend, s.caCPI,
	}

	s.events, err = marketevent.NewEvents(given, mustDateRange(t, year2021))
	require.NoError(t, err)

	return s
}

// all returns every event of the schedule in the order Events keeps: by
// date, then regions before stocks and countries before currency zones, each
// by its code, then as given.
func (s schedule) all() []marketevent.Event {
	return []marketevent.Event{
		s.aaplDividend,
		s.caCPI, s.usCPI, s.ecbDecision, s.aaplEarnings, s.aaplBuyback, s.nvdaEarnings,
		s.usAuction, s.usGDP,
	}
}

// assertEvents asserts that got holds the events of want, in order.
func assertEvents(t *testing.T, want []marketevent.Event, got marketevent.Events) {
	t.Helper()

	assert.Equal(t, len(want), got.Len())

	if len(want) == 0 {
		assert.Empty(t, slices.Collect(got.All()))

		return
	}

	assert.Equal(t, want, slices.Collect(got.All()))
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

// regional is any event that is about a region.
type regional interface {
	marketevent.Event

	// Region returns the region the event is about.
	Region() marketevent.Region
}

func TestNewEventsKeepsTheSlice(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)
	given := []marketevent.Event{s.usGDP, s.aaplDividend, s.usGDP}

	events, err := marketevent.NewEvents(given, mustDateRange(t, year2021))
	require.NoError(t, err)

	want := []marketevent.Event{s.aaplDividend, s.usGDP, s.usGDP}

	assert.Equal(t, want, given)
	assertEvents(t, want, events)
}

func TestNewEventsRejects(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)
	year := mustDateRange(t, year2021)

	tests := []struct {
		name     string
		events   []marketevent.Event
		coverage civil.DateRange
		wantErr  string
	}{
		{name: "zero coverage", events: []marketevent.Event{s.usCPI}, wantErr: "coverage is missing"},
		{name: "nil event", events: []marketevent.Event{nil}, coverage: year, wantErr: "event 0 is missing"},
		{
			name:     "zero timing",
			events:   []marketevent.Event{s.usCPI, auction{}},
			coverage: year,
			wantErr:  "event 1 has no timing",
		},
		{
			name:     "zero subject",
			events:   []marketevent.Event{subjectless{EconomicEvent: s.usCPI}},
			coverage: year,
			wantErr:  "event 0 has no subject",
		},
		{
			name:     "empty kind",
			events:   []marketevent.Event{kindless{EconomicEvent: s.usCPI}},
			coverage: year,
			wantErr:  "event 0 has no kind",
		},
		{
			name:     "dated before the coverage",
			events:   []marketevent.Event{s.usCPI},
			coverage: mustDateRange(t, "[2021-02-11,2022-01-01)"),
			wantErr:  "event 0 is dated 2021-02-10, outside [2021-02-11,2022-01-01)",
		},
		{
			name:     "dated on the end of the coverage, which is not covered",
			events:   []marketevent.Event{s.usCPI},
			coverage: mustDateRange(t, "[2021-01-01,2021-02-10)"),
			wantErr:  "event 0 is dated 2021-02-10, outside [2021-01-01,2021-02-10)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := marketevent.NewEvents(tt.events, tt.coverage)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestEventsCoverage(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)
	us := mustRegionSubject(t, mustCountryRegion(t, country.US))
	year := mustDateRange(t, year2021)
	busy := mustDateRange(t, "[2021-02-10,2021-02-11)")

	// The exclusive end of this one-day range falls in year 10000.
	last := mustDate(t, "9999-12-31")
	lastDay, err := civil.NewDateRange(last, last.AddDays(1))
	require.NoError(t, err)
	empty, err := marketevent.NewEvents(nil, lastDay)
	require.NoError(t, err)

	tests := []struct {
		name   string
		events marketevent.Events
		want   civil.DateRange
	}{
		{name: "as built", events: s.events, want: year},
		{name: "covered day without events at the year boundary", events: empty.On(last), want: lastDay},
		{name: "a subject", events: s.events.For(us), want: year},
		{name: "kinds", events: s.events.OfKind("CPI"), want: year},
		{name: "a date", events: s.events.On(mustDate(t, busyDay)), want: busy},
		{name: "a range", events: s.events.During(busy), want: busy},
		{
			name:   "a range wider than the coverage",
			events: s.events.During(mustDateRange(t, "[2020-01-01,2021-02-11)")),
			want:   mustDateRange(t, "[2021-01-01,2021-02-11)"),
		},
		{name: "a date outside the coverage", events: s.events.On(mustDate(t, "2022-02-10"))},
		{
			name:   "overlap at the end of coverage",
			events: s.events.During(mustDateRange(t, "[2021-12-31,2022-02-01)")),
			want:   mustDateRange(t, "[2021-12-31,2022-01-01)"),
		},
		{name: "widening a narrowed range", events: s.events.On(mustDate(t, busyDay)).During(year), want: busy},
		{name: "a range that starts where the coverage ends", events: s.events.During(mustDateRange(t, "[2022-01-01,2022-02-01)"))},
		{name: "the zero range", events: s.events.During(civil.DateRange{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.events.Coverage())
		})
	}
}

func TestZeroEvents(t *testing.T) {
	t.Parallel()

	var events marketevent.Events

	us := mustRegionSubject(t, mustCountryRegion(t, country.US))

	assert.False(t, events.Coverage().IsValid())
	assert.False(t, events.Has())
	assertEvents(t, nil, events)
	assertEvents(t, nil, events.On(mustDate(t, busyDay)))
	assertEvents(t, nil, events.During(mustDateRange(t, year2021)))
	assertEvents(t, nil, events.For(us))
	assertEvents(t, nil, events.OfKind("CPI"))
	assert.Empty(t, slices.Collect(events.Of[marketevent.Dividend]()))
}

func TestEventsOn(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)

	tests := []struct {
		name string
		date civil.Date
		want []marketevent.Event
	}{
		{name: "a date with several events, by subject", date: mustDate(t, busyDay), want: s.all()[1:7]},
		{name: "a date with one event", date: mustDate(t, dividendDay), want: []marketevent.Event{s.aaplDividend}},
		{name: "a date without events", date: mustDate(t, "2021-02-06")},
		{name: "a date outside the coverage", date: mustDate(t, "2022-02-10")},
		{name: "the zero date", date: civil.Date{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertEvents(t, tt.want, s.events.On(tt.date))
		})
	}
}

func TestEventsDuring(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)
	all := s.all()

	tests := []struct {
		name  string
		dates civil.DateRange
		want  []marketevent.Event
	}{
		{name: "the coverage", dates: mustDateRange(t, year2021), want: all},
		{name: "wider than the coverage", dates: mustDateRange(t, "[2020-01-01,2023-01-01)"), want: all},
		{name: "the start is included", dates: mustDateRange(t, "[2021-02-10,2021-03-02)"), want: all[1:]},
		{name: "the end is not", dates: mustDateRange(t, "[2021-02-05,2021-02-10)"), want: all[:1]},
		{name: "one date", dates: mustDateRange(t, "[2021-03-01,2021-03-02)"), want: []marketevent.Event{s.usAuction, s.usGDP}},
		{name: "dates without events", dates: mustDateRange(t, "[2021-04-01,2021-05-01)")},
		{name: "dates outside the coverage", dates: mustDateRange(t, "[2022-01-01,2022-02-01)")},
		{name: "the zero range", dates: civil.DateRange{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertEvents(t, tt.want, s.events.During(tt.dates))
		})
	}
}

func TestEventsWithTradingCalendar(t *testing.T) {
	t.Parallel()

	calendars, err := calendar.TradingCalendars()
	require.NoError(t, err)
	registry, err := calendar.NewRegistry(calendars)
	require.NoError(t, err)
	id, err := calendar.NewID("XNYS")
	require.NoError(t, err)
	xnys, err := registry.Get(id)
	require.NoError(t, err)

	// The Friday after Thanksgiving in 2021 closed at 13:00 New York time.
	at := time.Date(2021, time.November, 26, 18, 30, 0, 0, time.UTC)
	venue := xnys.Exchange()
	timing, err := marketevent.NewExactTimingForExchange(at, venue)
	require.NoError(t, err)
	report, err := marketevent.NewEarnings(mustFIGI(t, aaplFIGI), timing)
	require.NoError(t, err)

	// The event feed also covers Thanksgiving, when XNYS was closed.
	feed := mustDateRange(t, "[2021-11-25,2021-11-27)")
	events, err := marketevent.NewEvents([]marketevent.Event{report}, feed)
	require.NoError(t, err)
	shared := events.During(xnys.Coverage())
	assert.Equal(t, feed, shared.Coverage())

	friday := mustDate(t, "2021-11-26")
	assert.Equal(t, []civil.Date{friday}, xnys.TradingDaysBetween(shared.Coverage()))
	reports := slices.Collect(shared.On(friday).Of[marketevent.Earnings]())
	require.Len(t, reports, 1)

	instant, exact := reports[0].Timing().Instant()
	require.True(t, exact)
	assert.True(t, venue.IsRegularHours(instant))
	_, open, err := xnys.SessionAt(instant)
	require.NoError(t, err)
	assert.False(t, open)

	_, flagged := reports[0].Timing().Flag()
	assert.False(t, flagged)
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
		events marketevent.Events
		want   []marketevent.Event
	}{
		{name: "a stock", events: s.events.For(aapl), want: []marketevent.Event{s.aaplDividend, s.aaplEarnings, s.aaplBuyback}},
		{name: "a country", events: s.events.For(us), want: []marketevent.Event{s.usCPI, s.usAuction, s.usGDP}},
		{name: "a currency zone", events: s.events.For(euroArea), want: []marketevent.Event{s.ecbDecision}},
		{name: "a subject without events", events: s.events.For(japan)},
		{name: "the zero subject", events: s.events.For(marketevent.Subject{})},
		{name: "the same subject twice", events: s.events.For(us).For(us), want: []marketevent.Event{s.usCPI, s.usAuction, s.usGDP}},
		{name: "two subjects", events: s.events.For(us).For(canada)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertEvents(t, tt.want, tt.events)
		})
	}
}

func TestEventsOfKindAndHas(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)

	tests := []struct {
		name  string
		kinds []string
		want  []marketevent.Event
	}{
		{name: "one kind", kinds: []string{"CPI"}, want: []marketevent.Event{s.caCPI, s.usCPI}},
		{
			name:  "any of several kinds",
			kinds: []string{"earnings", "CPI"},
			want:  []marketevent.Event{s.caCPI, s.usCPI, s.aaplEarnings, s.nvdaEarnings},
		},
		{name: "a kind it lacks", kinds: []string{"FOMC"}},
		{name: "no kinds, which is every kind", want: s.all()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertEvents(t, tt.want, s.events.OfKind(tt.kinds...))
			assert.Equal(t, len(tt.want) > 0, s.events.Has(tt.kinds...))
		})
	}
}

func TestEventsNarrowInAnyOrder(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)
	us := mustRegionSubject(t, mustCountryRegion(t, country.US))
	busy := mustDate(t, busyDay)
	untilMarch := mustDateRange(t, "[2021-01-01,2021-03-01)")
	reports := []marketevent.Event{s.aaplEarnings, s.nvdaEarnings}

	tests := []struct {
		name   string
		events marketevent.Events
		want   []marketevent.Event
	}{
		{name: "a subject, then dates", events: s.events.For(us).During(untilMarch), want: []marketevent.Event{s.usCPI}},
		{name: "dates, then a subject", events: s.events.During(untilMarch).For(us), want: []marketevent.Event{s.usCPI}},
		{name: "a subject, then kinds", events: s.events.For(us).OfKind("GDP"), want: []marketevent.Event{s.usGDP}},
		{name: "kinds, then a subject", events: s.events.OfKind("GDP").For(us), want: []marketevent.Event{s.usGDP}},
		{name: "dates, then kinds", events: s.events.During(untilMarch).OfKind("earnings"), want: reports},
		{name: "kinds, then dates", events: s.events.OfKind("earnings").During(untilMarch), want: reports},
		{name: "all three", events: s.events.For(us).On(busy).OfKind("CPI"), want: []marketevent.Event{s.usCPI}},
		{name: "all three, the other way", events: s.events.OfKind("CPI").On(busy).For(us), want: []marketevent.Event{s.usCPI}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertEvents(t, tt.want, tt.events)
		})
	}

	// Narrowing leaves the Events it narrows as it is.
	assertEvents(t, s.all(), s.events)
}

func TestEventsHasOnADate(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)
	nvda := s.events.For(mustStockSubject(t, nvdaFIGI))
	busy := mustDate(t, busyDay)

	assert.True(t, nvda.On(busy).Has("earnings"))
	assert.False(t, nvda.On(busy).Has("dividend"))
	assert.True(t, nvda.On(busy).Has())
	assert.False(t, nvda.On(mustDate(t, dividendDay)).Has())
	assert.True(t, s.events.On(busy).Has("FOMC", "CPI"))
}

func TestEventsReadATypeThatEmbedsNothing(t *testing.T) {
	t.Parallel()

	day := mustDate(t, busyDay)
	aapl := mustStockSubject(t, aaplFIGI)
	expiry := lockup{subject: aapl, timing: mustDateTiming(t, busyDay)}

	events, err := marketevent.NewEvents([]marketevent.Event{expiry}, mustDateRange(t, "[2021-02-10,2021-02-11)"))
	require.NoError(t, err)

	assert.True(t, events.For(aapl).On(day).Has("lockup expiry"))
	assert.Equal(t, []lockup{expiry}, slices.Collect(events.On(day).Of[lockup]()))
}

func TestEventsOf(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)

	assert.Equal(t, []marketevent.Dividend{s.aaplDividend}, slices.Collect(s.events.Of[marketevent.Dividend]()))
	assert.Equal(t,
		[]marketevent.Earnings{s.aaplEarnings, s.nvdaEarnings},
		slices.Collect(s.events.Of[marketevent.Earnings]()),
	)

	// A concrete type keeps the events of exactly that type, so a program's
	// type that embeds EconomicEvent is left out, and so is its pointer.
	assert.Equal(t,
		[]marketevent.EconomicEvent{s.caCPI, s.usCPI, s.ecbDecision, s.usGDP},
		slices.Collect(s.events.Of[marketevent.EconomicEvent]()),
	)
	assert.Equal(t, []auction{s.usAuction}, slices.Collect(s.events.Of[auction]()))
	assert.Empty(t, slices.Collect(s.events.Of[*auction]()))

	// An interface keeps every event that implements it.
	assert.Equal(t,
		[]regional{s.caCPI, s.usCPI, s.ecbDecision, s.usAuction, s.usGDP},
		slices.Collect(s.events.Of[regional]()),
	)
}

func TestEventsOfStopsWithItsCaller(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)

	next, stop := iter.Pull(s.events.Of[marketevent.Earnings]())
	first, ok := next()

	stop()

	require.True(t, ok)
	assert.Equal(t, s.aaplEarnings, first)
}
