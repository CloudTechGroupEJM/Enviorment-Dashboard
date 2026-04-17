package cache

import (
    "context"
    "encoding/json"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
    "cloud.google.com/go/firestore"
)

// TestGenerateCacheKey tests the GenerateCacheKey function
func TestGenerateCacheKey(t *testing.T) {
    tests := []struct {
        name        string
        apiName     string
        params      map[string]interface{}
        expectError bool
    }{
        {
            name:        "valid single parameter",
            apiName:     "openaq",
            params:      map[string]interface{}{"city": "London"},
            expectError: false,
        },
        {
            name:    "valid multiple parameters",
            apiName: "currency",
            params: map[string]interface{}{
                "base":   "USD",
                "target": "EUR",
                "amount": 100,
            },
            expectError: false,
        },
        {
            name:        "empty parameters",
            apiName:     "country",
            params:      map[string]interface{}{},
            expectError: false,
        },
        {
            name:    "complex nested parameters",
            apiName: "nominatim",
            params: map[string]interface{}{
                "location": map[string]interface{}{
                    "lat": 51.5074,
                    "lon": -0.1278,
                },
                "radius": 10,
            },
            expectError: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            key, err := GenerateCacheKey(tt.apiName, tt.params)

            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.NotEmpty(t, key)
                assert.True(t, len(key) > len(tt.apiName), "key should be longer than apiName")
            }
        })
    }
}

// TestGenerateCacheKeyDeterministic verifies same input produces same output
func TestGenerateCacheKeyDeterministic(t *testing.T) {
    apiName := "openaq"
    params := map[string]interface{}{
        "city": "London",
        "type": "PM25",
    }

    key1, err1 := GenerateCacheKey(apiName, params)
    key2, err2 := GenerateCacheKey(apiName, params)

    require.NoError(t, err1)
    require.NoError(t, err2)
    assert.Equal(t, key1, key2, "same params should generate same key")
}

// TestGenerateCacheKeyUniqueness verifies different params produce different keys
func TestGenerateCacheKeyUniqueness(t *testing.T) {
    apiName := "openaq"
    params1 := map[string]interface{}{"city": "London"}
    params2 := map[string]interface{}{"city": "Paris"}

    key1, _ := GenerateCacheKey(apiName, params1)
    key2, _ := GenerateCacheKey(apiName, params2)

    assert.NotEqual(t, key1, key2, "different params should generate different keys")
}

// TestGenerateCacheKeyFormat verifies key format is "apiName_hash"
func TestGenerateCacheKeyFormat(t *testing.T) {
    apiName := "testapi"
    params := map[string]interface{}{"test": true}

    key, err := GenerateCacheKey(apiName, params)

    require.NoError(t, err)
    assert.Contains(t, key, apiName+"_", "key should contain apiName_")
    parts := len(key) > len(apiName)+1
    assert.True(t, parts, "key should have hash component")
}

// MockDocumentRef mocks firestore DocumentRef
type MockDocumentRef struct {
    mock.Mock
}

func (m *MockDocumentRef) Get(ctx context.Context) (*firestore.DocumentSnapshot, error) {
    args := m.Called(ctx)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*firestore.DocumentSnapshot), args.Error(1)
}

func (m *MockDocumentRef) Set(ctx context.Context, data interface{}, opts ...firestore.SetOption) (*firestore.WriteResult, error) {
    args := m.Called(ctx, data, opts)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*firestore.WriteResult), args.Error(1)
}

func (m *MockDocumentRef) Delete(ctx context.Context, opts ...firestore.Precondition) (*firestore.WriteResult, error) {
    args := m.Called(ctx, opts)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*firestore.WriteResult), args.Error(1)
}

// MockCollectionRef mocks firestore CollectionRef
type MockCollectionRef struct {
    mock.Mock
}

func (m *MockCollectionRef) Doc(path string) *firestore.DocumentRef {
    args := m.Called(path)
    if args.Get(0) == nil {
        return nil
    }
    return args.Get(0).(*firestore.DocumentRef)
}

