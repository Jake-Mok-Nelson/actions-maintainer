package output

import (
	"fmt"
	"testing"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
)

// TestSelectTopIssues_GroupsByWorkflowFile tests that issues are grouped by workflow file
func TestSelectTopIssues_GroupsByWorkflowFile(t *testing.T) {
	issues := []ActionIssue{
		{
			Repository:     "actions/checkout",
			CurrentVersion: "v3",
			IssueType:      "outdated",
			Severity:       "medium",
			Description:    "Update to v4",
			FilePath:       ".github/workflows/ci.yml",
		},
		{
			Repository:     "actions/setup-node",
			CurrentVersion: "v2",
			IssueType:      "outdated",
			Severity:       "high",
			Description:    "Update to v4",
			FilePath:       ".github/workflows/ci.yml",
		},
		{
			Repository:     "actions/cache",
			CurrentVersion: "v2",
			IssueType:      "deprecated",
			Severity:       "low",
			Description:    "Update to v4",
			FilePath:       ".github/workflows/deploy.yml",
		},
	}

	topIssues := selectTopIssues(issues, 10)

	// Should have 2 groups (one per workflow file)
	if len(topIssues) != 2 {
		t.Errorf("Expected 2 grouped issues, got %d", len(topIssues))
	}

	// First issue should be ci.yml (2 issues) with highest severity
	if topIssues[0].FilePath != ".github/workflows/ci.yml" {
		t.Errorf("Expected first issue to be ci.yml, got %s", topIssues[0].FilePath)
	}
	if topIssues[0].Context != "2 issues found" {
		t.Errorf("Expected context '2 issues found', got %s", topIssues[0].Context)
	}
	if topIssues[0].Severity != "high" {
		t.Errorf("Expected severity 'high' (highest among grouped issues), got %s", topIssues[0].Severity)
	}

	// Second issue should be deploy.yml (1 issue)
	if topIssues[1].FilePath != ".github/workflows/deploy.yml" {
		t.Errorf("Expected second issue to be deploy.yml, got %s", topIssues[1].FilePath)
	}
	if topIssues[1].Context != "1 issues found" {
		t.Errorf("Expected context '1 issues found', got %s", topIssues[1].Context)
	}
}

// TestSelectTopIssues_OrdersByOccurrenceCount tests that results are ordered by occurrence count
func TestSelectTopIssues_OrdersByOccurrenceCount(t *testing.T) {
	issues := []ActionIssue{
		// File A: 1 issue
		{
			Repository:  "actions/checkout",
			IssueType:   "outdated",
			Severity:    "critical", // High severity but fewer issues
			Description: "Critical issue",
			FilePath:    ".github/workflows/fileA.yml",
		},
		// File B: 3 issues
		{
			Repository:  "actions/setup-node",
			IssueType:   "outdated",
			Severity:    "low",
			Description: "Low severity issue 1",
			FilePath:    ".github/workflows/fileB.yml",
		},
		{
			Repository:  "actions/cache",
			IssueType:   "deprecated",
			Severity:    "low",
			Description: "Low severity issue 2",
			FilePath:    ".github/workflows/fileB.yml",
		},
		{
			Repository:  "actions/upload-artifact",
			IssueType:   "migration",
			Severity:    "medium",
			Description: "Medium severity issue",
			FilePath:    ".github/workflows/fileB.yml",
		},
	}

	topIssues := selectTopIssues(issues, 10)

	// Should have 2 groups
	if len(topIssues) != 2 {
		t.Errorf("Expected 2 grouped issues, got %d", len(topIssues))
	}

	// File B should be first (3 issues) despite lower individual severity
	if topIssues[0].FilePath != ".github/workflows/fileB.yml" {
		t.Errorf("Expected first issue to be fileB.yml (most issues), got %s", topIssues[0].FilePath)
	}
	if topIssues[0].Context != "3 issues found" {
		t.Errorf("Expected context '3 issues found', got %s", topIssues[0].Context)
	}

	// File A should be second (1 issue)
	if topIssues[1].FilePath != ".github/workflows/fileA.yml" {
		t.Errorf("Expected second issue to be fileA.yml, got %s", topIssues[1].FilePath)
	}
	if topIssues[1].Context != "1 issues found" {
		t.Errorf("Expected context '1 issues found', got %s", topIssues[1].Context)
	}
}

