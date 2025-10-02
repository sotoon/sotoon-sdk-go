# Interceptors

Interceptors let you plug cross‑cutting behaviors (auth, logging, retries, error shaping, etc.) into every HTTP request/response made by the SDK. They wrap the underlying `http.RoundTripper` and are executed in a chain for each request.

Interceptors make it easy to:

- Add auth headers
- Log requests and responses
- Retry failed calls with backoff
- Normalize non-2xx responses into Go errors

---

## How the interceptor chain works

The interceptor chain is executed by `InterceptorTransport` (see `sdk/interceptors/transport.go`):

- Before the request is sent, each interceptor's `BeforeRequest` is called in the order they were added.
- The actual HTTP call is performed once all `BeforeRequest` calls finish without setting an error or short‑circuiting with a response.
- After a response is received, each interceptor's `AfterResponse` is called in the same order.

An interceptor can:

- Modify the request before it's sent.
- Short‑circuit the call by returning a response early (e.g., cached response).
- Retry a failed request by re‑issuing the transport call.
- Convert a response into an error by setting `data.Error`.

If any interceptor returns a non‑nil error from `BeforeRequest`/`AfterResponse`, the chain stops and that error is returned.

---

## Interfaces and data model

Defined in `sdk/interceptors/interceptors.go`:

```go
type InterceptorData struct {
    ID             string
    Ctx            context.Context
    InitialRequest *http.Request
    Request        *http.Request
    Response       *http.Response
    Error          error
}

type Interceptor interface {
    BeforeRequest(data InterceptorData) (InterceptorData, error)
    AfterResponse(data InterceptorData) (InterceptorData, error)
}
```


Notes:

- `ID` is a unique identifier per request (UUID), useful for correlating logs and retries.
- `InitialRequest` is an immutable clone of the original request for reference during retries.
- Interceptors should write changes into `data.Request`, `data.Response`, or `data.Error` and return the updated `InterceptorData`.

---

## Attaching interceptors

You can attach interceptors globally across all modules via the SDK option, or add them per service handler.

1. Globally (recommended) via `sdk/sdk.go`:

```go
package main

import (
    "log"
    "time"

    sotton "github.com/sotoon/sotoon-sdk-go/sdk"
    "github.com/sotoon/sotoon-sdk-go/sdk/interceptors"
)

func main() {
    secretKey := "<YOUR_SECRET>"

    sdk, err := sotton.NewSDK(secretKey,
        sotton.WithInterceptor(
            interceptors.NewLogger(interceptors.LoggerOptions{
                LogBasicInfo:   true,
                LogHeaders:     true,
                LogBody:        false,
                MaxBodyLogSize: 1024,
                SkipHeaders:    []string{"authorization"},
                SkipPaths:      []string{"/health"},
            }),
            interceptors.NewRetryInterceptor(
                interceptors.NewDefaultInterceptorTransport(secretKey), // transporter
                interceptors.NewRetryInterceptor_ExponentialBackoff(200*time.Millisecond, 5*time.Second),
                interceptors.NewRetryInterceptor_RetryDeciderAll(5), // max 5 retries
            ),
            interceptors.NewTreatAsErrorInterceptor(
                interceptors.NewTreatAsErrorInterceptor_ErrorDetectorAll(), // treat all non-2xx as errors
            ),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    // use sdk.Compute / sdk.Iam / sdk.Engine ...
}
```

The `WithInterceptor` option propagates the interceptors to all module handlers:

- `sdk/core/compute/handler.go`
- `sdk/core/iam/handler.go`
- `sdk/core/sotoon-kubernetes-engine/handler.go`

1. Per module handler:

```go
sdk, _ := sotton.NewSDK(secretKey)
sdk.Compute.AddInterceptors(
    interceptors.NewLogger(interceptors.LoggerOptions{LogBasicInfo: true}),
)
```

Order matters:

