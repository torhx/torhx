package calendar_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/torhx/torhx/pkg/component/calendar"
)

func mustID(t *testing.T, s string) calendar.ID {
	t.Helper()

	id, err := calendar.NewID(s)
	require.NoError(t, err)

	return id
}

func TestNewID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "exchange code", input: "XNYS", want: "XNYS"},
		{name: "product group", input: "XCME_EQUITY_INDEX", want: "XCME_EQUITY_INDEX"},
		{name: "lower case and spaces", input: " xnys ", want: "XNYS"},
		{name: "longest", input: strings.Repeat("A", 32), want: strings.Repeat("A", 32)},
		{name: "empty", input: "", wantErr: "must be 2 to 32 characters"},
		{name: "too short", input: "X", wantErr: "must be 2 to 32 characters"},
		{name: "too long", input: strings.Repeat("A", 33), wantErr: "must be 2 to 32 characters"},
		{name: "hyphen", input: "XCME-ES", wantErr: "must be an ASCII letter, a digit or an underscore"},
		{name: "long s, which strings.ToUpper maps to S", input: "ſpx", wantErr: "must be an ASCII letter, a digit or an underscore"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := calendar.NewID(tt.input)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, id.String())
		})
	}
}
