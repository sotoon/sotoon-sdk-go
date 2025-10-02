package interceptors

import (
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"github.com/sotoon/sotoon-sdk-go/sdk/constants"
)

type CircuitBreakerInterceptor struct {
	abortOnFailure bool
	cb             *gobreaker.CircuitBreaker
}

var CircuteBreakerForJust429 = gobreaker.NewCircuitBreaker(gobreaker.Settings{
	Name:        "circuteBreakerForJust429",
	MaxRequests: 0,                // unlimited concurrent requests
	Interval:    10 * time.Second, // check status every 10 seconds
	Timeout:     20 * time.Second, // how long to wait before closing circuit after it's opened
	ReadyToTrip: func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures > 0 // instant circute openinig
	},
	IsSuccessful: func(err error) bool {
		return err.Error() != "429"
	},
})

// NewCircuitBreakerInterceptor creates a new circuit breaker interceptor
// faced status codes (including 2xx) are passed as simple string to ReadyToTrip function.
func NewCircuitBreakerInterceptor(cb *gobreaker.CircuitBreaker, abortOnFailure bool) *CircuitBreakerInterceptor {
	if cb == nil {
		panic("cb should not be nil")
	}

	return &CircuitBreakerInterceptor{
		abortOnFailure: abortOnFailure,
		cb:             cb,
	}
}

// BeforeRequest checks if the circuit breaker is open
func (c *CircuitBreakerInterceptor) BeforeRequest(data InterceptorData) (InterceptorData, error) {
	if c.cb.State() == gobreaker.StateOpen {
		if c.abortOnFailure {
			return data, constants.ErrCircuitBreakerOpen
		}
		data.Error = constants.ErrCircuitBreakerOpen
		return data, nil
	}
	return data, nil
}

// AfterResponse records failures in the circuit breaker
func (c *CircuitBreakerInterceptor) AfterResponse(data InterceptorData) (InterceptorData, error) {
	if data.Response != nil {
		c.cb.Execute(func() (interface{}, error) {
			// pass status code to IsSuccessful function
			return nil, fmt.Errorf("%d", data.Response.StatusCode)
		})

		if data.Error != nil {
			if c.abortOnFailure {
				return data, data.Error
			}
			return data, nil
		}
	}

	return data, nil
}
