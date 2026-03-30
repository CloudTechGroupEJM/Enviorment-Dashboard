package services

import (
	"envdash/internal/client"
	"envdash/internal/structs"
	"fmt"
)

// MetroInternal is the internal implementation of the currency service,
// wrapping an HTTP client for metro-related operations.
type MetroInternal struct {
	client *client.MetroClient
}

// NewMetroService returns a new MetroInternal instance with a configured HTTP client.
func NewMetroService() *MetroInternal {
	return &MetroInternal{
		client: client.NewMetroClient(),
	}
}

// GetMetro constructs and returns the metro response
func (mi *MetroInternal) GetMetro(lat float64, lon float64) (*structs.MetroResponse, error) {
	meanTemp, meanPrecip, err := mi.processMetroData(lat, lon)
	if err != nil {
		return nil, err
	}

	//Fields are string, as they are not used for anything after
	//If they are round with math.Round(val * 100) / 100, and change to fields to float
	//If tests are hard: change back to float
	return &structs.MetroResponse{
		MeanTemperature:   fmt.Sprintf("%.2f", meanTemp),
		MeanPrecipitation: fmt.Sprintf("%.2f", meanPrecip),
	}, nil
}

// processMetroData fetches incoming data and calculates the mean
func (mi *MetroInternal) processMetroData(latitude float64, longitude float64) (float64, float64, error) {
	metroData, err := mi.client.FetchMetroData(latitude, longitude)
	if err != nil {
		return 0, 0, fmt.Errorf("error metro info: %w", err)
	}

	// Validate data exists before processing
	if len(metroData.Daily.Temperature2mMean) == 0 {
		return 0, 0, fmt.Errorf("no temperature data available")
	}
	if len(metroData.Daily.PrecipitationSum) == 0 {
		return 0, 0, fmt.Errorf("no precipitation data available")
	}

	meanTemp := calculateMean(metroData.Daily.Temperature2mMean)
	meanPrecip := calculateMean(metroData.Daily.PrecipitationSum)

	return meanTemp, meanPrecip, nil
}

// calculateMean
// gets the mean values given a given input list
func calculateMean(values []float64) float64 {
	var sum float64
	for _, val := range values {
		sum += val
	}
	return sum / float64(len(values))
}
