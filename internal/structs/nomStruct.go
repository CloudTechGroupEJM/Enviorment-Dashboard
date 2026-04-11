package structs

type NomIncoming struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

type NomResponse struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}
