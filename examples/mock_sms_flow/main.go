package main

import (
	"context"
	"fmt"

	onlinesim "github.com/s00d/onlinesim-go-api/v2"
	"github.com/s00d/onlinesim-go-api/v2/mock"
)

func main() {
	m := mock.Start()
	defer m.Close()
	m.ScriptSMS(mock.SmsScript{
		Service:         "telegram",
		Number:          "+19001234567",
		Code:            "123456",
		PollsBeforeCode: 1,
	})

	client := m.Client()
	ctx := context.Background()

	ordered, err := client.Numbers().GetWithNumber(ctx, onlinesim.GetNumberParams{Service: "telegram"})
	if err != nil {
		panic(err)
	}
	code, err := client.Numbers().WaitCode(ctx, ordered.Tzid, onlinesim.WaitCodeOptions{
		Interval:    0,
		MaxAttempts: 5,
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("number=%s code=%s\n", ordered.Number, code)
}
