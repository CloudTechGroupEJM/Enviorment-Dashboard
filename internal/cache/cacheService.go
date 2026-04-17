package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"envdash/internal/store"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
)

// CacheEntry represents a cached API response
type CacheEntry struct {
	CacheKey         string    `firestore:"cacheKey"`
	CachedValue      string    `firestore:"cachedValue"`
	CreatedTimestamp time.Time `firestore:"createdTimestamp"`
	TimeToLiveHours  int       `firestore:"timeToLiveHours"`
}

// CacheService handles caching of API responses to Firestore
type CacheService struct {
	firestoreClient    *firestore.Client
	cacheCollectionRef *firestore.CollectionRef
}

// NewCacheService creates a new cache service instance
func NewCacheService(firestoreClient *firestore.Client) *CacheService {
	return &CacheService{
		firestoreClient:    firestoreClient,
		cacheCollectionRef: firestoreClient.Collection(store.CACHE_COLLECTION),
	}
}

// GenerateCacheKey creates a deterministic cache key from parameters
// Example: "openaq_abc123def456" for OpenAQ API calls
func GenerateCacheKey(apiNamespace string, params map[string]interface{}) (string, error) {
	parameterJSON, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("marshaling cache key parameters: %w", err)
	}

	hash := sha256.Sum256(parameterJSON)
	hashString := hex.EncodeToString(hash[:])

	cacheKeyValue := fmt.Sprintf("%s_%s", apiNamespace, hashString)
	return cacheKeyValue, nil
}

// GetCached retrieves a cached value if it exists and hasn't expired
func (cacheService *CacheService) GetCached(
	ctx context.Context,
	cacheKeyValue string,
) (interface{}, error) {
	documentSnapshot, err := cacheService.cacheCollectionRef.Doc(cacheKeyValue).Get(ctx)
	if err != nil {
		if err == context.Canceled {
			return nil, fmt.Errorf("cache get operation was cancelled")
		}
		// Document doesn't exist (cache miss)
		return nil, nil
	}

	var cacheEntryData CacheEntry
	if err := documentSnapshot.DataTo(&cacheEntryData); err != nil {
		return nil, fmt.Errorf("unmarshaling cache entry: %w", err)
	}

	// Check if cache has expired
	expirationTime := cacheEntryData.CreatedTimestamp.Add(
		time.Duration(cacheEntryData.TimeToLiveHours) * time.Hour,
	)

	if time.Now().After(expirationTime) {
		// Cache expired, delete it
		deleteErr := cacheService.DeleteCached(ctx, cacheKeyValue)
		if deleteErr != nil {
			log.Printf("error deleting expired cache entry %s: %v", cacheKeyValue, deleteErr)
		}
		return nil, nil
	}

	var cachedResponseValue interface{}
	if err := json.Unmarshal([]byte(cacheEntryData.CachedValue), &cachedResponseValue); err != nil {
		return nil, fmt.Errorf("unmarshaling cached response value: %w", err)
	}

	return cachedResponseValue, nil
}

// SetCached stores a response in the cache with the specified TTL
func (cacheService *CacheService) SetCached(
	ctx context.Context,
	cacheKeyValue string,
	responseValue interface{},
	timeToLiveHours int,
) error {
	responseJSON, err := json.Marshal(responseValue)
	if err != nil {
		return fmt.Errorf("marshaling response for cache: %w", err)
	}

	cacheEntryToStore := CacheEntry{
		CacheKey:         cacheKeyValue,
		CachedValue:      string(responseJSON),
		CreatedTimestamp: time.Now(),
		TimeToLiveHours:  timeToLiveHours,
	}

	_, err = cacheService.cacheCollectionRef.Doc(cacheKeyValue).Set(ctx, cacheEntryToStore)
	if err != nil {
		return fmt.Errorf("writing cache entry to firestore: %w", err)
	}

	return nil
}

// DeleteCached removes a specific cache entry
func (cacheService *CacheService) DeleteCached(
	ctx context.Context,
	cacheKeyValue string,
) error {
	_, err := cacheService.cacheCollectionRef.Doc(cacheKeyValue).Delete(ctx)
	if err != nil {
		return fmt.Errorf("deleting cache entry: %w", err)
	}
	return nil
}

// PurgeCacheExpiredEntries removes all cache entries that have expired
// Returns the number of entries deleted and any error encountered
func (cacheService *CacheService) PurgeCacheExpiredEntries(ctx context.Context) (int, error) {
	currentTime := time.Now()
	batchSize := 500
	totalDeletedCount := 0

	for {
		// Query expired entries in batches
		expiredEntriesQuery := cacheService.cacheCollectionRef.
			Where("createdTimestamp", "<",
				currentTime.Add(-time.Duration(24)*time.Hour)). // Start with 24h window for efficiency
			Limit(batchSize)

		documentsSnapshot, err := expiredEntriesQuery.Documents(ctx).GetAll()
		if err != nil {
			return totalDeletedCount, fmt.Errorf("querying expired cache entries: %w", err)
		}

		if len(documentsSnapshot) == 0 {
			break // No more expired entries
		}

		// Delete in batch
		batch := cacheService.firestoreClient.Batch()
		for _, documentRef := range documentsSnapshot {
			batch.Delete(documentRef.Ref)
		}

		_, err = batch.Commit(ctx)
		if err != nil {
			return totalDeletedCount, fmt.Errorf("committing batch delete: %w", err)
		}

		totalDeletedCount += len(documentsSnapshot)

		// If we got fewer documents than our batch size, we've cleaned everything
		if len(documentsSnapshot) < batchSize {
			break
		}
	}

	log.Printf("cache purge completed: deleted %d expired entries", totalDeletedCount)
	return totalDeletedCount, nil
}
