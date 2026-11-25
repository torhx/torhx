package marketevent_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talav/civil"

	"github.com/torhx/torhx/pkg/component/marketevent"
)

// newYork is the IANA name of the time zone most test timings are in.
const newYork = "America/New_York"

// mustDate returns the date written in s as YYYY-MM-DD.
func mustDate(t *testing.T, s string) civil.Date {
	t.Helper()

	d, err := civil.NewDateFromString(s)
	require.NoError(t, err)

	return d
}

// mustLocationDate returns the date written in s as YYYY-MM-DD in zone.
func mustLocationDate(t *testing.T, s, zone string) civil.LocationDate {
	t.Helper()

	day, err := civil.NewLocationDate(mustDate(t, s), zone)
	require.NoError(t, err)

	return day
}

func TestNewExactTiming(t *testing.T) {
	t.Parallel()

	// 01:30 UTC on February 5 is 20:30 on February 4 in New York.
	at := time.Date(2021, time.February, 5, 1, 30, 0, 123, time.UTC)
	previous := mustLocationDate(t, "2021-02-04", newYork)
	repeated := mustLocationDate(t, "2021-11-07", newYork)
	nextYear, err := civil.NewLocationDate(mustDate(t, "9999-12-31").AddDays(1), "UTC")
	require.NoError(t, err)

	tests := []struct {
		name    string
		at      time.Time
		wantDay civil.LocationDate
	}{
		{name: "UTC", at: at, wantDay: mustLocationDate(t, "2021-02-05", "UTC")},
		{name: "a day earlier in New York", at: at.In(previous.Location()), wantDay: previous},
		{name: "named fixed zone with the correct offset", at: at.In(time.FixedZone(newYork, -5*60*60)), wantDay: previous},
		// Both occurrences of 01:30 have the same local date and clock time.
		{name: "first 01:30", at: time.Date(2021, time.November, 7, 5, 30, 0, 0, time.UTC).In(repeated.Location()), wantDay: repeated},
		{name: "second 01:30", at: time.Date(2021, time.November, 7, 6, 30, 0, 0, time.UTC).In(repeated.Location()), wantDay: repeated},
		{name: "year 10000", at: time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC), wantDay: nextYear},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			timing, err := marketevent.NewExactTiming(tt.at)
			require.NoError(t, err)

			instant, exact := timing.Instant()
			require.True(t, exact)
			assert.True(t, instant.Equal(tt.at))
			assert.Equal(t, tt.at.Format(time.RFC3339Nano), instant.Format(time.RFC3339Nano))
			assert.True(t, tt.wantDay.Equal(timing.LocationDate()))
			assert.Equal(t, tt.wantDay.Date(), timing.Date())
			assert.Same(t, timing.Location(), instant.Location())
			assert.True(t, timing.IsValid())

			_, flagged := timing.Flag()
			assert.False(t, flagged)
		})
	}
}

func TestNewExactTimingLocal(t *testing.T) {
	t.Parallel()

	at := time.Date(2021, time.February, 5, 12, 0, 0, 0, time.Local)
	want, zoneErr := civil.NewLocationDate(mustDate(t, "2021-02-05"), time.Local.String())
	timing, err := marketevent.NewExactTiming(at)
	if zoneErr != nil {
		require.ErrorContains(t, err, zoneErr.Error())

		return
	}

	require.NoError(t, err)
	assert.True(t, want.Equal(timing.LocationDate()))
}

func TestNewExactTimingRejects(t *testing.T) {
	t.Parallel()

	at := time.Date(2021, time.February, 5, 17, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		at      time.Time
		wantErr string
	}{
		{name: "zero time", wantErr: "instant is missing"},
		{name: "numeric offset alone", at: at.In(time.FixedZone("", -5*60*60)), wantErr: "zone must not be empty"},
		{name: "unknown zone name", at: at.In(time.FixedZone("Unknown/Zone", 0)), wantErr: "invalid zone"},
		{
			name:    "wrong offset for the named zone on the same date",
			at:      at.In(time.FixedZone(newYork, -4*60*60)),
			wantErr: "disagrees with America/New_York",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			timing, err := marketevent.NewExactTiming(tt.at)
			require.ErrorContains(t, err, tt.wantErr)
			assert.False(t, timing.IsValid())
		})
	}
}

func TestNewDateTimings(t *testing.T) {
	t.Parallel()

	day := mustLocationDate(t, "2021-02-05", newYork)

	tests := []struct {
		name        string
		build       func(civil.LocationDate) (marketevent.Timing, error)
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

			timing, err := tt.build(day)
			require.NoError(t, err)

			_, exact := timing.Instant()
			flag, flagged := timing.Flag()

			assert.Equal(t, day.Date(), timing.Date())
			assert.Equal(t, day.Location(), timing.Location())
			assert.True(t, day.Equal(timing.LocationDate()))
			assert.False(t, exact)
			assert.Equal(t, tt.wantFlag, flag)
			assert.Equal(t, tt.wantFlagged, flagged)
			assert.True(t, timing.IsValid())

			_, err = tt.build(civil.LocationDate{})
			require.ErrorContains(t, err, "date is missing")
		})
	}

	assert.False(t, marketevent.Timing{}.IsValid())
}
