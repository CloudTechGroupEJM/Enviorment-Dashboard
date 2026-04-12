package aq

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

// AQClient
// holds the air quality client for OpenAQ API
type AQClient struct {
	httpClient *http.Client
}

// NewAQClient
// Creates a new air quality client
func NewAQClient() *AQClient {
	return &AQClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// FetchSensors
// fetches the sensors  for a given latitude and longitude
func (aq *AQClient) FetchSensors(lat float64, long float64) (*structs.AirQualityIncoming, error) {
	req, err := requestAQ(openAqUrl(lat, long))
	if err != nil {
		return nil, err
	}

	resp, err := aq.httpClient.Do(req) // gets the client to do the request
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openaq returned status %d for url: %s", resp.StatusCode, resp.Request.URL)
	}

	var locations structs.AirQualityIncoming
	if err := json.NewDecoder(resp.Body).Decode(&locations); err != nil {
		return nil, err
	}

	return &locations, nil //returns struct with the information associated with the coordinates
}

// FetchLatest
// fetches the latest sensor readings for a location ID.
// used to get the actual air quality data
func (aq *AQClient) FetchLatest(locationID int) (*structs.LatestIncoming, error) {
	urlLatestData := fmt.Sprintf("%s/locations/%d/latest", config.OPENAQ_API, locationID)

	req, err := requestAQ(urlLatestData)
	if err != nil {
		return nil, err
	}

	resp, err := aq.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openaq /latest returned status %d", resp.StatusCode)
	}

	var latest structs.LatestIncoming
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		return nil, err
	}
	return &latest, nil
}

// requestAQ
// creates a new GET request with the custom header needed for the endpoint
func requestAQ(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", os.Getenv("OPEN_AQ_API_KEY"))
	return req, nil
}

// openAqUrl
// creates the url used to the sensors
func openAqUrl(lat, lon float64) string {
	urlCreated, _ := url.Parse(os.Getenv("OPEN_AQ_API_KEY") + "/locations")
	q := urlCreated.Query()
	q.Set("coordinates", fmt.Sprintf("%.4f,%.4f", lat, lon))
	q.Set("radius", "25000")
	q.Set("limit", "100")
	urlCreated.RawQuery = q.Encode()
	log.Println(urlCreated.String())
	return urlCreated.String()
}
