package currency

import (
	"context"
	"encoding/json"
	"envdash/internal/cache"
	"envdash/internal/client/currency"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"log"
	"sort"
	"strings"
)

type CurrencyInternal struct {
	client         *currency.CurrencyClient
	cacheService   *cache.CacheService
	cacheTTLConfig config.CacheTTL
}

// NewCurrencyService returns a new CurrencyInternal instance with a configured HTTP client
func NewCurrencyService(cacheServiceInstance *cache.CacheService) *CurrencyInternal {
	return &CurrencyInternal{
		client:         currency.NewCurrencyClient(),
		cacheService:   cacheServiceInstance,
		cacheTTLConfig: config.GetCacheTTLConfig(),
	}
}

// GetCurrency retrieves the exchange rate information, using cache when available
func (currencyInternalService *CurrencyInternal) GetCurrency(
	ctx context.Context,
	currencyCode string,
	target []string,
) (*structs.CurrencyResponse, error) {
	if currencyCode == "" {
		return nil, fmt.Errorf("base currency code is required")
	}

	if len(target) == 0 {
		return &structs.CurrencyResponse{TargetCurrencies: map[string]float64{}}, nil
	}

	// Generate cache key from parameters
	// Sort target currencies for consistent key generation
	sortedTarget := make([]string, len(target))
	copy(sortedTarget, target)
	sort.Strings(sortedTarget)

	cacheKeyParams := map[string]interface{}{
		"currencyCode": strings.ToUpper(currencyCode),
		"targetCodes":  sortedTarget,
	}

	cacheKeyValue, err := cache.GenerateCacheKey("currency", cacheKeyParams)
	if err != nil {
		log.Printf("error generating cache key: %v", err)
		cacheKeyValue = ""
	}

	// Try to get from cache
	if cacheKeyValue != "" {
		cachedResponseValue, err := currencyInternalService.cacheService.GetCached(ctx, cacheKeyValue)
		if err != nil {
			log.Printf("error retrieving from cache: %v", err)
		} else if cachedResponseValue != nil {
			responseJSON, _ := json.Marshal(cachedResponseValue)
			var cachedCurrencyResponse structs.CurrencyResponse
			if err := json.Unmarshal(responseJSON, &cachedCurrencyResponse); err == nil {
				log.Printf("cache hit for currency query: %s to %v", currencyCode, sortedTarget)
				return &cachedCurrencyResponse, nil
			}
		}
	}

	// Cache miss - fetch from API
	allCurrencies, err := currencyInternalService.client.FetchExchangeRates(ctx, currencyCode)
	if err != nil {
		return nil, fmt.Errorf("getting exchange rates for %s: %w", currencyCode, err)
	}

	filtered := filterRates(allCurrencies.Rates, target)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("none of the requested currencies found in response")
	}

	currencyResponseData := &structs.CurrencyResponse{TargetCurrencies: filtered}

	// Store in cache
	if cacheKeyValue != "" {
		storeErr := currencyInternalService.cacheService.SetCached(
			ctx,
			cacheKeyValue,
			currencyResponseData,
			currencyInternalService.cacheTTLConfig.CurrencyHours,
		)
		if storeErr != nil {
			log.Printf("error caching currency response: %v", storeErr)
		}
	}

	return currencyResponseData, nil
}

// filterRates returns the subset of rates whose currency codes appear in target
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
