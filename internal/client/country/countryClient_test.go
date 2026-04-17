package country

import (
	"context"
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/structs"
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
// CountryClient whose HTTP transport routes all requests to that server.
func newStubbedClient(t *testing.T, handler http.HandlerFunc) *CountryClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	target, err := url.Parse(srv.URL)
	require.NoError(t, err)

	return &CountryClient{
		httpClient: &http.Client{
			Timeout:   5 * time.Second,
			Transport: &rewriteTransport{target: target},
		},
	}
}

func TestFetchCountryData(t *testing.T) {
	norwayStub := loadStubFixture(t, "countryNorwayStub.json")
	swedenStub := loadStubFixture(t, "countrySeStub.json")

	type requestAssertion func(t *testing.T, r *http.Request)
	type resultAssertion func(t *testing.T, got []structs.IncomingCountry)

	tests := []struct {
		name         string
		countryCode  string
		handler      http.HandlerFunc
		requestCheck requestAssertion
		resultCheck  resultAssertion
		wantErr      bool
		errContains  string
	}{
		{
			name:        "Returns Norway data",
			countryCode: "NO",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, norwayStub)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method, "method")
				// The country code must be the final path segment and the
				// alpha subpath must appear in the URL.
				assert.True(t, strings.HasSuffix(r.URL.Path, "/NO"),
					"path %q should end with /NO", r.URL.Path)
				assert.Contains(t, r.URL.Path, config.PATH_REST_ALPHA,
					"path should include alpha subpath")
			},
			resultCheck: func(t *testing.T, got []structs.IncomingCountry) {
				require.Len(t, got, 1)
				assert.Equal(t, "Norway", got[0].Name.Common)
				assert.Equal(t, "NO", got[0].IsoCode)
				assert.Equal(t, []string{"Oslo"}, got[0].Capital)
			},
		},
		{
			name:        "Returns Sweden data",
			countryCode: "SE",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, swedenStub)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				assert.True(t, strings.HasSuffix(r.URL.Path, "/SE"),
					"path %q should end with /SE", r.URL.Path)
			},
			resultCheck: func(t *testing.T, got []structs.IncomingCountry) {
				require.Len(t, got, 1)
				assert.Equal(t, "Sweden", got[0].Name.Common)
				assert.Equal(t, "SE", got[0].IsoCode)
				assert.Equal(t, []string{"Stockholm"}, got[0].Capital)
			},
		},
		{
			name:        "empty array decodes successfully to empty slice",
			countryCode: "NO",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `[]`)
			},
			resultCheck: func(t *testing.T, got []structs.IncomingCountry) {
				assert.Empty(t, got, "expected empty slice for empty array response")
			},
		},
		{
			name:        "404 returns status error with body",
			countryCode: "XX",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"status":404,"message":"Not Found"}`)
			},
			wantErr:     true,
			errContains: "404",
		},
		{
			name:        "500 returns status error",
			countryCode: "NO",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
			wantErr:     true,
			errContains: "500",
		},
		{
			name:        "malformed JSON surfaces decode error",
			countryCode: "NO",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{not-json`)
			},
			wantErr:     true,
			errContains: "decoding country response",
		},
		{
			name:        "object instead of array surfaces decode error",
			countryCode: "NO",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Note: API is expected to return an array; a bare object should fail.
				_, _ = io.WriteString(w, `{"name":{"common":"Norway"}}`)
			},
			wantErr:     true,
			errContains: "decoding country response",
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

			got, err := client.FetchByCode(context.Background(), tc.countryCode)

			if tc.wantErr {
				require.Error(t, err, "expected error, got result=%+v", got)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tc.resultCheck != nil {
				tc.resultCheck(t, got)
			}
			if tc.requestCheck != nil {
				require.NotNil(t, captured, "handler never received a request")
				tc.requestCheck(t, captured)
			}
		})
	}
}

// TestFetchCountryData_CancelledContext ensures a cancelled context aborts
// the request instead of reaching the upstream.
func TestFetchCountryData_CancelledContext(t *testing.T) {
	client := newStubbedClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Log("handler reached; ctx cancellation may have raced")
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.FetchByCode(ctx, "NO")
	assert.Error(t, err, "expected error from cancelled context")
}

// TestCountryUrl exercises the URL builder directly. Since we can't override
// the const base URL, we only assert on the parts the function owns: that
// the alpha subpath and country code appear correctly in the final path.
func TestCountryUrl(t *testing.T) {
	tests := []struct {
		name        string
		countryCode string
		wantTail    string
	}{
		{name: "Norway", countryCode: "NO", wantTail: "/NO"},
		{name: "Sweden", countryCode: "SE", wantTail: "/SE"},
		{name: "United States", countryCode: "US", wantTail: "/US"},
		{name: "lowercase passed through", countryCode: "no", wantTail: "/no"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := countryUrl(config.PATH_REST_ALPHA, tc.countryCode)
			require.NoError(t, err)

			u, err := url.Parse(got)
			require.NoError(t, err, "built URL is not parseable")

			assert.True(t, strings.HasSuffix(u.Path, tc.wantTail),
				"path %q should end with %q", u.Path, tc.wantTail)
			assert.Contains(t, u.Path, config.PATH_REST_ALPHA,
				"path should include the alpha subpath from config")
			assert.NotEmpty(t, u.Host, "URL should have a host")
			assert.NotEmpty(t, u.Scheme, "URL should have a scheme")
		})
	}
}
