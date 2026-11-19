---
title: Numbers
description: Order a number, read its SMS, close it — client.Numbers().
navigation:
  order: 1
---

An operation (`tzid`) is one ordered number. Order it with `Get`/`GetWithNumber`, read the SMS with `StateOne` or `WaitCode`, finish it with `Close` (or `Ban`). Countries are phone codes (`7`, `44`, …), services are slugs (`google`, `vkcom`, …) as returned by `Tariffs`.

## Price

Price of one number, USD; `nil` when the country has no price for the service.

```go
func (n *NumbersAPI) Price(ctx context.Context, service string, country int) (*float64, error)
```

| Param | Type | Description |
|---|---|---|
| `service` | `string` | Service slug. |
| `country` | `int` | Country code. |

```go
usd, err := client.Numbers().Price(ctx, "google", 44)
if err == nil && usd != nil {
	fmt.Println(*usd)
}
```

Errors: `*APIError` `ERROR_NO_SERVICE`, `UNDEFINED_COUNTRY`.

## Get

Order a number; returns the operation id. Freezes the price on the balance.

```go
func (n *NumbersAPI) Get(ctx context.Context, service string, country int, reject ...string) (int64, error)
```

| Param | Type | Description |
|---|---|---|
| `service` | `string` | Service slug. |
| `country` | `int` | Country code. |
| `reject` | `...string` | Number prefixes to skip, e.g. `"4470"`. None: the profile's list. |

```go
tzid, err := client.Numbers().Get(ctx, "google", 44, "4470")
fmt.Println(tzid, err)
```

Errors: `*NoNumberError`; `*APIError` `WARNING_LOW_BALANCE`, `ERROR_NO_SERVICE`, `UNDEFINED_COUNTRY`, `EXCEEDED_CONCURRENT_OPERATIONS`, `INTERVAL_CONCURRENT_REQUESTS_ERROR`, `TRY_AGAIN_LATER`.

## GetWithNumber

Same as `Get`, but returns the number right away.

```go
func (n *NumbersAPI) GetWithNumber(ctx context.Context, service string, country int, reject ...string) (NumberOrder, error)
```

Returns `NumberOrder`: `{Tzid, Number, Country, Time, Service, Title, GuardIntervalRemainingSeconds}` — `Number` with `+`, `Time` in seconds left.

```go
order, err := client.Numbers().GetWithNumber(ctx, "google", 44)
fmt.Println(order.Tzid, order.Number, err)
```

Errors: as `Get`.

## State

All operations of the account; an empty slice when there are none.

```go
func (n *NumbersAPI) State(ctx context.Context, opts StateOptions) ([]Operation, error)
```

| `StateOptions` field | Type | Zero value | Description |
|---|---|---|---|
| `FullMessage` | `bool` | `false` | `Msg.Text` is the full SMS text instead of the code. |
| `OrderBy` | `Order` | `OrderAsc` | `OrderAsc` or `OrderDesc`. |
| `MsgList` | `bool` | `false` | `Msg.List` holds every SMS (`[]OperationSMS`). |

`Operation`: `{Tzid, Response, Number, Service, Country, Sum, Time, Form, Msg, GuardIntervalRemainingSeconds, …}`.
`Response` is the state: `TZInPool` (number being assigned), `TZNumWait` (waiting for SMS), `TZNumAnswer` (SMS arrived), `TZOverOK` (finished), `TZErrorNoItems` (failed). `Msg` is empty until the first SMS. The server caches answers for a few seconds.

```go
ops, err := client.Numbers().State(ctx, onlinesim.StateOptions{OrderBy: onlinesim.OrderDesc})
if err != nil {
	return err
}
for _, op := range ops {
	fmt.Println(op.Tzid, op.Response, op.Msg.Text)
}
```

## StateOne

One operation. `OrderBy` is ignored.

```go
func (n *NumbersAPI) StateOne(ctx context.Context, tzid int64, opts StateOptions) (Operation, error)
```

