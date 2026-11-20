# calendar

Trading calendars: for every date a market covers, whether it trades and in which sessions. A date is a business day with its sessions, a holiday or a weekend. Queries answer which session is open at an instant, the next and previous sessions, how long trading continues, the trading day of an instant, and trading-day offsets and ranges. See the package documentation for the API.

## Built-in calendars

| Exchange | Covers | Sources |
|---|---|---|
| XNYS | 2000-01-01 to 2028-12-31 | 2026-2028: [NYSE holidays and trading hours](https://www.nyse.com/markets/hours-calendars). Earlier years: the XNYS calendar of [exchange_calendars](https://github.com/gerrymanoim/exchange_calendars). |

```go
calendars, err := calendar.TradingCalendars()
registry, err := calendar.NewRegistry(calendars)
id, err := calendar.NewID("XNYS")
xnys, err := registry.Get(id)
session, open, err := xnys.SessionAt(time.Now())
```

A query outside a calendar's dates returns `ErrOutOfCoverage`.

## Data

Each built-in calendar is a JSON file in `data`, named after its exchange's code, such as `xnys.json`:

```json
{
  "exchange": "XNYS",
  "from": "2000-01-01",
  "to": "2028-12-31",
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
- `to` is the last date the exchange has published holidays for. The calendar knows nothing beyond it.
- Entries stay in date order, one per line, with a note that names the reason.

Loading rejects an unknown key, a date listed twice and a date outside `from` to `to`.

## Updating

- When an exchange publishes a new year, add its holidays and early closes and move `to` to the year's last day. For XNYS, also add the year to `TestBuiltInXNYSTradingDaysPerYear` in `loader_test.go`.
- An unscheduled closure, such as a national day of mourning, is added by hand when it is announced.
- To add an exchange, add its file. Its code must be one the exchange package knows; its regular hours come from there.
