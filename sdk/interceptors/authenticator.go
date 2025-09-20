package interceptors

type Authenticator struct {
	secretKey string
}

func NewAuthenticator(secretKey string) *Authenticator {
	return &Authenticator{
		secretKey: secretKey,
	}
}

func (a *Authenticator) BeforeRequest(data InterceptorData) (InterceptorData, error) {
	data.Request.Header.Set("Authorization", "Bearer "+a.secretKey)
	return data, nil
}

func (a *Authenticator) AfterResponse(data InterceptorData) (InterceptorData, error) {
	return data, nil
}
