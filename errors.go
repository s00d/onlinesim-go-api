package onlinesim

import (
	"errors"
	"fmt"
	"time"
)

// APIError: the API answered with an error code in `response`.
// A wrong key (HTTP 456 with {"response":"ERROR_WRONG_KEY"}) is an APIError too.
type APIError struct {
	// Code is the raw error code, e.g. ERROR_WRONG_KEY.
	Code string
	// Message is the server's text, else a short English description of the code, else the code.
	Message string
}

// newAPIError fills Message from the fallback table when the server sent none or only echoed the code.
func newAPIError(code, message string) *APIError {
	if message == "" || message == code {
		message = errorMessages[code]
	}
	if message == "" {
		message = code
	}
	return &APIError{Code: code, Message: message}
}

func (e *APIError) Error() string {
	if e.Message == "" || e.Message == e.Code {
		return "onlinesim: " + e.Code
	}
	return "onlinesim: " + e.Code + ": " + e.Message
}

// NoNumberError: no number is available right now (NO_NUMBER, NO_NUMBER_FOR_FORWARD).
// Retry later or pick another country. errors.As matches it as *APIError too.
type NoNumberError struct {
	APIError
}

// Unwrap exposes the embedded *APIError to errors.As.
func (e *NoNumberError) Unwrap() error { return &e.APIError }

// TransportError: network failure, request timeout, or a response that is not JSON.
type TransportError struct {
	// Status is the HTTP status, 0 when no response arrived (network error, timeout).
	Status int
	// Body is the start of the response body (up to 500 bytes), empty when none.
	Body string
	// Err is the underlying error (net error, context.DeadlineExceeded, context.Canceled), nil otherwise.
	Err error

	msg string
}

func (e *TransportError) Error() string { return "onlinesim: " + e.msg }

// Unwrap returns the underlying error, so errors.Is(err, context.Canceled) works.
func (e *TransportError) Unwrap() error { return e.Err }

// ErrTimeout is returned (wrapped) by WaitCode when no code arrived within its Timeout.
var ErrTimeout = errors.New("onlinesim: no code within the wait timeout")

func waitTimeout(tzid int64, timeout time.Duration) error {
	return fmt.Errorf("%w: operation %d, %s", ErrTimeout, tzid, timeout)
}

// errorMessages are fallback messages for codes the server sends without a `message`.
var errorMessages = map[string]string{
	"ACCOUNT_BLOCKED":                    "Account is blocked",
	"ERROR_WRONG_KEY":                    "Wrong API key",
	"ERROR_NO_KEY":                       "API key is missing",
	"ERROR_NO_SERVICE":                   "Service is not specified",
	"API_ACCESS_DISABLED":                "API access is disabled in the profile",
	"API_ACCESS_IP":                      "API access from this IP is not allowed in the profile",
	"ERROR_NO_TZID":                      "tzid is not specified",
	"ERROR_WRONG_TZID":                   "Wrong tzid",
	"ERROR_NO_OPERATIONS":                "No operations",
	"NO_NUMBER":                          "No numbers available for this service right now",
	"NO_NUMBER_FOR_FORWARD":              "No numbers available for forwarding",
	"NO_COUNTRY":                         "Country not found",
	"UNDEFINED_COUNTRY":                  "Unknown country",
	"WARNING_LOW_BALANCE":                "Not enough balance",
	"EXCEEDED_CONCURRENT_OPERATIONS":     "Too many concurrent operations for this account",
	"INTERVAL_CONCURRENT_REQUESTS_ERROR": "Too many concurrent requests, try again in a few seconds",
	"TRY_AGAIN_LATER":                    "Temporarily unable to perform the request, try again later",
	"ACCOUNT_IDENTIFICATION_REQUIRED":    "Account identification is required",
	"ERROR_PARAMS":                       "Wrong parameters",
	"API_DISABLED":                       "API is disabled",
	"NO_COMPLETE_TZID":                   "The operation cannot be closed yet",
	"NO_CLOSE_RENT_US":                   "US rent numbers cannot be closed early",
	"UNDEFINED_DAYS":                     "Unknown rent period",
	"VALIDATION_ERROR":                   "Validation failed",
	"TZ_OVER_OK":                         "Operation finished without a new code",
	"ERROR_NO_ITEMS":                     "Operation failed",
}
