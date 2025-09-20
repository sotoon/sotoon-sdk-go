package interceptors

import (
	"context"
	"net/http"
)

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
