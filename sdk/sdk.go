package sotton

import (
	iam_v1 "github.com/sotoon/sotoon-sdk-go/sdk/core/iam_v1"
	"github.com/sotoon/sotoon-sdk-go/sdk/interceptors"
)

const (
	serverAddress = "https://api.sotoon.ir"
)

type SDK struct {
	Iam_v1 *iam_v1.Handler
}

type SDKOption func(SDK) SDK

func NewSDK(secretKey string, opts ...SDKOption) (*SDK, error) {

	iam_v1Client, err := iam_v1.NewHandler(serverAddress, secretKey)
	if err != nil {
		return nil, err
	}

	sdk := SDK{
		Iam_v1: iam_v1Client,
	}
	for _, opt := range opts {
		sdk = opt(sdk)
	}
	return &sdk, nil
}

func WithInterceptor(interceptors ...interceptors.Interceptor) SDKOption {
	return func(s SDK) SDK {
		s.Iam_v1.AddInterceptors(interceptors...)
		return s
	}
}
