package nominatim

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
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

// newStubbedClient spins up an httptest.Server running handler and returns a
// NomClient whose HTTP transport routes all requests to that server. The
// limiter is set to rate.Inf so tests aren't throttled to 1 req/s.
func newStubbedClient(t *testing.T, handler http.HandlerFunc) *NomClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	target, err := url.Parse(srv.URL)
	require.NoError(t, err)

	return &NomClient{
		httpClient: &http.Client{
			Timeout:   5 * time.Second,
			Transport: &rewriteTransport{target: target},
		},
		limiter: rate.NewLimiter(rate.Inf, 1),
	}
}

func TestFetchCapitalCoords(t *testing.T) {
	osloStub := loadStubFixture(t, "nomiOsloStub.json")

	type requestAssertion func(t *testing.T, r *http.Request)

	tests := []struct {
		name         string
		capital      string
		handler      http.HandlerFunc
		requestCheck requestAssertion
		wantErr      bool
		errContains  string
		wantLat      string
		wantLon      string
		wantName     string
	}{
		{
			name:    "happy path returns Oslo coordinates",
			capital: "Oslo",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, osloStub)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method, "method")
				assert.Equal(t, "Oslo", r.URL.Query().Get("q"), "query param q")
				assert.Equal(t, "json", r.URL.Query().Get("format"), "query param format")
				assert.Equal(t, "1", r.URL.Query().Get("limit"), "query param limit")
				assert.Equal(t, "@cloud2-group23", r.Header.Get("User-Agent"), "User-Agent header")
			},
			wantLat:  "59.9133301",
			wantLon:  "10.7389701",
			wantName: "Oslo",
		},
		{
			name:    "capital with spaces is properly encoded",
			capital: "New York",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, osloStub) // body irrelevant; we check the query
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "New York", r.URL.Query().Get("q"))
			},
			wantLat:  "59.9133301",
			wantLon:  "10.7389701",
			wantName: "Oslo",
		},
		{
			name:    "empty result array surfaces 'no results' error",
			capital: "Atlantis",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `[]`)
			},
			wantErr:     true,
			errContains: "no results found",
		},
		{
			name:    "non-200 status returns status error",
			capital: "Oslo",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
			wantErr:     true,
			errContains: "status 500",
		},
		{
			name:    "429 Too Many Requests surfaces status error",
			capital: "Oslo",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "slow down", http.StatusTooManyRequests)
			},
			wantErr:     true,
			errContains: "status 429",
		},
		{
			name:    "malformed JSON surfaces decode error",
			capital: "Oslo",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{not-json`)
			},
			wantErr:     true,
			errContains: "error decoding",
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

			got, err := client.FetchCapitalCoords(context.Background(), tc.capital)

			if tc.wantErr {
				require.Error(t, err, "expected error, got result=%+v", got)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tc.wantLat, got.Lat, "Lat")
			assert.Equal(t, tc.wantLon, got.Lon, "Lon")
			if tc.requestCheck != nil {
				require.NotNil(t, captured, "handler never received a request")
				tc.requestCheck(t, captured)
			}
		})
	}
}

// TestFetchCapitalCoords_CancelledContext ensures the limiter respects ctx
// cancellation and never reaches the upstream. A burst=0 limiter forces Wait
// to block, and the pre-cancelled context makes it return immediately.
func TestFetchCapitalCoords_CancelledContext(t *testing.T) {
	client := newStubbedClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("upstream should not be called when ctx is cancelled before Wait returns")
	})
	// Replace the limiter with one that always blocks so cancellation is the only exit.
	client.limiter = rate.NewLimiter(rate.Every(time.Hour), 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.FetchCapitalCoords(ctx, "Oslo")
	assert.Error(t, err, "expected error from cancelled context")
}

// TestNomUrl verifies the URL builder wires all required query params. We
// can't override the base URL (it's a const), but we can still assert on the
// query string since that's what the function owns.
func TestNomUrl(t *testing.T) {
	tests := []struct {
		name    string
		capital string
		wantQ   string
	}{
		{name: "simple name", capital: "Oslo", wantQ: "Oslo"},
		{name: "name with spaces", capital: "New York", wantQ: "New York"},
		{name: "name with diacritics", capital: "São Paulo", wantQ: "São Paulo"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := nomUrl(tc.capital)
			require.NoError(t, err)

			u, err := url.Parse(got)
			require.NoError(t, err, "built URL is not parseable")

			assert.Equal(t, tc.wantQ, u.Query().Get("q"), "q")
			assert.Equal(t, "json", u.Query().Get("format"), "format")
			assert.Equal(t, "1", u.Query().Get("limit"), "limit")
		})
	}
}
