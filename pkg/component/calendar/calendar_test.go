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

// market builds one of the fixture calendars.
type market func(t *testing.T) calendar.TradingCalendar

func mustExchange(t *testing.T, code string) exchange.Exchange {
	t.Helper()

	e, err := exchange.NewExchangeFromString(code)
	require.NoError(t, err)

	return e
}

func mustDates(t *testing.T, dates ...string) []civil.Date {
	t.Helper()

	parsed := make([]civil.Date, 0, len(dates))
	for _, s := range dates {
		parsed = append(parsed, mustDate(t, s))
	}

	return parsed
}

// newXNYS returns New York regular hours from Monday 2026-11-16 to Friday
// 2026-12-11, with Thanksgiving closed and the Friday after closing at 13:00.
func newXNYS(t *testing.T) calendar.TradingCalendar {
	t.Helper()

	e := mustExchange(t, "XNYS")
	week := mustWeek(t, "2026-01-01", workweek(), mustHours(t, calendar.SessionKindRegular, "09:30", "16:00", 0))

	return mustBuild(t, e, "2026-11-16", "2026-12-11", week, map[civil.Date][]calendar.Session{
		mustDate(t, "2026-11-26"): {},
		mustDate(t, "2026-11-27"): {
			mustSession(t, calendar.SessionKindRegular, e.Location(), "2026-11-27 09:30", "2026-11-27 13:00"),
		},
	})
}

// newXNYSExtended returns New York hours with pre-market and post-market
// sessions, for the week of Monday 2026-11-16.
func newXNYSExtended(t *testing.T) calendar.TradingCalendar {
	t.Helper()

	week := mustWeek(t, "2026-01-01", workweek(),
		mustHours(t, calendar.SessionKindExtended, "04:00", "09:30", 0),
		mustHours(t, calendar.SessionKindRegular, "09:30", "16:00", 0),
		mustHours(t, calendar.SessionKindExtended, "16:00", "20:00", 0),
	)

	return mustBuild(t, mustExchange(t, "XNYS"), "2026-11-16", "2026-11-20", week, nil)
}

// newFutures returns Chicago futures hours for the two weeks from Monday
// 2026-11-09: each day opens at 17:00 the evening before and closes at 16:00.
func newFutures(t *testing.T) calendar.TradingCalendar {
	t.Helper()

	week := mustWeek(t, "2026-01-01", workweek(), mustHours(t, calendar.SessionKindRegular, "17:00", "16:00", 1))

	return mustBuild(t, mustExchange(t, "XCBF"), "2026-11-09", "2026-11-20", week, nil)
}

// newFX returns a market that trades from Sunday 17:00 to Friday 17:00 New
// York time without a break, for the two weeks from Monday 2026-11-09.
func newFX(t *testing.T) calendar.TradingCalendar {
	t.Helper()

	week := mustWeek(t, "2026-01-01", workweek(), mustHours(t, calendar.SessionKindRegular, "17:00", "17:00", 1))

	return mustBuild(t, mustExchange(t, "XNYS"), "2026-11-09", "2026-11-20", week, nil)
}

// newCrypto returns a market that never stops, for the week from Monday
// 2026-11-09.
func newCrypto(t *testing.T) calendar.TradingCalendar {
	t.Helper()

	week := mustWeek(t, "2026-01-01", everyDay(), mustHours(t, calendar.SessionKindRegular, "00:00", "00:00", 0))

	return mustBuild(t, mustExchange(t, "BINANCE"), "2026-11-09", "2026-11-15", week, nil)
}

// newXTKS returns Tokyo hours, with their lunch break, for the week of
// Monday 2026-12-07.
func newXTKS(t *testing.T) calendar.TradingCalendar {
	t.Helper()

	week := mustWeek(t, "2026-01-01", workweek(),
		mustHours(t, calendar.SessionKindRegular, "09:00", "11:30", 0),
		mustHours(t, calendar.SessionKindRegular, "12:30", "15:30", 0),
	)

	return mustBuild(t, mustExchange(t, "XTKS"), "2026-12-07", "2026-12-11", week, nil)
}

