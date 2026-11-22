package marketevent

import (
	"errors"
	"fmt"
	"time"

	"github.com/talav/civil"
)

// Timing is when an event happens: an exact date and time, or a date alone,
// which may carry a flag for before the market opens or after it closes.
// Either way it has a time zone, and the event's date is its date in that
// zone.
//
// Timing comes only from a constructor. The zero value is not a timing.
type Timing struct {
	// date is the event's date in loc. For an exact timing it is the date
	// that instant falls on there.
	date civil.Date

	// loc is the time zone the timing is dated in. It is neither nil nor
	// time.Local.
	loc *time.Location

	// instant is the instant of an exact timing, shown in loc. It is nil for
	// a date.
	instant *time.Time

	// flag is the flag of a date. It is a fact given to the date, which a
	// timing cannot check against market hours. It is nil for an exact
	// timing and for a date without a flag.
	flag *Flag
}

// NewExactTiming returns the timing of the instant t, dated in loc.
//
// It returns an error if t is the zero time, or if loc is nil or time.Local,
// whose zone is the machine's and not one a source specifies.
func NewExactTiming(t time.Time, loc *time.Location) (Timing, error) {
	timing := Timing{loc: loc, instant: &t}
	if err := timing.validate(); err != nil {
		return Timing{}, fmt.Errorf("invalid exact timing %s: %w", t, err)
	}

	// The instant is shown and dated in loc only now, because time.Time.In
	// panics on a nil location.
	*timing.instant = t.In(loc)
	timing.date = civil.NewDateFromTime(*timing.instant)

	return timing, nil
}

// NewDateTiming returns the timing of the date d in loc, without a flag.
//
// It returns an error if d is the zero date, or if loc is nil or time.Local.
func NewDateTiming(d civil.Date, loc *time.Location) (Timing, error) {
	timing := Timing{date: d, loc: loc}
	if err := timing.validate(); err != nil {
		return Timing{}, fmt.Errorf("invalid date timing %s: %w", d, err)
	}

	return timing, nil
}

// NewBeforeMarketTiming returns the timing of the date d in loc, before the
// market opens.
//
// It returns an error as NewDateTiming does.
func NewBeforeMarketTiming(d civil.Date, loc *time.Location) (Timing, error) {
	timing := Timing{date: d, loc: loc, flag: new(FlagBeforeMarket)}
	if err := timing.validate(); err != nil {
		return Timing{}, fmt.Errorf("invalid before-market timing %s: %w", d, err)
	}

	return timing, nil
}

// NewAfterMarketTiming returns the timing of the date d in loc, after the
// market closes.
//
// It returns an error as NewDateTiming does.
func NewAfterMarketTiming(d civil.Date, loc *time.Location) (Timing, error) {
	timing := Timing{date: d, loc: loc, flag: new(FlagAfterMarket)}
	if err := timing.validate(); err != nil {
		return Timing{}, fmt.Errorf("invalid after-market timing %s: %w", d, err)
	}

	return timing, nil
}

// Date returns the event's date in its zone.
func (t Timing) Date() civil.Date { return t.date }

// Location returns the time zone of the timing.
func (t Timing) Location() *time.Location { return t.loc }

// Instant returns the instant of an exact timing, shown in its zone. It
// reports false for a date.
func (t Timing) Instant() (time.Time, bool) {
	if t.instant == nil {
		return time.Time{}, false
	}

	return *t.instant, true
}

// Flag returns the flag of a date: before the market opens or after it
// closes. It reports false for an exact timing, and for a date without a
// flag, whose source says nothing about the time of day.
func (t Timing) Flag() (Flag, bool) {
	if t.flag == nil {
		return "", false
	}

	return *t.flag, true
}

// IsValid reports whether t is a timing. It is false only for the zero
// value.
func (t Timing) IsValid() bool { return t.date.IsValid() }

func (t Timing) validate() error {
	if t.instant != nil && t.instant.IsZero() {
		return errors.New("instant is missing")
	}

	if t.instant == nil && !t.date.IsValid() {
		return errors.New("date is missing")
	}

	if t.loc == nil {
		return errors.New("time zone is missing")
	}

	if t.loc == time.Local {
		return errors.New("time zone is the machine's local zone, not one a source specifies")
	}

	return nil
}

// A Flag says when on its date an event comes, for a date whose source gives
// no time: before the market opens or after it closes. It is the source's
// statement about the regular session, on that date, of the market the event
// belongs to: for a company event, the market its stock trades on.
//
// A timing keeps a flag and never works one out, so an exact timing has
// none, however early or late its instant: whether an instant comes before
// an open depends on the market that asks, whose hours a timing does not
// know.
//
// The zero value is not a flag.
type Flag string

const (
	// FlagBeforeMarket says that the event comes before the market opens.
	FlagBeforeMarket Flag = "BeforeMarket"

	// FlagAfterMarket says that the event comes after the market closes. The
	// event keeps its date, though trading on it starts on the next trading
	// day.
	FlagAfterMarket Flag = "AfterMarket"
)
