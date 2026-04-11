package currency

import (
	"bytes"
	"encoding/json"
	"envdash/internal/structs"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// MockTransport allows us to mock HTTP responses by implementing http.RoundTripper
type MockTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

type ErrorReader struct{}

func (e *ErrorReader) Read(_ []byte) (n int, err error) {
	return 0, fmt.Errorf("mock body read error")
}

func (e *ErrorReader) Close() error {
	return nil
}

func TestFetchExchangeRates_Success(t *testing.T) {
	client := NewCurrencyClient()

	mockData := structs.IncomingCurrency{
		Rates: map[string]float64{
			"NOK": 10.5,
			"EUR": 0.9,
		},
	}
	responseBody, _ := json.Marshal(mockData)

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(responseBody)),
			}, nil
		},
	}

	result, err := client.FetchExchangeRates("USD")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result == nil || result.Rates == nil {
		t.Fatalf("Expected valid results, got nil")
	}
	if result.Rates["NOK"] != 10.5 {
		t.Errorf("Expected NOK rate 10.5, got %f", result.Rates["NOK"])
	}
}

func TestFetchExchangeRates_ErrorFetching(t *testing.T) {
	client := NewCurrencyClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		},
	}

	_, err := client.FetchExchangeRates("USD")
	if err == nil {
		t.Fatalf("Expected network error, got nil")
	}
}
