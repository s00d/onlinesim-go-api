---
title: Changelog
description: Release notes of onlinesim-go-api and the v2 → v3 migration.
navigation:
  order: 6
---

The full history is in [CHANGELOG.md](https://github.com/s00d/onlinesim-go-api/blob/master/CHANGELOG.md).

## 3.0.0

Module path `github.com/s00d/onlinesim-go-api/v3`. The API follows the shared OnlineSim SDK contract; v2 code does not compile unchanged.

- The key is sent as `Authorization: Bearer <key>`, never in the URL.
- Typed errors: `*APIError`, `*NoNumberError`, `*TransportError`, `ErrTimeout`; a wrong key (HTTP 456) is an `*APIError`.
- `WaitCode` polls every 5 s for up to 10 min, stops at once on a finished operation, retries network errors, takes an injectable `Sleep`/`Now`.
- No client-side pacing or retries; `INTERVAL_CONCURRENT_REQUESTS_ERROR` reaches the caller.
- New: `Free().List` options, `ParseWebhook`, `AddRentSMS` in the mock; the mock checks paths and the key.
- Removed: `Numbers().Repeat` / `Service` / `ServiceNumber`, `Rent().PortReload`, `User().CreateEmpty`, `Free().Countries` / `Numbers` / `Messages`.

## Migrating from v2

Replace the import path, then:

| v2 | v3 |
|---|---|
| `github.com/s00d/onlinesim-go-api/v2` | `github.com/s00d/onlinesim-go-api/v3` |
| `Numbers().Get(ctx, GetNumberParams{Service, Country, Reject})` | `Numbers().Get(ctx, service, country, reject...)` (`reject` are strings) |
| `GetWithNumber(ctx, GetNumberParams{…})` → `NumberWithTz` | `GetWithNumber(ctx, service, country, reject...)` → `NumberOrder` |
| `State(ctx, StateParams{MessageToCode: IntPtr(0), OrderBy: "ASC", …})` | `State(ctx, StateOptions{FullMessage: true, OrderBy: OrderAsc})` |
| `StateOne` type, `st.MsgString()` | `Operation`, `op.Msg.Text` (or `op.Msg.List` with `MsgList`) |
| `WaitCode(ctx, tzid, WaitCodeOptions{Interval, MaxAttempts, NotEnd})`, `DefaultWaitCodeOptions()` | `WaitCode(ctx, tzid, WaitCodeOptions{Interval, Timeout, KeepOpen, Since})`; zero = 5 s / 600 s |
| `ErrNoOperations` | `State` returns an empty slice; `StateOne` returns `*APIError` `ERROR_NO_OPERATIONS` |
| `TariffCountryOne`, `TariffService` | `CountryStat`, `ServiceStat` |
| `Rent().Get(ctx, GetRentParams{Country, Days, Extension: BoolPtr(false)})` | `Rent().Get(ctx, country, days, extension)` |
| `Rent().Extend(ctx, tzid, 0)` (defaulted to 1) | `Rent().Extend(ctx, tzid, days)`; pass the days |
| `RentItem.Sum`, `Extend`, `DayExtend` as `any` | `float64`, `map[string]float64`, `float64` |
| `User().Profile` → `User` | → `Profile` |
| `User().PaymentHistory(ctx)` → `PayList` | `User().PaymentHistory(ctx, page)` → `PaymentHistory` |
| `User().SetWebhookURL(ctx, &url)` / `(ctx, nil)` | `SetWebhookURL(ctx, url)` / `ClearWebhookURL(ctx)` (sends `null`) |
| `User().WebhookLogs(ctx, int64)` → `WebhookLogsPage` | `WebhookLogs(ctx, int)` → `Paginator[WebhookLog]` |
| `Free().FreeList(ctx)` | `Free().List(ctx, FreeListOptions{})` |
| `ParseWebhookJSON(b)` | `ParseWebhook(b)`; no `{"data":…}` wrapper, missing fields are zero values, not errors |
| `NoNumberError{Code}` | `*NoNumberError` embeds `APIError`; `errors.As` matches both |
| HTTP errors as plain `error` | `*TransportError` with `Status`, `Body`, `Err` |
| `WithOAuth`, `WithUserAgent`, `WithRateLimit` | removed (Bearer key; wrap the transport for throttling) |
| `WithDevID`, `WithLang`, `WithBaseURL`, `WithHTTPClient`, `WithTimeout` | same; the timeout now applies per request, also to an injected client |
| `DefaultCountry`, country `0` → default | removed: pass the country (`7`, `44`, …) |
| `RequestErrorMessage`, `ErrorFromAPICode`, `APIError.Temporary` | removed; `APIError.Message` carries the text |
| `mock.SmsScript.Price` (string) | `float64`; `mock.APIKey`; `BaseURL()` ends with `/api/` |
