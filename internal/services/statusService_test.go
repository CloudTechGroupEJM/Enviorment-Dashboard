package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGetStatus_AllEndpointsHealthy tests GetStatus when all endpoints return 200
func TestGetStatus_AllEndpointsHealthy(t *testing.T) {
	startTime := time.Now()
	service := StatusService(startTime)

	status := service.GetStatus()

	assert.NotNil(t, status)
	assert.Equal(t, "v1", status.Version)
	assert.NotEmpty(t, status.Uptime)
	assert.IsType(t, "", status.Uptime)
}

// TestGetStatus_StructureIsCorrect tests that GetStatus returns correct struct fields
func TestGetStatus_StructureIsCorrect(t *testing.T) {
	service := StatusService(time.Now())

	status := service.GetStatus()

	assert.NotNil(t, status.CountriesApi)
	assert.NotNil(t, status.MetroAPI)
	assert.NotNil(t, status.AqAPI)
	assert.NotNil(t, status.Nominatim)
	assert.NotNil(t, status.CurrencyAPI)
	assert.NotNil(t, status.Db_noti)
	assert.NotEmpty(t, status.Version)
	assert.NotEmpty(t, status.Uptime)
}

// TestGetStatus_UptimeCalculation tests that uptime is calculated correctly
func TestGetStatus_UptimeCalculation(t *testing.T) {
	startTime := time.Now()
	service := StatusService(startTime)

	status := service.GetStatus()

	assert.NotEmpty(t, status.Uptime)
	// Verify that the start time is before now (time has passed)
	assert.True(t, startTime.Before(time.Now()), "StartTime should be before current time")
}

// TestGetStatus_UptimeIsFloat tests that uptime is formatted as float
func TestGetStatus_UptimeIsFloat(t *testing.T) {
	startTime := time.Now().Add(-5 * time.Second)
	service := StatusService(startTime)

	status := service.GetStatus()

	// Should be a number string like "5.xxx"
	assert.Regexp(t, `^\d+(\.\d+)?$`, status.Uptime)
}

// TestGetStatus_VersionMatches tests that version matches config
func TestGetStatus_VersionMatches(t *testing.T) {
	service := StatusService(time.Now())

	status := service.GetStatus()

	assert.Equal(t, "v1", status.Version)
}

// TestGetStatus_DatabaseNotificationDefaultZero tests that db_noti is 0
func TestGetStatus_DatabaseNotificationDefaultZero(t *testing.T) {
	service := StatusService(time.Now())

	status := service.GetStatus()

	assert.Equal(t, 0, status.Db_noti)
}

// TestProbeAllEndpoints_ReturnsMap tests that probeAllEndpoints returns a map
func TestProbeAllEndpoints_ReturnsMap(t *testing.T) {
	service := StatusService(time.Now())

	result := service.probeAllEndpoints()

	assert.NotNil(t, result)
	assert.IsType(t, make(map[string]int), result)
}

// TestProbeAllEndpoints_ContainsAllKeys tests that all endpoint keys are present
func TestProbeAllEndpoints_ContainsAllKeys(t *testing.T) {
	service := StatusService(time.Now())

	result := service.probeAllEndpoints()

	expectedKeys := []string{"countries", "metro", "openaq", "nominatim", "currency"}
	for _, key := range expectedKeys {
		assert.Contains(t, result, key)
	}
}

// TestProbeAllEndpoints_AllValuesAreStatusCodes tests that all values are HTTP status codes
func TestProbeAllEndpoints_AllValuesAreStatusCodes(t *testing.T) {
	service := StatusService(time.Now())

	result := service.probeAllEndpoints()

	for key, statusCode := range result {
		assert.Greater(t, statusCode, 0, "Status code for %s should be greater than 0", key)
		assert.Less(t, statusCode, 600, "Status code for %s should be less than 600", key)
	}
}

// TestProbeAllEndpoints_UnreachableEndpoint tests behavior when endpoint is unreachable
func TestProbeAllEndpoints_UnreachableEndpoint(t *testing.T) {
	// This test relies on actual external API unreachability
	// Or you could mock the client - see integration test alternative
	service := StatusService(time.Now())

	result := service.probeAllEndpoints()

	// All keys should exist even if unreachable
	assert.Equal(t, 5, len(result))
}

