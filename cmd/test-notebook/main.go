package main

import (
	"fmt"
	"os"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/output"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
)

func main() {
	// Create sample scan result data for testing
	scanResult := createSampleScanResult()

	fmt.Println("Testing Jupyter notebook output...")
	file, err := os.Create("/tmp/test-notebook.ipynb")
	if err != nil {
		fmt.Printf("Error creating test file: %v\n", err)
		return
	}
	defer file.Close()

	if err := output.FormatNotebook(scanResult, file); err != nil {
		fmt.Printf("Notebook format error: %v\n", err)
		return
	}

	fmt.Println("Notebook saved to /tmp/test-notebook.ipynb")
	
	// Show the file size to verify it was created
	if info, err := os.Stat("/tmp/test-notebook.ipynb"); err == nil {
		fmt.Printf("File size: %d bytes\n", info.Size())
	}
}

func createSampleScanResult() *output.ScanResult {
	// Create sample action references
	actions := []workflow.ActionReference{
		{Repository: "actions/checkout", Version: "v3", Context: "uses: actions/checkout@v3"},
		{Repository: "actions/setup-node", Version: "v3", Context: "uses: actions/setup-node@v3"},
		{Repository: "actions/cache", Version: "v2", Context: "uses: actions/cache@v2"},
	}

	// Create sample workflow files
	workflowFiles := []output.WorkflowFileResult{
		{
			Path:        ".github/workflows/ci.yml",
			ActionCount: 3,
			Actions:     actions,
		},
	}

	// Create sample issues
	issues := []output.ActionIssue{
		{
			Repository:       "actions/cache",
			CurrentVersion:   "v2",
			SuggestedVersion: "v4",
			IssueType:        "outdated",
			Severity:         "medium",
			Description:      "actions/cache@v2 is outdated. Latest version is v4.",
			Context:          "uses: actions/cache@v2",
			FilePath:         ".github/workflows/ci.yml",
		},
		{
			Repository:       "actions/checkout",
			CurrentVersion:   "v3",
			SuggestedVersion: "v4",
			IssueType:        "outdated",
			Severity:         "low",
			Description:      "actions/checkout@v3 has a newer version available.",
			Context:          "uses: actions/checkout@v3",
			FilePath:         ".github/workflows/ci.yml",
		},
	}

	// Create sample repository result
	repo := output.RepositoryResult{
		Name:          "sample-repo",
		FullName:      "test-org/sample-repo",
		DefaultBranch: "main",
		WorkflowFiles: workflowFiles,
		Actions:       actions,
		Issues:        issues,
	}

	// Create sample PR
	samplePR := output.CreatedPR{
		Repository:  "test-org/sample-repo",
		URL:         "https://github.com/test-org/sample-repo/pull/42",
		Title:       "Update GitHub Actions to latest versions",
		Number:      42,
		UpdateCount: 2,
	}

	// Build the scan result
	scanResult := output.BuildScanResult("test-org", []output.RepositoryResult{repo})
	
	// Add the sample PR
	output.AddCreatedPR(scanResult, samplePR)
	
	// Finalize with timing
	output.FinalizeScanResult(scanResult)

	return scanResult
}