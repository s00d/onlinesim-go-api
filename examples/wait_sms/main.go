// Orders a number and waits for its SMS code (paid): ONLINESIM_KEY=... go run ./examples/wait_sms
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

func main() {
	client := onlinesim.New(os.Getenv("ONLINESIM_KEY"))
	ctx := context.Background()

	order, err := client.Numbers().GetWithNumber(ctx, "telegram", 44)
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
