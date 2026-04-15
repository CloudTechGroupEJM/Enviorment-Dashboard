package metro

import (
	"context"
	"envdash/internal/client/metro"
	"envdash/internal/structs"
	"fmt"
	"math"
)

// MetroInternal is the internal implementation of the metro service,
// wrapping an HTTP client for metro-related operations.
type MetroInternal struct {
	client *metro.MetroClient
}

// NewMetroService returns a new MetroInternal instance with a configured HTTP client.
func NewMetroService() *MetroInternal {
	return &MetroInternal{
		client: metro.NewMetroClient(),
	}
}

// GetMetro constructs and returns the metro response
func (mi *MetroInternal) GetMetro(ctx context.Context, lat float64, lon float64) (*structs.MetroResponse, error) {
	if err := validateLatLong(lat, lon); err != nil {
		return nil, err
	}

	meanTemp, meanPrecip, err := mi.processMetroData(ctx, lat, lon)
	if err != nil {
		return nil, err
	}

	return &structs.MetroResponse{
		MeanTemperature:   roundOneDeci(meanTemp),
		MeanPrecipitation: roundOneDeci(meanPrecip),
	}, nil
}

// processMetroData fetches incoming data and calculates the mean
func (mi *MetroInternal) processMetroData(ctx context.Context, latitude float64, longitude float64) (float64, float64, error) {
	metroData, err := mi.client.FetchMetroData(ctx, latitude, longitude)
	if err != nil {
		return 0, 0, fmt.Errorf("computing metro means: %w", err)
	}

	if len(metroData.Daily.Temperature2mMean) == 0 {
		return 0, 0, fmt.Errorf("no temperature data available")
	}
	if len(metroData.Daily.PrecipitationSum) == 0 {
		return 0, 0, fmt.Errorf("no precipitation data available")
	}

	//round to one decimal for extra precision
	meanTemp := calculateMean(metroData.Daily.Temperature2mMean)
	meanPrecip := calculateMean(metroData.Daily.PrecipitationSum)

	return meanTemp, meanPrecip, nil
}

// calculateMean
// gets the mean values given a given input list
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

// validateLatLong
// Validates that the latitude and longitude are acceptable values
func validateLatLong(lat, long float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("invalid latitude: must be between -90 and 90")
	}
	if long < -180 || long > 180 {
		return fmt.Errorf("invalid longitude: must be between -180 and 180")
	}
	return nil
}

// roundOneDeci rounds to 1 decimal place
func roundOneDeci(value float64) float64 {
	return math.Round(value*10) / 10
}