func TestNewTradingCalendarRejects(t *testing.T) {
	t.Parallel()

	xnys := mustExchange(t, "XNYS")
	id := mustID(t, "XNYS")

	day := func(date, openAt, closeAt string) calendar.TradingDay {
		d, err := calendar.NewTradingDay(
			mustDate(t, date),
			[]calendar.Session{mustSession(t, calendar.SessionKindRegular, xnys.Location(), openAt, closeAt)},
			false, false,
		)
		require.NoError(t, err)

		return d
	}

	monday := day("2026-11-16", "2026-11-16 09:30", "2026-11-16 16:00")
	tuesday := day("2026-11-17", "2026-11-17 09:30", "2026-11-17 16:00")
	mondayAgain := day("2026-11-16", "2026-11-16 17:00", "2026-11-16 18:00")
	tuesdayEarly := day("2026-11-17", "2026-11-16 15:00", "2026-11-17 16:00")
	wednesday := day("2026-11-18", "2026-11-18 09:30", "2026-11-18 16:00")

	saturday, err := calendar.NewWeekend(mustDate(t, "2026-11-21"))
	require.NoError(t, err)

	tests := []struct {
		name     string
		id       calendar.ID
		exchange exchange.Exchange
		days     []calendar.TradingDay
		wantErr  string
	}{
		{name: "no ID", id: calendar.ID{}, exchange: xnys, days: []calendar.TradingDay{monday}, wantErr: "ID is missing"},
		{name: "no exchange", id: id, exchange: exchange.Exchange{}, days: []calendar.TradingDay{monday}, wantErr: "exchange is missing"},
		{name: "no day", id: id, exchange: xnys, days: nil, wantErr: "no day"},
		{name: "zero day", id: id, exchange: xnys, days: []calendar.TradingDay{{}}, wantErr: "is not a day"},
		{name: "dates out of order", id: id, exchange: xnys, days: []calendar.TradingDay{tuesday, monday}, wantErr: "must be the date after"},
		{name: "one date twice", id: id, exchange: xnys, days: []calendar.TradingDay{monday, mondayAgain}, wantErr: "must be the date after"},
		{name: "a date left out", id: id, exchange: xnys, days: []calendar.TradingDay{monday, wednesday}, wantErr: "must be the date after"},
		{name: "no business day", id: id, exchange: xnys, days: []calendar.TradingDay{saturday}, wantErr: "no business day"},
		{name: "days overlap", id: id, exchange: xnys, days: []calendar.TradingDay{monday, tuesdayEarly}, wantErr: "before day 2026-11-16 closes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := calendar.NewTradingCalendar(tt.id, tt.exchange, tt.days)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestTradingCalendarIdentityAndCoverage(t *testing.T) {
	t.Parallel()

	c := newXNYS(t)
	from, to := c.Coverage()

	assert.True(t, c.IsValid())
	assert.False(t, calendar.TradingCalendar{}.IsValid())
	assert.Equal(t, "XNYS", c.ID().String())
	assert.Equal(t, "XNYS", c.Exchange().Code().String())
	assert.Equal(t, mustDate(t, "2026-11-16"), from)
	assert.Equal(t, mustDate(t, "2026-12-11"), to)
}

func TestTradingCalendarShowsInstantsInTheExchangeZone(t *testing.T) {
	t.Parallel()

	xnys := mustExchange(t, "XNYS")
	session := mustSession(t, calendar.SessionKindRegular, time.UTC, "2026-11-16 14:30", "2026-11-16 21:00")

	day, err := calendar.NewTradingDay(mustDate(t, "2026-11-16"), []calendar.Session{session}, false, false)
	require.NoError(t, err)

	c, err := calendar.NewTradingCalendar(mustID(t, "XNYS"), xnys, []calendar.TradingDay{day})
	require.NoError(t, err)

	got, ok, err := c.SessionAt(session.Open())
	require.NoError(t, err)
	require.True(t, ok)

	assert.Equal(t, xnys.Location(), got.Open().Location())
	assert.Equal(t, mustAt(t, xnys.Location(), "2026-11-16 09:30"), got.Open())
	assert.True(t, got.Open().Equal(session.Open()))
}

func TestSessionAtAndDayAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		market    market
		at        string
		wantOpen  string
		wantClose string
		wantDate  string
		wantOK    bool
		wantErr   error
	}{
		{
			name: "inside regular hours", market: newXNYS, at: "2026-11-16 10:00",
			wantOpen: "2026-11-16 09:30", wantClose: "2026-11-16 16:00", wantDate: "2026-11-16", wantOK: true,
		},
		{
			name: "at the open", market: newXNYS, at: "2026-11-17 09:30",
			wantOpen: "2026-11-17 09:30", wantClose: "2026-11-17 16:00", wantDate: "2026-11-17", wantOK: true,
		},
		{name: "at the close", market: newXNYS, at: "2026-11-16 16:00"},
		{name: "overnight", market: newXNYS, at: "2026-11-16 22:00"},
		{name: "on a holiday", market: newXNYS, at: "2026-11-26 12:00"},
		{name: "in the lunch break", market: newXTKS, at: "2026-12-07 12:00"},
		{
			name: "pre-market", market: newXNYSExtended, at: "2026-11-17 08:00",
			wantOpen: "2026-11-17 04:00", wantClose: "2026-11-17 09:30", wantDate: "2026-11-17", wantOK: true,
		},
		{
			name: "futures on Sunday evening belong to Monday", market: newFutures, at: "2026-11-15 22:35",
			wantOpen: "2026-11-15 17:00", wantClose: "2026-11-16 16:00", wantDate: "2026-11-16", wantOK: true,
		},
		{name: "futures on Saturday", market: newFutures, at: "2026-11-14 12:00"},
		{name: "futures in the daily break", market: newFutures, at: "2026-11-16 16:30"},
		{name: "before the first open", market: newXNYS, at: "2026-11-16 09:00", wantErr: calendar.ErrOutOfCoverage},
		{name: "at the last close", market: newXNYS, at: "2026-12-11 16:00", wantErr: calendar.ErrOutOfCoverage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := tt.market(t)
			loc := c.Exchange().Location()
			at := mustAt(t, loc, tt.at)

			session, ok, err := c.SessionAt(at)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantOK, ok)

			day, dayOK, err := c.DayAt(at)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantOK, dayOK)

			if !tt.wantOK {
				return
			}

			assert.Equal(t, mustAt(t, loc, tt.wantOpen), session.Open())
			assert.Equal(t, mustAt(t, loc, tt.wantClose), session.Close())
			assert.Equal(t, mustDate(t, tt.wantDate), day.Date())
		})
	}
}

func TestNextSessionAndPreviousSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		market       market
		at           string
		wantNext     string
		wantNextErr  error
		wantPrevious string
		wantPrevErr  error
	}{
		{
			name: "between two days", market: newXNYS, at: "2026-11-16 22:00",
			wantNext: "2026-11-17 09:30", wantPrevious: "2026-11-16 16:00",
		},
		{
			name: "inside a session", market: newXNYS, at: "2026-11-17 10:00",
			wantNext: "2026-11-18 09:30", wantPrevious: "2026-11-16 16:00",
		},
		{
			name: "at a close", market: newXNYS, at: "2026-11-17 16:00",
			wantNext: "2026-11-18 09:30", wantPrevious: "2026-11-17 16:00",
		},
		{
			name: "in the lunch break", market: newXTKS, at: "2026-12-08 12:00",
			wantNext: "2026-12-08 12:30", wantPrevious: "2026-12-08 11:30",
		},
		{
			name: "across a holiday", market: newXNYS, at: "2026-11-26 12:00",
			wantNext: "2026-11-27 09:30", wantPrevious: "2026-11-25 16:00",
		},
		{
			name: "touching sessions", market: newCrypto, at: "2026-11-10 12:00",
			wantNext: "2026-11-11 00:00", wantPrevious: "2026-11-10 00:00",
		},
		{
			name: "in the first session", market: newXNYS, at: "2026-11-16 10:00",
			wantNext: "2026-11-17 09:30", wantPrevErr: calendar.ErrOutOfCoverage,
		},
		{
			name: "in the last session", market: newXNYS, at: "2026-12-11 10:00",
			wantNextErr: calendar.ErrOutOfCoverage, wantPrevious: "2026-12-10 16:00",
		},
		{
			name: "outside coverage", market: newXNYS, at: "2026-12-12 10:00",
			wantNextErr: calendar.ErrOutOfCoverage, wantPrevErr: calendar.ErrOutOfCoverage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := tt.market(t)
			loc := c.Exchange().Location()
			at := mustAt(t, loc, tt.at)

			next, err := c.NextSession(at)
			require.ErrorIs(t, err, tt.wantNextErr)

			if tt.wantNextErr == nil {
				assert.Equal(t, mustAt(t, loc, tt.wantNext), next.Open())
			}

			previous, err := c.PreviousSession(at)
			require.ErrorIs(t, err, tt.wantPrevErr)

			if tt.wantPrevErr == nil {
				assert.Equal(t, mustAt(t, loc, tt.wantPrevious), previous.Close())
			}
		})
	}
}

