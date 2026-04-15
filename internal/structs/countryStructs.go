package structs

type IncomingCountry struct {
	Name struct { //todo: dont think we need these, as you get both name and iso code from user in registration
		Common string `json:"common"`
	} `json:"name"`
	IsoCode     string              `json:"cca2"`
	Capital     []string            `json:"capital"`
	Coordinates []float64           `json:"latlng"`
	Population  int                 `json:"population"`
	Area        float64             `json:"area"`
	Currencies  map[string]struct{} `json:"currencies"`
}
