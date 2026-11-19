---
title: onlinesim-go-api
description: Typed, dependency-free Go client for the OnlineSim API — SMS numbers, rent, account and free numbers.
---

## Install

Go 1.22 or newer. Standard library only, no third-party dependencies.

```bash
go get github.com/s00d/onlinesim-go-api/v3
```

Upgrading from v2? See [Migrating from v2](/changelog#migrating-from-v2).

## Quick start

Check the balance, order a number, wait for the SMS code.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

func main() {
	ctx := context.Background()
	client := onlinesim.New(os.Getenv("ONLINESIM_KEY"))

	b, err := client.User().Balance(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("available: %.2f USD\n", b.Balance-b.Zbalance)

	order, err := client.Numbers().GetWithNumber(ctx, "google", 44)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("enter", order.Number, "in the app")

	// Polls every 5 s for up to 10 min, then closes the operation.
	code, err := client.Numbers().WaitCode(ctx, order.Tzid, onlinesim.WaitCodeOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("code:", code)
}
```

## Client options

```go
func New(apiKey string, opts ...Option) *Client
```

```go
client := onlinesim.New("your-api-key",
	onlinesim.WithLang("en"),
	onlinesim.WithTimeout(15*time.Second),
)
fmt.Println(client.Numbers() != nil)
```

| Option | Default | Description |
|---|---|---|
| `apiKey` (argument) | — | Sent as `Authorization: Bearer <apiKey>`, never in the URL. May be `""` for tariffs and free numbers. |
| `WithBaseURL(string)` | `https://onlinesim.host/api/` | API base; trailing slash optional. |
| `WithLang(string)` | `en` | Language of server texts (`lang` query parameter). |
| `WithDevID(int64)` | — | Partner developer id (`dev_id`), sent only when set. |
| `WithTimeout(time.Duration)` | `30 * time.Second` | Per-request timeout; applies to an injected client too. |
| `WithHTTPClient(*http.Client)` | `&http.Client{}` | Transport: a proxy, a custom TLS setup, a rate-limiting `RoundTripper`, or a fake in tests. |

Sections: [`client.Numbers()`](/numbers), [`client.Rent()`](/rent), [`client.User()`](/user), [`client.Free()`](/free). Webhook bodies are parsed by the standalone [`ParseWebhook`](/webhooks). Every method takes a `context.Context` first; the client is safe for concurrent use.

Go has no default arguments: pass every value explicitly (the API's usual choices are country `7`, rent `days` `1`, `extension` `true`, `page` `1`). Option structs (`StateOptions`, `WaitCodeOptions`, `FreeListOptions`) are defaults when zero. Durations are `time.Duration`.

Result structs keep the server's field set with Go names (`guard_interval_remaining_seconds` → `GuardIntervalRemainingSeconds`). Money that the server sends as a decimal string (`balance`, rent `sum`) is a `float64`, maps that arrive as an empty PHP array `[]` are empty maps, and JSON `null` is the zero value.

The client does not throttle or retry. The server answers `INTERVAL_CONCURRENT_REQUESTS_ERROR` to requests that are too frequent (and locks a tzid for 5 s after each `Close`); wait and retry, or wrap the transport with your own limiter.

## Error handling

| Error | When | Fields |
|---|---|---|
| `*APIError` | The API answered with an error code (also a wrong key, HTTP 456). | `Code`, `Message` |
| `*NoNumberError` | `NO_NUMBER` / `NO_NUMBER_FOR_FORWARD`: no number right now. `errors.As` also matches it as `*APIError`. | `Code`, `Message` |
| `*TransportError` | Network error, request timeout, canceled context, or a body that is not JSON. | `Status` (`0` without a response), `Body` (first 500 bytes), `Err` |
| `ErrTimeout` | `WaitCode` got no code within its `Timeout` (match with `errors.Is`). | — |

```go
_, err := client.Numbers().Get(ctx, "google", 44)
var noNum *onlinesim.NoNumberError
var apiErr *onlinesim.APIError
var netErr *onlinesim.TransportError
switch {
case errors.As(err, &noNum):
	// try another country or later
case errors.As(err, &apiErr):
	log.Println(apiErr.Code, apiErr.Message) // e.g. WARNING_LOW_BALANCE
case errors.As(err, &netErr):
	log.Println("network", netErr.Status, netErr.Body)
case err != nil:
	return err
}
```

`Message` is the server's text when it sends one, otherwise a short English description of the code, otherwise the code itself. Profile validation failures become `*APIError` with code `VALIDATION_ERROR`. `*TransportError` unwraps to its cause, so `errors.Is(err, context.Canceled)` works.

## Testing

The bundled mock (`github.com/s00d/onlinesim-go-api/v3/mock`) answers like the real API in-process, so your tests need no network, key or money:

```go
m := mock.New(mock.WithBalance(10))
client := m.Client()
tzid, err := client.Numbers().Get(ctx, "google", 44)
if err != nil {
	return err
}
_ = m.AddSMSAfter(tzid, "Your code: 123-456", 2) // arrives on the third poll
noSleep := func(context.Context, time.Duration) error { return nil }
code, err := client.Numbers().WaitCode(ctx, tzid, onlinesim.WaitCodeOptions{Sleep: noSleep})
fmt.Println(code, err) // 123456 <nil>
```

More in [Testing](/testing).
