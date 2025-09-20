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
		createVersionComparisonCell(result),
		createIssuesOverviewCell(result),
		createUpgradeFlowCell(result),
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
		"## 📈 Issue Breakdown & Visual Summary\n",
		"\n",
	}

	// Issues by type with visual charts
	if len(result.Summary.IssuesByType) > 0 {
		source = append(source, "### By Issue Type\n")
		
		totalIssues := 0
		for _, count := range result.Summary.IssuesByType {
			totalIssues += count
		}

		// Visual overview with icons and progress bars
		source = append(source, "#### 📊 Visual Overview\n")
		source = append(source, "```\n")
		
		maxCount := 0
		for _, count := range result.Summary.IssuesByType {
			if count > maxCount {
				maxCount = count
			}
		}
		
		for issueType, count := range result.Summary.IssuesByType {
			percentage := float64(count) / float64(totalIssues) * 100
			barLength := int(float64(count) / float64(maxCount) * 20) // Max 20 chars
			
			var icon string
			switch issueType {
			case "outdated":
				icon = "📅"
			case "deprecated":
				icon = "⚠️"
			case "migration":
				icon = "🚚"
			case "vulnerable":
				icon = "🔒"
			default:
				icon = "❓"
			}
			
			bar := ""
			for i := 0; i < 20; i++ {
				if i < barLength {
					bar += "█"
				} else {
					bar += "▱"
				}
			}
			
			source = append(source, fmt.Sprintf("%s %-12s |%s| %d (%.1f%%)\n", 
				icon, strings.Title(issueType), bar, count, percentage))
		}
		source = append(source, "```\n")
		source = append(source, "\n")

		// Traditional table
		source = append(source, "#### 📋 Detailed Breakdown\n")
		source = append(source, "| Issue Type | Count | Percentage | Impact |\n")
		source = append(source, "|------------|-------|------------|--------|\n")

		for issueType, count := range result.Summary.IssuesByType {
			percentage := float64(count) / float64(totalIssues) * 100
			
			var impact string
			if percentage >= 50 {
				impact = "🔴 High"
			} else if percentage >= 25 {
				impact = "🟠 Medium"
			} else {
				impact = "🟢 Low"
			}
			
			source = append(source, fmt.Sprintf("| %s | %d | %.1f%% | %s |\n", issueType, count, percentage, impact))
		}
		source = append(source, "\n")
	}

	// Issues by severity with enhanced visualization
	if len(result.Summary.IssuesBySeverity) > 0 {
		source = append(source, "### By Severity Level\n")
		
		// Calculate total for percentages
		totalSeverityIssues := 0
		for _, count := range result.Summary.IssuesBySeverity {
			totalSeverityIssues += count
		}

		// Visual severity chart
		source = append(source, "#### 🌡️ Severity Distribution\n")
		source = append(source, "```\n")
		
		severities := []string{"critical", "high", "medium", "low"}
		severityIcons := map[string]string{
			"critical": "🔴",
			"high":     "🟠",
			"medium":   "🟡",
			"low":      "🟢",
		}

		maxSeverityCount := 0
		for _, count := range result.Summary.IssuesBySeverity {
			if count > maxSeverityCount {
				maxSeverityCount = count
			}
		}

		for _, severity := range severities {
			if count, exists := result.Summary.IssuesBySeverity[severity]; exists {
				percentage := float64(count) / float64(totalSeverityIssues) * 100
				barLength := int(float64(count) / float64(maxSeverityCount) * 20)
				
				bar := ""
				for i := 0; i < 20; i++ {
					if i < barLength {
						bar += "█"
					} else {
						bar += "▱"
					}
				}
				
				icon := severityIcons[severity]
				source = append(source, fmt.Sprintf("%s %-9s |%s| %d (%.1f%%)\n", 
					icon, strings.Title(severity), bar, count, percentage))
			}
		}
		source = append(source, "```\n")
		source = append(source, "\n")

		// Traditional severity table with enhanced priority indicators
		source = append(source, "#### ⚡ Priority Matrix\n")
		source = append(source, "| Severity | Count | Priority | Action Required |\n")
		source = append(source, "|----------|-------|----------|------------------|\n")

		for _, severity := range severities {
			if count, exists := result.Summary.IssuesBySeverity[severity]; exists {
				icon := severityIcons[severity]
				
				var actionRequired string
				switch severity {
				case "critical":
					actionRequired = "🚨 Immediate"
				case "high":
					actionRequired = "⏰ Within 1 week"
				case "medium":
					actionRequired = "📅 Within 1 month"
				case "low":
					actionRequired = "🗓️ Next maintenance"
				}
				
				source = append(source, fmt.Sprintf("| %s %s | %d | %s | %s |\n", 
					icon, strings.Title(severity), count, strings.ToUpper(severity), actionRequired))
			}
		}
		source = append(source, "\n")
	}

	// Add overall health score
	if len(result.Summary.IssuesByType) > 0 || len(result.Summary.IssuesBySeverity) > 0 {
		source = append(source, "### 🏥 Repository Health Score\n")
		source = append(source, "\n")
		
		totalActions := result.Summary.TotalActions
		totalIssues := 0
		for _, count := range result.Summary.IssuesByType {
			totalIssues += count
		}
		
		healthScore := 100.0
		if totalActions > 0 {
			healthScore = float64(totalActions-totalIssues) / float64(totalActions) * 100
		}
		
		var healthIcon, healthStatus string
		if healthScore >= 90 {
			healthIcon = "💚"
			healthStatus = "Excellent"
		} else if healthScore >= 75 {
			healthIcon = "💛"
			healthStatus = "Good"
		} else if healthScore >= 50 {
			healthIcon = "🧡"
			healthStatus = "Needs Attention"
		} else {
			healthIcon = "❤️"
			healthStatus = "Critical"
		}

		source = append(source, fmt.Sprintf("**Overall Score:** %s **%.1f%%** (%s)\n", healthIcon, healthScore, healthStatus))
		source = append(source, "\n")
		
		// Health score bar
		healthBars := int(healthScore / 5) // 20 bars total (100/5)
		source = append(source, "```\n")
		source = append(source, "Health Score: ")
		for i := 0; i < 20; i++ {
			if i < healthBars {
				source = append(source, "█")
			} else {
				source = append(source, "▱")
			}
		}
		source = append(source, fmt.Sprintf(" %.1f%%\n", healthScore))
		source = append(source, "```\n")
		source = append(source, "\n")
		
		// Recommendations based on health score
		source = append(source, "#### 💡 Recommendations\n")
		if healthScore >= 90 {
			source = append(source, "- ✅ Your repository is in excellent shape!\n")
			source = append(source, "- 🔄 Consider setting up automated dependency updates\n")
		} else if healthScore >= 75 {
			source = append(source, "- 👍 Good job maintaining your actions!\n")
			source = append(source, "- 🎯 Focus on the remaining high-priority issues\n")
		} else if healthScore >= 50 {
			source = append(source, "- ⚠️ Several issues need attention\n")
			source = append(source, "- 🚀 Prioritize critical and high-severity updates\n")
			source = append(source, "- 📋 Create a maintenance plan for systematic updates\n")
		} else {
			source = append(source, "- 🚨 Immediate action required!\n")
			source = append(source, "- 🔥 Address critical issues first\n")
			source = append(source, "- 📞 Consider reaching out to your team for help\n")
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

			// Use file path as the main identifier since issues are now grouped by workflow file
			source = append(source, fmt.Sprintf("### %d. %s %s\n", i+1, severityIcon, issue.FilePath))
			source = append(source, "\n")
			source = append(source, fmt.Sprintf("- **Finding:** %s\n", issue.IssueType))
			source = append(source, fmt.Sprintf("- **Description:** %s\n", issue.Description))
			if issue.Context != "" {
				source = append(source, fmt.Sprintf("- **Issues Found:** %s\n", issue.Context))
			}
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
		"## 📊 Detailed Action Statistics & Analytics\n",
		"\n",
	}

	if len(result.Summary.UniqueActions) == 0 {
		source = append(source, "No actions found in the scanned repositories.\n")
		return NotebookCell{
			CellType: "markdown",
			Source:   source,
		}
	}

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

	// Visual usage chart
	source = append(source, "### 📈 Action Usage Visualization\n")
	source = append(source, "\n")
	
	// Show top 10 most used actions with visual bars
	limit := len(actionStats)
	if limit > 10 {
		limit = 10
	}

	if limit > 0 {
		maxUsage := actionStats[0].Stats.UsageCount
		
		source = append(source, "#### 🏆 Top Action Usage (Visual Chart)\n")
		source = append(source, "```\n")
		for i := 0; i < limit; i++ {
			stat := actionStats[i]
			barLength := int(float64(stat.Stats.UsageCount) / float64(maxUsage) * 30) // Max 30 chars
			
			bar := ""
			for j := 0; j < 30; j++ {
				if j < barLength {
					bar += "█"
				} else {
					bar += "▱"
				}
			}
			
			// Truncate action name if too long
			actionName := stat.Name
			if len(actionName) > 25 {
				actionName = actionName[:22] + "..."
			}
			
			source = append(source, fmt.Sprintf("%-25s |%s| %d\n", actionName, bar, stat.Stats.UsageCount))
		}
		source = append(source, "```\n")
		source = append(source, "\n")
	}

	source = append(source, "### 📋 Most Used Actions (Detailed Table)\n")
	source = append(source, "| Rank | Action | Usage Count | Unique Versions | Repositories | Popularity |\n")
	source = append(source, "|------|--------|-------------|-----------------|--------------|-------------|\n")

	for i := 0; i < limit; i++ {
		stat := actionStats[i]
		
		// Calculate popularity indicator
		var popularityIcon string
		percentage := float64(stat.Stats.UsageCount) / float64(result.Summary.TotalActions) * 100
		if percentage >= 25 {
			popularityIcon = "🔥 Very Popular"
		} else if percentage >= 10 {
			popularityIcon = "⭐ Popular"
		} else if percentage >= 5 {
			popularityIcon = "👍 Common"
		} else {
			popularityIcon = "📄 Occasional"
		}
		
		rank := ""
		if i == 0 {
			rank = "🥇"
		} else if i == 1 {
			rank = "🥈"
		} else if i == 2 {
			rank = "🥉"
		} else {
			rank = fmt.Sprintf("#%d", i+1)
		}
		
		source = append(source, fmt.Sprintf("| %s | `%s` | %d | %d | %d | %s |\n",
			rank, stat.Name, stat.Stats.UsageCount, len(stat.Stats.Versions), len(stat.Stats.Repositories), popularityIcon))
	}

	source = append(source, "\n")

	// Version distribution for top actions with enhanced visualization
	source = append(source, "### 🔢 Version Distribution Analysis (Top 5 Actions)\n")
	source = append(source, "\n")

	versionLimit := len(actionStats)
	if versionLimit > 5 {
		versionLimit = 5
	}

	for i := 0; i < versionLimit; i++ {
		stat := actionStats[i]
		source = append(source, fmt.Sprintf("#### `%s` - Version Usage Distribution\n", stat.Name))
		source = append(source, "\n")

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

		// Visual version distribution
		if len(versions) > 0 {
			maxVersionCount := versions[0].Count
			
			source = append(source, "**Visual Distribution:**\n")
			source = append(source, "```\n")
			for _, vc := range versions {
				barLength := int(float64(vc.Count) / float64(maxVersionCount) * 20)
				percentage := float64(vc.Count) / float64(stat.Stats.UsageCount) * 100
				
				bar := ""
				for j := 0; j < 20; j++ {
					if j < barLength {
						bar += "█"
					} else {
						bar += "▱"
					}
				}
				
				// Add version status indicator
				var statusIcon string
				if strings.Contains(vc.Version, "v4") || strings.Contains(vc.Version, "v5") {
					statusIcon = "✅"
				} else if strings.Contains(vc.Version, "v3") {
					statusIcon = "⚠️"
				} else if strings.Contains(vc.Version, "v1") || strings.Contains(vc.Version, "v2") {
					statusIcon = "🔴"
				} else {
					statusIcon = "❓"
				}
				
				source = append(source, fmt.Sprintf("%s %-8s |%s| %d (%.1f%%)\n", 
					statusIcon, vc.Version, bar, vc.Count, percentage))
			}
			source = append(source, "```\n")
			source = append(source, "\n")
		}

		// Traditional table
		source = append(source, "**Detailed Breakdown:**\n")
		source = append(source, "| Version | Count | Percentage | Status |\n")
		source = append(source, "|---------|-------|------------|--------|\n")

		for _, vc := range versions {
			percentage := float64(vc.Count) / float64(stat.Stats.UsageCount) * 100
			
			var status string
			if strings.Contains(vc.Version, "v4") || strings.Contains(vc.Version, "v5") {
				status = "✅ Current"
			} else if strings.Contains(vc.Version, "v3") {
				status = "⚠️ Outdated"
			} else {
				status = "🔴 Legacy"
			}
			
			source = append(source, fmt.Sprintf("| `%s` | %d | %.1f%% | %s |\n", vc.Version, vc.Count, percentage, status))
		}
		source = append(source, "\n")
	}

	// Action diversity analysis
	source = append(source, "### 🌈 Action Diversity Analysis\n")
	source = append(source, "\n")
	
	// Calculate ecosystem distribution
	ecosystems := make(map[string]int)
	for _, stat := range actionStats {
		if strings.HasPrefix(stat.Name, "actions/") {
			ecosystems["GitHub Official"]++
		} else if strings.Contains(stat.Name, "/") {
			ecosystems["Third-party"]++
		} else {
			ecosystems["Other"]++
		}
	}
	
	source = append(source, "#### 🏢 Ecosystem Distribution\n")
	source = append(source, "```\n")
	totalEcosystems := len(actionStats)
	for ecosystem, count := range ecosystems {
		percentage := float64(count) / float64(totalEcosystems) * 100
		barLength := int(percentage / 5) // Max 20 chars (100/5)
		
		bar := ""
		for j := 0; j < 20; j++ {
			if j < barLength {
				bar += "█"
			} else {
				bar += "▱"
			}
		}
		
		source = append(source, fmt.Sprintf("%-15s |%s| %d (%.1f%%)\n", ecosystem, bar, count, percentage))
	}
	source = append(source, "```\n")

	return NotebookCell{
		CellType: "markdown",
		Source:   source,
	}
}
// createVersionComparisonCell creates a visual comparison of current vs target versions
func createVersionComparisonCell(result *ScanResult) NotebookCell {
	source := []string{
		"## 🔄 Version Comparison Dashboard\n",
		"\n",
		"Visual overview of action versions and their upgrade recommendations:\n",
		"\n",
	}

	if len(result.Summary.UniqueActions) == 0 {
		source = append(source, "No actions found for version comparison.\n")
		return NotebookCell{
			CellType: "markdown",
			Source:   source,
		}
	}

	// Create visual version comparison table
	source = append(source, "### 📊 Current vs Recommended Versions\n")
	source = append(source, "\n")
	source = append(source, "| Action | Current | → | Target | Status | Progress |\n")
	source = append(source, "|--------|---------|---|--------|--------|-----------|\n")

	// Collect version issues from repositories
	versionIssues := make(map[string]struct {
		currentVersions []string
		targetVersion   string
		severity        string
	})

	for _, repo := range result.Repositories {
		for _, issue := range repo.Issues {
			if issue.IssueType == "outdated" {
				entry := versionIssues[issue.Repository]
				entry.targetVersion = issue.SuggestedVersion
				entry.severity = issue.Severity
				
				// Add current version if not already present
				found := false
				for _, v := range entry.currentVersions {
					if v == issue.CurrentVersion {
						found = true
						break
					}
				}
				if !found {
					entry.currentVersions = append(entry.currentVersions, issue.CurrentVersion)
				}
				versionIssues[issue.Repository] = entry
			}
		}
	}

	// Add entries for actions with issues
	for action, issue := range versionIssues {
		currentVersionsStr := strings.Join(issue.currentVersions, ", ")
		
		var statusIcon, progressBar string
		switch issue.severity {
		case "critical":
			statusIcon = "🔴 Critical"
			progressBar = "▱▱▱▱▱ 0%"
		case "high":
			statusIcon = "🟠 High"
			progressBar = "▰▱▱▱▱ 20%"
		case "medium":
			statusIcon = "🟡 Medium"  
			progressBar = "▰▰▱▱▱ 40%"
		case "low":
			statusIcon = "🟢 Low"
			progressBar = "▰▰▰▰▱ 80%"
		default:
			statusIcon = "⚪ Unknown"
			progressBar = "▱▱▱▱▱ ?"
		}

		source = append(source, fmt.Sprintf("| `%s` | `%s` | ➡️ | `%s` | %s | %s |\n",
			action, currentVersionsStr, issue.targetVersion, statusIcon, progressBar))
	}

	// Add entries for up-to-date actions
	for action, stats := range result.Summary.UniqueActions {
		if _, hasIssue := versionIssues[action]; !hasIssue {
			// This action is up-to-date
			var versions []string
			for version := range stats.Versions {
				versions = append(versions, version)
			}
			currentVersionsStr := strings.Join(versions, ", ")
			source = append(source, fmt.Sprintf("| `%s` | `%s` | ✅ | Current | 🟢 Up-to-date | ▰▰▰▰▰ 100%% |\n",
				action, currentVersionsStr))
		}
	}

	source = append(source, "\n")
	source = append(source, "### 📈 Version Upgrade Impact Analysis\n")
	source = append(source, "\n")

	// Calculate version gap statistics
	majorUpgrades := 0
	minorUpgrades := 0
	for _, issue := range versionIssues {
		for _, current := range issue.currentVersions {
			currentMajor := extractMajorVersionFromString(current)
			targetMajor := extractMajorVersionFromString(issue.targetVersion)
			if currentMajor != "" && targetMajor != "" && currentMajor != targetMajor {
				majorUpgrades++
			} else {
				minorUpgrades++
			}
		}
	}

	totalUpgrades := majorUpgrades + minorUpgrades
	if totalUpgrades > 0 {
		majorPercent := float64(majorUpgrades) / float64(totalUpgrades) * 100
		minorPercent := float64(minorUpgrades) / float64(totalUpgrades) * 100

		source = append(source, "#### 📊 Upgrade Type Distribution\n")
		source = append(source, fmt.Sprintf("- **Major Version Upgrades:** %d (%.1f%%) - May require workflow changes\n", majorUpgrades, majorPercent))
		source = append(source, fmt.Sprintf("- **Minor Version Upgrades:** %d (%.1f%%) - Generally safe upgrades\n", minorUpgrades, minorPercent))
		source = append(source, "\n")

		// Visual bar chart using Unicode blocks
		majorBlocks := int(majorPercent / 10)
		minorBlocks := int(minorPercent / 10)
		
		source = append(source, "```\n")
		source = append(source, "Major Upgrades  ")
		for i := 0; i < 10; i++ {
			if i < majorBlocks {
				source = append(source, "█")
			} else {
				source = append(source, "▱")
			}
		}
		source = append(source, fmt.Sprintf(" %.1f%%\n", majorPercent))
		
		source = append(source, "Minor Upgrades  ")
		for i := 0; i < 10; i++ {
			if i < minorBlocks {
				source = append(source, "█")
			} else {
				source = append(source, "▱")
			}
		}
		source = append(source, fmt.Sprintf(" %.1f%%\n", minorPercent))
		source = append(source, "```\n")
	}

	return NotebookCell{
		CellType: "markdown",
		Source:   source,
	}
}

