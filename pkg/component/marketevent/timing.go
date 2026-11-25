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
	// locationDate is the event's date in its zone. For an exact timing it
	// is the date that instant falls on there.
	locationDate civil.LocationDate

	// instant is the instant of an exact timing, shown in the zone of
	// locationDate. It is nil for a date.
	instant *time.Time

	// flag is the flag of a date. It is a fact given to the date, which a
	// timing cannot check against market hours. It is nil for an exact
	// timing and for a date without a flag.
	flag *Flag
}

// NewExactTiming returns the instant t dated in its own named time zone.
// It returns an error for the zero instant, [time.Local], an unnamed or
// unknown zone, or an offset that disagrees with the named zone at t.
func NewExactTiming(t time.Time) (Timing, error) {
	// time.Local can report a different IANA name on each machine.
	if t.Location() == time.Local {
		return Timing{}, errors.New("invalid exact timing: instant uses the machine's local time zone")
	}

	day, err := civil.NewLocationDate(civil.NewDateFromTime(t), t.Location().String())
	if err != nil {
		return Timing{}, fmt.Errorf("invalid exact timing %s: %w", t, err)
	}

	// A fixed zone can borrow an IANA name while using a different offset.
	instant := t.In(day.Location())
	_, sourceOffsetSeconds := t.Zone()
	_, zoneOffsetSeconds := instant.Zone()
	if sourceOffsetSeconds != zoneOffsetSeconds {
		return Timing{}, fmt.Errorf(
			"invalid exact timing %s: UTC offset %d seconds disagrees with %s (%d seconds)",
			t, sourceOffsetSeconds, day.Location(), zoneOffsetSeconds,
		)
	}

	timing := Timing{locationDate: day, instant: &instant}
	if err := timing.validate(); err != nil {
		return Timing{}, fmt.Errorf("invalid exact timing %s: %w", t, err)
	}

	return timing, nil
}

// NewDateTiming returns the timing of day, a date in a zone, without a flag.
//
// It returns an error if day is the zero value.
func NewDateTiming(day civil.LocationDate) (Timing, error) {
	timing := Timing{locationDate: day}
	if err := timing.validate(); err != nil {
		return Timing{}, fmt.Errorf("invalid date timing: %w", err)
	}

	return timing, nil
}

// NewBeforeMarketTiming returns the timing of day, a date in a zone, before
// the market opens.
//
// It returns an error if day is the zero value.
func NewBeforeMarketTiming(day civil.LocationDate) (Timing, error) {
	timing := Timing{locationDate: day, flag: new(FlagBeforeMarket)}
	if err := timing.validate(); err != nil {
		return Timing{}, fmt.Errorf("invalid before-market timing: %w", err)
	}

	return timing, nil
}

// NewAfterMarketTiming returns the timing of day, a date in a zone, after
// the market closes.
//
// It returns an error if day is the zero value.
func NewAfterMarketTiming(day civil.LocationDate) (Timing, error) {
	timing := Timing{locationDate: day, flag: new(FlagAfterMarket)}
	if err := timing.validate(); err != nil {
		return Timing{}, fmt.Errorf("invalid after-market timing: %w", err)
	}

	return timing, nil
}

// LocationDate returns the event's date together with its zone.
func (t Timing) LocationDate() civil.LocationDate { return t.locationDate }

// Date returns the event's date in its zone.
func (t Timing) Date() civil.Date { return t.locationDate.Date() }

// Location returns the time zone of the timing.
func (t Timing) Location() *time.Location { return t.locationDate.Location() }

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
func (t Timing) IsValid() bool { return t.locationDate.IsValid() }

func (t Timing) validate() error {
	if !t.locationDate.IsValid() {
		return errors.New("date is missing")
	}

	if t.instant != nil && t.instant.IsZero() {
		return errors.New("instant is missing")
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

	// FlagAfterMarket says that the event comes after the regular close on
	// its local date. An extended session may still be trading.
	FlagAfterMarket Flag = "AfterMarket"
)
