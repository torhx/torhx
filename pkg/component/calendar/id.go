package calendar

import (
	"fmt"
	"strings"
)

const (
	minIDLen = 2
	maxIDLen = 32
)

// An ID names a calendar, so one exchange can have several and an instrument
// can say which one it trades on: XNYS, XCME_EQUITY_INDEX. An exchange's
// only calendar takes the exchange code as its ID. It is 2 to 32 upper-case
// ASCII letters, digits or underscores.
//
// An ID comes only from a constructor, so it is always well formed. IDs are
// comparable with == and usable as map keys. The zero value is not an ID.
type ID struct {
	// value is the ID in upper case.
	value string
}

// NewID returns the ID written in s. Letters may be in any case, and white
// space around it is ignored.
//
// It returns an error if s is not 2 to 32 ASCII letters, digits or
// underscores.
func NewID(s string) (ID, error) {
	id := ID{value: strings.Map(toUpperASCII, strings.TrimSpace(s))}
	if err := id.validate(); err != nil {
		return ID{}, fmt.Errorf("invalid calendar ID %q: %w", s, err)
	}

	return id, nil
}

// String returns the ID as text, such as "XNYS".
func (id ID) String() string { return id.value }

// IsValid reports whether id is an ID. It is false only for the zero value,
// which stands for a missing one; every ID from a constructor is valid.
func (id ID) IsValid() bool { return id.value != "" }

func (id ID) validate() error {
	if len(id.value) < minIDLen || len(id.value) > maxIDLen {
		return fmt.Errorf("must be %d to %d characters", minIDLen, maxIDLen)
	}

	for _, r := range id.value {
		if !isUpper(r) && !isDigit(r) && r != '_' {
			return fmt.Errorf("character %q must be an ASCII letter, a digit or an underscore", r)
		}
	}

	return nil
}

// toUpperASCII maps the letters a to z to upper case and leaves any other
// rune alone. strings.ToUpper would also map some non-ASCII letters to A to
// Z, such as ſ to S, and let them into an ID.
func toUpperASCII(r rune) rune {
	if isLower(r) {
		return r - 'a' + 'A'
	}

	return r
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }

func isLower(r rune) bool { return r >= 'a' && r <= 'z' }

func isDigit(r rune) bool { return r >= '0' && r <= '9' }
