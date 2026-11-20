package calendar

import (
	"fmt"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talav/civil"
)

// builtInXNYS returns the built-in XNYS calendar.
func builtInXNYS(t *testing.T) TradingCalendar {
	t.Helper()

	calendars, err := TradingCalendars()
	require.NoError(t, err)

	registry, err := NewRegistry(calendars)
	require.NoError(t, err)

	id, err := NewID("XNYS")
	require.NoError(t, err)

	c, err := registry.Get(id)
	require.NoError(t, err)

	return c
}

// parseDate returns the date written in s.
func parseDate(t *testing.T, s string) civil.Date {
	t.Helper()

	d, err := civil.NewDateFromString(s)
	require.NoError(t, err)

	return d
}

// wallClock returns the sessions of day as wall-clock times in loc, such as
// "09:30-16:00", or nil when it has none.
func wallClock(day TradingDay, loc *time.Location) []string {
	if len(day.Sessions()) == 0 {
		return nil
	}

	sessions := make([]string, 0, len(day.Sessions()))
	for _, s := range day.Sessions() {
		sessions = append(sessions, s.Open().In(loc).Format("15:04")+"-"+s.Close().In(loc).Format("15:04"))
	}

	return sessions
}

func TestBuiltInXNYSCoverage(t *testing.T) {
	t.Parallel()

	xnys := builtInXNYS(t)
	from, to := xnys.Coverage()

	assert.Equal(t, "XNYS", xnys.Exchange().Code().String())
	assert.Equal(t, parseDate(t, "2000-01-01"), from)
	assert.Equal(t, parseDate(t, "2030-12-31"), to)
}

func TestBuiltInXNYSDays(t *testing.T) {
	t.Parallel()

	xnys := builtInXNYS(t)
	loc := xnys.Exchange().Location()
	full := []string{"09:30-16:00"}
	early := []string{"09:30-13:00"}

	tests := []struct {
		name         string
		date         string
		wantSessions []string
		wantWeekend  bool
	}{
		{name: "September 11 attacks", date: "2001-09-11"},
		{name: "reopening after September 11", date: "2001-09-17", wantSessions: full},
		{name: "Friday after Independence Day before 2013", date: "2002-07-05", wantSessions: early},
		{name: "day after Christmas 2003", date: "2003-12-26", wantSessions: early},
		{name: "Reagan day of mourning", date: "2004-06-11"},
		{name: "New Year's Day on a Saturday is not observed", date: "2004-12-31", wantSessions: full},
		{name: "New Year's Day on a Sunday moves to Monday", date: "2006-01-02"},
		{name: "Hurricane Sandy", date: "2012-10-30"},
		{name: "Wednesday before Independence Day from 2013", date: "2013-07-03", wantSessions: early},
		{name: "Friday after Independence Day from 2013", date: "2013-07-05", wantSessions: full},
		{name: "Christmas on a Saturday moves to Friday", date: "2021-12-24"},
		{name: "first Juneteenth", date: "2022-06-20"},
		{name: "Carter day of mourning", date: "2025-01-09"},
		{name: "Thanksgiving", date: "2026-11-26"},
		{name: "day after Thanksgiving", date: "2026-11-27", wantSessions: early},
		{name: "Christmas Eve", date: "2026-12-24", wantSessions: early},
		{name: "Juneteenth on a Saturday moves to Friday", date: "2027-06-18"},
		{name: "day before Independence Day", date: "2028-07-03", wantSessions: early},
		{name: "Saturday", date: "2026-11-21", wantWeekend: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			day, err := xnys.Day(parseDate(t, tt.date))
			require.NoError(t, err)

			assert.Equal(t, tt.wantSessions, wallClock(day, loc))
			assert.Equal(t, tt.wantWeekend, day.IsWeekend())
			assert.Equal(t, tt.wantSessions == nil && !tt.wantWeekend, day.IsHoliday())
			assert.Equal(t, slices.Equal(tt.wantSessions, early), day.IsEarlyClose())
		})
	}
}

