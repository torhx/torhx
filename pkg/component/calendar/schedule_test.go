package calendar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talav/civil"
	"github.com/torhx/exchange"

	"github.com/torhx/torhx/pkg/component/calendar"
)

// mustHours returns the hours of a session of the given kind, with start and
// end written as wall-clock times such as "09:30".
func mustHours(t *testing.T, kind calendar.SessionKind, start, end string, daysBefore int) calendar.Hours {
	t.Helper()

	from, err := civil.NewTimeFromString(start)
	require.NoError(t, err)

	to, err := civil.NewTimeFromString(end)
	require.NoError(t, err)

	h, err := calendar.NewHours(kind, from, to, daysBefore)
	require.NoError(t, err)

	return h
}

func workweek() []time.Weekday {
	return []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
}

func everyDay() []time.Weekday {
	return []time.Weekday{
		time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday,
	}
}

// mustWeek returns the schedule that gives each of the weekdays the same
// hours, from the date from.
func mustWeek(t *testing.T, from string, weekdays []time.Weekday, hours ...calendar.Hours) calendar.WeeklySchedule {
	t.Helper()

	var days [7][]calendar.Hours
	for _, weekday := range weekdays {
		days[weekday] = hours
	}

	w, err := calendar.NewWeeklySchedule(mustDate(t, from), days)
	require.NoError(t, err)

	return w
}

func mustBuild(t *testing.T, e exchange.Exchange, from, to string, schedule calendar.WeeklySchedule, exceptions map[civil.Date][]calendar.Session) calendar.TradingCalendar {
	t.Helper()

	c, err := calendar.BuildTradingCalendar(
		mustID(t, e.Code().String()), e, mustDate(t, from), mustDate(t, to),
		[]calendar.WeeklySchedule{schedule}, exceptions,
	)
	require.NoError(t, err)

	return c
}

// mustVenue returns an exchange in the given time zone, for zones no
// built-in exchange keeps.
func mustVenue(t *testing.T, timeZone string) exchange.Exchange {
	t.Helper()

	code, err := exchange.NewCode("TEST")
	require.NoError(t, err)

	wholeDay, err := exchange.NewSession(civil.Time{}, civil.Time{})
	require.NoError(t, err)

	e, err := exchange.NewExchange(code, "Test venue", timeZone, everyDay(), []exchange.Session{wholeDay})
	require.NoError(t, err)

	return e
}

func TestNewHoursRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		kind       calendar.SessionKind
		daysBefore int
		wantErr    string
	}{
		{name: "no kind", kind: "", wantErr: "unrecognised session kind"},
		{name: "negative days before", kind: calendar.SessionKindRegular, daysBefore: -1, wantErr: "days before must be 0 to 6"},
		{name: "a week before", kind: calendar.SessionKindRegular, daysBefore: 7, wantErr: "days before must be 0 to 6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := calendar.NewHours(tt.kind, civil.Time{}, civil.Time{}, tt.daysBefore)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestNewWeeklyScheduleRejects(t *testing.T) {
	t.Parallel()

	from := mustDate(t, "2026-01-01")
	morning := mustHours(t, calendar.SessionKindRegular, "09:00", "11:30", 0)
	afternoon := mustHours(t, calendar.SessionKindRegular, "12:30", "15:30", 0)
	overnight := mustHours(t, calendar.SessionKindRegular, "18:00", "17:00", 1)
	preMarket := mustHours(t, calendar.SessionKindExtended, "04:00", "09:00", 0)

	tests := []struct {
		name    string
		from    civil.Date
		monday  []calendar.Hours
		wantErr string
	}{
		{name: "no first date", from: civil.Date{}, monday: []calendar.Hours{morning}, wantErr: "first date is missing"},
		{name: "zero hours", from: from, monday: []calendar.Hours{{}}, wantErr: "are not hours"},
		{name: "out of order", from: from, monday: []calendar.Hours{afternoon, morning}, wantErr: "start before hours"},
		{name: "overnight session overlaps the morning", from: from, monday: []calendar.Hours{overnight, morning}, wantErr: "start before hours"},
		{name: "extended only", from: from, monday: []calendar.Hours{preMarket}, wantErr: "no regular session"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var days [7][]calendar.Hours
			days[time.Monday] = tt.monday

			_, err := calendar.NewWeeklySchedule(tt.from, days)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestBuildTradingCalendarRejects(t *testing.T) {
	t.Parallel()

	xnys := mustExchange(t, "XNYS")
	loc := xnys.Location()
	regular := mustHours(t, calendar.SessionKindRegular, "09:30", "16:00", 0)
	week := mustWeek(t, "2026-01-01", workweek(), regular)
	laterWeek := mustWeek(t, "2026-06-01", workweek(), regular)

	tests := []struct {
		name       string
		exchange   exchange.Exchange
		from       string
		to         string
		schedules  []calendar.WeeklySchedule
		exceptions map[civil.Date][]calendar.Session
		wantErr    string
	}{
		{
			name: "no exchange", exchange: exchange.Exchange{}, from: "2026-11-16", to: "2026-11-20",
			schedules: []calendar.WeeklySchedule{week}, wantErr: "exchange is missing",
		},
		{
			name: "reversed range", exchange: xnys, from: "2026-11-20", to: "2026-11-16",
			schedules: []calendar.WeeklySchedule{week}, wantErr: "are not a range",
		},
		{name: "no schedule", exchange: xnys, from: "2026-11-16", to: "2026-11-20", wantErr: "no weekly schedule"},
		{
			name: "zero schedule", exchange: xnys, from: "2026-11-16", to: "2026-11-20",
			schedules: []calendar.WeeklySchedule{{}}, wantErr: "is not a schedule",
		},
		{
			name: "versions out of order", exchange: xnys, from: "2026-11-16", to: "2026-11-20",
			schedules: []calendar.WeeklySchedule{laterWeek, week}, wantErr: "must come after",
		},
		{
			name: "no version in force at the start", exchange: xnys, from: "2026-03-02", to: "2026-11-20",
			schedules: []calendar.WeeklySchedule{laterWeek}, wantErr: "no weekly schedule is in force on 2026-03-02",
		},
		{
			name: "exception outside the range", exchange: xnys, from: "2026-11-16", to: "2026-11-20",
			schedules:  []calendar.WeeklySchedule{week},
			exceptions: map[civil.Date][]calendar.Session{mustDate(t, "2026-11-26"): {}},
			wantErr:    "exception on 2026-11-26 is outside",
		},
		{
			name: "several exceptions outside the range", exchange: xnys, from: "2026-11-16", to: "2026-11-20",
			schedules: []calendar.WeeklySchedule{week},
			exceptions: map[civil.Date][]calendar.Session{
				mustDate(t, "2026-11-26"): {}, mustDate(t, "2026-11-17"): {}, mustDate(t, "2026-11-09"): {}, mustDate(t, "2026-11-02"): {},
			},
			wantErr: "exception on 2026-11-02 is outside",
		},
		{
			name: "exception without a regular session", exchange: xnys, from: "2026-11-16", to: "2026-11-20",
			schedules: []calendar.WeeklySchedule{week},
			exceptions: map[civil.Date][]calendar.Session{
				mustDate(t, "2026-11-17"): {mustSession(t, calendar.SessionKindExtended, loc, "2026-11-17 04:00", "2026-11-17 09:30")},
			},
			wantErr: "no regular session",
		},
		{
			name: "exception runs into the next day", exchange: xnys, from: "2026-11-16", to: "2026-11-20",
			schedules: []calendar.WeeklySchedule{week},
			exceptions: map[civil.Date][]calendar.Session{
				mustDate(t, "2026-11-17"): {mustSession(t, calendar.SessionKindRegular, loc, "2026-11-17 09:30", "2026-11-18 10:00")},
			},
			wantErr: "before day 2026-11-17 closes",
		},
		{
			name: "every date closed", exchange: xnys, from: "2026-11-21", to: "2026-11-22",
			schedules: []calendar.WeeklySchedule{week}, wantErr: "no business day",
		},
		{
			name: "time skipped by the clock change", exchange: xnys, from: "2026-03-02", to: "2026-03-13",
			schedules: []calendar.WeeklySchedule{
				mustWeek(t, "2026-01-01", []time.Weekday{time.Sunday}, mustHours(t, calendar.SessionKindRegular, "02:30", "04:00", 0)),
			},
			wantErr: "02:30:00 is skipped on 2026-03-08",
		},
		{
			name: "time repeated by the clock change", exchange: xnys, from: "2026-10-26", to: "2026-11-06",
			schedules: []calendar.WeeklySchedule{
				mustWeek(t, "2026-01-01", []time.Weekday{time.Sunday}, mustHours(t, calendar.SessionKindRegular, "01:30", "04:00", 0)),
			},
			wantErr: "01:30:00 is repeated on 2026-11-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := calendar.BuildTradingCalendar(
				mustID(t, "XNYS"), tt.exchange, mustDate(t, tt.from), mustDate(t, tt.to), tt.schedules, tt.exceptions,
			)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestBuildTradingCalendarSetsLateOpenAndEarlyClose(t *testing.T) {
	t.Parallel()

	xnys := mustExchange(t, "XNYS")
	loc := xnys.Location()

	regular := func(openAt, closeAt string) calendar.Session {
		return mustSession(t, calendar.SessionKindRegular, loc, "2026-11-17 "+openAt, "2026-11-17 "+closeAt)
	}
	preMarket := mustSession(t, calendar.SessionKindExtended, loc, "2026-11-17 07:00", "2026-11-17 09:30")

	week := mustWeek(t, "2026-01-01", workweek(),
		mustHours(t, calendar.SessionKindExtended, "04:00", "09:30", 0),
		mustHours(t, calendar.SessionKindRegular, "09:30", "16:00", 0),
	)

	tests := []struct {
		name           string
		sessions       []calendar.Session
		wantLateOpen   bool
		wantEarlyClose bool
	}{
		{name: "the scheduled times", sessions: []calendar.Session{regular("09:30", "16:00")}},
		{name: "early close", sessions: []calendar.Session{regular("09:30", "13:00")}, wantEarlyClose: true},
		{name: "late open", sessions: []calendar.Session{regular("11:00", "16:00")}, wantLateOpen: true},
		{name: "both", sessions: []calendar.Session{regular("11:00", "13:00")}, wantLateOpen: true, wantEarlyClose: true},
		{name: "longer hours", sessions: []calendar.Session{regular("09:00", "17:00")}},
		{name: "a midday break", sessions: []calendar.Session{regular("09:30", "12:00"), regular("13:00", "16:00")}},
		{name: "only the pre-market differs", sessions: []calendar.Session{preMarket, regular("09:30", "16:00")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := mustBuild(t, xnys, "2026-11-16", "2026-11-20", week, map[civil.Date][]calendar.Session{
				mustDate(t, "2026-11-17"): tt.sessions,
			})

			day, err := c.Day(mustDate(t, "2026-11-17"))
			require.NoError(t, err)
			require.True(t, day.IsBusinessDay())

			assert.Equal(t, tt.wantLateOpen, day.IsLateOpen())
			assert.Equal(t, tt.wantEarlyClose, day.IsEarlyClose())
			assert.Equal(t, !tt.wantLateOpen && !tt.wantEarlyClose, day.IsFullDay())
			assert.Equal(t, tt.sessions, day.Sessions())
		})
	}
}

func TestBuildTradingCalendarScheduledDaysAreFullDays(t *testing.T) {
	t.Parallel()

	c := newXNYS(t)
	from, to := c.Coverage()

	dates, err := c.TradingDaysBetween(from, to)
	require.NoError(t, err)

	for _, date := range dates {
		day, err := c.Day(date)
		require.NoError(t, err)
		require.True(t, day.IsBusinessDay())

		assert.Equal(t, date != mustDate(t, "2026-11-27"), day.IsFullDay(), date.String())
	}
}

func TestBuildTradingCalendarHalfDayDropsTheAfternoon(t *testing.T) {
	t.Parallel()

	xhkg := mustExchange(t, "XHKG")
	week := mustWeek(t, "2026-01-01", workweek(),
		mustHours(t, calendar.SessionKindRegular, "09:30", "12:00", 0),
		mustHours(t, calendar.SessionKindRegular, "13:00", "16:00", 0),
	)

	c := mustBuild(t, xhkg, "2026-12-21", "2026-12-25", week, map[civil.Date][]calendar.Session{
		mustDate(t, "2026-12-24"): {mustSession(t, calendar.SessionKindRegular, xhkg.Location(), "2026-12-24 09:30", "2026-12-24 12:00")},
	})

	day, err := c.Day(mustDate(t, "2026-12-24"))
	require.NoError(t, err)
	require.True(t, day.IsBusinessDay())

	assert.Len(t, day.Sessions(), 1)
	assert.True(t, day.IsEarlyClose())
	assert.False(t, day.IsLateOpen())
	assert.False(t, day.IsFullDay())
}

func TestBuildTradingCalendarExceptionAddsADay(t *testing.T) {
	t.Parallel()

	xnys := mustExchange(t, "XNYS")
	week := mustWeek(t, "2026-01-01", workweek(), mustHours(t, calendar.SessionKindRegular, "09:30", "16:00", 0))
	special := mustSession(t, calendar.SessionKindRegular, xnys.Location(), "2026-11-21 10:00", "2026-11-21 12:00")

	c := mustBuild(t, xnys, "2026-11-16", "2026-11-27", week, map[civil.Date][]calendar.Session{
		mustDate(t, "2026-11-21"): {special},
	})

	day, err := c.Day(mustDate(t, "2026-11-21"))
	require.NoError(t, err)
	require.True(t, day.IsBusinessDay())

	// Nothing is scheduled for a Saturday, so there is nothing to be late or
	// early against.
	assert.True(t, day.IsFullDay())
	assert.Equal(t, special.Open(), mustOpen(t, day))
}

func TestBuildTradingCalendarTellsAHolidayFromAWeekend(t *testing.T) {
	t.Parallel()

	week := mustWeek(t, "2026-01-01", workweek(), mustHours(t, calendar.SessionKindRegular, "09:30", "16:00", 0))

	// Thanksgiving is closed, and so is the Saturday after it, which never
	// trades anyway.
	c := mustBuild(t, mustExchange(t, "XNYS"), "2026-11-23", "2026-11-29", week, map[civil.Date][]calendar.Session{
		mustDate(t, "2026-11-26"): {},
		mustDate(t, "2026-11-28"): {},
	})

	tests := []struct {
		date         string
		wantBusiness bool
		wantHoliday  bool
		wantWeekend  bool
	}{
		{date: "2026-11-25", wantBusiness: true},
		{date: "2026-11-26", wantHoliday: true},
		{date: "2026-11-27", wantBusiness: true},
		{date: "2026-11-28", wantWeekend: true},
		{date: "2026-11-29", wantWeekend: true},
	}

	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			t.Parallel()

			day, err := c.Day(mustDate(t, tt.date))
			require.NoError(t, err)

			assert.Equal(t, tt.wantBusiness, day.IsBusinessDay())
			assert.Equal(t, tt.wantHoliday, day.IsHoliday())
			assert.Equal(t, tt.wantWeekend, day.IsWeekend())
		})
	}
}

func TestBuildTradingCalendarAppliesTheVersionInForce(t *testing.T) {
	t.Parallel()

	xtks := mustExchange(t, "XTKS")
	loc := xtks.Location()
	morning := mustHours(t, calendar.SessionKindRegular, "09:00", "11:30", 0)

	c, err := calendar.BuildTradingCalendar(
		mustID(t, "XTKS"), xtks, mustDate(t, "2024-10-28"), mustDate(t, "2024-11-08"),
		[]calendar.WeeklySchedule{
			mustWeek(t, "2024-01-01", workweek(), morning, mustHours(t, calendar.SessionKindRegular, "12:30", "15:00", 0)),
			mustWeek(t, "2024-11-05", workweek(), morning, mustHours(t, calendar.SessionKindRegular, "12:30", "15:30", 0)),
		},
		nil,
	)
	require.NoError(t, err)

	before, err := c.Day(mustDate(t, "2024-11-01"))
	require.NoError(t, err)
	require.True(t, before.IsBusinessDay())
	assert.Equal(t, mustAt(t, loc, "2024-11-01 15:00"), mustClose(t, before))
	assert.True(t, before.IsFullDay(), "a schedule change is not an early close")

	after, err := c.Day(mustDate(t, "2024-11-05"))
	require.NoError(t, err)
	require.True(t, after.IsBusinessDay())
	assert.Equal(t, mustAt(t, loc, "2024-11-05 15:30"), mustClose(t, after))
}

func TestBuildTradingCalendarWeekdaySpecificHours(t *testing.T) {
	t.Parallel()

	xnys := mustExchange(t, "XNYS")

	var days [7][]calendar.Hours
	for _, weekday := range workweek() {
		days[weekday] = []calendar.Hours{mustHours(t, calendar.SessionKindRegular, "09:30", "16:00", 0)}
	}

	days[time.Friday] = []calendar.Hours{mustHours(t, calendar.SessionKindRegular, "09:30", "15:00", 0)}

	week, err := calendar.NewWeeklySchedule(mustDate(t, "2026-01-01"), days)
	require.NoError(t, err)

	c := mustBuild(t, xnys, "2026-11-16", "2026-11-20", week, nil)

	friday, err := c.Day(mustDate(t, "2026-11-20"))
	require.NoError(t, err)
	require.True(t, friday.IsBusinessDay())

	assert.Equal(t, mustAt(t, xnys.Location(), "2026-11-20 15:00"), mustClose(t, friday))
	assert.True(t, friday.IsFullDay())
}

func TestBuildTradingCalendarKeepsWallClockTimesAcrossAClockChange(t *testing.T) {
	t.Parallel()

	xnys := mustExchange(t, "XNYS")
	week := mustWeek(t, "2026-01-01", workweek(), mustHours(t, calendar.SessionKindRegular, "09:30", "16:00", 0))

	// New York moves its clock forward on Sunday 2026-03-08.
	c := mustBuild(t, xnys, "2026-03-02", "2026-03-13", week, nil)

	tests := []struct {
		date    string
		wantUTC string
	}{
		{date: "2026-03-06", wantUTC: "2026-03-06 14:30"},
		{date: "2026-03-09", wantUTC: "2026-03-09 13:30"},
	}

	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			t.Parallel()

			day, err := c.Day(mustDate(t, tt.date))
			require.NoError(t, err)
			require.True(t, day.IsBusinessDay())

			hour, minute, _ := mustOpen(t, day).Clock()
			assert.Equal(t, 9, hour)
			assert.Equal(t, 30, minute)
			assert.True(t, mustOpen(t, day).Equal(mustAt(t, time.UTC, tt.wantUTC)))
		})
	}
}

