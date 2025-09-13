package workflow

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// GitHubClient interface defines the methods needed from the GitHub client for version resolution
type GitHubClient interface {
	ResolveRef(owner, repo, ref string) (string, error)
	GetTagsForRepo(owner, repo string) (map[string]string, error)
}

// VersionResolver handles resolution of version aliases to commit SHAs
//
// Alias Resolution Design:
// Version aliases in GitHub Actions workflows can refer to the same underlying commit
// using different references (e.g., v1, v1.2.4, commit SHA). This resolver enables
// intelligent version comparison by resolving all references to their commit SHAs.
//
// Key Design Principles:
// 1. Performance: Uses caching to minimize GitHub API calls with 1-hour TTL
// 2. Resilience: Falls back to string comparison on API failures
// 3. Flexibility: --skip-resolution flag allows purely string-based matching
// 4. Accuracy: SHA-based comparison provides authoritative version equivalence
//
// Example: If v1 tag and commit SHA abc123 both point to the same commit,
// they are considered equivalent even though the strings differ.
type VersionResolver struct {
	client      GitHubClient
	skipResolve bool
	cache       map[string]*cacheEntry
	cacheMutex  sync.RWMutex
	cacheTTL    time.Duration
}

// cacheEntry represents a cached resolution result
type cacheEntry struct {
	sha       string
	timestamp time.Time
	tags      map[string]string // maps tag names to SHAs for a repository
}

// ResolvedAction represents an action with resolved version information
type ResolvedAction struct {
	ActionReference
	ResolvedSHA string   // The commit SHA this version resolves to
	Aliases     []string // Other version references that resolve to the same SHA
}

// NewVersionResolver creates a new version resolver
func NewVersionResolver(client GitHubClient, skipResolve bool) *VersionResolver {
	return &VersionResolver{
		client:      client,
		skipResolve: skipResolve,
		cache:       make(map[string]*cacheEntry),
		cacheTTL:    time.Hour, // Cache for 1 hour
	}
}

// ResolveActionReferences resolves version aliases for a list of action references
func (vr *VersionResolver) ResolveActionReferences(actions []ActionReference) ([]ResolvedAction, error) {
	if vr.skipResolve {
		// Skip resolution, just convert to ResolvedAction without SHA resolution
		resolved := make([]ResolvedAction, len(actions))
		for i, action := range actions {
			resolved[i] = ResolvedAction{
				ActionReference: action,
				ResolvedSHA:     "", // Empty when skipping resolution
				Aliases:         []string{},
			}
		}
		return resolved, nil
	}

	var resolved []ResolvedAction
	for _, action := range actions {
		resolvedAction, err := vr.resolveAction(action)
		if err != nil {
			// If resolution fails, fall back to unresolved action
			// This ensures the tool doesn't break on API failures
			resolved = append(resolved, ResolvedAction{
				ActionReference: action,
				ResolvedSHA:     "",
				Aliases:         []string{},
			})
		} else {
			resolved = append(resolved, resolvedAction)
		}
	}

	return resolved, nil
}

// resolveAction resolves a single action reference to its commit SHA and finds aliases
func (vr *VersionResolver) resolveAction(action ActionReference) (ResolvedAction, error) {
	// Parse the repository from the action reference
	parts := strings.Split(action.Repository, "/")
	if len(parts) != 2 {
		return ResolvedAction{}, fmt.Errorf("invalid repository format: %s", action.Repository)
	}
	owner, repo := parts[0], parts[1]

	// Resolve the version to a commit SHA
	sha, err := vr.resolveRefWithCache(owner, repo, action.Version)
	if err != nil {
		return ResolvedAction{}, fmt.Errorf("failed to resolve %s@%s: %w", action.Repository, action.Version, err)
	}

	// Find aliases (other tags that point to the same commit)
	aliases, err := vr.findAliases(owner, repo, sha, action.Version)
	if err != nil {
		// Don't fail if we can't find aliases, just proceed without them
		aliases = []string{}
	}

	return ResolvedAction{
		ActionReference: action,
		ResolvedSHA:     sha,
		Aliases:         aliases,
	}, nil
}

