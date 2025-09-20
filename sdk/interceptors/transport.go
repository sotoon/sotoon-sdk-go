package interceptors

import (
	"net/http"

	"github.com/google/uuid"
)

type InterceptorTransport struct {
	rt           http.RoundTripper
	interceptors []Interceptor
}

func NewDefaultInterceptorTransport(secretKey string) *InterceptorTransport {
	return &InterceptorTransport{
		rt: http.DefaultTransport,
		interceptors: []Interceptor{
			NewAuthenticator(secretKey),
		},
	}
}

func NewInterceptorTransport(rt http.RoundTripper, interceptors []Interceptor) *InterceptorTransport {
	return &InterceptorTransport{
		rt:           rt,
		interceptors: interceptors,
	}
}

func (it *InterceptorTransport) AddInterceptors(interceptors ...Interceptor) {
	it.interceptors = append(it.interceptors, interceptors...)
}

func (it *InterceptorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return it.RoundTripWithID(req, uuid.New().String())
}

func (it *InterceptorTransport) RoundTripWithID(req *http.Request, id string) (*http.Response, error) {

	initialReq := req.Clone(req.Context())

	var InterceptorData InterceptorData = InterceptorData{
		ID:             id,
		Ctx:            req.Context(),
		InitialRequest: initialReq,
		Request:        req,
		Response:       nil,
		Error:          nil,
	}
	var err error
	for _, interceptor := range it.interceptors {
		InterceptorData, err = interceptor.BeforeRequest(InterceptorData)
		if err != nil {
			return nil, err
		}
		if InterceptorData.Response != nil {
			return InterceptorData.Response, nil
		}
	}
	if InterceptorData.Error != nil {
		return nil, InterceptorData.Error
	}

	req = InterceptorData.Request
	resp, err := it.rt.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	InterceptorData.Response = resp

	for _, interceptor := range it.interceptors {
		InterceptorData, err = interceptor.AfterResponse(InterceptorData)
		if err != nil {
			return nil, err
		}
	}
	if InterceptorData.Error != nil {
		return nil, InterceptorData.Error
	}
	return resp, nil
}
