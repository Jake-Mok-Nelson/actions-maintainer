package actions

import (
	"testing"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/output"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
)

// MockVersionResolver implements VersionResolver for testing
type MockVersionResolver struct {
	equivalentVersions map[string]bool   // maps "repo:v1:v2" to bool
	outdatedVersions   map[string]bool   // maps "repo:current:latest" to bool
	refResolutions     map[string]string // maps "owner/repo:ref" to SHA
}

func NewMockVersionResolver() *MockVersionResolver {
	return &MockVersionResolver{
		equivalentVersions: make(map[string]bool),
		outdatedVersions:   make(map[string]bool),
		refResolutions:     make(map[string]string),
	}
}

func (m *MockVersionResolver) AreVersionsEquivalent(repository, version1, version2 string) (bool, error) {
	key := repository + ":" + version1 + ":" + version2
	if result, exists := m.equivalentVersions[key]; exists {
		return result, nil
	}
	// Default to false if not explicitly set
	return false, nil
}

func (m *MockVersionResolver) IsVersionOutdated(repository, currentVersion, latestVersion string) (bool, error) {
	key := repository + ":" + currentVersion + ":" + latestVersion
	if result, exists := m.outdatedVersions[key]; exists {
		return result, nil
	}

	// Don't flag branch references as outdated (same logic as real resolver)
	if currentVersion == "main" || currentVersion == "master" {
		return false, nil
	}

	// Check if versions are equivalent first - if so, not outdated
	if equivalent, err := m.AreVersionsEquivalent(repository, currentVersion, latestVersion); err == nil && equivalent {
		return false, nil
	}

	// Default to checking if versions are different
	return currentVersion != latestVersion, nil
}

func (m *MockVersionResolver) ResolveRefWithCache(owner, repo, ref string) (string, error) {
	key := owner + "/" + repo + ":" + ref
	if sha, exists := m.refResolutions[key]; exists {
		return sha, nil
	}
	// Return a mock SHA if not found
	return "abc123def456ghi789jkl012mno345pqr678stu901", nil
}

// SetRefResolution sets a mock resolution for a ref to SHA
func (m *MockVersionResolver) SetRefResolution(owner, repo, ref, sha string) {
	key := owner + "/" + repo + ":" + ref
	m.refResolutions[key] = sha
}

func (m *MockVersionResolver) SetVersionsEquivalent(repository, version1, version2 string, equivalent bool) {
	key := repository + ":" + version1 + ":" + version2
	m.equivalentVersions[key] = equivalent
	// Also set the reverse mapping
	reverseKey := repository + ":" + version2 + ":" + version1
	m.equivalentVersions[reverseKey] = equivalent
}

func (m *MockVersionResolver) SetVersionOutdated(repository, currentVersion, latestVersion string, outdated bool) {
	key := repository + ":" + currentVersion + ":" + latestVersion
	m.outdatedVersions[key] = outdated
}

func TestManager_AnalyzeActions_WithoutResolver(t *testing.T) {
	manager := NewManager()

	actions := []workflow.ActionReference{
		{
			Repository: "actions/checkout",
			Version:    "v1", // deprecated according to default rules
			Context:    "job:test/step:checkout",
			FilePath:   ".github/workflows/test.yml",
		},
		{
			Repository: "actions/setup-node",
			Version:    "v2", // outdated according to default rules
			Context:    "job:test/step:setup-node",
			FilePath:   ".github/workflows/test.yml",
		},
	}

	issues := manager.AnalyzeActions(actions)

	// Should find issues using traditional string-based comparison
	if len(issues) < 2 {
		t.Errorf("Expected at least 2 issues, got %d", len(issues))
	}

	// Check for deprecated version issue
	foundDeprecated := false
	for _, issue := range issues {
		if issue.Repository == "actions/checkout" && issue.IssueType == "deprecated" {
			foundDeprecated = true
		}
	}
	if !foundDeprecated {
		t.Error("Expected to find deprecated version issue for actions/checkout@v1")
	}
}

func TestManager_AnalyzeActions_WithResolver_EquivalentVersions(t *testing.T) {
	resolver := NewMockVersionResolver()
	manager := NewManagerWithResolver(resolver)

	// Set up scenario where v4 and v4.2.1 are equivalent (same SHA)
	resolver.SetVersionsEquivalent("actions/checkout", "v4.2.1", "v4", true)

	actions := []workflow.ActionReference{
		{
			Repository: "actions/checkout",
			Version:    "v4.2.1", // equivalent to latest v4
			Context:    "job:test/step:checkout",
			FilePath:   ".github/workflows/test.yml",
		},
	}

	issues := manager.AnalyzeActions(actions)

	// Should not find outdated issue since v4.2.1 is equivalent to v4
	for _, issue := range issues {
		if issue.Repository == "actions/checkout" && issue.IssueType == "outdated" {
			t.Errorf("Expected no outdated issue for equivalent versions, but found: %+v", issue)
		}
	}
}

