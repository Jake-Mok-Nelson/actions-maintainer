package transformer

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

// Operation represents a type of transformation operation
type Operation string

const (
	// OperationAdd adds a new field to the action's with block
	OperationAdd Operation = "add"
	// OperationRemove removes a field from the action's with block
	OperationRemove Operation = "remove"
	// OperationRename changes the name of a field in the action's with block
	OperationRename Operation = "rename"
	// OperationModify changes the value or structure of an existing field
	OperationModify Operation = "modify"
)

// FieldPatch represents a single field transformation
type FieldPatch struct {
	Operation Operation   `yaml:"operation"`
	Field     string      `yaml:"field"`
	NewField  string      `yaml:"new_field,omitempty"` // For rename operations
	Value     interface{} `yaml:"value,omitempty"`     // For add/modify operations
	Reason    string      `yaml:"reason"`              // Why this change is needed
}

// VersionPatch represents transformations needed for a version transition
type VersionPatch struct {
	FromVersion string       `yaml:"from_version"`
	ToVersion   string       `yaml:"to_version"`
	Patches     []FieldPatch `yaml:"patches"`
	Description string       `yaml:"description"` // High-level description of the migration
}

// ActionPatchRule defines transformation rules for a specific action
type ActionPatchRule struct {
	Repository     string         `yaml:"repository"`
	VersionPatches []VersionPatch `yaml:"version_patches"`
}

// PatchResult represents the result of applying patches
type PatchResult struct {
	Applied    bool     `json:"applied"`
	Changes    []string `json:"changes"`    // Description of changes made
	Warnings   []string `json:"warnings"`   // Non-critical issues
	OriginalWith interface{} `json:"original_with"` // Original with block for reference
	UpdatedWith  interface{} `json:"updated_with"`  // Updated with block
}

// Transformer handles applying patches to workflow steps
type Transformer struct {
	rules map[string]ActionPatchRule // repository -> rules
}

// NewTransformer creates a new transformer with default rules
func NewTransformer() *Transformer {
	transformer := &Transformer{
		rules: make(map[string]ActionPatchRule),
	}
	
	// Load default patch rules for common actions
	transformer.loadDefaultRules()
	
	return transformer
}

// ApplyPatches applies version-based patches to an action's with block
// This is the main entry point for transforming action configurations
func (t *Transformer) ApplyPatches(repository, fromVersion, toVersion string, withBlock interface{}) (*PatchResult, error) {
	result := &PatchResult{
		Applied:      false,
		Changes:      []string{},
		Warnings:     []string{},
		OriginalWith: withBlock,
		UpdatedWith:  withBlock,
	}
	
	// Find the patch rule for this repository
	rule, exists := t.rules[repository]
	if !exists {
		return result, nil // No patch rules defined for this action
	}
	
	// Find the appropriate version patch
	var versionPatch *VersionPatch
	for _, patch := range rule.VersionPatches {
		if patch.FromVersion == fromVersion && patch.ToVersion == toVersion {
			versionPatch = &patch
			break
		}
	}
	
	if versionPatch == nil {
		return result, nil // No specific patch for this version transition
	}
	
	// Apply the patches
	updatedWith, changes, warnings, err := t.applyVersionPatch(withBlock, *versionPatch)
	if err != nil {
		return nil, fmt.Errorf("failed to apply patches: %w", err)
	}
	
	result.Applied = len(changes) > 0
	result.Changes = changes
	result.Warnings = warnings
	result.UpdatedWith = updatedWith
	
	return result, nil
}

// applyVersionPatch applies a specific version patch to a with block
func (t *Transformer) applyVersionPatch(withBlock interface{}, patch VersionPatch) (interface{}, []string, []string, error) {
	var changes []string
	var warnings []string
	
	// Convert with block to a map for easier manipulation
	withMap, err := t.toMap(withBlock)
	if err != nil {
		return withBlock, changes, warnings, fmt.Errorf("failed to convert with block to map: %w", err)
	}
	
	// Apply each field patch
	for _, fieldPatch := range patch.Patches {
		change, warning, err := t.applyFieldPatch(withMap, fieldPatch)
		if err != nil {
			return withBlock, changes, warnings, fmt.Errorf("failed to apply field patch: %w", err)
		}
		
		if change != "" {
			changes = append(changes, change)
		}
		if warning != "" {
			warnings = append(warnings, warning)
		}
	}
	
	return withMap, changes, warnings, nil
}

