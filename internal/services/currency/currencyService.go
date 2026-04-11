package currency

import (
	"envdash/internal/client/currency"
	"envdash/internal/structs"
	"fmt"
)

// CurrencyInternal is the internal implementation of the currency service,
// wrapping an HTTP client for currency-related operations.
type CurrencyInternal struct {
	client *currency.CurrencyClient
}

// NewCurrencyService returns a new CurrencyInternal instance with a configured HTTP client.
func NewCurrencyService() *CurrencyInternal {
	return &CurrencyInternal{
		client: currency.NewCurrencyClient(),
	}
}

// GetCurrency retrieves the exchange rate information for a given set of currencies, from the external currency API.
// Based on the 3-letter currency code (ISO 4217), ie ("NOK"). And some 3-letter currency codes for the target currencies.
//
// Parameters:
//   - currencyCode: 3-letter currency code (ISO 4217)
//   - target: list of  3-letter currency codes (ISO 4217) to match with
//
// Returns:
//   - *structs.CurrencyResponse: The currencies names and exchange rate that match the target
//   - error: an error if the HTTP request fails or if the response body cannot be decoded.
//   - error: if there is no base currency or targets
func (curI *CurrencyInternal) GetCurrency(currencyCode string, target []string) (*structs.CurrencyResponse, error) {
	allCur, err := curI.client.FetchExchangeRates(currencyCode)
	if err != nil {
		return nil, err
	}

	if allCur == nil || len(target) == 0 {
		return nil, fmt.Errorf("no exchange rates available")
	}

	//Claude matching algorithm
	targetSet := make(map[string]bool, len(target))
	for _, code := range target {
		targetSet[code] = true
	}

	filtered := make(map[string]float64)
	for code, rate := range allCur.Rates {
		if targetSet[code] {
			filtered[code] = rate
		}
	}

	return &structs.CurrencyResponse{TargetCurrencies: filtered}, nil
}
