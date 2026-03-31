package structs

type RegisterCountry struct {
	Name struct {
		Common string `json:"common"`
	} `json:"name"`

	IsoCode string `json:"isoCode"`

	Features struct {
		Temperature      bool     `json:"temperature"`
		Precipitation    bool     `json:"precipitation"`
		AirQuality       bool     `json:"airQuality"`
		Capital          bool     `json:"capital"`
		Coordinates      bool     `json:"coordinates"`
		Area             bool     `json:"area"`
		TargetCurrencies []string `json:"targetCurrencies"`
	} `json:"features"`
}


type RegistrationComplete struct{
	ID string `json:"id"`
	LastChange string `json:"lastChange"`
}
