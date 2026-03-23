package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//Test for the statusClient
//Made with claude 4.6

// TestProbeHeadEndpoint_Success tests successful HEAD request
func TestProbeHeadEndpoint_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "HEAD", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewStatusClient()
	statusCode, err := client.ProbeHeadEndpoint(server.URL)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
}

// TestProbeHeadEndpoint_ServerError tests HEAD request returning error status
func TestProbeHeadEndpoint_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewStatusClient()
	statusCode, err := client.ProbeHeadEndpoint(server.URL)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, statusCode)
}

// TestProbeHeadEndpoint_InvalidURL tests HEAD request with invalid URL
func TestProbeHeadEndpoint_InvalidURL(t *testing.T) {
	client := NewStatusClient()
	statusCode, err := client.ProbeHeadEndpoint("http://invalid-url-that-does-not-exist:9999")

	assert.Error(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, statusCode)
}

// TestProbeGetEndpoint_Success tests successful GET request
func TestProbeGetEndpoint_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	client := NewStatusClient()
	statusCode := client.ProbeGetEndpoint(server.URL, "", "")

	assert.Equal(t, http.StatusOK, statusCode)
}

// TestProbeGetEndpoint_WithCustomHeader tests GET request with custom header
func TestProbeGetEndpoint_WithCustomHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "CustomValue", r.Header.Get("CustomKey"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewStatusClient()
	statusCode := client.ProbeGetEndpoint(server.URL, "CustomKey", "CustomValue")

	assert.Equal(t, http.StatusOK, statusCode)
}

// TestProbeGetEndpoint_InvalidURL tests GET request with invalid URL
func TestProbeGetEndpoint_InvalidURL(t *testing.T) {
	client := NewStatusClient()
	statusCode := client.ProbeGetEndpoint("http://invalid-url-that-does-not-exist:9999", "", "")

	assert.Equal(t, http.StatusServiceUnavailable, statusCode)
}

// TestProbeGetEndpoint_NotFound tests GET request returning 404
func TestProbeGetEndpoint_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewStatusClient()
	statusCode := client.ProbeGetEndpoint(server.URL, "", "")

	assert.Equal(t, http.StatusNotFound, statusCode)
}

// TestProbeHeadEndpoint_Timeout tests HEAD request timeout
func TestProbeHeadEndpoint_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Sleep longer than client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewStatusClient()
	statusCode, err := client.ProbeHeadEndpoint(server.URL)

	assert.Error(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, statusCode)
}

// TestProbeHeadEndpoint_ConnectionRefused tests HEAD request to closed port
func TestProbeHeadEndpoint_ConnectionRefused(t *testing.T) {
	client := NewStatusClient()
	statusCode, err := client.ProbeHeadEndpoint("http://localhost:1")

	assert.Error(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, statusCode)
}

// TestProbeHeadEndpoint_Forbidden tests HEAD request returning 403
func TestProbeHeadEndpoint_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewStatusClient()
	statusCode, err := client.ProbeHeadEndpoint(server.URL)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, statusCode)
}

// TestProbeGetEndpoint_Timeout tests GET request timeout
func TestProbeGetEndpoint_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewStatusClient()
	statusCode := client.ProbeGetEndpoint(server.URL, "", "")

	assert.Equal(t, http.StatusServiceUnavailable, statusCode)
}

// TestProbeGetEndpoint_BadRequest tests GET request returning 400
func TestProbeGetEndpoint_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewStatusClient()
	statusCode := client.ProbeGetEndpoint(server.URL, "", "")

	assert.Equal(t, http.StatusBadRequest, statusCode)
}

// TestProbeGetEndpoint_Unauthorized tests GET request returning 401
func TestProbeGetEndpoint_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewStatusClient()
	statusCode := client.ProbeGetEndpoint(server.URL, "", "")

	assert.Equal(t, http.StatusUnauthorized, statusCode)
}

// TestProbeGetEndpoint_OnlyHeaderValueProvided tests GET with value but no key
func TestProbeGetEndpoint_OnlyHeaderValueProvided(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When only value provided, no header should be set
		assert.Equal(t, "", r.Header.Get("CustomKey"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewStatusClient()
	statusCode := client.ProbeGetEndpoint(server.URL, "", "SomeValue")

	assert.Equal(t, http.StatusOK, statusCode)
}

// TestProbeGetEndpoint_ServerError tests GET request returning 500
func TestProbeGetEndpoint_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewStatusClient()
	statusCode := client.ProbeGetEndpoint(server.URL, "", "")

	assert.Equal(t, http.StatusInternalServerError, statusCode)
}