- `BeforeRequest` and `AfterResponse` both run in the order you add interceptors.
- Place `Authenticator` early. Place `Logger` early if you want to log pre‑mutation state.

---

## Available interceptors

### 1) Authenticator

File: `sdk/interceptors/authenticator.go`

Adds a bearer token header to every request.

```go
a := interceptors.NewAuthenticator(secretKey)
```

Behavior:

- `BeforeRequest`: sets `Authorization: Bearer <secretKey>`
- `AfterResponse`: no‑op

Note: This interceptor is included by default by `NewDefaultInterceptorTransport(secretKey)`.

---

### 2) Logger

File: `sdk/interceptors/logger.go`

Configurable request/response logging.

```go
logOpts := interceptors.LoggerOptions{
    Logger:         log.Default(), // optional, defaults to log.Default()
    LogBasicInfo:   true,          // method, URL, status
    LogHeaders:     true,          // request/response headers
    LogBody:        true,          // request/response bodies (truncated)
    MaxBodyLogSize: 2048,          // default 1024 bytes
    SkipHeaders:    []string{"authorization"}, // case-insensitive
    SkipPaths:      []string{"/health"},       // prefixes
}
logger := interceptors.NewLogger(logOpts)
```

Behavior:

- `BeforeRequest`: logs outgoing request (honoring skip/path/body/headers options).
- `AfterResponse`: logs incoming response (status, headers, body).
- Bodies are safely re‑buffered so downstream interceptors/consumers can still read them.
- `ID` is included in logs to correlate request/response pairs.

---

### 3) Retry

File: `sdk/interceptors/retry.go`

Retries failed requests with configurable backoff and decision logic.

Key interfaces:

```go
type Transporter interface {
    RoundTripWithID(req *http.Request, id string) (*http.Response, error)
}

type BackoffTimer interface {
    TimeToWait(iteration int) time.Duration
}

type RetryDecider interface {
    // Return true to retry, false to stop.
    // Returning a non-nil error aborts the chain.
    ShouldRetry(resp *http.Response, err error, meta RetryInternalData) (bool, error)
}

type RetryInternalData struct {
    RetryCount int
}
```


Provided helpers:

- Backoff strategies:
  - `NewRetryInterceptor_ExponentialBackoff(base, max time.Duration)` → `BackoffStrategyExpnential`
  - `NewRetryInterceptor_BackoffStrategyLinier(base time.Duration)` → fixed interval strategy
- Decider:
  - `NewRetryInterceptor_RetryDeciderAll(maxRetries int)` → retries on any error or non‑2xx status until `maxRetries`

Construction:

```go
retry := interceptors.NewRetryInterceptor(
    interceptors.NewDefaultInterceptorTransport(secretKey), // Transporter used for re‑issuing the request
    interceptors.NewRetryInterceptor_ExponentialBackoff(300*time.Millisecond, 5*time.Second),
    interceptors.NewRetryInterceptor_RetryDeciderAll(5),
)
```

Behavior:

- If `data.Error` is already set before send, `BeforeRequest` can trigger a retry.
- After the response, `AfterResponse` checks the decider; if it returns true, it sleeps based on the backoff and re‑issues the original `InitialRequest` via `RoundTripWithID`, incrementing `RetryCount`.
- Stops retrying when decider returns false or `maxRetries` exceeded, in which case it returns the last error or `constants.ErrMaxRetriesExceeded`.
---

### 4) Treat‑As‑Error

File: `sdk/interceptors/treat_as_error.go`

Converts non‑2xx responses into Go errors with structured messages.

Interfaces and helpers:

```go
type ErrorDetector interface {
    IsError(data InterceptorData) error
}

func NewTreatAsErrorInterceptor(errorDetector ErrorDetector) *TreatAsErrorInterceptor
func NewTreatAsErrorInterceptor_ErrorDetectorAll() ErrorDetector // non‑2xx => error
```


Default detector behavior (`ErrorDetectorAll`):

