package calendar

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/talav/civil"
	"github.com/torhx/exchange"
)

const (
	daysPerWeek = 7
	hoursPerDay = 24

	// maxDaysBefore is the longest a session can start ahead of its day: a
	// session that starts a week ahead would belong to the week before.
	maxDaysBefore = daysPerWeek - 1

	// startSearchSpan is how far before noon the search for the start of a
	// civil date begins. A day and a half before noon is always on an
	// earlier date, whatever the zone does to its clock.
	startSearchSpan = 36 * time.Hour
)

// Hours is one session written the way an exchange publishes it: wall-clock
// times in its own zone, on no particular date. A weekly schedule holds the
// hours of each weekday.
//
// A session whose end is at or before its start ends on the civil date after
// the one it starts on, so 18:00 to 17:00 runs overnight and 00:00 to 00:00
// is a whole day. A session can start before its day's date: a futures
// session booked to Monday that opens on Sunday evening starts one day
// before.
//
// Hours comes only from a constructor. The zero value is not hours.
type Hours struct {
	// kind says whether the session is regular or extended trading.
	kind SessionKind

	// start is the wall-clock time trading starts.
	start civil.Time

	// end is the wall-clock time trading stops. At or before start, it is on
	// the civil date after the start's.
	end civil.Time

	// daysBefore is how many civil days before its day's date the session
	// starts: 0 for the date itself, 1 for the evening before. It is 0 to 6.
	daysBefore int
}

// NewHours returns the hours of a session of the given kind from start to
// end, starting daysBefore civil days before the date of its day.
//
// It returns an error if kind is not a kind, or if daysBefore is not 0 to 6.
func NewHours(kind SessionKind, start, end civil.Time, daysBefore int) (Hours, error) {
	h := Hours{kind: kind, start: start, end: end, daysBefore: daysBefore}
	if err := h.validate(); err != nil {
		return Hours{}, fmt.Errorf("invalid hours %s-%s: %w", start, end, err)
	}

	return h, nil
}

func (h Hours) validate() error {
	if err := h.kind.validate(); err != nil {
		return err
	}

	if h.daysBefore < 0 || h.daysBefore > maxDaysBefore {
		return fmt.Errorf("days before must be 0 to %d", maxDaysBefore)
	}

	return nil
}

// startOffset returns how long after the first midnight of its day's date
// the session starts, on a day without a clock change. It is negative for a
// session that starts before that date.
func (h Hours) startOffset() time.Duration {
	return sinceMidnight(h.start) - time.Duration(h.daysBefore)*hoursPerDay*time.Hour
}

// endOffset returns how long after the first midnight of its day's date the
// session ends, on a day without a clock change.
func (h Hours) endOffset() time.Duration {
	offset := h.startOffset() - sinceMidnight(h.start) + sinceMidnight(h.end)
	if h.endsNextDate() {
		offset += hoursPerDay * time.Hour
	}

	return offset
}

// endsNextDate reports whether the session ends on the civil date after the
// one it starts on.
func (h Hours) endsNextDate() bool {
	return !h.end.After(h.start)
}

// session returns the session these hours make for the day dated date, with
// its instants in loc.
func (h Hours) session(date civil.Date, loc *time.Location) (Session, error) {
	startDate := date.AddDays(-h.daysBefore)

	endDate := startDate
	if h.endsNextDate() {
		endDate = startDate.AddDays(1)
	}

	openAt, err := instantAt(startDate, h.start, loc)
	if err != nil {
		return Session{}, err
	}

	closeAt, err := instantAt(endDate, h.end, loc)
	if err != nil {
		return Session{}, err
	}

	return NewSession(h.kind, openAt, closeAt)
}

// A WeeklySchedule is a market's normal week from a given date on: the hours
// of each weekday. A later version replaces the whole week from its own
// date, which keeps history right when an exchange changes its hours.
//
// A WeeklySchedule comes only from a constructor. The zero value is not a
// schedule.
type WeeklySchedule struct {
	// from is the first date this version of the week applies to.
	from civil.Date

	// days holds the hours of each weekday, indexed by the time.Weekday of
	// the day's date and in time order. A weekday with no hours is a
	// weekend; any other has at least one regular session.
	days [7][]Hours
}

// NewWeeklySchedule returns the schedule that applies from the date from,
// with the given hours for each weekday, indexed by [time.Weekday]. A
// weekday with no hours is a weekend.
//
// It returns an error if from is the zero date, or if a weekday's hours hold
// the zero Hours, are out of time order or overlap, or include no regular
// session.
//
// The schedule keeps the slices in days without a copy, so the caller must
// not modify them afterwards.
func NewWeeklySchedule(from civil.Date, days [7][]Hours) (WeeklySchedule, error) {
	w := WeeklySchedule{from: from, days: days}
	if err := w.validate(); err != nil {
		return WeeklySchedule{}, fmt.Errorf("invalid weekly schedule from %s: %w", from, err)
	}

	return w, nil
}

func (w WeeklySchedule) validate() error {
	if !w.from.IsValid() {
		return errors.New("first date is missing")
	}

	for weekday, hours := range w.days {
		if err := validateWeekday(hours); err != nil {
			return fmt.Errorf("%s: %w", time.Weekday(weekday), err)
		}
	}

	return nil
}

