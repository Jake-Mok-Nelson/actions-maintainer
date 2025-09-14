package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// NotebookCell represents a Jupyter notebook cell
type NotebookCell struct {
	CellType string   `json:"cell_type"`
	Source   []string `json:"source"`
	Metadata struct {
	} `json:"metadata"`
}

// JupyterNotebook represents a Jupyter notebook structure
type JupyterNotebook struct {
	Cells    []NotebookCell `json:"cells"`
	Metadata struct {
		KernelSpec struct {
			DisplayName string `json:"display_name"`
			Language    string `json:"language"`
			Name        string `json:"name"`
		} `json:"kernelspec"`
		LanguageInfo struct {
			Name string `json:"name"`
		} `json:"language_info"`
	} `json:"metadata"`
	NBFormat      int `json:"nbformat"`
	NBFormatMinor int `json:"nbformat_minor"`
}

// FormatNotebook outputs the scan results as a Jupyter notebook
func FormatNotebook(result *ScanResult, writer io.Writer) error {
	notebook := createNotebook(result)

	data, err := json.MarshalIndent(notebook, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal notebook JSON: %w", err)
	}

	_, err = writer.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write notebook: %w", err)
	}

	return nil
}

// createNotebook constructs a Jupyter notebook from scan results
func createNotebook(result *ScanResult) *JupyterNotebook {
	notebook := &JupyterNotebook{
		NBFormat:      4,
		NBFormatMinor: 4,
	}

	// Set up notebook metadata
	notebook.Metadata.KernelSpec.DisplayName = "Python 3"
	notebook.Metadata.KernelSpec.Language = "python"
	notebook.Metadata.KernelSpec.Name = "python3"
	notebook.Metadata.LanguageInfo.Name = "python"

	// Build cells
	cells := []NotebookCell{
		createHeaderCell(result),
		createSummaryCell(result),
		createIssuesOverviewCell(result),
		createRepositoryDetailsCell(result),
	}

	// Add PR links section if PRs were created
	if len(result.CreatedPRs) > 0 {
		cells = append(cells, createPRLinksCell(result))
	}

	// Add detailed statistics
	cells = append(cells, createDetailedStatsCell(result))

	notebook.Cells = cells
	return notebook
}

// createHeaderCell creates the main header with scan metadata
func createHeaderCell(result *ScanResult) NotebookCell {
	duration := "N/A"
	if !result.ScanEndTime.IsZero() {
		duration = result.Duration.String()
	}

	endTime := "In Progress"
	if !result.ScanEndTime.IsZero() {
		endTime = result.ScanEndTime.Format("2006-01-02 15:04:05 UTC")
	}

	source := []string{
		"# 📊 GitHub Actions Maintenance Report\n",
		"\n",
		fmt.Sprintf("**Organization/User:** `%s`\n", result.Owner),
		fmt.Sprintf("**Scan Started:** %s\n", result.ScanTime.Format("2006-01-02 15:04:05 UTC")),
		fmt.Sprintf("**Scan Completed:** %s\n", endTime),
		fmt.Sprintf("**Duration:** %s\n", duration),
		"\n",
		"---\n",
		"\n",
		"## 🎯 Executive Summary\n",
		"\n",
		fmt.Sprintf("- **%d** repositories scanned\n", result.Summary.TotalRepositories),
		fmt.Sprintf("- **%d** workflow files analyzed\n", result.Summary.TotalWorkflowFiles),
		fmt.Sprintf("- **%d** actions found across all workflows\n", result.Summary.TotalActions),
		fmt.Sprintf("- **%d** unique action types identified\n", len(result.Summary.UniqueActions)),
	}

	// Add issue summary
	totalIssues := 0
	for _, count := range result.Summary.IssuesByType {
		totalIssues += count
	}

	if totalIssues > 0 {
		source = append(source, fmt.Sprintf("- **%d** issues identified requiring attention\n", totalIssues))
	} else {
		source = append(source, "- ✅ **No issues found** - all actions are up to date!\n")
	}

	// Add PR summary if any were created
	if len(result.CreatedPRs) > 0 {
		source = append(source, fmt.Sprintf("- **%d** pull requests created for automated fixes\n", len(result.CreatedPRs)))
	}

	return NotebookCell{
		CellType: "markdown",
		Source:   source,
	}
}

