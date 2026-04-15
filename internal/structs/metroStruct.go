package structs

type MetroAPIIncoming struct {
	Daily struct {
		Temperature2mMean []float64 `json:"temperature_2m_mean"`
		PrecipitationSum  []float64 `json:"precipitation_sum"`
	} `json:"daily"`
}

type MetroResponse struct {
	MeanTemperature   float64
	MeanPrecipitation float64
}
