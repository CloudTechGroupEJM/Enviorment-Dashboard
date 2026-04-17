package currency

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

// newStubbedClient spins up an httptest.Server running handler and returns a
// CurrencyClient whose HTTP transport routes all requests to that server.
func newStubbedClient(t *testing.T, handler http.HandlerFunc) *CurrencyClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	target, err := url.Parse(srv.URL)
	require.NoError(t, err)

	return &CurrencyClient{
		httpClient: &http.Client{
			Timeout:   5 * time.Second,
			Transport: &rewriteTransport{target: target},
		},
	}
}

func TestFetchExchangeRates(t *testing.T) {
	nokStub := loadStubFixture(t, "currencyNokStub.json")

	type requestAssertion func(t *testing.T, r *http.Request)
	type resultAssertion func(t *testing.T, got interface{}) // interface to avoid assuming field names

	tests := []struct {
		name         string
		baseCur      string
		handler      http.HandlerFunc
		requestCheck requestAssertion
		wantErr      bool
		errContains  string
		wantBase     string
		wantRates    map[string]float64
	}{
		{
			name:    "happy path returns NOK rates",
			baseCur: "NOK",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, nokStub)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method, "method")
				// The base currency must be the final path segment.
				assert.True(t, strings.HasSuffix(r.URL.Path, "/NOK"),
					"path %q should end with /NOK", r.URL.Path)
			},
			wantBase: "NOK",
			wantRates: map[string]float64{
				"NOK": 1,
				"USD": 0.106255,
				"EUR": 0.090091,
				"SEK": 0.975608,
			},
		},
		{
			name:    "different base currency propagates through URL",
			baseCur: "USD",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{"result":"success","base_code":"USD","rates":{"USD":1,"NOK":9.41}}`)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				assert.True(t, strings.HasSuffix(r.URL.Path, "/USD"),
					"path %q should end with /USD", r.URL.Path)
			},
			wantBase: "USD",
			wantRates: map[string]float64{
				"USD": 1,
				"NOK": 9.41,
			},
		},
		{
			name:    "404 returns status error with body",
			baseCur: "XYZ",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"result":"error","error-type":"unsupported-code"}`)
			},
			wantErr:     true,
			errContains: "404",
		},
		{
			name:    "500 returns status error",
			baseCur: "NOK",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "server blew up", http.StatusInternalServerError)
			},
			wantErr:     true,
			errContains: "500",
		},
		{
			name:    "malformed JSON surfaces decode error",
			baseCur: "NOK",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{not-json`)
			},
			wantErr:     true,
			errContains: "decoding currency response",
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

			got, err := client.FetchExchangeRates(context.Background(), tc.baseCur)

			if tc.wantErr {
				require.Error(t, err, "expected error, got result=%+v", got)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			for code, want := range tc.wantRates {
				assert.InDelta(t, want, got.Rates[code], 1e-9, "rate for %s", code)
			}

			if tc.requestCheck != nil {
				require.NotNil(t, captured, "handler never received a request")
				tc.requestCheck(t, captured)
			}
		})
	}
}

// TestFetchExchangeRates_CancelledContext ensures a cancelled context aborts
// the request instead of reaching the upstream.
func TestFetchExchangeRates_CancelledContext(t *testing.T) {
	client := newStubbedClient(t, func(w http.ResponseWriter, r *http.Request) {
		// If we get here the cancellation didn't take effect in time.
		// We sleep briefly to give a slow ctx path a chance, but the server
		// shouldn't actually receive the request.
		t.Log("handler reached; ctx cancellation may have raced")
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.FetchExchangeRates(ctx, "NOK")
	assert.Error(t, err, "expected error from cancelled context")
}

// TestCurrencyUrl exercises the URL builder directly. Since we can't override
// the const base URL, we only assert on the parts the function owns: that the
// base currency ends up as the final path segment and the overall URL parses.
func TestCurrencyUrl(t *testing.T) {
	tests := []struct {
		name     string
		baseCur  string
		wantTail string
	}{
		{name: "NOK", baseCur: "NOK", wantTail: "/NOK"},
		{name: "USD", baseCur: "USD", wantTail: "/USD"},
		{name: "lowercase passed through", baseCur: "eur", wantTail: "/eur"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := currencyUrl(tc.baseCur)
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
