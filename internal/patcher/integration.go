package patcher

import (
	"fmt"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
	"gopkg.in/yaml.v3"
	"regexp"
	"strings"
)

// WorkflowPatcher provides high-level workflow patching capabilities
type WorkflowPatcher struct {
	patcher *Patcher
}

// NewWorkflowPatcher creates a new workflow patcher
func NewWorkflowPatcher() *WorkflowPatcher {
	return &WorkflowPatcher{
		patcher: NewPatcher(),
	}
}

// parseActionRef parses an action reference string (e.g., "actions/checkout@v4")
// This is a local implementation since the workflow package function is not exported
func parseActionRef(uses string) *workflow.ActionReference {
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

	return &workflow.ActionReference{
		Repository: repository,
		Version:    version,
		IsReusable: false,
	}
}

// PatchStep applies patches to a workflow step when upgrading action versions
// This is the integration point with the existing workflow update logic
func (wp *WorkflowPatcher) PatchStep(step *workflow.Step, fromVersion, toVersion string) (*Patch, error) {
	if step.Uses == "" {
		return nil, fmt.Errorf("step does not use an action")
	}

	// Parse the action reference to get repository name
	actionRef := parseActionRef(step.Uses)
	if actionRef == nil {
		return nil, fmt.Errorf("failed to parse action reference: %s", step.Uses)
	}

	// Build patch for the with block
	patch, err := wp.patcher.BuildPatch(actionRef.Repository, fromVersion, toVersion, step.With)
	if err != nil {
		return nil, fmt.Errorf("failed to build patch: %w", err)
	}

	// Update the step's with block if patches were applied
	if patch.Applied {
		step.With = patch.UpdatedWith
	}

	return patch, nil
}

// PatchWorkflowContent applies patches to workflow YAML content during version updates
// This function can be used by the PR creator to apply schema changes when updating workflow files
func (wp *WorkflowPatcher) PatchWorkflowContent(content string, updates []ActionVersionUpdate) (string, []string, error) {
	// Parse the workflow
	var workflow workflow.Workflow
	if err := yaml.Unmarshal([]byte(content), &workflow); err != nil {
		return content, nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	var allChanges []string
	patchingApplied := false

	// Process each job
	for jobName, job := range workflow.Jobs {
		// Process job steps
		for stepIdx, step := range job.Steps {
			if step.Uses == "" {
				continue
			}

			// Check if this step needs patching
			for _, update := range updates {
				if wp.stepMatchesUpdate(&step, update) {
					// Apply patching
					patch, err := wp.PatchStep(&step, update.FromVersion, update.ToVersion)
					if err != nil {
						return content, allChanges, fmt.Errorf("failed to patch step in job %s, step %d: %w", jobName, stepIdx, err)
					}

					if patch.Applied {
						// Update the step in the workflow
						workflow.Jobs[jobName].Steps[stepIdx] = step
						patchingApplied = true

						// Add changes to the list based on the new patch structure
						for _, addition := range patch.Additions {
							changeDescription := fmt.Sprintf("Job '%s', Step %d: Added '%s' = '%v' (%s)", jobName, stepIdx+1, addition.Field, addition.Value, addition.Reason)
							allChanges = append(allChanges, changeDescription)
						}
						for _, removal := range patch.Removals {
							changeDescription := fmt.Sprintf("Job '%s', Step %d: Removed '%s' (%s)", jobName, stepIdx+1, removal.Field, removal.Reason)
							allChanges = append(allChanges, changeDescription)
						}
						for _, rename := range patch.Renames {
							changeDescription := fmt.Sprintf("Job '%s', Step %d: Renamed '%s' to '%s' (%s)", jobName, stepIdx+1, rename.OldField, rename.NewField, rename.Reason)
							allChanges = append(allChanges, changeDescription)
						}
						for _, modification := range patch.Modifications {
							changeDescription := fmt.Sprintf("Job '%s', Step %d: Modified '%s' from '%v' to '%v' (%s)", jobName, stepIdx+1, modification.Field, modification.OldValue, modification.NewValue, modification.Reason)
							allChanges = append(allChanges, changeDescription)
						}
					}
					break
				}
			}
		}
	}

	// If no patching was applied, return original content
	if !patchingApplied {
		return content, allChanges, nil
	}

	// Marshal the updated workflow back to YAML
	updatedData, err := yaml.Marshal(&workflow)
	if err != nil {
		return content, allChanges, fmt.Errorf("failed to marshal updated workflow: %w", err)
	}

	return string(updatedData), allChanges, nil
}

// ActionVersionUpdate represents an action version update that needs transformation
type ActionVersionUpdate struct {
	ActionRepo  string
	FromVersion string
	ToVersion   string
	FilePath    string // workflow file path for context
}

// stepMatchesUpdate checks if a step matches an action version update
func (wp *WorkflowPatcher) stepMatchesUpdate(step *workflow.Step, update ActionVersionUpdate) bool {
	if step.Uses == "" {
		return false
	}

	// Parse the action reference
	actionRef := parseActionRef(step.Uses)
	if actionRef == nil {
		return false
	}

	// Check if repository and current version match
	return actionRef.Repository == update.ActionRepo && actionRef.Version == update.FromVersion
}

// GetSupportedActions returns a list of actions that have patch rules defined
func (wp *WorkflowPatcher) GetSupportedActions() []string {
	rules := wp.patcher.GetPatchRules()
	actions := make([]string, 0, len(rules))

	for repo := range rules {
		actions = append(actions, repo)
	}

	return actions
}

// GetPatchInfo returns information about what patches would be applied for a version transition
func (wp *WorkflowPatcher) GetPatchInfo(repository, fromVersion, toVersion string) (*VersionPatch, bool) {
	rules := wp.patcher.GetPatchRules()
	rule, exists := rules[repository]
	if !exists {
		return nil, false
	}

	// Find the version patch
	for _, patch := range rule.VersionPatches {
		if patch.FromVersion == fromVersion && patch.ToVersion == toVersion {
			return &patch, true
		}
	}

	return nil, false
}

// PreviewChanges shows what changes would be made without actually applying them
func (wp *WorkflowPatcher) PreviewChanges(repository, fromVersion, toVersion string, withBlock interface{}) (*Patch, error) {
	// Create a copy of the with block for preview
	var withCopy interface{}
	if withBlock != nil {
		// Deep copy via YAML marshal/unmarshal
		data, err := yaml.Marshal(withBlock)
		if err != nil {
			return nil, fmt.Errorf("failed to copy with block for preview: %w", err)
		}

		if err := yaml.Unmarshal(data, &withCopy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal with block copy: %w", err)
		}
	}

	// Build patch for the copy
	return wp.patcher.BuildPatch(repository, fromVersion, toVersion, withCopy)
}

// HasPatch checks if a patch is available for the given repository and version transition
func (wp *WorkflowPatcher) HasPatch(repository, fromVersion, toVersion string) bool {
	return wp.patcher.HasPatch(repository, fromVersion, toVersion)
}
