package actions

import (
	"fmt"
	"log"
	"strings"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/output"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/patcher"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
)

// Config holds configuration options for the actions manager
type Config struct {
	Verbose bool
}

// Manager handles action version management and issue detection
type Manager struct {
	rules    []Rule
	patcher  *patcher.WorkflowPatcher
	resolver VersionResolver // Interface for version resolution
	verbose  bool
}

// VersionResolver interface for resolving version aliases
type VersionResolver interface {
	AreVersionsEquivalent(repository, version1, version2 string) (bool, error)
	IsVersionOutdated(repository, currentVersion, latestVersion string) (bool, error)
	ResolveRefWithCache(owner, repo, ref string) (string, error)
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
		verbose: false,
	}
}

// NewManagerWithResolver creates a new actions manager with a version resolver
func NewManagerWithResolver(resolver VersionResolver) *Manager {
	return &Manager{
		rules:    getDefaultRules(),
		patcher:  patcher.NewWorkflowPatcher(),
		resolver: resolver,
		verbose:  false,
	}
}

// NewManagerWithConfig creates a new actions manager with configuration
func NewManagerWithConfig(config *Config) *Manager {
	if config == nil {
		config = &Config{Verbose: false}
	}

	if config.Verbose {
		log.Printf("Actions manager initialized with verbose logging enabled")
	}

	return &Manager{
		rules:   getDefaultRules(),
		patcher: patcher.NewWorkflowPatcher(),
		verbose: config.Verbose,
	}
}

// NewManagerWithResolverAndConfig creates a new actions manager with a version resolver and configuration
func NewManagerWithResolverAndConfig(resolver VersionResolver, config *Config) *Manager {
	if config == nil {
		config = &Config{Verbose: false}
	}

	if config.Verbose {
		log.Printf("Actions manager initialized with version resolver and verbose logging enabled")
	}

	return &Manager{
		rules:    getDefaultRules(),
		patcher:  patcher.NewWorkflowPatcher(),
		resolver: resolver,
		verbose:  config.Verbose,
	}
}

// AnalyzeActions analyzes action references and identifies issues
func (m *Manager) AnalyzeActions(actions []workflow.ActionReference) []output.ActionIssue {
	if m.verbose {
		log.Printf("Rule evaluation: Starting analysis of %d action references", len(actions))
	}

	var issues []output.ActionIssue

	for i, action := range actions {
		if m.verbose {
			log.Printf("Rule evaluation: Analyzing action %d/%d - %s@%s (context: %s)", i+1, len(actions), action.Repository, action.Version, action.Context)
		}

		actionIssues := m.analyzeAction(action)
		issues = append(issues, actionIssues...)

		if m.verbose {
			log.Printf("Rule evaluation: Found %d issues for %s@%s", len(actionIssues), action.Repository, action.Version)
		}
	}

	if m.verbose {
		log.Printf("Rule evaluation: Completed analysis, found %d total issues", len(issues))
	}

	return issues
}

// analyzeAction analyzes a single action reference for issues
func (m *Manager) analyzeAction(action workflow.ActionReference) []output.ActionIssue {
	var issues []output.ActionIssue

	rule := m.findRule(action.Repository)
	if rule == nil {
		if m.verbose {
			log.Printf("Rule evaluation: No rules found for repository %s, skipping analysis", action.Repository)
		}
		return issues // No rules for this action
	}

	if m.verbose {
		log.Printf("Rule evaluation: Found rule for %s - latest: %s, minimum: %s, deprecated: %v", action.Repository, rule.LatestVersion, rule.MinimumVersion, rule.DeprecatedVersions)
	}

	// Check for outdated versions
	if m.isOutdatedForRepository(action.Repository, action.Version, rule.LatestVersion) {
		if m.verbose {
			log.Printf("Rule evaluation: Version %s is outdated for %s (latest: %s)", action.Version, action.Repository, rule.LatestVersion)
		}

		// Suggest version in the same format as current version (like for like)
		suggestedVersion := m.suggestLikeForLikeVersion(action.Repository, action.Version, rule.LatestVersion)

		if m.verbose {
			log.Printf("Rule evaluation: Suggested version for %s: %s -> %s", action.Repository, action.Version, suggestedVersion)
		}

		issue := output.ActionIssue{
			Repository:       action.Repository,
			CurrentVersion:   action.Version,
			SuggestedVersion: suggestedVersion,
			IssueType:        "outdated",
			Severity:         m.determineSeverity(action.Version, rule),
			Description:      fmt.Sprintf("Action %s is using version %s, latest is %s", action.Repository, action.Version, rule.LatestVersion),
			Context:          action.Context,
			FilePath:         action.FilePath,
		}

		if m.verbose {
			log.Printf("Rule evaluation: Created outdated issue for %s with severity %s", action.Repository, issue.Severity)
		}

		// Check if there are schema transformations for this version upgrade
		if patchInfo, hasPatches := m.GetTransformationInfo(action.Repository, action.Version, rule.LatestVersion); hasPatches {
			issue.HasTransformations = true
			issue.SchemaChanges = []string{patchInfo.Description}

			if m.verbose {
				log.Printf("Rule evaluation: Found schema transformations for %s (%s -> %s)", action.Repository, action.Version, rule.LatestVersion)
			}

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
			if m.verbose {
				log.Printf("Rule evaluation: Version %s is deprecated for %s", action.Version, action.Repository)
			}

			// Suggest version in the same format as current version (like for like)
			suggestedVersion := m.suggestLikeForLikeVersion(action.Repository, action.Version, rule.LatestVersion)

			issue := output.ActionIssue{
				Repository:       action.Repository,
				CurrentVersion:   action.Version,
				SuggestedVersion: suggestedVersion,
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

// extractMajorVersion extracts the major version number from a version string
func extractMajorVersion(version string) string {
	// Unconditionally trim leading 'v' to normalize version strings
	version = strings.TrimPrefix(version, "v")

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

// VersionFormat represents the format type of a version reference
type VersionFormat int

const (
	VersionFormatTag VersionFormat = iota
	VersionFormatSHA
	VersionFormatBranch
)

// detectVersionFormat determines the format of a version string
func (m *Manager) detectVersionFormat(version string) VersionFormat {
	// Branch references
	if version == "main" || version == "master" {
		return VersionFormatBranch
	}

	// SHA format (7-40 hex characters, not starting with v)
	if len(version) >= 7 && len(version) <= 41 && !strings.HasPrefix(version, "v") && isHexString(version) {
		return VersionFormatSHA
	}

	// Tag format (starts with v and has dots or is just vN, or anything else)
	return VersionFormatTag
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	for _, char := range s {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

// suggestLikeForLikeVersion suggests a version in the same format as the current version
func (m *Manager) suggestLikeForLikeVersion(repository, currentVersion, latestTagVersion string) string {
	format := m.detectVersionFormat(currentVersion)

	switch format {
	case VersionFormatBranch:
		// For branch references, suggest the latest tag as-is since branches are not outdated
		return latestTagVersion
	case VersionFormatTag:
		// For tag references, suggest the latest tag as-is
		return latestTagVersion
	case VersionFormatSHA:
		// For SHA references, resolve the latest tag to its SHA
		if m.resolver != nil && repository != "" {
			parts := strings.Split(repository, "/")
			if len(parts) == 2 {
				owner, repo := parts[0], parts[1]
				if sha, err := m.resolver.ResolveRefWithCache(owner, repo, latestTagVersion); err == nil {
					return sha
				}
			}
		}
		// Fallback to tag if resolution fails
		return latestTagVersion
	default:
		return latestTagVersion
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
