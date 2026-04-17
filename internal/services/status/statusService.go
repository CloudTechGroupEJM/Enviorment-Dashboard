package status

import (
	"context"
	"envdash/internal/client/status"
	"envdash/internal/config"
	"envdash/internal/services/notification"
	"envdash/internal/structs"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// StatusInternal
// get the start time
// holds a statusClient to get the HTTP functinaltiy
type StatusInternal struct {
	startTime time.Time
	client    *status.StatusClient
	firestore *firestore.Client
}

// StatusService
// start the status service, creates a client and sets startTime
// used as a receiver to organize the status related methods
// Needed to access the start time
func StatusService(startTime time.Time, client *firestore.Client) *StatusInternal {
	return &StatusInternal{
		startTime: startTime,
		client:    status.NewStatusClient(),
		firestore: client,
	}
}

// GetStatus
// Construct the response for status endpoint
func (s *StatusInternal) GetStatus() *structs.StatusResponse {
	healthStatus := s.probeAllEndpoints()
	return &structs.StatusResponse{
		CountriesApi: healthStatus["countries"],
		MetroAPI:     healthStatus["metro"],
		AqAPI:        healthStatus["openaq"],
		Nominatim:    healthStatus["nominatim"],
		CurrencyAPI:  healthStatus["currency"],
		Db_noti:      s.probeFirestore(),
		Webhooks:     notification.GetNotificationCount(),
		Version:      config.APPLICATION_VERSION,
		Uptime:       fmt.Sprintf("%.f", time.Since(s.startTime).Seconds()),
	}
}

// probeAllEndpoints
// checks all the endpoints using get probeHeadEndPoint and getStatus
// returns a map of the status codes for all endpoints
func (s *StatusInternal) probeAllEndpoints() map[string]int {
	endpoints := map[string]string{
		"countries": config.REST_COUNTRIES_API_PROBE,
		"metro":     config.METRO_API,
		"nominatim": config.NOMINATIM_PROBE,
	}

	healthStatuses := make(map[string]int)
	for name, url := range endpoints {
		healthStatuses[name] = s.client.ProbeHeadEndpoint(url)
	}

	healthStatuses["openaq"] = s.client.ProbeGetEndpoint(config.OPENAQ_PROBE, "X-API-Key", os.Getenv("OPEN_AQ_API_KEY"))
	healthStatuses["currency"] = s.client.ProbeGetEndpoint(config.CURRENCIES_API_PROBE, "", "")

	return healthStatuses
}

// probeFirestore
// Uses Collections() iterator to verify the Firestore client can reach the backend.
// This is a lightweight metadata RPC — it does not incur document read charges.
// Returns 200 if reachable, 503 otherwise.
func (s *StatusInternal) probeFirestore() int {
	if s.firestore == nil {
		return http.StatusServiceUnavailable
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	it := s.firestore.Collections(ctx)
	_, err := it.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		log.Printf("Firestore probe failed: %v", err)
		return http.StatusServiceUnavailable
	}
	return http.StatusOK
}