// applyFieldPatch applies a single field patch to a with map
func (t *Transformer) applyFieldPatch(withMap map[string]interface{}, patch FieldPatch) (string, string, error) {
	switch patch.Operation {
	case OperationAdd:
		return t.applyAddPatch(withMap, patch)
	case OperationRemove:
		return t.applyRemovePatch(withMap, patch)
	case OperationRename:
		return t.applyRenamePatch(withMap, patch)
	case OperationModify:
		return t.applyModifyPatch(withMap, patch)
	default:
		return "", "", fmt.Errorf("unknown operation: %s", patch.Operation)
	}
}

// applyAddPatch adds a new field to the with block
func (t *Transformer) applyAddPatch(withMap map[string]interface{}, patch FieldPatch) (string, string, error) {
	if _, exists := withMap[patch.Field]; exists {
		warning := fmt.Sprintf("Field %s already exists, skipping add operation", patch.Field)
		return "", warning, nil
	}
	
	withMap[patch.Field] = patch.Value
	change := fmt.Sprintf("Added field '%s' with value '%v' (%s)", patch.Field, patch.Value, patch.Reason)
	return change, "", nil
}

// applyRemovePatch removes a field from the with block
func (t *Transformer) applyRemovePatch(withMap map[string]interface{}, patch FieldPatch) (string, string, error) {
	if _, exists := withMap[patch.Field]; !exists {
		warning := fmt.Sprintf("Field %s does not exist, skipping remove operation", patch.Field)
		return "", warning, nil
	}
	
	delete(withMap, patch.Field)
	change := fmt.Sprintf("Removed field '%s' (%s)", patch.Field, patch.Reason)
	return change, "", nil
}

// applyRenamePatch renames a field in the with block
func (t *Transformer) applyRenamePatch(withMap map[string]interface{}, patch FieldPatch) (string, string, error) {
	if patch.NewField == "" {
		return "", "", fmt.Errorf("new_field must be specified for rename operation")
	}
	
	value, exists := withMap[patch.Field]
	if !exists {
		warning := fmt.Sprintf("Field %s does not exist, skipping rename operation", patch.Field)
		return "", warning, nil
	}
	
	if _, newExists := withMap[patch.NewField]; newExists {
		warning := fmt.Sprintf("Target field %s already exists, skipping rename operation", patch.NewField)
		return "", warning, nil
	}
	
	withMap[patch.NewField] = value
	delete(withMap, patch.Field)
	change := fmt.Sprintf("Renamed field '%s' to '%s' (%s)", patch.Field, patch.NewField, patch.Reason)
	return change, "", nil
}

// applyModifyPatch modifies the value of an existing field
func (t *Transformer) applyModifyPatch(withMap map[string]interface{}, patch FieldPatch) (string, string, error) {
	if _, exists := withMap[patch.Field]; !exists {
		warning := fmt.Sprintf("Field %s does not exist, skipping modify operation", patch.Field)
		return "", warning, nil
	}
	
	oldValue := withMap[patch.Field]
	withMap[patch.Field] = patch.Value
	change := fmt.Sprintf("Modified field '%s' from '%v' to '%v' (%s)", patch.Field, oldValue, patch.Value, patch.Reason)
	return change, "", nil
}

// toMap converts various types to map[string]interface{} for easier manipulation
func (t *Transformer) toMap(input interface{}) (map[string]interface{}, error) {
	if input == nil {
		return make(map[string]interface{}), nil
	}
	
	switch v := input.(type) {
	case map[string]interface{}:
		return v, nil
	case map[interface{}]interface{}:
		// Convert map[interface{}]interface{} to map[string]interface{}
		result := make(map[string]interface{})
		for key, value := range v {
			strKey, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("non-string key found in map: %T", key)
			}
			result[strKey] = value
		}
		return result, nil
	default:
		// Try to marshal and unmarshal through YAML
		data, err := yaml.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal input to YAML: %w", err)
		}
		
		var result map[string]interface{}
		if err := yaml.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML to map: %w", err)
		}
		
		return result, nil
	}
}

// GetPatchRules returns all loaded patch rules (for testing/debugging)
func (t *Transformer) GetPatchRules() map[string]ActionPatchRule {
	return t.rules
}

// AddPatchRule adds a custom patch rule
func (t *Transformer) AddPatchRule(rule ActionPatchRule) {
	t.rules[rule.Repository] = rule
}