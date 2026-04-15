package structs

type DashboardResponse struct {
	Country       string    `json:"country"`
	IsoCode       string    `json:"isoCode"`
	Features      *Features `json:"features,omitempty"`
	LastRetrieval string    `json:"lastRetrieval"`
}

type Features struct {
	Temperature      float64            `json:"temperature,omitempty"`
	Precipitation    float64            `json:"precipitation,omitempty"`
	AirQuality       *AqResponse        `json:"airQuality,omitempty"`
	Capital          string             `json:"capital,omitempty"`
	Coordinates      *CoordinateDetails `json:"coordinates,omitempty"`
	Population       int                `json:"population,omitempty"`
	Area             float64            `json:"area,omitempty"`
	TargetCurrencies map[string]float64 `json:"targetCurrencies,omitempty"`
}

type CoordinateDetails struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
