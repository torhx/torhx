package marketevent

import (
	"errors"
	"fmt"
	"iter"
	"slices"
	"sort"

	"github.com/talav/civil"
)

// Events is a collection of events, of every type, and the dates it covers.
// Narrowing it, by subject, by date or by kind, gives the Events of the
// events that match, which narrows and reads in the same way.
//
// Within its coverage, a date without an event has none; outside it, a read
// finds no event, which does not mean that none happens, so a program checks
// Coverage where it builds its Events.
//
// Events holds a schedule as its events state it, and not when each event
// became known: it cannot say what a program knew on an earlier date.
//
// Events borrows its storage: it keeps the slice NewEvents is given and never
// changes it afterwards, so concurrent reads are safe while no caller changes
// the slice or an event. The zero value holds no events and covers no date.
type Events struct {
	// byDate holds the events in order of date, then subject.
	byDate []Event

	// bySubject holds the events in order of subject, then date, so that the
	// events about one subject are a run in date order. After narrowing by
	// date it also holds the events that narrowing left out, which are all
	// dated outside coverage.
	bySubject []Event

	// coverage is the half-open range of dates the collection covers.
	// A zero range means the collection covers no dates.
	coverage civil.DateRange
}

// NewEvents builds a collection for the given half-open coverage range.
// It sorts events in place and keeps the slice. Events with the same date
// and subject keep their input order, including duplicates. The caller
// must leave the slice unchanged after construction.
//
// It returns an error if coverage is the zero range, or if an event is a nil
// interface, has the zero timing or subject or an empty kind, or is dated
// outside coverage. A typed nil pointer is not supported, and its methods
// may panic.
func NewEvents(events []Event, coverage civil.DateRange) (Events, error) {
	es := Events{byDate: events, coverage: coverage}
	if err := es.validate(); err != nil {
		return Events{}, fmt.Errorf("invalid events: %w", err)
	}

	slices.SortStableFunc(es.byDate, compareByDate)
	es.bySubject = sortedBySubject(es.byDate)

	return es, nil
}

// Coverage returns the dates the events cover, including the start and
// excluding the end. It returns the zero range when no dates are covered.
func (es Events) Coverage() civil.DateRange { return es.coverage }

// For returns the events about subject. Narrowing twice keeps the events
// about both subjects, so For(us).For(ca) holds none, and so does For with
// the zero Subject.
func (es Events) For(subject Subject) Events {
	start := sort.Search(len(es.bySubject), func(i int) bool {
		return es.bySubject[i].Subject().compare(subject) >= 0
	})
	end := sort.Search(len(es.bySubject), func(i int) bool {
		return es.bySubject[i].Subject().compare(subject) > 0
	})

	// The events about one subject are in date order, which is both orders
	// at once.
	about := dated(es.bySubject[start:end], es.coverage)

	return Events{byDate: about, bySubject: about, coverage: es.coverage}
}

// On returns the events of date. The result covers that date alone, or no
// date when this Events does not cover it.
func (es Events) On(date civil.Date) Events {
	if !date.IsValid() || !es.coverage.IsValid() || !es.coverage.ContainsDate(date) {
		return Events{}
	}

	day, err := civil.NewDateRange(date, date.AddDays(1))
	if err != nil {
		return Events{}
	}

	return es.withCoverage(day)
}

// During returns the events in the intersection of dates and the collection's
// coverage. A missing range or a disjoint range yields an empty collection
// with zero coverage.
func (es Events) During(dates civil.DateRange) Events {
	if !es.coverage.IsValid() || !dates.IsValid() {
		return Events{}
	}

	if dates.Contains(es.coverage) {
		return es
	}

	coverage, ok := es.coverage.Intersect(dates)
	if !ok {
		return Events{}
	}

	return es.withCoverage(coverage)
}

// OfKind returns the events of any of kinds, or every event when kinds is
// empty. It makes a new list of them, so Has is the cheaper way to ask only
// whether there is one.
func (es Events) OfKind(kinds ...string) Events {
	if len(kinds) == 0 {
		return es
	}

	var ofKind []Event

	for _, e := range es.byDate {
		if slices.Contains(kinds, e.Kind()) {
			ofKind = append(ofKind, e)
		}
	}

	return Events{byDate: ofKind, bySubject: sortedBySubject(ofKind), coverage: es.coverage}
}

// Has reports whether there is an event of any of kinds, or any event at all
// when kinds is empty: whether OfKind would hold an event.
func (es Events) Has(kinds ...string) bool {
	for _, e := range es.byDate {
		if len(kinds) == 0 || slices.Contains(kinds, e.Kind()) {
			return true
		}
	}

	return false
}

// Len returns the number of events.
func (es Events) Len() int { return len(es.byDate) }

// All returns the events in order: by date, then by subject, then as
// NewEvents was given them.
func (es Events) All() iter.Seq[Event] { return slices.Values(es.byDate) }

// Of returns the events of type T, as T values, in the order of All. For a
// concrete T, it keeps the events whose dynamic type is exactly T, so
// Of[Auction] and Of[*Auction] differ, and Of[EconomicEvent] leaves out a
// program's types that embed EconomicEvent. For an interface T, it keeps
// every event that implements T.
func (es Events) Of[T Event]() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, e := range es.byDate {
			if typed, ok := e.(T); ok && !yield(typed) {
				return
			}
		}
	}
}

func (es Events) validate() error {
	if !es.coverage.IsValid() {
		return errors.New("coverage is missing")
	}

	for i, e := range es.byDate {
		if e == nil {
			return fmt.Errorf("event %d is missing", i)
		}

		timing := e.Timing()
		if !timing.IsValid() {
			return fmt.Errorf("event %d has no timing", i)
		}

		if !e.Subject().IsValid() {
			return fmt.Errorf("event %d has no subject", i)
		}

		if e.Kind() == "" {
			return fmt.Errorf("event %d has no kind", i)
		}

		if date := timing.Date(); !es.coverage.ContainsDate(date) {
			return fmt.Errorf("event %d is dated %s, outside %s", i, date, es.coverage)
		}
	}

	return nil
}

// withCoverage narrows the date index to a range within the current coverage.
// For applies the same bounds when it reads the shared subject index.
func (es Events) withCoverage(coverage civil.DateRange) Events {
	return Events{byDate: dated(es.byDate, coverage), bySubject: es.bySubject, coverage: coverage}
}

// sortedBySubject returns a copy of byDate, which is in order of date, in
// order of subject, then date: a stable sort by subject alone leaves each
// subject's events in the date order they already have.
func sortedBySubject(byDate []Event) []Event {
	bySubject := slices.Clone(byDate)
	slices.SortStableFunc(bySubject, compareBySubject)

	return bySubject
}

// dated returns the slice of date-ordered events in dates. Both bounds are
// the first event on or after their respective endpoint.
func dated(events []Event, dates civil.DateRange) []Event {
	lowerBound := func(date civil.Date) int {
		return sort.Search(len(events), func(i int) bool {
			return !events[i].Timing().Date().Before(date)
		})
	}

	return events[lowerBound(dates.Start()):lowerBound(dates.End())]
}

// compareByDate orders events by date, then by subject.
func compareByDate(a, b Event) int {
	if c := a.Timing().Date().Compare(b.Timing().Date()); c != 0 {
		return c
	}

	return a.Subject().compare(b.Subject())
}

// compareBySubject orders events by subject.
func compareBySubject(a, b Event) int {
	return a.Subject().compare(b.Subject())
}
