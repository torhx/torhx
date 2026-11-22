package marketevent

import (
	"errors"
	"fmt"

	"github.com/torhx/identifier"
)

// An Event is an event of any type: EconomicEvent, Earnings, Dividend,
// Split, or a program's own type. A program's type has the three methods
// itself, or embeds EconomicEvent or CompanyEvent, which have them, and adds
// fields. Since a program's type can return a zero timing or subject or an
// empty kind, NewEvents checks every event it is given. An event's Timing,
// Kind and Subject must return the same values on every call and be safe to
// call concurrently, as the built-in types' are.
type Event interface {
	// Timing returns when the event happens.
	Timing() Timing

	// Kind returns the kind of event, such as "CPI" or "earnings".
	Kind() string

	// Subject returns what the event is about: a region or a stock.
	Subject() Subject
}

// An EconomicEvent is a scheduled economic event, such as a release of the
// US CPI: its region, its timing, and its kind, such as "CPI", with a detail
// that narrows it. Kinds and details are Torhx's own names, onto which each
// source's names are mapped, so an event reads the same whatever its source.
// A program's own type of economic event can embed it and add fields.
//
// An EconomicEvent comes only from NewEconomicEvent. The zero value is not
// an event.
type EconomicEvent struct {
	// region is the region the event is about.
	region Region

	// timing is when the event happens.
	timing Timing

	// kind is the kind of event, such as "CPI". It is not empty.
	kind string

	// detail narrows the kind, such as "advance" for a GDP estimate. It is
	// empty when nothing does.
	detail string
}

// NewEconomicEvent returns the event about region at timing, of kind,
// narrowed by detail, such as "advance" for a GDP estimate. An empty detail
// narrows nothing.
//
// It returns an error if region or timing is the zero value, or if kind is
// empty.
func NewEconomicEvent(region Region, timing Timing, kind, detail string) (EconomicEvent, error) {
	e := EconomicEvent{region: region, timing: timing, kind: kind, detail: detail}
	if err := e.validate(); err != nil {
		return EconomicEvent{}, fmt.Errorf("invalid economic event: %w", err)
	}

	return e, nil
}

// Region returns the region the event is about.
func (e EconomicEvent) Region() Region { return e.region }

// Subject returns the event's region, as its subject.
func (e EconomicEvent) Subject() Subject { return Subject{region: e.region} }

// Timing returns when the event happens.
func (e EconomicEvent) Timing() Timing { return e.timing }

// Kind returns the kind of event, such as "CPI".
func (e EconomicEvent) Kind() string { return e.kind }

// Detail returns what narrows the kind, such as "advance" for a GDP
// estimate, or "" when nothing does.
func (e EconomicEvent) Detail() string { return e.detail }

func (e EconomicEvent) validate() error {
	if !e.region.IsValid() {
		return errors.New("region is missing")
	}

	if !e.timing.IsValid() {
		return errors.New("timing is missing")
	}

	if e.kind == "" {
		return errors.New("kind is missing")
	}

	return nil
}

// A CompanyEvent is the common part of a program's own type of company
// event: the stock it is about, by its composite FIGI, its timing and its
// kind, such as "buyback". The program's type embeds it and adds fields; the
// built-in types have a stock, a timing and a kind of their own.
//
// A CompanyEvent comes only from NewCompanyEvent. The zero value is not an
// event.
type CompanyEvent struct {
	// stock is the composite FIGI of the stock the event is about.
	stock identifier.FIGI

	// timing is when the event happens.
	timing Timing

	// kind is the kind of event, such as "buyback". It is not empty.
	kind string
}

// NewCompanyEvent returns the event about stock, by its composite FIGI, at
// timing, of kind.
//
// It returns an error if stock or timing is the zero value, or if kind is
// empty.
func NewCompanyEvent(stock identifier.FIGI, timing Timing, kind string) (CompanyEvent, error) {
	e := CompanyEvent{stock: stock, timing: timing, kind: kind}
	if err := e.validate(); err != nil {
		return CompanyEvent{}, fmt.Errorf("invalid company event: %w", err)
	}

	return e, nil
}

// Stock returns the composite FIGI of the stock the event is about.
func (e CompanyEvent) Stock() identifier.FIGI { return e.stock }

// Subject returns the event's stock, as its subject.
func (e CompanyEvent) Subject() Subject { return Subject{stock: e.stock} }

// Timing returns when the event happens.
func (e CompanyEvent) Timing() Timing { return e.timing }

// Kind returns the kind of event, such as "buyback".
func (e CompanyEvent) Kind() string { return e.kind }

func (e CompanyEvent) validate() error {
	if !e.stock.IsValid() {
		return errors.New("stock is missing")
	}

	if !e.timing.IsValid() {
		return errors.New("timing is missing")
	}

	if e.kind == "" {
		return errors.New("kind is missing")
	}

	return nil
}
