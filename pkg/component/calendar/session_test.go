package calendar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/torhx/torhx/pkg/component/calendar"
)

const instantLayout = "2006-01-02 15:04"

// mustAt returns the instant at which a clock in loc shows s.
func mustAt(t *testing.T, loc *time.Location, s string) time.Time {
	t.Helper()

	at, err := time.ParseInLocation(instantLayout, s, loc)
	require.NoError(t, err)

	return at
}

// mustSession returns the session of the given kind between two wall-clock
// readings in loc.
func mustSession(t *testing.T, kind calendar.SessionKind, loc *time.Location, openAt, closeAt string) calendar.Session {
	t.Helper()

	s, err := calendar.NewSession(kind, mustAt(t, loc, openAt), mustAt(t, loc, closeAt))
	require.NoError(t, err)

	return s
}

func TestNewSession(t *testing.T) {
	t.Parallel()

	openAt := time.Date(2026, time.November, 16, 9, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		kind    calendar.SessionKind
		closeAt time.Time
		wantErr string
	}{
		{name: "regular", kind: calendar.SessionKindRegular, closeAt: openAt.Add(time.Hour)},
		{name: "extended", kind: calendar.SessionKindExtended, closeAt: openAt.Add(time.Hour)},
		{name: "no kind", kind: "", closeAt: openAt.Add(time.Hour), wantErr: "unrecognised session kind"},
		{name: "unknown kind", kind: "overnight", closeAt: openAt.Add(time.Hour), wantErr: "unrecognised session kind"},
		{name: "close at open", kind: calendar.SessionKindRegular, closeAt: openAt, wantErr: "must be after open"},
		{name: "close before open", kind: calendar.SessionKindRegular, closeAt: openAt.Add(-time.Hour), wantErr: "must be after open"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, err := calendar.NewSession(tt.kind, openAt, tt.closeAt)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.kind, s.Kind())
			assert.Equal(t, openAt, s.Open())
			assert.Equal(t, tt.closeAt, s.Close())
			assert.Equal(t, time.Hour, s.Duration())
		})
	}
}

func TestSessionContains(t *testing.T) {
	t.Parallel()

	openAt := time.Date(2026, time.November, 16, 9, 30, 0, 0, time.UTC)
	closeAt := openAt.Add(time.Hour)

	s, err := calendar.NewSession(calendar.SessionKindRegular, openAt, closeAt)
	require.NoError(t, err)

	tests := []struct {
		name string
		at   time.Time
		want bool
	}{
		{name: "before the open", at: openAt.Add(-time.Nanosecond), want: false},
		{name: "at the open", at: openAt, want: true},
		{name: "inside", at: openAt.Add(time.Minute), want: true},
		{name: "just before the close", at: closeAt.Add(-time.Nanosecond), want: true},
		{name: "at the close", at: closeAt, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, s.Contains(tt.at))
		})
	}
}