// createSummaryCell creates a visual summary of the scan results
func createSummaryCell(result *ScanResult) NotebookCell {
	source := []string{
		"## 📈 Issue Breakdown\n",
		"\n",
	}

	// Issues by type
	if len(result.Summary.IssuesByType) > 0 {
		source = append(source, "### By Issue Type\n")
		source = append(source, "| Issue Type | Count | Percentage |\n")
		source = append(source, "|------------|-------|------------|\n")

		totalIssues := 0
		for _, count := range result.Summary.IssuesByType {
			totalIssues += count
		}

		for issueType, count := range result.Summary.IssuesByType {
			percentage := float64(count) / float64(totalIssues) * 100
			source = append(source, fmt.Sprintf("| %s | %d | %.1f%% |\n", issueType, count, percentage))
		}
		source = append(source, "\n")
	}

	// Issues by severity
	if len(result.Summary.IssuesBySeverity) > 0 {
		source = append(source, "### By Severity Level\n")
		source = append(source, "| Severity | Count | Priority |\n")
		source = append(source, "|----------|-------|----------|\n")

		// Order by severity priority
		severities := []string{"critical", "high", "medium", "low"}
		icons := map[string]string{
			"critical": "🔴",
			"high":     "🟠",
			"medium":   "🟡",
			"low":      "🟢",
		}

		for _, severity := range severities {
			if count, exists := result.Summary.IssuesBySeverity[severity]; exists {
				icon := icons[severity]
				source = append(source, fmt.Sprintf("| %s %s | %d | %s |\n", icon, strings.Title(severity), count, strings.ToUpper(severity)))
			}
		}
		source = append(source, "\n")
	}

	return NotebookCell{
		CellType: "markdown",
		Source:   source,
	}
}

// createIssuesOverviewCell creates an overview of the most critical issues
func createIssuesOverviewCell(result *ScanResult) NotebookCell {
	source := []string{
		"## 🚨 Top Issues Requiring Attention\n",
		"\n",
	}

	if len(result.Summary.TopIssues) == 0 {
		source = append(source, "✅ **No critical issues found!** All actions appear to be up to date.\n")
	} else {
		source = append(source, "The following issues require immediate attention:\n")
		source = append(source, "\n")

		for i, issue := range result.Summary.TopIssues {
			severityIcon := "🟢"
			switch issue.Severity {
			case "critical":
				severityIcon = "🔴"
			case "high":
				severityIcon = "🟠"
			case "medium":
				severityIcon = "🟡"
			}

			source = append(source, fmt.Sprintf("### %d. %s %s\n", i+1, severityIcon, issue.Repository))
			source = append(source, "\n")
			source = append(source, fmt.Sprintf("- **File:** `%s`\n", issue.FilePath))
			source = append(source, fmt.Sprintf("- **Current Version:** `%s`\n", issue.CurrentVersion))
			if issue.SuggestedVersion != "" {
				source = append(source, fmt.Sprintf("- **Suggested Version:** `%s`\n", issue.SuggestedVersion))
			}
			source = append(source, fmt.Sprintf("- **Issue Type:** %s\n", issue.IssueType))
			source = append(source, fmt.Sprintf("- **Description:** %s\n", issue.Description))
			source = append(source, "\n")
		}
	}

	return NotebookCell{
		CellType: "markdown",
		Source:   source,
	}
}

// createRepositoryDetailsCell creates detailed repository information
func createRepositoryDetailsCell(result *ScanResult) NotebookCell {
	source := []string{
		"## 📁 Repository Details\n",
		"\n",
	}

	if len(result.Repositories) == 0 {
		source = append(source, "No repositories found for the specified owner.\n")
		return NotebookCell{
			CellType: "markdown",
			Source:   source,
		}
	}

	source = append(source, "### Repository Summary\n")
	source = append(source, "| Repository | Workflows | Actions | Issues |\n")
	source = append(source, "|------------|-----------|---------|--------|\n")

	for _, repo := range result.Repositories {
		issueCount := len(repo.Issues)
		issueDisplay := fmt.Sprintf("%d", issueCount)
		if issueCount > 0 {
			issueDisplay = fmt.Sprintf("⚠️ %d", issueCount)
		}

		source = append(source, fmt.Sprintf("| [`%s`](https://github.com/%s) | %d | %d | %s |\n",
			repo.Name, repo.FullName, len(repo.WorkflowFiles), len(repo.Actions), issueDisplay))
	}

	source = append(source, "\n")

	// Add detailed breakdown for repositories with issues
	reposWithIssues := []RepositoryResult{}
	for _, repo := range result.Repositories {
		if len(repo.Issues) > 0 {
			reposWithIssues = append(reposWithIssues, repo)
		}
	}

	if len(reposWithIssues) > 0 {
		source = append(source, "### Repositories Requiring Updates\n")
		source = append(source, "\n")

		for _, repo := range reposWithIssues {
			source = append(source, fmt.Sprintf("#### [`%s`](https://github.com/%s)\n", repo.Name, repo.FullName))
			source = append(source, "\n")

			// Group issues by file
			fileIssues := make(map[string][]ActionIssue)
			for _, issue := range repo.Issues {
				fileIssues[issue.FilePath] = append(fileIssues[issue.FilePath], issue)
			}

			for filePath, issues := range fileIssues {
				source = append(source, fmt.Sprintf("**File:** `%s`\n", filePath))
				source = append(source, "\n")

				for _, issue := range issues {
					source = append(source, fmt.Sprintf("- **%s**: %s → %s (%s)\n",
						issue.Repository, issue.CurrentVersion, issue.SuggestedVersion, issue.IssueType))
				}
				source = append(source, "\n")
			}
		}
	}

	return NotebookCell{
		CellType: "markdown",
		Source:   source,
	}
}

