package constants

import "errors"

var (
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	ErrCircuitBreakerOpen    = errors.New("circuit breaker is open")
)
