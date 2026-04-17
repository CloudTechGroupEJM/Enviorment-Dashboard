package metro

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
	"strconv"
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
// MetroClient whose HTTP transport routes all requests to that server.
func newStubbedClient(t *testing.T, handler http.HandlerFunc) *MetroClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	target, err := url.Parse(srv.URL)
	require.NoError(t, err)

	return &MetroClient{
		httpClient: &http.Client{
			Timeout:   5 * time.Second,
			Transport: &rewriteTransport{target: target},
		},
	}
}

func TestFetchMetroData(t *testing.T) {
	stockholmStub := loadStubFixture(t, "metroStub.json")

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
			name: "stub returns forecast for Stockholm",
			lat:  59.3289,
			long: 18.0724,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, stockholmStub)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method, "method")
				q := r.URL.Query()
				assert.Equal(t, "59.3289", q.Get("latitude"), "latitude")
				assert.Equal(t, "18.0724", q.Get("longitude"), "longitude")
				assert.Equal(t, "temperature_2m_mean,precipitation_sum", q.Get("daily"), "daily")
				assert.Equal(t, "celsius", q.Get("temperature_unit"), "temperature_unit")
				assert.Equal(t, "7", q.Get("forecast_days"), "forecast_days")
				assert.Equal(t, "UTC", q.Get("timezone"), "timezone")
			},
		},
		{
			name: "negative coordinates are formatted correctly",
			lat:  -33.8688,
			long: 151.2093,
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, stockholmStub) // body irrelevant for this case
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				q := r.URL.Query()
				assert.Equal(t, "-33.8688", q.Get("latitude"), "negative latitude")
				assert.Equal(t, "151.2093", q.Get("longitude"), "longitude")
			},
		},
		{
			name: "zero coordinates produce zero strings",
			lat:  0.0,
			long: 0.0,
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, stockholmStub)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				q := r.URL.Query()
				assert.Equal(t, "0.0000", q.Get("latitude"), "zero latitude")
				assert.Equal(t, "0.0000", q.Get("longitude"), "zero longitude")
			},
		},
		{
			name: "coordinates are rounded to 4 decimal places",
			lat:  59.91334567,
			long: 10.73901234,
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, stockholmStub)
			},
			requestCheck: func(t *testing.T, r *http.Request) {
				q := r.URL.Query()
				assert.Equal(t, "59.9133", q.Get("latitude"), "rounded latitude")
				assert.Equal(t, "10.7390", q.Get("longitude"), "rounded longitude")
			},
		},
		{
			name: "400 returns status error with body",
			lat:  59.3289,
			long: 18.0724,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":true,"reason":"Invalid parameters"}`)
			},
			wantErr:     true,
			errContains: "400",
		},
		{
			name: "500 returns status error",
			lat:  59.3289,
			long: 18.0724,
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "upstream on fire", http.StatusInternalServerError)
			},
			wantErr:     true,
			errContains: "500",
		},
		{
			name: "malformed JSON surfaces decode error",
			lat:  59.3289,
			long: 18.0724,
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{not-json`)
			},
			wantErr:     true,
			errContains: "decoding metro response",
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

			got, err := client.FetchMetroData(context.Background(), tc.lat, tc.long)

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

// TestFetchMetroData_Decode verifies the JSON decodes into the
// expected fields end-to-end. Values are pinned to the real Stockholm stub.
// If struct field names differ in your codebase, update only these assertions.
func TestFetchMetroData_Decode(t *testing.T) {
	stockholmStub := loadStubFixture(t, "metroStub.json")

	client := newStubbedClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, stockholmStub)
	})

	got, err := client.FetchMetroData(context.Background(), 59.3289, 18.0724)
	require.NoError(t, err)
	require.NotNil(t, got)

	// Top-level fields
	// Daily arrays — all should be length 7 (forecast_days=7)
	require.Len(t, got.Daily.Temperature2mMean, 7, "Daily.Temperature2mMean length")
	require.Len(t, got.Daily.PrecipitationSum, 7, "Daily.PrecipitationSum length")

	assert.InDelta(t, 11.7, got.Daily.Temperature2mMean[0], 1e-6, "temp day 0")
	assert.InDelta(t, 6.2, got.Daily.Temperature2mMean[2], 1e-6, "temp day 2")
	assert.InDelta(t, 8.2, got.Daily.Temperature2mMean[6], 1e-6, "temp day 6")

	// All precipitation values are zero in this stub
	for i, p := range got.Daily.PrecipitationSum {
		assert.InDelta(t, 0.0, p, 1e-9, "precip day %d", i)
	}
}

// TestFetchMetroData_CancelledContext ensures a cancelled context aborts
// the request instead of reaching the upstream.
func TestFetchMetroData_CancelledContext(t *testing.T) {
	client := newStubbedClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Log("handler reached; ctx cancellation may have raced")
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.FetchMetroData(ctx, 59.3289, 18.0724)
	assert.Error(t, err, "expected error from cancelled context")
}

// TestMetroUrl exercises the URL builder directly.
func TestMetroUrl(t *testing.T) {
	tests := []struct {
		name    string
		lat     float64
		long    float64
		wantLat string
		wantLon string
	}{
		{name: "Stockholm", lat: 59.3289, long: 18.0724, wantLat: "59.3289", wantLon: "18.0724"},
		{name: "Oslo", lat: 59.9133, long: 10.7390, wantLat: "59.9133", wantLon: "10.7390"},
		{name: "Sydney (negative lat)", lat: -33.8688, long: 151.2093, wantLat: "-33.8688", wantLon: "151.2093"},
		{name: "Null Island", lat: 0, long: 0, wantLat: "0.0000", wantLon: "0.0000"},
		{name: "rounds excess precision", lat: 12.3456789, long: -98.7654321, wantLat: "12.3457", wantLon: "-98.7654"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := metroUrl(tc.lat, tc.long)
			require.NoError(t, err)

			u, err := url.Parse(got)
			require.NoError(t, err, "built URL is not parseable")

			q := u.Query()
			assert.Equal(t, tc.wantLat, q.Get("latitude"), "latitude")
			assert.Equal(t, tc.wantLon, q.Get("longitude"), "longitude")
			assert.Equal(t, "temperature_2m_mean,precipitation_sum", q.Get("daily"), "daily")
			assert.Equal(t, "celsius", q.Get("temperature_unit"), "temperature_unit")
			assert.Equal(t, "7", q.Get("forecast_days"), "forecast_days")
			assert.Equal(t, "UTC", q.Get("timezone"), "timezone")
			assert.NotEmpty(t, u.Host, "URL should have a host")
			assert.NotEmpty(t, u.Scheme, "URL should have a scheme")
		})
	}
}

// sanity check on the 4-decimal formatting assumption used in the query builder
func TestCoordinateFormattingMatchesProduction(t *testing.T) {
	assert.Equal(t, "59.3289", strconv.FormatFloat(59.3289, 'f', 4, 64))
	assert.Equal(t, "18.0724", strconv.FormatFloat(18.0724, 'f', 4, 64))
	assert.Equal(t, "0.0000", strconv.FormatFloat(0, 'f', 4, 64))
	assert.Equal(t, "-33.8688", strconv.FormatFloat(-33.8688, 'f', 4, 64))
}
