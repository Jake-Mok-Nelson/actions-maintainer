package cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteCache provides TTL-based caching using SQLite
type SQLiteCache struct {
	db *sql.DB
}

// NewSQLiteCache creates a new SQLite cache
func NewSQLiteCache(dbPath string) (Cache, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}

	cache := &SQLiteCache{db: db}

	if err := cache.initializeSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize cache schema: %w", err)
	}

	return cache, nil
}

// initializeSchema creates the cache table if it doesn't exist
func (c *SQLiteCache) initializeSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS cache_results (
		owner TEXT PRIMARY KEY,
		scan_time DATETIME NOT NULL,
		results BLOB NOT NULL,
		expires_at DATETIME NOT NULL
	);
	
	CREATE INDEX IF NOT EXISTS idx_expires_at ON cache_results(expires_at);
	`

	_, err := c.db.Exec(query)
	return err
}

// Get retrieves a cached result if it exists and hasn't expired
func (c *SQLiteCache) Get(owner string) (*CachedResult, error) {
	now := time.Now()

	query := `
	SELECT owner, scan_time, results, expires_at 
	FROM cache_results 
	WHERE owner = ? AND expires_at > ?
	`

	row := c.db.QueryRow(query, owner, now)

	var result CachedResult
	err := row.Scan(&result.Owner, &result.ScanTime, &result.Results, &result.ExpiresAt)

	if err == sql.ErrNoRows {
		return nil, nil // No cached result found
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get cached result: %w", err)
	}

	return &result, nil
}

// Set stores a result in the cache with TTL
func (c *SQLiteCache) Set(owner string, results interface{}, ttl time.Duration) error {
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	query := `
	INSERT OR REPLACE INTO cache_results (owner, scan_time, results, expires_at)
	VALUES (?, ?, ?, ?)
	`

	_, err = c.db.Exec(query, owner, now, resultsJSON, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to cache results: %w", err)
	}

	return nil
}

// CleanExpired removes expired entries from the cache
func (c *SQLiteCache) CleanExpired() error {
	now := time.Now()

	query := `DELETE FROM cache_results WHERE expires_at <= ?`

	result, err := c.db.Exec(query, now)
	if err != nil {
		return fmt.Errorf("failed to clean expired cache entries: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Cleaned %d expired cache entries\n", rowsAffected)
	}

	return nil
}

// Close closes the database connection
func (c *SQLiteCache) Close() error {
	return c.db.Close()
}

// GetStats returns cache statistics
func (c *SQLiteCache) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total entries
	var totalEntries int
	err := c.db.QueryRow("SELECT COUNT(*) FROM cache_results").Scan(&totalEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to get total entries: %w", err)
	}
	stats["total_entries"] = totalEntries

	// Expired entries
	var expiredEntries int
	now := time.Now()
	err = c.db.QueryRow("SELECT COUNT(*) FROM cache_results WHERE expires_at <= ?", now).Scan(&expiredEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired entries: %w", err)
	}
	stats["expired_entries"] = expiredEntries
	stats["valid_entries"] = totalEntries - expiredEntries

	return stats, nil
}
