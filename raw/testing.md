# Testing

> Test your integration offline with the bundled mock — no network, key or money.

Package `github.com/s00d/onlinesim-go-api/v3/mock` is a fake of the OnlineSim API that ships with the module. `mock.Mock` is an `http.RoundTripper` that answers like the real backend — same shapes, same error codes, HTTP 456 for a wrong key, HTTP 200 `ERROR_WRONG_KEY` for none — and keeps a balance, operations and rents. Your code talks to a normal `*onlinesim.Client`, so nothing in it changes for tests. It is deterministic: no randomness, no clock, numbers and ids come from a counter. Standard library only; it is not linked into your binary unless you import it.

## Setup

```go
m := mock.New(mock.WithBalance(10))
client := m.Client() // a *onlinesim.Client wired to the mock, in-process

b, err := client.User().Balance(ctx)
fmt.Println(b.Balance, err) // 10 <nil>
```

```go
func New(opts ...Option) *Mock

func WithAPIKey(key string) Option              // default "mock-key" (mock.DefaultAPIKey)
func WithBalance(usd float64) Option            // default 100
func WithPrices(p Prices) Option                // country → service → USD
func WithRentTariffs(t map[int]RentTariff) Option
func WithGuardInterval(seconds int) Option      // default 0
```

<table>
<thead>
  <tr>
    <th>
      Option
    </th>
    
    <th>
      Default
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
        WithAPIKey
      </code>
    </td>
    
    <td>
      <code>
        "mock-key"
      </code>
    </td>
    
    <td>
      Key the mock accepts. <code>
        m.Client()
      </code>
      
       uses it.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        WithBalance
      </code>
    </td>
    
    <td>
      <code>
        100
      </code>
    </td>
    
    <td>
      Starting balance, USD.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        WithPrices
      </code>
    </td>
    
    <td>
      <code>
        {7: {vkcom: 0.5, telegram: 1.2, google: 0.3}, 44: {google: 0.2, vkcom: 0.36}}
      </code>
    </td>
    
    <td>
      Number prices; replaces the defaults. Also feeds <code>
        Numbers().Tariffs
      </code>
      
      .
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        WithRentTariffs
      </code>
    </td>
    
    <td>
      <code>
        {7: {Days: {1: 1, 3: 2.5, 7: 5}, Extend: 1}}
      </code>
    </td>
    
    <td>
      Rent prices; <code>
        Extend: 0
      </code>
      
       makes rents of that country non-extendable.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        WithGuardInterval
      </code>
    </td>
    
    <td>
      <code>
        0
      </code>
    </td>
    
    <td>
      <code>
        > 0
      </code>
      
      : an operation without SMS, or an unconfirmed rent without any SMS, cannot be closed (<code>
        NO_COMPLETE_TZID
      </code>
      
      ).
    </td>
  </tr>
</tbody>
</table>

