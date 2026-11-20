# fxcalendar

FX module for [pkg/component/calendar](../../component/calendar). Provides the built-in trading calendars.

## Quick start

```go
fx.New(
    fxcalendar.FxCalendarModule,
    fx.Provide(fx.Annotate(entrygate.New, fx.ParamTags(`name:"XNYS"`))),
)
```

## Injected types

- `calendar.TradingCalendar` for every exchange the exchange package knows, named after its code, such as `name:"XNYS"`.
- `calendar.Registry`: every built-in calendar, looked up by ID with `registry.Get(id)`, for code that picks a calendar at run time.

## Asking for a calendar

A component takes a plain `calendar.TradingCalendar`, and its fx module names the exchange:

```go
// pkg/component/entrygate
func New(cal calendar.TradingCalendar) *Gate

// pkg/fx/fxentrygate
fx.Provide(fx.Annotate(entrygate.New, fx.ParamTags(`name:"XNYS"`)))
```

A field of an `fx.In` struct works too:

```go
Calendar calendar.TradingCalendar `name:"XNYS"`
```

The built-in calendars are built once, the first time something asks for a calendar or the registry. Asking for an exchange that has no calendar data fails when the application starts, with `no trading calendar <code>`.
