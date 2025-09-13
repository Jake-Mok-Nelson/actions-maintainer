package output

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
)

// ScanResult represents the complete result of a repository scan
type ScanResult struct {
	Owner        string             `json:"owner"`
	ScanTime     time.Time          `json:"scan_time"`
	Repositories []RepositoryResult `json:"repositories"`
	Summary      Summary            `json:"summary"`
}

// RepositoryResult represents the scan result for a single repository
type RepositoryResult struct {
	Name          string                     `json:"name"`
	FullName      string                     `json:"full_name"`
	DefaultBranch string                     `json:"default_branch"`
	WorkflowFiles []WorkflowFileResult       `json:"workflow_files"`
	Actions       []workflow.ActionReference `json:"actions"`
	Issues        []ActionIssue              `json:"issues,omitempty"`
}

// WorkflowFileResult represents a workflow file scan result
type WorkflowFileResult struct {
	Path        string                     `json:"path"`
	ActionCount int                        `json:"action_count"`
	Actions     []workflow.ActionReference `json:"actions"`
}

// ActionIssue represents an issue with an action (outdated version, deprecated, etc.)
type ActionIssue struct {
	Repository       string `json:"repository"`
	CurrentVersion   string `json:"current_version"`
	SuggestedVersion string `json:"suggested_version,omitempty"`
	IssueType        string `json:"issue_type"` // "outdated", "deprecated"
	Severity         string `json:"severity"`   // "low", "medium", "high", "critical"
	Description      string `json:"description"`
	Context          string `json:"context"` // where the issue was found
	FilePath         string `json:"file_path"`
}

// Summary provides aggregate statistics about the scan
type Summary struct {
	TotalRepositories  int                        `json:"total_repositories"`
	TotalWorkflowFiles int                        `json:"total_workflow_files"`
	TotalActions       int                        `json:"total_actions"`
	UniqueActions      map[string]ActionUsageStat `json:"unique_actions"`
	IssuesByType       map[string]int             `json:"issues_by_type"`
	IssuesBySeverity   map[string]int             `json:"issues_by_severity"`
	TopIssues          []ActionIssue              `json:"top_issues"`
}

// ActionUsageStat represents usage statistics for a specific action
type ActionUsageStat struct {
	Repository   string         `json:"repository"`
	UsageCount   int            `json:"usage_count"`
	Versions     map[string]int `json:"versions"`
	Repositories []string       `json:"repositories"`
}

// FormatJSON outputs the scan results as JSON
func FormatJSON(result *ScanResult, writer io.Writer, pretty bool) error {
	var data []byte
	var err error

	if pretty {
		data, err = json.MarshalIndent(result, "", "  ")
	} else {
		data, err = json.Marshal(result)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	_, err = writer.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	return nil
}

// BuildScanResult constructs a complete scan result from repository data
func BuildScanResult(owner string, repositories []RepositoryResult) *ScanResult {
	scanTime := time.Now()

	// Calculate summary statistics
	summary := calculateSummary(repositories)

	return &ScanResult{
		Owner:        owner,
		ScanTime:     scanTime,
		Repositories: repositories,
		Summary:      summary,
	}
}

// calculateSummary generates summary statistics from repository results
func calculateSummary(repositories []RepositoryResult) Summary {
	summary := Summary{
		UniqueActions:    make(map[string]ActionUsageStat),
		IssuesByType:     make(map[string]int),
		IssuesBySeverity: make(map[string]int),
	}

	totalWorkflowFiles := 0
	totalActions := 0
	var allIssues []ActionIssue

	// Process each repository
	for _, repo := range repositories {
		summary.TotalRepositories++
		totalWorkflowFiles += len(repo.WorkflowFiles)

		// Process actions in this repository
		for _, action := range repo.Actions {
			totalActions++

			// Update unique actions statistics
			stat, exists := summary.UniqueActions[action.Repository]
			if !exists {
				stat = ActionUsageStat{
					Repository:   action.Repository,
					Versions:     make(map[string]int),
					Repositories: make([]string, 0),
				}
			}

			stat.UsageCount++
			stat.Versions[action.Version]++

			// Add repository to list if not already present
			found := false
			for _, repoName := range stat.Repositories {
				if repoName == repo.FullName {
					found = true
					break
				}
			}
			if !found {
				stat.Repositories = append(stat.Repositories, repo.FullName)
			}

			summary.UniqueActions[action.Repository] = stat
		}

		// Process issues
		for _, issue := range repo.Issues {
			allIssues = append(allIssues, issue)
			summary.IssuesByType[issue.IssueType]++
			summary.IssuesBySeverity[issue.Severity]++
		}
	}

	summary.TotalWorkflowFiles = totalWorkflowFiles
	summary.TotalActions = totalActions

	// Select top issues (limit to 10)
	summary.TopIssues = selectTopIssues(allIssues, 10)

	return summary
}

// selectTopIssues selects the most important issues based on severity
func selectTopIssues(issues []ActionIssue, limit int) []ActionIssue {
	if len(issues) <= limit {
		return issues
	}

	// Simple selection - in practice you might want more sophisticated sorting
	topIssues := make([]ActionIssue, 0, limit)

	// First, add critical issues
	for _, issue := range issues {
		if issue.Severity == "critical" && len(topIssues) < limit {
			topIssues = append(topIssues, issue)
		}
	}

	// Then add high severity issues
	for _, issue := range issues {
		if issue.Severity == "high" && len(topIssues) < limit {
			topIssues = append(topIssues, issue)
		}
	}

	// Fill remaining slots with medium and low severity
	for _, issue := range issues {
		if (issue.Severity == "medium" || issue.Severity == "low") && len(topIssues) < limit {
			topIssues = append(topIssues, issue)
		}
	}

	return topIssues
}
