package aq

import (
	"context"
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

// AQClient
// holds the air quality client for OpenAQ API
type AQClient struct {
	httpClient *http.Client
	apiKey     string
}

// NewAQClient
// Creates a new air quality client
func NewAQClient() *AQClient {
	return &AQClient{
		httpClient: &http.Client{Timeout: 3 * time.Second},
		apiKey:     os.Getenv("OPEN_AQ_API_KEY"), // dont need to reassing for every call
	}
}

// FetchSensors
// fetches the sensors for a given latitude and longitude
func (aq *AQClient) FetchSensors(ctx context.Context, lat float64, long float64) (*structs.AirQualityIncoming, error) {
	u, err := openAqUrl(lat, long)
	if err != nil {
		return nil, err
	}

	var locations structs.AirQualityIncoming
	if err := aq.getJSON(ctx, u, &locations); err != nil {
		return nil, fmt.Errorf("fetching openaq sensors: %w", err)
	}
	return &locations, nil
}

// FetchLatest
// fetches the latest sensor readings for a location ID.
// used to get the actual air quality data
func (aq *AQClient) FetchLatest(ctx context.Context, locationID int) (*structs.LatestIncoming, error) {
	u, err := latestUrl(locationID)
	if err != nil {
		return nil, err
	}

	var latest structs.LatestIncoming
	if err := aq.getJSON(ctx, u, &latest); err != nil {
		return nil, fmt.Errorf("fetching openaq latest for location %d: %w", locationID, err)
	}
	return &latest, nil
}

// getJSON performs a GET with the OpenAQ API key and decodes the response into out.
func (aq *AQClient) getJSON(ctx context.Context, u string, out any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("X-API-Key", aq.apiKey)

	resp, err := aq.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openaq returned %d: %s", resp.StatusCode, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

// openAqUrl
// creates the url used to find sensors near coordinates
func openAqUrl(lat, lon float64) (string, error) {
	u, err := url.ParseRequestURI(config.OPENAQ_API)
	if err != nil {
		return "", fmt.Errorf("invalid openaq base URL: %w", err)
	}
	u.Path = path.Join(u.Path, "locations")
	q := u.Query()
	q.Set("coordinates", fmt.Sprintf("%.4f,%.4f", lat, lon))
	q.Set("radius", "25000")
	q.Set("limit", "10")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// latestUrl
// creates the URL for the latest readings for a location
func latestUrl(locationID int) (string, error) {
	u, err := url.ParseRequestURI(config.OPENAQ_API)
	if err != nil {
		return "", fmt.Errorf("invalid openaq base URL: %w", err)
	}
	u.Path = path.Join(u.Path, "locations", fmt.Sprintf("%d", locationID), "latest")
	return u.String(), nil
}
