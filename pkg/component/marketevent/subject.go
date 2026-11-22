package marketevent

import (
	"errors"
	"fmt"

	"github.com/torhx/identifier"
)

// A Subject is what an event is about: a region, or a stock by its
// composite FIGI. An economic event's subject is its region, and a company
// event's is its stock.
//
// A Subject comes only from a constructor. Subjects are comparable with ==
// and usable as map keys. The zero value is not a subject: no event has it,
// so a read narrowed to it finds nothing.
type Subject struct {
	// region is the region of a region's subject. It is the zero region for
	// a stock's subject.
	region Region

	// stock is the composite FIGI of a stock's subject. It is the zero FIGI
	// for a region's subject.
	stock identifier.FIGI
}

// NewSubjectFromRegion returns the subject of the region r.
//
// It returns an error if r is the zero region.
func NewSubjectFromRegion(r Region) (Subject, error) {
	s := Subject{region: r}
	if err := s.validate(); err != nil {
		return Subject{}, fmt.Errorf("invalid subject: %w", err)
	}

	return s, nil
}

// NewSubjectFromStock returns the subject of the stock s, by its composite
// FIGI.
//
// It returns an error if s is the zero FIGI.
func NewSubjectFromStock(s identifier.FIGI) (Subject, error) {
	subject := Subject{stock: s}
	if err := subject.validate(); err != nil {
		return Subject{}, fmt.Errorf("invalid subject: %w", err)
	}

	return subject, nil
}

// Region returns the region of a region's subject. It reports false for a
// stock's subject and for the zero value.
func (s Subject) Region() (Region, bool) { return s.region, s.region.IsValid() }

// Stock returns the composite FIGI of a stock's subject. It reports false
// for a region's subject and for the zero value.
func (s Subject) Stock() (identifier.FIGI, bool) { return s.stock, s.stock.IsValid() }

// IsValid reports whether s is a subject. It is false only for the zero
// value.
func (s Subject) IsValid() bool { return s != Subject{} }

// String returns the region's code, such as "US" or "EUR", or the stock's
// FIGI.
func (s Subject) String() string {
	if s.region.IsValid() {
		return s.region.String()
	}

	return s.stock.String()
}

func (s Subject) validate() error {
	if !s.region.IsValid() && !s.stock.IsValid() {
		return errors.New("no region and no stock")
	}

	return nil
}
