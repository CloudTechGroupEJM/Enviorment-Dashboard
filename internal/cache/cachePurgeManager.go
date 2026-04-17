package cache

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"
)

// CachePurgeManager handles periodic automatic purging of expired cache entries
type CachePurgeManager struct {
	cacheService       *CacheService
	purgeIntervalHours int
	purgeTicker        *time.Ticker
	stopChannel        chan bool
}

// NewCachePurgeManager creates a new cache purge manager
func NewCachePurgeManager(
	cacheServiceInstance *CacheService,
) *CachePurgeManager {
	purgeIntervalHours := getPurgeIntervalFromEnvironment()

	return &CachePurgeManager{
		cacheService:       cacheServiceInstance,
		purgeIntervalHours: purgeIntervalHours,
		stopChannel:        make(chan bool, 1), // Buffered to prevent deadlock
	}
}

// StartPurgeWorker begins the background cache purge routine
// Runs in a separate goroutine and purges expired entries at configured intervals
func (cachePurgeManager *CachePurgeManager) StartPurgeWorker() {
	go func() {
		purgeIntervalDuration := time.Duration(cachePurgeManager.purgeIntervalHours) * time.Hour
		cachePurgeManager.purgeTicker = time.NewTicker(purgeIntervalDuration)

		log.Printf(
			"cache purge worker started: purging expired entries every %d hours",
			cachePurgeManager.purgeIntervalHours,
		)

		for {
			select {
			case <-cachePurgeManager.purgeTicker.C:
				// Execute cache purge
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				deletedEntriesCount, err := cachePurgeManager.cacheService.PurgeCacheExpiredEntries(ctx)
				cancel()

				if err != nil {
					log.Printf("error during cache purge: %v", err)
				} else {
					log.Printf("cache purge cycle completed: %d entries deleted", deletedEntriesCount)
				}

			case <-cachePurgeManager.stopChannel:
				cachePurgeManager.purgeTicker.Stop()
				log.Println("cache purge worker stopped")
				return
			}
		}
	}()
}

// StopPurgeWorker gracefully stops the background purge routine
func (cachePurgeManager *CachePurgeManager) StopPurgeWorker() {
	cachePurgeManager.stopChannel <- true
}

// getPurgeIntervalFromEnvironment reads CACHE_PURGE_INTERVAL_HOURS from environment
// Defaults to 1 hour if not set
func getPurgeIntervalFromEnvironment() int {
	purgeIntervalHoursString := os.Getenv("CACHE_PURGE_INTERVAL_HOURS")
	if purgeIntervalHoursString == "" {
		return 1 // Default: 1 hour
	}

	purgeIntervalHours, err := strconv.Atoi(purgeIntervalHoursString)
	if err != nil {
		log.Printf("invalid CACHE_PURGE_INTERVAL_HOURS value: %s, using default 1 hour", purgeIntervalHoursString)
		return 1
	}

	return purgeIntervalHours
}