// sessions returns the sessions the schedule gives the day dated date, with
// their instants in loc. There are none on a weekday without hours.
func (w WeeklySchedule) sessions(date civil.Date, loc *time.Location) ([]Session, error) {
	hours := w.days[date.Weekday()]
	sessions := make([]Session, 0, len(hours))

	for _, h := range hours {
		s, err := h.session(date, loc)
		if err != nil {
			return nil, err
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
}

// BuildTradingCalendar returns the calendar named id for the exchange e,
// holding a day for every date from from to to, both included.
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
// It returns an error if e is the zero exchange, if from is after to, if
// there is no schedule or the versions are not in date order, if none is in
// force on from, if an exception is outside the range, if a scheduled time
// is skipped or repeated on a date without an exception, or if the days do
// not make a calendar.
func BuildTradingCalendar(id ID, e exchange.Exchange, from, to civil.Date, schedules []WeeklySchedule, exceptions map[civil.Date][]Session) (TradingCalendar, error) {
	if err := validateBuild(e, from, to, schedules, exceptions); err != nil {
		return TradingCalendar{}, fmt.Errorf("build trading calendar %s: %w", id, err)
	}

	days := make([]TradingDay, 0, to.DaysSince(from)+1)

	for date := from; !date.After(to); date = date.AddDays(1) {
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
func validateBuild(e exchange.Exchange, from, to civil.Date, schedules []WeeklySchedule, exceptions map[civil.Date][]Session) error {
	if !e.IsValid() {
		return errors.New("exchange is missing")
	}

	if !from.IsValid() || !to.IsValid() || from.After(to) {
		return fmt.Errorf("dates %s to %s are not a range", from, to)
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

	if schedules[0].from.After(from) {
		return fmt.Errorf("no weekly schedule is in force on %s", from)
	}

	// Map order is random, so the earliest stray date is found and named,
	// which keeps the error the same from one run to the next.
	var stray *civil.Date

	for date := range exceptions {
		if !date.Before(from) && !date.After(to) {
			continue
		}

		if stray == nil || date.Before(*stray) {
			stray = &date
		}
	}

	if stray != nil {
		return fmt.Errorf("exception on %s is outside %s to %s", *stray, from, to)
	}

	return nil
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

// versionOn returns the schedule version in force on date: the last one that
// applies from date or earlier. One must exist.
func versionOn(schedules []WeeklySchedule, date civil.Date) WeeklySchedule {
	after := sort.Search(len(schedules), func(i int) bool { return schedules[i].from.After(date) })

	return schedules[after-1]
}

// validateWeekday checks the hours of one weekday.
func validateWeekday(hours []Hours) error {
	if len(hours) == 0 {
		return nil
	}

	hasRegular := false

	for i, h := range hours {
		if h.kind == "" {
			return fmt.Errorf("hours %d are not hours", i)
		}

		if i > 0 && h.startOffset() < hours[i-1].endOffset() {
			return fmt.Errorf("hours %s-%s start before hours %s-%s end", h.start, h.end, hours[i-1].start, hours[i-1].end)
		}

		hasRegular = hasRegular || h.kind == SessionKindRegular
	}

	if !hasRegular {
		return errors.New("no regular session")
	}

	return nil
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

// instantAt returns the instant at which a clock in loc shows t on date.
// Midnight is the first instant of the date, which every date has. Any
// other time must be shown exactly once on that date.
func instantAt(date civil.Date, t civil.Time, loc *time.Location) (time.Time, error) {
	if t == (civil.Time{}) {
		return startOfDate(date, loc), nil
	}

	instant := date.At(t, loc)

	// time.Date moves a time the zone skips to a nearby one that exists, so
	// the clock then shows something other than what was asked for.
	if civil.NewDateFromTime(instant) != date || civil.NewTimeFromTime(instant) != t {
		return time.Time{}, fmt.Errorf("%s is skipped on %s in %s", t, date, loc)
	}

	if isRepeated(instant) {
		return time.Time{}, fmt.Errorf("%s is repeated on %s in %s", t, date, loc)
	}

	return instant, nil
}

// startOfDate returns the first instant whose date in loc is date. It
// searches for it, because a zone that changes its clock at midnight has no
// 00:00 on that date, or has two.
func startOfDate(date civil.Date, loc *time.Location) time.Time {
	noon := time.Date(date.Year(), date.Month(), date.Day(), hoursPerDay/2, 0, 0, 0, loc)
	earliest := noon.Add(-startSearchSpan)

	seconds := sort.Search(int(startSearchSpan/time.Second), func(i int) bool {
		at := earliest.Add(time.Duration(i) * time.Second)

		return !civil.NewDateFromTime(at).Before(date)
	})

	return earliest.Add(time.Duration(seconds) * time.Second)
}

// isRepeated reports whether a clock in t's location shows t's wall time a
// second time, before or after t, because the zone sets its clock back.
func isRepeated(t time.Time) bool {
	start, end := t.ZoneBounds()
	_, offset := t.Zone()

	// The same wall time under a neighbouring offset is another instant. The
	// time is repeated when that instant lies inside the neighbour's span.
	if !start.IsZero() {
		_, before := start.Add(-time.Nanosecond).Zone()
		if t.Add(time.Duration(offset-before) * time.Second).Before(start) {
			return true
		}
	}

	if !end.IsZero() {
		_, after := end.Zone()
		if !t.Add(time.Duration(offset-after) * time.Second).Before(end) {
			return true
		}
	}

	return false
}

// sinceMidnight returns how far into the day t is.
func sinceMidnight(t civil.Time) time.Duration {
	return time.Duration(t.Hour())*time.Hour +
		time.Duration(t.Minute())*time.Minute +
		time.Duration(t.Second())*time.Second +
		time.Duration(t.Nanosecond())
}