// resolveRefWithCache resolves a reference to a commit SHA with caching
func (vr *VersionResolver) resolveRefWithCache(owner, repo, ref string) (string, error) {
	cacheKey := fmt.Sprintf("%s/%s:%s", owner, repo, ref)

	vr.cacheMutex.RLock()
	if entry, exists := vr.cache[cacheKey]; exists {
		if time.Since(entry.timestamp) < vr.cacheTTL {
			vr.cacheMutex.RUnlock()
			return entry.sha, nil
		}
	}
	vr.cacheMutex.RUnlock()

	// Resolve using GitHub API
	sha, err := vr.client.ResolveRef(owner, repo, ref)
	if err != nil {
		return "", err
	}

	// Cache the result
	vr.cacheMutex.Lock()
	vr.cache[cacheKey] = &cacheEntry{
		sha:       sha,
		timestamp: time.Now(),
	}
	vr.cacheMutex.Unlock()

	return sha, nil
}

// findAliases finds other version references that resolve to the same commit SHA
func (vr *VersionResolver) findAliases(owner, repo, targetSHA, currentVersion string) ([]string, error) {
	// Get all tags for the repository with caching
	tags, err := vr.getTagsWithCache(owner, repo)
	if err != nil {
		return nil, err
	}

	var aliases []string
	for tagName, tagSHA := range tags {
		// Skip the current version itself
		if tagName == currentVersion {
			continue
		}

		// If this tag points to the same commit, it's an alias
		if tagSHA == targetSHA {
			aliases = append(aliases, tagName)
		}
	}

	return aliases, nil
}

// getTagsWithCache gets all tags for a repository with caching
func (vr *VersionResolver) getTagsWithCache(owner, repo string) (map[string]string, error) {
	cacheKey := fmt.Sprintf("%s/%s:tags", owner, repo)

	vr.cacheMutex.RLock()
	if entry, exists := vr.cache[cacheKey]; exists {
		if time.Since(entry.timestamp) < vr.cacheTTL && entry.tags != nil {
			vr.cacheMutex.RUnlock()
			return entry.tags, nil
		}
	}
	vr.cacheMutex.RUnlock()

	// Fetch tags using GitHub API
	tags, err := vr.client.GetTagsForRepo(owner, repo)
	if err != nil {
		return nil, err
	}

	// Cache the result
	vr.cacheMutex.Lock()
	vr.cache[cacheKey] = &cacheEntry{
		tags:      tags,
		timestamp: time.Now(),
	}
	vr.cacheMutex.Unlock()

	return tags, nil
}

// AreVersionsEquivalent checks if two versions are equivalent (resolve to same SHA)
// This is used by the actions manager for version comparison.
//
// Alias Resolution Logic:
// When skipResolve is false, this method resolves both versions to their commit SHAs
// using the GitHub API and compares the SHAs for equivalence. This allows different
// version references (e.g., v1, v1.2.4, commit SHA) to be considered equivalent
// if they point to the same underlying commit.
//
// Fallback Behavior:
// - If skipResolve is true: Uses string comparison only
// - If API resolution fails: Falls back to string comparison
// - If repository format is invalid: Falls back to string comparison
//
// This design ensures the tool remains functional even when GitHub API access
// is limited or unavailable, while providing enhanced accuracy when possible.
func (vr *VersionResolver) AreVersionsEquivalent(repository, version1, version2 string) (bool, error) {
	if vr.skipResolve {
		// Fall back to string comparison when resolution is skipped
		return version1 == version2, nil
	}

	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid repository format: %s", repository)
	}
	owner, repo := parts[0], parts[1]

	sha1, err := vr.resolveRefWithCache(owner, repo, version1)
	if err != nil {
		// Fall back to string comparison on resolution failure
		return version1 == version2, nil
	}

	sha2, err := vr.resolveRefWithCache(owner, repo, version2)
	if err != nil {
		// Fall back to string comparison on resolution failure
		return version1 == version2, nil
	}

	return sha1 == sha2, nil
}
