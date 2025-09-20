package output

import (
	"strings"
	"testing"
	"time"
)

// TestCreateVersionComparisonCell tests the new version comparison visualization
func TestCreateVersionComparisonCell(t *testing.T) {
	result := &ScanResult{
		Owner:    "testorg",
		ScanTime: time.Now(),
		Summary: Summary{
			UniqueActions: map[string]ActionUsageStat{
				"actions/checkout": {
					UsageCount: 5,
					Versions: map[string]int{
						"v3": 3,
						"v4": 2,
					},
				},
				"actions/setup-node": {
					UsageCount: 3,
					Versions: map[string]int{
						"v2": 2,
						"v4": 1,
					},
				},
			},
		},
		Repositories: []RepositoryResult{
			{
				Name: "repo1",
				Issues: []ActionIssue{
					{
						Repository:       "actions/checkout",
						CurrentVersion:   "v3",
						SuggestedVersion: "v4",
						IssueType:        "outdated",
						Severity:         "medium",
					},
					{
						Repository:       "actions/setup-node",
						CurrentVersion:   "v2",
						SuggestedVersion: "v4",
						IssueType:        "outdated",
						Severity:         "high",
					},
				},
			},
		},
	}

	cell := createVersionComparisonCell(result)

	if cell.CellType != "markdown" {
		t.Errorf("Expected cell type 'markdown', got %s", cell.CellType)
	}

	source := strings.Join(cell.Source, "")

	// Test for presence of key visual elements
	expectedElements := []string{
		"🔄 Version Comparison Dashboard",
		"📊 Current vs Recommended Versions",
		"actions/checkout",
		"actions/setup-node", 
		"🟠 High",
		"🟡 Medium",
		"➡️",
		"▰▱▱▱▱ 20%",
		"▰▰▱▱▱ 40%",
		"📈 Version Upgrade Impact Analysis",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(source, expected) {
			t.Errorf("Expected element '%s' not found in output", expected)
		}
	}

	t.Log("Version comparison cell output preview:")
	t.Log(source[:500] + "...") // Log first 500 chars
}

// TestCreateUpgradeFlowCell tests the upgrade flow diagrams
func TestCreateUpgradeFlowCell(t *testing.T) {
	result := &ScanResult{
		Repositories: []RepositoryResult{
			{
				Issues: []ActionIssue{
					{
						Repository:       "actions/checkout",
						CurrentVersion:   "v3",
						SuggestedVersion: "v4",
						IssueType:        "outdated",
						Severity:         "high",
					},
					{
						Repository:       "actions/setup-node",
						CurrentVersion:   "v2",
						SuggestedVersion: "v4",
						IssueType:        "migration",
						Severity:         "medium",
						MigrationTarget:  "new-org/setup-node@v4",
					},
				},
			},
		},
	}

	cell := createUpgradeFlowCell(result)

	source := strings.Join(cell.Source, "")

	expectedElements := []string{
		"🔀 Upgrade Flow Diagrams",
		"🔄 `actions/checkout` Upgrade Path",
		"```mermaid",
		"flowchart LR",
		"Vv3 --> Vv4",
		"#### Upgrade Steps:",
		"🟠 Update from `v3` to `v4` (high priority)",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(source, expected) {
			t.Errorf("Expected element '%s' not found in output", expected)
		}
	}

	t.Log("Upgrade flow cell output preview:")
	t.Log(source[:500] + "...")
}

// TestEnhancedSummaryCell tests the enhanced summary with visual charts
func TestEnhancedSummaryCell(t *testing.T) {
	result := &ScanResult{
		Summary: Summary{
			TotalActions: 10,
			IssuesByType: map[string]int{
				"outdated":   4,
				"deprecated": 2,
				"migration":  1,
			},
			IssuesBySeverity: map[string]int{
				"critical": 1,
				"high":     2,
				"medium":   3,
				"low":      1,
			},
		},
	}

	cell := createSummaryCell(result)

	source := strings.Join(cell.Source, "")

	expectedElements := []string{
		"📈 Issue Breakdown & Visual Summary",
		"📊 Visual Overview",
		"📅 Outdated", // Changed from format string to actual content
		"█", // Visual bars
		"▱", // Empty bars
		"🌡️ Severity Distribution",
		"🔴",
		"🟠",
		"🟡",
		"🟢",
		"🏥 Repository Health Score",
		"💡 Recommendations",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(source, expected) {
			t.Errorf("Expected element '%s' not found in output", expected)
		}
	}

	// Test health score calculation
	if !strings.Contains(source, "30.0%") { // (10-7)/10 * 100 = 30%
		t.Error("Expected health score calculation not found")
	}

	t.Log("Enhanced summary cell output preview:")
	t.Log(source[:800] + "...")
}

// TestEnhancedDetailedStatsCell tests the enhanced detailed statistics
func TestEnhancedDetailedStatsCell(t *testing.T) {
	result := &ScanResult{
		Summary: Summary{
			TotalActions: 20,
			UniqueActions: map[string]ActionUsageStat{
				"actions/checkout": {
					UsageCount: 8,
					Versions: map[string]int{
						"v3": 5,
						"v4": 3,
					},
					Repositories: []string{"repo1", "repo2"},
				},
				"actions/setup-node": {
					UsageCount: 5,
					Versions: map[string]int{
						"v2": 3,
						"v4": 2,
					},
					Repositories: []string{"repo1"},
				},
				"third-party/action": {
					UsageCount: 2,
					Versions: map[string]int{
						"v1": 2,
					},
					Repositories: []string{"repo3"},
				},
			},
		},
	}

	cell := createDetailedStatsCell(result)

	source := strings.Join(cell.Source, "")

	expectedElements := []string{
		"📊 Detailed Action Statistics & Analytics",
		"📈 Action Usage Visualization",
		"🏆 Top Action Usage (Visual Chart)",
		"🥇", // First place medal
		"🔥 Very Popular", // High usage indicator
		"🔢 Version Distribution Analysis",
		"Visual Distribution:",
		"✅", // Current version indicator
		"⚠️", // Outdated version indicator
		"🔴", // Legacy version indicator
		"🌈 Action Diversity Analysis",
		"🏢 Ecosystem Distribution",
		"GitHub Official",
		"Third-party",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(source, expected) {
			t.Errorf("Expected element '%s' not found in output", expected)
		}
	}

	t.Log("Enhanced detailed stats cell output preview:")
	t.Log(source[:800] + "...")
}

// TestHelperFunctions tests the helper functions
func TestHelperFunctions(t *testing.T) {
	// Test extractMajorVersionFromString
	tests := []struct {
		input    string
		expected string
	}{
		{"v4.1.0", "4"},
		{"v3", "3"},
		{"4.1.0", "4"},
		{"main", "main"},
		{"", ""},
	}

	for _, test := range tests {
		result := extractMajorVersionFromString(test.input)
		if result != test.expected {
			t.Errorf("extractMajorVersionFromString(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}

	// Test sanitizeNodeName
	sanitizeTests := []struct {
		input    string
		expected string
	}{
		{"v4.1.0", "Vv4_1_0"},
		{"v3", "Vv3"},
		{"main-branch", "Vmain_branch"},
	}

	for _, test := range sanitizeTests {
		result := sanitizeNodeName(test.input)
		if result != test.expected {
			t.Errorf("sanitizeNodeName(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}