func TestBuildTradingCalendarDaysAroundAClockChange(t *testing.T) {
	t.Parallel()

	regular := mustHours(t, calendar.SessionKindRegular, "09:30", "16:00", 0)
	overnight := mustHours(t, calendar.SessionKindRegular, "17:00", "16:00", 1)
	wholeDate := mustHours(t, calendar.SessionKindRegular, "00:00", "00:00", 0)

	// New York and Chicago move their clocks forward on Sunday 2026-03-08 and
	// back on Sunday 2026-11-01. Lord Howe Island moves its clock back by
	// half an hour on Sunday 2026-04-05.
	tests := []struct {
		name         string
		venue        exchange.Exchange
		weekdays     []time.Weekday
		hours        calendar.Hours
		date         string
		wantOpenUTC  string
		wantDuration time.Duration
	}{
		{
			name: "stock open on the Monday after the clock goes back", venue: mustExchange(t, "XNYS"),
			weekdays: workweek(), hours: regular, date: "2026-11-02",
			wantOpenUTC: "2026-11-02 14:30", wantDuration: 6*time.Hour + 30*time.Minute,
		},
		{
			name: "futures open on the Sunday the clock goes back", venue: mustExchange(t, "XCBF"),
			weekdays: workweek(), hours: overnight, date: "2026-11-02",
			wantOpenUTC: "2026-11-01 23:00", wantDuration: 23 * time.Hour,
		},
		{
			name: "futures open on the Sunday the clock goes forward", venue: mustExchange(t, "XCBF"),
			weekdays: workweek(), hours: overnight, date: "2026-03-09",
			wantOpenUTC: "2026-03-08 22:00", wantDuration: 23 * time.Hour,
		},
		{
			name: "whole date the clock goes back", venue: mustVenue(t, "America/New_York"),
			weekdays: everyDay(), hours: wholeDate, date: "2026-11-01",
			wantOpenUTC: "2026-11-01 04:00", wantDuration: 25 * time.Hour,
		},
		{
			name: "whole date the clock goes forward", venue: mustVenue(t, "America/New_York"),
			weekdays: everyDay(), hours: wholeDate, date: "2026-03-08",
			wantOpenUTC: "2026-03-08 05:00", wantDuration: 23 * time.Hour,
		},
		{
			name: "whole date the clock goes back by half an hour", venue: mustVenue(t, "Australia/Lord_Howe"),
			weekdays: everyDay(), hours: wholeDate, date: "2026-04-05",
			wantOpenUTC: "2026-04-04 13:00", wantDuration: 24*time.Hour + 30*time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			date := mustDate(t, tt.date)
			week := mustWeek(t, "2026-01-01", tt.weekdays, tt.hours)

			c := mustBuild(t, tt.venue, date.AddDays(-3).String(), date.AddDays(3).String(), week, nil)

			day, err := c.Day(date)
			require.NoError(t, err)
			require.True(t, day.IsBusinessDay())

			assert.True(t, mustOpen(t, day).Equal(mustAt(t, time.UTC, tt.wantOpenUTC)), "opens at %s", mustOpen(t, day).UTC())
			assert.Equal(t, tt.wantDuration, day.Duration())
			assert.True(t, day.IsFullDay())
		})
	}
}

