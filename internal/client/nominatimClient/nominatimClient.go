package nominatimClient

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// NomClient
// holds the nominatim client for Nominatim API
type NomClient struct {
	httpClient *http.Client
}

// NewNomClient
// Creates a new nominatim client
func NewNomClient() *NomClient {
	return &NomClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// FetchCapitalCoords
// Uses Nominatim to get the coordinates of a capital city
func (n *NomClient) FetchCapitalCoords(capital string) (*structs.NomIncoming, error) {
	time.Sleep(time.Second) // sleep 1s, nominatim has 1req/sec limit

	req, err := requestNom(capital)
	if err != nil {
		return nil, err
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nominatim returned status %d", resp.StatusCode)
	}

	var result []structs.NomIncoming
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result) == 0 { //check empty result
		return nil, fmt.Errorf("no results found for capital: %s", capital)
	}

	return &result[0], nil
}

// requestNom
// creates a request using the nomUrl with a custom header
func requestNom(capital string) (*http.Request, error) {
	req, err := http.NewRequest("GET", nomUrl(capital), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "@cloud2-group23") //todo change
	return req, nil
}

// nomUrl
// creates the url to get the coordinates
func nomUrl(capital string) string {
	urlCreated, _ := url.Parse(config.NOMINATIM_API + "/search")
	q := urlCreated.Query()
	q.Set("q", capital)
	q.Set("format", "json")
	q.Set("limit", "1")
	urlCreated.RawQuery = q.Encode()
	return urlCreated.String()
}
