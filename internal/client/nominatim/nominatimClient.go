package nominatim

import (
	"context"
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"
)

// NomClient
// holds the nominatim client for Nominatim API
type NomClient struct {
	httpClient *http.Client
	limiter    *rate.Limiter
}

// NewNomClient
// Creates a new nominatim client
func NewNomClient() *NomClient {
	return &NomClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		limiter:    rate.NewLimiter(rate.Every(time.Second), 1), //rate limit 1req/s, as per Nominatim doc
	}
}

// FetchCapitalCoords
// Uses Nominatim to get the coordinates of a capital city
func (n *NomClient) FetchCapitalCoords(ctx context.Context, capital string) (*structs.NomIncoming, error) {
	err := n.limiter.Wait(ctx)
	if err != nil {
		return nil, err
	}

	req, err := requestNom(ctx, capital)
	if err != nil {
		return nil, err
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nominatim request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nominatim returned status %d", resp.StatusCode)
	}

	var result []structs.NomIncoming
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding Nominatim: %w", err)
	}

	if len(result) == 0 { //check empty result
		return nil, fmt.Errorf("no results found for capital: %s", capital)
	}

	return &result[0], nil
}

// requestNom
// creates a request using the nomUrl with a custom header
func requestNom(ctx context.Context, capital string) (*http.Request, error) {
	u, err := nomUrl(capital)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "@cloud2-group23")
	return req, nil
}

// nomUrl
// creates the url to get the coordinates
func nomUrl(capital string) (string, error) {
	urlCreated, err := url.ParseRequestURI(config.NOMINATIM_API + "/search")
	if err != nil {
		return "", fmt.Errorf("invalid nominatim base URL: %w", err)
	}
	q := urlCreated.Query()
	q.Set("q", capital)
	q.Set("format", "json")
	q.Set("limit", "1")
	urlCreated.RawQuery = q.Encode()
	return urlCreated.String(), nil
}
