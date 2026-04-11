package status

import (
	"net/http"
	"time"
)

// provides HTTP functionaltiy
type StatusClient struct {
	httpClient *http.Client
}

// NewStatusClient
// Creates a status client to execute HTTP calls.
func NewStatusClient() *StatusClient {
	return &StatusClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// ProbeHeadEndpoint
// sends a head request to the endpoint to get the status code
// only useful for endpoints that support it
func (sc *StatusClient) ProbeHeadEndpoint(url string) (int, error) {
	resp, err := sc.httpClient.Head(url)
	if err != nil {
		return http.StatusServiceUnavailable, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

// ProbeGetEndpoint
// gets the status code, with configurable header key and header value
// some endpoint need this to allow for usage
func (sc *StatusClient) ProbeGetEndpoint(url string, headerKey string, headerValue string) int {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return http.StatusServiceUnavailable
	}

	if headerValue != "" {
		req.Header.Set("X-API-Key", headerValue)
	}

	if headerKey != "" {
		req.Header.Set(headerKey, headerValue)
	}

	res, err := sc.httpClient.Do(req)
	if err != nil {
		return http.StatusServiceUnavailable
	}
	defer res.Body.Close()

	return res.StatusCode
}
