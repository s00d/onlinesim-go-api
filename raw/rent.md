# Rent

> Rent a number for days and receive every SMS sent to it — client.Rent().

A rent keeps a number for `days` days. Its SMS come in `RentItem.Messages`, newest first.

`RentItem`: `{Tzid, Status, Number, Country, Sum, Days, Hours, Time, TimeSeconds, Extension, Extend, Messages, StartTime, StopTime, DayExtend, …}` —
`Sum` is the cost of the rent incl. extensions in USD (a `float64`; the server sends a number from `Get`/`Extend` and a decimal string from `State`),
reserved (in `Zbalance`) while `Status` is `0` and charged once confirmed, `Time` minutes left, `TimeSeconds` seconds left,
`Extension` `0` not extendable / `1` extendable / `2` auto-extension, `Extend` maps days to the extension price (empty when not extendable),
`Messages` is `[]RentSMS`: `{ID, Service, Text, Code, CreatedAt}`. `Code` is `""` without a code; an SMS from `Service` `"notify"`
(a notice about a prohibited sender) comes with the integer `0`, which the SDK turns into `"0"`, and confirms nothing.

## Get

Rent a number. The price is reserved (frozen in `Zbalance`), not charged: the rent starts with `Status` `0`, `Checked` `false`
and waits up to 20 minutes for its first SMS. The first SMS that is not `notify` confirms it (`Status` `1`) and charges the price.

```go
func (r *RentAPI) Get(ctx context.Context, country, days int, extension bool) (RentItem, error)
```

<table>
<thead>
  <tr>
    <th>
      Param
    </th>
    
    <th>
      Type
    </th>
    
    <th>
      Description
    </th>
  </tr>
</thead>

<tbody>
  <tr>
    <td>
      <code>
        country
      </code>
    </td>
    
    <td>
      <code>
        int
      </code>
    </td>
    
    <td>
      Country code.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        days
      </code>
    </td>
    
    <td>
      <code>
        int
      </code>
    </td>
    
    <td>
      One of the tariff's <code>
        Days
      </code>
      
       keys.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        extension
      </code>
    </td>
    
    <td>
      <code>
        bool
      </code>
    </td>
    
    <td>
      <code>
        true
      </code>
      
       is the usual choice (the server default); <code>
        false
      </code>
      
       makes the rent non-extendable for good.
    </td>
  </tr>
</tbody>
</table>

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

Extend a rent by `days` days and charge the balance at once. Only confirmed rents (`Status` `1`) created with `extension` `true`.

```go
func (r *RentAPI) Extend(ctx context.Context, tzid int64, days int) (RentItem, error)
```

```go
rent, err := client.Rent().Extend(ctx, tzid, 3)
fmt.Println(rent.StopTime, err)
```

Errors: `*APIError` `TRY_AGAIN_LATER` for every refusal (unknown tzid, rent not confirmed yet, not extendable, unknown period, a repeat within 10 s), `WARNING_LOW_BALANCE`.

## Close

Return the number before the period ends. A confirmed rent closes without a refund. A rent that is not confirmed yet (`Status` `0`) and has no SMS
can be closed only after 120 s; then, with no SMS or only `notify` ones, the reservation is released. US numbers cannot be closed early.

```go
func (r *RentAPI) Close(ctx context.Context, tzid int64) error
```

```go
err := client.Rent().Close(ctx, tzid)
fmt.Println(err)
```

Errors: `*APIError` `ERROR_NO_OPERATIONS` (unknown tzid), `NO_COMPLETE_TZID`, `NO_CLOSE_RENT_US`, `TRY_AGAIN_LATER`.

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
