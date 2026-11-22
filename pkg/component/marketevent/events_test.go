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

	afterMarket, err := marketevent.NewAfterMarketTiming(mustDate(t, busyDay), mustLocation(t, "America/New_York"))
	require.NoError(t, err)

	dividend, err := marketevent.NewDividend(mustFIGI(t, aaplFIGI), mustDateTiming(t, dividendDay), 0.205)
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

	s.events, err = marketevent.NewEvents(given, mustDate(t, firstDay), mustDate(t, lastDay))
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

	events, err := marketevent.NewEvents(given, mustDate(t, firstDay), mustDate(t, lastDay))
	require.NoError(t, err)

	want := []marketevent.Event{s.aaplDividend, s.usGDP, s.usGDP}

	assert.Equal(t, want, given)
	assertEvents(t, want, events)
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

	s := mustSchedule(t)
	us := mustRegionSubject(t, mustCountryRegion(t, country.US))
	first := mustDate(t, firstDay)
	last := mustDate(t, lastDay)
	dividend := mustDate(t, dividendDay)
	busy := mustDate(t, busyDay)

	tests := []struct {
		name     string
		events   marketevent.Events
		wantFrom civil.Date
		wantTo   civil.Date
	}{
		{name: "as built", events: s.events, wantFrom: first, wantTo: last},
		{name: "a subject", events: s.events.For(us), wantFrom: first, wantTo: last},
		{name: "kinds", events: s.events.OfKind("CPI"), wantFrom: first, wantTo: last},
		{name: "a date", events: s.events.On(busy), wantFrom: busy, wantTo: busy},
		{name: "a range", events: s.events.Between(dividend, busy), wantFrom: dividend, wantTo: busy},
		{
			name:     "a range wider than the coverage",
			events:   s.events.Between(mustDate(t, "2020-01-01"), busy),
			wantFrom: first,
			wantTo:   busy,
		},
		{name: "a date outside the coverage", events: s.events.On(mustDate(t, "2022-02-10"))},
		{name: "a range that ends before it starts", events: s.events.Between(last, first)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			from, to := tt.events.Coverage()

			assert.Equal(t, tt.wantFrom, from)
			assert.Equal(t, tt.wantTo, to)
		})
	}
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
	assert.False(t, events.Has())
	assertEvents(t, nil, events)
	assertEvents(t, nil, events.On(first))
	assertEvents(t, nil, events.Between(first, last))
	assertEvents(t, nil, events.For(us))
	assertEvents(t, nil, events.OfKind("CPI"))
	assert.Empty(t, slices.Collect(events.Of[marketevent.Dividend]()))
}

func TestEventsOn(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t)

	tests := []struct {
		name string
		date string
		want []marketevent.Event
	}{
		{name: "a date with several events, by subject", date: busyDay, want: s.all()[1:7]},
		{name: "a date with one event", date: dividendDay, want: []marketevent.Event{s.aaplDividend}},
		{name: "a date without events", date: "2021-02-06"},
		{name: "a date outside the coverage", date: "2022-02-10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertEvents(t, tt.want, s.events.On(mustDate(t, tt.date)))
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
		want []marketevent.Event
	}{
		{name: "the coverage", from: firstDay, to: lastDay, want: all},
		{name: "wider than the coverage", from: "2020-01-01", to: "2022-12-31", want: all},
		{name: "both ends included", from: dividendDay, to: busyDay, want: all[:7]},
		{name: "one date", from: auctionDay, to: auctionDay, want: []marketevent.Event{s.usAuction, s.usGDP}},
		{name: "dates without events", from: "2021-04-01", to: "2021-04-30"},
		{name: "from after to", from: lastDay, to: firstDay},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertEvents(t, tt.want, s.events.Between(mustDate(t, tt.from), mustDate(t, tt.to)))
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
	first := mustDate(t, firstDay)
	busy := mustDate(t, busyDay)
	reports := []marketevent.Event{s.aaplEarnings, s.nvdaEarnings}

	tests := []struct {
		name   string
		events marketevent.Events
		want   []marketevent.Event
	}{
		{name: "a subject, then dates", events: s.events.For(us).Between(first, busy), want: []marketevent.Event{s.usCPI}},
		{name: "dates, then a subject", events: s.events.Between(first, busy).For(us), want: []marketevent.Event{s.usCPI}},
		{name: "a subject, then kinds", events: s.events.For(us).OfKind("GDP"), want: []marketevent.Event{s.usGDP}},
		{name: "kinds, then a subject", events: s.events.OfKind("GDP").For(us), want: []marketevent.Event{s.usGDP}},
		{name: "dates, then kinds", events: s.events.Between(first, busy).OfKind("earnings"), want: reports},
		{name: "kinds, then dates", events: s.events.OfKind("earnings").Between(first, busy), want: reports},
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

	events, err := marketevent.NewEvents([]marketevent.Event{expiry}, day, day)
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
