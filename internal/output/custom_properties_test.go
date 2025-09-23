package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
)

// TestCustomPropertiesDisplay tests that custom properties are properly displayed in both JSON and notebook outputs
func TestCustomPropertiesDisplay(t *testing.T) {
	// Create test data with custom properties
	repositories := []RepositoryResult{
		{
			Name:          "test-repo-1",
			FullName:      "testorg/test-repo-1",
			DefaultBranch: "main",
			WorkflowFiles: []WorkflowFileResult{
				{
					Path:        ".github/workflows/ci.yml",
					ActionCount: 2,
					Actions: []workflow.ActionReference{
						{Repository: "actions/checkout", Version: "v3"},
						{Repository: "actions/setup-node", Version: "v2"},
					},
				},
			},
			Actions: []workflow.ActionReference{
				{Repository: "actions/checkout", Version: "v3"},
				{Repository: "actions/setup-node", Version: "v2"},
			},
			Issues: []ActionIssue{
				{
					Repository:       "actions/checkout",
					CurrentVersion:   "v3",
					SuggestedVersion: "v4",
					IssueType:        "outdated",
					Severity:         "medium",
					Description:      "Update to v4",
					FilePath:         ".github/workflows/ci.yml",
				},
			},
			CustomProperties: map[string]string{
				"ProductId":   "product-123",
				"Team":        "backend",
				"Environment": "production",
			},
		},
		{
			Name:          "test-repo-2",
			FullName:      "testorg/test-repo-2",
			DefaultBranch: "main",
			WorkflowFiles: []WorkflowFileResult{
				{
					Path:        ".github/workflows/deploy.yml",
					ActionCount: 1,
					Actions: []workflow.ActionReference{
						{Repository: "actions/cache", Version: "v2"},
					},
				},
			},
			Actions: []workflow.ActionReference{
				{Repository: "actions/cache", Version: "v2"},
			},
			Issues: []ActionIssue{},
			CustomProperties: map[string]string{
				"ProductId":   "product-456",
				"Team":        "frontend",
				"Environment": "development",
			},
		},
		{
			Name:          "test-repo-3",
			FullName:      "testorg/test-repo-3",
			DefaultBranch: "main",
			WorkflowFiles: []WorkflowFileResult{},
			Actions:       []workflow.ActionReference{},
			Issues:        []ActionIssue{},
			CustomProperties: map[string]string{
				"ProductId": "product-789",
				"Team":      "devops",
				// No Environment for this repo
			},
		},
	}

	// Build scan result
	scanResult := BuildScanResult("testorg", repositories)
	scanResult.ScanTime = time.Now()
	FinalizeScanResult(scanResult)

	// Test JSON output
	t.Log("=== JSON OUTPUT TEST ===")
	var jsonBuf bytes.Buffer
	err := FormatJSON(scanResult, &jsonBuf, true)
	if err != nil {
		t.Fatal("JSON format error:", err)
	}

	// Check if custom properties are in JSON
	jsonStr := jsonBuf.String()
	if strings.Contains(jsonStr, "custom_properties") {
		t.Log("✅ Custom properties found in JSON output")
		
		// Parse and show specific custom properties
		var result ScanResult
		json.Unmarshal(jsonBuf.Bytes(), &result)
		for _, repo := range result.Repositories {
			t.Logf("Repository %s custom properties: %v", repo.Name, repo.CustomProperties)
		}
	} else {
		t.Error("❌ Custom properties NOT found in JSON output")
	}

	// Test Notebook output
	t.Log("=== NOTEBOOK OUTPUT TEST ===")
	var notebookBuf bytes.Buffer
	err = FormatNotebook(scanResult, &notebookBuf)
	if err != nil {
		t.Fatal("Notebook format error:", err)
	}

	notebookStr := notebookBuf.String()
	if strings.Contains(notebookStr, "ProductId") {
		t.Log("✅ Custom properties found in notebook output")
	} else {
		t.Error("❌ Custom properties NOT found in notebook output")
	}

	// Check for custom property filtering interface
	if strings.Contains(notebookStr, "Custom Property Filters") {
		t.Log("✅ Custom property filtering interface found")
	} else {
		t.Error("❌ Custom property filtering interface NOT found")
	}

	// Check for custom property columns in table
	if strings.Contains(notebookStr, "ProductId") && strings.Contains(notebookStr, "Team") && strings.Contains(notebookStr, "Environment") {
		t.Log("✅ Custom property columns found in table")
	} else {
		t.Error("❌ Custom property columns NOT found in table")
	}

	// Check table structure more specifically
	if strings.Contains(notebookStr, "| Repository | Workflows | Actions | Issues | Environment | ProductId | Team |") {
		t.Log("✅ Table header includes custom property columns in alphabetical order")
	} else {
		t.Error("❌ Table header does not include custom property columns properly")
		// Show what was actually generated
		lines := strings.Split(notebookStr, "\\n")
		for i, line := range lines {
			if strings.Contains(line, "| Repository |") {
				t.Logf("Found table header at line %d: %s", i, line)
				break
			}
		}
	}

	// Show detailed notebook structure for debugging
	t.Log("=== NOTEBOOK STRUCTURE ===")
	var notebook struct {
		Cells []struct {
			CellType string   `json:"cell_type"`
			Source   []string `json:"source"`
		} `json:"cells"`
	}
	json.Unmarshal(notebookBuf.Bytes(), &notebook)
	
	for i, cell := range notebook.Cells {
		if cell.CellType == "markdown" {
			sourceStr := strings.Join(cell.Source, "")
			if strings.Contains(sourceStr, "Repository Details") {
				t.Logf("Cell %d (Repository Details):", i)
				// Show the entire table section
				for j, line := range cell.Source {
					if strings.Contains(line, "| Repository |") || strings.Contains(line, "|------------|") || strings.Contains(line, "| [`test-repo") {
						t.Logf("  Table Line %d: %s", j, strings.TrimSuffix(line, "\\n"))
					}
				}
				break
			}
		}
	}
}