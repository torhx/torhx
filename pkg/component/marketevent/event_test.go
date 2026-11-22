package marketevent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talav/country"
	"github.com/torhx/identifier"

	"github.com/torhx/torhx/pkg/component/marketevent"
)

// mustDateTiming returns the timing of the date written in s as YYYY-MM-DD,
// in New York and without a flag.
func mustDateTiming(t *testing.T, s string) marketevent.Timing {
	t.Helper()

	timing, err := marketevent.NewDateTiming(mustDate(t, s), mustLocation(t, "America/New_York"))
	require.NoError(t, err)

	return timing
}

// auction is a program's own type of economic event.
type auction struct {
	marketevent.EconomicEvent

	// term is the term of the security on offer.
	term string
}

// buyback is a program's own type of company event.
type buyback struct {
	marketevent.CompanyEvent

	// shares is the number of shares the company buys back.
	shares float64
}

// lockup is a program's own type of event that embeds neither built-in
// type and has the methods of an Event itself: the end of a lock-up on a
// stock's shares.
type lockup struct {
	// subject is the stock whose shares the lock-up holds.
	subject marketevent.Subject

	// timing is when the lock-up ends.
	timing marketevent.Timing
}

// Timing returns when the lock-up ends.
func (l lockup) Timing() marketevent.Timing { return l.timing }

// Kind returns "lockup expiry".
func (l lockup) Kind() string { return "lockup expiry" }

// Subject returns the stock whose shares the lock-up holds.
func (l lockup) Subject() marketevent.Subject { return l.subject }

func TestNewEconomicEvent(t *testing.T) {
	t.Parallel()

	us := mustCountryRegion(t, country.US)
	timing := mustDateTiming(t, "2021-02-05")

	tests := []struct {
		name   string
		kind   string
		detail string
	}{
		{name: "kind alone", kind: "CPI"},
		{name: "kind and detail", kind: "GDP", detail: "advance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			event, err := marketevent.NewEconomicEvent(us, timing, tt.kind, tt.detail)
			require.NoError(t, err)

			assert.Equal(t, us, event.Region())
			assert.Equal(t, mustRegionSubject(t, us), event.Subject())
			assert.Equal(t, timing, event.Timing())
			assert.Equal(t, tt.kind, event.Kind())
			assert.Equal(t, tt.detail, event.Detail())
		})
	}
}

func TestNewEconomicEventRejects(t *testing.T) {
	t.Parallel()

	us := mustCountryRegion(t, country.US)
	timing := mustDateTiming(t, "2021-02-05")

	tests := []struct {
		name    string
		region  marketevent.Region
		timing  marketevent.Timing
		kind    string
		wantErr string
	}{
		{name: "zero region", region: marketevent.Region{}, timing: timing, kind: "CPI", wantErr: "region is missing"},
		{name: "zero timing", region: us, timing: marketevent.Timing{}, kind: "CPI", wantErr: "timing is missing"},
		{name: "empty kind", region: us, timing: timing, kind: "", wantErr: "kind is missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := marketevent.NewEconomicEvent(tt.region, tt.timing, tt.kind, "")
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestNewCompanyEvent(t *testing.T) {
	t.Parallel()

	aapl := mustFIGI(t, aaplFIGI)
	timing := mustDateTiming(t, "2020-08-31")

	event, err := marketevent.NewCompanyEvent(aapl, timing, "buyback")
	require.NoError(t, err)

	assert.Equal(t, aapl, event.Stock())
	assert.Equal(t, mustStockSubject(t, aaplFIGI), event.Subject())
	assert.Equal(t, timing, event.Timing())
	assert.Equal(t, "buyback", event.Kind())
}

func TestNewCompanyEventRejects(t *testing.T) {
	t.Parallel()

	aapl := mustFIGI(t, aaplFIGI)
	timing := mustDateTiming(t, "2021-02-05")

	tests := []struct {
		name    string
		stock   identifier.FIGI
		timing  marketevent.Timing
		kind    string
		wantErr string
	}{
		{name: "zero stock", stock: identifier.FIGI{}, timing: timing, kind: "buyback", wantErr: "stock is missing"},
		{name: "zero timing", stock: aapl, timing: marketevent.Timing{}, kind: "buyback", wantErr: "timing is missing"},
		{name: "empty kind", stock: aapl, timing: timing, kind: "", wantErr: "kind is missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := marketevent.NewCompanyEvent(tt.stock, tt.timing, tt.kind)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestProgramTypeIsAnEvent(t *testing.T) {
	t.Parallel()

	us := mustCountryRegion(t, country.US)
	timing := mustDateTiming(t, "2021-02-05")

	economic, err := marketevent.NewEconomicEvent(us, timing, "Treasury auction", "")
	require.NoError(t, err)

	company, err := marketevent.NewCompanyEvent(mustFIGI(t, aaplFIGI), timing, "buyback")
	require.NoError(t, err)

	tests := []struct {
		name        string
		event       marketevent.Event
		wantKind    string
		wantSubject marketevent.Subject
	}{
		{
			name:        "embeds an economic event",
			event:       auction{EconomicEvent: economic, term: "10-Year"},
			wantKind:    "Treasury auction",
			wantSubject: mustRegionSubject(t, us),
		},
		{
			name:        "embeds a company event",
			event:       buyback{CompanyEvent: company, shares: 1e6},
			wantKind:    "buyback",
			wantSubject: mustStockSubject(t, aaplFIGI),
		},
		{
			name:        "has the methods itself",
			event:       lockup{subject: mustStockSubject(t, aaplFIGI), timing: timing},
			wantKind:    "lockup expiry",
			wantSubject: mustStockSubject(t, aaplFIGI),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, timing, tt.event.Timing())
			assert.Equal(t, tt.wantKind, tt.event.Kind())
			assert.Equal(t, tt.wantSubject, tt.event.Subject())
		})
	}
}
