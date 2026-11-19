// Package fxcalendar wires the built-in trading calendars into an Uber FX
// application.
//
// [FxCalendarModule] provides a [calendar.Registry] of them, and the calendar
// of every built-in exchange as a [calendar.TradingCalendar] named after the
// exchange's code. A consumer asks for one by name:
//
//	fx.Provide(fx.Annotate(entrygate.New, fx.ParamTags(`name:"XNYS"`)))
package fxcalendar
