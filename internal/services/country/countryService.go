package country

import (
	"context"
	"encoding/json"
	"envdash/internal/cache"
	"envdash/internal/client/country"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"log"
	"strings"
)

type CountryInternal struct {
	client         *country.CountryClient
	cacheService   *cache.CacheService
	cacheTTLConfig config.CacheTTL
}

// NewCountryService creates and returns a new CountryInternal service
func NewCountryService(cacheServiceInstance *cache.CacheService) *CountryInternal {
	return &CountryInternal{
		client:         country.NewCountryClient(),
		cacheService:   cacheServiceInstance,
		cacheTTLConfig: config.GetCacheTTLConfig(),
	}
}

// GetCountry retrieves country information by either an ISO code or country name, using cache when available
func (countryInternalService *CountryInternal) GetCountry(
	ctx context.Context,
	query string,
) (*structs.IncomingCountry, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("country query is required")
	}

	// Generate cache key from query
	cacheKeyParams := map[string]interface{}{
		"query": query,
	}

	cacheKeyValue, err := cache.GenerateCacheKey("countries", cacheKeyParams)
	if err != nil {
		log.Printf("error generating cache key: %v", err)
		cacheKeyValue = ""
	}

	// Try to get from cache
	if cacheKeyValue != "" {
		cachedResponseValue, err := countryInternalService.cacheService.GetCached(ctx, cacheKeyValue)
		if err != nil {
			log.Printf("error retrieving from cache: %v", err)
		} else if cachedResponseValue != nil {
			responseJSON, _ := json.Marshal(cachedResponseValue)
			var cachedCountryResponse structs.IncomingCountry
			if err := json.Unmarshal(responseJSON, &cachedCountryResponse); err == nil {
				log.Printf("cache hit for country query: %s", query)
				return &cachedCountryResponse, nil
			}
		}
	}

	// Cache miss - fetch from API
	var countries []structs.IncomingCountry
	var apiErr error

	switch {
	case len(query) == 2:
		countries, apiErr = countryInternalService.client.FetchByCode(ctx, strings.ToUpper(query))
	case len(query) > 2:
		countries, apiErr = countryInternalService.client.FetchByName(ctx, query)
	default:
		return nil, fmt.Errorf("country query %q is too short: must be a 2-letter ISO code or a full name", query)
	}

	if apiErr != nil {
		return nil, fmt.Errorf("getting country %q: %w", query, apiErr)
	}
	if len(countries) == 0 {
		return nil, fmt.Errorf("no country data returned for %q", query)
	}

	countryResponseData := &countries[0]

	// Store in cache
	if cacheKeyValue != "" {
		storeErr := countryInternalService.cacheService.SetCached(
			ctx,
			cacheKeyValue,
			countryResponseData,
			countryInternalService.cacheTTLConfig.CountriesHours,
		)
		if storeErr != nil {
			log.Printf("error caching country response: %v", storeErr)
		}
	}

	return countryResponseData, nil
}
