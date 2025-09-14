package cache

import (
	"time"
)

// Cache defines the interface for caching version resolution data with TTL support
type Cache interface {
	// GetRef retrieves a cached ref resolution if it exists and hasn't expired
	GetRef(owner, repo, ref string) (string, bool, error)

	// SetRef stores a ref resolution in the cache with TTL
	SetRef(owner, repo, ref, sha string, ttl time.Duration) error

	// GetTags retrieves cached tag mappings for a repository if they exist and haven't expired
	GetTags(owner, repo string) (map[string]string, bool, error)

	// SetTags stores tag mappings for a repository in the cache with TTL
	SetTags(owner, repo string, tags map[string]string, ttl time.Duration) error

	// GetComprehensiveVersionInfo retrieves comprehensive version information from cache
	GetComprehensiveVersionInfo(owner, repo string) (map[string]string, map[string][]string, bool, error)

	// SetComprehensiveVersionInfo stores comprehensive version information in the cache
	SetComprehensiveVersionInfo(owner, repo string, versions map[string]string, aliases map[string][]string, ttl time.Duration) error

	// CleanExpired removes expired entries from the cache
	CleanExpired() error

	// Close closes the cache and cleans up resources
	Close() error

	// GetStats returns cache statistics
	GetStats() (map[string]interface{}, error)
}

// CachedVersionInfo represents cached version resolution data
type CachedVersionInfo struct {
	Key       string    `json:"key"`        // Cache key
	CacheTime time.Time `json:"cache_time"` // When this was cached
	ExpiresAt time.Time `json:"expires_at"` // When this expires
	DataType  string    `json:"data_type"`  // "ref", "tags", or "comprehensive"

	// For ref resolution
	SHA string `json:"sha,omitempty"`

	// For tag mappings
	Tags map[string]string `json:"tags,omitempty"`

	// For comprehensive version info
	Versions map[string]string   `json:"versions,omitempty"` // version -> SHA
	Aliases  map[string][]string `json:"aliases,omitempty"`  // SHA -> []version
}
