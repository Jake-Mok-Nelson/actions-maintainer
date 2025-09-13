package workflow

import (
	"fmt"
	"testing"
)

// MockGitHubClient implements GitHubClient for testing
type MockGitHubClient struct {
	refResolutions map[string]string            // maps "owner/repo:ref" to SHA
	repoTags       map[string]map[string]string // maps "owner/repo" to tag->SHA map
}

func NewMockGitHubClient() *MockGitHubClient {
	return &MockGitHubClient{
		refResolutions: make(map[string]string),
		repoTags:       make(map[string]map[string]string),
	}
}

func (m *MockGitHubClient) ResolveRef(owner, repo, ref string) (string, error) {
	key := fmt.Sprintf("%s/%s:%s", owner, repo, ref)
	if sha, exists := m.refResolutions[key]; exists {
		return sha, nil
	}
	return "", fmt.Errorf("reference not found: %s", key)
}

func (m *MockGitHubClient) GetTagsForRepo(owner, repo string) (map[string]string, error) {
	key := fmt.Sprintf("%s/%s", owner, repo)
	if tags, exists := m.repoTags[key]; exists {
		return tags, nil
	}
	return make(map[string]string), nil
}

func (m *MockGitHubClient) AddRefResolution(owner, repo, ref, sha string) {
	key := fmt.Sprintf("%s/%s:%s", owner, repo, ref)
	m.refResolutions[key] = sha
}

func (m *MockGitHubClient) AddRepoTags(owner, repo string, tags map[string]string) {
	key := fmt.Sprintf("%s/%s", owner, repo)
	m.repoTags[key] = tags
}

func TestVersionResolver_SkipResolution(t *testing.T) {
	client := NewMockGitHubClient()
	resolver := NewVersionResolver(client, true) // skipResolve = true

	actions := []ActionReference{
		{
			Repository: "actions/checkout",
			Version:    "v4",
		},
		{
			Repository: "actions/setup-node",
			Version:    "v3",
		},
	}

	resolved, err := resolver.ResolveActionReferences(actions)
	if err != nil {
		t.Fatalf("Expected no error when skipping resolution, got: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("Expected 2 resolved actions, got %d", len(resolved))
	}

	// When skipping resolution, SHA should be empty
	for _, action := range resolved {
		if action.ResolvedSHA != "" {
			t.Errorf("Expected empty ResolvedSHA when skipping resolution, got: %s", action.ResolvedSHA)
		}
		if len(action.Aliases) != 0 {
			t.Errorf("Expected no aliases when skipping resolution, got: %v", action.Aliases)
		}
	}
}

func TestVersionResolver_BasicResolution(t *testing.T) {
	client := NewMockGitHubClient()

	// Setup mock data
	sha := "abc123def456"
	client.AddRefResolution("actions", "checkout", "v4", sha)
	client.AddRepoTags("actions", "checkout", map[string]string{
		"v4":     sha,
		"v4.2.1": sha,
		"v4.2.0": "different_sha",
	})

	resolver := NewVersionResolver(client, false) // skipResolve = false

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

	action := resolved[0]
	if action.ResolvedSHA != sha {
		t.Errorf("Expected ResolvedSHA to be %s, got: %s", sha, action.ResolvedSHA)
	}

	// Should find v4.2.1 as an alias (same SHA)
	expectedAliases := []string{"v4.2.1"}
	if len(action.Aliases) != len(expectedAliases) {
		t.Errorf("Expected %d aliases, got %d: %v", len(expectedAliases), len(action.Aliases), action.Aliases)
	}

	found := false
	for _, alias := range action.Aliases {
		if alias == "v4.2.1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find v4.2.1 in aliases, got: %v", action.Aliases)
	}
}