func TestOpenUntil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		market  market
		at      string
		want    string
		wantOK  bool
		wantErr error
	}{
		{name: "one session", market: newXNYS, at: "2026-11-16 15:00", want: "2026-11-16 16:00", wantOK: true},
		{name: "regular hours run into post-market", market: newXNYSExtended, at: "2026-11-16 15:00", want: "2026-11-16 20:00", wantOK: true},
		{name: "before a lunch break", market: newXTKS, at: "2026-12-07 10:00", want: "2026-12-07 11:30", wantOK: true},
		{name: "a week without a break", market: newFX, at: "2026-11-11 10:00", want: "2026-11-13 17:00", wantOK: true},
		{name: "no session open", market: newXNYS, at: "2026-11-16 22:00"},
		{name: "a market that never stops", market: newCrypto, at: "2026-11-10 12:00", wantErr: calendar.ErrOutOfCoverage},
		{name: "the last session", market: newXNYS, at: "2026-12-11 10:00", wantErr: calendar.ErrOutOfCoverage},
		{name: "outside coverage", market: newXNYS, at: "2026-12-12 10:00", wantErr: calendar.ErrOutOfCoverage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := tt.market(t)
			loc := c.Exchange().Location()

			until, ok, err := c.OpenUntil(mustAt(t, loc, tt.at))
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantOK, ok)

			if tt.wantOK {
				assert.Equal(t, mustAt(t, loc, tt.want), until)
			}
		})
	}
}

func TestFXSessionClosesAtTheDailyRoll(t *testing.T) {
	t.Parallel()

	c := newFX(t)
	loc := c.Exchange().Location()

	session, ok, err := c.SessionAt(mustAt(t, loc, "2026-11-11 10:00"))
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, mustAt(t, loc, "2026-11-11 17:00"), session.Close())
}

func TestDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		market         market
		date           string
		wantOK         bool
		wantHoliday    bool
		wantWeekend    bool
		wantOpen       string
		wantClose      string
		wantFull       bool
		wantEarlyClose bool
		wantErr        error
	}{
		{
			name: "full day", market: newXNYS, date: "2026-11-25", wantOK: true,
			wantOpen: "2026-11-25 09:30", wantClose: "2026-11-25 16:00", wantFull: true,
		},
		{
			name: "early close", market: newXNYS, date: "2026-11-27", wantOK: true,
			wantOpen: "2026-11-27 09:30", wantClose: "2026-11-27 13:00", wantEarlyClose: true,
		},
		{name: "holiday", market: newXNYS, date: "2026-11-26", wantHoliday: true},
		{name: "Saturday", market: newXNYS, date: "2026-11-21", wantWeekend: true},
		{
			name: "futures Monday opens on Sunday", market: newFutures, date: "2026-11-16", wantOK: true,
			wantOpen: "2026-11-15 17:00", wantClose: "2026-11-16 16:00", wantFull: true,
		},
		{name: "futures Sunday", market: newFutures, date: "2026-11-15", wantWeekend: true},
		{name: "before coverage", market: newXNYS, date: "2026-11-15", wantErr: calendar.ErrOutOfCoverage},
		{name: "weekend after the last day", market: newXNYS, date: "2026-12-12", wantErr: calendar.ErrOutOfCoverage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := tt.market(t)
			loc := c.Exchange().Location()
			date := mustDate(t, tt.date)

			day, err := c.Day(date)
			require.ErrorIs(t, err, tt.wantErr)

			isTradingDay, err := c.IsTradingDay(date)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantOK, isTradingDay)

			if tt.wantErr != nil {
				return
			}

			assert.Equal(t, date, day.Date())
			assert.Equal(t, tt.wantOK, day.IsBusinessDay())
			assert.Equal(t, tt.wantHoliday, day.IsHoliday())
			assert.Equal(t, tt.wantWeekend, day.IsWeekend())
			assert.Equal(t, tt.wantFull, day.IsFullDay())
			assert.Equal(t, tt.wantEarlyClose, day.IsEarlyClose())

			if !tt.wantOK {
				_, opens := day.Open()
				_, closes := day.Close()

				assert.False(t, opens)
				assert.False(t, closes)
				assert.Empty(t, day.Sessions())
				assert.Zero(t, day.Duration())

				return
			}

			assert.Equal(t, mustAt(t, loc, tt.wantOpen), mustOpen(t, day))
			assert.Equal(t, mustAt(t, loc, tt.wantClose), mustClose(t, day))
		})
	}
}

func TestDayAtOrBefore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		market  market
		at      string
		want    string
		wantErr error
	}{
		{name: "inside regular hours", market: newXNYS, at: "2026-11-16 10:00", want: "2026-11-16"},
		{name: "at the close", market: newXNYS, at: "2026-11-16 16:00", want: "2026-11-16"},
		{name: "overnight", market: newXNYS, at: "2026-11-16 22:00", want: "2026-11-16"},
		{name: "just before the next open", market: newXNYS, at: "2026-11-17 09:29", want: "2026-11-16"},
		{name: "at the next open", market: newXNYS, at: "2026-11-17 09:30", want: "2026-11-17"},
		{name: "over a weekend", market: newXNYS, at: "2026-11-22 12:00", want: "2026-11-20"},
		{name: "on a holiday", market: newXNYS, at: "2026-11-26 12:00", want: "2026-11-25"},
		{name: "after an early close", market: newXNYS, at: "2026-11-27 14:00", want: "2026-11-27"},
		{name: "in the lunch break", market: newXTKS, at: "2026-12-07 12:00", want: "2026-12-07"},
		{name: "between extended sessions of two days", market: newXNYSExtended, at: "2026-11-17 02:00", want: "2026-11-16"},
		{name: "futures on Sunday evening", market: newFutures, at: "2026-11-15 22:35", want: "2026-11-16"},
		{name: "futures in the daily break", market: newFutures, at: "2026-11-16 16:30", want: "2026-11-16"},
		{name: "futures on Saturday", market: newFutures, at: "2026-11-14 12:00", want: "2026-11-13"},
		{name: "crypto at the midnight roll", market: newCrypto, at: "2026-11-11 00:00", want: "2026-11-11"},
		{name: "before the first open", market: newXNYS, at: "2026-11-16 09:00", wantErr: calendar.ErrOutOfCoverage},
		{name: "at the last close", market: newXNYS, at: "2026-12-11 16:00", wantErr: calendar.ErrOutOfCoverage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := tt.market(t)

			day, err := c.DayAtOrBefore(mustAt(t, c.Exchange().Location(), tt.at))
			require.ErrorIs(t, err, tt.wantErr)

			if tt.wantErr == nil {
				assert.Equal(t, mustDate(t, tt.want), day.Date())
			}
		})
	}
}

