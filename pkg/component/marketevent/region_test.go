package marketevent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talav/country"
	"github.com/talav/currency"

	"github.com/torhx/torhx/pkg/component/marketevent"
)

// mustCurrency returns the ISO 4217 currency with the given code.
func mustCurrency(t *testing.T, code string) currency.Currency {
	t.Helper()

	c, err := currency.NewCurrencyFromString(code)
	require.NoError(t, err)

	return c
}

// mustCountryRegion returns the region of the country c.
func mustCountryRegion(t *testing.T, c country.Country) marketevent.Region {
	t.Helper()

	r, err := marketevent.NewRegionFromCountry(c)
	require.NoError(t, err)

	return r
}

// mustCurrencyRegion returns the currency zone of the ISO 4217 currency with
// the given code.
func mustCurrencyRegion(t *testing.T, code string) marketevent.Region {
	t.Helper()

	r, err := marketevent.NewRegionFromCurrency(mustCurrency(t, code))
	require.NoError(t, err)

	return r
}

func TestNewRegion(t *testing.T) {
	t.Parallel()

	eur := mustCurrency(t, "EUR")

	tests := []struct {
		name         string
		build        func() (marketevent.Region, error)
		wantCountry  country.Country
		wantCurrency currency.Currency
		wantString   string
	}{
		{
			name:        "country",
			build:       func() (marketevent.Region, error) { return marketevent.NewRegionFromCountry(country.US) },
			wantCountry: country.US,
			wantString:  "US",
		},
		{
			name:         "currency zone",
			build:        func() (marketevent.Region, error) { return marketevent.NewRegionFromCurrency(eur) },
			wantCurrency: eur,
			wantString:   "EUR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			region, err := tt.build()
			require.NoError(t, err)

			gotCountry, isCountry := region.Country()
			gotCurrency, isCurrency := region.Currency()

			assert.Equal(t, tt.wantCountry, gotCountry)
			assert.Equal(t, tt.wantCountry.IsValid(), isCountry)
			assert.Equal(t, tt.wantCurrency, gotCurrency)
			assert.Equal(t, tt.wantCurrency.IsValid(), isCurrency)
			assert.Equal(t, tt.wantString, region.String())
			assert.True(t, region.IsValid())
		})
	}
}

func TestNewRegionRejects(t *testing.T) {
	t.Parallel()

	_, err := marketevent.NewRegionFromCountry(country.Country{})
	require.ErrorContains(t, err, "no country and no currency")

	_, err = marketevent.NewRegionFromCurrency(currency.Currency{})
	require.ErrorContains(t, err, "no country and no currency")
}

func TestRegionIsComparable(t *testing.T) {
	t.Parallel()

	us := mustCountryRegion(t, country.US)
	euroArea := mustCurrencyRegion(t, "EUR")

	byRegion := map[marketevent.Region]string{us: "US", euroArea: "euro area"}

	assert.Equal(t, "US", byRegion[mustCountryRegion(t, country.US)])
	assert.Equal(t, "euro area", byRegion[mustCurrencyRegion(t, "EUR")])
	assert.NotContains(t, byRegion, mustCountryRegion(t, country.CA))
	assert.NotContains(t, byRegion, mustCurrencyRegion(t, "USD"))
	assert.False(t, marketevent.Region{}.IsValid())
}
