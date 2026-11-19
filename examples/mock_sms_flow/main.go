// The order → code → close flow on the in-process mock, offline: go run ./examples/mock_sms_flow
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
	m := mock.New(mock.WithBalance(10))
	client := m.Client()
	ctx := context.Background()

	order, err := client.Numbers().GetWithNumber(ctx, "google", 44)
	if err != nil {
		log.Fatal(err)
	}
	if err := m.AddSMSAfter(order.Tzid, "Your Google code: 123-456", 2); err != nil {
		log.Fatal(err)
	}
	noSleep := func(context.Context, time.Duration) error { return nil }
	code, err := client.Numbers().WaitCode(ctx, order.Tzid, onlinesim.WaitCodeOptions{Sleep: noSleep})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("number=%s code=%s balance=%.2f\n", order.Number, code, m.Balance())
}
