// Package onlinesim is a typed, dependency-free Go client for the OnlineSim API
// (https://onlinesim.io): SMS numbers, rent, account and free numbers.
//
//	client := onlinesim.New(os.Getenv("ONLINESIM_KEY"))
//	order, err := client.Numbers().GetWithNumber(ctx, "google", 44)
//	code, err := client.Numbers().WaitCode(ctx, order.Tzid, onlinesim.WaitCodeOptions{})
//
// Errors are *APIError (with *NoNumberError for NO_NUMBER), *TransportError and ErrTimeout.
// A local HTTP mock for tests lives in github.com/s00d/onlinesim-go-api/v3/mock.
//
// Documentation: https://s00d.github.io/onlinesim-go-api/
package onlinesim
