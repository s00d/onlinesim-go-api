package main

import (
	"context"
	"fmt"
	"os"

	onlinesim "github.com/s00d/onlinesim-go-api/v2"
)

func main() {
	apiKey := os.Getenv("ONLINESIM_APIKEY")
	if apiKey == "" {
		apiKey = "your-apikey"
	}
	client := onlinesim.New(apiKey)
	bal, err := client.User().Balance(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Printf("balance = %v\n", bal.Balance)
}
