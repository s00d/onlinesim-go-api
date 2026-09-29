package main

import (
	"context"
	"fmt"
	"os"
	"time"

	onlinesim "github.com/s00d/onlinesim-go-api/v2"
)

func main() {
	apiKey := os.Getenv("ONLINESIM_APIKEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "set ONLINESIM_APIKEY")
		os.Exit(1)
	}
	client := onlinesim.New(apiKey)
	ctx := context.Background()

	tzid, err := client.Numbers().Get(ctx, onlinesim.GetNumberParams{Service: "telegram"})
	if err != nil {
		panic(err)
	}
	fmt.Println("tzid", tzid)

	code, err := client.Numbers().WaitCode(ctx, tzid, onlinesim.WaitCodeOptions{
		Interval:    5 * time.Second,
		MaxAttempts: 24,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("code", code)
}