func TestVersionResolver_AreVersionsEquivalent_SkipResolution(t *testing.T) {
	client := NewMockGitHubClient()
	resolver := NewVersionResolver(client, true) // skipResolve = true

	// Should fall back to string comparison
	equivalent, err := resolver.AreVersionsEquivalent("actions/checkout", "v4", "v4")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !equivalent {
		t.Error("Expected v4 == v4 to be equivalent")
	}

	equivalent, err = resolver.AreVersionsEquivalent("actions/checkout", "v4", "v3")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if equivalent {
		t.Error("Expected v4 != v3 to not be equivalent")
	}
}

func TestVersionResolver_AreVersionsEquivalent_WithResolution(t *testing.T) {
	client := NewMockGitHubClient()

	// Setup mock data - v4 and v4.2.1 point to same SHA
	sha := "abc123def456"
	client.AddRefResolution("actions", "checkout", "v4", sha)
	client.AddRefResolution("actions", "checkout", "v4.2.1", sha)
	client.AddRefResolution("actions", "checkout", "v3", "different_sha")

	resolver := NewVersionResolver(client, false) // skipResolve = false

	// Test equivalent versions (same SHA)
	equivalent, err := resolver.AreVersionsEquivalent("actions/checkout", "v4", "v4.2.1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !equivalent {
		t.Error("Expected v4 and v4.2.1 to be equivalent (same SHA)")
	}

	// Test non-equivalent versions (different SHA)
	equivalent, err = resolver.AreVersionsEquivalent("actions/checkout", "v4", "v3")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if equivalent {
		t.Error("Expected v4 and v3 to not be equivalent (different SHA)")
	}
}

func TestVersionResolver_ErrorHandling(t *testing.T) {
	client := NewMockGitHubClient()
	resolver := NewVersionResolver(client, false) // skipResolve = false

	actions := []ActionReference{
		{
			Repository: "nonexistent/action",
			Version:    "v1",
		},
	}

	// Should handle resolution failures gracefully
	resolved, err := resolver.ResolveActionReferences(actions)
	if err != nil {
		t.Fatalf("Expected no error even with failed resolution, got: %v", err)
	}

	if len(resolved) != 1 {
		t.Fatalf("Expected 1 resolved action, got %d", len(resolved))
	}

	action := resolved[0]
	if action.ResolvedSHA != "" {
		t.Errorf("Expected empty ResolvedSHA on resolution failure, got: %s", action.ResolvedSHA)
	}
}

func TestVersionResolver_Caching(t *testing.T) {
	client := NewMockGitHubClient()

	sha := "abc123def456"
	client.AddRefResolution("actions", "checkout", "v4", sha)

	resolver := NewVersionResolver(client, false)

	// First call should hit the API
	sha1, err := resolver.resolveRefWithCache("actions", "checkout", "v4")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if sha1 != sha {
		t.Errorf("Expected SHA %s, got %s", sha, sha1)
	}

	// Second call should use cache (we can verify this by removing the mock data)
	delete(client.refResolutions, "actions/checkout:v4")

	sha2, err := resolver.resolveRefWithCache("actions", "checkout", "v4")
	if err != nil {
		t.Fatalf("Expected no error on cached call, got: %v", err)
	}
	if sha2 != sha {
		t.Errorf("Expected cached SHA %s, got %s", sha, sha2)
	}
}

func TestVersionResolver_InvalidRepository(t *testing.T) {
	client := NewMockGitHubClient()
	resolver := NewVersionResolver(client, false)

	actions := []ActionReference{
		{
			Repository: "invalid-format",
			Version:    "v1",
		},
	}

	resolved, err := resolver.ResolveActionReferences(actions)
	if err != nil {
		t.Fatalf("Expected no error even with invalid repository format, got: %v", err)
	}

	if len(resolved) != 1 {
		t.Fatalf("Expected 1 resolved action, got %d", len(resolved))
	}

	action := resolved[0]
	if action.ResolvedSHA != "" {
		t.Errorf("Expected empty ResolvedSHA for invalid repository format, got: %s", action.ResolvedSHA)
	}
}
