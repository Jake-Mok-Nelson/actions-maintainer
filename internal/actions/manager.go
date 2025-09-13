package actions

import (
	"fmt"
	"strings"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/output"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
)

// Manager handles action version management and issue detection
type Manager struct {
	rules []Rule
}

// Rule defines a version enforcement rule for actions
type Rule struct {
	Repository         string          `json:"repository"`
	LatestVersion      string          `json:"latest_version"`
	MinimumVersion     string          `json:"minimum_version,omitempty"`
	DeprecatedVersions []string        `json:"deprecated_versions,omitempty"`
	SecurityIssues     []SecurityIssue `json:"security_issues,omitempty"`
	Recommendation     string          `json:"recommendation,omitempty"`
}

// SecurityIssue represents a known security issue with specific versions
type SecurityIssue struct {
	Versions    []string `json:"versions"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	CVE         string   `json:"cve,omitempty"`
}

// NewManager creates a new actions manager with default rules
func NewManager() *Manager {
	return &Manager{
		rules: getDefaultRules(),
	}
}

// AnalyzeActions analyzes action references and identifies issues
func (m *Manager) AnalyzeActions(actions []workflow.ActionReference) []output.ActionIssue {
	var issues []output.ActionIssue

	for _, action := range actions {
		actionIssues := m.analyzeAction(action)
		issues = append(issues, actionIssues...)
	}

	return issues
}

// analyzeAction analyzes a single action reference for issues
func (m *Manager) analyzeAction(action workflow.ActionReference) []output.ActionIssue {
	var issues []output.ActionIssue

	rule := m.findRule(action.Repository)
	if rule == nil {
		return issues // No rules for this action
	}

	// Check for outdated versions
	if m.isOutdated(action.Version, rule.LatestVersion) {
		issues = append(issues, output.ActionIssue{
			Repository:       action.Repository,
			CurrentVersion:   action.Version,
			SuggestedVersion: rule.LatestVersion,
			IssueType:        "outdated",
			Severity:         m.determineSeverity(action.Version, rule),
			Description:      fmt.Sprintf("Action %s is using version %s, latest is %s", action.Repository, action.Version, rule.LatestVersion),
			Context:          action.Context,
			FilePath:         action.FilePath,
		})
	}

	// Check for deprecated versions
	for _, deprecatedVersion := range rule.DeprecatedVersions {
		if action.Version == deprecatedVersion {
			issues = append(issues, output.ActionIssue{
				Repository:       action.Repository,
				CurrentVersion:   action.Version,
				SuggestedVersion: rule.LatestVersion,
				IssueType:        "deprecated",
				Severity:         "high",
				Description:      fmt.Sprintf("Action %s version %s is deprecated", action.Repository, action.Version),
				Context:          action.Context,
				FilePath:         action.FilePath,
			})
		}
	}

	// Check for security issues
	for _, securityIssue := range rule.SecurityIssues {
		for _, vulnerableVersion := range securityIssue.Versions {
			if m.versionMatches(action.Version, vulnerableVersion) {
				issues = append(issues, output.ActionIssue{
					Repository:       action.Repository,
					CurrentVersion:   action.Version,
					SuggestedVersion: "", // Security issues don't determine upgrade requirements
					IssueType:        "security",
					Severity:         securityIssue.Severity,
					Description:      fmt.Sprintf("Security issue: %s", securityIssue.Description),
					Context:          action.Context,
					FilePath:         action.FilePath,
				})
			}
		}
	}

	return issues
}

// findRule finds a rule for the given repository
func (m *Manager) findRule(repository string) *Rule {
	for _, rule := range m.rules {
		if rule.Repository == repository {
			return &rule
		}
	}
	return nil
}

// isOutdated checks if a version is outdated compared to the latest
func (m *Manager) isOutdated(current, latest string) bool {
	if current == latest {
		return false
	}

	// Don't flag branch references as outdated
	if current == "main" || current == "master" {
		return false
	}

	// Simple version comparison (in practice, use proper semver)
	currentMajor := extractMajorVersion(current)
	latestMajor := extractMajorVersion(latest)

	if currentMajor != "" && latestMajor != "" {
		return currentMajor < latestMajor
	}

	return current != latest
}

// determineSeverity determines the severity of an outdated version
func (m *Manager) determineSeverity(version string, rule *Rule) string {
	// Check if minimum version is specified
	if rule.MinimumVersion != "" {
		if m.isOutdated(version, rule.MinimumVersion) {
			return "high" // Below minimum version
		}
	}

	// Check major version difference
	currentMajor := extractMajorVersion(version)
	latestMajor := extractMajorVersion(rule.LatestVersion)

	if currentMajor != "" && latestMajor != "" {
		diff := parseVersion(latestMajor) - parseVersion(currentMajor)
		if diff >= 2 {
			return "medium" // Multiple major versions behind
		}
		if diff >= 1 {
			return "low" // One major version behind
		}
	}

	return "low"
}

// versionMatches checks if a version matches a pattern (exact match for now)
func (m *Manager) versionMatches(version, pattern string) bool {
	return version == pattern
}

// extractMajorVersion extracts the major version number from a version string
func extractMajorVersion(version string) string {
	if strings.HasPrefix(version, "v") {
		version = version[1:]
	}

	parts := strings.Split(version, ".")
	if len(parts) > 0 {
		return parts[0]
	}

	return version
}

// parseVersion converts a version string to an integer for comparison
func parseVersion(version string) int {
	if len(version) == 0 {
		return 0
	}

	// Simple conversion - in practice use proper semver parsing
	switch version {
	case "1":
		return 1
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	case "5":
		return 5
	default:
		return 0
	}
}

// getDefaultRules returns the default set of action rules
func getDefaultRules() []Rule {
	return []Rule{
		{
			Repository:         "actions/checkout",
			LatestVersion:      "v4",
			MinimumVersion:     "v3",
			DeprecatedVersions: []string{"v1"},
			SecurityIssues: []SecurityIssue{
				{
					Versions:    []string{"v1"},
					Severity:    "medium",
					Description: "Older versions may have security vulnerabilities",
				},
			},
			Recommendation: "Use v4 for the latest features and security fixes",
		},
		{
			Repository:         "actions/setup-node",
			LatestVersion:      "v4",
			MinimumVersion:     "v3",
			DeprecatedVersions: []string{"v1"},
			SecurityIssues:     []SecurityIssue{},
		},
		{
			Repository:         "actions/setup-python",
			LatestVersion:      "v5",
			MinimumVersion:     "v4",
			DeprecatedVersions: []string{"v1", "v2"},
			SecurityIssues:     []SecurityIssue{},
		},
		{
			Repository:         "actions/upload-artifact",
			LatestVersion:      "v4",
			MinimumVersion:     "v3",
			DeprecatedVersions: []string{"v1"},
			SecurityIssues:     []SecurityIssue{},
		},
		{
			Repository:         "actions/download-artifact",
			LatestVersion:      "v4",
			MinimumVersion:     "v3",
			DeprecatedVersions: []string{"v1"},
			SecurityIssues:     []SecurityIssue{},
		},
		{
			Repository:     "actions/cache",
			LatestVersion:  "v4",
			MinimumVersion: "v3",
			SecurityIssues: []SecurityIssue{},
		},
		{
			Repository:     "actions/setup-go",
			LatestVersion:  "v5",
			MinimumVersion: "v4",
			SecurityIssues: []SecurityIssue{},
		},
		{
			Repository:     "actions/setup-java",
			LatestVersion:  "v4",
			MinimumVersion: "v3",
			SecurityIssues: []SecurityIssue{},
		},
	}
}
