package calendar

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/talav/civil"
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
