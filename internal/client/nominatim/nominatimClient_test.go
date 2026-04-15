package nominatim

import (
	"bytes"
	"encoding/json"
	"envdash/internal/structs"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// MockTransport allows us to mock HTTP responses by implementing http.RoundTripper
type MockTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestFetchCapitalCoords_Success(t *testing.T) {
	client := NewNomClient()

	mockData := []structs.NomIncoming{
		{
			Lat: "59.9139",
			Lon: "10.7522",
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

	result, err := client.FetchCapitalCoords("Oslo")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Fatalf("Expected valid results, got nil")
	}
	if result.Lat != "59.9139" {
		t.Errorf("Expected lat 59.9139, got %s", result.Lat)
	}
	if result.Lon != "10.7522" {
		t.Errorf("Expected lon 10.7522, got %s", result.Lon)
	}
}

func TestFetchCapitalCoords_NetworkError(t *testing.T) {
	client := NewNomClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network connection failed")
		},
	}

	_, err := client.FetchCapitalCoords("Oslo")
	if err == nil {
		t.Fatalf("Expected network error, got nil")
	}
}

func TestFetchCapitalCoords_BadStatusCode(t *testing.T) {
	client := NewNomClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBuffer([]byte(""))),
			}, nil
		},
	}

	_, err := client.FetchCapitalCoords("InvalidCity")
	if err == nil {
		t.Fatalf("Expected status code error, got nil")
	}
	if err.Error() != "nominatim returned status 404" {
		t.Errorf("Expected status code error, got %v", err)
	}
}

func TestFetchCapitalCoords_InvalidJSON(t *testing.T) {
	client := NewNomClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer([]byte("invalid json {"))),
			}, nil
		},
	}

	_, err := client.FetchCapitalCoords("Oslo")
	if err == nil {
		t.Fatalf("Expected JSON parse error, got nil")
	}
}

func TestFetchCapitalCoords_EmptyResponse(t *testing.T) {
	client := NewNomClient()

	mockData := []structs.NomIncoming{}
	responseBody, _ := json.Marshal(mockData)

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(responseBody)),
			}, nil
		},
	}

	_, err := client.FetchCapitalCoords("NonExistentCity")
	if err == nil {
		t.Fatalf("Expected empty result error, got nil")
	}
	if err.Error() != "no results found for capital: NonExistentCity" {
		t.Errorf("Expected empty result error, got %v", err)
	}
}

func TestFetchCapitalCoords_ServerError(t *testing.T) {
	client := NewNomClient()

	client.httpClient.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBuffer([]byte(""))),
			}, nil
		},
	}

	_, err := client.FetchCapitalCoords("Oslo")
	if err == nil {
		t.Fatalf("Expected server error, got nil")
	}
}

func TestRequestNom_HeaderSet(t *testing.T) {
	req, err := requestNom("Oslo")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if req == nil {
		t.Fatalf("Expected valid request, got nil")
	}

	userAgent := req.Header.Get("User-Agent")
	if userAgent != "@cloud2-group23" {
		t.Errorf("Expected User-Agent @cloud2-group23, got %s", userAgent)
	}
}

func TestNomUrl_ValidCapital(t *testing.T) {
	url := nomUrl("Oslo")
	if url == "" {
		t.Fatalf("Expected non-empty URL")
	}
	if !contains(url, "q=Oslo") {
		t.Errorf("Expected q parameter in URL, got %s", url)
	}
	if !contains(url, "format=json") {
		t.Errorf("Expected format parameter in URL, got %s", url)
	}
	if !contains(url, "limit=1") {
		t.Errorf("Expected limit parameter in URL, got %s", url)
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

func TestFetchCapitalCoords_MultipleResults(t *testing.T) {
	client := NewNomClient()

	mockData := []structs.NomIncoming{
		{
			Lat: "59.9139",
			Lon: "10.7522",
		},
		{
			Lat: "51.5074",
			Lon: "-0.1278",
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

	result, err := client.FetchCapitalCoords("Oslo")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Fatalf("Expected valid results, got nil")
	}
	// Should return first result
	if result.Lat != "59.9139" {
		t.Errorf("Expected first result lat 59.9139, got %s", result.Lat)
	}
}

func TestNewNomClient(t *testing.T) {
	client := NewNomClient()
	if client == nil {
		t.Fatalf("Expected valid client, got nil")
	}
	if client.httpClient == nil {
		t.Fatalf("Expected valid httpClient, got nil")
	}
	if client.httpClient.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", client.httpClient.Timeout)
	}
}
