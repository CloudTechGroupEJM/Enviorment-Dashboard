package metro

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

func TestFetchMetroData_Success(t *testing.T) {
	client := NewMetroClient()

	mockData := structs.MetroAPIIncoming{
		Daily: struct {
			Temperature2mMean []float64 `json:"temperature_2m_mean"`
			PrecipitationSum  []float64 `json:"precipitation_sum"`
		}{
			Temperature2mMean: []float64{15.5, 16.2, 17.1, 18.0, 16.5, 15.3, 14.8},
			PrecipitationSum:  []float64{0.5, 1.2, 0.0, 2.3, 0.8, 0.1, 1.5},
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

	result, err := client.FetchMetroData(59.9139, 10.7522)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Fatalf("Expected valid results, got nil")
	}
	if len(result.Daily.Temperature2mMean) != 7 {
		t.Errorf("Expected 7 temperature values, got %d", len(result.Daily.Temperature2mMean))
	}
}

func TestFetchMetroData_NetworkError(t *testing.T) {
	client := NewMetroClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network connection failed")
		},
	}

	_, err := client.FetchMetroData(59.9139, 10.7522)
	if err == nil {
		t.Fatalf("Expected network error, got nil")
	}
}

func TestFetchMetroData_InvalidStatusCode(t *testing.T) {
	client := NewMetroClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(bytes.NewBuffer([]byte(""))),
			}, nil
		},
	}

	_, err := client.FetchMetroData(999.9, 999.9)
	if err == nil {
		t.Fatalf("Expected status code error, got nil")
	}
}

func TestFetchMetroData_InvalidJSON(t *testing.T) {
	client := NewMetroClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer([]byte("invalid json {"))),
			}, nil
		},
	}

	_, err := client.FetchMetroData(59.9139, 10.7522)
	if err == nil {
		t.Fatalf("Expected JSON parse error, got nil")
	}
}

func TestMetroUrl_ValidCoordinates(t *testing.T) {
	url := metroUrl(59.9139, 10.7522)
	if url == "" {
		t.Fatalf("Expected non-empty URL")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
