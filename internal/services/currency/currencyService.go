package currency

import (
	"context"
	"envdash/internal/client/currency"
	"envdash/internal/structs"
	"fmt"
	"strings"
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
//   - ctx: request context for cancellation and timeouts
//   - currencyCode: 3-letter currency code (ISO 4217)
//   - target: list of 3-letter currency codes (ISO 4217) to match with
//
// Returns:
//   - *structs.CurrencyResponse: the currencies and exchange rates that match the target.
//     If target is empty, returns an empty map without calling the upstream API.
//   - error: if the base currency is empty, the upstream request fails,
//     or none of the requested targets are found in the response
func (curI *CurrencyInternal) GetCurrency(ctx context.Context, currencyCode string, target []string) (*structs.CurrencyResponse, error) {
	if currencyCode == "" {
		return nil, fmt.Errorf("base currency code is required")
	}

	//returns empty response immediately
	if len(target) == 0 {
		return &structs.CurrencyResponse{TargetCurrencies: map[string]float64{}}, nil
	}

	allCur, err := curI.client.FetchExchangeRates(ctx, currencyCode)
	if err != nil {
		return nil, fmt.Errorf("getting exchange rates for %s: %w", currencyCode, err)
	}

	filtered := filterRates(allCur.Rates, target)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("none of the requested currencies found in response")
	}

	return &structs.CurrencyResponse{TargetCurrencies: filtered}, nil
}

// filterRates returns the subset of rates whose currency codes appear in target.
// claude inspired filtering
func filterRates(rates map[string]float64, target []string) map[string]float64 {
	targetSet := make(map[string]struct{}, len(target))
	for _, code := range target {
		targetSet[code] = struct{}{}
		targetSet[strings.ToUpper(code)] = struct{}{}
	}

	filtered := make(map[string]float64)
	for code, rate := range rates {
		upperCode := strings.ToUpper(code)
		if _, ok := targetSet[upperCode]; ok {
			filtered[code] = rate
		}
	}
	return filtered
}