func TestBuildTradingCalendarExceptionStatesADateWhoseScheduleDoesNotResolve(t *testing.T) {
	t.Parallel()

	xnys := mustExchange(t, "XNYS")
	week := mustWeek(t, "2026-01-01", []time.Weekday{time.Sunday}, mustHours(t, calendar.SessionKindRegular, "02:30", "04:00", 0))
	given := mustSession(t, calendar.SessionKindRegular, xnys.Location(), "2026-03-08 03:00", "2026-03-08 04:00")

	// 02:30 does not exist in New York on 2026-03-08, so the date is stated
	// by an exception, which gives instants.
	c := mustBuild(t, xnys, "2026-03-01", "2026-03-15", week, map[civil.Date][]calendar.Session{
		mustDate(t, "2026-03-08"): {given},
	})

	day, err := c.Day(mustDate(t, "2026-03-08"))
	require.NoError(t, err)
	require.True(t, day.IsBusinessDay())

	assert.Equal(t, given.Open(), mustOpen(t, day))
	assert.True(t, day.IsFullDay(), "with no scheduled time to compare with, the day is neither")
}

func TestBuildTradingCalendarMidnightIsTheStartOfTheDate(t *testing.T) {
	t.Parallel()

	// Santiago moves its clock forward at midnight on Sunday 2026-09-06, so
	// that date has no 00:00: it starts at 01:00.
	venue := mustVenue(t, "America/Santiago")
	week := mustWeek(t, "2026-01-01", everyDay(), mustHours(t, calendar.SessionKindRegular, "00:00", "00:00", 0))

	c := mustBuild(t, venue, "2026-09-04", "2026-09-08", week, nil)

	saturday, err := c.Day(mustDate(t, "2026-09-05"))
	require.NoError(t, err)
	require.True(t, saturday.IsBusinessDay())

	sunday, err := c.Day(mustDate(t, "2026-09-06"))
	require.NoError(t, err)
	require.True(t, sunday.IsBusinessDay())

	assert.Equal(t, mustClose(t, saturday), mustOpen(t, sunday), "the days touch")
	assert.Equal(t, 24*time.Hour, saturday.Duration())
	assert.Equal(t, 23*time.Hour, sunday.Duration())

	hour, minute, _ := mustOpen(t, sunday).Clock()
	assert.Equal(t, 1, hour)
	assert.Equal(t, 0, minute)
}

