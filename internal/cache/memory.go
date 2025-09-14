package cache

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Config holds configuration options for the cache
type Config struct {
	Verbose bool
}

// MemoryCache provides TTL-based caching using in-memory storage for version resolution data
type MemoryCache struct {
	data    map[string]*CachedVersionInfo
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
		data:    make(map[string]*CachedVersionInfo),
		mutex:   sync.RWMutex{},
		verbose: config.Verbose,
	}
}

// GetRef retrieves a cached ref resolution if it exists and hasn't expired
func (c *MemoryCache) GetRef(owner, repo, ref string) (string, bool, error) {
	key := fmt.Sprintf("%s/%s:%s", owner, repo, ref)

	if c.verbose {
		log.Printf("Cache: Checking for cached ref resolution '%s'", key)
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		if c.verbose {
			log.Printf("Cache: MISS - No cached ref resolution found for '%s'", key)
		}
		return "", false, nil
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		if c.verbose {
			log.Printf("Cache: MISS - Cached ref resolution for '%s' has expired (was valid until %s)", key, entry.ExpiresAt.Format(time.RFC3339))
		}
		// Remove expired entry
		c.mutex.RUnlock()
		c.mutex.Lock()
		delete(c.data, key)
		c.mutex.Unlock()
		c.mutex.RLock()
		return "", false, nil
	}

	if entry.DataType != "ref" {
		if c.verbose {
			log.Printf("Cache: MISS - Cached entry for '%s' is not a ref resolution (type: %s)", key, entry.DataType)
		}
		return "", false, nil
	}

	if c.verbose {
		log.Printf("Cache: HIT - Found valid cached ref resolution for '%s' -> %s (expires at %s)", key, entry.SHA, entry.ExpiresAt.Format(time.RFC3339))
	}

	return entry.SHA, true, nil
}

