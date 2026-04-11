package currency

import (
	"encoding/json"
	"envdash/internal/config"
	"envdash/internal/structs"
	"net/http"
	"time"
)

// CurrencyClient
// holds the currency client
type CurrencyClient struct {
	httpClient *http.Client
}

// NewCurrencyClient
// Creates a new currency client
func NewCurrencyClient() *CurrencyClient {
	return &CurrencyClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// FetchExchangeRates
// Gets only the exchange rate information for a given base currency
// pass error upstream if there is one
func (cur *CurrencyClient) FetchExchangeRates(baseCur string) (*structs.IncomingCurrency, error) {

	//gets the exchange rates for the input country
	//returns an error if there is an error
	res, err := cur.httpClient.Get(config.CURRENCIES_API + baseCur)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var rates *structs.IncomingCurrency
	err2 := json.NewDecoder(res.Body).Decode(&rates)
	if err2 != nil {
		return nil, err
	}

	//returns the map with the exchange rates
	return rates, nil
}
