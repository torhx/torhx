package calendar

import "fmt"

// A Registry holds trading calendars by their ID, so a caller can look up the
// calendar an instrument names.
//
// A Registry comes only from a constructor and never changes, so it is safe
// for concurrent use by multiple goroutines. The zero value holds no calendar.
type Registry struct {
	// byID holds each calendar under its ID.
	byID map[ID]TradingCalendar
}

// NewRegistry returns the registry of the given calendars.
//
// It returns an error if a calendar is the zero calendar, or if two calendars
// have the same ID.
func NewRegistry(calendars []TradingCalendar) (Registry, error) {
	byID := make(map[ID]TradingCalendar, len(calendars))

	for i, c := range calendars {
		if !c.IsValid() {
			return Registry{}, fmt.Errorf("invalid calendar registry: calendar %d is not a calendar", i)
		}

		if _, ok := byID[c.ID()]; ok {
			return Registry{}, fmt.Errorf("invalid calendar registry: calendar %s is given twice", c.ID())
		}

		byID[c.ID()] = c
	}

	return Registry{byID: byID}, nil
}

// Get returns the calendar named id.
//
// It returns an error if the registry holds no calendar named id.
func (r Registry) Get(id ID) (TradingCalendar, error) {
	c, ok := r.byID[id]
	if !ok {
		return TradingCalendar{}, fmt.Errorf("no trading calendar %s", id)
	}

	return c, nil
}
