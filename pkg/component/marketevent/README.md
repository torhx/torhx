# marketevent

Scheduled market events: economic releases, company events and a program's own. Every event says when it happens, what kind it is and what it is about, and a reader finds events by date and by subject. See the package documentation for the API.

## Events

| Type | What it is | Kind |
|---|---|---|
| `EconomicEvent` | An economic event about a region, such as a release of the US CPI. | The name its source's normalizer gives it, such as `CPI`, with an optional detail. |
| `Earnings` | An earnings report on a stock. | `earnings` |
| `Dividend` | A cash dividend with its amount per share, dated by its ex-date. | `dividend` |
| `Split` | A split with the factor that earlier prices are multiplied by, dated by its ex-date. | `split` |
| A program's own | A type that embeds `EconomicEvent` or `CompanyEvent` and adds fields. | Its own. |

An event's subject is a region or a stock. A region is a country, or a currency zone named by its currency, as `EUR` names the euro area. A stock is named by its composite FIGI.

## Timing

A `Timing` is an exact instant, or a date that may carry a flag: `FlagBeforeMarket` or `FlagAfterMarket`. Either way it has a time zone, and the event's date is its date in that zone.

- A flag is what the source states about the regular session of the event's own market: for a company event, the market its stock trades on. It is kept and never worked out.
- An exact timing has no flag, however early or late its instant. Whether 08:30 in New York comes before the open depends on the market that asks, so compare `Instant` with that market's open and close from [calendar](../calendar).
- A date without a flag says nothing about the time of day. It does not mean during market hours.
- An event after the market closes keeps its date, though trading on it starts on the next trading day.
- A dividend's and a split's timing is the ex-date, a date alone: `NewDividend` and `NewSplit` refuse an exact timing and a date with a flag.

## Reading

```go
events, err := marketevent.NewEvents(all, from, to)

// Does NVDA report today, and does today have a CPI or an FOMC event?
nvdaReports := events.For(nvda).Has(today, "earnings")
cpiOrFOMC := events.Has(today, "CPI", "FOMC")

// The report's flag, when its source gives one.
for report := range events.For(nvda).Find(today, "earnings").Of[marketevent.Earnings]() {
	flag, flagged := report.Timing().Flag()
	afterClose := flagged && flag == marketevent.FlagAfterMarket
}

// AAPL's dividends, as Dividend values.
for d := range events.For(aapl).Between(from, to).Of[marketevent.Dividend]() {
	fmt.Println(d.Timing().Date(), d.Amount())
}
```

- A read outside the dates an `Events` covers finds nothing, which does not mean that nothing happens. Check `Coverage` once, where the `Events` is built.
- An `Events` holds a schedule as its events state it, and not when each event became known. It cannot say what a program knew on an earlier date.
- `NewEvents` sorts the slice it is given and keeps it. Neither that slice nor a `Selection` may be changed afterwards.
- Code that reads events takes a `Reader`, so its tests can give it an `Events` of a few events or a `Reader` of their own.
