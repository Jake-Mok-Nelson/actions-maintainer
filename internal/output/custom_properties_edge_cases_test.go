package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
)

// TestCustomPropertiesEdgeCases tests edge cases for custom properties display
func TestCustomPropertiesEdgeCases(t *testing.T) {
	t.Run("No repositories with custom properties", func(t *testing.T) {
		// Create test data without custom properties
		repositories := []RepositoryResult{
			{
				Name:          "test-repo-1",
				FullName:      "testorg/test-repo-1",
				DefaultBranch: "main",
				WorkflowFiles: []WorkflowFileResult{
					{
						Path:        ".github/workflows/ci.yml",
						ActionCount: 1,
						Actions: []workflow.ActionReference{
							{Repository: "actions/checkout", Version: "v3"},
						},
					},
				},
				Actions: []workflow.ActionReference{
					{Repository: "actions/checkout", Version: "v3"},
				},
				Issues:           []ActionIssue{},
				CustomProperties: nil, // No custom properties
			},
			{
				Name:          "test-repo-2",
				FullName:      "testorg/test-repo-2",
				DefaultBranch: "main",
				WorkflowFiles: []WorkflowFileResult{},
				Actions:       []workflow.ActionReference{},
				Issues:        []ActionIssue{},
				CustomProperties: map[string]string{}, // Empty custom properties
			},
		}

		scanResult := BuildScanResult("testorg", repositories)
		FinalizeScanResult(scanResult)

		var notebookBuf bytes.Buffer
		err := FormatNotebook(scanResult, &notebookBuf)
		if err != nil {
			t.Fatal("Notebook format error:", err)
		}

		notebookStr := notebookBuf.String()
		
		// Should NOT have custom property filters when no properties exist
		if strings.Contains(notebookStr, "Custom Property Filters") {
			t.Error("❌ Custom property filtering interface should NOT appear when no custom properties exist")
		} else {
			t.Log("✅ No custom property filtering interface when no custom properties exist")
		}

		// Should have standard table headers without custom properties
		if strings.Contains(notebookStr, "| Repository | Workflows | Actions | Issues |\\n") {
			t.Log("✅ Standard table header without custom properties")
		} else {
			t.Error("❌ Expected standard table header without custom properties")
		}
	})

	t.Run("Some repositories with custom properties, some without", func(t *testing.T) {
		// Mix of repositories with and without custom properties
		repositories := []RepositoryResult{
			{
				Name:          "test-repo-1",
				FullName:      "testorg/test-repo-1",
				DefaultBranch: "main",
				WorkflowFiles: []WorkflowFileResult{},
				Actions:       []workflow.ActionReference{},
				Issues:        []ActionIssue{},
				CustomProperties: map[string]string{
					"ProductId": "product-123",
					"Team":      "backend",
				},
			},
			{
				Name:          "test-repo-2",
				FullName:      "testorg/test-repo-2",
				DefaultBranch: "main",
				WorkflowFiles: []WorkflowFileResult{},
				Actions:       []workflow.ActionReference{},
				Issues:        []ActionIssue{},
				CustomProperties: nil, // No custom properties
			},
			{
				Name:          "test-repo-3",
				FullName:      "testorg/test-repo-3",
				DefaultBranch: "main",
				WorkflowFiles: []WorkflowFileResult{},
				Actions:       []workflow.ActionReference{},
				Issues:        []ActionIssue{},
				CustomProperties: map[string]string{
					"ProductId": "product-456",
					// Missing Team property
				},
			},
		}

		scanResult := BuildScanResult("testorg", repositories)
		FinalizeScanResult(scanResult)

		var notebookBuf bytes.Buffer
		err := FormatNotebook(scanResult, &notebookBuf)
		if err != nil {
			t.Fatal("Notebook format error:", err)
		}

		notebookStr := notebookBuf.String()
		
		// Should have custom property filters since some repos have properties
		if strings.Contains(notebookStr, "Custom Property Filters") {
			t.Log("✅ Custom property filtering interface appears when some repositories have custom properties")
		} else {
			t.Error("❌ Custom property filtering interface missing when some repositories have custom properties")
		}

		// Should include custom property columns
		if strings.Contains(notebookStr, "| Repository | Workflows | Actions | Issues | ProductId | Team |") {
			t.Log("✅ Table header includes custom property columns")
		} else {
			t.Error("❌ Table header missing custom property columns")
		}

		// Check that missing properties show as "-"
		var notebook struct {
			Cells []struct {
				CellType string   `json:"cell_type"`
				Source   []string `json:"source"`
			} `json:"cells"`
		}
		if err := json.Unmarshal(notebookBuf.Bytes(), &notebook); err == nil {
			for _, cell := range notebook.Cells {
				sourceStr := strings.Join(cell.Source, "")
				if strings.Contains(sourceStr, "test-repo-2") {
					if strings.Contains(sourceStr, "| - | - |") {
						t.Log("✅ Missing custom properties show as '-' for repositories without properties")
					} else {
						t.Error("❌ Missing custom properties not handled correctly for repositories without properties")
						t.Logf("Cell content: %s", sourceStr)
					}
					break
				}
			}
		}
	})

	t.Run("Empty repository list", func(t *testing.T) {
		repositories := []RepositoryResult{}
		scanResult := BuildScanResult("testorg", repositories)
		FinalizeScanResult(scanResult)

		var notebookBuf bytes.Buffer
		err := FormatNotebook(scanResult, &notebookBuf)
		if err != nil {
			t.Fatal("Notebook format error:", err)
		}

		notebookStr := notebookBuf.String()
		
		// Should handle empty repository list gracefully
		if strings.Contains(notebookStr, "No repositories found") {
			t.Log("✅ Empty repository list handled gracefully")
		} else {
			t.Error("❌ Empty repository list not handled properly")
		}
	})
}