func TestCoverageMayEndOnAWeekend(t *testing.T) {
	t.Parallel()

	week := mustWeek(t, "2026-01-01", workweek(), mustHours(t, calendar.SessionKindRegular, "09:30", "16:00", 0))
	c := mustBuild(t, mustExchange(t, "XNYS"), "2026-11-14", "2026-11-22", week, nil)

	from, to := c.Coverage()
	assert.Equal(t, mustDate(t, "2026-11-14"), from)
	assert.Equal(t, mustDate(t, "2026-11-22"), to)

	sunday, err := c.Day(mustDate(t, "2026-11-22"))
	require.NoError(t, err)
	assert.True(t, sunday.IsWeekend())

	next, err := c.NextTradingDay(mustDate(t, "2026-11-14"))
	require.NoError(t, err)
	assert.Equal(t, mustDate(t, "2026-11-16"), next)

	previous, err := c.PreviousTradingDay(mustDate(t, "2026-11-22"))
	require.NoError(t, err)
	assert.Equal(t, mustDate(t, "2026-11-20"), previous)

	_, err = c.NextTradingDay(mustDate(t, "2026-11-20"))
	require.ErrorIs(t, err, calendar.ErrOutOfCoverage)

	_, err = c.PreviousTradingDay(mustDate(t, "2026-11-16"))
	require.ErrorIs(t, err, calendar.ErrOutOfCoverage)
}

func TestTradingDayOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		market  market
		date    string
		n       int
		want    string
		wantErr error
	}{
		{name: "next day", market: newXNYS, date: "2026-11-17", n: 1, want: "2026-11-18"},
		{name: "over a holiday", market: newXNYS, date: "2026-11-25", n: 1, want: "2026-11-27"},
		{name: "back over a holiday", market: newXNYS, date: "2026-11-27", n: -1, want: "2026-11-25"},
		{name: "from a holiday", market: newXNYS, date: "2026-11-26", n: 1, want: "2026-11-27"},
		{name: "back from a holiday", market: newXNYS, date: "2026-11-26", n: -1, want: "2026-11-25"},
		{name: "back from a Saturday", market: newXNYS, date: "2026-11-21", n: -3, want: "2026-11-18"},
		{name: "forward from a Saturday", market: newXNYS, date: "2026-11-21", n: 2, want: "2026-11-24"},
		{name: "zero on a trading day", market: newXNYS, date: "2026-11-20", n: 0, want: "2026-11-20"},
		{name: "zero on a Saturday", market: newXNYS, date: "2026-11-21", n: 0, wantErr: calendar.ErrNotTradingDay},
		{name: "futures Friday to Monday", market: newFutures, date: "2026-11-13", n: 1, want: "2026-11-16"},
		{name: "from a Sunday onto the last day", market: newXNYS, date: "2026-12-06", n: 5, want: "2026-12-11"},
		{name: "from a Sunday past the last day", market: newXNYS, date: "2026-12-06", n: 6, wantErr: calendar.ErrOutOfCoverage},
		{name: "back from a Saturday onto the first day", market: newXNYS, date: "2026-11-21", n: -5, want: "2026-11-16"},
		{name: "back from a Saturday before the first day", market: newXNYS, date: "2026-11-21", n: -6, wantErr: calendar.ErrOutOfCoverage},
		{name: "past the last day", market: newXNYS, date: "2026-12-11", n: 1, wantErr: calendar.ErrOutOfCoverage},
		{name: "before the first day", market: newXNYS, date: "2026-11-16", n: -1, wantErr: calendar.ErrOutOfCoverage},
		{name: "far past the last day", market: newXNYS, date: "2026-11-16", n: 1 << 40, wantErr: calendar.ErrOutOfCoverage},
		{name: "date outside coverage", market: newXNYS, date: "2027-01-04", n: 1, wantErr: calendar.ErrOutOfCoverage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.market(t).TradingDayOffset(mustDate(t, tt.date), tt.n)
			require.ErrorIs(t, err, tt.wantErr)

			if tt.wantErr == nil {
				assert.Equal(t, mustDate(t, tt.want), got)
			}
		})
	}
}

