package structs

import "time"

type RegisterCountry struct {
	// ID         string    `firestore:"id,omitempty" json:"id"`
	Name       string    `firestore:"name" json:"name"`
	IsoCode    string    `firesstore:"isoCode" json:"isoCode"`
	Features   Features  `firestore:"features" json:"features"`
	LastChange time.Time `firestore:"lastChange" json:"lastChange"`
}

type Features struct {
	Temperature      bool     `firesstore:"temperature" json:"temperature"`
	Precipitation    bool     `firesstore:"precipitation" json:"precipitation"`
	AirQuality       bool     `firesstore:"airQuality" json:"airQuality"`
	Capital          bool     `firesstore:"capital" json:"capital"`
	Coordinates      bool     `firesstore:"coordinates" json:"coordinates"`
	Area             bool     `firesstore:"area" json:"area"`
	TargetCurrencies []string `firesstore:"targetCurrencies" json:"targetCurrencies"`
}
