# calendar

Trading calendars: for every date a market covers, whether it trades and in which sessions. A date is a business day with its sessions, a holiday or a weekend. Queries answer which session is open at an instant, the next and previous sessions, how long trading continues, the trading day of an instant, and trading-day offsets and ranges. See the package documentation for the API.

## Built-in calendars

| Exchange | Covers | Sources |
|---|---|---|
| XNYS | `[2000-01-01,2031-01-01)` | 2029–2030 are projected. 2026–2028 follow [NYSE holidays and trading hours](https://www.nyse.com/markets/hours-calendars). Earlier years come from the XNYS calendar of [exchange_calendars](https://github.com/gerrymanoim/exchange_calendars). |

The XNYS projections observe Saturday holidays on the preceding Friday, except for New Year's Day, and Sunday holidays on the following Monday. They use a 13:00 close on the Friday after Thanksgiving and on July 3 and December 24 when those dates fall Monday through Thursday. New holidays and unscheduled closures can change a projection.

```go
calendars, err := calendar.TradingCalendars()
registry, err := calendar.NewRegistry(calendars)
id, err := calendar.NewID("XNYS")
xnys, err := registry.Get(id)
session, open, err := xnys.SessionAt(time.Now())
dates := xnys.TradingDaysBetween(xnys.Coverage())
```

Calendar coverage and range queries use `civil.DateRange`. Its start is inclusive and its end is exclusive, so the last date covered for XNYS is December 31, 2030. `TradingDaysBetween` returns an empty slice for a missing range, a range that extends outside coverage, or a range with no trading days. Single-date and instant queries report `ErrOutOfCoverage` when they need data beyond coverage.

## Data

Each built-in calendar is a JSON file in `data`, named after its exchange's code, such as `xnys.json`:

```json
{
  "exchange": "XNYS",
  "coverage": {"start": "2000-01-01", "end": "2031-01-01"},
  "exceptions": [
    {"date": "2026-11-26", "note": "Thanksgiving Day", "sessions": []},
    {"date": "2026-11-27", "note": "Day after Thanksgiving", "sessions": [{"open": "09:30", "close": "13:00"}]}
  ]
}
```

- A date that is not listed follows the exchange's regular hours from [github.com/torhx/exchange](https://github.com/torhx/exchange).
- A listed date has exactly the sessions it lists, in time order. With none, it is closed.
- Times are wall-clock times in the exchange's time zone. A close at or before its open is on the next date.
- Late opens and early closes are not marked: the calendar finds them by comparing a date's sessions with the regular hours.
- `coverage.end` is the first date beyond the published or projected calendar.
- Entries stay in date order, one per line, with a note that names the reason.

Loading rejects unknown calendar, exception, and session fields, missing or invalid coverage, duplicate exception dates, and exceptions outside coverage.

## Updating

- When an exchange publishes a new year, add its holidays and early closes and move `coverage.end` to January 1 of the following year. For XNYS, also add the year to `TestBuiltInXNYSTradingDaysPerYear` in `loader_test.go`.
- When it publishes a projected year, correct any date that differs and move the year to the published sources in the table.
- An unscheduled closure, such as a national day of mourning, is added by hand when it is announced.
- To add an exchange, add its file. Its code must be one the exchange package knows; its regular hours come from there.
