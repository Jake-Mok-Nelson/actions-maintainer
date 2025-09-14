package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Config holds configuration options for the cache
type Config struct {
	Verbose bool
}

// MemoryCache provides TTL-based caching using in-memory storage
type MemoryCache struct {
	data    map[string]*CachedResult
	mutex   sync.RWMutex
	verbose bool
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() Cache {
	return NewMemoryCacheWithConfig(&Config{Verbose: false})
}

// NewMemoryCacheWithConfig creates a new in-memory cache with configuration
func NewMemoryCacheWithConfig(config *Config) Cache {
	if config == nil {
		config = &Config{Verbose: false}
	}

	if config.Verbose {
		log.Printf("Memory cache initialized with verbose logging enabled")
	}

	return &MemoryCache{
		data:    make(map[string]*CachedResult),
		mutex:   sync.RWMutex{},
		verbose: config.Verbose,
	}
}

// Get retrieves a cached result if it exists and hasn't expired
func (c *MemoryCache) Get(owner string) (*CachedResult, error) {
	if c.verbose {
		log.Printf("Cache: Checking for cached results for owner '%s'", owner)
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	result, exists := c.data[owner]
	if !exists {
		if c.verbose {
			log.Printf("Cache: MISS - No cached result found for owner '%s'", owner)
		}
		return nil, nil // No cached result found
	}

	// Check if expired
	if time.Now().After(result.ExpiresAt) {
		if c.verbose {
			log.Printf("Cache: MISS - Cached result for owner '%s' has expired (was valid until %s)", owner, result.ExpiresAt.Format(time.RFC3339))
		}
		// Remove expired entry
		c.mutex.RUnlock()
		c.mutex.Lock()
		delete(c.data, owner)
		c.mutex.Unlock()
		c.mutex.RLock()
		return nil, nil
	}

	if c.verbose {
		log.Printf("Cache: HIT - Found valid cached result for owner '%s' (expires at %s)", owner, result.ExpiresAt.Format(time.RFC3339))
	}

	return result, nil
}

// Set stores a result in the cache with TTL
func (c *MemoryCache) Set(owner string, results interface{}, ttl time.Duration) error {
	if c.verbose {
		log.Printf("Cache: Storing results for owner '%s' with TTL %s", owner, ttl)
	}

	resultsJSON, err := json.Marshal(results)
	if err != nil {
		if c.verbose {
			log.Printf("Cache: Failed to marshal results for owner '%s' - %v", owner, err)
		}
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	result := &CachedResult{
		Owner:     owner,
		ScanTime:  now,
		Results:   resultsJSON,
		ExpiresAt: expiresAt,
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[owner] = result

	if c.verbose {
		log.Printf("Cache: Successfully stored results for owner '%s' (expires at %s)", owner, expiresAt.Format(time.RFC3339))
	}

	return nil
}

// CleanExpired removes expired entries from the cache
func (c *MemoryCache) CleanExpired() error {
	if c.verbose {
		log.Printf("Cache: Cleaning expired entries")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	var removed int

	for owner, result := range c.data {
		if now.After(result.ExpiresAt) {
			delete(c.data, owner)
			removed++
			if c.verbose {
				log.Printf("Cache: Removed expired entry for owner '%s'", owner)
			}
		}
	}

	if removed > 0 {
		fmt.Printf("Cleaned %d expired cache entries\n", removed)
		if c.verbose {
			log.Printf("Cache: Cleaning complete - removed %d expired entries", removed)
		}
	} else if c.verbose {
		log.Printf("Cache: No expired entries found during cleaning")
	}

	return nil
}

// Close is a no-op for memory cache but implements the interface
func (c *MemoryCache) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data = make(map[string]*CachedResult)
	return nil
}

// GetStats returns cache statistics
func (c *MemoryCache) GetStats() (map[string]interface{}, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	stats := make(map[string]interface{})
	now := time.Now()

	totalEntries := len(c.data)
	expiredEntries := 0

	for _, result := range c.data {
		if now.After(result.ExpiresAt) {
			expiredEntries++
		}
	}

	stats["total_entries"] = totalEntries
	stats["expired_entries"] = expiredEntries
	stats["valid_entries"] = totalEntries - expiredEntries

	return stats, nil
}
