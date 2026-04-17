package cache

import (
    // "context"
    "testing"
    // "time"

    // "cloud.google.com/go/firestore"
    "github.com/stretchr/testify/assert"
)

type MockTestResponse struct {
    Temperature float64 `json:"temperature"`
    Humidity    int     `json:"humidity"`
}

func TestGenerateCacheKey(t *testing.T) {
    apiNamespace := "openaq"
    params := map[string]interface{}{
        "latitude":  40.7128,
        "longitude": -74.0060,
    }

    cacheKey1, err := GenerateCacheKey(apiNamespace, params)
    assert.NoError(t, err)
    assert.NotEmpty(t, cacheKey1)
    assert.True(t, len(cacheKey1) > len(apiNamespace))

    // Same params should generate same key (deterministic)
    cacheKey2, err := GenerateCacheKey(apiNamespace, params)
    assert.NoError(t, err)
    assert.Equal(t, cacheKey1, cacheKey2)

    // Different params should generate different key
    differentParams := map[string]interface{}{
        "latitude":  51.5074,
        "longitude": -0.1278,
    }
    cacheKey3, err := GenerateCacheKey(apiNamespace, differentParams)
    assert.NoError(t, err)
    assert.NotEqual(t, cacheKey1, cacheKey3)
}

func TestCacheKeyIncludesNamespace(t *testing.T) {
    cacheKey, err := GenerateCacheKey("countries", map[string]interface{}{})
    assert.NoError(t, err)
    assert.True(t, len(cacheKey) > 0)
    assert.Contains(t, cacheKey, "countries_")
}

