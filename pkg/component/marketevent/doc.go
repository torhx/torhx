// Package marketevent models scheduled market events: economic releases,
// company events and a program's own.
//
// Every [Event] says when it happens, by its [Timing]; what it is, by its
// kind, such as "CPI" or "earnings"; and what it is about, by its [Subject]:
// a [Region] for an economic event, a stock for a company event.
// [EconomicEvent] is every economic event, and [Earnings], [Dividend] and
// [Split] are the built-in company events. A program's own type of event has
// the three methods of [Event] itself, or embeds [EconomicEvent] or
// [CompanyEvent] and adds fields.
//
// A [Timing] is an exact instant, or a date that may carry a [Flag] for
// before the market opens or after it closes. A flag is what the event's
// source states. Nothing here knows a market's hours, so nothing here works
// out whether an instant comes before an open.
//
// Code reads events through a [Reader], which narrows to a subject and reads
// by date. [Events] is the Reader built from the events a program loads. A
// read gives a [Selection], whose Of takes the events of one type out as
// that type.
//
// Nothing here loads or stores events. A program maps its rows to events
// and builds one [Events] of them.
package marketevent
