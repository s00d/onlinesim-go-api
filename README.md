# onlinesim-go-api

[![CI](https://github.com/s00d/onlinesim-go-api/actions/workflows/ci.yml/badge.svg)](https://github.com/s00d/onlinesim-go-api/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/s00d/onlinesim-go-api/v3.svg)](https://pkg.go.dev/github.com/s00d/onlinesim-go-api/v3)

<p align="center">
  <img src="assets/banner.png" alt="onlinesim-go-api — Typed Go client for OnlineSim SMS" width="100%">
</p>

Typed Go client for the [OnlineSim](https://onlinesim.io) API: SMS numbers, rent, account and free numbers.
Standard library only, Go 1.22+, `context`-first methods, a built-in mock for tests.

```bash
go get github.com/s00d/onlinesim-go-api/v3
```

```go
client := onlinesim.New(os.Getenv("ONLINESIM_KEY"))

b, err := client.User().Balance(ctx)
if err != nil {
	return err
}
fmt.Printf("available: %.2f USD\n", b.Balance-max(b.Zbalance, 0))

order, err := client.Numbers().GetWithNumber(ctx, "google", 44)
if err != nil {
	return err
}
fmt.Println("enter", order.Number, "in the app")
code, err := client.Numbers().WaitCode(ctx, order.Tzid, onlinesim.WaitCodeOptions{})
fmt.Println("code:", code, err)
```

## Testing

Package `mock` is a built-in offline fake of the API: test your integration without network, key or money.
More in the [testing guide](https://s00d.github.io/onlinesim-go-api/testing).

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
	"github.com/s00d/onlinesim-go-api/v3/mock"
)

func main() {
	ctx := context.Background()
	m := mock.New(mock.WithBalance(10))
	client := m.Client()

	tzid, err := client.Numbers().Get(ctx, "google", 44)
	if err != nil {
		log.Fatal(err)
	}
	_ = m.AddSMSAfter(tzid, "Your code: 123-456", 2) // arrives on the third poll
	noSleep := func(context.Context, time.Duration) error { return nil }
	code, err := client.Numbers().WaitCode(ctx, tzid, onlinesim.WaitCodeOptions{Sleep: noSleep})
	if err != nil || code != "123456" || m.Balance() != 9.8 {
		log.Fatalf("code=%q balance=%v err=%v", code, m.Balance(), err)
	}
	fmt.Println("code:", code)
}
```

Documentation: **https://s00d.github.io/onlinesim-go-api/** · Upgrading from v2: [migration table](https://s00d.github.io/onlinesim-go-api/changelog#migrating-from-v2) · Changes: [CHANGELOG.md](CHANGELOG.md) · License: Apache-2.0
