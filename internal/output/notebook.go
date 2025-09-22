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
		fmt.Sprintf("  - **%d** regular GitHub Actions\n", result.Summary.TotalRegularActions),
		fmt.Sprintf("  - **%d** reusable workflows\n", result.Summary.TotalReusableWorkflows),
		fmt.Sprintf("- **%d** unique action types identified\n", len(result.Summary.UniqueActions)),
		fmt.Sprintf("  - **%d** unique regular actions\n", len(result.Summary.UniqueRegularActions)),
		fmt.Sprintf("  - **%d** unique reusable workflows\n", len(result.Summary.UniqueReusableWorkflows)),
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

	// Check if any repositories have custom properties
	hasCustomProperties := false
	customPropertyKeys := make(map[string]bool)
	for _, repo := range result.Repositories {
		if len(repo.CustomProperties) > 0 {
			hasCustomProperties = true
			for key := range repo.CustomProperties {
				customPropertyKeys[key] = true
			}
		}
	}

	// Create sorted list of custom property keys
	var sortedPropertyKeys []string
	for key := range customPropertyKeys {
		sortedPropertyKeys = append(sortedPropertyKeys, key)
	}
	sort.Strings(sortedPropertyKeys)

	// Add custom property filtering interface if properties exist
	if hasCustomProperties {
		source = append(source, createCustomPropertyFilterInterface(sortedPropertyKeys)...)
	}

	source = append(source, "### Repository Summary\n")

	// Create table headers based on whether custom properties exist
	if hasCustomProperties {
		header := "| Repository | Workflows | Actions | Issues |"
		separator := "|------------|-----------|---------|--------|"
		for _, key := range sortedPropertyKeys {
			header += fmt.Sprintf(" %s |", key)
			separator += "--------|"
		}
		source = append(source, header+"\n")
		source = append(source, separator+"\n")
	} else {
		source = append(source, "| Repository | Workflows | Actions | Issues |\n")
		source = append(source, "|------------|-----------|---------|--------|\n")
	}

	for _, repo := range result.Repositories {
		issueCount := len(repo.Issues)
		issueDisplay := fmt.Sprintf("%d", issueCount)
		if issueCount > 0 {
			issueDisplay = fmt.Sprintf("⚠️ %d", issueCount)
		}

		row := fmt.Sprintf("| [`%s`](https://github.com/%s) | %d | %d | %s |",
			repo.Name, repo.FullName, len(repo.WorkflowFiles), len(repo.Actions), issueDisplay)

		// Add custom property values to the row
		if hasCustomProperties {
			for _, key := range sortedPropertyKeys {
				value := repo.CustomProperties[key]
				if value == "" {
					value = "-"
				}
				row += fmt.Sprintf(" %s |", value)
			}
		}

		source = append(source, row+"\n")
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

			// Add custom properties for this repository if they exist
			if len(repo.CustomProperties) > 0 {
				source = append(source, "**Custom Properties:**\n")
				for _, key := range sortedPropertyKeys {
					if value, exists := repo.CustomProperties[key]; exists && value != "" {
						source = append(source, fmt.Sprintf("- %s: %s\n", key, value))
					}
				}
				source = append(source, "\n")
			}

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

	// Add overview section with breakdown
	source = append(source, "### Overview\n")
	source = append(source, "| Type | Total Usage | Unique Items |\n")
	source = append(source, "|------|-------------|---------------|\n")
	source = append(source, fmt.Sprintf("| **Regular Actions** | %d | %d |\n", result.Summary.TotalRegularActions, len(result.Summary.UniqueRegularActions)))
	source = append(source, fmt.Sprintf("| **Reusable Workflows** | %d | %d |\n", result.Summary.TotalReusableWorkflows, len(result.Summary.UniqueReusableWorkflows)))
	source = append(source, fmt.Sprintf("| **Total** | %d | %d |\n", result.Summary.TotalActions, len(result.Summary.UniqueActions)))
	source = append(source, "\n")

	// Function to create stats table for a given action map
	createStatsTable := func(title string, actionsMap map[string]ActionUsageStat) []string {
		var tableSource []string
		
		if len(actionsMap) == 0 {
			tableSource = append(tableSource, fmt.Sprintf("No %s found.\n", strings.ToLower(title)))
			return tableSource
		}

		tableSource = append(tableSource, fmt.Sprintf("### %s\n", title))
		tableSource = append(tableSource, "| Action/Workflow | Usage Count | Unique Versions | Repositories |\n")
		tableSource = append(tableSource, "|-----------------|-------------|-----------------|---------------|\n")

		// Sort by usage count
		type ActionStat struct {
			Name  string
			Stats ActionUsageStat
		}

		var actionStats []ActionStat
		for name, stats := range actionsMap {
			actionStats = append(actionStats, ActionStat{Name: name, Stats: stats})
		}

		sort.Slice(actionStats, func(i, j int) bool {
			return actionStats[i].Stats.UsageCount > actionStats[j].Stats.UsageCount
		})

		// Show top 10 most used
		limit := len(actionStats)
		if limit > 10 {
			limit = 10
		}

		for i := 0; i < limit; i++ {
			stat := actionStats[i]
			tableSource = append(tableSource, fmt.Sprintf("| `%s` | %d | %d | %d |\n",
				stat.Name, stat.Stats.UsageCount, len(stat.Stats.Versions), len(stat.Stats.Repositories)))
		}

		tableSource = append(tableSource, "\n")
		return tableSource
	}

	// Add separate tables for regular actions and reusable workflows
	source = append(source, createStatsTable("Most Used Regular Actions", result.Summary.UniqueRegularActions)...)
	source = append(source, createStatsTable("Most Used Reusable Workflows", result.Summary.UniqueReusableWorkflows)...)

	// Version distribution for top combined actions (keep existing behavior)
	source = append(source, "### Version Distribution (Top 5 Overall)\n")
	source = append(source, "\n")

	// Sort all actions by usage count
	type ActionStat struct {
		Name  string
		Stats ActionUsageStat
	}

	var allActionStats []ActionStat
	for name, stats := range result.Summary.UniqueActions {
		allActionStats = append(allActionStats, ActionStat{Name: name, Stats: stats})
	}

	sort.Slice(allActionStats, func(i, j int) bool {
		return allActionStats[i].Stats.UsageCount > allActionStats[j].Stats.UsageCount
	})

	versionLimit := len(allActionStats)
	if versionLimit > 5 {
		versionLimit = 5
	}

	for i := 0; i < versionLimit; i++ {
		stat := allActionStats[i]
		actionType := "Action"
		if stat.Stats.IsReusableWorkflow {
			actionType = "Reusable Workflow"
		}
		source = append(source, fmt.Sprintf("#### `%s` (%s)\n", stat.Name, actionType))
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

// createCustomPropertyFilterInterface creates an interactive filtering interface for custom properties
func createCustomPropertyFilterInterface(propertyKeys []string) []string {
	var source []string

	source = append(source, "### 🔍 Custom Property Filters\n")
	source = append(source, "\n")
	source = append(source, "Use the dropdown filters below to filter repositories by custom properties:\n")
	source = append(source, "\n")

	// Create HTML-based filter interface that works in Jupyter notebooks
	source = append(source, "<div style='background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 10px 0;'>\n")
	source = append(source, "<h4>Property Filters</h4>\n")

	for _, key := range propertyKeys {
		source = append(source, fmt.Sprintf(`
<div style='margin: 5px 0;'>
<label for='filter_%s' style='font-weight: bold; margin-right: 10px;'>%s:</label>
<select id='filter_%s' onchange='filterRepositories()' style='padding: 5px; margin-right: 10px;'>
<option value=''>All</option>
</select>
<button onclick='clearFilter("%s")' style='padding: 3px 8px; font-size: 12px;'>Clear</button>
</div>
`, key, key, key, key))
	}

	source = append(source, `
<div style='margin-top: 10px;'>
<button onclick='clearAllFilters()' style='background-color: #dc3545; color: white; padding: 8px 16px; border: none; border-radius: 3px; cursor: pointer;'>Clear All Filters</button>
<span id='filterStatus' style='margin-left: 15px; font-weight: bold;'></span>
</div>
</div>

<script>
// Repository filtering functionality
let allRepositories = [];
let filteredRepositories = [];

// Initialize filters with data from the table
function initializeFilters() {
    // Get all table rows (skip header)
    const table = document.querySelector('table');
    if (!table) return;
    
    const rows = table.querySelectorAll('tr');
    const headerRow = rows[0];
    const dataRows = Array.from(rows).slice(1);
    
    // Parse header to find column indices for custom properties
    const headers = Array.from(headerRow.querySelectorAll('th, td')).map(th => th.textContent.trim());
    const propertyIndices = {};
`)

	for i, key := range propertyKeys {
		source = append(source, fmt.Sprintf("    propertyIndices['%s'] = %d; // Column index for %s\n", key, 4+i, key))
	}

	source = append(source, `
    
    // Extract repository data
    allRepositories = dataRows.map(row => {
        const cells = row.querySelectorAll('td');
        const repo = {
            element: row,
            name: cells[0] ? cells[0].textContent.trim() : '',
            properties: {}
        };
        
        // Extract custom property values
`)

	for _, key := range propertyKeys {
		source = append(source, fmt.Sprintf(`        if (propertyIndices['%s'] && cells[propertyIndices['%s']]) {
            repo.properties['%s'] = cells[propertyIndices['%s']].textContent.trim();
        }
`, key, key, key, key))
	}

	source = append(source, `        return repo;
    });
    
    // Populate filter dropdowns with unique values
`)

	for _, key := range propertyKeys {
		source = append(source, fmt.Sprintf(`    populateFilterDropdown('%s');
`, key))
	}

	source = append(source, `}

function populateFilterDropdown(propertyKey) {
    const select = document.getElementById('filter_' + propertyKey);
    if (!select) return;
    
    // Get unique values for this property
    const values = new Set();
    allRepositories.forEach(repo => {
        const value = repo.properties[propertyKey];
        if (value && value !== '-') {
            values.add(value);
        }
    });
    
    // Clear existing options except "All"
    select.innerHTML = '<option value="">All</option>';
    
    // Add unique values as options
    Array.from(values).sort().forEach(value => {
        const option = document.createElement('option');
        option.value = value;
        option.textContent = value;
        select.appendChild(option);
    });
}

function filterRepositories() {
    const filters = {};
`)

	for _, key := range propertyKeys {
		source = append(source, fmt.Sprintf(`    filters['%s'] = document.getElementById('filter_%s').value;
`, key, key))
	}

	source = append(source, `    
    // Apply filters
    filteredRepositories = allRepositories.filter(repo => {
        for (const [key, filterValue] of Object.entries(filters)) {
            if (filterValue && repo.properties[key] !== filterValue) {
                return false;
            }
        }
        return true;
    });
    
    // Show/hide repository rows
    allRepositories.forEach(repo => {
        const isVisible = filteredRepositories.includes(repo);
        repo.element.style.display = isVisible ? '' : 'none';
    });
    
    // Update status
    updateFilterStatus();
}

function clearFilter(propertyKey) {
    document.getElementById('filter_' + propertyKey).value = '';
    filterRepositories();
}

function clearAllFilters() {
`)

	for _, key := range propertyKeys {
		source = append(source, fmt.Sprintf(`    document.getElementById('filter_%s').value = '';
`, key))
	}

	source = append(source, `    filterRepositories();
}

function updateFilterStatus() {
    const statusElement = document.getElementById('filterStatus');
    if (filteredRepositories.length === allRepositories.length) {
        statusElement.textContent = 'Showing all repositories';
        statusElement.style.color = '#28a745';
    } else {
        statusElement.textContent = 'Showing ' + filteredRepositories.length + ' of ' + allRepositories.length + ' repositories';
        statusElement.style.color = '#007bff';
    }
}

// Initialize filters when the page loads
setTimeout(initializeFilters, 100);
</script>

`)

	source = append(source, "\n")
	return source
}
