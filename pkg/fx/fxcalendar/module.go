package fxcalendar

import (
	"fmt"

	"github.com/torhx/exchange"
	"go.uber.org/fx"

	"github.com/torhx/torhx/pkg/component/calendar"
)

// FxCalendarModule provides a [calendar.Registry] of the built-in trading
// calendars, and the calendar of every built-in exchange as a
// [calendar.TradingCalendar] named after the exchange's code, such as
// name:"XNYS". FX runs a provider only when something asks for its value, so
// asking for an exchange that has no calendar data fails when the application
// starts.
var FxCalendarModule = fx.Module(
	"calendar",
	fx.Provide(newRegistry),
	namedCalendars(),
)

// newRegistry returns the registry of the built-in trading calendars.
func newRegistry() (calendar.Registry, error) {
	calendars, err := calendar.TradingCalendars()
	if err != nil {
		return calendar.Registry{}, fmt.Errorf("built-in trading calendars: %w", err)
	}

	return calendar.NewRegistry(calendars)
}

// namedCalendars provides the calendar of every built-in exchange from the
// registry, named after the exchange's code.
func namedCalendars() fx.Option {
	exchanges, err := exchange.Exchanges()
	if err != nil {
		return fx.Error(fmt.Errorf("built-in exchanges: %w", err))
	}

	options := make([]fx.Option, 0, len(exchanges))

	for _, e := range exchanges {
		code := e.Code().String()

		options = append(options, fx.Provide(fx.Annotate(
			func(r calendar.Registry) (calendar.TradingCalendar, error) {
				id, err := calendar.NewID(code)
				if err != nil {
					return calendar.TradingCalendar{}, err
				}

				return r.Get(id)
			},
			fx.ResultTags(`name:"`+code+`"`),
		)))
	}

	return fx.Options(options...)
}
