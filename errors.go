package onlinesim

import (
	"fmt"
)

type HTTPClientError struct {
	StatusCode int
	Err        error
}

func (e *HTTPClientError) Error() string {
	return fmt.Sprintf("status %d, err: %v", e.StatusCode, e.Err)
}