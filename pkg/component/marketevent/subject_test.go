package marketevent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talav/country"
	"github.com/torhx/identifier"

	"github.com/torhx/torhx/pkg/component/marketevent"
)

// The composite FIGIs of the stocks the tests are about.
const (
	aaplFIGI = "BBG000B9XRY4"
	nvdaFIGI = "BBG000BBJQV0"
)

// mustFIGI returns the FIGI written in s.
func mustFIGI(t *testing.T, s string) identifier.FIGI {
	t.Helper()

	id, err := identifier.NewFIGI(s)
	require.NoError(t, err)

	return id
}

// mustRegionSubject returns the subject of the region r.
func mustRegionSubject(t *testing.T, r marketevent.Region) marketevent.Subject {
	t.Helper()

	s, err := marketevent.NewSubjectFromRegion(r)
	require.NoError(t, err)

	return s
}

// mustStockSubject returns the subject of the stock with the FIGI written in
// figi.
func mustStockSubject(t *testing.T, figi string) marketevent.Subject {
	t.Helper()

	s, err := marketevent.NewSubjectFromStock(mustFIGI(t, figi))
	require.NoError(t, err)

	return s
}

func TestNewSubject(t *testing.T) {
	t.Parallel()

	us := mustCountryRegion(t, country.US)
	aapl := mustFIGI(t, aaplFIGI)

	tests := []struct {
		name       string
		build      func() (marketevent.Subject, error)
		wantRegion marketevent.Region
		wantStock  identifier.FIGI
		wantString string
	}{
		{
			name:       "region",
			build:      func() (marketevent.Subject, error) { return marketevent.NewSubjectFromRegion(us) },
			wantRegion: us,
			wantString: "US",
		},
		{
			name:       "stock",
			build:      func() (marketevent.Subject, error) { return marketevent.NewSubjectFromStock(aapl) },
			wantStock:  aapl,
			wantString: aaplFIGI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			subject, err := tt.build()
			require.NoError(t, err)

			gotRegion, isRegion := subject.Region()
			gotStock, isStock := subject.Stock()

			assert.Equal(t, tt.wantRegion, gotRegion)
			assert.Equal(t, tt.wantRegion.IsValid(), isRegion)
			assert.Equal(t, tt.wantStock, gotStock)
			assert.Equal(t, tt.wantStock.IsValid(), isStock)
			assert.Equal(t, tt.wantString, subject.String())
			assert.True(t, subject.IsValid())
		})
	}
}

func TestNewSubjectRejects(t *testing.T) {
	t.Parallel()

	_, err := marketevent.NewSubjectFromRegion(marketevent.Region{})
	require.ErrorContains(t, err, "no region and no stock")

	_, err = marketevent.NewSubjectFromStock(identifier.FIGI{})
	require.ErrorContains(t, err, "no region and no stock")
}

func TestSubjectIsComparable(t *testing.T) {
	t.Parallel()

	us := mustRegionSubject(t, mustCountryRegion(t, country.US))
	aapl := mustStockSubject(t, aaplFIGI)

	bySubject := map[marketevent.Subject]string{us: "US", aapl: "AAPL"}

	assert.Equal(t, "US", bySubject[mustRegionSubject(t, mustCountryRegion(t, country.US))])
	assert.Equal(t, "AAPL", bySubject[mustStockSubject(t, aaplFIGI)])
	assert.NotContains(t, bySubject, mustStockSubject(t, nvdaFIGI))
	assert.NotContains(t, bySubject, mustRegionSubject(t, mustCurrencyRegion(t, "EUR")))

	_, isRegion := marketevent.Subject{}.Region()
	_, isStock := marketevent.Subject{}.Stock()

	assert.False(t, isRegion)
	assert.False(t, isStock)
	assert.False(t, marketevent.Subject{}.IsValid())
}
