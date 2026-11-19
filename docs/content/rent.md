---
title: Rent
description: Rent a number for days and receive every SMS sent to it — client.Rent().
navigation:
  order: 2
---

A rent keeps a number for `days` days. Its SMS come in `RentItem.Messages`, newest first.

`RentItem`: `{Tzid, Status, Number, Country, Sum, Days, Hours, Time, TimeSeconds, Extension, Extend, Messages, StartTime, StopTime, DayExtend, …}` —
`Sum` is the price paid in USD (a `float64`; the server sends a decimal string), `Time` minutes left, `TimeSeconds` seconds left,
`Extension` `0` not extendable / `1` extendable / `2` auto-extension, `Extend` maps days to the extension price (empty when not extendable),
`Messages` is `[]RentSMS`: `{ID, Service, Text, Code, CreatedAt}`.

## Get

Rent a number and charge the balance.

```go
func (r *RentAPI) Get(ctx context.Context, country, days int, extension bool) (RentItem, error)
```

| Param | Type | Description |
|---|---|---|
| `country` | `int` | Country code. |
| `days` | `int` | One of the tariff's `Days` keys. |
| `extension` | `bool` | `true` is the usual choice (the server default); `false` makes the rent non-extendable for good. |

```go
rent, err := client.Rent().Get(ctx, 7, 7, true)
fmt.Println(rent.Tzid, rent.Number, rent.Sum, err)
```

Errors: `*NoNumberError`; `*APIError` `UNDEFINED_DAYS`, `UNDEFINED_COUNTRY`, `WARNING_LOW_BALANCE`, `INTERVAL_CONCURRENT_REQUESTS_ERROR`, `TRY_AGAIN_LATER`.

## State

All rents of the account.

```go
func (r *RentAPI) State(ctx context.Context) ([]RentItem, error)
```

```go
rents, err := client.Rent().State(ctx)
if err != nil {
	return err
}
for _, rent := range rents {
	if len(rent.Messages) > 0 {
		fmt.Println(rent.Number, rent.Messages[0].Code)
	}
}
```

## StateOne

One rent.

```go
func (r *RentAPI) StateOne(ctx context.Context, tzid int64) (RentItem, error)
```

```go
rent, err := client.Rent().StateOne(ctx, tzid)
fmt.Println(len(rent.Messages), err)
```

Errors: `*APIError` `ERROR_NO_OPERATIONS` for an unknown tzid.

## Extend

Extend a rent by `days` days and charge the balance. Only rents created with `extension` `true`.

```go
func (r *RentAPI) Extend(ctx context.Context, tzid int64, days int) (RentItem, error)
```

```go
rent, err := client.Rent().Extend(ctx, tzid, 3)
fmt.Println(rent.StopTime, err)
```

Errors: `*APIError` `TRY_AGAIN_LATER` (not extendable, unknown tzid or period, a repeat within 10 s), `WARNING_LOW_BALANCE`.

## Close

Return the number before the period ends. A rent that is not confirmed yet and has no SMS can be closed only after 120 s; US numbers cannot be closed early.

```go
func (r *RentAPI) Close(ctx context.Context, tzid int64) error
```

```go
err := client.Rent().Close(ctx, tzid)
fmt.Println(err)
```

Errors: `*APIError` `ERROR_NO_OPERATIONS`, `NO_COMPLETE_TZID`, `NO_CLOSE_RENT_US`, `TRY_AGAIN_LATER`.

## Tariffs

Rent prices of every country, keyed by country code. No API key needed.

```go
func (r *RentAPI) Tariffs(ctx context.Context) (map[string]RentTariff, error)
```

`RentTariff`: `{Code, Name, Enabled, New, Position, Extend, Confirm, Days, Count, ExtCounts}` — `Days` maps rent days to the price in USD, `Count` to the numbers available.

```go
tariffs, err := client.Rent().Tariffs(ctx)
fmt.Println(len(tariffs), err)
```

## TariffsOne

Rent prices of one country.

```go
func (r *RentAPI) TariffsOne(ctx context.Context, country int) (RentTariff, error)
```

```go
t, err := client.Rent().TariffsOne(ctx, 7)
fmt.Println(t.Days["7"], err) // price of 7 days
```

Errors: `*APIError` `NO_COUNTRY` for a country without rent.
