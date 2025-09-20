package sotton

import (
	compute "github.com/sotoon/sotoon-sdk-go/sdk/core/compute"
	sotoon_kubernetes_engine "github.com/sotoon/sotoon-sdk-go/sdk/core/sotoon-kubernetes-engine"
	"github.com/sotoon/sotoon-sdk-go/sdk/interceptors"
)

const (
	serverAddress = "https://api.sotoon.ir"
)

type SDK struct {
	Compute *compute.Handler
	Engine *sotoon_kubernetes_engine.Handler
}

type SDKOption func(SDK) SDK

func NewSDK(secretKey string, opts ...SDKOption) (*SDK, error) {

	computeClient, err := compute.NewHandler(serverAddress, secretKey)
	if err != nil {
		return nil, err
	}

	engineClient, err := sotoon_kubernetes_engine.NewHandler(serverAddress, secretKey)
	if err != nil {
		return nil, err
	}

	sdk := SDK{
		Compute: computeClient,
		Engine: engineClient,
	}
	for _, opt := range opts {
		sdk = opt(sdk)
	}
	return &sdk, nil
}

func WithInterceptor(interceptors ...interceptors.Interceptor) SDKOption {
	return func(s SDK) SDK {
		s.Compute.AddInterceptors(interceptors...)
		s.Engine.AddInterceptors(interceptors...)
		return s
	}
}