- If `StatusCode >= 400`, tries to parse JSON from the response body and extract one of:
  - `message.detail`
  - `reason`
  - `error`
- Falls back to a generic error like `non-2xx response: <status>`.
- The response body is re‑buffered so it remains readable by subsequent consumers.

Usage:

```go
treat := interceptors.NewTreatAsErrorInterceptor(
    interceptors.NewTreatAsErrorInterceptor_ErrorDetectorAll(),
)
```

---


### 5) Circuit Breaker

File: `sdk/interceptors/circute_breaker.go`

Implements the circuit breaker pattern to prevent cascading failures by temporarily stopping requests when the system is under stress or experiencing high failure rates.

```go
type CircuitBreakerInterceptor struct {
    abortOnFailure bool
    cb             *gobreaker.CircuitBreaker
}
```

Behavior:

- `BeforeRequest`: Checks if the circuit is open. If open and `abortOnFailure` is true, returns `constants.ErrCircuitBreakerOpen`. Otherwise, sets `data.Error` to `constants.ErrCircuitBreakerOpen` but allows the chain to continue.
- `AfterResponse`: Records the response status code in the circuit breaker to track failure rates.

Usage example:

```go
sdk, err := sotton.NewSDK(secretKey,
    sotton.WithInterceptor(
        interceptors.NewCircuitBreakerInterceptor(
            interceptors.CircuteBreakerForJust429,
            true,
        ),
        // Other interceptors...
    ),
)
```

---

## Transport layer

`InterceptorTransport` (see `sdk/interceptors/transport.go`) is a custom `http.RoundTripper` that executes the interceptor chain.

Constructors:

```go
func NewDefaultInterceptorTransport(secretKey string) *InterceptorTransport
func NewInterceptorTransport(rt http.RoundTripper, interceptors []Interceptor) *InterceptorTransport
```


You can also wire this transport to a custom `http.Client` if you're building clients by hand:

```go
it := interceptors.NewDefaultInterceptorTransport(secretKey)
it.AddInterceptors(
    interceptors.NewLogger(interceptors.LoggerOptions{LogBasicInfo: true}),
)
httpClient := &http.Client{Transport: it}
```

In normal usage, SDK handlers construct their HTTP clients internally. Prefer adding interceptors via `sotton.WithInterceptor(...)` or per‑handler `AddInterceptors(...)`.

---

## Practical examples

- Global setup

```go
sdk, err := sotton.NewSDK(secretKey,
    sotton.WithInterceptor(
        interceptors.NewLogger(interceptors.LoggerOptions{
            LogBasicInfo: true,
            LogHeaders:   true,
            LogBody:      false,
            SkipHeaders:  []string{"authorization"},
        }),
        interceptors.NewRetryInterceptor(
            interceptors.NewDefaultInterceptorTransport(secretKey),
            interceptors.NewRetryInterceptor_ExponentialBackoff(250*time.Millisecond, 3*time.Second),
            interceptors.NewRetryInterceptor_RetryDeciderAll(4),
        ),
        interceptors.NewTreatAsErrorInterceptor(
            interceptors.NewTreatAsErrorInterceptor_ErrorDetectorAll(),
        ),
    ),
)
if err != nil { log.Fatal(err) }
```

- Add per‑module

```go
sdk, _ := sotton.NewSDK(secretKey)
sdk.Engine.AddInterceptors(
    interceptors.NewLogger(interceptors.LoggerOptions{LogBasicInfo: true}),
)
```

---

## Tips

- Place `Authenticator` early so auth headers are always present.
- Use Treat‑As‑Error to fail fast on non‑2xx responses and surface helpful messages.
- Tuning retry:
  - Keep `maxRetries` small and use exponential backoff with jitter to avoid thundering herds.
  - Think carefully about idempotency when enabling retries.
- Logging:
  - Avoid logging sensitive headers or large bodies. Use `SkipHeaders`, `SkipPaths`, and `MaxBodyLogSize`.
