package metro

import (
	"context"
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// MetroClient
// holds the metro client
type MetroClient struct {
	httpClient *http.Client
}

// NewMetroClient
// Creates a new metro client
func NewMetroClient() *MetroClient {
	return &MetroClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// FetchMetroData
// Gets the raw weather data response from the metro API
func (mc *MetroClient) FetchMetroData(ctx context.Context, lat float64, long float64) (*structs.MetroAPIIncoming, error) {
	u, err := metroUrl(lat, long)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("building metro request: %w", err)
	}

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching metro data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("metro returned %d: %s", resp.StatusCode, body)
	}

	var metroApi structs.MetroAPIIncoming
	if err := json.NewDecoder(resp.Body).Decode(&metroApi); err != nil {
		return nil, fmt.Errorf("decoding metro response: %w", err)
	}
	return &metroApi, nil
}

// metroUrl
// Creates the URL for the metro api
func metroUrl(lat float64, long float64) (string, error) {
	urlCreated, err := url.ParseRequestURI(config.METRO_API)
	if err != nil {
		return "", fmt.Errorf("invalid metro base URL: %w", err)
	}
	q := urlCreated.Query()
	q.Set("latitude", strconv.FormatFloat(lat, 'f', 4, 64))
	q.Set("longitude", strconv.FormatFloat(long, 'f', 4, 64))
	q.Set("daily", "temperature_2m_mean,precipitation_sum") // gets daily mean for 7 days
	q.Set("temperature_unit", "celsius")
	q.Set("forecast_days", "7")
	q.Set("timezone", "UTC")
	urlCreated.RawQuery = q.Encode()
	return urlCreated.String(), nil
}
