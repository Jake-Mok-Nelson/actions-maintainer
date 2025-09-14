package actions

import (
	"fmt"
	"strings"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/output"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/patcher"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
)

// Manager handles action version management and issue detection
type Manager struct {
	rules    []Rule
	patcher  *patcher.WorkflowPatcher
	resolver VersionResolver // Interface for version resolution
}

// VersionResolver interface for resolving version aliases
type VersionResolver interface {
	AreVersionsEquivalent(repository, version1, version2 string) (bool, error)
	IsVersionOutdated(repository, currentVersion, latestVersion string) (bool, error)
}

// Rule defines a version enforcement rule for actions
type Rule struct {
	Repository         string   `json:"repository"`
	LatestVersion      string   `json:"latest_version"`
	MinimumVersion     string   `json:"minimum_version,omitempty"`
	DeprecatedVersions []string `json:"deprecated_versions,omitempty"`
	Recommendation     string   `json:"recommendation,omitempty"`
}

// NewManager creates a new actions manager with default rules
func NewManager() *Manager {
	return &Manager{
		rules:   getDefaultRules(),
		patcher: patcher.NewWorkflowPatcher(),
	}
}

// NewManagerWithResolver creates a new actions manager with a version resolver
func NewManagerWithResolver(resolver VersionResolver) *Manager {
	return &Manager{
		rules:    getDefaultRules(),
		patcher:  patcher.NewWorkflowPatcher(),
		resolver: resolver,
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
	if m.isOutdatedForRepository(action.Repository, action.Version, rule.LatestVersion) {
		issue := output.ActionIssue{
			Repository:       action.Repository,
			CurrentVersion:   action.Version,
			SuggestedVersion: rule.LatestVersion,
			IssueType:        "outdated",
			Severity:         m.determineSeverity(action.Version, rule),
			Description:      fmt.Sprintf("Action %s is using version %s, latest is %s", action.Repository, action.Version, rule.LatestVersion),
			Context:          action.Context,
			FilePath:         action.FilePath,
		}

		// Check if there are schema transformations for this version upgrade
		if patchInfo, hasPatches := m.GetTransformationInfo(action.Repository, action.Version, rule.LatestVersion); hasPatches {
			issue.HasTransformations = true
			issue.SchemaChanges = []string{patchInfo.Description}

			// Add details about specific field changes
			for _, patch := range patchInfo.Patches {
				change := fmt.Sprintf("%s: %s", patch.Operation, patch.Reason)
				issue.SchemaChanges = append(issue.SchemaChanges, change)
			}
		}

		issues = append(issues, issue)
	}

	// Check for deprecated versions
	for _, deprecatedVersion := range rule.DeprecatedVersions {
		if action.Version == deprecatedVersion {
			issue := output.ActionIssue{
				Repository:       action.Repository,
				CurrentVersion:   action.Version,
				SuggestedVersion: rule.LatestVersion,
				IssueType:        "deprecated",
				Severity:         "high",
				Description:      fmt.Sprintf("Action %s version %s is deprecated", action.Repository, action.Version),
				Context:          action.Context,
				FilePath:         action.FilePath,
			}

			// Check if there are schema transformations for this version upgrade
			if patchInfo, hasPatches := m.GetTransformationInfo(action.Repository, action.Version, rule.LatestVersion); hasPatches {
				issue.HasTransformations = true
				issue.SchemaChanges = []string{patchInfo.Description}

				// Add details about specific field changes
				for _, patch := range patchInfo.Patches {
					change := fmt.Sprintf("%s: %s", patch.Operation, patch.Reason)
					issue.SchemaChanges = append(issue.SchemaChanges, change)
				}
			}

			issues = append(issues, issue)
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
	return m.isOutdatedForRepository("", current, latest)
}

// isOutdatedForRepository checks if a version is outdated compared to the latest for a specific repository
//
// Version Alias Integration:
// This method integrates with the version resolver to provide intelligent version comparison.
// When a resolver is available and repository is provided, it first attempts to resolve
// both versions to their commit SHAs using the GitHub API. If the SHAs are identical,
// the versions are considered equivalent regardless of their string representation.
//
// This enables scenarios like:
// - v1 tag pointing to the same commit as v1.2.4 -> not outdated
// - v4 and commit SHA abc123 pointing to same commit -> not outdated
// - Branch references (main, master) -> never considered outdated
//
// Fallback Chain:
// 1. Try resolver-based SHA comparison (if resolver available and repository provided)
// 2. Fall back to traditional string-based major version comparison
// 3. Fall back to simple string inequality check
func (m *Manager) isOutdatedForRepository(repository, current, latest string) bool {
	if current == latest {
		return false
	}

	// Use cache-first version resolver if available and repository is provided
	if m.resolver != nil && repository != "" {
		// First try the new cache-first outdated check method
		if outdated, err := m.resolver.IsVersionOutdated(repository, current, latest); err == nil {
			return outdated
		}
		
		// Fall back to equivalence check if IsVersionOutdated fails
		equivalent, err := m.resolver.AreVersionsEquivalent(repository, current, latest)
		if err == nil && equivalent {
			return false // Versions are equivalent (same SHA)
		}
		// Continue with fallback logic if resolver fails or versions are not equivalent
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
			Recommendation:     "Use v4 for the latest features and bug fixes",
		},
		{
			Repository:         "actions/setup-node",
			LatestVersion:      "v4",
			MinimumVersion:     "v3",
			DeprecatedVersions: []string{"v1"},
		},
		{
			Repository:         "actions/setup-python",
			LatestVersion:      "v5",
			MinimumVersion:     "v4",
			DeprecatedVersions: []string{"v1", "v2"},
		},
		{
			Repository:         "actions/upload-artifact",
			LatestVersion:      "v4",
			MinimumVersion:     "v3",
			DeprecatedVersions: []string{"v1"},
		},
		{
			Repository:         "actions/download-artifact",
			LatestVersion:      "v4",
			MinimumVersion:     "v3",
			DeprecatedVersions: []string{"v1"},
		},
		{
			Repository:     "actions/cache",
			LatestVersion:  "v4",
			MinimumVersion: "v3",
		},
		{
			Repository:     "actions/setup-go",
			LatestVersion:  "v5",
			MinimumVersion: "v4",
		},
		{
			Repository:     "actions/setup-java",
			LatestVersion:  "v4",
			MinimumVersion: "v3",
		},
	}
}

// GetTransformationInfo returns information about schema transformations for a version upgrade
// This provides insight into what changes will be made to action inputs/outputs
func (m *Manager) GetTransformationInfo(repository, currentVersion, targetVersion string) (*patcher.VersionPatch, bool) {
	return m.patcher.GetPatchInfo(repository, currentVersion, targetVersion)
}

// PreviewTransformation shows what changes would be made to an action's with block
// without actually applying them
func (m *Manager) PreviewTransformation(repository, currentVersion, targetVersion string, withBlock interface{}) (*patcher.Patch, error) {
	return m.patcher.PreviewChanges(repository, currentVersion, targetVersion, withBlock)
}

// GetSupportedTransformations returns a list of actions that have transformation rules
func (m *Manager) GetSupportedTransformations() []string {
	return m.patcher.GetSupportedActions()
}