func TestBuiltInXNYSTradingDaysPerYear(t *testing.T) {
	t.Parallel()

	xnys := builtInXNYS(t)

	// The counts check every holiday of a year at once. The unusual ones come
	// from closures and from how holidays fall on weekends.
	tests := []struct {
		year int
		want int
	}{
		{year: 2000, want: 252},
		{year: 2001, want: 248},
		{year: 2008, want: 253},
		{year: 2012, want: 250},
		{year: 2020, want: 253},
		{year: 2025, want: 250},
		{year: 2028, want: 251},
		{year: 2029, want: 251},
		{year: 2030, want: 251},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.year), func(t *testing.T) {
			t.Parallel()

			days, err := xnys.TradingDaysBetween(parseDate(t, fmt.Sprintf("%d-01-01", tt.year)), parseDate(t, fmt.Sprintf("%d-12-31", tt.year)))
			require.NoError(t, err)
			assert.Len(t, days, tt.want)
		})
	}
}

func TestLoadTradingCalendar(t *testing.T) {
	t.Parallel()

	// Hong Kong trades 09:30-12:00 and 13:00-16:00, so its exception days show
	// that the listed sessions replace both of a day's sessions.
	c, err := loadTradingCalendar([]byte(`{
		"exchange": "XHKG",
		"from": "2026-11-23",
		"to": "2026-11-29",
		"exceptions": [
			{"date": "2026-11-24", "note": "Late open", "sessions": [{"open": "10:30", "close": "12:00"}, {"open": "13:00", "close": "16:00"}]},
			{"date": "2026-11-25", "note": "Morning only", "sessions": [{"open": "09:30", "close": "12:00"}]},
			{"date": "2026-11-26", "note": "Closed", "sessions": []},
			{"date": "2026-11-28", "note": "Saturday session", "sessions": [{"open": "10:00", "close": "12:00"}]}
		]
	}`))
	require.NoError(t, err)

	loc := c.Exchange().Location()

	tests := []struct {
		date           string
		wantSessions   []string
		wantLateOpen   bool
		wantEarlyClose bool
	}{
		{date: "2026-11-23", wantSessions: []string{"09:30-12:00", "13:00-16:00"}},
		{date: "2026-11-24", wantSessions: []string{"10:30-12:00", "13:00-16:00"}, wantLateOpen: true},
		{date: "2026-11-25", wantSessions: []string{"09:30-12:00"}, wantEarlyClose: true},
		{date: "2026-11-26"},
		{date: "2026-11-28", wantSessions: []string{"10:00-12:00"}},
	}

	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			t.Parallel()

			day, err := c.Day(parseDate(t, tt.date))
			require.NoError(t, err)

			assert.Equal(t, tt.wantSessions, wallClock(day, loc))
			assert.Equal(t, tt.wantLateOpen, day.IsLateOpen())
			assert.Equal(t, tt.wantEarlyClose, day.IsEarlyClose())
		})
	}
}

func TestLoadTradingCalendarRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		exchange   string
		exceptions string
		wantErr    string
	}{
		{name: "mistyped key", exchange: "XNYS", exceptions: `{"date": "2026-11-17", "sesions": []}`, wantErr: `unknown field "sesions"`},
		{name: "malformed date", exchange: "XNYS", exceptions: `{"date": "2026-11-31", "sessions": []}`, wantErr: "2026-11-31"},
		{name: "malformed time", exchange: "XNYS", exceptions: `{"date": "2026-11-17", "sessions": [{"open": "9.30", "close": "13:00"}]}`, wantErr: "9.30"},
		{name: "unknown exchange", exchange: "QQQQ", exceptions: `{"date": "2026-11-17", "sessions": []}`, wantErr: "QQQQ"},
		{
			name: "date listed twice", exchange: "XNYS",
			exceptions: `{"date": "2026-11-17", "sessions": []}, {"date": "2026-11-17", "sessions": []}`,
			wantErr:    "2026-11-17 is listed twice",
		},
		{
			name: "time the clock skips", exchange: "XNYS",
			exceptions: `{"date": "2026-03-08", "sessions": [{"open": "02:30", "close": "04:00"}]}`,
			wantErr:    "2026-03-08: 02:30:00 is skipped",
		},
		{name: "date outside the range", exchange: "XNYS", exceptions: `{"date": "2026-12-01", "sessions": []}`, wantErr: "exception on 2026-12-01 is outside"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content := `{"exchange": "` + tt.exchange + `", "from": "2026-11-16", "to": "2026-11-20", "exceptions": [` + tt.exceptions + `]}`

			_, err := loadTradingCalendar([]byte(content))
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