func TestManager_AnalyzeActions_WithResolver_NonEquivalentVersions(t *testing.T) {
	resolver := NewMockVersionResolver()
	manager := NewManagerWithResolver(resolver)

	// Set up scenario where v3 and v4 are NOT equivalent (different SHAs)
	resolver.SetVersionsEquivalent("actions/checkout", "v3", "v4", false)

	actions := []workflow.ActionReference{
		{
			Repository: "actions/checkout",
			Version:    "v3", // not equivalent to latest v4
			Context:    "job:test/step:checkout",
			FilePath:   ".github/workflows/test.yml",
		},
	}

	issues := manager.AnalyzeActions(actions)

	// Should find outdated issue since v3 is not equivalent to v4
	foundOutdated := false
	for _, issue := range issues {
		if issue.Repository == "actions/checkout" && issue.IssueType == "outdated" {
			foundOutdated = true
		}
	}
	if !foundOutdated {
		t.Error("Expected to find outdated issue for non-equivalent versions")
	}
}

func TestManager_AnalyzeActions_WithResolver_ResolverFailure(t *testing.T) {
	// Create a resolver that will return false (fall back to string comparison)
	resolver := NewMockVersionResolver()
	manager := NewManagerWithResolver(resolver)

	// Don't set any equivalencies, so resolver will return false/error

	actions := []workflow.ActionReference{
		{
			Repository: "actions/checkout",
			Version:    "v2", // different from latest v4
			Context:    "job:test/step:checkout",
			FilePath:   ".github/workflows/test.yml",
		},
	}

	issues := manager.AnalyzeActions(actions)

	// Should still find issues using fallback string comparison
	foundOutdated := false
	for _, issue := range issues {
		if issue.Repository == "actions/checkout" && issue.IssueType == "outdated" {
			foundOutdated = true
		}
	}
	if !foundOutdated {
		t.Error("Expected to find outdated issue when resolver fails")
	}
}

func TestManager_IsOutdatedForRepository_WithResolver(t *testing.T) {
	resolver := NewMockVersionResolver()
	manager := NewManagerWithResolver(resolver)

	// Test equivalent versions
	resolver.SetVersionsEquivalent("actions/checkout", "v4.1.0", "v4", true)

	isOutdated := manager.isOutdatedForRepository("actions/checkout", "v4.1.0", "v4")
	if isOutdated {
		t.Error("Expected equivalent versions to not be considered outdated")
	}

	// Test non-equivalent versions
	resolver.SetVersionsEquivalent("actions/checkout", "v3", "v4", false)

	isOutdated = manager.isOutdatedForRepository("actions/checkout", "v3", "v4")
	if !isOutdated {
		t.Error("Expected non-equivalent versions to be considered outdated")
	}
}

func TestManager_IsOutdatedForRepository_WithoutRepository(t *testing.T) {
	resolver := NewMockVersionResolver()
	manager := NewManagerWithResolver(resolver)

	// When repository is empty, should fall back to traditional comparison
	isOutdated := manager.isOutdatedForRepository("", "v1", "v4")
	if !isOutdated {
		t.Error("Expected v1 to be outdated compared to v4 using traditional comparison")
	}
}

func TestManager_BranchReferences(t *testing.T) {
	resolver := NewMockVersionResolver()
	manager := NewManagerWithResolver(resolver)

	// Branch references should never be considered outdated
	isOutdated := manager.isOutdatedForRepository("actions/checkout", "main", "v4")
	if isOutdated {
		t.Error("Expected 'main' branch reference to not be considered outdated")
	}

	isOutdated = manager.isOutdatedForRepository("actions/checkout", "master", "v4")
	if isOutdated {
		t.Error("Expected 'master' branch reference to not be considered outdated")
	}
}

func TestManager_AnalyzeActions_MultipleAliases(t *testing.T) {
	resolver := NewMockVersionResolver()
	manager := NewManagerWithResolver(resolver)

	// Set up scenario where multiple versions are equivalent to latest
	resolver.SetVersionsEquivalent("actions/checkout", "v4", "v4", true)
	resolver.SetVersionsEquivalent("actions/checkout", "v4.1.0", "v4", true)
	resolver.SetVersionsEquivalent("actions/checkout", "v4.1.1", "v4", true)

	actions := []workflow.ActionReference{
		{
			Repository: "actions/checkout",
			Version:    "v4.1.0",
			Context:    "job:test/step:checkout",
			FilePath:   ".github/workflows/test.yml",
		},
		{
			Repository: "actions/checkout",
			Version:    "v4.1.1",
			Context:    "job:test/step:checkout-2",
			FilePath:   ".github/workflows/test.yml",
		},
	}

	issues := manager.AnalyzeActions(actions)

	// Should not find any outdated issues since all versions are equivalent to latest
	for _, issue := range issues {
		if issue.IssueType == "outdated" {
			t.Errorf("Expected no outdated issues for equivalent versions, but found: %+v", issue)
		}
	}
}

