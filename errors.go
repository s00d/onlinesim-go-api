package onlinesim

import (
	"errors"
	"fmt"
)

// Sentinel errors.
var (
	ErrTimeout      = errors.New("timeout waiting for SMS code")
	ErrNoOperations = errors.New("no operations")
)

// APIError is returned when OnlineSim responds with a non-success response code.
type APIError struct {
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return e.Code
	}
	return fmt.Sprintf("%s (%s)", e.Message, e.Code)
}

// Temporary reports whether the error is safe to retry after a short delay.
// INTERVAL_CONCURRENT_REQUESTS_ERROR is documented as a frequency limit
// (setOperationOk: min 5s between closes of the same tzid).
func (e *APIError) Temporary() bool {
	return e != nil && e.Code == "INTERVAL_CONCURRENT_REQUESTS_ERROR"
}

// NoNumberError is returned when no numbers are available.
type NoNumberError struct {
	Code string
}

func (e *NoNumberError) Error() string {
	return fmt.Sprintf("no number available: %s", e.Code)
}

// RequestErrorMessage returns a human-readable description for an API error code.
func RequestErrorMessage(code string) string {
	if msg, ok := apiErrorMessages[code]; ok {
		return msg
	}
	return "unknown API error"
}

var apiErrorMessages = map[string]string{
	"ACCOUNT_BLOCKED":                    "account blocked",
	"ERROR_WRONG_KEY":                    "wrong apikey",
	"ERROR_NO_KEY":                       "no apikey",
	"ERROR_NO_SERVICE":                   "service not specified",
	"REQUEST_NOT_FOUND":                  "API method not specified",
	"API_ACCESS_DISABLED":                "api disabled",
	"API_ACCESS_IP":                      "access from this ip is disabled in the profile",
	"WARNING_NO_NUMS":                    "no matching numbers",
	"WARNING_LOW_BALANCE":                "low balance",
	"TZ_INPOOL":                          "waiting for a number to be dedicated to the operation",
	"TZ_NUM_WAIT":                        "waiting for response",
	"TZ_NUM_ANSWER":                      "response has arrived",
	"TZ_OVER_EMPTY":                      "response did not arrive within the specified time",
	"TZ_OVER_OK":                         "operation has been completed",
	"ERROR_NO_TZID":                      "tzid is not specified",
	"ERROR_NO_OPERATIONS":                "no operations",
	"ACCOUNT_IDENTIFICATION_REQUIRED":    "You have to go through an identification process: to order a messenger - in any way, for forward - on the passport.",
	"EXCEEDED_CONCURRENT_OPERATIONS":     "maximum quantity of numbers booked concurrently is exceeded for your account",
	"NO_NUMBER":                          "temporarily no numbers available for the selected service",
	"NO_COUNTRY":                         "country is not available",
	"UNDEFINED_COUNTRY":                  "country is not defined or not available",
	"TIME_INTERVAL_ERROR":                "delayed SMS reception is not possible at this interval of time",
	"INTERVAL_CONCURRENT_REQUESTS_ERROR": "request frequency exceeded (setOperationOk: max once per 5s per tzid)",
	"TRY_AGAIN_LATER":                    "temporarily unable to perform the request",
	"NO_FORWARD_FOR_DEFFER":              "forwarding can be activated only for online reception",
	"NO_NUMBER_FOR_FORWARD":              "there are no numbers for forwarding",
	"ERROR_LENGTH_NUMBER_FOR_FORWARD":    "wrong length of the number for forwarding",
	"DUPLICATE_OPERATION":                "adding operations with identical parameters",
	"ERROR_NO_NUMBER":                    "number is not specified",
	"ERROR_PARAMS":                       "one or both parameters are wrong",
	"LIFICYCLE_NUM_EXPIRED":              "the number has expired",
	"NEED_EXTENSION_NUMBER":              "you have to extend the number, see the Extension tab",
	"ERROR_NUMBERS_PARAMS":               "error in the number format",
	"ERROR_WRONG_TZID":                   "error in the number format",
	"NO_COMPLETE_TZID":                   "unable to complete the operation.",
	"NO_CONFIRM_FORWARD":                 "unable to confirm forwarding",
	"ERROR_NO_SERVICE_REPEAT":            "no services for repeated reception",
	"SERVICE_TO_NUMBER_EMPTY":            "no numbers for repeated reception for this service",
	"repeat disabled":                    "repeated reception is disabled for this account",
}

// ErrorFromAPICode maps an OnlineSim response code to a typed error.
func ErrorFromAPICode(code string) error {
	if code == "NO_NUMBER" || code == "NO_NUMBER_FOR_FORWARD" {
		return &NoNumberError{Code: code}
	}
	if code == "ERROR_NO_OPERATIONS" {
		return ErrNoOperations
	}
	return &APIError{Code: code, Message: RequestErrorMessage(code)}
}
