package nominatim

import (
	"context"
	"encoding/json"
	"envdash/internal/cache"
	"envdash/internal/client/nominatim"
	"envdash/internal/structs"
	"fmt"
	"log"
	"math"
	"strconv"
)

type NomInternal struct {
	client         *nominatim.NomClient
	cacheService   *cache.CacheService
	cacheTTLConfig cache.CacheTTL
}

// NewNomService creates and returns a new NomInternal service
func NewNomService(cacheServiceInstance *cache.CacheService) *NomInternal {
	return &NomInternal{
		client:         nominatim.NewNomClient(),
		cacheService:   cacheServiceInstance,
		cacheTTLConfig: cache.GetCacheTTLConfig(),
	}
}

// GetCapitalCoords retrieves coordinates for the given capital city, using cache when available
func (nominatimInternalService *NomInternal) GetCapitalCoords(
	ctx context.Context,
	capital string,
) (*structs.NomResponse, error) {
	if capital == "" {
		return nil, fmt.Errorf("capital city name is empty")
	}

	// Generate cache key from capital name
	cacheKeyParams := map[string]interface{}{
		"capital": capital,
	}

	cacheKeyValue, err := cache.GenerateCacheKey("nominatim", cacheKeyParams)
	if err != nil {
		log.Printf("error generating cache key: %v", err)
		cacheKeyValue = ""
	}

	// Try to get from cache
	if cacheKeyValue != "" {
		cachedResponseValue, err := nominatimInternalService.cacheService.GetCached(ctx, cacheKeyValue)
		if err != nil {
			log.Printf("error retrieving from cache: %v", err)
		} else if cachedResponseValue != nil {
			responseJSON, _ := json.Marshal(cachedResponseValue)
			var cachedNomResponse structs.NomResponse
			if err := json.Unmarshal(responseJSON, &cachedNomResponse); err == nil {
				log.Printf("cache hit for nominatim query: %s", capital)
				return &cachedNomResponse, nil
			}
		}
	}

	// Cache miss - fetch from API
	incoming, err := nominatimInternalService.client.FetchCapitalCoords(ctx, capital)
	if err != nil {
		return nil, err
	}

	latitude, err := strconv.ParseFloat(incoming.Lat, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lat: %w", err)
	}

	longitude, err := strconv.ParseFloat(incoming.Lon, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lon: %w", err)
	}

	nomResponseData := &structs.NomResponse{
		Lat: roundTwoDeci(latitude),
		Lon: roundTwoDeci(longitude),
	}

	// Store in cache
	if cacheKeyValue != "" {
		storeErr := nominatimInternalService.cacheService.SetCached(
			ctx,
			cacheKeyValue,
			nomResponseData,
			nominatimInternalService.cacheTTLConfig.NominatimHours,
		)
		if storeErr != nil {
			log.Printf("error caching nominatim response: %v", storeErr)
		}
	}

	return nomResponseData, nil
}

// roundTwoDeci rounds to two decimal places
func roundTwoDeci(value float64) float64 {
	return math.Round(value*100) / 100
}
