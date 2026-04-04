package structs

// Response Struct for the status
type StatusResponse struct {
	CountriesApi int    `json:"restCountriesApi"`
	MetroAPI     int    `json:"openMetroApi"`
	AqAPI        int    `json:"openAqApi"`
	Nominatim    int    `json:"nominatimApi"`
	CurrencyAPI  int    `json:"currencyApi"`
	Db_noti      int    `json:"notification_db"`
	Version      string `json:"version"`
	Uptime       string `json:"uptime"`
}

type DashboardResponse struct {
	Country       string   `json:"country"`
	IsoCode       string   `json:"isoCode"`
	Features      Features `json:"features"`
	LastRetrieval string   `json:"lastRetrieval"`
}

type Features struct {
	Temperature      string             `json:"temperature"`
	Precipitation    string             `json:"precipitation"`
	AirQuality       AqResponse         `json:"airQuality"`
	Capital          string             `json:"capital"`
	Coordinates      CoordinateDetails  `json:"coordinates"`
	Population       int                `json:"population"`
	Area             float64            `json:"area"`
	TargetCurrencies map[string]float64 `json:"targetCurrencies"`
}

type CoordinateDetails struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
