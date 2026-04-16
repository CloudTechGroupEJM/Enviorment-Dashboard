package status

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// redirectTransport rewrites every outbound request to hit the test server
// instead of the real upstream, regardless of the URL in config constants.
type redirectTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (r *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = r.target.Scheme
	req.URL.Host = r.target.Host
	return r.base.RoundTrip(req)
}

// withStubbedHTTP starts an httptest server that responds based on method,
// redirects DefaultTransport at it, and cleans up after the test.
func withStubbedHTTP(t *testing.T, headStatus, getStatus int) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(headStatus)
			return
		}
		w.WriteHeader(getStatus)
	}))

	target, _ := url.Parse(srv.URL)
	original := http.DefaultTransport
	http.DefaultTransport = &redirectTransport{target: target, base: original}

	t.Cleanup(func() {
		http.DefaultTransport = original
		srv.Close()
	})
}

func TestGetStatus_AllEndpointsHealthy(t *testing.T) {
	withStubbedHTTP(t, 200, 200)
	service := StatusService(time.Now(), nil)

	status := service.GetStatus()

	assert.Equal(t, "v1", status.Version)
	assert.Equal(t, 200, status.CountriesApi)
	assert.Equal(t, 200, status.MetroAPI)
	assert.Equal(t, 200, status.Nominatim)
	assert.Equal(t, 200, status.AqAPI)
	assert.Equal(t, 200, status.CurrencyAPI)
}

func TestGetStatus_AllEndpointsDown(t *testing.T) {
	withStubbedHTTP(t, 503, 503)
	service := StatusService(time.Now(), nil)

	status := service.GetStatus()

	assert.Equal(t, 503, status.CountriesApi)
	assert.Equal(t, 503, status.MetroAPI)
	assert.Equal(t, 503, status.Nominatim)
	assert.Equal(t, 503, status.AqAPI)
	assert.Equal(t, 503, status.CurrencyAPI)
}

func TestGetStatus_MixedEndpointStatuses(t *testing.T) {
	withStubbedHTTP(t, 200, 429)
	service := StatusService(time.Now(), nil)

	status := service.GetStatus()

	assert.Equal(t, 200, status.CountriesApi)
	assert.Equal(t, 200, status.MetroAPI)
	assert.Equal(t, 200, status.Nominatim)
	assert.Equal(t, 429, status.AqAPI)
	assert.Equal(t, 429, status.CurrencyAPI)
}

func TestGetStatus_UptimeIsNumeric(t *testing.T) {
	withStubbedHTTP(t, 200, 200)
	service := StatusService(time.Now().Add(-5*time.Second), nil)

	status := service.GetStatus()

	assert.Regexp(t, `^\d+$`, status.Uptime)
}

func TestGetStatus_DatabaseNotificationDefault503(t *testing.T) {
	withStubbedHTTP(t, 200, 200)
	service := StatusService(time.Now(), nil)

	status := service.GetStatus()

	assert.Equal(t, 503, status.Db_noti)
}
