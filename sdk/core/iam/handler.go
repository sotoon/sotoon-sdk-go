package iam

import (
	"github.com/sotoon/sotoon-sdk-go/sdk/interceptors"
	"net/http"
)

type Handler struct {
	*ClientWithResponses
	interceptorTransport *interceptors.InterceptorTransport
}

type HandlerOption func(*Handler) *Handler

func WithInterceptor(interceptors ...interceptors.Interceptor) HandlerOption {
	return func(handler *Handler) *Handler {
		handler.interceptorTransport.AddInterceptors(interceptors...)
		return handler
	}
}

func NewHandler(serverAddress, secretKey string, opts ...HandlerOption) (*Handler, error) {
	interceptorTransport := interceptors.NewDefaultInterceptorTransport(secretKey)
	client, err := NewClientWithResponses(
		serverAddress,
		WithHTTPClient(&http.Client{
			Transport: interceptorTransport,
		}))

	if err != nil {
		return nil, err
	}

	handler := &Handler{
		ClientWithResponses:  client,
		interceptorTransport: interceptorTransport,
	}
	for _, opt := range opts {
		handler = opt(handler)
	}
	return handler, nil
}

func (h *Handler) AddInterceptors(interceptors ...interceptors.Interceptor) {
	h.interceptorTransport.AddInterceptors(interceptors...)
}
