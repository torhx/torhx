package marketevent

import (
	"fmt"
	"iter"
	"slices"
	"sort"

	"github.com/talav/civil"
)

// A Reader reads events: it narrows to the events about a subject, and
// reads them by date. Code that reads events takes a Reader, so that its
// tests can give it a Reader of their own.
type Reader interface {
	// For returns a new Reader of the events about subject, and leaves this
	// one as it is. Narrowing a narrowed Reader keeps only the events about
	// both subjects, so For(us).For(ca) reads nothing, and so does For with
	// the zero Subject.
	For(subject Subject) Reader

	// Has reports whether date has an event of any of kinds, or of any kind
	// when kinds is empty: whether Find would return an event.
	Has(date civil.Date, kinds ...string) bool

	// Find returns the events of date of any of kinds, or of every kind
	// when kinds is empty, in order.
	Find(date civil.Date, kinds ...string) Selection

	// Between returns the events dated from from to to, both included, by
	// date. It returns none when from is after to.
	Between(from, to civil.Date) Selection
}

// Events is the Reader of a program's events, of every type. Within its
// coverage, a date without an event has none; outside it, a read finds no
// event, which does not mean that none happens, so a program checks
// Coverage before it reads.
//
// Events holds a schedule as its events state it, and not when each event
// became known: it cannot say what a program knew on an earlier date.
//
// Events borrows its storage: it keeps the slice NewEvents is given, and the
// Selections its reads return share that memory. It never changes either
// after NewEvents, so concurrent reads are safe while no caller changes the
// slice, a Selection or an event. The zero value holds no events and covers
// no date.
type Events struct {
	// byDate holds the events in order of date, then subject. It is the
	// slice NewEvents is given, or a part of bySubject after narrowing.
	byDate []Event

	// bySubject holds the same events in order of subject, then date, so
	// that the events about one subject are a run in date order.
	bySubject []Event

	// from is the first date the events cover. It is the zero date only in
	// the zero value.
	from civil.Date

	// to is the last date the events cover, never before from. It is the
	// zero date only in the zero value.
	to civil.Date
}

// NewEvents returns the Events of events, which cover the dates from from
// to to. It sorts events in place and keeps the slice, so the caller must
// not change it afterwards. Events of the same date and subject keep the
// order they are given in, and an event given twice is kept twice.
//
// It returns an error if from or to is the zero date, if from is after to,
// or if an event is a nil interface, has the zero timing or subject or an
// empty kind, or is dated outside from to to. A typed nil pointer is not
// supported, and its methods may panic.
func NewEvents(events []Event, from, to civil.Date) (Events, error) {
	es := Events{byDate: events, from: from, to: to}
	if err := es.validate(); err != nil {
		return Events{}, fmt.Errorf("invalid events: %w", err)
	}

	slices.SortStableFunc(es.byDate, compareByDate)

	// A stable sort by subject alone leaves each subject's events in the
	// date order they already have.
	es.bySubject = slices.Clone(es.byDate)
	slices.SortStableFunc(es.bySubject, compareBySubject)

	return es, nil
}

// Coverage returns the first and the last date the events cover. The zero
// Events returns two zero dates.
func (es Events) Coverage() (from, to civil.Date) { return es.from, es.to }

// For returns a new Reader of the events about subject.
func (es Events) For(subject Subject) Reader {
	start := sort.Search(len(es.bySubject), func(i int) bool {
		return es.bySubject[i].Subject().compare(subject) >= 0
	})
	end := sort.Search(len(es.bySubject), func(i int) bool {
		return es.bySubject[i].Subject().compare(subject) > 0
	})

	// The events about one subject are in date order, which is both orders
	// at once.
	about := es.bySubject[start:end:end]

	return Events{byDate: about, bySubject: about, from: es.from, to: es.to}
}

// Has reports whether date has an event of any of kinds, or of any kind
// when kinds is empty.
func (es Events) Has(date civil.Date, kinds ...string) bool {
	for _, e := range es.between(date, date) {
		if len(kinds) == 0 || slices.Contains(kinds, e.Kind()) {
			return true
		}
	}

	return false
}

// Find returns the events of date of any of kinds, or of every kind when
// kinds is empty, in order.
func (es Events) Find(date civil.Date, kinds ...string) Selection {
	on := es.between(date, date)
	if len(kinds) == 0 {
		return on
	}

	var found Selection

	for _, e := range on {
		if slices.Contains(kinds, e.Kind()) {
			found = append(found, e)
		}
	}

	return found
}

// Between returns the events dated from from to to, both included, by date.
// It returns none when from is after to.
func (es Events) Between(from, to civil.Date) Selection { return es.between(from, to) }

// between returns the events dated from from to to, both included, by date.
// The result shares the memory of byDate and has no room to grow into it.
func (es Events) between(from, to civil.Date) Selection {
	if from.After(to) {
		return nil
	}

	start := sort.Search(len(es.byDate), func(i int) bool {
		return !es.byDate[i].Timing().Date().Before(from)
	})
	end := sort.Search(len(es.byDate), func(i int) bool {
		return es.byDate[i].Timing().Date().After(to)
	})

	return es.byDate[start:end:end]
}

//nolint:cyclop // one flat list of checks reads better than a split
func (es Events) validate() error {
	if !es.from.IsValid() || !es.to.IsValid() || es.from.After(es.to) {
		return fmt.Errorf("dates %s to %s are not a range", es.from, es.to)
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

		if date := timing.Date(); date.Before(es.from) || date.After(es.to) {
			return fmt.Errorf("event %d is dated %s, outside %s to %s", i, date, es.from, es.to)
		}
	}

	return nil
}

// A Selection is the events a read gives, of every type, in order. It may
// share the memory of the Events that gave it, so it must not be changed.
type Selection []Event

// Of returns the events of type T, as T values. For a concrete T, it keeps
// the events whose dynamic type is exactly T, so Of[Auction] and
// Of[*Auction] differ, and Of[EconomicEvent] leaves out a program's types
// that embed EconomicEvent. For an interface T, it keeps every event that
// implements T.
func (s Selection) Of[T Event]() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, e := range s {
			if typed, ok := e.(T); ok && !yield(typed) {
				return
			}
		}
	}
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
