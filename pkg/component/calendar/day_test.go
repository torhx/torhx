package calendar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talav/civil"

	"github.com/torhx/torhx/pkg/component/calendar"
)

func mustDate(t *testing.T, s string) civil.Date {
	t.Helper()

	d, err := civil.NewDateFromString(s)
	require.NoError(t, err)

	return d
}

// mustOpen returns the open of a business day.
func mustOpen(t *testing.T, day calendar.TradingDay) time.Time {
	t.Helper()

	openAt, ok := day.Open()
	require.True(t, ok)

	return openAt
}

// mustClose returns the close of a business day.
func mustClose(t *testing.T, day calendar.TradingDay) time.Time {
	t.Helper()

	closeAt, ok := day.Close()
	require.True(t, ok)

	return closeAt
}

func TestNewHolidayAndNewWeekend(t *testing.T) {
	t.Parallel()

	date := mustDate(t, "2026-11-26")

	tests := []struct {
		name        string
		build       func(civil.Date) (calendar.TradingDay, error)
		wantHoliday bool
		wantWeekend bool
	}{
		{name: "holiday", build: calendar.NewHoliday, wantHoliday: true},
		{name: "weekend", build: calendar.NewWeekend, wantWeekend: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.build(civil.Date{})
			require.ErrorContains(t, err, "date is missing")

			day, err := tt.build(date)
			require.NoError(t, err)

			_, opens := day.Open()
			_, closes := day.Close()

			assert.True(t, day.IsValid())
			assert.Equal(t, date, day.Date())
			assert.Equal(t, tt.wantHoliday, day.IsHoliday())
			assert.Equal(t, tt.wantWeekend, day.IsWeekend())
			assert.False(t, day.IsBusinessDay())
			assert.False(t, day.IsFullDay())
			assert.False(t, opens)
			assert.False(t, closes)
			assert.Empty(t, day.Sessions())
		})
	}

	assert.False(t, calendar.TradingDay{}.IsValid())
}

func TestNewTradingDayRejects(t *testing.T) {
	t.Parallel()

	loc := time.UTC
	date := mustDate(t, "2026-11-16")
	morning := mustSession(t, calendar.SessionKindRegular, loc, "2026-11-16 09:00", "2026-11-16 11:30")
	afternoon := mustSession(t, calendar.SessionKindRegular, loc, "2026-11-16 12:30", "2026-11-16 15:30")
	overlapping := mustSession(t, calendar.SessionKindRegular, loc, "2026-11-16 11:00", "2026-11-16 15:30")
	preMarket := mustSession(t, calendar.SessionKindExtended, loc, "2026-11-16 04:00", "2026-11-16 09:00")

	tests := []struct {
		name     string
		date     civil.Date
		sessions []calendar.Session
		wantErr  string
	}{
		{name: "no date", date: civil.Date{}, sessions: []calendar.Session{morning}, wantErr: "date is missing"},
		{name: "no session", date: date, sessions: nil, wantErr: "no regular session"},
		{name: "extended only", date: date, sessions: []calendar.Session{preMarket}, wantErr: "no regular session"},
		{name: "zero session", date: date, sessions: []calendar.Session{{}}, wantErr: "is not a session"},
		{name: "out of order", date: date, sessions: []calendar.Session{afternoon, morning}, wantErr: "before session 0 closes"},
		{name: "overlap", date: date, sessions: []calendar.Session{morning, overlapping}, wantErr: "before session 0 closes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := calendar.NewTradingDay(tt.date, tt.sessions, false, false)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestTradingDayWithBreakAndExtendedHours(t *testing.T) {
	t.Parallel()

	loc := time.UTC
	sessions := []calendar.Session{
		mustSession(t, calendar.SessionKindExtended, loc, "2026-11-16 08:00", "2026-11-16 09:00"),
		mustSession(t, calendar.SessionKindRegular, loc, "2026-11-16 09:00", "2026-11-16 11:30"),
		mustSession(t, calendar.SessionKindRegular, loc, "2026-11-16 12:30", "2026-11-16 15:30"),
		mustSession(t, calendar.SessionKindExtended, loc, "2026-11-16 15:30", "2026-11-16 17:00"),
	}

	day, err := calendar.NewTradingDay(mustDate(t, "2026-11-16"), sessions, false, false)
	require.NoError(t, err)

	assert.Equal(t, mustDate(t, "2026-11-16"), day.Date())
	assert.Equal(t, sessions, day.Sessions())
	assert.Equal(t, mustAt(t, loc, "2026-11-16 09:00"), mustOpen(t, day))
	assert.Equal(t, mustAt(t, loc, "2026-11-16 15:30"), mustClose(t, day))
	assert.Equal(t, 5*time.Hour+30*time.Minute, day.Duration())
	assert.Equal(t, 6*time.Hour+30*time.Minute, mustClose(t, day).Sub(mustOpen(t, day)))
}

func TestTradingDayBookedToALaterDate(t *testing.T) {
	t.Parallel()

	// Trading over a holiday can be booked to the next business day, so the
	// whole session lies before the day's date.
	session := mustSession(t, calendar.SessionKindRegular, time.UTC, "2026-11-25 23:00", "2026-11-26 18:00")

	day, err := calendar.NewTradingDay(mustDate(t, "2026-11-27"), []calendar.Session{session}, false, false)
	require.NoError(t, err)
	assert.Equal(t, session.Open(), mustOpen(t, day))
}

func TestTradingDayLateOpenAndEarlyClose(t *testing.T) {
	t.Parallel()

	session := mustSession(t, calendar.SessionKindRegular, time.UTC, "2026-11-16 09:00", "2026-11-16 15:30")

	tests := []struct {
		name       string
		lateOpen   bool
		earlyClose bool
		wantFull   bool
	}{
		{name: "full day", wantFull: true},
		{name: "late open", lateOpen: true},
		{name: "early close", earlyClose: true},
		{name: "both", lateOpen: true, earlyClose: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			day, err := calendar.NewTradingDay(mustDate(t, "2026-11-16"), []calendar.Session{session}, tt.lateOpen, tt.earlyClose)
			require.NoError(t, err)

			assert.Equal(t, tt.lateOpen, day.IsLateOpen())
			assert.Equal(t, tt.earlyClose, day.IsEarlyClose())
			assert.Equal(t, tt.wantFull, day.IsFullDay())
			assert.True(t, day.IsBusinessDay())
		})
	}
}
