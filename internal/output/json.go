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
	ScanEndTime  time.Time          `json:"scan_end_time"`
	Duration     time.Duration      `json:"duration"`
	Repositories []RepositoryResult `json:"repositories"`
	Summary      Summary            `json:"summary"`
	CreatedPRs   []CreatedPR        `json:"created_prs,omitempty"`
}

// RepositoryResult represents the scan result for a single repository
type RepositoryResult struct {
	Name             string                     `json:"name"`
	FullName         string                     `json:"full_name"`
	DefaultBranch    string                     `json:"default_branch"`
	WorkflowFiles    []WorkflowFileResult       `json:"workflow_files"`
	Actions          []workflow.ActionReference `json:"actions"`
	Issues           []ActionIssue              `json:"issues,omitempty"`
	CustomProperties map[string]string          `json:"custom_properties,omitempty"`
}

// WorkflowFileResult represents a workflow file scan result
type WorkflowFileResult struct {
	Path        string                     `json:"path"`
	ActionCount int                        `json:"action_count"`
	Actions     []workflow.ActionReference `json:"actions"`
}

// ActionIssue represents an issue with an action (outdated version, deprecated, etc.)
type ActionIssue struct {
	Repository         string   `json:"repository"`
	CurrentVersion     string   `json:"current_version"`
	SuggestedVersion   string   `json:"suggested_version,omitempty"`
	IssueType          string   `json:"issue_type"` // "outdated", "deprecated", "migration"
	Severity           string   `json:"severity"`   // "low", "medium", "high", "critical"
	Description        string   `json:"description"`
	Context            string   `json:"context"` // where the issue was found
	FilePath           string   `json:"file_path"`
	SchemaChanges      []string `json:"schema_changes,omitempty"`      // Description of schema changes that will be applied
	HasTransformations bool     `json:"has_transformations,omitempty"` // Whether this upgrade includes schema transformations

	// Migration support: for actions that have moved to a new repository
	MigrationTarget string `json:"migration_target,omitempty"` // Target repository for migration (e.g., "new-org/action@v1")
}

// Summary provides aggregate statistics about the scan
type Summary struct {
	TotalRepositories     int                        `json:"total_repositories"`
	TotalWorkflowFiles    int                        `json:"total_workflow_files"`
	TotalActions          int                        `json:"total_actions"`           // Total of both actions and workflows
	TotalRegularActions   int                        `json:"total_regular_actions"`   // Only regular GitHub Actions
	TotalReusableWorkflows int                       `json:"total_reusable_workflows"` // Only reusable workflows
	UniqueActions         map[string]ActionUsageStat `json:"unique_actions"`         // Combined actions and workflows
	UniqueRegularActions  map[string]ActionUsageStat `json:"unique_regular_actions"` // Only regular actions
	UniqueReusableWorkflows map[string]ActionUsageStat `json:"unique_reusable_workflows"` // Only reusable workflows
	IssuesByType          map[string]int             `json:"issues_by_type"`
	IssuesBySeverity      map[string]int             `json:"issues_by_severity"`
	TopIssues             []ActionIssue              `json:"top_issues"`
}

// ActionUsageStat represents usage statistics for a specific action
type ActionUsageStat struct {
	Repository   string         `json:"repository"`
	UsageCount   int            `json:"usage_count"`
	Versions     map[string]int `json:"versions"`
	Repositories []string       `json:"repositories"`
	IsReusableWorkflow bool     `json:"is_reusable_workflow"` // true if this represents a reusable workflow
}

