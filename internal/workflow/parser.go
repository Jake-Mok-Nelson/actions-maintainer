package workflow

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds configuration options for the workflow parser
type Config struct {
	Verbose      bool
	WorkflowOnly bool // Only target reusable workflows, exclude regular actions
}

// Workflow represents a parsed GitHub Actions workflow
type Workflow struct {
	Name string         `yaml:"name"`
	On   interface{}    `yaml:"on"`
	Jobs map[string]Job `yaml:"jobs"`
}

// Job represents a job in a workflow
type Job struct {
	RunsOn interface{} `yaml:"runs-on"`
	Uses   string      `yaml:"uses,omitempty"`
	Steps  []Step      `yaml:"steps,omitempty"`
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
	WorkflowPath string // e.g., ".github/workflows/ci.yml" (for reusable workflows)
	IsReusable   bool   // true if this is a reusable workflow call
	Context      string // where this action was found (job name, step name)
	FilePath     string // path to the workflow file
	RepoFullName string // full name of the repo containing this workflow
}

// ParseWorkflow parses a YAML workflow file and extracts action references
func ParseWorkflow(content, filePath, repoFullName string) ([]ActionReference, error) {
	return ParseWorkflowWithResolver(content, filePath, repoFullName, nil)
}

// ParseWorkflowWithConfig parses a YAML workflow file with configuration
func ParseWorkflowWithConfig(content, filePath, repoFullName string, config *Config) ([]ActionReference, error) {
	return ParseWorkflowWithResolverAndConfig(content, filePath, repoFullName, nil, config)
}

// ParseWorkflowWithResolver parses a YAML workflow file and extracts action references
// with optional version resolution
func ParseWorkflowWithResolver(content, filePath, repoFullName string, resolver *VersionResolver) ([]ActionReference, error) {
	return ParseWorkflowWithResolverAndConfig(content, filePath, repoFullName, resolver, &Config{Verbose: false})
}

// ParseWorkflowWithResolverAndConfig parses a YAML workflow file and extracts action references
// with optional version resolution and configuration
func ParseWorkflowWithResolverAndConfig(content, filePath, repoFullName string, resolver *VersionResolver, config *Config) ([]ActionReference, error) {
	if config == nil {
		config = &Config{Verbose: false}
	}

	if config.Verbose {
		log.Printf("Workflow parsing: Starting to parse %s in repository %s", filePath, repoFullName)
	}

	var workflow Workflow
	if err := yaml.Unmarshal([]byte(content), &workflow); err != nil {
		if config.Verbose {
			log.Printf("Workflow parsing: Failed to parse YAML in %s - %v", filePath, err)
		}
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	if config.Verbose {
		log.Printf("Workflow parsing: Successfully parsed YAML for %s, found %d jobs", filePath, len(workflow.Jobs))
	}

	var references []ActionReference

	// Process each job
	for jobName, job := range workflow.Jobs {
		if config.Verbose {
			log.Printf("Workflow parsing: Processing job '%s' in %s", jobName, filePath)
		}

		// Check if job uses a reusable workflow
		if job.Uses != "" {
			if config.Verbose {
				log.Printf("Workflow parsing: Found reusable workflow reference '%s' in job '%s'", job.Uses, jobName)
			}
			ref := parseActionRef(job.Uses, true)
			if ref != nil {
				ref.Context = fmt.Sprintf("job:%s", jobName)
				ref.FilePath = filePath
				ref.RepoFullName = repoFullName
				references = append(references, *ref)
				if config.Verbose {
					log.Printf("Workflow parsing: Extracted reusable workflow reference - repository: %s, version: %s", ref.Repository, ref.Version)
				}
			}
		}

		// Process job steps (skip if workflow-only mode is enabled)
		if !config.WorkflowOnly {
			for stepIdx, step := range job.Steps {
				if step.Uses != "" {
					if config.Verbose {
						log.Printf("Workflow parsing: Found action reference '%s' in job '%s', step %d", step.Uses, jobName, stepIdx+1)
					}
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
						if config.Verbose {
							log.Printf("Workflow parsing: Extracted action reference - repository: %s, version: %s, context: %s", ref.Repository, ref.Version, ref.Context)
						}
					}
				}
			}
		} else if config.Verbose {
			log.Printf("Workflow parsing: Skipping step processing for job '%s' due to workflow-only mode", jobName)
		}
	}

	if config.Verbose {
		log.Printf("Workflow parsing: Completed parsing %s, extracted %d action references", filePath, len(references))
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
	re := regexp.MustCompile(`^([^/@]+/[^/@]+)(?:/([^@]*))?@(.+)$`)
	matches := re.FindStringSubmatch(uses)

	if len(matches) != 4 {
		return nil // Invalid format
	}

	repository := matches[1]
	workflowPath := matches[2] // Will be empty for regular actions
	version := matches[3]

	return &ActionReference{
		Repository:   repository,
		Version:      version,
		WorkflowPath: workflowPath,
		IsReusable:   isReusable,
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