// createUpgradeFlowCell creates visual upgrade flow diagrams
func createUpgradeFlowCell(result *ScanResult) NotebookCell {
	source := []string{
		"## 🔀 Upgrade Flow Diagrams\n",
		"\n",
		"Visual representation of recommended upgrade paths:\n",
		"\n",
	}

	// Track upgrade paths
	upgradePaths := make(map[string][]struct {
		from string
		to   string
		severity string
	})

	for _, repo := range result.Repositories {
		for _, issue := range repo.Issues {
			if issue.IssueType == "outdated" || issue.IssueType == "migration" {
				paths := upgradePaths[issue.Repository]
				paths = append(paths, struct {
					from string
					to   string
					severity string
				}{
					from: issue.CurrentVersion,
					to:   issue.SuggestedVersion,
					severity: issue.Severity,
				})
				upgradePaths[issue.Repository] = paths
			}
		}
	}

	if len(upgradePaths) == 0 {
		source = append(source, "✅ No upgrades needed - all actions are current!\n")
		return NotebookCell{
			CellType: "markdown",
			Source:   source,
		}
	}

	for action, paths := range upgradePaths {
		source = append(source, fmt.Sprintf("### 🔄 `%s` Upgrade Path\n", action))
		source = append(source, "\n")
		source = append(source, "```mermaid\n")
		source = append(source, "flowchart LR\n")
		
		// Create unique path visualization
		pathMap := make(map[string]bool)
		for _, path := range paths {
			pathKey := fmt.Sprintf("%s->%s", path.from, path.to)
			if !pathMap[pathKey] {
				var arrow string
				switch path.severity {
				case "critical":
					arrow = "==>"
				case "high":
					arrow = "-->"
				default:
					arrow = "-.->"
				}
				
				source = append(source, fmt.Sprintf("    %s %s %s\n", 
					sanitizeNodeName(path.from), arrow, sanitizeNodeName(path.to)))
				pathMap[pathKey] = true
			}
		}
		source = append(source, "```\n")
		source = append(source, "\n")

		// Add textual upgrade steps
		source = append(source, "#### Upgrade Steps:\n")
		uniquePaths := make(map[string]string)
		for _, path := range paths {
			key := fmt.Sprintf("%s->%s", path.from, path.to)
			if _, exists := uniquePaths[key]; !exists {
				var priorityIcon string
				switch path.severity {
				case "critical":
					priorityIcon = "🔴"
				case "high":
					priorityIcon = "🟠"
				case "medium":
					priorityIcon = "🟡"
				default:
					priorityIcon = "🟢"
				}
				
				source = append(source, fmt.Sprintf("1. %s Update from `%s` to `%s` (%s priority)\n",
					priorityIcon, path.from, path.to, path.severity))
				uniquePaths[key] = path.severity
			}
		}
		source = append(source, "\n")
	}

	// Add migration flows if any
	migrationPaths := make(map[string]struct {
		fromRepo string
		toRepo   string
		fromVersion string
		toVersion string
	})

	for _, repo := range result.Repositories {
		for _, issue := range repo.Issues {
			if issue.IssueType == "migration" && issue.MigrationTarget != "" {
				// Parse migration target (format: "new-org/action@version")
				parts := strings.Split(issue.MigrationTarget, "@")
				if len(parts) == 2 {
					migrationPaths[issue.Repository] = struct {
						fromRepo string
						toRepo   string
						fromVersion string
						toVersion string
					}{
						fromRepo: issue.Repository,
						toRepo:   parts[0],
						fromVersion: issue.CurrentVersion,
						toVersion: parts[1],
					}
				}
			}
		}
	}

	if len(migrationPaths) > 0 {
		source = append(source, "### 🚚 Repository Migration Paths\n")
		source = append(source, "\n")
		
		for _, migration := range migrationPaths {
			source = append(source, fmt.Sprintf("#### `%s` → `%s`\n", migration.fromRepo, migration.toRepo))
			source = append(source, "\n")
			source = append(source, "```mermaid\n")
			source = append(source, "flowchart TD\n")
			source = append(source, fmt.Sprintf("    A[\"%s@%s\"] --> B[\"%s@%s\"]\n",
				migration.fromRepo, migration.fromVersion,
				migration.toRepo, migration.toVersion))
			source = append(source, "    A -.-> C[\"⚠️ Deprecated\"]\n")
			source = append(source, "    B -.-> D[\"✅ Recommended\"]\n")
			source = append(source, "```\n")
			source = append(source, "\n")
		}
	}

	return NotebookCell{
		CellType: "markdown",
		Source:   source,
	}
}

// Helper function to extract major version from version string
func extractMajorVersionFromString(version string) string {
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// Helper function to sanitize node names for mermaid diagrams
func sanitizeNodeName(name string) string {
	// Replace special characters that might break mermaid syntax
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return fmt.Sprintf("V%s", name)
}
