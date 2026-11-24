package calendar

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/talav/civil"
	"github.com/torhx/exchange"
)

// builtinData holds one JSON file per exchange that has a built-in calendar.
//
//go:embed data/*.json
var builtinData embed.FS

// calendarFile is the JSON form of a built-in calendar.
type calendarFile struct {
	// Exchange is the code of the calendar's exchange, which is also the
	// calendar's ID.
	Exchange exchange.Code `json:"exchange"`

	// Coverage holds the dates the calendar covers, including holidays and
	// weekends. Its latest years may be projected from the exchange's rules.
	Coverage civil.DateRange `json:"coverage"`

	// Exceptions are the dates whose sessions differ from the exchange's
	// regular hours, in date order.
	Exceptions []exceptionDay `json:"exceptions"`
}

// exceptionDay gives every session of a date that differs from the exchange's
// regular hours. A date with no session is closed.
type exceptionDay struct {
	// Date is the date the sessions are booked to.
	Date civil.Date `json:"date"`

	// Note says why the date differs, for people who check the data.
	Note string `json:"note"`

	// Sessions are the regular sessions of the date, in time order.
	Sessions []sessionTimes `json:"sessions"`
}

// sessionTimes is a session as wall-clock times in the exchange's time zone.
// Pointers distinguish missing or null fields from an explicit midnight.
type sessionTimes struct {
	// Open is the required start time.
	Open *civil.Time `json:"open"`

	// Close is the required end time. At or before Open, it is on the next date.
	Close *civil.Time `json:"close"`
}

// TradingCalendars returns the built-in calendars, one for each exchange that
// has calendar data, such as XNYS.
//
// It returns an error only if the built-in data is broken, which is a defect
// in Torhx.
func TradingCalendars() ([]TradingCalendar, error) {
	names, err := fs.Glob(builtinData, "data/*.json")
	if err != nil {
		return nil, err
	}

	calendars := make([]TradingCalendar, 0, len(names))

	for _, name := range names {
		content, err := builtinData.ReadFile(name)
		if err != nil {
			return nil, err
		}

		c, err := loadTradingCalendar(content)
		if err != nil {
			return nil, fmt.Errorf("built-in calendar %s: %w", name, err)
		}

		calendars = append(calendars, c)
	}

	return calendars, nil
}

// loadTradingCalendar builds the calendar in the JSON content: the exchange's
// regular hours on the weekdays it trades, with the exceptions in their place
// on the dates they list.
func loadTradingCalendar(content []byte) (TradingCalendar, error) {
	// A mistyped key is rejected rather than ignored, so it cannot silently
	// close a date.
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()

	var file calendarFile
	if err := decoder.Decode(&file); err != nil {
		return TradingCalendar{}, err
	}

	if !file.Coverage.IsValid() {
		return TradingCalendar{}, errors.New("calendar coverage is missing")
	}

	e, err := exchange.NewExchangeFromCode(file.Exchange)
	if err != nil {
		return TradingCalendar{}, err
	}

	id, err := NewID(file.Exchange.String())
	if err != nil {
		return TradingCalendar{}, err
	}

	week, err := regularWeek(e, file.Coverage.Start())
	if err != nil {
		return TradingCalendar{}, err
	}

	exceptions, err := exceptionSessions(file.Exceptions, e.Location())
	if err != nil {
		return TradingCalendar{}, err
	}

	return BuildTradingCalendar(id, e, file.Coverage, []WeeklySchedule{week}, exceptions)
}

// regularWeek returns the exchange's regular sessions on each weekday it
// trades, as a weekly schedule that applies from the date from.
func regularWeek(e exchange.Exchange, from civil.Date) (WeeklySchedule, error) {
	var days [7][]Hours

	for _, s := range e.Sessions() {
		// An exchange session that ends at midnight ends with its day, as do
		// Hours that end at or before they start. Regular hours that start on
		// their own date are always valid, so they need no constructor.
		h := Hours{kind: SessionKindRegular, start: s.Start(), end: s.End()}

		for _, weekday := range e.Weekdays() {
			days[weekday] = append(days[weekday], h)
		}
	}

	return NewWeeklySchedule(from, days)
}

// exceptionSessions returns the sessions of each exception date, with their
// times read on a clock in loc.
func exceptionSessions(days []exceptionDay, loc *time.Location) (map[civil.Date][]Session, error) {
	exceptions := make(map[civil.Date][]Session, len(days))

	for _, day := range days {
		if _, ok := exceptions[day.Date]; ok {
			return nil, fmt.Errorf("%s is listed twice", day.Date)
		}

		// Present with no session, the date is closed.
		exceptions[day.Date] = nil

		for i, times := range day.Sessions {
			if times.Open == nil {
				return nil, fmt.Errorf("%s: session %d: open is required", day.Date, i)
			}

			if times.Close == nil {
				return nil, fmt.Errorf("%s: session %d: close is required", day.Date, i)
			}

			s, err := Hours{kind: SessionKindRegular, start: *times.Open, end: *times.Close}.session(day.Date, loc)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", day.Date, err)
			}

			exceptions[day.Date] = append(exceptions[day.Date], s)
		}
	}

	return exceptions, nil
}