// TestDetectVersionFormat tests the version format detection
func TestDetectVersionFormat(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		version  string
		expected VersionFormat
	}{
		{"v4", VersionFormatTag},
		{"v4.1.0", VersionFormatTag},
		{"abc123def456", VersionFormatSHA},
		{"abc123def456abc789def012abc345def678abc90", VersionFormatSHA}, // 40 char SHA
		{"main", VersionFormatBranch},
		{"master", VersionFormatBranch},
		{"feature-branch", VersionFormatTag}, // non-standard branch names default to tag
		{"1.2.3", VersionFormatTag},          // non-v prefixed versions default to tag
	}

	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			actual := manager.detectVersionFormat(test.version)
			if actual != test.expected {
				t.Errorf("Expected format %v for version %s, got %v", test.expected, test.version, actual)
			}
		})
	}
}

// TestSuggestLikeForLikeVersion_WithResolver tests like-for-like suggestions with a resolver
func TestSuggestLikeForLikeVersion_WithResolver(t *testing.T) {
	resolver := NewMockVersionResolver()
	manager := NewManagerWithResolver(resolver)

	// Set up mock resolution for v4 tag to a specific SHA
	expectedSHA := "def456ghi789jkl012mno345pqr678stu901abc123"
	resolver.SetRefResolution("actions", "checkout", "v4", expectedSHA)

	tests := []struct {
		name               string
		currentVersion     string
		latestTag          string
		expectedSuggestion string
	}{
		{
			name:               "Tag to tag",
			currentVersion:     "v3",
			latestTag:          "v4",
			expectedSuggestion: "v4",
		},
		{
			name:               "SHA to SHA",
			currentVersion:     "abc123def456",
			latestTag:          "v4",
			expectedSuggestion: expectedSHA,
		},
		{
			name:               "Branch to tag",
			currentVersion:     "main",
			latestTag:          "v4",
			expectedSuggestion: "v4",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := manager.suggestLikeForLikeVersion("actions/checkout", test.currentVersion, test.latestTag)
			if actual != test.expectedSuggestion {
				t.Errorf("Expected suggestion %s, got %s", test.expectedSuggestion, actual)
			}
		})
	}
}

// TestSuggestLikeForLikeVersion_WithoutResolver tests like-for-like suggestions without a resolver
func TestSuggestLikeForLikeVersion_WithoutResolver(t *testing.T) {
	manager := NewManager() // No resolver

	tests := []struct {
		name               string
		currentVersion     string
		latestTag          string
		expectedSuggestion string
	}{
		{
			name:               "Tag to tag",
			currentVersion:     "v3",
			latestTag:          "v4",
			expectedSuggestion: "v4",
		},
		{
			name:               "SHA fallback to tag",
			currentVersion:     "abc123def456",
			latestTag:          "v4",
			expectedSuggestion: "v4", // Falls back to tag when no resolver
		},
		{
			name:               "Branch to tag",
			currentVersion:     "main",
			latestTag:          "v4",
			expectedSuggestion: "v4",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := manager.suggestLikeForLikeVersion("actions/checkout", test.currentVersion, test.latestTag)
			if actual != test.expectedSuggestion {
				t.Errorf("Expected suggestion %s, got %s", test.expectedSuggestion, actual)
			}
		})
	}
}

// TestAnalyzeActions_LikeForLikeSuggestions tests that the analyze method uses like-for-like suggestions
func TestAnalyzeActions_LikeForLikeSuggestions(t *testing.T) {
	resolver := NewMockVersionResolver()
	manager := NewManagerWithResolver(resolver)

	// Set up v3 as outdated compared to v4
	resolver.SetVersionsEquivalent("actions/checkout", "v3", "v4", false)
	resolver.SetVersionsEquivalent("actions/checkout", "abc123def456", "v4", false)

	// Set up mock SHA resolution for v4
	expectedSHA := "def456ghi789jkl012mno345pqr678stu901abc123"
	resolver.SetRefResolution("actions", "checkout", "v4", expectedSHA)

	actions := []workflow.ActionReference{
		{
			Repository: "actions/checkout",
			Version:    "v3", // Tag version
			Context:    "job:test/step:checkout-tag",
			FilePath:   ".github/workflows/test.yml",
		},
		{
			Repository: "actions/checkout",
			Version:    "abc123def456", // SHA version
			Context:    "job:test/step:checkout-sha",
			FilePath:   ".github/workflows/test.yml",
		},
	}

	issues := manager.AnalyzeActions(actions)

	// Should find 2 outdated issues
	var tagIssue, shaIssue *output.ActionIssue
	for i := range issues {
		if issues[i].CurrentVersion == "v3" {
			tagIssue = &issues[i]
		} else if issues[i].CurrentVersion == "abc123def456" {
			shaIssue = &issues[i]
		}
	}

	if tagIssue == nil {
		t.Fatal("Expected to find outdated issue for tag version v3")
	}
	if tagIssue.SuggestedVersion != "v4" {
		t.Errorf("Expected tag suggestion v4, got %s", tagIssue.SuggestedVersion)
	}

	if shaIssue == nil {
		t.Fatal("Expected to find outdated issue for SHA version")
	}
	if shaIssue.SuggestedVersion != expectedSHA {
		t.Errorf("Expected SHA suggestion %s, got %s", expectedSHA, shaIssue.SuggestedVersion)
	}
}
