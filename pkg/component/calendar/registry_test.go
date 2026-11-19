package calendar_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/torhx/torhx/pkg/component/calendar"
)

func TestNewRegistryRejects(t *testing.T) {
	t.Parallel()

	xnys := newXNYS(t)

	tests := []struct {
		name      string
		calendars []calendar.TradingCalendar
		wantErr   string
	}{
		{name: "zero calendar", calendars: []calendar.TradingCalendar{xnys, {}}, wantErr: "calendar 1 is not a calendar"},
		{name: "one ID twice", calendars: []calendar.TradingCalendar{xnys, xnys}, wantErr: "calendar XNYS is given twice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := calendar.NewRegistry(tt.calendars)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestRegistryGet(t *testing.T) {
	t.Parallel()

	registry, err := calendar.NewRegistry([]calendar.TradingCalendar{newXNYS(t), newXTKS(t)})
	require.NoError(t, err)

	_, err = calendar.Registry{}.Get(mustID(t, "XNYS"))
	require.ErrorContains(t, err, "no trading calendar XNYS", "the zero value holds no calendar")

	tests := []struct {
		name    string
		id      string
		wantErr string
	}{
		{name: "first calendar", id: "XNYS"},
		{name: "second calendar", id: "XTKS"},
		{name: "unknown calendar", id: "XLON", wantErr: "no trading calendar XLON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := registry.Get(mustID(t, tt.id))
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.id, got.ID().String())
		})
	}
}