`m.Client(opts...)` takes the usual [client options](/#client-options) (`WithLang`, `WithTimeout`, …). To build the client yourself — for example to test a wrong key — pass the mock as the transport; any base URL works:

```go
m := mock.New()
client := onlinesim.New("wrong-key", onlinesim.WithHTTPClient(&http.Client{Transport: m}))
_, err := client.User().Balance(ctx) // *APIError ERROR_WRONG_KEY
fmt.Println(err)
```

Code that needs a real URL (another process, a client you cannot inject an `http.Client` into) can serve the same mock on `127.0.0.1`. `mock.Start` takes the same options and every helper below works on the returned `*mock.Server`:

```go
srv := mock.Start()
defer srv.Close()
client := onlinesim.New(mock.DefaultAPIKey, onlinesim.WithBaseURL(srv.BaseURL())) // or srv.Client()
fmt.Println(client.User().Balance(ctx))
```

## A full example

Order a number, let an SMS arrive while `WaitCode` polls, check the code and the money. `AddSMSAfter` delivers the SMS after a number of polls instead of after some time, so `WaitCode` runs with a no-op `Sleep` and the test takes milliseconds.

```go
// receive_test.go
package myapp

import (
    "context"
    "testing"
    "time"

    onlinesim "github.com/s00d/onlinesim-go-api/v3"
    "github.com/s00d/onlinesim-go-api/v3/mock"
)

func noSleep(context.Context, time.Duration) error { return nil }

func TestReceivesTheCode(t *testing.T) {
    ctx := context.Background()
    m := mock.New(mock.WithBalance(10))
    client := m.Client()

    order, err := client.Numbers().GetWithNumber(ctx, "google", 44)
    if err != nil {
        t.Fatal(err)
    }
    if order.Number != "+44900000001" {
        t.Fatalf("number %s", order.Number)
    }

    if err := m.AddSMSAfter(order.Tzid, "Your Google code: 123-456", 2); err != nil { // the third poll sees it
        t.Fatal(err)
    }
    code, err := client.Numbers().WaitCode(ctx, order.Tzid, onlinesim.WaitCodeOptions{Sleep: noSleep})
    if err != nil {
        t.Fatal(err)
    }

    if code != "123456" {
        t.Fatalf("code %q", code)
    }
    if m.Operations()[0].Status != onlinesim.TZOverOK { // WaitCode closed it
        t.Fatal("operation still open")
    }
    if m.Balance() != 9.8 { // the price is spent once an SMS arrived
        t.Fatalf("balance %v", m.Balance())
    }
}
```

The `msg` of an operation is the code by default: the first run of 4–8 digits, with spaces and dashes between digits removed (`123-456` → `123456`); a text without such a run comes back whole. `StateOptions{FullMessage: true}` (or `FullMessage` in `WaitCodeOptions`) gives the full text.

## Errors

`FailNext(code)` makes the next request answer `{"response": code}` — the client turns it into `*APIError` (or `*NoNumberError`) exactly as for the real server. `mock.OnEndpoint(path)` fails only that endpoint. `FailNextTransport()` makes the next request fail with a non-JSON HTTP 502 (`*TransportError`); `mock.WithStatus(n)` sets another status.

```go
// errors_test.go
package myapp

import (
    "context"
    "errors"
    "testing"

    onlinesim "github.com/s00d/onlinesim-go-api/v3"
    "github.com/s00d/onlinesim-go-api/v3/mock"
)

func TestMissingNumberAndNetworkHiccup(t *testing.T) {
    ctx := context.Background()
    m := mock.New()
    client := m.Client()

    m.FailNext("NO_NUMBER", mock.OnEndpoint("getNum.php"))
    _, err := client.Numbers().Get(ctx, "google", 7)
    var noNumber *onlinesim.NoNumberError
    if !errors.As(err, &noNumber) {
        t.Fatalf("want NoNumberError, got %v", err)
    }

    m.FailNextTransport(mock.WithStatus(503))
    _, err = client.User().Balance(ctx)
    var transport *onlinesim.TransportError
    if !errors.As(err, &transport) {
        t.Fatalf("want TransportError, got %v", err)
    }

    if _, err := client.User().Balance(ctx); err != nil { // one-shot
        t.Fatal(err)
    }
}
```

The state itself produces the real error codes too: `WARNING_LOW_BALANCE` when the balance is short, `NO_NUMBER` when `reject` matches the next number, `UNDEFINED_COUNTRY` / `ERROR_NO_SERVICE` for unknown prices, `ERROR_WRONG_TZID` for an unknown or closed operation (`Next` too; `ERROR_NO_TZID` for tzid `0`), `UNDEFINED_COUNTRY` / `UNDEFINED_DAYS` for an unknown rent country or period, `TRY_AGAIN_LATER` for every refused `Rent().Extend`, `ERROR_NO_OPERATIONS` for an unknown rent in `Rent().Close`, `ERROR_WRONG_KEY` for a wrong key (HTTP 456) or no key (HTTP 200).

## Inspecting requests

`m.Requests()` lists every request received: `mock.Request{Method, Path, Query, Body}`, `Path` relative to `/api/`, `Body` the decoded JSON of a POST.

```go
// requests_test.go
package myapp

import (
    "context"
    "reflect"
    "testing"

    "github.com/s00d/onlinesim-go-api/v3/mock"
)

func TestSendsRejectAsACommaList(t *testing.T) {
    m := mock.New()
    if _, err := m.Client().Numbers().Get(context.Background(), "google", 44, "4470", "4471"); err != nil {
        t.Fatal(err)
    }

    want := []mock.Request{{
        Method: "GET",
        Path:   "getNum.php",
        Query:  map[string]string{"service": "google", "country": "44", "reject": "4470,4471", "lang": "en"},
    }}
    if got := m.Requests(); !reflect.DeepEqual(got, want) {
        t.Fatalf("%+v", got)
    }
}
```

## Control API

<table>
<thead>
  <tr>
    <th>
      Member
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
        Client(opts...)
      </code>
    </td>
    
    <td>
      A normal <code>
        *onlinesim.Client
      </code>
      
       wired to the mock.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        RoundTrip(req)
      </code>
    </td>
    
    <td>
      The mock as an <code>
        http.RoundTripper
      </code>
      
      : <code>
        &http.Client{Transport: m}
      </code>
      
      .
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        AddSMS(tzid, text, opts...) error
      </code>
    </td>
    
    <td>
      An SMS arrives now at a number operation or a rent (detected by <code>
        tzid
      </code>
      
      ). <code>
        mock.FromService(s)
      </code>
      
       sets the sender. Fails for an unknown or closed one.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        AddSMSAfter(tzid, text, polls, opts...) error
      </code>
    </td>
    
    <td>
      The SMS arrives after <code>
        polls
      </code>
      
       state requests for <code>
        tzid
      </code>
      
       (<code>
        Numbers().State
      </code>
      
      /<code>
        StateOne
      </code>
      
      , <code>
        Rent().State
      </code>
      
      /<code>
        StateOne
      </code>
      
      ).
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        FailNext(code, opts...)
      </code>
    </td>
    
    <td>
      The next request (to <code>
        mock.OnEndpoint(path)
      </code>
      
       only, e.g. <code>
        "getNum.php"
      </code>
      
      ) answers <code>
        {"response": code}
      </code>
      
      .
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        FailNextTransport(opts...)
      </code>
    </td>
    
    <td>
      The next request fails with a non-JSON body and HTTP 502 (<code>
        mock.WithStatus(n)
      </code>
      
       for another).
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        SetBalance(usd)
      </code>
      
       / <code>
        Balance()
      </code>
    </td>
    
    <td>
      Set / read the total balance, USD (it includes the part frozen by open operations and unconfirmed rents, as <code>
        User().Balance
      </code>
      
       reports it).
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        Operations()
      </code>
      
       / <code>
        Rents()
      </code>
    </td>
    
    <td>
      Copies of the current number operations / open rents, for assertions.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        Requests()
      </code>
    </td>
    
    <td>
      Log of the requests received.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        Reset()
      </code>
    </td>
    
    <td>
      Back to the initial state.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        mock.Start(opts...)
      </code>
    </td>
    
    <td>
      The same mock on a local HTTP listener: <code>
        BaseURL()
      </code>
      
      , <code>
        Client()
      </code>
      
      , <code>
        Close()
      </code>
      
      .
    </td>
  </tr>
</tbody>
</table>

A `Mock` is safe for concurrent use.

## What the mock does

- **Numbers**: `Get` freezes the price; `Close` after an SMS spends it, `Close` without one refunds it; `Ban` after an SMS answers `"0"`, like the server. Numbers are `+<country>9000000NN`, ids count from 1. `Next` switches to the following SMS. The operation's `time` stays at 1200 s — there is no clock.
- **Rent**: `Get` reserves the price (`Status` `0`, `Checked` `false`) and keeps the `extension` flag; the first SMS that is not `notify` confirms the rent (`Status` `1`) and charges it — `AddSMS(tzid, text, mock.FromService("notify"))` adds a notice with `code` `0` that confirms nothing. `Extend` works on a confirmed rent and charges at once. `Close` releases the reservation of an unconfirmed rent and gives no refund for a confirmed one. `sum` is a number in `Get`/`Extend` and a decimal string on the wire of `State`, like the server. SMS appear in `Messages`, newest first.
- **User**: `Balance` (frozen part included), a fixed profile, `SetWebhookURL` accepts any `scheme://…` URL (`VALIDATION_ERROR` otherwise), `WebhookLogs` lists the deliveries the mock *recorded* when an SMS arrived while a webhook URL was set — it makes no HTTP call; their `params` follow the real payload (`code` `""` without a code, rent `number` in digits without `+`, rent `time_left` in minutes). `PaymentHistory` has no orders.
- **Free**: `Free().List` returns two fixed countries (44 first) with one number and a few SMS each.
- Not imitated: anything that needs real time — the 5 s lock after `setOperationOk` (`INTERVAL_CONCURRENT_REQUESTS_ERROR`), the 10 s lock of `Rent().Extend`, the 120 s before an unconfirmed rent may be closed and its 20-minute wait for the first SMS. `WithGuardInterval(n > 0)` stands in for "too early to close".