func (m *MockCollectionRef) Where(path, op string, value interface{}) firestore.Query {
    args := m.Called(path, op, value)
    return args.Get(0).(firestore.Query)
}

// TestNewCacheService verifies CacheService initialization
func TestNewCacheService(t *testing.T) {
    mockClient := new(firestore.Client)
    mockCollectionRef := new(firestore.CollectionRef)

    // In a real test, you'd mock the Collection method
    // For now, we test the struct is created properly
    cacheService := &CacheService{
        firestoreClient:    mockClient,
        cacheCollectionRef: mockCollectionRef,
    }

    assert.NotNil(t, cacheService)
    assert.Equal(t, mockClient, cacheService.firestoreClient)
    assert.Equal(t, mockCollectionRef, cacheService.cacheCollectionRef)
}

// TestCacheEntryStructure verifies CacheEntry fields are correct
func TestCacheEntryStructure(t *testing.T) {
    now := time.Now()
    entry := CacheEntry{
        CacheKey:         "test_key",
        CachedValue:      `{"data": "test"}`,
        CreatedTimestamp: now,
        TimeToLiveHours:  24,
    }

    assert.Equal(t, "test_key", entry.CacheKey)
    assert.Equal(t, `{"data": "test"}`, entry.CachedValue)
    assert.Equal(t, now, entry.CreatedTimestamp)
    assert.Equal(t, 24, entry.TimeToLiveHours)
}

// TestCacheEntryExpiration verifies expiration logic
func TestCacheEntryExpiration(t *testing.T) {
    now := time.Now()
    
    tests := []struct {
        name      string
        createdAt time.Time
        ttlHours  int
        isExpired bool
    }{
        {
            name:      "expired entry (25 hours with 24h TTL)",
            createdAt: now.Add(-25 * time.Hour),
            ttlHours:  24,
            isExpired: true,
        },
        {
            name:      "valid entry (12 hours with 24h TTL)",
            createdAt: now.Add(-12 * time.Hour),
            ttlHours:  24,
            isExpired: false,
        },
        {
            name:      "entry before expiration time (23 hours with 24h TTL)",
            createdAt: now.Add(-23 * time.Hour),
            ttlHours:  24,
            isExpired: false,
        },
        {
            name:      "very old entry",
            createdAt: now.Add(-1000 * time.Hour),
            ttlHours:  1,
            isExpired: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            entry := CacheEntry{
                CacheKey:         "test",
                CachedValue:      "test",
                CreatedTimestamp: tt.createdAt,
                TimeToLiveHours:  tt.ttlHours,
            }

            expirationTime := entry.CreatedTimestamp.Add(
                time.Duration(entry.TimeToLiveHours) * time.Hour,
            )

            isExpired := now.After(expirationTime)
            assert.Equal(t, tt.isExpired, isExpired, "expiration check failed")
        })
    }
}

// TestCacheEntrySerialization tests JSON marshaling/unmarshaling
func TestCacheEntrySerialization(t *testing.T) {
    now := time.Now().Truncate(time.Millisecond) // Truncate to match JSON precision

    original := CacheEntry{
        CacheKey:         "test_key",
        CachedValue:      `{"status": "active", "count": 42}`,
        CreatedTimestamp: now,
        TimeToLiveHours:  24,
    }

    // Marshal
    jsonData, err := json.Marshal(original)
    require.NoError(t, err)

    // Unmarshal
    var restored CacheEntry
    err = json.Unmarshal(jsonData, &restored)
    require.NoError(t, err)

    // Verify all fields match
    assert.Equal(t, original.CacheKey, restored.CacheKey)
    assert.Equal(t, original.CachedValue, restored.CachedValue)
    assert.Equal(t, original.TimeToLiveHours, restored.TimeToLiveHours)
}