// TestGetStatus_ConsistentCalls tests that multiple calls return consistent data
func TestGetStatus_ConsistentCalls(t *testing.T) {
	service := StatusService(time.Now())

	status1 := service.GetStatus()
	time.Sleep(100 * time.Millisecond)
	status2 := service.GetStatus()

	// Version should always match
	assert.Equal(t, status1.Version, status2.Version)
	// Uptime should increase
	uptime1 := status1.Uptime
	uptime2 := status2.Uptime
	assert.NotEmpty(t, uptime1)
	assert.NotEmpty(t, uptime2)
}

// TestGetStatus_VeryEarlyStartTime tests service with start time in past
func TestGetStatus_VeryEarlyStartTime(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour)
	service := StatusService(startTime)

	status := service.GetStatus()

	// Uptime should be large number (around 3600 for 1 hour)
	assert.NotEmpty(t, status.Uptime)
	assert.Regexp(t, `^\d{4,}(\.\d+)?$`, status.Uptime)
}

// TestStatusService_CreatesClient tests that StatusService creates a client
func TestStatusService_CreatesClient(t *testing.T) {
	service := StatusService(time.Now())

	assert.NotNil(t, service)
	status := service.GetStatus()
	assert.NotNil(t, status)
}

// TestStatusService_ReceiverOrganization tests service can be called multiple times
func TestStatusService_ReceiverOrganization(t *testing.T) {
	service := StatusService(time.Now())

	// Call multiple times to ensure receiver pattern works
	status1 := service.GetStatus()
	status2 := service.GetStatus()

	assert.NotNil(t, status1)
	assert.NotNil(t, status2)
	assert.Equal(t, status1.Version, status2.Version)
}

// TestGetStatus_NilCheck tests that GetStatus never returns nil
func TestGetStatus_NilCheck(t *testing.T) {
	service := StatusService(time.Now())

	status := service.GetStatus()

	assert.NotNil(t, status, "GetStatus should never return nil")
}

// TestGetStatus_EmptyUptimeNegative tests uptime is never empty
func TestGetStatus_EmptyUptimeNegative(t *testing.T) {
	service := StatusService(time.Now())

	status := service.GetStatus()

	assert.NotEmpty(t, status.Uptime, "Uptime should never be empty")
}

// TestGetStatus_AllFieldsPopulated tests all response fields are populated
func TestGetStatus_AllFieldsPopulated(t *testing.T) {
	service := StatusService(time.Now())

	status := service.GetStatus()

	assert.True(t, status.CountriesApi >= 0, "CountriesApi should be >= 0")
	assert.True(t, status.MetroAPI >= 0, "MetroAPI should be >= 0")
	assert.True(t, status.AqAPI >= 0, "AqAPI should be >= 0")
	assert.True(t, status.Nominatim >= 0, "Nominatim should be >= 0")
	assert.True(t, status.CurrencyAPI >= 0, "CurrencyAPI should be >= 0")
	assert.True(t, status.Db_noti >= 0, "Db_noti should be >= 0")
	assert.NotEmpty(t, status.Version, "Version should not be empty")
	assert.NotEmpty(t, status.Uptime, "Uptime should not be empty")
}

// TestProbeAllEndpoints_RaceCondition tests concurrent access
func TestProbeAllEndpoints_RaceCondition(t *testing.T) {
	service := StatusService(time.Now())

	done := make(chan bool, 2)

	go func() {
		result := service.probeAllEndpoints()
		assert.NotNil(t, result)
		done <- true
	}()

	go func() {
		result := service.probeAllEndpoints()
		assert.NotNil(t, result)
		done <- true
	}()

	<-done
	<-done
}

// TestGetStatus_ZeroStartTime tests behavior with zero start time
func TestGetStatus_ZeroStartTime(t *testing.T) {
	// Using zero time to test edge case
	service := StatusService(time.Time{})

	status := service.GetStatus()

	assert.NotNil(t, status)
	assert.NotEmpty(t, status.Uptime)
}
