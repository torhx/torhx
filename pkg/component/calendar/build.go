package calendar

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/talav/civil"
	"github.com/torhx/exchange"
)

// BuildTradingCalendar returns the calendar named id for the exchange e,
// holding one day for each date in coverage, from its inclusive start up to
// its exclusive end.
//
// Each date takes the hours of its weekday from the schedule version in
// force on it. A weekday with no hours is a weekend. exceptions lists the
// dates that differ: each gives every session of that date's day, in place of
// the scheduled ones. An empty list closes the date: it is a holiday, or
// still a weekend when its weekday has no hours. A weekend date listed with
// sessions is a business day. A date that is not listed follows the
// schedule, so the range asserts that every such date does.
//
// A scheduled day is never a late open or an early close. An exception day
// is a late open when its first regular session opens after the first
// regular open scheduled for that date, and an early close when its last
// regular session closes before the last regular close scheduled for it.
// With no regular hours scheduled for the date it is neither.
//
// A scheduled time of 00:00 is the first instant of its civil date. Any
// other scheduled time must be shown exactly once by a clock in the
// exchange's zone on its date; a date on which one is skipped or repeated
// needs an exception, which gives instants.
//
// It returns an error if e is the zero exchange, if coverage is the zero
// range, if no schedule applies at the start, if the versions are out of
// order, or if an exception falls outside coverage. It also returns an
// error if a scheduled time is skipped or repeated on a date without an
// exception, or if the days do not make a calendar.
func BuildTradingCalendar(id ID, e exchange.Exchange, coverage civil.DateRange, schedules []WeeklySchedule, exceptions map[civil.Date][]Session) (TradingCalendar, error) {
	if err := validateBuild(e, coverage, schedules, exceptions); err != nil {
		return TradingCalendar{}, fmt.Errorf("build trading calendar %s: %w", id, err)
	}

	days := make([]TradingDay, 0, coverage.Days())

	for date := coverage.Start(); date.Before(coverage.End()); date = date.AddDays(1) {
		exception, isException := exceptions[date]

		day, err := buildDay(date, e.Location(), versionOn(schedules, date), exception, isException)
		if err != nil {
			return TradingCalendar{}, fmt.Errorf("build trading calendar %s: %w", id, err)
		}

		days = append(days, day)
	}

	return NewTradingCalendar(id, e, days)
}

//nolint:cyclop // one flat list of checks reads better than a split
func validateBuild(e exchange.Exchange, coverage civil.DateRange, schedules []WeeklySchedule, exceptions map[civil.Date][]Session) error {
	if !e.IsValid() {
		return errors.New("exchange is missing")
	}

	if !coverage.IsValid() {
		return errors.New("coverage is missing")
	}

	if len(schedules) == 0 {
		return errors.New("no weekly schedule")
	}

	for i, schedule := range schedules {
		if !schedule.from.IsValid() {
			return fmt.Errorf("weekly schedule %d is not a schedule", i)
		}

		if i > 0 && !schedule.from.After(schedules[i-1].from) {
			return fmt.Errorf("weekly schedule from %s must come after the one from %s", schedule.from, schedules[i-1].from)
		}
	}

	if schedules[0].from.After(coverage.Start()) {
		return fmt.Errorf("no weekly schedule is in force on %s", coverage.Start())
	}

	// Map order is random, so the earliest stray date is found and named,
	// which keeps the error the same from one run to the next.
	var stray *civil.Date

	for date := range exceptions {
		if date.IsValid() && coverage.Contains(date) {
			continue
		}

		if stray == nil || date.Before(*stray) {
			stray = &date
		}
	}

	if stray != nil {
		return fmt.Errorf("exception on %s is outside %s", *stray, coverage)
	}

	return nil
}

// versionOn returns the schedule version in force on date: the last one that
// applies from date or earlier. One must exist.
func versionOn(schedules []WeeklySchedule, date civil.Date) WeeklySchedule {
	after := sort.Search(len(schedules), func(i int) bool { return schedules[i].from.After(date) })

	return schedules[after-1]
}

// buildDay returns the day dated date: a business day with the exception's
// sessions when the date is an exception that has some, and with the
// schedule's otherwise. A date left with no session is a weekend when its
// weekday has no scheduled hours, and a holiday when it has.
func buildDay(date civil.Date, loc *time.Location, schedule WeeklySchedule, exception []Session, isException bool) (TradingDay, error) {
	hasHours := len(schedule.days[date.Weekday()]) > 0

	switch {
	case len(exception) > 0:
		return buildExceptionDay(date, loc, schedule, exception)
	case isException && hasHours:
		return NewHoliday(date)
	case !hasHours:
		return NewWeekend(date)
	}

	scheduled, err := schedule.sessions(date, loc)
	if err != nil {
		return TradingDay{}, err
	}

	return NewTradingDay(date, scheduled, false, false)
}

// buildExceptionDay returns the business day dated date with the sessions an
// exception gives it, marked as a late open or an early close against the
// hours scheduled for the date.
func buildExceptionDay(date civil.Date, loc *time.Location, schedule WeeklySchedule, exception []Session) (TradingDay, error) {
	// A scheduled time that does not resolve on this date is no error here:
	// the exception is how such a date is stated. It leaves nothing to
	// compare with, so the day is neither a late open nor an early close.
	lateOpen, earlyClose := false, false
	if scheduled, err := schedule.sessions(date, loc); err == nil {
		lateOpen, earlyClose = deviation(exception, scheduled)
	}

	return NewTradingDay(date, exception, lateOpen, earlyClose)
}

// deviation compares the regular sessions given for a date with those
// scheduled for it. It reports a late open when the given ones start after
// the scheduled ones, and an early close when they end before. Both are
// false when either side has no regular session.
func deviation(given, scheduled []Session) (lateOpen, earlyClose bool) {
	givenOpen, ok := firstRegularOpen(given)
	if !ok {
		return false, false
	}

	scheduledOpen, ok := firstRegularOpen(scheduled)
	if !ok {
		return false, false
	}

	givenClose, _ := lastRegularClose(given)
	scheduledClose, _ := lastRegularClose(scheduled)

	return givenOpen.After(scheduledOpen), givenClose.Before(scheduledClose)
}
