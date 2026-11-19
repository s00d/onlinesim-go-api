// Prints the available balance: ONLINESIM_KEY=... go run ./examples/balance
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
	b, err := client.User().Balance(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("available: %.2f USD\n", b.Balance-b.Zbalance)
}