// createPRLinksCell creates a section with links to created PRs
func createPRLinksCell(result *ScanResult) NotebookCell {
	source := []string{
		"## 🔗 Created Pull Requests\n",
		"\n",
		"The following pull requests were automatically created to resolve identified issues:\n",
		"\n",
	}

	// Sort PRs by repository name for consistent output
	prs := make([]CreatedPR, len(result.CreatedPRs))
	copy(prs, result.CreatedPRs)
	sort.Slice(prs, func(i, j int) bool {
		return prs[i].Repository < prs[j].Repository
	})

	source = append(source, "| Repository | PR | Title | Updates |\n")
	source = append(source, "|------------|----|-------|----------|\n")

	for _, pr := range prs {
		source = append(source, fmt.Sprintf("| [`%s`](https://github.com/%s) | [#%d](%s) | %s | %d |\n",
			pr.Repository, pr.Repository, pr.Number, pr.URL, pr.Title, pr.UpdateCount))
	}

	source = append(source, "\n")
	source = append(source, "💡 **Next Steps:**\n")
	source = append(source, "1. Review each pull request for accuracy\n")
	source = append(source, "2. Test the updated workflows in a safe environment\n")
	source = append(source, "3. Merge approved changes\n")
	source = append(source, "4. Monitor for any issues after deployment\n")

	return NotebookCell{
		CellType: "markdown",
		Source:   source,
	}
}

// createDetailedStatsCell creates detailed statistics about action usage
func createDetailedStatsCell(result *ScanResult) NotebookCell {
	source := []string{
		"## 📊 Detailed Action Statistics\n",
		"\n",
	}

	if len(result.Summary.UniqueActions) == 0 {
		source = append(source, "No actions found in the scanned repositories.\n")
		return NotebookCell{
			CellType: "markdown",
			Source:   source,
		}
	}

	source = append(source, "### Most Used Actions\n")
	source = append(source, "| Action | Usage Count | Unique Versions | Repositories |\n")
	source = append(source, "|--------|-------------|-----------------|---------------|\n")

	// Sort actions by usage count
	type ActionStat struct {
		Name  string
		Stats ActionUsageStat
	}

	var actionStats []ActionStat
	for name, stats := range result.Summary.UniqueActions {
		actionStats = append(actionStats, ActionStat{Name: name, Stats: stats})
	}

	sort.Slice(actionStats, func(i, j int) bool {
		return actionStats[i].Stats.UsageCount > actionStats[j].Stats.UsageCount
	})

	// Show top 10 most used actions
	limit := len(actionStats)
	if limit > 10 {
		limit = 10
	}

	for i := 0; i < limit; i++ {
		stat := actionStats[i]
		source = append(source, fmt.Sprintf("| `%s` | %d | %d | %d |\n",
			stat.Name, stat.Stats.UsageCount, len(stat.Stats.Versions), len(stat.Stats.Repositories)))
	}

	source = append(source, "\n")

	// Version distribution for top actions
	source = append(source, "### Version Distribution (Top 5 Actions)\n")
	source = append(source, "\n")

	versionLimit := len(actionStats)
	if versionLimit > 5 {
		versionLimit = 5
	}

	for i := 0; i < versionLimit; i++ {
		stat := actionStats[i]
		source = append(source, fmt.Sprintf("#### `%s`\n", stat.Name))
		source = append(source, "| Version | Count |\n")
		source = append(source, "|---------|-------|\n")

		// Sort versions by count
		type VersionCount struct {
			Version string
			Count   int
		}

		var versions []VersionCount
		for version, count := range stat.Stats.Versions {
			versions = append(versions, VersionCount{Version: version, Count: count})
		}

		sort.Slice(versions, func(i, j int) bool {
			return versions[i].Count > versions[j].Count
		})

		for _, vc := range versions {
			source = append(source, fmt.Sprintf("| `%s` | %d |\n", vc.Version, vc.Count))
		}
		source = append(source, "\n")
	}

	return NotebookCell{
		CellType: "markdown",
		Source:   source,
	}
}
