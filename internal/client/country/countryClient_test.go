package country

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

func TestFetchCountryData_Success(t *testing.T) {
	client := NewCountryClient()

	mockResponseData := []structs.IncomingCountry{
		{
			Name: struct {
				Common string `json:"common"`
			}{Common: "Norway"},
			IsoCode:    "NO",
			Capital:    []string{"Oslo"},
			Population: 5000000,
		},
	}
	responseBody, _ := json.Marshal(mockResponseData)

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(responseBody)),
			}, nil
		},
	}

	result, err := client.FetchCountryData("NO")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) == 0 {
		t.Fatalf("Expected results, got %v", result)
	}

	if result[0].Name.Common != "Norway" {
		t.Errorf("Expected 'Norway', got %v", result[0].Name.Common)
	}
}

func TestFetchCountryData_ErrorFetching(t *testing.T) {
	client := NewCountryClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		},
	}

	_, err := client.FetchCountryData("NO")
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestFetchCountryData_InvalidStatusCode(t *testing.T) {
	client := NewCountryClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
			}, nil
		},
	}

	_, err := client.FetchCountryData("XX")
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestFetchCountryData_InvalidJSON(t *testing.T) {
	client := NewCountryClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("{invalid json")),
			}, nil
		},
	}

	_, err := client.FetchCountryData("NO")
	if err == nil {
		t.Fatalf("Expected error due to invalid JSON, got nil")
	}
}

type ErrorReader struct{}

func (e *ErrorReader) Read(_ []byte) (n int, err error) {
	return 0, fmt.Errorf("mock body read error")
}

func (e *ErrorReader) Close() error {
	return nil
}

func TestFetchCountryData_BodyReadError(t *testing.T) {
	client := NewCountryClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       &ErrorReader{},
			}, nil
		},
	}

	_, err := client.FetchCountryData("NO")
	if err == nil {
		t.Fatalf("Expected error due to body read failure, got nil")
	}
}
