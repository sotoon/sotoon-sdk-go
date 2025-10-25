package interceptors

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/sotoon/sotoon-sdk-go/sdk/constants"

	"github.com/patrickmn/go-cache"
)

type Transporter interface {
	RoundTripWithID(req *http.Request, id string) (*http.Response, error)
}

type BackoffTimer interface {
	TimeToWait(iteration int) time.Duration
}

type RetryDecider interface {
	// ShouldRetry determines whether a failed HTTP request should be retried.
	// It receives the HTTP response (if any), the error (if any), and retry metadata.
	// Returns:
	//   - bool: true if the request should be retried, false otherwise
	//   - error: non-nil if an error occurred during the decision process. returning error stops the whole interceptorsChain.
	ShouldRetry(*http.Response, error, RetryInternalData) (bool, error)
}

type RetryInternalData struct {
	RetryCount int
}

type RetryInterceptor struct {
	cache           *cache.Cache
	transporter     Transporter
	backoffStrategy BackoffTimer
	retryDecider    RetryDecider
}

func NewRetryInterceptor(transporter Transporter, backoffStrategy BackoffTimer, retryDecider RetryDecider) *RetryInterceptor {
	return &RetryInterceptor{
		cache:           cache.New(time.Minute, time.Minute*15),
		transporter:     transporter,
		backoffStrategy: backoffStrategy,
		retryDecider:    retryDecider,
	}
}

func (e *RetryInterceptor) BeforeRequest(data InterceptorData) (InterceptorData, error) {
	if data.Error != nil {

		d := e.getRetryInternalData(data)
		if shouldRetry, err := e.retryDecider.ShouldRetry(data.Response, data.Error, d); !shouldRetry {
			if err != nil {
				panic(err)
			}
			return data, data.Error
		}

		time.Sleep(e.backoffStrategy.TimeToWait(d.RetryCount))

		response, err := e.transporter.RoundTripWithID(data.Request, data.ID)
		if err != nil || response.StatusCode >= 400 {
			if err != nil {
				data.Error = err
			}
			return e.BeforeRequest(data)
		}
		data.Error = nil
		data.Response = response
	}
	return data, nil
}

func (e *RetryInterceptor) AfterResponse(data InterceptorData) (InterceptorData, error) {

	d := e.getRetryInternalData(data)
	shouldRetry, err := e.retryDecider.ShouldRetry(data.Response, data.Error, d)

	if err != nil {
		return data, err
	}
	if !shouldRetry {
		return data, data.Error
	}

	time.Sleep(e.backoffStrategy.TimeToWait(d.RetryCount))

	response, err := e.transporter.RoundTripWithID(data.InitialRequest, data.ID)
	data.Response = response
	if err != nil || (response != nil && response.StatusCode >= 400) {
		if err != nil {
			data.Error = err
		}
		return e.AfterResponse(data)
	}
	data.Error = nil

	return data, nil
}

func (e *RetryInterceptor) getRetryInternalData(data InterceptorData) RetryInternalData {
	internalData, found := e.cache.Get(data.ID)
	var d RetryInternalData
	if found {
		d = internalData.(RetryInternalData)
		d.RetryCount++
		e.cache.Set(data.ID, d, cache.DefaultExpiration)
	} else {
		d = RetryInternalData{RetryCount: 1}
		e.cache.Set(data.ID, d, cache.DefaultExpiration)
	}
	return d
}

/////////////////////////////////////////

type BackoffStrategyExpnential struct {
	baseDuration time.Duration
	maxBackoff   time.Duration
}

func NewRetryInterceptor_ExponentialBackoff(baseDuration, maxBackoff time.Duration) BackoffStrategyExpnential {
	return BackoffStrategyExpnential{
		baseDuration: baseDuration,
		maxBackoff:   maxBackoff,
	}
}

func (b BackoffStrategyExpnential) TimeToWait(iteration int) time.Duration {
	return exponentialBackoff(iteration, b.baseDuration, b.maxBackoff)
}

// exponentialBackoff calculates the backoff duration with jitter for retries
func exponentialBackoff(retry int, initialBackoff, maxBackoff time.Duration) time.Duration {
	if retry <= 0 {
		return initialBackoff
	}

	backoff := initialBackoff * time.Duration(1<<uint(retry))

	jitter := time.Duration(rand.Int63n(int64(backoff) / 2))
	backoff = backoff + jitter

	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	return backoff
}

type BackoffStrategyLinier struct {
	baseDuration time.Duration
}

func NewRetryInterceptor_BackoffStrategyLinier(baseDuration time.Duration) BackoffStrategyExpnential {
	return BackoffStrategyExpnential{
		baseDuration: baseDuration,
	}
}

func (b BackoffStrategyLinier) TimeToWait(iteration int) time.Duration {
	return b.baseDuration
}

/////////////////////////////////////

// RetryDeciderAll retries all requests that fail
type RetryDeciderAll struct {
	maxRetries int
}

func NewRetryInterceptor_RetryDeciderAll(maxRetries int) RetryDeciderAll {
	return RetryDeciderAll{
		maxRetries: maxRetries,
	}
}

func (r RetryDeciderAll) ShouldRetry(response *http.Response, err error, retryData RetryInternalData) (bool, error) {

	if retryData.RetryCount >= r.maxRetries {
		// Return the actual error that caused the failure, not a generic "max retries exceeded" error
		if err != nil {
			return false, err
		}
		// If no error but we have a bad response, return a generic error
		// (though this shouldn't happen if TreatAsErrorInterceptor runs before RetryInterceptor)
		return false, constants.ErrMaxRetriesExceeded
	}

	if err != nil || (response != nil && response.StatusCode >= 400) {
		return true, nil
	}

	return false, nil
}
