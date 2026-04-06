package services

import (
	"envdash/internal/client"
	"envdash/internal/structs"
	"fmt"
	"math"
	"strconv"
)

type NomInternal struct {
	client *client.NomClient
}

// NewNomService
// Creates and returns a new NomInternal service
func NewNomService() *NomInternal {
	return &NomInternal{
		client: client.NewNomClient(),
	}
}

// GetCapitalCords
// Retrieves coordinates for the given capital city
func (ni *NomInternal) GetCapitalCords(capital string) (*structs.NomResponse, error) {
	if capital == "" {
		return nil, fmt.Errorf("capital city name is empty")
	}

	incoming, err := ni.client.FetchCapitalCoords(capital)
	if err != nil {
		return nil, err
	}

	lat, err := strconv.ParseFloat(incoming.Lat, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lat: %w", err)
	}

	lon, err := strconv.ParseFloat(incoming.Lon, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lon: %w", err)
	}

	return &structs.NomResponse{
			Lat: roundTwoDeci(lat),
			Lon: roundTwoDeci(lon),
		},
		nil
}

func roundTwoDeci(value float64) float64 {
	return math.Round(value*100) / 100
}
