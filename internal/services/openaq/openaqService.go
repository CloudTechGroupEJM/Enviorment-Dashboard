package openaq

import (
	"envdash/internal/client/aq"
	"envdash/internal/structs"
	"log"
	"math"
)

// AQInternal
// The internal implementation of the aq service,
// wrapping an HTTP client for air quality  calculation.
type AQInternal struct {
	client *aq.AQClient
}

// NewAqService
// returns a new AQInternal instance with a client
func NewAqService() *AQInternal {
	return &AQInternal{
		client: aq.NewAQClient(),
	}
}

// GetAQ
// Fetches air quality data for the given coordinates
// lat/long -> associated sensors (max 5) -> fetch sensor data -> calculate PM2.5/PM10 means -> EPA level
// Returns AqResponse with rounded PM values and EPA air quality category
func (aqI *AQInternal) GetAQ(lat, long float64) (*structs.AqResponse, error) {
	locations, err := aqI.client.FetchSensors(lat, long)
	if err != nil {
		return nil, err
	}

	// Build sensor ID → parameter name map from locations
	sensorParams := buildSensorMap(locations)

	// Get location IDs (max 5)
	locationIDs := make([]int, 0)
	for _, loc := range locations.Results {
		if len(locationIDs) >= 5 {
			break
		}
		locationIDs = append(locationIDs, loc.ID)
	}

	// Aggregate data from all locations
	pm25Values, pm10Values := aqI.aggregateAirQualityData(locationIDs, sensorParams)

	// Calculate means
	pm25 := meanAq(pm25Values)
	pm10 := meanAq(pm10Values)

	return &structs.AqResponse{
		PM25:  math.Round(pm25*100) / 100,
		PM10:  math.Round(pm10*100) / 100,
		Level: epaLevel(pm25),
	}, nil
}

// aggregateAirQualityData fetches and aggregates PM2.5 and PM10 readings
// from multiple sensor locations
// claude assisted
func (aqI *AQInternal) aggregateAirQualityData(
	locationIDs []int,
	sensorParams map[int]string,
) (pm25Values, pm10Values []float64) {

	pm25Values, pm10Values = []float64{}, []float64{}

	for _, locID := range locationIDs {
		latest, err := aqI.client.FetchLatest(locID)
		if err != nil {
			log.Printf("skipping location %d: %v", locID, err)
			continue
		}

		for _, result := range latest.Results {
			if result.Value <= 0 {
				continue
			}

			paramName := sensorParams[result.SensorsID]

			if paramName == "pm25" {
				pm25Values = append(pm25Values, result.Value)
			} else if paramName == "pm10" {
				pm10Values = append(pm10Values, result.Value)
			}
		}
	}

	return pm25Values, pm10Values
}

// buildSensorMap
// creates a sensor ID → parameter name mapping from locations
// claude assisted
func buildSensorMap(locations *structs.AirQualityIncoming) map[int]string {
	sensorParams := make(map[int]string)
	for _, loc := range locations.Results {
		for _, sensor := range loc.Sensors {
			if sensor.Parameter.Name == "pm25" || sensor.Parameter.Name == "pm10" {
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
