package client

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
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
func (mc *MetroClient) FetchMetroData(lat float64, long float64) (*structs.MetroAPIIncoming, error) {

	resp, err := mc.httpClient.Get(metroUrl(lat, long))
	if err != nil {
		return nil, fmt.Errorf("error metro info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error, likely lat, long: status code %d", resp.StatusCode)
	}

	var metroApi structs.MetroAPIIncoming
	if err := json.NewDecoder(resp.Body).Decode(&metroApi); err != nil {
		return nil, fmt.Errorf("error parsing metro info: %w", err)
	}
	return &metroApi, nil
}

// metroUrl
// Creates the URL for the metro api
// todo: handle "wrong lat, long in the handler"
func metroUrl(lat float64, long float64) string {
	urlCreated, _ := url.Parse(config.METRO_API)
	q := urlCreated.Query()
	q.Set("latitude", strconv.FormatFloat(lat, 'f', 4, 64))
	q.Set("longitude", strconv.FormatFloat(long, 'f', 4, 64))
	q.Set("daily", "temperature_2m_mean,precipitation_sum")
	q.Set("temperature_unit", "celsius")
	q.Set("forecast_days", "7")
	q.Set("timezone", "UTC")
	urlCreated.RawQuery = q.Encode()
	return urlCreated.String()
}
