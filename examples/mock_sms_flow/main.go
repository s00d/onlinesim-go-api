// The order → code → close flow on the local mock, offline: go run ./examples/mock_sms_flow
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
	m := mock.Start()
	defer m.Close()
	m.ScriptSMS(mock.SmsScript{Service: "telegram", Number: "+19001234567", Code: "123456", PollsBeforeCode: 1})

	client := m.Client()
	ctx := context.Background()

	order, err := client.Numbers().GetWithNumber(ctx, "telegram", 1)
	if err != nil {
		log.Fatal(err)
	}
	code, err := client.Numbers().WaitCode(ctx, order.Tzid, onlinesim.WaitCodeOptions{Interval: 100 * time.Millisecond})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("number=%s code=%s\n", order.Number, code)
}
