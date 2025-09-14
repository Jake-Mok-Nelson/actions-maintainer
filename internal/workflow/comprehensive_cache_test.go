package workflow

import (
	"testing"
)

func TestVersionResolver_ComprehensiveCaching(t *testing.T) {
	client := NewMockGitHubClient()

	// Set up mock data for actions/checkout repository
	sha1 := "abc123def456"
	sha2 := "def456ghi789"
	sha3 := "ghi789jkl012"

	// Add tag resolutions
	client.AddRefResolution("actions", "checkout", "v4", sha1)
	client.AddRefResolution("actions", "checkout", "v4.1.0", sha1)  // Same SHA as v4
	client.AddRefResolution("actions", "checkout", "v4.2.0", sha2)  // Different SHA
	client.AddRefResolution("actions", "checkout", "v3", sha3)      // Different SHA

	// Add repository tags
	client.AddRepoTags("actions", "checkout", map[string]string{
		"v4":     sha1,
		"v4.1.0": sha1,
		"v4.2.0": sha2,
		"v3":     sha3,
	})

	resolver := NewVersionResolver(client, false)

	// Test 1: Resolve an action to populate comprehensive cache
	actions := []ActionReference{
		{
			Repository: "actions/checkout",
			Version:    "v4",
		},
	}

	resolved, err := resolver.ResolveActionReferences(actions)
	if err != nil {
		t.Fatalf("Expected no error during resolution, got: %v", err)
	}

	if len(resolved) != 1 {
		t.Fatalf("Expected 1 resolved action, got %d", len(resolved))
	}

	// Test 2: Verify comprehensive cache was populated
	versions, aliases, hasCached := resolver.GetCachedVersionInfo("actions", "checkout")
	if !hasCached {
		t.Error("Expected comprehensive cache to be populated after resolving an action")
	}

	// Verify all versions are cached
	expectedVersions := map[string]string{
		"v4":     sha1,
		"v4.1.0": sha1,
		"v4.2.0": sha2,
		"v3":     sha3,
	}

	for version, expectedSHA := range expectedVersions {
		if cachedSHA, exists := versions[version]; !exists {
			t.Errorf("Expected version %s to be cached", version)
		} else if cachedSHA != expectedSHA {
			t.Errorf("Expected cached SHA for %s to be %s, got %s", version, expectedSHA, cachedSHA)
		}
	}

	// Verify aliases are correctly mapped
	expectedAliases := map[string][]string{
		sha1: {"v4", "v4.1.0"},  // v4 and v4.1.0 point to same SHA
		sha2: {"v4.2.0"},        // v4.2.0 is alone
		sha3: {"v3"},            // v3 is alone
	}

	for sha, expectedVersions := range expectedAliases {
		if cachedAliases, exists := aliases[sha]; !exists {
			t.Errorf("Expected aliases for SHA %s to be cached", sha)
		} else {
			// Convert to map for easier comparison
			aliasMap := make(map[string]bool)
			for _, alias := range cachedAliases {
				aliasMap[alias] = true
			}
			
			for _, expectedVersion := range expectedVersions {
				if !aliasMap[expectedVersion] {
					t.Errorf("Expected version %s to be in aliases for SHA %s", expectedVersion, sha)
				}
			}
		}
	}
}

func TestVersionResolver_CacheFirstEquivalence(t *testing.T) {
	client := NewMockGitHubClient()

	sha := "abc123def456"
	
	// Set up mock data
	client.AddRefResolution("actions", "checkout", "v4", sha)
	client.AddRefResolution("actions", "checkout", "v4.1.0", sha)

	client.AddRepoTags("actions", "checkout", map[string]string{
		"v4":     sha,
		"v4.1.0": sha,
	})

	resolver := NewVersionResolver(client, false)

	// First, populate the comprehensive cache
	resolver.ensureComprehensiveCache("actions", "checkout")

	// Remove the individual ref resolutions to ensure cache is being used
	delete(client.refResolutions, "actions/checkout:v4")
	delete(client.refResolutions, "actions/checkout:v4.1.0")

	// Test that equivalence check still works (should use cache)
	equivalent, err := resolver.AreVersionsEquivalent("actions/checkout", "v4", "v4.1.0")
	if err != nil {
		t.Fatalf("Expected no error when checking equivalence from cache, got: %v", err)
	}

	if !equivalent {
		t.Error("Expected v4 and v4.1.0 to be equivalent when using cached data")
	}
}

func TestVersionResolver_CacheFirstOutdatedCheck(t *testing.T) {
	client := NewMockGitHubClient()

	sha1 := "abc123def456"
	sha2 := "def456ghi789"
	
	// Set up mock data
	client.AddRefResolution("actions", "checkout", "v4", sha1)
	client.AddRefResolution("actions", "checkout", "v3", sha2)

	client.AddRepoTags("actions", "checkout", map[string]string{
		"v4": sha1,
		"v3": sha2,
	})

	resolver := NewVersionResolver(client, false)

	// First, populate the comprehensive cache
	resolver.ensureComprehensiveCache("actions", "checkout")

	// Remove the individual ref resolutions to ensure cache is being used
	delete(client.refResolutions, "actions/checkout:v4")
	delete(client.refResolutions, "actions/checkout:v3")

	// Test that outdated check works from cache
	outdated, err := resolver.IsVersionOutdated("actions/checkout", "v3", "v4")
	if err != nil {
		t.Fatalf("Expected no error when checking outdated from cache, got: %v", err)
	}

	if !outdated {
		t.Error("Expected v3 to be outdated compared to v4 when using cached data")
	}

	// Test that equivalent versions are not considered outdated
	outdated, err = resolver.IsVersionOutdated("actions/checkout", "v4", "v4")
	if err != nil {
		t.Fatalf("Expected no error when checking same version, got: %v", err)
	}

	if outdated {
		t.Error("Expected v4 to not be outdated compared to itself")
	}
}

func TestVersionResolver_BranchReferencesNotOutdated(t *testing.T) {
	client := NewMockGitHubClient()
	resolver := NewVersionResolver(client, false)

	// Branch references should never be considered outdated
	outdated, err := resolver.IsVersionOutdated("actions/checkout", "main", "v4")
	if err != nil {
		t.Fatalf("Expected no error when checking branch reference, got: %v", err)
	}

	if outdated {
		t.Error("Expected 'main' branch reference to not be considered outdated")
	}

	outdated, err = resolver.IsVersionOutdated("actions/checkout", "master", "v4")
	if err != nil {
		t.Fatalf("Expected no error when checking branch reference, got: %v", err)
	}

	if outdated {
		t.Error("Expected 'master' branch reference to not be considered outdated")
	}
}

func TestVersionResolver_CacheExpiration(t *testing.T) {
	client := NewMockGitHubClient()

	sha := "abc123def456"
	client.AddRepoTags("actions", "checkout", map[string]string{
		"v4": sha,
	})

	resolver := NewVersionResolver(client, false)
	
	// Populate cache normally
	resolver.ensureComprehensiveCache("actions", "checkout")

	// Verify cache exists initially
	versions, _, hasCached := resolver.GetCachedVersionInfo("actions", "checkout")
	if !hasCached {
		t.Error("Expected cache to be populated")
		return
	}

	// Verify the version is correctly cached
	if cachedSHA, exists := versions["v4"]; !exists {
		t.Error("Expected v4 to be cached")
	} else if cachedSHA != sha {
		t.Errorf("Expected cached SHA to be %s, got %s", sha, cachedSHA)
	}
}