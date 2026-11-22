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

A `Timing` is an exact instant, or a date that may carry a flag: `FlagBeforeMarket` or `FlagAfterMarket`. Either way it has a time zone, and the event's date is its date in that zone.

- A flag is what the source states about the regular session of the event's own market: for a company event, the market its stock trades on. It is kept and never worked out.
- An exact timing has no flag, however early or late its instant. Whether 08:30 in New York comes before the open depends on the market that asks, so compare `Instant` with that market's open and close from [calendar](../calendar).
- A date without a flag says nothing about the time of day. It does not mean during market hours.
- An event after the market closes keeps its date, though trading on it starts on the next trading day.
- A dividend's and a split's timing is the ex-date, a date alone: `NewDividend` and `NewSplit` refuse an exact timing and a date with a flag.

## Reading

An `Events` is a collection of events and the dates it covers. Narrowing it gives an `Events` again, so the narrowings combine in any order:

| Narrow | Keeps the events |
|---|---|
| `For(subject)` | about a region or a stock |
| `On(date)` | of a date |
| `Between(from, to)` | dated in a range, both ends included |
| `OfKind(kinds...)` | of any of some kinds |

`Has` says whether there is an event of some kinds, `Len` counts the events, `All` lists them by date, and `Of` takes the events of one type out as that type.

```go
events, err := marketevent.NewEvents(all, from, to)

// Does NVDA report today, and does today have a CPI or an FOMC event?
nvdaReports := events.For(nvda).On(today).Has("earnings")
cpiOrFOMC := events.On(today).Has("CPI", "FOMC")

// The report's flag, when its source gives one.
for report := range events.For(nvda).On(today).Of[marketevent.Earnings]() {
	flag, flagged := report.Timing().Flag()
	afterClose := flagged && flag == marketevent.FlagAfterMarket
}

// The US rate decisions of the year, and how many there are.
decisions := events.For(us).Between(from, to).OfKind("FOMC")
fmt.Println(decisions.Len())

// AAPL's dividends, as Dividend values.
for d := range events.For(aapl).Between(from, to).Of[marketevent.Dividend]() {
	fmt.Println(d.Timing().Date(), d.Amount())
}
```

- A read outside the dates an `Events` covers finds nothing, which does not mean that nothing happens. Check `Coverage` once, where the `Events` is built. A narrowed `Events` covers only the dates it is narrowed to.
- An `Events` holds a schedule as its events state it, and not when each event became known. It cannot say what a program knew on an earlier date.
- `NewEvents` sorts the slice it is given and keeps it, so that slice may not be changed afterwards.
- `OfKind` makes a new list. `Has` does not, so it is the cheaper way to ask only whether there is an event.
- Code that reads events takes an `Events`, and its tests give it one built from the few events they need.
