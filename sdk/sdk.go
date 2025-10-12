package sotton

import (
	iamV1 "github.com/sotoon/sotoon-sdk-go/sdk/core/iamV1"
	"github.com/sotoon/sotoon-sdk-go/sdk/interceptors"
)

const (
	serverAddress = "https://api.sotoon.ir"
)

type SDK struct {
	IamV1 *iamV1.Handler
}

type SDKOption func(SDK) SDK

func NewSDK(secretKey string, opts ...SDKOption) (*SDK, error) {

	iamV1Client, err := iamV1.NewHandler(serverAddress, secretKey)
	if err != nil {
		return nil, err
	}

	sdk := SDK{
		IamV1: iamV1Client,
	}
	for _, opt := range opts {
		sdk = opt(sdk)
	}
	return &sdk, nil
}

func WithInterceptor(interceptors ...interceptors.Interceptor) SDKOption {
	return func(s SDK) SDK {
		s.IamV1.AddInterceptors(interceptors...)
		return s
	}
}