// CreatedPR represents a pull request that was created during the scan
type CreatedPR struct {
	Repository  string `json:"repository"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Number      int    `json:"number"`
	UpdateCount int    `json:"update_count"`
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
		ScanEndTime:  time.Time{}, // Will be set when scan completes
		Duration:     0,           // Will be calculated when scan completes
		Repositories: repositories,
		Summary:      summary,
		CreatedPRs:   []CreatedPR{}, // Will be populated if PRs are created
	}
}

// FinalizeScanResult updates the scan result with completion timing
func FinalizeScanResult(result *ScanResult) {
	result.ScanEndTime = time.Now()
	result.Duration = result.ScanEndTime.Sub(result.ScanTime)
}

// AddCreatedPR adds a created PR to the scan result
func AddCreatedPR(result *ScanResult, pr CreatedPR) {
	result.CreatedPRs = append(result.CreatedPRs, pr)
}

// calculateSummary generates summary statistics from repository results
func calculateSummary(repositories []RepositoryResult) Summary {
	summary := Summary{
		UniqueActions:           make(map[string]ActionUsageStat),
		UniqueRegularActions:    make(map[string]ActionUsageStat),
		UniqueReusableWorkflows: make(map[string]ActionUsageStat),
		IssuesByType:            make(map[string]int),
		IssuesBySeverity:        make(map[string]int),
	}

	totalWorkflowFiles := 0
	totalActions := 0
	totalRegularActions := 0
	totalReusableWorkflows := 0
	var allIssues []ActionIssue

	// Process each repository
	for _, repo := range repositories {
		summary.TotalRepositories++
		totalWorkflowFiles += len(repo.WorkflowFiles)

		// Process actions in this repository
		for _, action := range repo.Actions {
			totalActions++

			if action.IsReusable {
				totalReusableWorkflows++
			} else {
				totalRegularActions++
			}

			// Update combined unique actions statistics
			stat, exists := summary.UniqueActions[action.Repository]
			if !exists {
				stat = ActionUsageStat{
					Repository:         action.Repository,
					Versions:           make(map[string]int),
					Repositories:       make([]string, 0),
					IsReusableWorkflow: action.IsReusable,
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

			// Update type-specific statistics
			var typeSpecificMap map[string]ActionUsageStat
			if action.IsReusable {
				typeSpecificMap = summary.UniqueReusableWorkflows
			} else {
				typeSpecificMap = summary.UniqueRegularActions
			}

			typeStat, exists := typeSpecificMap[action.Repository]
			if !exists {
				typeStat = ActionUsageStat{
					Repository:         action.Repository,
					Versions:           make(map[string]int),
					Repositories:       make([]string, 0),
					IsReusableWorkflow: action.IsReusable,
				}
			}

			typeStat.UsageCount++
			typeStat.Versions[action.Version]++

			// Add repository to list if not already present
			found = false
			for _, repoName := range typeStat.Repositories {
				if repoName == repo.FullName {
					found = true
					break
				}
			}
			if !found {
				typeStat.Repositories = append(typeStat.Repositories, repo.FullName)
			}

			typeSpecificMap[action.Repository] = typeStat
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
	summary.TotalRegularActions = totalRegularActions
	summary.TotalReusableWorkflows = totalReusableWorkflows

	// Select top issues (limit to 10)
	summary.TopIssues = selectTopIssues(allIssues, 10)

	return summary
}

// WorkflowIssueGroup represents consolidated issues for a single workflow file
type WorkflowIssueGroup struct {
	FilePath    string
	IssueCount  int
	IssueType   string // Primary issue type (most common)
	Description string // Primary description (most severe issue)
	Severity    string // Highest severity among issues
}

// selectTopIssues selects the most important workflow files based on issue occurrence count
func selectTopIssues(issues []ActionIssue, limit int) []ActionIssue {
	if len(issues) == 0 {
		return []ActionIssue{}
	}

	// Group issues by workflow file path
	workflowGroups := make(map[string]*WorkflowIssueGroup)

	for _, issue := range issues {
		group, exists := workflowGroups[issue.FilePath]
		if !exists {
			group = &WorkflowIssueGroup{
				FilePath:    issue.FilePath,
				IssueCount:  0,
				IssueType:   issue.IssueType,
				Description: issue.Description,
				Severity:    issue.Severity,
			}
			workflowGroups[issue.FilePath] = group
		}

		group.IssueCount++

		// Update primary issue type to most common one
		// For simplicity, we'll keep the first issue type encountered
		// but prioritize higher severity issues for description
		if isHigherSeverity(issue.Severity, group.Severity) {
			group.IssueType = issue.IssueType
			group.Description = issue.Description
			group.Severity = issue.Severity
		}
	}

	// Convert to slice and sort by issue count (descending)
	type groupWithCount struct {
		group *WorkflowIssueGroup
		count int
	}

	var sortedGroups []groupWithCount
	for _, group := range workflowGroups {
		sortedGroups = append(sortedGroups, groupWithCount{group: group, count: group.IssueCount})
	}

	// Sort by issue count (descending), then by severity as tiebreaker
	for i := 0; i < len(sortedGroups)-1; i++ {
		for j := i + 1; j < len(sortedGroups); j++ {
			if sortedGroups[i].count < sortedGroups[j].count ||
				(sortedGroups[i].count == sortedGroups[j].count &&
					!isHigherSeverity(sortedGroups[i].group.Severity, sortedGroups[j].group.Severity)) {
				sortedGroups[i], sortedGroups[j] = sortedGroups[j], sortedGroups[i]
			}
		}
	}

	// Convert back to ActionIssue format, limiting to the specified count
	topIssues := make([]ActionIssue, 0, limit)
	for i, groupWithCount := range sortedGroups {
		if i >= limit {
			break
		}

		group := groupWithCount.group
		// Create a representative ActionIssue for this workflow file
		topIssues = append(topIssues, ActionIssue{
			Repository:       "", // Not applicable for workflow-level grouping
			CurrentVersion:   "", // Not applicable for workflow-level grouping
			SuggestedVersion: "",
			IssueType:        group.IssueType,
			Severity:         group.Severity,
			Description:      group.Description,
			Context:          fmt.Sprintf("%d issues found", group.IssueCount),
			FilePath:         group.FilePath,
		})
	}

	return topIssues
}

// isHigherSeverity returns true if severity1 is higher than severity2
func isHigherSeverity(severity1, severity2 string) bool {
	severityOrder := map[string]int{
		"critical": 4,
		"high":     3,
		"medium":   2,
		"low":      1,
	}

	return severityOrder[severity1] > severityOrder[severity2]
}
