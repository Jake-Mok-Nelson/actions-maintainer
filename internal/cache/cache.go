package cache

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CachedResult represents a cached scan result
type CachedResult struct {
	Owner     string    `json:"owner"`
	ScanTime  time.Time `json:"scan_time"`
	Results   []byte    `json:"results"` // JSON-encoded scan results
	ExpiresAt time.Time `json:"expires_at"`
}

// Cache provides TTL-based caching using in-memory storage
type Cache struct {
	data  map[string]*CachedResult
	mutex sync.RWMutex
}

// NewCache creates a new in-memory cache
func NewCache() *Cache {
	return &Cache{
		data:  make(map[string]*CachedResult),
		mutex: sync.RWMutex{},
	}
}

// Get retrieves a cached result if it exists and hasn't expired
func (c *Cache) Get(owner string) (*CachedResult, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	result, exists := c.data[owner]
	if !exists {
		return nil, nil // No cached result found
	}

	// Check if expired
	if time.Now().After(result.ExpiresAt) {
		// Remove expired entry
		c.mutex.RUnlock()
		c.mutex.Lock()
		delete(c.data, owner)
		c.mutex.Unlock()
		c.mutex.RLock()
		return nil, nil
	}

	return result, nil
}

// Set stores a result in the cache with TTL
func (c *Cache) Set(owner string, results interface{}, ttl time.Duration) error {
	resultsJSON, err := json.Marshal(results)
	if err != nil {
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

	return nil
}

// CleanExpired removes expired entries from the cache
func (c *Cache) CleanExpired() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	var removed int

	for owner, result := range c.data {
		if now.After(result.ExpiresAt) {
			delete(c.data, owner)
			removed++
		}
	}

	if removed > 0 {
		fmt.Printf("Cleaned %d expired cache entries\n", removed)
	}

	return nil
}

// Close is a no-op for memory cache but implements the interface
func (c *Cache) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data = make(map[string]*CachedResult)
	return nil
}

// GetStats returns cache statistics
func (c *Cache) GetStats() (map[string]interface{}, error) {
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
