package marketevent

import (
	"cmp"
	"errors"
	"fmt"

	"github.com/talav/country"
	"github.com/talav/currency"
)

// A Region is an economic event's region: a country, or a currency zone
// named by its currency, as EUR names the euro area.
//
// A Region comes only from a constructor. Regions are comparable with == and
// usable as map keys. The zero value is not a region.
type Region struct {
	// country is the country of a country's region. It is the zero country
	// for a currency zone.
	country country.Country

	// currency is the currency that names a currency zone. It is the zero
	// currency for a country's region.
	currency currency.Currency
}

// NewRegionFromCountry returns the region of the country c.
//
// It returns an error if c is the zero country.
func NewRegionFromCountry(c country.Country) (Region, error) {
	r := Region{country: c}
	if err := r.validate(); err != nil {
		return Region{}, fmt.Errorf("invalid region: %w", err)
	}

	return r, nil
}

// NewRegionFromCurrency returns the currency zone of c.
//
// It returns an error if c is the zero currency.
func NewRegionFromCurrency(c currency.Currency) (Region, error) {
	r := Region{currency: c}
	if err := r.validate(); err != nil {
		return Region{}, fmt.Errorf("invalid region: %w", err)
	}

	return r, nil
}

// Country returns the country of a country's region. It reports false for a
// currency zone.
func (r Region) Country() (country.Country, bool) { return r.country, r.country.IsValid() }

// Currency returns the currency of a currency zone. It reports false for a
// country.
func (r Region) Currency() (currency.Currency, bool) { return r.currency, r.currency.IsValid() }

// IsValid reports whether r is a region. It is false only for the zero
// value.
func (r Region) IsValid() bool { return r != Region{} }

// String returns the region's code: a country's ISO 3166-1 alpha-2 code,
// such as "US", or a currency zone's ISO 4217 code, such as "EUR".
func (r Region) String() string {
	if r.country.IsValid() {
		return r.country.Alpha2()
	}

	return r.currency.Code()
}

func (r Region) validate() error {
	if !r.country.IsValid() && !r.currency.IsValid() {
		return errors.New("no country and no currency")
	}

	return nil
}

// compare orders regions: countries before currency zones, each by its
// code. It is zero only for equal regions, so it also tells two currencies
// of one code apart, by their decimal places.
func (r Region) compare(other Region) int {
	return cmp.Or(
		cmp.Compare(r.currency.Code(), other.currency.Code()),
		cmp.Compare(r.currency.Decimals(), other.currency.Decimals()),
		cmp.Compare(r.country.Alpha2(), other.country.Alpha2()),
	)
}
