// Package calendar says when a market trades.
//
// A [TradingCalendar] holds a day for every date of one exchange or product.
// A [TradingDay] is a date and the sessions the market books to it: a
// business day, a weekend or a holiday. A [Session] is one stretch of trading
// between two instants. A session belongs to one day and the day has the
// date, so trading on a Sunday evening can belong to Monday.
//
// A calendar is built once and never changes. [NewTradingCalendar] takes days
// that are already settled; [BuildTradingCalendar] expands weekly schedules
// and dated exceptions into them. The schedule rules stay in the builder,
// and the calendar holds only the result.
//
// Nothing here reads a clock. Every query takes the instant or the date it
// asks about.
package calendar
