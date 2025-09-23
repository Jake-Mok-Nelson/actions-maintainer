package output

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
)

// TestCreateIssuesOverviewCell_WithGroupedIssues tests that the notebook output handles grouped issues correctly
func TestCreateIssuesOverviewCell_WithGroupedIssues(t *testing.T) {
	// Create a scan result with grouped top issues
	scanResult := &ScanResult{
		Owner:    "testowner",
		ScanTime: time.Now(),
		Summary: Summary{
			TopIssues: []ActionIssue{
				{
					Repository:  "", // Empty for grouped issues
					IssueType:   "outdated",
					Severity:    "high",
					Description: "Multiple actions need updates",
					Context:     "2 issues found",
					FilePath:    ".github/workflows/ci.yml",
				},
				{
					Repository:  "", // Empty for grouped issues
					IssueType:   "deprecated",
					Severity:    "medium",
					Description: "Legacy action usage detected",
					Context:     "1 issues found",
					FilePath:    ".github/workflows/deploy.yml",
				},
			},
		},
	}

	cell := createIssuesOverviewCell(scanResult)

	// Verify the cell is markdown
	if cell.CellType != "markdown" {
		t.Errorf("Expected cell type 'markdown', got %s", cell.CellType)
	}

	// Convert source to string for easier testing
	source := strings.Join(cell.Source, "")

	// Verify the output format matches expectations
	if !strings.Contains(source, "## 🚨 Top Issues Requiring Attention") {
		t.Error("Expected header not found in output")
	}

	if !strings.Contains(source, "1. 🟠 .github/workflows/ci.yml") {
		t.Error("Expected first workflow with high severity icon not found")
	}

	if !strings.Contains(source, "- **Finding:** outdated") {
		t.Error("Expected 'Finding:' format not found")
	}

	if !strings.Contains(source, "- **Description:** Multiple actions need updates") {
		t.Error("Expected description not found")
	}

	if !strings.Contains(source, "- **Issues Found:** 2 issues found") {
		t.Error("Expected issues count not found")
	}

	if !strings.Contains(source, "2. 🟡 .github/workflows/deploy.yml") {
		t.Error("Expected second workflow with medium severity icon not found")
	}

	t.Log("Notebook output preview:")
	t.Log(source)
}

// TestCreateCustomPropertyFilterInterface tests the JavaScript filter generation
func TestCreateCustomPropertyFilterInterface(t *testing.T) {
	propertyKeys := []string{"TeamName", "Environment", "Owner"}

	source := createCustomPropertyFilterInterface(propertyKeys)

	// Convert to string for easier testing
	content := strings.Join(source, "")

	// Check that filter interface is created
	if !strings.Contains(content, "### 🔍 Custom Property Filters") {
		t.Error("Expected filter header not found")
	}

	// Check that dropdowns are created for each property
	for _, key := range propertyKeys {
		expectedSelect := fmt.Sprintf(`<select id='filter_%s'`, key)
		if !strings.Contains(content, expectedSelect) {
			t.Errorf("Expected select element for %s not found", key)
		}

		expectedLabel := fmt.Sprintf(`<label for='filter_%s'`, key)
		if !strings.Contains(content, expectedLabel) {
			t.Errorf("Expected label for %s not found", key)
		}

		expectedClearButton := fmt.Sprintf(`onclick='clearFilter("%s")'`, key)
		if !strings.Contains(content, expectedClearButton) {
			t.Errorf("Expected clear button for %s not found", key)
		}
	}

	// Check that JavaScript functions are included
	expectedFunctions := []string{
		"function initializeFilters()",
		"function populateFilterDropdown(propertyKey)",
		"function filterRepositories()",
		"function clearFilter(propertyKey)",
		"function clearAllFilters()",
		"function updateFilterStatus()",
	}

	for _, fn := range expectedFunctions {
		if !strings.Contains(content, fn) {
			t.Errorf("Expected JavaScript function '%s' not found", fn)
		}
	}

	// Check that property-specific code is generated correctly
	if !strings.Contains(content, "populateFilterDropdown('TeamName')") {
		t.Error("Expected populateFilterDropdown call for TeamName not found")
	}

	if !strings.Contains(content, "filters['Environment'] = document.getElementById('filter_Environment').value") {
		t.Error("Expected filter assignment for Environment not found")
	}

	t.Log("Custom property filter interface generated successfully")
	t.Log("Generated content length:", len(content))
}

// TestCreateRepositoryDetailsCell_WithCustomProperties tests notebook generation with custom properties
func TestCreateRepositoryDetailsCell_WithCustomProperties(t *testing.T) {
	// Create test data with custom properties
	result := &ScanResult{
		Repositories: []RepositoryResult{
			{
				Name:     "repo1",
				FullName: "owner/repo1",
				CustomProperties: map[string]string{
					"TeamName":    "Backend",
					"Environment": "Production",
					"Owner":       "team-alpha",
				},
				WorkflowFiles: []WorkflowFileResult{
					{Path: ".github/workflows/ci.yml"},
				},
				Actions: []workflow.ActionReference{
					{Repository: "actions/checkout", Version: "v3"},
				},
				Issues: []ActionIssue{
					{
						Repository:       "actions/checkout",
						CurrentVersion:   "v3",
						SuggestedVersion: "v4",
						IssueType:        "outdated",
						Severity:         "medium",
					},
				},
			},
			{
				Name:     "repo2",
				FullName: "owner/repo2",
				CustomProperties: map[string]string{
					"TeamName":    "Frontend",
					"Environment": "Development",
					"Owner":       "team-beta",
				},
				WorkflowFiles: []WorkflowFileResult{
					{Path: ".github/workflows/test.yml"},
				},
				Actions: []workflow.ActionReference{
					{Repository: "actions/setup-node", Version: "v2"},
				},
				Issues: []ActionIssue{},
			},
		},
	}

	cell := createRepositoryDetailsCell(result)
	content := strings.Join(cell.Source, "")

	// Check that custom property filter interface is included
	if !strings.Contains(content, "### 🔍 Custom Property Filters") {
		t.Error("Expected custom property filter interface not found")
	}

	// Check that table includes custom property columns (properties are sorted alphabetically)
	if !strings.Contains(content, "| Repository | Workflows | Actions | Issues | Environment | Owner | TeamName |") {
		t.Error("Expected table header with custom properties not found")
		t.Log("Actual content:")
		t.Log(content)
	}

	// Check that repository rows include custom property values
	if !strings.Contains(content, "| [`repo1`](https://github.com/owner/repo1) | 1 | 1 | ⚠️ 1 | Production | team-alpha | Backend |") {
		t.Error("Expected repo1 row with custom properties not found")
	}

	if !strings.Contains(content, "| [`repo2`](https://github.com/owner/repo2) | 1 | 1 | 0 | Development | team-beta | Frontend |") {
		t.Error("Expected repo2 row with custom properties not found")
	}

	// Check that detailed repository sections include custom properties
	if !strings.Contains(content, "- Environment: Production") {
		t.Error("Expected custom property in detailed section not found")
	}

	t.Log("Repository details with custom properties generated successfully")
}

// TestCreateIssuesOverviewCell_EmptyIssues tests that empty issues are handled correctly
func TestCreateIssuesOverviewCell_EmptyIssues(t *testing.T) {
	scanResult := &ScanResult{
		Summary: Summary{
			TopIssues: []ActionIssue{},
		},
	}

	cell := createIssuesOverviewCell(scanResult)
	source := strings.Join(cell.Source, "")

	if !strings.Contains(source, "✅ **No critical issues found!**") {
		t.Error("Expected no issues message not found")
	}
}
