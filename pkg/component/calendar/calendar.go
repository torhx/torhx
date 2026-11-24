package calendar

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/talav/civil"
	"github.com/torhx/exchange"
)

var (
	// ErrOutOfCoverage is returned when a single-date or instant query needs
	// data beyond the calendar's coverage.
	ErrOutOfCoverage = errors.New("outside the calendar's coverage")

	// ErrNotTradingDay is returned when a query needs a date the market
	// trades on and is given one it does not.
	ErrNotTradingDay = errors.New("not a trading day")
)

// A TradingCalendar says when one market trades: an exchange, or a product
// on it with hours of its own. It holds a day for every date from its first
// to its last, business days, weekends and holidays alike, and that range is
// its coverage. Single-date and instant queries return [ErrOutOfCoverage]
// when they need a date or instant outside that coverage.
//
// In its queries a trading day is a date whose day is a business day.
//
// Coverage is strict at both ends, because the calendar cannot tell what
// trades next to it: before the first open it does not name the next
// session, and in the last run of trading it does not say when trading
// stops. A caller that asks about the edges, such as a live system at the
// start of its first day, loads a day more at each end.
//
// A TradingCalendar comes only from a constructor and never changes, so it is
// safe for concurrent use by multiple goroutines. Every instant it returns is
// shown in the exchange's time zone. The zero value is not a calendar.
type TradingCalendar struct {
	// id names the calendar among those of its exchange.
	id ID

	// exchange is the exchange the calendar belongs to. Its time zone is the
	// calendar's.
	exchange exchange.Exchange

	// coverage runs from the first day's date up to the date after the last
	// day, including holidays and weekends.
	coverage civil.DateRange

	// days holds one day for every date from the first to the last, in date
	// order with no date left out, so the day of a date is found by its
	// distance from the first.
	days []TradingDay

	// business holds the indexes in days of the business days, in date
	// order, at least one. The sessions of one business day end before those
	// of the next begin, so the business days are also in time order.
	business []int
}

// NewTradingCalendar returns the calendar named id for the exchange e,
// holding the given days.
//
// It returns an error if id is the zero ID, if e is the zero exchange, if
// there is no day or one is the zero day, if the dates of the days do not
// follow one another with none left out, if no day is a business day, or if
// a business day's first session opens before the last session of the
// business day before it closes.
func NewTradingCalendar(id ID, e exchange.Exchange, days []TradingDay) (TradingCalendar, error) {
	c := TradingCalendar{id: id, exchange: e, days: days}
	if err := c.validate(); err != nil {
		return TradingCalendar{}, fmt.Errorf("invalid trading calendar %s: %w", id, err)
	}

	coverage, err := civil.NewDateRange(days[0].date, days[len(days)-1].date.AddDays(1))
	if err != nil {
		return TradingCalendar{}, fmt.Errorf("invalid coverage for calendar %s: %w", id, err)
	}

	c.coverage = coverage

	// The days are rewritten only now, because showing an instant in the
	// exchange's zone needs an exchange that has passed validation.
	c.days = make([]TradingDay, len(days))

	for i, day := range days {
		c.days[i] = day.in(e.Location())

		if day.IsBusinessDay() {
			c.business = append(c.business, i)
		}
	}

	return c, nil
}

// ID returns the name of the calendar.
func (c TradingCalendar) ID() ID { return c.id }

// Exchange returns the exchange the calendar belongs to.
func (c TradingCalendar) Exchange() exchange.Exchange { return c.exchange }

// IsValid reports whether c is a calendar. It is false only for the zero
// value, which stands for a missing one; every TradingCalendar from a
// constructor is valid.
func (c TradingCalendar) IsValid() bool { return len(c.days) > 0 }

// Coverage returns the range of dates the calendar holds. Its start is the
// first calendar date and its end is the day after the last. The zero
// calendar has a zero range. Instant queries include the first business
// day's first open and exclude the last business day's last close.
func (c TradingCalendar) Coverage() civil.DateRange { return c.coverage }

// SessionAt returns the session of any kind that is open at the instant t.
// It reports false in a break or any other gap between sessions.
//
// It returns [ErrOutOfCoverage] if t is outside coverage.
func (c TradingCalendar) SessionAt(t time.Time) (Session, bool, error) {
	if !c.coversInstant(t) {
		return Session{}, false, ErrOutOfCoverage
	}

	day, session, ok := c.locate(t)
	if !ok {
		return Session{}, false, nil
	}

	return c.businessDay(day).sessions[session], true, nil
}

// NextSession returns the first session that opens after the instant t.
//
// It returns [ErrOutOfCoverage] if t is outside coverage or no session opens
// after it.
func (c TradingCalendar) NextSession(t time.Time) (Session, error) {
	if !c.coversInstant(t) {
		return Session{}, ErrOutOfCoverage
	}

	day := c.businessIndexAt(t)
	for _, s := range c.businessDay(day).sessions {
		if s.open.After(t) {
			return s, nil
		}
	}

	if day+1 == len(c.business) {
		return Session{}, ErrOutOfCoverage
	}

	return c.businessDay(day + 1).sessions[0], nil
}

