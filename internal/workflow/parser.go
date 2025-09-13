package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Workflow represents a parsed GitHub Actions workflow
type Workflow struct {
	Name string                 `yaml:"name"`
	On   interface{}            `yaml:"on"`
	Jobs map[string]Job         `yaml:"jobs"`
}

// Job represents a job in a workflow
type Job struct {
	RunsOn interface{}           `yaml:"runs-on"`
	Uses   string               `yaml:"uses,omitempty"`
	Steps  []Step               `yaml:"steps,omitempty"`
}

// Step represents a step in a job
type Step struct {
	Name string      `yaml:"name,omitempty"`
	Uses string      `yaml:"uses,omitempty"`
	With interface{} `yaml:"with,omitempty"`
	Run  string      `yaml:"run,omitempty"`
}

// ActionReference represents a referenced action with version information
type ActionReference struct {
	Repository   string // e.g., "actions/checkout"
	Version      string // e.g., "v4", "main", commit SHA
	IsReusable   bool   // true if this is a reusable workflow call
	Context      string // where this action was found (job name, step name)
	FilePath     string // path to the workflow file
	RepoFullName string // full name of the repo containing this workflow
}

// ParseWorkflow parses a YAML workflow file and extracts action references
func ParseWorkflow(content, filePath, repoFullName string) ([]ActionReference, error) {
	var workflow Workflow
	if err := yaml.Unmarshal([]byte(content), &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}
	
	var references []ActionReference
	
	// Process each job
	for jobName, job := range workflow.Jobs {
		// Check if job uses a reusable workflow
		if job.Uses != "" {
			ref := parseActionRef(job.Uses, true)
			if ref != nil {
				ref.Context = fmt.Sprintf("job:%s", jobName)
				ref.FilePath = filePath
				ref.RepoFullName = repoFullName
				references = append(references, *ref)
			}
		}
		
		// Process job steps
		for stepIdx, step := range job.Steps {
			if step.Uses != "" {
				ref := parseActionRef(step.Uses, false)
				if ref != nil {
					stepName := step.Name
					if stepName == "" {
						stepName = fmt.Sprintf("step-%d", stepIdx+1)
					}
					ref.Context = fmt.Sprintf("job:%s/step:%s", jobName, stepName)
					ref.FilePath = filePath
					ref.RepoFullName = repoFullName
					references = append(references, *ref)
				}
			}
		}
	}
	
	return references, nil
}

// parseActionRef parses an action reference string (e.g., "actions/checkout@v4")
func parseActionRef(uses string, isReusable bool) *ActionReference {
	// Handle local actions (starting with "./")
	if strings.HasPrefix(uses, "./") {
		return nil // Skip local actions
	}
	
	// Handle Docker actions (starting with "docker://")
	if strings.HasPrefix(uses, "docker://") {
		return nil // Skip Docker actions
	}
	
	// Regular expression to parse action references
	// Supports: owner/repo@version, owner/repo/path@version
	re := regexp.MustCompile(`^([^/@]+/[^/@]+)(?:/[^@]*)?@(.+)$`)
	matches := re.FindStringSubmatch(uses)
	
	if len(matches) != 3 {
		return nil // Invalid format
	}
	
	repository := matches[1]
	version := matches[2]
	
	return &ActionReference{
		Repository: repository,
		Version:    version,
		IsReusable: isReusable,
	}
}

// IsVersionOutdated checks if a version string appears to be outdated
// This is a simple heuristic - in practice you'd want to check against
// actual release information from GitHub
func IsVersionOutdated(current, latest string) bool {
	// Simple heuristic: if current version is clearly older
	// This would need proper semver comparison in a real implementation
	if current == "main" || current == "master" {
		return false // Branch refs are considered up-to-date
	}
	
	// Check for major version differences (v1 vs v4, etc.)
	currentMajor := extractMajorVersion(current)
	latestMajor := extractMajorVersion(latest)
	
	if currentMajor != "" && latestMajor != "" {
		return currentMajor < latestMajor
	}
	
	return false
}

// extractMajorVersion extracts major version from version string
func extractMajorVersion(version string) string {
	re := regexp.MustCompile(`^v?(\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// SuggestVersionUpdate suggests an updated version for an action
func SuggestVersionUpdate(repository, currentVersion string) string {
	// This is a simplified implementation - in practice you'd want to
	// fetch the latest release from GitHub API
	
	// Common version updates for popular actions
	updates := map[string]map[string]string{
		"actions/checkout": {
			"v1": "v4",
			"v2": "v4", 
			"v3": "v4",
		},
		"actions/setup-node": {
			"v1": "v4",
			"v2": "v4",
			"v3": "v4",
		},
		"actions/setup-python": {
			"v1": "v5",
			"v2": "v5",
			"v3": "v5",
			"v4": "v5",
		},
		"actions/upload-artifact": {
			"v1": "v4",
			"v2": "v4",
			"v3": "v4",
		},
		"actions/download-artifact": {
			"v1": "v4",
			"v2": "v4",
			"v3": "v4",
		},
	}
	
	if repoUpdates, exists := updates[repository]; exists {
		if newVersion, exists := repoUpdates[currentVersion]; exists {
			return newVersion
		}
	}
	
	return currentVersion // No update suggestion
}