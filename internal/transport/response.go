package transport

import (
	"encoding/json"
	"fmt"
)

// ResponseChecker maps API response codes to errors.
type ResponseChecker func(code string) error

// CheckResponseField inspects a top-level "response" field in raw JSON.
// Success is response missing, empty, or equal to "1" / 1.
// When expectArray is true and the body is a JSON array, it is treated as success.
func CheckResponseField(body []byte, toErr ResponseChecker) error {
	if len(body) == 0 {
		return nil
	}
	trimmed := skipWS(body)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		return nil
	}

	var probe struct {
		Response any `json:"response"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return nil // let caller decode the real type
	}
	if probe.Response == nil {
		return nil
	}
	code := fmt.Sprint(probe.Response)
	if code == "" || code == "1" || code == "<nil>" {
		return nil
	}
	if toErr != nil {
		return toErr(code)
	}
	return fmt.Errorf("api error: %s", code)
}

func skipWS(b []byte) []byte {
	i := 0
	for i < len(b) && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
		i++
	}
	return b[i:]
}
