package aq

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadStubFixture(t *testing.T, fileName string) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to resolve current test file path")

	stubPath := filepath.Join(filepath.Dir(thisFile), "..", "stubs", fileName)
	bytes, err := os.ReadFile(stubPath)
	require.NoError(t, err, "failed reading stub fixture %s", stubPath)
	require.True(t, json.Valid(bytes), "fixture is not valid JSON: %s", stubPath)

	return string(bytes)
}

// rewriteTransport redirects every outgoing request to target's host/scheme
// while preserving path, query, headers, and body. This lets us stub the
// upstream without touching config constants or production code.
type rewriteTransport struct {
	target *url.URL
}

func (rt *rewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	clone := r.Clone(r.Context())
	clone.URL.Scheme = rt.target.Scheme
	clone.URL.Host = rt.target.Host
	clone.Host = rt.target.Host
	return http.DefaultTransport.RoundTrip(clone)
}

// newStubbedClient spins up an httptest.Server running handler and returns an
// AQClient whose HTTP transport routes all requests to that server. The API
// key is fixed so tests can assert the X-API-Key header regardless of the
// OPEN_AQ_API_KEY env var in the developer's shell.
func newStubbedClient(t *testing.T, handler http.HandlerFunc) *AQClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	target, err := url.Parse(srv.URL)
	require.NoError(t, err)

	return &AQClient{
		httpClient: &http.Client{
			Timeout:   3 * time.Second,
			Transport: &rewriteTransport{target: target},
		},
		apiKey: "test-api-key-12345",
	}
}

