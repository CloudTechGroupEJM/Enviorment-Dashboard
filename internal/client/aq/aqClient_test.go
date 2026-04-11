package aq

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

func TestFetchSensors_Success(t *testing.T) {
	client := NewAQClient()

	mockData := structs.AirQualityIncoming{
		Results: []struct {
			ID      int `json:"id"`
			Sensors []struct {
				ID        int `json:"id"`
				Parameter struct {
					Name string `json:"name"`
				} `json:"parameter"`
			} `json:"sensors"`
		}{
			{
				ID: 101,
				Sensors: []struct {
					ID        int `json:"id"`
					Parameter struct {
						Name string `json:"name"`
					} `json:"parameter"`
				}{
					{
						ID: 201,
						Parameter: struct {
							Name string `json:"name"`
						}{Name: "pm25"},
					},
				},
			},
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

	result, err := client.FetchSensors(59.91, 10.75)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(result.Results) == 0 || result.Results[0].ID != 101 {
		t.Fatalf("Expected valid results, got %v", result)
	}
}

func TestFetchSensors_Error(t *testing.T) {
	client := NewAQClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		},
	}

	_, err := client.FetchSensors(59.91, 10.75)
	if err == nil {
		t.Fatalf("Expected network error, got nil")
	}
}

func TestFetchSensors_InvalidStatusCode(t *testing.T) {
	client := NewAQClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Request:    req,
			}, nil
		},
	}

	_, err := client.FetchSensors(59.91, 10.75)
	if err == nil {
		t.Fatalf("Expected status code error, got nil")
	}
}

func TestFetchSensors_InvalidJSON(t *testing.T) {
	client := NewAQClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("invalid json")),
			}, nil
		},
	}

	_, err := client.FetchSensors(59.91, 10.75)
	if err == nil {
		t.Fatalf("Expected json decode error, got nil")
	}
}

func TestFetchLatest_Success(t *testing.T) {
	client := NewAQClient()

	mockData := structs.LatestIncoming{
		Results: []struct {
			Value     float64 `json:"value"`
			SensorsID int     `json:"sensorsId"`
			Parameter string  `json:"parameter"`
		}{
			{Value: 12.5, SensorsID: 201, Parameter: "pm25"},
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

	result, err := client.FetchLatest(101)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(result.Results) == 0 || result.Results[0].Value != 12.5 {
		t.Fatalf("Expected valid result, got %v", result)
	}
}

func TestFetchLatest_Error(t *testing.T) {
	client := NewAQClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		},
	}

	_, err := client.FetchLatest(101)
	if err == nil {
		t.Fatalf("Expected network error, got nil")
	}
}

func TestFetchLatest_InvalidStatusCode(t *testing.T) {
	client := NewAQClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("Not found")),
			}, nil
		},
	}

	_, err := client.FetchLatest(101)
	if err == nil {
		t.Fatalf("Expected status code error, got nil")
	}
}

func TestFetchLatest_ReadBodyError(t *testing.T) {
	client := NewAQClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       &ErrorReader{},
			}, nil
		},
	}

	_, err := client.FetchLatest(101)
	if err == nil {
		t.Fatalf("Expected error reading body, got nil")
	}
}
