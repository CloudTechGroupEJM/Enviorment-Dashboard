package status

import (
	"log"
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
// Return status code 503 for unavailable services
func (sc *StatusClient) ProbeHeadEndpoint(url string) int {
	resp, err := sc.httpClient.Head(url)
	if err != nil {
		log.Printf("HEAD %s is down: %v", url, err)
		return http.StatusServiceUnavailable
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// ProbeGetEndpoint
// gets the status code, with configurable header key and header value
// some endpoint need this to allow for usage
func (sc *StatusClient) ProbeGetEndpoint(url string, headerKey string, headerValue string) int {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("GET %s request build failed: %v", url, err)
		return http.StatusServiceUnavailable
	}

	//sets specific header key and value if given
	if headerKey != "" && headerValue != "" {
		req.Header.Set(headerKey, headerValue)
	}

	res, err := sc.httpClient.Do(req)
	if err != nil {
		log.Printf("GET %s is down: %v", url, err)
		return http.StatusServiceUnavailable
	}
	defer res.Body.Close()

	return res.StatusCode
}