// TestSelectTopIssues_LimitResults tests that results are limited correctly
func TestSelectTopIssues_LimitResults(t *testing.T) {
	issues := []ActionIssue{}

	// Create issues for 5 different workflow files
	for i := 0; i < 5; i++ {
		for j := 0; j < i+1; j++ {
			issues = append(issues, ActionIssue{
				Repository:  "actions/checkout",
				IssueType:   "outdated",
				Severity:    "medium",
				Description: "Test issue",
				FilePath:    fmt.Sprintf(".github/workflows/file%d.yml", i),
			})
		}
	}

	// Request only top 3
	topIssues := selectTopIssues(issues, 3)

	if len(topIssues) != 3 {
		t.Errorf("Expected 3 grouped issues (limit), got %d", len(topIssues))
	}

	// Should be ordered by count: file4 (5), file3 (4), file2 (3)
	expectedFiles := []string{
		".github/workflows/file4.yml",
		".github/workflows/file3.yml",
		".github/workflows/file2.yml",
	}

	for i, expected := range expectedFiles {
		if topIssues[i].FilePath != expected {
			t.Errorf("Expected issue %d to be %s, got %s", i, expected, topIssues[i].FilePath)
		}
	}
}

// TestSelectTopIssues_EmptyInput tests empty input handling
func TestSelectTopIssues_EmptyInput(t *testing.T) {
	issues := []ActionIssue{}
	topIssues := selectTopIssues(issues, 10)

	if len(topIssues) != 0 {
		t.Errorf("Expected 0 issues for empty input, got %d", len(topIssues))
	}
}

// TestSelectTopIssues_ManualValidation manually tests the new functionality with sample data
func TestSelectTopIssues_ManualValidation(t *testing.T) {
	// Create mock issues to validate the new format
	issues := []ActionIssue{
		{
			Repository:       "actions/checkout",
			CurrentVersion:   "v3",
			SuggestedVersion: "v4",
			IssueType:        "outdated",
			Severity:         "medium",
			Description:      "actions/checkout v3 is outdated, upgrade to v4 for better performance",
			FilePath:         ".github/workflows/ci.yml",
		},
		{
			Repository:       "actions/setup-node",
			CurrentVersion:   "v2",
			SuggestedVersion: "v4",
			IssueType:        "outdated",
			Severity:         "high",
			Description:      "actions/setup-node v2 has security vulnerabilities, upgrade to v4",
			FilePath:         ".github/workflows/ci.yml",
		},
		{
			Repository:       "actions/cache",
			CurrentVersion:   "v2",
			SuggestedVersion: "v4",
			IssueType:        "deprecated",
			Severity:         "low",
			Description:      "actions/cache v2 is deprecated, migrate to v4",
			FilePath:         ".github/workflows/deploy.yml",
		},
		{
			Repository:       "actions/upload-artifact",
			CurrentVersion:   "v3",
			SuggestedVersion: "v4",
			IssueType:        "migration",
			Severity:         "medium",
			Description:      "actions/upload-artifact v3 will be deprecated, migrate to v4",
			FilePath:         ".github/workflows/release.yml",
		},
		{
			Repository:       "actions/download-artifact",
			CurrentVersion:   "v3",
			SuggestedVersion: "v4",
			IssueType:        "migration",
			Severity:         "medium",
			Description:      "actions/download-artifact v3 will be deprecated, migrate to v4",
			FilePath:         ".github/workflows/release.yml",
		},
	}

	topIssues := selectTopIssues(issues, 10)

	// Log results in the expected format
	t.Log("=== Top Issues Output ===")
	for i, issue := range topIssues {
		t.Logf("%d. %s", i+1, issue.FilePath)
		t.Logf("Finding: %s", issue.IssueType)
		t.Logf("Description: %s", issue.Description)
		if issue.Context != "" {
			t.Logf("Issues Found: %s", issue.Context)
		}
		t.Log("")
	}

	// Validate results:
	// 1. ci.yml should be first (2 issues)
	// 2. release.yml should be second (2 issues, but lower severity)
	// 3. deploy.yml should be third (1 issue)

	if len(topIssues) != 3 {
		t.Errorf("Expected 3 grouped workflow files, got %d", len(topIssues))
	}

	// First should be ci.yml with 2 issues and high severity (highest individual severity wins)
	if topIssues[0].FilePath != ".github/workflows/ci.yml" {
		t.Errorf("Expected first workflow to be ci.yml, got %s", topIssues[0].FilePath)
	}
	if topIssues[0].Context != "2 issues found" {
		t.Errorf("Expected '2 issues found', got %s", topIssues[0].Context)
	}
	if topIssues[0].Severity != "high" {
		t.Errorf("Expected severity 'high', got %s", topIssues[0].Severity)
	}

	// Second should be release.yml with 2 issues but medium severity
	if topIssues[1].FilePath != ".github/workflows/release.yml" {
		t.Errorf("Expected second workflow to be release.yml, got %s", topIssues[1].FilePath)
	}
	if topIssues[1].Context != "2 issues found" {
		t.Errorf("Expected '2 issues found', got %s", topIssues[1].Context)
	}

	// Third should be deploy.yml with 1 issue
	if topIssues[2].FilePath != ".github/workflows/deploy.yml" {
		t.Errorf("Expected third workflow to be deploy.yml, got %s", topIssues[2].FilePath)
	}
	if topIssues[2].Context != "1 issues found" {
		t.Errorf("Expected '1 issues found', got %s", topIssues[2].Context)
	}
}

