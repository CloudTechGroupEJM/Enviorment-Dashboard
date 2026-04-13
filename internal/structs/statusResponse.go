package structs

// Response Struct for the status
type StatusResponse struct {
	CountriesApi int    `json:"restCountriesApi"`
	MetroAPI     int    `json:"openMetroApi"`
	AqAPI        int    `json:"openAqApi"`
	Nominatim    int    `json:"nominatimApi"`
	CurrencyAPI  int    `json:"currencyApi"`
	Db_noti      int    `json:"notification_db"`
	Webhooks     int    `json:"webhooks"`
	Version      string `json:"version"`
	Uptime       string `json:"uptime"`
}