func TestNextTradingDayAndPreviousTradingDay(t *testing.T) {
	t.Parallel()

	c := newXNYS(t)

	next, err := c.NextTradingDay(mustDate(t, "2026-11-20"))
	require.NoError(t, err)
	assert.Equal(t, mustDate(t, "2026-11-23"), next)

	previous, err := c.PreviousTradingDay(mustDate(t, "2026-11-22"))
	require.NoError(t, err)
	assert.Equal(t, mustDate(t, "2026-11-20"), previous)

	_, err = c.NextTradingDay(mustDate(t, "2026-12-11"))
	require.ErrorIs(t, err, calendar.ErrOutOfCoverage)
}

func TestTradingDaysBetween(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		from    string
		to      string
		want    []string
		wantErr string
	}{
		{name: "a week with a holiday", from: "2026-11-23", to: "2026-11-27", want: []string{"2026-11-23", "2026-11-24", "2026-11-25", "2026-11-27"}},
		{name: "ends on a Sunday", from: "2026-11-19", to: "2026-11-22", want: []string{"2026-11-19", "2026-11-20"}},
		{name: "one trading day", from: "2026-11-23", to: "2026-11-23", want: []string{"2026-11-23"}},
		{name: "a weekend", from: "2026-11-21", to: "2026-11-22", want: []string{}},
		{name: "starts on a holiday", from: "2026-11-26", to: "2026-11-30", want: []string{"2026-11-27", "2026-11-30"}},
		{name: "ends on a holiday", from: "2026-11-25", to: "2026-11-26", want: []string{"2026-11-25"}},
		{name: "a holiday alone", from: "2026-11-26", to: "2026-11-26", want: []string{}},
		{name: "the first day alone", from: "2026-11-16", to: "2026-11-16", want: []string{"2026-11-16"}},
		{name: "the last day alone", from: "2026-12-11", to: "2026-12-11", want: []string{"2026-12-11"}},
		{
			name: "the whole coverage", from: "2026-11-16", to: "2026-12-11",
			want: []string{
				"2026-11-16", "2026-11-17", "2026-11-18", "2026-11-19", "2026-11-20",
				"2026-11-23", "2026-11-24", "2026-11-25", "2026-11-27",
				"2026-11-30", "2026-12-01", "2026-12-02", "2026-12-03", "2026-12-04",
				"2026-12-07", "2026-12-08", "2026-12-09", "2026-12-10", "2026-12-11",
			},
		},
		{name: "reversed", from: "2026-11-27", to: "2026-11-23", wantErr: "the first date is after the second"},
		{name: "starts before coverage", from: "2026-11-01", to: "2026-11-23", wantErr: calendar.ErrOutOfCoverage.Error()},
		{name: "ends after coverage", from: "2026-11-23", to: "2026-12-31", wantErr: calendar.ErrOutOfCoverage.Error()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newXNYS(t).TradingDaysBetween(mustDate(t, tt.from), mustDate(t, tt.to))
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, mustDates(t, tt.want...), got)
		})
	}
}

func TestCryptoTradesEveryDate(t *testing.T) {
	t.Parallel()

	c := newCrypto(t)

	days, err := c.TradingDaysBetween(mustDate(t, "2026-11-09"), mustDate(t, "2026-11-15"))
	require.NoError(t, err)
	assert.Len(t, days, 7)

	day, err := c.Day(mustDate(t, "2026-11-10"))
	require.NoError(t, err)
	require.True(t, day.IsBusinessDay())
	assert.Equal(t, 24*time.Hour, day.Duration())
}

func TestTokyoDayHasALunchBreak(t *testing.T) {
	t.Parallel()

	day, err := newXTKS(t).Day(mustDate(t, "2026-12-07"))
	require.NoError(t, err)
	require.True(t, day.IsBusinessDay())

	assert.Len(t, day.Sessions(), 2)
	assert.Equal(t, 5*time.Hour+30*time.Minute, day.Duration())
	assert.Equal(t, 6*time.Hour+30*time.Minute, mustClose(t, day).Sub(mustOpen(t, day)))
}
