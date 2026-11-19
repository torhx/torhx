package calendar

import (
	"fmt"
	"slices"
	"time"
)

// A SessionKind tells regular trading hours from extended hours, where order
// rules and liquidity differ.
//
// The zero value is not a kind.
type SessionKind string

const (
	// SessionKindRegular is trading in the exchange's regular hours.
	SessionKindRegular SessionKind = "regular"

	// SessionKindExtended is trading outside the regular hours: before the
	// market opens, after it closes, or overnight.
	SessionKindExtended SessionKind = "extended"
)

func (k SessionKind) validate() error {
	switch k {
	case SessionKindRegular, SessionKindExtended:
		return nil
	default:
		return fmt.Errorf("unrecognised session kind %q", k)
	}
}

// A Session is one stretch of trading: from an open instant to a close
// instant, of one kind. Its hours are the continuous trading hours the
// exchange publishes; it says nothing of auctions or other phases inside
// them.
//
// A Session comes only from a constructor, so it has a kind and closes after
// it opens. The zero value is not a session.
type Session struct {
	// kind says whether the session is regular or extended trading.
	kind SessionKind

	// open is the instant trading starts. It is part of the session.
	open time.Time

	// close is the instant trading stops, after open. It is not part of the
	// session.
	close time.Time
}

// NewSession returns the session of the given kind from openAt to closeAt.
//
// It returns an error if kind is not a kind, or if closeAt is not after
// openAt.
func NewSession(kind SessionKind, openAt, closeAt time.Time) (Session, error) {
	s := Session{kind: kind, open: openAt, close: closeAt}
	if err := s.validate(); err != nil {
		return Session{}, fmt.Errorf("invalid session: %w", err)
	}

	return s, nil
}

// Kind returns whether the session is regular or extended trading.
func (s Session) Kind() SessionKind { return s.kind }

// Open returns the instant trading starts.
func (s Session) Open() time.Time { return s.open }

// Close returns the instant trading stops.
func (s Session) Close() time.Time { return s.close }

// Duration returns how long the session lasts.
func (s Session) Duration() time.Duration { return s.close.Sub(s.open) }

// IsValid reports whether s is a session. It is false only for the zero
// value, which stands for a missing one; every Session from a constructor is
// valid.
func (s Session) IsValid() bool { return s.kind != "" }

// Contains reports whether trading is on at the instant t. The open is
// inside the session and the close is outside it.
func (s Session) Contains(t time.Time) bool {
	return !t.Before(s.open) && t.Before(s.close)
}

func (s Session) validate() error {
	if err := s.kind.validate(); err != nil {
		return err
	}

	if !s.close.After(s.open) {
		return fmt.Errorf("close %s must be after open %s", s.close, s.open)
	}

	return nil
}

// in returns the session with its instants shown in loc.
func (s Session) in(loc *time.Location) Session {
	return Session{kind: s.kind, open: s.open.In(loc), close: s.close.In(loc)}
}

// firstRegularOpen returns the open of the first regular session among
// sessions, which are in time order. It reports false when none is regular.
func firstRegularOpen(sessions []Session) (time.Time, bool) {
	for _, s := range sessions {
		if s.kind == SessionKindRegular {
			return s.open, true
		}
	}

	return time.Time{}, false
}

// lastRegularClose returns the close of the last regular session among
// sessions, which are in time order. It reports false when none is regular.
func lastRegularClose(sessions []Session) (time.Time, bool) {
	for _, s := range slices.Backward(sessions) {
		if s.kind == SessionKindRegular {
			return s.close, true
		}
	}

	return time.Time{}, false
}
