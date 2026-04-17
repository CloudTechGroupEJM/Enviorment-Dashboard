package metro

import (
	"context"
	"encoding/json"
	"envdash/internal/cache"
	"envdash/internal/client/metro"
	"envdash/internal/config"
	"envdash/internal/structs"
	"fmt"
	"log"
	"math"
)

type MetroInternal struct {
	client         *metro.MetroClient
	cacheService   *cache.CacheService
	cacheTTLConfig config.CacheTTL
}

// NewMetroService returns a new MetroInternal instance with a configured HTTP client
func NewMetroService(cacheServiceInstance *cache.CacheService) *MetroInternal {
	return &MetroInternal{
		client:         metro.NewMetroClient(),
		cacheService:   cacheServiceInstance,
		cacheTTLConfig: config.GetCacheTTLConfig(),
	}
}

// GetMetro constructs and returns the metro response, using cache when available
func (metroInternalService *MetroInternal) GetMetro(
	ctx context.Context,
	latitude float64,
	longitude float64,
) (*structs.MetroResponse, error) {
	if err := validateLatLong(latitude, longitude); err != nil {
		return nil, err
	}

	// Generate cache key from coordinates
	cacheKeyParams := map[string]interface{}{
		"latitude":  latitude,
		"longitude": longitude,
	}

	cacheKeyValue, err := cache.GenerateCacheKey("metro", cacheKeyParams)
	if err != nil {
		log.Printf("error generating cache key: %v", err)
		cacheKeyValue = ""
	}

	// Try to get from cache
	if cacheKeyValue != "" {
		cachedResponseValue, err := metroInternalService.cacheService.GetCached(ctx, cacheKeyValue)
		if err != nil {
			log.Printf("error retrieving from cache: %v", err)
		} else if cachedResponseValue != nil {
			responseJSON, _ := json.Marshal(cachedResponseValue)
			var cachedMetroResponse structs.MetroResponse
			if err := json.Unmarshal(responseJSON, &cachedMetroResponse); err == nil {
				log.Printf("cache hit for metro query: lat=%f, lon=%f", latitude, longitude)
				return &cachedMetroResponse, nil
			}
		}
	}

	// Cache miss - process metro data
	meanTemp, meanPrecip, err := metroInternalService.processMetroData(ctx, latitude, longitude)
	if err != nil {
		return nil, err
	}

	metroResponseData := &structs.MetroResponse{
		MeanTemperature:   meanTemp,
		MeanPrecipitation: meanPrecip,
	}

	// Store in cache
	if cacheKeyValue != "" {
		storeErr := metroInternalService.cacheService.SetCached(
			ctx,
			cacheKeyValue,
			metroResponseData,
			metroInternalService.cacheTTLConfig.MetroHours,
		)
		if storeErr != nil {
			log.Printf("error caching metro response: %v", storeErr)
		}
	}

	return metroResponseData, nil
}

// processMetroData fetches incoming data and calculates the mean
func (metroInternalService *MetroInternal) processMetroData(
	ctx context.Context,
	latitude float64,
	longitude float64,
) (float64, float64, error) {
	metroData, err := metroInternalService.client.FetchMetroData(ctx, latitude, longitude)
	if err != nil {
		return 0, 0, fmt.Errorf("fetching metro data: %w", err)
	}

	if len(metroData.Daily.Temperature2mMean) == 0 {
		return 0, 0, fmt.Errorf("no temperature data available")
	}
	if len(metroData.Daily.PrecipitationSum) == 0 {
		return 0, 0, fmt.Errorf("no precipitation data available")
	}

	meanTemp := roundDeci(calculateMean(metroData.Daily.Temperature2mMean))
	meanPrecip := roundDeci(calculateMean(metroData.Daily.PrecipitationSum))

	return meanTemp, meanPrecip, nil
}

// calculateMean gets the mean values given a given input list
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, val := range values {
		sum += val
	}
	return sum / float64(len(values))
}

// validateLatLong validates that the latitude and longitude are acceptable values
func validateLatLong(latitude, longitude float64) error {
	if latitude < -90 || latitude > 90 {
		return fmt.Errorf("invalid latitude: must be between -90 and 90")
	}
	if longitude < -180 || longitude > 180 {
		return fmt.Errorf("invalid longitude: must be between -180 and 180")
	}
	return nil
}

// roundDeci rounds to one decimal place
func roundDeci(value float64) float64 {
	return math.Round(value*10) / 10
}
