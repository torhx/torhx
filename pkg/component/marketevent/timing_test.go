package marketevent_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talav/civil"

	"github.com/torhx/torhx/pkg/component/marketevent"
)

// mustDate returns the date written in s as YYYY-MM-DD.
func mustDate(t *testing.T, s string) civil.Date {
	t.Helper()

	d, err := civil.NewDateFromString(s)
	require.NoError(t, err)

	return d
}

// mustLocation returns the time zone that the IANA database calls name.
func mustLocation(t *testing.T, name string) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation(name)
	require.NoError(t, err)

	return loc
}

func TestNewExactTiming(t *testing.T) {
	t.Parallel()

	newYork := mustLocation(t, "America/New_York")
	tokyo := mustLocation(t, "Asia/Tokyo")

	// 01:30 UTC on 2021-02-05 is 20:30 on 2021-02-04 in New York and 10:30
	// on 2021-02-05 in Tokyo.
	at := time.Date(2021, time.February, 5, 1, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		at       time.Time
		loc      *time.Location
		wantDate civil.Date
	}{
		{name: "UTC", at: at, loc: time.UTC, wantDate: mustDate(t, "2021-02-05")},
		{name: "a day earlier in New York", at: at, loc: newYork, wantDate: mustDate(t, "2021-02-04")},
		{name: "dated in loc, not in the instant's zone", at: at.In(tokyo), loc: newYork, wantDate: mustDate(t, "2021-02-04")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			timing, err := marketevent.NewExactTiming(tt.at, tt.loc)
			require.NoError(t, err)

			instant, exact := timing.Instant()
			require.True(t, exact)

			assert.True(t, instant.Equal(at))
			assert.Equal(t, tt.loc, instant.Location())
			assert.Equal(t, tt.wantDate, timing.Date())
			assert.Equal(t, tt.loc, timing.Location())
			assert.True(t, timing.IsValid())

			_, flagged := timing.Flag()
			assert.False(t, flagged)
		})
	}
}

func TestNewExactTimingRejects(t *testing.T) {
	t.Parallel()

	at := time.Date(2021, time.February, 5, 13, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		at      time.Time
		loc     *time.Location
		wantErr string
	}{
		{name: "zero time", at: time.Time{}, loc: time.UTC, wantErr: "instant is missing"},
		{name: "no zone", at: at, loc: nil, wantErr: "time zone is missing"},
		{name: "local zone", at: at, loc: time.Local, wantErr: "the machine's local zone"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := marketevent.NewExactTiming(tt.at, tt.loc)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestNewDateTimings(t *testing.T) {
	t.Parallel()

	newYork := mustLocation(t, "America/New_York")
	day := mustDate(t, "2021-02-05")

	tests := []struct {
		name        string
		build       func(civil.Date, *time.Location) (marketevent.Timing, error)
		wantFlag    marketevent.Flag
		wantFlagged bool
	}{
		{name: "no flag", build: marketevent.NewDateTiming},
		{name: "before market", build: marketevent.NewBeforeMarketTiming, wantFlag: marketevent.FlagBeforeMarket, wantFlagged: true},
		{name: "after market", build: marketevent.NewAfterMarketTiming, wantFlag: marketevent.FlagAfterMarket, wantFlagged: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			timing, err := tt.build(day, newYork)
			require.NoError(t, err)

			_, exact := timing.Instant()
			flag, flagged := timing.Flag()

			assert.Equal(t, day, timing.Date())
			assert.Equal(t, newYork, timing.Location())
			assert.False(t, exact)
			assert.Equal(t, tt.wantFlag, flag)
			assert.Equal(t, tt.wantFlagged, flagged)
			assert.True(t, timing.IsValid())

			_, err = tt.build(civil.Date{}, newYork)
			require.ErrorContains(t, err, "date is missing")

			_, err = tt.build(day, nil)
			require.ErrorContains(t, err, "time zone is missing")

			_, err = tt.build(day, time.Local)
			require.ErrorContains(t, err, "the machine's local zone")
		})
	}

	assert.False(t, marketevent.Timing{}.IsValid())
}
