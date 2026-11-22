package marketevent

import (
	"errors"
	"fmt"
	"math"

	"github.com/torhx/identifier"
)

// Earnings is an earnings report: the stock it is on and when it comes. It
// holds no figures; a program that needs them, such as an estimate, has a
// type of its own.
//
// Earnings comes only from NewEarnings. The zero value is not a report.
type Earnings struct {
	// stock is the composite FIGI of the stock the report is on.
	stock identifier.FIGI

	// timing is when the report comes.
	timing Timing
}

// NewEarnings returns the report on stock, by its composite FIGI, at timing.
//
// It returns an error if stock or timing is the zero value.
func NewEarnings(stock identifier.FIGI, timing Timing) (Earnings, error) {
	e := Earnings{stock: stock, timing: timing}
	if err := e.validate(); err != nil {
		return Earnings{}, fmt.Errorf("invalid earnings: %w", err)
	}

	return e, nil
}

// Stock returns the composite FIGI of the stock the report is on.
func (e Earnings) Stock() identifier.FIGI { return e.stock }

// Subject returns the report's stock, as its subject.
func (e Earnings) Subject() Subject { return Subject{stock: e.stock} }

// Timing returns when the report comes.
func (e Earnings) Timing() Timing { return e.timing }

// Kind returns "earnings".
func (e Earnings) Kind() string { return "earnings" }

func (e Earnings) event() {}

func (e Earnings) validate() error {
	if !e.stock.IsValid() {
		return errors.New("stock is missing")
	}

	if !e.timing.IsValid() {
		return errors.New("timing is missing")
	}

	return nil
}

// A Dividend is a cash dividend on a stock. Its timing is the ex-date, a
// date alone: a stock goes ex at the open of a date, not at a time of day.
// The holder on the day before is owed the amount, and the cash comes later,
// on the payment date, which a Dividend does not hold.
//
// A Dividend comes only from NewDividend. The zero value is not a dividend.
type Dividend struct {
	// stock is the composite FIGI of the stock the dividend is paid on.
	stock identifier.FIGI

	// timing is the ex-date: a date without a flag.
	timing Timing

	// amount is the amount per share, in the currency of the stock's price.
	// It is positive and finite.
	amount float64
}

// NewDividend returns the dividend on stock, by its composite FIGI, going ex
// at timing, of amount per share in the currency of the stock's price.
//
// It returns an error if stock or timing is the zero value, if timing is
// exact or has a flag, or if amount is not positive or not finite.
func NewDividend(stock identifier.FIGI, timing Timing, amount float64) (Dividend, error) {
	d := Dividend{stock: stock, timing: timing, amount: amount}
	if err := d.validate(); err != nil {
		return Dividend{}, fmt.Errorf("invalid dividend: %w", err)
	}

	return d, nil
}

// Stock returns the composite FIGI of the stock the dividend is paid on.
func (d Dividend) Stock() identifier.FIGI { return d.stock }

// Subject returns the dividend's stock, as its subject.
func (d Dividend) Subject() Subject { return Subject{stock: d.stock} }

// Timing returns the ex-date.
func (d Dividend) Timing() Timing { return d.timing }

// Kind returns "dividend".
func (d Dividend) Kind() string { return "dividend" }

// Amount returns the amount per share, in the currency of the stock's price.
func (d Dividend) Amount() float64 { return d.amount }

func (d Dividend) event() {}

func (d Dividend) validate() error {
	if !d.stock.IsValid() {
		return errors.New("stock is missing")
	}

	if err := validateExDate(d.timing); err != nil {
		return err
	}

	if math.IsNaN(d.amount) || math.IsInf(d.amount, 0) || d.amount <= 0 {
		return fmt.Errorf("amount %v is not positive and finite", d.amount)
	}

	return nil
}

// A Split changes the number of shares of a stock. Its timing is the
// ex-date, a date alone.
//
// A Split comes only from NewSplit. The zero value is not a split.
type Split struct {
	// stock is the composite FIGI of the stock that splits.
	stock identifier.FIGI

	// timing is the ex-date: a date without a flag.
	timing Timing

	// factor is the number that prices before the ex-date are multiplied
	// by, 0.25 for a four-for-one split. It is positive, finite and not 1.
	factor float64
}

// NewSplit returns the split of stock, by its composite FIGI, going ex at
// timing, by factor: the number that prices before the ex-date are
// multiplied by, 0.25 for a four-for-one split.
//
// It returns an error if stock or timing is the zero value, if timing is
// exact or has a flag, or if factor is not positive, not finite, or 1.
func NewSplit(stock identifier.FIGI, timing Timing, factor float64) (Split, error) {
	s := Split{stock: stock, timing: timing, factor: factor}
	if err := s.validate(); err != nil {
		return Split{}, fmt.Errorf("invalid split: %w", err)
	}

	return s, nil
}

// Stock returns the composite FIGI of the stock that splits.
func (s Split) Stock() identifier.FIGI { return s.stock }

// Subject returns the split's stock, as its subject.
func (s Split) Subject() Subject { return Subject{stock: s.stock} }

// Timing returns the ex-date.
func (s Split) Timing() Timing { return s.timing }

// Kind returns "split".
func (s Split) Kind() string { return "split" }

// Factor returns the number that prices before the ex-date are multiplied
// by.
func (s Split) Factor() float64 { return s.factor }

func (s Split) event() {}

func (s Split) validate() error {
	if !s.stock.IsValid() {
		return errors.New("stock is missing")
	}

	if err := validateExDate(s.timing); err != nil {
		return err
	}

	if math.IsNaN(s.factor) || math.IsInf(s.factor, 0) || s.factor <= 0 {
		return fmt.Errorf("factor %v is not positive and finite", s.factor)
	}

	if s.factor == 1 {
		return errors.New("factor 1 changes no price")
	}

	return nil
}

// validateExDate returns an error unless timing is a date without a flag: a
// stock goes ex at the open of a date, so an ex-date has no time of day.
func validateExDate(timing Timing) error {
	if !timing.IsValid() {
		return errors.New("timing is missing")
	}

	if _, exact := timing.Instant(); exact {
		return errors.New("ex-date is an exact time, not a date")
	}

	if flag, flagged := timing.Flag(); flagged {
		return fmt.Errorf("ex-date has the flag %s", flag)
	}

	return nil
}
