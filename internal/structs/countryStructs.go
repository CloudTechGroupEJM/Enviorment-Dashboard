package structs

type IncomingCountry struct {
	Name struct {
		Common string `json:"common"`
	} `json:"name"`
	IsoCode     string              `json:"cca2"`
	Capital     []string            `json:"capital"`
	Coordinates []float64           `json:"latlng"`
	Population  int                 `json:"population"`
	Area        float64             `json:"area"`
	Currencies  map[string]struct{} `json:"currencies"`
}
