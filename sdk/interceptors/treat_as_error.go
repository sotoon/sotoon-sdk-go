package interceptors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

type ErrorDetector interface {
	// IsError will return error if any of the conditions are met
	IsError(data InterceptorData) error
}

type TreatAsErrorInterceptor struct {
	ErrorDetector ErrorDetector
}

func NewTreatAsErrorInterceptor(errorDetector ErrorDetector) *TreatAsErrorInterceptor {
	return &TreatAsErrorInterceptor{
		ErrorDetector: errorDetector,
	}
}

func (a *TreatAsErrorInterceptor) BeforeRequest(data InterceptorData) (InterceptorData, error) {
	return data, nil
}

func (a *TreatAsErrorInterceptor) AfterResponse(data InterceptorData) (InterceptorData, error) {
	if err := a.ErrorDetector.IsError(data); err != nil {
		data.Error = err
	}
	return data, nil
}

/////////////////////////////////////////////////////////

type treatAsErrorInterceptor_ErrorDetectorAll struct{}

type errorTemplate struct {
	Details string `json:"details"`
	Reason  string `json:"reason"`
	Message struct {
		Detail string `json:"detail"`
	} `json:"message"`
	Error string `json:"error"`
}

func (a *treatAsErrorInterceptor_ErrorDetectorAll) IsError(data InterceptorData) error {
	if data.Response != nil && data.Response.StatusCode >= 400 {
		defaultError := fmt.Errorf("non-2xx response: %d", data.Response.StatusCode)
		if data.Response.Body != nil {
			body, err := io.ReadAll(data.Response.Body)
			if err != nil {
				return err
			}
			data.Response.Body = io.NopCloser(bytes.NewReader(body))
			var errorTemplate errorTemplate
			if err := json.Unmarshal(body, &errorTemplate); err != nil {
				log.Println("failed to Unmarshall errorTemplate", err)
			}

			switch {
			case errorTemplate.Message.Detail != "":
				return fmt.Errorf("%s", errorTemplate.Message.Detail)
			case errorTemplate.Reason != "":
				return fmt.Errorf("%s", errorTemplate.Reason)
			case errorTemplate.Error != "":
				return fmt.Errorf("%s", errorTemplate.Error)
			default:
				return defaultError
			}
		}

		return defaultError
	}
	return nil
}

// TreatAsErrorInterceptor_ErrorDetectorAll will treat all non-2xx (>=400) responses as errors
func NewTreatAsErrorInterceptor_ErrorDetectorAll() ErrorDetector {
	return &treatAsErrorInterceptor_ErrorDetectorAll{}
}
