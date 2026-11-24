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
// Every [Timing] carries a [civil.LocationDate] in the event source's time
// zone. An exact timing also carries its instant. A date alone may carry a
// [Flag] when the source states that the event comes before the market opens
// or after it closes. The exchange's trading calendar supplies session hours.
//
// [Events] holds the events a program loads and their coverage as a half-open
// [civil.DateRange]. [Events.During] narrows coverage to the shared dates.
// Narrowing by subject, date or kind gives an Events again.
// Its reads say whether it has events, list them, and take the events of one
// type out as that type.
//
// Nothing here loads or stores events. A program maps its rows to events
// and builds one [Events] of them.
package marketevent