func TestFetchSensors(t *testing.T) {
	sensorsStub := loadStubFixture(t, "aqSensorStub.json")

	type requestAssertion func(t *testing.T, r *http.Request)

	tests := []struct {
		name         string
		lat          float64
		long         float64
		handler      http.HandlerFunc
		requestCheck requestAssertion
		wantErr      bool
		errContains  string
	}{
		{
			name: "Returns sensors",
			lat:  59.9133,
			long: 10.7390,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, sensorsStub)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method, "method")
				assert.True(t, strings.HasSuffix(r.URL.Path, "/locations"),
					"path %q should end with /locations", r.URL.Path)
				q := r.URL.Query()
				assert.Equal(t, "59.9133,10.7390", q.Get("coordinates"), "coordinates")
				assert.Equal(t, "25000", q.Get("radius"), "radius")
				assert.Equal(t, "10", q.Get("limit"), "limit")
				assert.Equal(t, "test-api-key-12345", r.Header.Get("X-API-Key"), "X-API-Key header")
			},
		},
		{
			name: "negative coordinates are formatted correctly",
			lat:  -33.8688,
			long: 151.2093,
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, sensorsStub)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "-33.8688,151.2093", r.URL.Query().Get("coordinates"))
			},
		},
		{
			name: "coordinates are rounded to 4 decimal places",
			lat:  59.91334567,
			long: 10.73901234,
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, sensorsStub)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "59.9133,10.7390", r.URL.Query().Get("coordinates"))
			},
		},
		{
			name: "401 unauthorized surfaces status error",
			lat:  59.9133,
			long: 10.7390,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = io.WriteString(w, `{"message":"Invalid API key"}`)
			},
			wantErr:     true,
			errContains: "401",
		},
		{
			name: "429 rate-limited surfaces status error",
			lat:  59.9133,
			long: 10.7390,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = io.WriteString(w, `{"message":"Rate limit exceeded"}`)
			},
			wantErr:     true,
			errContains: "429",
		},
		{
			name: "500 surfaces status error",
			lat:  59.9133,
			long: 10.7390,
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
			wantErr:     true,
			errContains: "500",
		},
		{
			name: "malformed JSON surfaces decode error",
			lat:  59.9133,
			long: 10.7390,
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{not-json`)
			},
			wantErr:     true,
			errContains: "decoding response",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var captured *http.Request
			handler := func(w http.ResponseWriter, r *http.Request) {
				captured = r
				tc.handler(w, r)
			}

			client := newStubbedClient(t, handler)

			got, err := client.FetchSensors(context.Background(), tc.lat, tc.long)

			if tc.wantErr {
				require.Error(t, err, "expected error, got result=%+v", got)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			if tc.requestCheck != nil {
				require.NotNil(t, captured, "handler never received a request")
				tc.requestCheck(t, captured)
			}
		})
	}
}

func TestFetchLatest(t *testing.T) {
	latestStub := loadStubFixture(t, "aqlocationStub.json")

	type requestAssertion func(t *testing.T, r *http.Request)

	tests := []struct {
		name         string
		locationID   int
		handler      http.HandlerFunc
		requestCheck requestAssertion
		wantErr      bool
		errContains  string
	}{
		{
			name:       "returns latest readings",
			locationID: 2178,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, latestStub)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method, "method")
				assert.True(t, strings.HasSuffix(r.URL.Path, "/locations/2178/latest"),
					"path %q should end with /locations/2178/latest", r.URL.Path)
				assert.Equal(t, "test-api-key-12345", r.Header.Get("X-API-Key"), "X-API-Key header")
			},
		},
		{
			name:       "different location id propagates into path",
			locationID: 99999,
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, latestStub)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				assert.True(t, strings.HasSuffix(r.URL.Path, "/locations/99999/latest"),
					"path %q should end with /locations/99999/latest", r.URL.Path)
			},
		},
		{
			name:       "zero location id still builds a path",
			locationID: 0,
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, latestStub)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				assert.True(t, strings.HasSuffix(r.URL.Path, "/locations/0/latest"),
					"path %q should end with /locations/0/latest", r.URL.Path)
			},
		},
		{
			name:       "404 for unknown location surfaces status error",
			locationID: 123456789,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"message":"Not Found"}`)
			},
			wantErr:     true,
			errContains: "404",
		},
		{
			name:       "error includes location id in wrapping context",
			locationID: 2178,
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
			wantErr:     true,
			errContains: "location 2178",
		},
		{
			name:       "malformed JSON surfaces decode error",
			locationID: 2178,
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{not-json`)
			},
			wantErr:     true,
			errContains: "decoding response",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var captured *http.Request
			handler := func(w http.ResponseWriter, r *http.Request) {
				captured = r
				tc.handler(w, r)
			}

			client := newStubbedClient(t, handler)

			got, err := client.FetchLatest(context.Background(), tc.locationID)

			if tc.wantErr {
				require.Error(t, err, "expected error, got result=%+v", got)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			if tc.requestCheck != nil {
				require.NotNil(t, captured, "handler never received a request")
				tc.requestCheck(t, captured)
			}
		})
	}
}

// TestAQClient_RoutingMux verifies both endpoints work against one server
// when handled by a mux — closer to how the two calls compose in production
// (find sensors, then fetch latest for one of them).
func TestAQClient_RoutingMux(t *testing.T) {
	sensorsStub := loadStubFixture(t, "aqSensorStub.json")
	latestStub := loadStubFixture(t, "aqlocationStub.json")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Match by suffix so test stays stable even if OPENAQ_API includes a version prefix.
		if strings.HasSuffix(r.URL.Path, "/latest") {
			_, _ = io.WriteString(w, latestStub)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/locations") {
			_, _ = io.WriteString(w, sensorsStub)
			return
		}
		http.NotFound(w, r)
	})

	client := newStubbedClient(t, mux.ServeHTTP)

	sensors, err := client.FetchSensors(context.Background(), 59.9133, 10.7390)
	require.NoError(t, err)
	require.NotNil(t, sensors)

	latest, err := client.FetchLatest(context.Background(), 2178)
	require.NoError(t, err)
	require.NotNil(t, latest)
}

// TestAQClient_CancelledContext ensures a cancelled context aborts
// the request instead of reaching the upstream.
func TestAQClient_CancelledContext(t *testing.T) {
	client := newStubbedClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Log("handler reached; ctx cancellation may have raced")
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, errSensors := client.FetchSensors(ctx, 59.9133, 10.7390)
	assert.Error(t, errSensors, "FetchSensors should error on cancelled context")

	_, errLatest := client.FetchLatest(ctx, 2178)
	assert.Error(t, errLatest, "FetchLatest should error on cancelled context")
}

// TestOpenAqUrl exercises the sensors URL builder directly.
func TestOpenAqUrl(t *testing.T) {
	tests := []struct {
		name       string
		lat        float64
		lon        float64
		wantCoords string
	}{
		{name: "Oslo", lat: 59.9133, lon: 10.7390, wantCoords: "59.9133,10.7390"},
		{name: "Sydney (negative lat)", lat: -33.8688, lon: 151.2093, wantCoords: "-33.8688,151.2093"},
		{name: "Null Island", lat: 0, lon: 0, wantCoords: "0.0000,0.0000"},
		{name: "rounds excess precision", lat: 12.3456789, lon: -98.7654321, wantCoords: "12.3457,-98.7654"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := openAqUrl(tc.lat, tc.lon)
			require.NoError(t, err)

			u, err := url.Parse(got)
			require.NoError(t, err, "built URL is not parseable")

			assert.True(t, strings.HasSuffix(u.Path, "/locations"),
				"path %q should end with /locations", u.Path)
			q := u.Query()
			assert.Equal(t, tc.wantCoords, q.Get("coordinates"), "coordinates")
			assert.Equal(t, "25000", q.Get("radius"), "radius")
			assert.Equal(t, "10", q.Get("limit"), "limit")
			assert.NotEmpty(t, u.Host, "URL should have a host")
			assert.NotEmpty(t, u.Scheme, "URL should have a scheme")
		})
	}
}

// TestLatestUrl exercises the latest-readings URL builder directly.
func TestLatestUrl(t *testing.T) {
	tests := []struct {
		name       string
		locationID int
		wantTail   string
	}{
		{name: "typical id", locationID: 2178, wantTail: "/locations/2178/latest"},
		{name: "large id", locationID: 99999999, wantTail: "/locations/99999999/latest"},
		{name: "zero id", locationID: 0, wantTail: "/locations/0/latest"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := latestUrl(tc.locationID)
			require.NoError(t, err)

			u, err := url.Parse(got)
			require.NoError(t, err, "built URL is not parseable")

			assert.True(t, strings.HasSuffix(u.Path, tc.wantTail),
				"path %q should end with %q", u.Path, tc.wantTail)
			assert.NotEmpty(t, u.Host, "URL should have a host")
			assert.NotEmpty(t, u.Scheme, "URL should have a scheme")
		})
	}
}
