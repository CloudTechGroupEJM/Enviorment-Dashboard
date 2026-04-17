package config

import (
    "os"
    "strconv"
)

// CacheTTL holds time-to-live durations for different API cache types
type CacheTTL struct {
    OpenAQHours      int
    CountriesHours   int
    CurrencyHours    int
    MetroHours       int
    NominatimHours   int
}

// GetCacheTTLConfig reads cache TTL values from environment variables
// Returns defaults if environment variables are not set
func GetCacheTTLConfig() CacheTTL {
    return CacheTTL{
        OpenAQHours:    getEnvIntOrDefault("CACHE_TTL_OPENAQ_HOURS", 1),
        CountriesHours: getEnvIntOrDefault("CACHE_TTL_COUNTRIES_HOURS", 12),
        CurrencyHours:  getEnvIntOrDefault("CACHE_TTL_CURRENCY_HOURS", 1),
        MetroHours:     getEnvIntOrDefault("CACHE_TTL_METRO_HOURS", 6),
        NominatimHours: getEnvIntOrDefault("CACHE_TTL_NOMINATIM_HOURS", 12),
    }
}

// getEnvIntOrDefault retrieves an environment variable as int or returns default
func getEnvIntOrDefault(envVariableName string, defaultValue int) int {
    envValue := os.Getenv(envVariableName)
    if envValue == "" {
        return defaultValue
    }

    parsedValue, err := strconv.Atoi(envValue)
    if err != nil {
        return defaultValue
    }

    return parsedValue
}