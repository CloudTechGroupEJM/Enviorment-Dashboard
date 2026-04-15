package structs

// v3/locations response
type AirQualityIncoming struct {
	Results []struct {
		ID      int `json:"id"`
		Sensors []struct {
			ID        int `json:"id"`
			Parameter struct {
				Name string `json:"name"` // "pm25", "pm10"
			} `json:"parameter"`
		} `json:"sensors"`
	} `json:"results"`
}

// v3/locations/{id}/latest response
type LatestIncoming struct {
	Results []struct {
		Value     float64 `json:"value"`
		SensorsID int     `json:"sensorsId"`
		Parameter string  `json:"parameter"`
	} `json:"results"`
}

// Represents the outgoing air quality data to be used in dashboard response
type AqResponse struct {
	PM25  float64 `json:"pm25"`
	PM10  float64 `json:"pm10"`
	Level string  `json:"level"`
}
