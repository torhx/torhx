# marketevent

Scheduled market events: economic releases, company events and a program's own. Every event says when it happens, what kind it is and what it is about, and an `Events` finds them by subject, by date and by kind. See the package documentation for the API.

## Events

| Type | What it is | Kind |
|---|---|---|
| `EconomicEvent` | An economic event about a region, such as a release of the US CPI. | The name its source's normalizer gives it, such as `CPI`. |
| `Earnings` | An earnings report on a stock. | `earnings` |
| `Dividend` | A cash dividend with its amount per share, dated by its ex-date. | `dividend` |
| `Split` | A split with the factor that earlier prices are multiplied by, dated by its ex-date. | `split` |
| A program's own | Any type with the `Timing`, `Kind` and `Subject` methods of `Event`. Embedding `EconomicEvent` or `CompanyEvent` gives a type all three. | Its own. |

An event's subject is a region or a stock. A region is a country, or a currency zone named by its currency, as `EUR` names the euro area. A stock is named by its composite FIGI.

## Timing

Every timing carries a `civil.LocationDate`, the event's calendar date in its source's named time zone. For a source that reports only a date, pass that day to `NewDateTiming`, `NewBeforeMarketTiming` or `NewAfterMarketTiming` according to what the source states.

For an exact timestamp reported against an exchange, the exchange supplies its local date and time zone.

```go
release, err := marketevent.NewExactTimingForExchange(instant, xnys.Exchange())
```

When the source gives a local date and zone alongside the instant, `NewExactTiming(instant, day)` checks that they agree. Events on the same local date can reuse the same `civil.LocationDate`. `Timing.LocationDate()` returns that value, and `Date()` and `Location()` return its parts.

- A flag is what the source states about the regular session of the event's own market: for a company event, the market its stock trades on. It is kept and never worked out.
- For an exact instant, `TradingCalendar.SessionAt` reports the active session, taking holidays and early closes into account. `TradingCalendar.DayAt` returns the trading day that owns an open session. An overnight session can belong to the following date. [Trading calendars](../calendar)
- A date without a flag says nothing about the time of day. It does not mean during market hours.
- An after-close event keeps its local date. An extended session may still be open after the regular close.
- Pass the ex-date directly to `NewDividend(stock, exDate, amount)` or `NewSplit(stock, exDate, factor)` as a `civil.LocationDate`. Their `Timing()` exposes that date and zone.

## Reading

An `Events` is a collection of events and the dates it covers. Narrowing it gives an `Events` again, so the narrowings combine in any order:

| Narrow | Keeps the events |
|---|---|
| `For(subject)` | about a region or a stock |
| `On(date)` | of a date |
| `During(dates)` | dated in a `civil.DateRange`, including its start and excluding its end |
| `OfKind(kinds...)` | of any of some kinds |

`Has` says whether there is an event of some kinds, `Len` counts the events, `All` lists them by date, and `Of` takes the events of one type out as that type.

```go
// For all of 2021, start is 2021-01-01 and end is 2022-01-01.
year, err := civil.NewDateRange(start, end)
if err != nil {
	return err
}
events, err := marketevent.NewEvents(all, year)
if err != nil {
	return err
}

// Does NVDA report today, and does today have a CPI or an FOMC event?
nvdaReports := events.For(nvda).On(today).Has("earnings")
cpiOrFOMC := events.On(today).Has("CPI", "FOMC")

// The report's flag, when its source gives one.
for report := range events.For(nvda).On(today).Of[marketevent.Earnings]() {
	flag, flagged := report.Timing().Flag()
	afterClose := flagged && flag == marketevent.FlagAfterMarket
}

// The US rate decisions of the year, and how many there are.
decisions := events.For(us).During(year).OfKind("FOMC")
fmt.Println(decisions.Len())

// AAPL's dividends, as Dividend values.
for d := range events.For(aapl).During(year).Of[marketevent.Dividend]() {
	fmt.Println(d.Timing().Date(), d.Amount())
}
```

- Give `NewEvents` the range the event source covers. An empty result on a covered date means there are no events on that date. `events.During(xnys.Coverage())` narrows the events to the dates shared with the exchange calendar. Its `Coverage()` returns that intersection, or the zero range when the ranges are disjoint.
- When migrating a query that previously included its last date, construct the range with `civil.NewDateRange(first, last.AddDays(1))`.
- An `Events` holds a schedule as its events state it, and not when each event became known. It cannot say what a program knew on an earlier date.
- `NewEvents` sorts the slice it is given and keeps it, so that slice may not be changed afterwards.
- `OfKind` makes a new list. `Has` does not, so it is the cheaper way to ask only whether there is an event.
- Code that reads events takes an `Events`, and its tests give it one built from the few events they need.