// PreviousSession returns the last session that closes at or before the
// instant t.
//
// It returns [ErrOutOfCoverage] if t is outside coverage or no session
// closes at or before it.
func (c TradingCalendar) PreviousSession(t time.Time) (Session, error) {
	if !c.coversInstant(t) {
		return Session{}, ErrOutOfCoverage
	}

	day := c.businessIndexAt(t)
	for _, s := range slices.Backward(c.businessDay(day).sessions) {
		if !s.close.After(t) {
			return s, nil
		}
	}

	if day == 0 {
		return Session{}, ErrOutOfCoverage
	}

	previous := c.businessDay(day - 1).sessions

	return previous[len(previous)-1], nil
}

// OpenUntil returns the instant trading stops for a market that is open at
// the instant t: the first instant at or after t at which no session is
// open. It follows sessions that touch, so the close of a regular session
// that runs straight into an extended one is not the answer, and neither is
// the daily roll of a market that never stops. It reports false when no
// session is open at t.
//
// It returns [ErrOutOfCoverage] if t is outside coverage, or if trading runs
// to the end of coverage, where the calendar does not know when it stops.
func (c TradingCalendar) OpenUntil(t time.Time) (time.Time, bool, error) {
	if !c.coversInstant(t) {
		return time.Time{}, false, ErrOutOfCoverage
	}

	day, session, ok := c.locate(t)
	if !ok {
		return time.Time{}, false, nil
	}

	end := c.businessDay(day).sessions[session].close

	for {
		session++
		if session == len(c.businessDay(day).sessions) {
			day, session = day+1, 0
		}

		if day == len(c.business) {
			return time.Time{}, false, ErrOutOfCoverage
		}

		next := c.businessDay(day).sessions[session]
		if !next.open.Equal(end) {
			return end, true, nil
		}

		end = next.close
	}
}

// DayAt returns the business day the instant t belongs to: the day of the
// session open at t. It reports false when no session is open at t.
//
// It returns [ErrOutOfCoverage] if t is outside coverage.
func (c TradingCalendar) DayAt(t time.Time) (TradingDay, bool, error) {
	if !c.coversInstant(t) {
		return TradingDay{}, false, ErrOutOfCoverage
	}

	day, _, ok := c.locate(t)
	if !ok {
		return TradingDay{}, false, nil
	}

	return *c.businessDay(day), true, nil
}

// DayAtOrBefore returns the business day in progress at the instant t, or
// the last one before it when none is: the day of the session open at t, or
// else of the last session that closed at or before t. It gives a business
// day to what happens between sessions, such as a closing-auction print just
// after the close, which belongs to the day that has just ended.
//
// It returns [ErrOutOfCoverage] if t is outside coverage.
func (c TradingCalendar) DayAtOrBefore(t time.Time) (TradingDay, error) {
	if !c.coversInstant(t) {
		return TradingDay{}, ErrOutOfCoverage
	}

	// This is the only business day that can have opened and not yet
	// finished at t. When it has not opened, t is in the gap after the one
	// before it, which exists because coverage starts at the first open.
	day := c.businessIndexAt(t)
	if t.Before(c.businessDay(day).firstOpen()) {
		day--
	}

	return *c.businessDay(day), nil
}

// Day returns the day dated d: a business day, a weekend or a holiday.
//
// It returns [ErrOutOfCoverage] if d is outside coverage.
func (c TradingCalendar) Day(d civil.Date) (TradingDay, error) {
	if !c.coversDate(d) {
		return TradingDay{}, ErrOutOfCoverage
	}

	return c.days[d.DaysSince(c.coverage.Start())], nil
}

// IsTradingDay reports whether the market trades on d.
//
// It returns [ErrOutOfCoverage] if d is outside coverage.
func (c TradingCalendar) IsTradingDay(d civil.Date) (bool, error) {
	day, err := c.Day(d)
	if err != nil {
		return false, err
	}

	return day.IsBusinessDay(), nil
}

// NextTradingDay returns the first trading day after d. d need not be a
// trading day.
//
// It returns [ErrOutOfCoverage] if d is outside coverage or no trading day
// follows it.
func (c TradingCalendar) NextTradingDay(d civil.Date) (civil.Date, error) {
	return c.TradingDayOffset(d, 1)
}

// PreviousTradingDay returns the last trading day before d. d need not be a
// trading day.
//
// It returns [ErrOutOfCoverage] if d is outside coverage or no trading day
// comes before it.
func (c TradingCalendar) PreviousTradingDay(d civil.Date) (civil.Date, error) {
	return c.TradingDayOffset(d, -1)
}

