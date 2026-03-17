package services

import (
	"envdash/internal/structs"
	"fmt"
	"time"
)

type StatusInternal struct {
	startTime time.Time
}

func StatusService(startTime time.Time) *StatusInternal {
	return &StatusInternal{startTime: startTime}
}

func (s *StatusInternal) GetStatus(healthStatus map[string]int) *structs.StatusResponse {
	return &structs.StatusResponse{
		CountriesApi: healthStatus["countries"],
		MetroAPI:     healthStatus["metro"],
		AqAPI:        healthStatus["openaq"],
		Nominatim:    healthStatus["nominatim"],
		CurrencyAPI:  healthStatus["currency"],
		Db_noti:      0,
		Version:      "v1",
		Uptime:       fmt.Sprintf("%.f", time.Since(s.startTime).Seconds()),
	}
}
