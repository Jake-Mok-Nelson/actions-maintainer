package output

import (
	"strings"
	"testing"
	"time"
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
					Repository:   "", // Empty for grouped issues
					IssueType:    "outdated",
					Severity:     "high",
					Description:  "Multiple actions need updates",
					Context:      "2 issues found",
					FilePath:     ".github/workflows/ci.yml",
				},
				{
					Repository:   "", // Empty for grouped issues
					IssueType:    "deprecated",
					Severity:     "medium",
					Description:  "Legacy action usage detected",
					Context:      "1 issues found",
					FilePath:     ".github/workflows/deploy.yml",
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