// TradingDayOffset returns the nth trading day after d, or before it when n
// is negative. d itself is not counted and need not be a trading day. With n
// of 0 it returns d, which must then be a trading day.
//
// It returns [ErrNotTradingDay] if n is 0 and d is not a trading day, and
// [ErrOutOfCoverage] if d is outside coverage or the count runs past it.
func (c TradingCalendar) TradingDayOffset(d civil.Date, n int) (civil.Date, error) {
	if !c.coversDate(d) || n > len(c.business) || n < -len(c.business) {
		return civil.Date{}, ErrOutOfCoverage
	}

	// i is the business day dated d when there is one, and otherwise the
	// first one after d. Either way the last business day before d is i-1.
	i, ok := c.businessIndexOn(d)

	target := i + n

	switch {
	case n == 0 && !ok:
		return civil.Date{}, ErrNotTradingDay
	case n > 0 && !ok:
		target--
	}

	if target < 0 || target >= len(c.business) {
		return civil.Date{}, ErrOutOfCoverage
	}

	return c.businessDay(target).date, nil
}

// TradingDaysBetween returns the trading days in period, in date order.
// The start is included and the end is excluded.
//
// It returns an empty slice if period has no trading days, is the zero
// range, or extends outside coverage. A partial list would undercount the
// trading days in a period the calendar cannot fully answer for.
func (c TradingCalendar) TradingDaysBetween(period civil.DateRange) []civil.Date {
	if !period.IsValid() || period.Start().Before(c.coverage.Start()) || period.End().After(c.coverage.End()) {
		return []civil.Date{}
	}

	first, _ := c.businessIndexOn(period.Start())
	after, _ := c.businessIndexOn(period.End())
	business := c.business[first:after]

	dates := make([]civil.Date, len(business))
	for i, dayIndex := range business {
		dates[i] = c.days[dayIndex].date
	}

	return dates
}

//nolint:cyclop // one flat list of checks reads better than a split
func (c TradingCalendar) validate() error {
	if !c.id.IsValid() {
		return errors.New("ID is missing")
	}

	if !c.exchange.IsValid() {
		return errors.New("exchange is missing")
	}

	if len(c.days) == 0 {
		return errors.New("no day")
	}

	var previous *TradingDay

	for i, day := range c.days {
		if !day.IsValid() {
			return fmt.Errorf("day %d is not a day", i)
		}

		if i > 0 && day.date != c.days[i-1].date.AddDays(1) {
			return fmt.Errorf("day %s must be the date after day %s", day.date, c.days[i-1].date)
		}

		if !day.IsBusinessDay() {
			continue
		}

		if previous != nil && day.firstOpen().Before(previous.lastClose()) {
			return fmt.Errorf("day %s opens at %s, before day %s closes", day.date, day.firstOpen(), previous.date)
		}

		previous = &c.days[i]
	}

	if previous == nil {
		return errors.New("no business day")
	}

	return nil
}

// businessDay returns the business day at position i among the business
// days. It returns a pointer so that a search does not copy every day it
// looks at.
func (c TradingCalendar) businessDay(i int) *TradingDay { return &c.days[c.business[i]] }

// locate finds the session open at the instant t, which is inside coverage,
// and returns the position of its day among the business days and its index
// within that day. It reports false when no session is open at t.
func (c TradingCalendar) locate(t time.Time) (day, session int, ok bool) {
	day = c.businessIndexAt(t)
	for i, s := range c.businessDay(day).sessions {
		if s.Contains(t) {
			return day, i, true
		}
	}

	return 0, 0, false
}

// coversInstant reports whether t is at or after the first business day's
// first open and before the last business day's last close.
func (c TradingCalendar) coversInstant(t time.Time) bool {
	return !t.Before(c.businessDay(0).firstOpen()) && t.Before(c.businessDay(len(c.business)-1).lastClose())
}

// coversDate excludes the zero date even when coverage spans year zero.
func (c TradingCalendar) coversDate(d civil.Date) bool {
	return d.IsValid() && c.coverage.Contains(d)
}

// businessIndexAt returns the position among the business days of the first
// one whose last session closes after t. For an instant inside coverage
// there always is one, and it is the only day that can hold a session open
// at t.
func (c TradingCalendar) businessIndexAt(t time.Time) int {
	return sort.Search(len(c.business), func(i int) bool { return c.businessDay(i).lastClose().After(t) })
}

// businessIndexOn returns the position among the business days of the first
// one dated d or later, and whether that day is dated d. The position is the
// number of business days when every one is before d.
func (c TradingCalendar) businessIndexOn(d civil.Date) (int, bool) {
	i := sort.Search(len(c.business), func(i int) bool { return !c.businessDay(i).date.Before(d) })

	return i, i < len(c.business) && c.businessDay(i).date == d
}