```go
op, err := client.Numbers().StateOne(ctx, tzid, onlinesim.StateOptions{})
if err == nil && op.Response == onlinesim.TZNumAnswer {
	fmt.Println(op.Msg.Text)
}
```

Errors: `*APIError` `ERROR_NO_OPERATIONS` when the operation does not exist (any more).

## WaitCode

Poll the operation until a code arrives, then close it and return the code.

```go
func (n *NumbersAPI) WaitCode(ctx context.Context, tzid int64, opts WaitCodeOptions) (string, error)
```

| `WaitCodeOptions` field | Type | Zero value | Description |
|---|---|---|---|
| `Interval` | `time.Duration` | 5 s | Poll interval. |
| `Timeout` | `time.Duration` | 600 s | Give up after this long. |
| `KeepOpen` | `bool` | `false` | Call `Next(tzid)` instead of `Close(tzid)`, to wait for another SMS. |
| `FullMessage` | `bool` | `false` | Return the full SMS text instead of the code. |
| `Since` | `string` | `""` | Ignore this value (the previous code). |
| `Sleep` | `func(ctx, d) error` | real timer | Injectable for tests. |
| `Now` | `func() time.Time` | `time.Now` | Injectable for tests. |

A transient `*TransportError` is retried until the timeout; any `*APIError` is returned at once, and so is the context's error.

```go
n := client.Numbers()
first, err := n.WaitCode(ctx, tzid, onlinesim.WaitCodeOptions{KeepOpen: true})
if err != nil {
	return err
}
second, err := n.WaitCode(ctx, tzid, onlinesim.WaitCodeOptions{Since: first})
fmt.Println(first, second, err)
```

Errors: `ErrTimeout` (wrapped); `*APIError` with code `TZ_OVER_OK` or `ERROR_NO_ITEMS` when the operation ended without a new code; any other `*APIError` from `StateOne`/`Close`/`Next`.

## Next

Switch the operation to its next SMS and keep it open (for services that send several SMS).

```go
func (n *NumbersAPI) Next(ctx context.Context, tzid int64) error
```

```go
err := client.Numbers().Next(ctx, tzid)
fmt.Println(err)
```

## Close

Finish the operation. An operation without SMS can be closed only after its guard interval (`GuardIntervalRemainingSeconds`); every attempt locks the tzid for 5 s. The client does not retry.

```go
func (n *NumbersAPI) Close(ctx context.Context, tzid int64) error
```

```go
err := client.Numbers().Close(ctx, tzid)
fmt.Println(err)
```

Errors: `*APIError` `NO_COMPLETE_TZID`, `INTERVAL_CONCURRENT_REQUESTS_ERROR`, `ERROR_WRONG_TZID`.

## Ban

Finish the operation and report the number as bad. Only before any SMS arrived.

```go
func (n *NumbersAPI) Ban(ctx context.Context, tzid int64) error
```

```go
err := client.Numbers().Ban(ctx, tzid)
fmt.Println(err)
```

Errors: as `Close`.

## Tariffs

Services, prices and stock of every country, keyed by country code. No API key needed. The answer is large (a few MB); prefer `TariffsOne`.

```go
func (n *NumbersAPI) Tariffs(ctx context.Context) (map[string]CountryStat, error)
```

`CountryStat`: `{Name, LocaleName, Position, Code, Other, New, Enabled, Services}`; `Services` maps `service_<slug>` to `ServiceStat{Count, Popular, Code, Price, ID, Service, Slug}` (`Count` is `-1` when the server reports `"?"`).

```go
all, err := client.Numbers().Tariffs(ctx)
fmt.Println(len(all), "countries", err)
```

## TariffsOne

Services, prices and stock of one country.

```go
func (n *NumbersAPI) TariffsOne(ctx context.Context, country int) (CountryStat, error)
```

```go
uk, err := client.Numbers().TariffsOne(ctx, 44)
if err != nil {
	return err
}
for _, s := range uk.Services {
	fmt.Println(s.Slug, s.Price, s.Count)
}
```

Errors: `*APIError` `NO_COUNTRY`.
