package openaq

import (
	"context"
	"encoding/json"
	"envdash/internal/cache"
	aqclient "envdash/internal/client/aq"
	"envdash/internal/config"
	"envdash/internal/structs"
	"log"
	"math"
)

const (
	maxLocations = 5 //locatoins used for sensor data
	paramPM25    = "pm25"
	paramPM10    = "pm10"
)

// AQInternal
// The internal implementation of the aq service,
// wrapping an HTTP client for air quality calculation.
type AQInternal struct {
	client         *aqclient.AQClient
	cacheService   *cache.CacheService
	cacheTTLConfig config.CacheTTL
}

// NewAqService returns a new AQInternal instance with a client
func NewAqService(cacheServiceInstance *cache.CacheService) *AQInternal {
	return &AQInternal{
		client:         aqclient.NewAQClient(),
		cacheService:   cacheServiceInstance,
		cacheTTLConfig: config.GetCacheTTLConfig(),
	}
}

// GetAQ
// Fetches air quality data for the given coordinates.
// Uses cache when available, with CACHE_TTL_OPENAQ_HOURS TTL from environment
// lat/long -> cache check -> sensors -> fetch sensor data -> calculate PM2.5/PM10 means -> EPA level
// Returns AqResponse with rounded PM values and EPA air quality category.
func (aqI *AQInternal) GetAQ(ctx context.Context, lat, long float64) (*structs.AqResponse, error) {
	// Generate cache key from coordinates
	cacheKeyParams := map[string]interface{}{
		"lat":  lat,
		"long": long,
	}

	cacheKeyValue, err := cache.GenerateCacheKey("openaq", cacheKeyParams)
	if err != nil {
		log.Printf("error generating cache key: %v", err)
		cacheKeyValue = ""
	}

	// Try to get from cache
	if cacheKeyValue != "" {
		cachedResponseValue, err := aqI.cacheService.GetCached(ctx, cacheKeyValue)
		if err != nil {
			log.Printf("error retrieving from cache: %v", err)
		} else if cachedResponseValue != nil {
			responseJSON, _ := json.Marshal(cachedResponseValue)
			var cachedAQResponse structs.AqResponse
			if err := json.Unmarshal(responseJSON, &cachedAQResponse); err == nil {
				log.Printf("cache hit for openaq query: lat=%f, long=%f", lat, long)
				return &cachedAQResponse, nil
			}
		}
	}

	// Cache miss - fetch from API
	locations, err := aqI.client.FetchSensors(ctx, lat, long)
	if err != nil {
		return nil, err
	}

	sensorParams := buildSensorMap(locations)
	locationIDs := pickLocationIDs(locations, maxLocations)

	pm25Values, pm10Values := aqI.aggregateAirQualityData(ctx, locationIDs, sensorParams)

	pm25 := meanAq(pm25Values)
	pm10 := meanAq(pm10Values)

	aqResponseData := &structs.AqResponse{
		PM25:  roundTwoDeci(pm25),
		PM10:  roundTwoDeci(pm10),
		Level: epaLevel(pm25),
	}

	// Store in cache
	if cacheKeyValue != "" {
		storeErr := aqI.cacheService.SetCached(
			ctx,
			cacheKeyValue,
			aqResponseData,
			aqI.cacheTTLConfig.OpenAQHours,
		)
		if storeErr != nil {
			log.Printf("error caching openaq response: %v", storeErr)
		}
	}

	return aqResponseData, nil
}

// pickLocationIDs returns up to limit location IDs from the response.
// calude inspired
func pickLocationIDs(locations *structs.AirQualityIncoming, limit int) []int {
	ids := make([]int, 0, limit)
	for _, loc := range locations.Results {
		if len(ids) >= limit {
			break
		}
		ids = append(ids, loc.ID)
	}
	return ids
}

// aggregateAirQualityData fetches and aggregates PM2.5 and PM10 readings
// from multiple sensor locations.
// claude inspired
func (aqI *AQInternal) aggregateAirQualityData(
	ctx context.Context,
	locationIDs []int,
	sensorParams map[int]string,
) (pm25Values, pm10Values []float64) {
	for _, locID := range locationIDs {
		latest, err := aqI.client.FetchLatest(ctx, locID)
		if err != nil {
			log.Printf("skipping location %d: %v", locID, err)
			continue
		}

		for _, result := range latest.Results {
			if result.Value <= 0 {
				continue
			}
			switch sensorParams[result.SensorsID] {
			case paramPM25:
				pm25Values = append(pm25Values, result.Value)
			case paramPM10:
				pm10Values = append(pm10Values, result.Value)
			}
		}
	}
	return pm25Values, pm10Values
}

// buildSensorMap
// creates a sensor ID → parameter name mapping from locations
// claude inspired
func buildSensorMap(locations *structs.AirQualityIncoming) map[int]string {
	sensorParams := make(map[int]string)
	for _, loc := range locations.Results {
		for _, sensor := range loc.Sensors {
			if sensor.Parameter.Name == paramPM25 || sensor.Parameter.Name == paramPM10 {
				sensorParams[sensor.ID] = sensor.Parameter.Name
			}
		}
	}
	return sensorParams
}

// meanAq
// returns the average of values, or -1 if empty
func meanAq(values []float64) float64 {
	if len(values) == 0 {
		return -1
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// roundTwoDeci rounds to 2 decimal places
func roundTwoDeci(value float64) float64 {
	return math.Round(value*100) / 100
}

// epaLevel
// maps a PM2.5 µg/m³ reading to an EPA AQI category.
func epaLevel(pm25 float64) string {
	switch {
	case pm25 < 0:
		return "unknown"
	case pm25 <= 12.0:
		return "Good"
	case pm25 <= 35.4:
		return "Moderate"
	case pm25 <= 55.4:
		return "Unhealthy for Sensitive Groups"
	case pm25 <= 150.4:
		return "Unhealthy"
	case pm25 <= 250.4:
		return "Very Unhealthy"
	default:
		return "Hazardous"
	}
}
