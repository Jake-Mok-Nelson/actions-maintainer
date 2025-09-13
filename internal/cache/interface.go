package cache

import (
	"time"
)

// Cache defines the interface for caching scan results with TTL support
type Cache interface {
	// Get retrieves a cached result if it exists and hasn't expired
	Get(owner string) (*CachedResult, error)

	// Set stores a result in the cache with TTL
	Set(owner string, results interface{}, ttl time.Duration) error

	// CleanExpired removes expired entries from the cache
	CleanExpired() error

	// Close closes the cache and cleans up resources
	Close() error

	// GetStats returns cache statistics
	GetStats() (map[string]interface{}, error)
}

// CachedResult represents a cached scan result
type CachedResult struct {
	Owner     string    `json:"owner"`
	ScanTime  time.Time `json:"scan_time"`
	Results   []byte    `json:"results"` // JSON-encoded scan results
	ExpiresAt time.Time `json:"expires_at"`
}
