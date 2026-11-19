package calendar

import (
	"errors"
	"fmt"
	"time"

	"github.com/talav/civil"
)

// dayKind says whether a day trades and, when it does not, why. The zero
// value is no kind.
type dayKind string

const (
	businessDay dayKind = "business_day"
	holiday     dayKind = "holiday"
	weekend     dayKind = "weekend"
)

// A TradingDay is one date of a trading calendar: the date the market books
// trading to, and the sessions booked to it. It is exactly one of three
// things: a business day, on which the market trades; a weekend, which falls
// on a day of the week the market never trades on; or a holiday, a closed
// date on a day of the week it trades on.
//
// A session of a business day may open before the date or close after it, so
// a futures session that opens on Sunday evening belongs to Monday's day.
//
// A TradingDay comes only from a constructor. A business day has at least
// one regular session, and its sessions are in time order without overlap; a
// weekend or a holiday has no session. The zero value is not a day.
type TradingDay struct {
	// date is the date the market books the day's trading to.
	date civil.Date

	// kind says whether the day is a business day, a holiday or a weekend.
	kind dayKind

	// sessions are the sessions of a business day in time order, without
	// overlap, at least one of them regular. A holiday or a weekend has
	// none.
	sessions []Session

	// lateOpen says that regular trading started after the usual time. It is
	// a fact given to a business day, which cannot check it against its
	// sessions. It is false on a holiday or a weekend.
	lateOpen bool

	// earlyClose says that regular trading ended before the usual time. It
	// is a fact given to a business day, which cannot check it against its
	// sessions. It is false on a holiday or a weekend.
	earlyClose bool
}

// NewTradingDay returns the business day dated date with the given
// sessions. lateOpen and earlyClose say whether regular trading started
// after, or ended before, the usual time.
//
// It returns an error if the date is the zero date, if a session is the zero
// session, if the sessions are out of time order or overlap, or if none is
// regular. It puts no limit on where the sessions lie relative to the date:
// trading over a holiday can be booked to a later date.
//
// The day keeps sessions without a copy, so the caller must not modify the
// slice afterwards.
func NewTradingDay(date civil.Date, sessions []Session, lateOpen, earlyClose bool) (TradingDay, error) {
	d := TradingDay{date: date, kind: businessDay, sessions: sessions, lateOpen: lateOpen, earlyClose: earlyClose}
	if err := d.validate(); err != nil {
		return TradingDay{}, fmt.Errorf("invalid trading day %s: %w", date, err)
	}

	return d, nil
}

// NewHoliday returns the day dated date as a holiday: a closed date on a day
// of the week the market trades on.
//
// It returns an error if the date is the zero date.
func NewHoliday(date civil.Date) (TradingDay, error) {
	d := TradingDay{date: date, kind: holiday}
	if err := d.validate(); err != nil {
		return TradingDay{}, fmt.Errorf("invalid holiday %s: %w", date, err)
	}

	return d, nil
}

// NewWeekend returns the day dated date as a weekend: a date on a day of the
// week the market never trades on. Which days those are is the market's own
// matter: Saturday and Sunday for most, none for a market that trades every
// date.
//
// It returns an error if the date is the zero date.
func NewWeekend(date civil.Date) (TradingDay, error) {
	d := TradingDay{date: date, kind: weekend}
	if err := d.validate(); err != nil {
		return TradingDay{}, fmt.Errorf("invalid weekend %s: %w", date, err)
	}

	return d, nil
}

// Date returns the date the market books the day's trading to.
func (d TradingDay) Date() civil.Date { return d.date }

// IsBusinessDay reports whether the market trades on the day.
func (d TradingDay) IsBusinessDay() bool { return d.kind == businessDay }

// IsHoliday reports whether the day is a closed date on a day of the week the
// market trades on.
func (d TradingDay) IsHoliday() bool { return d.kind == holiday }

// IsWeekend reports whether the day falls on a day of the week the market
// never trades on.
func (d TradingDay) IsWeekend() bool { return d.kind == weekend }

// Sessions returns the sessions of the day in time order. There are none on
// a holiday or a weekend. The slice is the day's own, and the caller must not
// modify it.
func (d TradingDay) Sessions() []Session { return d.sessions }

// Open returns the instant regular trading starts: the open of the first
// regular session. It can fall on an earlier civil date than the day's. It
// reports false on a holiday or a weekend.
func (d TradingDay) Open() (time.Time, bool) { return firstRegularOpen(d.sessions) }

// Close returns the instant regular trading ends: the close of the last
// regular session. It reports false on a holiday or a weekend.
func (d TradingDay) Close() (time.Time, bool) { return lastRegularClose(d.sessions) }

// Duration returns how long regular trading lasts, breaks left out. With a
// break it is shorter than the time from Open to Close. It is zero on a
// holiday or a weekend.
func (d TradingDay) Duration() time.Duration {
	var total time.Duration

	for _, s := range d.sessions {
		if s.kind == SessionKindRegular {
			total += s.Duration()
		}
	}

	return total
}

// IsLateOpen reports whether regular trading started after the usual time.
func (d TradingDay) IsLateOpen() bool { return d.lateOpen }

// IsEarlyClose reports whether regular trading ended before the usual time.
func (d TradingDay) IsEarlyClose() bool { return d.earlyClose }

// IsFullDay reports whether the day is a business day that is neither a
// late open nor an early close.
func (d TradingDay) IsFullDay() bool { return d.kind == businessDay && !d.lateOpen && !d.earlyClose }

// IsValid reports whether d is a day. It is false only for the zero value,
// which stands for a missing one; every TradingDay from a constructor is
// valid.
func (d TradingDay) IsValid() bool { return d.kind != "" }

func (d TradingDay) validate() error {
	if !d.date.IsValid() {
		return errors.New("date is missing")
	}

	if d.kind != businessDay {
		return nil
	}

	for i, s := range d.sessions {
		if !s.IsValid() {
			return fmt.Errorf("session %d is not a session", i)
		}

		if i > 0 && s.open.Before(d.sessions[i-1].close) {
			return fmt.Errorf("session %d opens at %s, before session %d closes", i, s.open, i-1)
		}
	}

	if _, ok := firstRegularOpen(d.sessions); !ok {
		return errors.New("no regular session")
	}

	return nil
}

// firstOpen returns the open of a business day's first session of any kind.
func (d TradingDay) firstOpen() time.Time { return d.sessions[0].open }

// lastClose returns the close of a business day's last session of any kind.
func (d TradingDay) lastClose() time.Time { return d.sessions[len(d.sessions)-1].close }

// in returns the day with the instants of its sessions shown in loc.
func (d TradingDay) in(loc *time.Location) TradingDay {
	// A holiday or a weekend has nothing to show and keeps its nil slice.
	if len(d.sessions) == 0 {
		return d
	}

	shown := d
	shown.sessions = make([]Session, len(d.sessions))

	for i, s := range d.sessions {
		shown.sessions[i] = s.in(loc)
	}

	return shown
}