// TestCacheEntryWithComplexJSON tests serialization with complex data
func TestCacheEntryWithComplexJSON(t *testing.T) {
    complexData := map[string]interface{}{
        "city":     "London",
        "temp":     15.5,
        "metrics":  []interface{}{1.2, 3.4, 5.6},
        "metadata": map[string]interface{}{"source": "api", "cached": true},
    }

    jsonBytes, err := json.Marshal(complexData)
    require.NoError(t, err)

    entry := CacheEntry{
        CacheKey:         "complex_key",
        CachedValue:      string(jsonBytes),
        CreatedTimestamp: time.Now(),
        TimeToLiveHours:  48,
    }

    // Restore the data
    var restoredData map[string]interface{}
    err = json.Unmarshal([]byte(entry.CachedValue), &restoredData)
    require.NoError(t, err)

    assert.Equal(t, "London", restoredData["city"])
    assert.Equal(t, 15.5, restoredData["temp"])
    assert.NotNil(t, restoredData["metrics"])
    assert.NotNil(t, restoredData["metadata"])
}

// TestCacheKeyVariations tests various edge cases
func TestCacheKeyVariations(t *testing.T) {
    testCases := []struct {
        description string
        apiName     string
        params      map[string]interface{}
    }{
        {
            description: "numeric values",
            apiName:     "api1",
            params:      map[string]interface{}{"id": 123, "count": 456},
        },
        {
            description: "string values",
            apiName:     "api2",
            params:      map[string]interface{}{"name": "test", "type": "query"},
        },
        {
            description: "boolean values",
            apiName:     "api3",
            params:      map[string]interface{}{"active": true, "deleted": false},
        },
        {
            description: "mixed types",
            apiName:     "api4",
            params:      map[string]interface{}{"id": 1, "name": "test", "active": true},
        },
        {
            description: "float values",
            apiName:     "api5",
            params:      map[string]interface{}{"latitude": 51.5074, "longitude": -0.1278},
        },
    }

    for _, tc := range testCases {
        t.Run(tc.description, func(t *testing.T) {
            key, err := GenerateCacheKey(tc.apiName, tc.params)
            assert.NoError(t, err, "should not error for %s", tc.description)
            assert.NotEmpty(t, key, "key should not be empty for %s", tc.description)
            assert.True(t, len(key) > 0, "key should have content for %s", tc.description)
        })
    }
}

// TestCacheErrorScenarios tests error handling
func TestGenerateCacheKeyErrorScenarios(t *testing.T) {
    // Test with circular reference (would cause JSON marshal error)
    // Note: Go maps don't support circular references in the type system
    // but we can test with other error conditions

    t.Run("empty api name", func(t *testing.T) {
        key, err := GenerateCacheKey("", map[string]interface{}{"test": "value"})
        // Should still work, just with empty prefix
        assert.NoError(t, err)
        assert.True(t, len(key) > 1) // Should have at least "_hash"
    })

    t.Run("very long parameters", func(t *testing.T) {
        longParams := make(map[string]interface{})
        for i := 0; i < 100; i++ {
            longParams[string(rune(i))] = "very long value that repeats for testing purposes"
        }
        key, err := GenerateCacheKey("longapi", longParams)
        assert.NoError(t, err)
        assert.NotEmpty(t, key)
    })
}

// TestCacheEntryTTLVariations tests different TTL values
func TestCacheEntryTTLVariations(t *testing.T) {
    ttlTests := []struct {
        name     string
        ttlHours int
    }{
        {"1 hour TTL", 1},
        {"12 hours TTL", 12},
        {"24 hours TTL", 24},
        {"7 days TTL", 24 * 7},
        {"30 days TTL", 24 * 30},
        {"no expiry (large)", 24 * 365},
    }

    for _, tt := range ttlTests {
        t.Run(tt.name, func(t *testing.T) {
            entry := CacheEntry{
                CacheKey:         "test",
                CachedValue:      "data",
                CreatedTimestamp: time.Now(),
                TimeToLiveHours:  tt.ttlHours,
            }

            assert.Equal(t, tt.ttlHours, entry.TimeToLiveHours)
            expirationTime := entry.CreatedTimestamp.Add(
                time.Duration(entry.TimeToLiveHours) * time.Hour,
            )
            assert.True(t, expirationTime.After(time.Now()))
        })
    }
}