package transformer

import (
	"fmt"
	"regexp"
	"strings"
	"gopkg.in/yaml.v3"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
)

// WorkflowTransformer provides high-level workflow transformation capabilities
type WorkflowTransformer struct {
	transformer *Transformer
}

// NewWorkflowTransformer creates a new workflow transformer
func NewWorkflowTransformer() *WorkflowTransformer {
	return &WorkflowTransformer{
		transformer: NewTransformer(),
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

// TransformStep applies patches to a workflow step when upgrading action versions
// This is the integration point with the existing workflow update logic
func (wt *WorkflowTransformer) TransformStep(step *workflow.Step, fromVersion, toVersion string) (*PatchResult, error) {
	if step.Uses == "" {
		return nil, fmt.Errorf("step does not use an action")
	}
	
	// Parse the action reference to get repository name
	actionRef := parseActionRef(step.Uses)
	if actionRef == nil {
		return nil, fmt.Errorf("failed to parse action reference: %s", step.Uses)
	}
	
	// Apply patches to the with block
	result, err := wt.transformer.ApplyPatches(actionRef.Repository, fromVersion, toVersion, step.With)
	if err != nil {
		return nil, fmt.Errorf("failed to apply patches: %w", err)
	}
	
	// Update the step's with block if patches were applied
	if result.Applied {
		step.With = result.UpdatedWith
	}
	
	return result, nil
}

// TransformWorkflowContent applies patches to workflow YAML content during version updates
// This function can be used by the PR creator to apply schema changes when updating workflow files
func (wt *WorkflowTransformer) TransformWorkflowContent(content string, updates []ActionVersionUpdate) (string, []string, error) {
	// Parse the workflow
	var workflow workflow.Workflow
	if err := yaml.Unmarshal([]byte(content), &workflow); err != nil {
		return content, nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}
	
	var allChanges []string
	transformationApplied := false
	
	// Process each job
	for jobName, job := range workflow.Jobs {
		// Process job steps
		for stepIdx, step := range job.Steps {
			if step.Uses == "" {
				continue
			}
			
			// Check if this step needs transformation
			for _, update := range updates {
				if wt.stepMatchesUpdate(&step, update) {
					// Apply transformation
					result, err := wt.TransformStep(&step, update.FromVersion, update.ToVersion)
					if err != nil {
						return content, allChanges, fmt.Errorf("failed to transform step in job %s, step %d: %w", jobName, stepIdx, err)
					}
					
					if result.Applied {
						// Update the step in the workflow
						workflow.Jobs[jobName].Steps[stepIdx] = step
						transformationApplied = true
						
						// Add changes to the list
						for _, change := range result.Changes {
							changeDescription := fmt.Sprintf("Job '%s', Step %d: %s", jobName, stepIdx+1, change)
							allChanges = append(allChanges, changeDescription)
						}
					}
					break
				}
			}
		}
	}
	
	// If no transformations were applied, return original content
	if !transformationApplied {
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
	ActionRepo    string
	FromVersion   string
	ToVersion     string
	FilePath      string // workflow file path for context
}

// stepMatchesUpdate checks if a step matches an action version update
func (wt *WorkflowTransformer) stepMatchesUpdate(step *workflow.Step, update ActionVersionUpdate) bool {
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
func (wt *WorkflowTransformer) GetSupportedActions() []string {
	rules := wt.transformer.GetPatchRules()
	actions := make([]string, 0, len(rules))
	
	for repo := range rules {
		actions = append(actions, repo)
	}
	
	return actions
}

// GetPatchInfo returns information about what patches would be applied for a version transition
func (wt *WorkflowTransformer) GetPatchInfo(repository, fromVersion, toVersion string) (*VersionPatch, bool) {
	rules := wt.transformer.GetPatchRules()
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
func (wt *WorkflowTransformer) PreviewChanges(repository, fromVersion, toVersion string, withBlock interface{}) (*PatchResult, error) {
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
	
	// Apply patches to the copy
	return wt.transformer.ApplyPatches(repository, fromVersion, toVersion, withCopy)
}