// TestIsHigherSeverity tests the severity comparison function
func TestIsHigherSeverity(t *testing.T) {
	testCases := []struct {
		severity1 string
		severity2 string
		expected  bool
	}{
		{"critical", "high", true},
		{"high", "medium", true},
		{"medium", "low", true},
		{"low", "critical", false},
		{"medium", "high", false},
		{"critical", "critical", false},
	}

	for _, tc := range testCases {
		result := isHigherSeverity(tc.severity1, tc.severity2)
		if result != tc.expected {
			t.Errorf("isHigherSeverity(%s, %s) = %v, expected %v",
				tc.severity1, tc.severity2, result, tc.expected)
		}
	}
}

// TestCalculateSummary_ActionsAndWorkflowsBreakdown tests the enhanced summary with actions/workflows breakdown
func TestCalculateSummary_ActionsAndWorkflowsBreakdown(t *testing.T) {
	// Mock repository results with both actions and workflows
	repositories := []RepositoryResult{
		{
			Name:     "repo1",
			FullName: "org/repo1",
			WorkflowFiles: []WorkflowFileResult{
				{
					Path:        ".github/workflows/ci.yml",
					ActionCount: 3,
					Actions: []workflow.ActionReference{
						{Repository: "actions/checkout", Version: "v3", IsReusable: false},
						{Repository: "actions/setup-node", Version: "v2", IsReusable: false},
						{Repository: "org/shared-workflows", Version: "v1", IsReusable: true, WorkflowPath: ".github/workflows/test.yml"},
					},
				},
			},
			Actions: []workflow.ActionReference{
				{Repository: "actions/checkout", Version: "v3", IsReusable: false},
				{Repository: "actions/setup-node", Version: "v2", IsReusable: false},
				{Repository: "org/shared-workflows", Version: "v1", IsReusable: true, WorkflowPath: ".github/workflows/test.yml"},
			},
			Issues: []ActionIssue{
				{Repository: "actions/checkout", IssueType: "outdated", Severity: "low"},
				{Repository: "actions/setup-node", IssueType: "outdated", Severity: "high"},
			},
		},
		{
			Name:     "repo2",
			FullName: "org/repo2",
			WorkflowFiles: []WorkflowFileResult{
				{
					Path:        ".github/workflows/deploy.yml",
					ActionCount: 2,
					Actions: []workflow.ActionReference{
						{Repository: "actions/checkout", Version: "v4", IsReusable: false},
						{Repository: "org/deploy-workflows", Version: "v2", IsReusable: true, WorkflowPath: ".github/workflows/deploy.yml"},
					},
				},
			},
			Actions: []workflow.ActionReference{
				{Repository: "actions/checkout", Version: "v4", IsReusable: false},
				{Repository: "org/deploy-workflows", Version: "v2", IsReusable: true, WorkflowPath: ".github/workflows/deploy.yml"},
			},
			Issues: []ActionIssue{},
		},
	}

	summary := calculateSummary(repositories)

	// Test basic counts
	if summary.TotalRepositories != 2 {
		t.Errorf("Expected 2 repositories, got %d", summary.TotalRepositories)
	}
	if summary.TotalWorkflowFiles != 2 {
		t.Errorf("Expected 2 workflow files, got %d", summary.TotalWorkflowFiles)
	}
	if summary.TotalActions != 5 {
		t.Errorf("Expected 5 total actions, got %d", summary.TotalActions)
	}
	if summary.TotalRegularActions != 3 {
		t.Errorf("Expected 3 regular actions, got %d", summary.TotalRegularActions)
	}
	if summary.TotalReusableWorkflows != 2 {
		t.Errorf("Expected 2 reusable workflows, got %d", summary.TotalReusableWorkflows)
	}

	// Test unique action counts
	if len(summary.UniqueActions) != 4 {
		t.Errorf("Expected 4 unique actions total, got %d", len(summary.UniqueActions))
	}
	if len(summary.UniqueRegularActions) != 2 {
		t.Errorf("Expected 2 unique regular actions, got %d", len(summary.UniqueRegularActions))
	}
	if len(summary.UniqueReusableWorkflows) != 2 {
		t.Errorf("Expected 2 unique reusable workflows, got %d", len(summary.UniqueReusableWorkflows))
	}

	// Test that actions are properly categorized
	checkoutStat, exists := summary.UniqueRegularActions["actions/checkout"]
	if !exists {
		t.Error("actions/checkout should be in UniqueRegularActions")
	} else if checkoutStat.IsReusableWorkflow {
		t.Error("actions/checkout should not be marked as reusable workflow")
	}

	sharedWorkflowStat, exists := summary.UniqueReusableWorkflows["org/shared-workflows"]
	if !exists {
		t.Error("org/shared-workflows should be in UniqueReusableWorkflows")
	} else if !sharedWorkflowStat.IsReusableWorkflow {
		t.Error("org/shared-workflows should be marked as reusable workflow")
	}

	// Test checkout usage across repositories (should appear in both)
	if checkoutStat.UsageCount != 2 {
		t.Errorf("actions/checkout should have usage count 2, got %d", checkoutStat.UsageCount)
	}
	if len(checkoutStat.Repositories) != 2 {
		t.Errorf("actions/checkout should be used in 2 repositories, got %d", len(checkoutStat.Repositories))
	}
	if len(checkoutStat.Versions) != 2 {
		t.Errorf("actions/checkout should have 2 different versions, got %d", len(checkoutStat.Versions))
	}

	// Test issue counts
	if summary.IssuesByType["outdated"] != 2 {
		t.Errorf("Expected 2 outdated issues, got %d", summary.IssuesByType["outdated"])
	}
	if summary.IssuesBySeverity["low"] != 1 {
		t.Errorf("Expected 1 low severity issue, got %d", summary.IssuesBySeverity["low"])
	}
	if summary.IssuesBySeverity["high"] != 1 {
		t.Errorf("Expected 1 high severity issue, got %d", summary.IssuesBySeverity["high"])
	}

	t.Log("=== Enhanced Summary Statistics ===")
	t.Logf("Total Actions: %d", summary.TotalActions)
	t.Logf("  - Regular Actions: %d", summary.TotalRegularActions)
	t.Logf("  - Reusable Workflows: %d", summary.TotalReusableWorkflows)
	t.Logf("Unique Actions: %d", len(summary.UniqueActions))
	t.Logf("  - Unique Regular Actions: %d", len(summary.UniqueRegularActions))
	t.Logf("  - Unique Reusable Workflows: %d", len(summary.UniqueReusableWorkflows))

	t.Log("\n=== Regular Actions ===")
	for name, stat := range summary.UniqueRegularActions {
		t.Logf("%s: %d uses, %d versions, %d repos", name, stat.UsageCount, len(stat.Versions), len(stat.Repositories))
	}

	t.Log("\n=== Reusable Workflows ===")
	for name, stat := range summary.UniqueReusableWorkflows {
		t.Logf("%s: %d uses, %d versions, %d repos", name, stat.UsageCount, len(stat.Versions), len(stat.Repositories))
	}
}
