package structs

// RegisterCountry represents a country registration with features and metadata.
// Fields: ID (document ID), Name (country name), IsoCode (ISO country code),
// Features (feature flags), LastChange (last update time)
type RegisterCountry struct {
	ID         string      `firestore:"id,omitempty" json:"id"`
	Name       string      `firestore:"name" json:"name"`
	IsoCode    string      `firestore:"isoCode" json:"isoCode"` // Fix this too if it has the typo
	Features   BoolFeature `firestore:"features" json:"features"`
	LastChange string      `firestore:"lastChange" json:"lastChange"`
}

// BoolFeature contains boolean flags for available country features and target currencies.
// Fields: Temperature, Precipitation, AirQuality, Capital, Coordinates,
// Area (all bool), TargetCurrencies (currency codes)
type BoolFeature struct {
	Temperature      bool     `firestore:"temperature" json:"temperature"`
	Precipitation    bool     `firestore:"precipitation" json:"precipitation"`
	Population       bool     `firestore:"population" json:"population"`
	AirQuality       bool     `firestore:"airQuality" json:"airQuality"`
	Capital          bool     `firestore:"capital" json:"capital"`
	Coordinates      bool     `firestore:"coordinates" json:"coordinates"`
	Area             bool     `firestore:"area" json:"area"`
	TargetCurrencies []string `firestore:"targetCurrencies" json:"targetCurrencies"`
}
