---
title: Testing
description: Test your integration offline with the bundled mock — no network, key or money.
navigation:
  order: 6
---

Package `github.com/s00d/onlinesim-go-api/v3/mock` is a fake of the OnlineSim API that ships with the module. `mock.Mock` is an `http.RoundTripper` that answers like the real backend — same shapes, same error codes, HTTP 456 for a wrong key — and keeps a balance, operations and rents. Your code talks to a normal `*onlinesim.Client`, so nothing in it changes for tests. It is deterministic: no randomness, no clock, numbers and ids come from a counter. Standard library only; it is not linked into your binary unless you import it.

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

| Option | Default | Description |
|---|---|---|
| `WithAPIKey` | `"mock-key"` | Key the mock accepts. `m.Client()` uses it. |
| `WithBalance` | `100` | Starting balance, USD. |
| `WithPrices` | `{7: {vkcom: 0.5, telegram: 1.2, google: 0.3}, 44: {google: 0.2, vkcom: 0.36}}` | Number prices; replaces the defaults. Also feeds `Numbers().Tariffs`. |
| `WithRentTariffs` | `{7: {Days: {1: 1, 3: 2.5, 7: 5}, Extend: 1}}` | Rent prices; `Extend: 0` makes rents of that country non-extendable. |
| `WithGuardInterval` | `0` | `> 0`: an operation without SMS cannot be closed (`NO_COMPLETE_TZID`). |

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

The state itself produces the real error codes too: `WARNING_LOW_BALANCE` when the balance is short, `NO_NUMBER` when `reject` matches the next number, `UNDEFINED_COUNTRY` / `ERROR_NO_SERVICE` for unknown prices, `ERROR_WRONG_TZID` for an unknown or closed operation, `TRY_AGAIN_LATER` when extending a non-extendable rent, `ERROR_WRONG_KEY` (HTTP 456) for a wrong key.

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

| Member | Description |
|---|---|
| `Client(opts...)` | A normal `*onlinesim.Client` wired to the mock. |
| `RoundTrip(req)` | The mock as an `http.RoundTripper`: `&http.Client{Transport: m}`. |
| `AddSMS(tzid, text, opts...) error` | An SMS arrives now at a number operation or a rent (detected by `tzid`). `mock.FromService(s)` sets the sender. Fails for an unknown or closed one. |
| `AddSMSAfter(tzid, text, polls, opts...) error` | The SMS arrives after `polls` state requests for `tzid` (`Numbers().State`/`StateOne`, `Rent().State`/`StateOne`). |
| `FailNext(code, opts...)` | The next request (to `mock.OnEndpoint(path)` only, e.g. `"getNum.php"`) answers `{"response": code}`. |
| `FailNextTransport(opts...)` | The next request fails with a non-JSON body and HTTP 502 (`mock.WithStatus(n)` for another). |
| `SetBalance(usd)` / `Balance()` | Set / read the total balance, USD (it includes the part frozen by open operations, as `User().Balance` reports it). |
| `Operations()` / `Rents()` | Copies of the current number operations / open rents, for assertions. |
| `Requests()` | Log of the requests received. |
| `Reset()` | Back to the initial state. |
| `mock.Start(opts...)` | The same mock on a local HTTP listener: `BaseURL()`, `Client()`, `Close()`. |

A `Mock` is safe for concurrent use.

## What the mock does

- **Numbers**: `Get` freezes the price; `Close` after an SMS spends it, `Close` without one refunds it; `Ban` after an SMS answers `"0"`, like the server. Numbers are `+<country>9000000NN`, ids count from 1. `Next` switches to the following SMS. The operation's `time` stays at 1200 s — there is no clock.
- **Rent**: `Get` charges at once and keeps the `extension` flag; `Extend` charges again; `Close` removes the rent (no refund). SMS appear in `Messages`, newest first.
- **User**: `Balance` (frozen part included), a fixed profile, `SetWebhookURL` validates `http(s)` URLs (`VALIDATION_ERROR` otherwise), `WebhookLogs` lists the deliveries the mock *recorded* when an SMS arrived while a webhook URL was set — it makes no HTTP call. `PaymentHistory` is empty.
- **Free**: `Free().List` returns two fixed countries with one number and a few SMS each.
- Not imitated: the server's 5 s lock after `setOperationOk` (`INTERVAL_CONCURRENT_REQUESTS_ERROR`) and anything else that needs real time.
