package structs

// IncomingCurrency represents the raw API response
type IncomingCurrency struct {
	Rates map[string]float64 `json:"rates"`
}

// CurrencyResponse is the filtered output
type CurrencyResponse struct {
	TargetCurrencies map[string]float64 `json:"targetCurrencies"`
}