// SetRef stores a ref resolution in the cache with TTL
func (c *MemoryCache) SetRef(owner, repo, ref, sha string, ttl time.Duration) error {
	key := fmt.Sprintf("%s/%s:%s", owner, repo, ref)

	if c.verbose {
		log.Printf("Cache: Storing ref resolution '%s' -> %s with TTL %s", key, sha, ttl)
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	entry := &CachedVersionInfo{
		Key:       key,
		CacheTime: now,
		ExpiresAt: expiresAt,
		DataType:  "ref",
		SHA:       sha,
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[key] = entry

	if c.verbose {
		log.Printf("Cache: Successfully stored ref resolution for '%s' (expires at %s)", key, expiresAt.Format(time.RFC3339))
	}

	return nil
}

// GetTags retrieves cached tag mappings for a repository if they exist and haven't expired
func (c *MemoryCache) GetTags(owner, repo string) (map[string]string, bool, error) {
	key := fmt.Sprintf("%s/%s:tags", owner, repo)

	if c.verbose {
		log.Printf("Cache: Checking for cached tags '%s'", key)
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		if c.verbose {
			log.Printf("Cache: MISS - No cached tags found for '%s'", key)
		}
		return nil, false, nil
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		if c.verbose {
			log.Printf("Cache: MISS - Cached tags for '%s' has expired (was valid until %s)", key, entry.ExpiresAt.Format(time.RFC3339))
		}
		// Remove expired entry
		c.mutex.RUnlock()
		c.mutex.Lock()
		delete(c.data, key)
		c.mutex.Unlock()
		c.mutex.RLock()
		return nil, false, nil
	}

	if entry.DataType != "tags" {
		if c.verbose {
			log.Printf("Cache: MISS - Cached entry for '%s' is not tags (type: %s)", key, entry.DataType)
		}
		return nil, false, nil
	}

	if c.verbose {
		log.Printf("Cache: HIT - Found valid cached tags for '%s' (%d tags, expires at %s)", key, len(entry.Tags), entry.ExpiresAt.Format(time.RFC3339))
	}

	return entry.Tags, true, nil
}

// SetTags stores tag mappings for a repository in the cache with TTL
func (c *MemoryCache) SetTags(owner, repo string, tags map[string]string, ttl time.Duration) error {
	key := fmt.Sprintf("%s/%s:tags", owner, repo)

	if c.verbose {
		log.Printf("Cache: Storing tags for '%s' (%d tags) with TTL %s", key, len(tags), ttl)
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	entry := &CachedVersionInfo{
		Key:       key,
		CacheTime: now,
		ExpiresAt: expiresAt,
		DataType:  "tags",
		Tags:      tags,
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[key] = entry

	if c.verbose {
		log.Printf("Cache: Successfully stored tags for '%s' (expires at %s)", key, expiresAt.Format(time.RFC3339))
	}

	return nil
}

// GetComprehensiveVersionInfo retrieves comprehensive version information from cache
func (c *MemoryCache) GetComprehensiveVersionInfo(owner, repo string) (map[string]string, map[string][]string, bool, error) {
	key := fmt.Sprintf("%s/%s:comprehensive", owner, repo)

	if c.verbose {
		log.Printf("Cache: Checking for cached comprehensive version info '%s'", key)
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		if c.verbose {
			log.Printf("Cache: MISS - No cached comprehensive version info found for '%s'", key)
		}
		return nil, nil, false, nil
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		if c.verbose {
			log.Printf("Cache: MISS - Cached comprehensive version info for '%s' has expired (was valid until %s)", key, entry.ExpiresAt.Format(time.RFC3339))
		}
		// Remove expired entry
		c.mutex.RUnlock()
		c.mutex.Lock()
		delete(c.data, key)
		c.mutex.Unlock()
		c.mutex.RLock()
		return nil, nil, false, nil
	}

	if entry.DataType != "comprehensive" {
		if c.verbose {
			log.Printf("Cache: MISS - Cached entry for '%s' is not comprehensive version info (type: %s)", key, entry.DataType)
		}
		return nil, nil, false, nil
	}

	if c.verbose {
		log.Printf("Cache: HIT - Found valid cached comprehensive version info for '%s' (%d versions, expires at %s)", key, len(entry.Versions), entry.ExpiresAt.Format(time.RFC3339))
	}

	return entry.Versions, entry.Aliases, true, nil
}

// SetComprehensiveVersionInfo stores comprehensive version information in the cache
func (c *MemoryCache) SetComprehensiveVersionInfo(owner, repo string, versions map[string]string, aliases map[string][]string, ttl time.Duration) error {
	key := fmt.Sprintf("%s/%s:comprehensive", owner, repo)

	if c.verbose {
		log.Printf("Cache: Storing comprehensive version info for '%s' (%d versions) with TTL %s", key, len(versions), ttl)
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	entry := &CachedVersionInfo{
		Key:       key,
		CacheTime: now,
		ExpiresAt: expiresAt,
		DataType:  "comprehensive",
		Versions:  versions,
		Aliases:   aliases,
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[key] = entry

	if c.verbose {
		log.Printf("Cache: Successfully stored comprehensive version info for '%s' (expires at %s)", key, expiresAt.Format(time.RFC3339))
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

	for key, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			delete(c.data, key)
			removed++
			if c.verbose {
				log.Printf("Cache: Removed expired entry for key '%s'", key)
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
	c.data = make(map[string]*CachedVersionInfo)
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
	refEntries := 0
	tagEntries := 0
	comprehensiveEntries := 0

	for _, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			expiredEntries++
		}

		switch entry.DataType {
		case "ref":
			refEntries++
		case "tags":
			tagEntries++
		case "comprehensive":
			comprehensiveEntries++
		}
	}

	stats["total_entries"] = totalEntries
	stats["expired_entries"] = expiredEntries
	stats["valid_entries"] = totalEntries - expiredEntries
	stats["ref_entries"] = refEntries
	stats["tag_entries"] = tagEntries
	stats["comprehensive_entries"] = comprehensiveEntries

	return stats, nil
}