func TestBuildTradingCalendarHolidayOnAFuturesCalendar(t *testing.T) {
	t.Parallel()

	xcbf := mustExchange(t, "XCBF")
	loc := xcbf.Location()
	week := mustWeek(t, "2026-01-01", workweek(), mustHours(t, calendar.SessionKindRegular, "17:00", "16:00", 1))

	// Thursday is closed, and Friday's day opens on Thursday evening as
	// usual. Wednesday's session still runs to its normal close.
	c := mustBuild(t, xcbf, "2026-11-23", "2026-11-27", week, map[civil.Date][]calendar.Session{
		mustDate(t, "2026-11-26"): {},
	})

	isTradingDay, err := c.IsTradingDay(mustDate(t, "2026-11-26"))
	require.NoError(t, err)
	assert.False(t, isTradingDay)

	wednesday, err := c.Day(mustDate(t, "2026-11-25"))
	require.NoError(t, err)
	require.True(t, wednesday.IsBusinessDay())
	assert.Equal(t, mustAt(t, loc, "2026-11-25 16:00"), mustClose(t, wednesday))

	_, open, err := c.SessionAt(mustAt(t, loc, "2026-11-25 20:00"))
	require.NoError(t, err)
	assert.False(t, open, "the evening that would open Thursday's day does not trade")

	next, err := c.TradingDayOffset(mustDate(t, "2026-11-25"), 1)
	require.NoError(t, err)
	assert.Equal(t, mustDate(t, "2026-11-27"), next)
}
