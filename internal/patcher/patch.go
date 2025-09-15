package patcher

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

// Operation represents a type of transformation operation (kept for internal use with rules)
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

// FieldPatch represents a single field transformation (internal for rules)
type FieldPatch struct {
	Operation Operation   `yaml:"operation"`
	Field     string      `yaml:"field"`
	NewField  string      `yaml:"new_field,omitempty"` // For rename operations
	Value     interface{} `yaml:"value,omitempty"`     // For add/modify operations
	Reason    string      `yaml:"reason"`              // Why this change is needed
}

// VersionPatch represents transformations needed for a version transition (internal for rules)
type VersionPatch struct {
	FromVersion    string       `yaml:"from_version"`
	ToVersion      string       `yaml:"to_version"`
	FromRepository string       `yaml:"from_repository,omitempty"` // Source repository if different from rule repository
	ToRepository   string       `yaml:"to_repository,omitempty"`   // Target repository if different from rule repository
	Patches        []FieldPatch `yaml:"patches"`
	Description    string       `yaml:"description"` // High-level description of the migration
}

// ActionPatchRule defines transformation rules for a specific action (internal for rules)
type ActionPatchRule struct {
	Repository     string         `yaml:"repository"`
	VersionPatches []VersionPatch `yaml:"version_patches"`
}

// FieldAddition represents a field that needs to be added
type FieldAddition struct {
	Field  string      `json:"field"`
	Value  interface{} `json:"value"`
	Reason string      `json:"reason"`
}

// FieldRemoval represents a field that needs to be removed
type FieldRemoval struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// FieldRename represents a field that needs to be renamed
type FieldRename struct {
	OldField string `json:"old_field"`
	NewField string `json:"new_field"`
	Reason   string `json:"reason"`
}

// FieldModification represents a field value that needs to be changed
type FieldModification struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value"`
	Reason   string      `json:"reason"`
}

// Patch represents all changes needed for an action upgrade
type Patch struct {
	Repository     string `json:"repository"`
	FromVersion    string `json:"from_version"`
	ToVersion      string `json:"to_version"`
	FromRepository string `json:"from_repository,omitempty"` // Source repository if different
	ToRepository   string `json:"to_repository,omitempty"`   // Target repository if different
	Description    string `json:"description"`

	// Clear categorization of changes
	Additions     []FieldAddition     `json:"additions,omitempty"`
	Removals      []FieldRemoval      `json:"removals,omitempty"`
	Renames       []FieldRename       `json:"renames,omitempty"`
	Modifications []FieldModification `json:"modifications,omitempty"`

	// Results after applying the patch
	Applied      bool        `json:"applied"`
	OriginalWith interface{} `json:"original_with"`
	UpdatedWith  interface{} `json:"updated_with"`
	Warnings     []string    `json:"warnings,omitempty"`
}

// Patcher handles building patches for workflow action upgrades
type Patcher struct {
	rules map[string]ActionPatchRule // repository -> rules
}

// NewPatcher creates a new patcher with default rules
func NewPatcher() *Patcher {
	patcher := &Patcher{
		rules: make(map[string]ActionPatchRule),
	}

	// Load default patch rules for common actions
	patcher.loadDefaultRules()

	return patcher
}

// BuildPatch builds a patch for upgrading an action from one version to another
// This is the main entry point that handles conditional logic and returns a complete patch
func (p *Patcher) BuildPatch(repository, fromVersion, toVersion string, withBlock interface{}) (*Patch, error) {
	return p.BuildPatchWithLocation(repository, fromVersion, toVersion, repository, withBlock)
}

// BuildPatchWithLocation builds a patch for upgrading an action with potential location change
// This method supports both version updates and repository location migrations
func (p *Patcher) BuildPatchWithLocation(fromRepository, fromVersion, toVersion, toRepository string, withBlock interface{}) (*Patch, error) {
	patch := &Patch{
		Repository:     fromRepository, // Keep original for compatibility
		FromRepository: fromRepository,
		ToRepository:   toRepository,
		FromVersion:    fromVersion,
		ToVersion:      toVersion,
		Applied:        false,
		OriginalWith:   withBlock,
		UpdatedWith:    withBlock,
		Additions:      []FieldAddition{},
		Removals:       []FieldRemoval{},
		Renames:        []FieldRename{},
		Modifications:  []FieldModification{},
		Warnings:       []string{},
	}

	// Try to find patch rules - check both source and target repositories
	var rule ActionPatchRule
	var exists bool
	
	// First, try the source repository
	rule, exists = p.rules[fromRepository]
	if !exists {
		// If not found, try the target repository (for rules defined on the new location)
		rule, exists = p.rules[toRepository]
	}
	
	if !exists {
		return patch, nil // No patch rules defined for either repository
	}

	// Find the appropriate version patch that matches our migration
	var versionPatch *VersionPatch
	for _, vp := range rule.VersionPatches {
		// Check for exact match with location change
		if vp.FromVersion == fromVersion && vp.ToVersion == toVersion {
			// If both repositories are specified in the patch, ensure they match
			if vp.FromRepository != "" && vp.ToRepository != "" {
				if vp.FromRepository == fromRepository && vp.ToRepository == toRepository {
					versionPatch = &vp
					break
				}
			} else {
				// If no repository migration specified, match any same-repo transition
				if fromRepository == toRepository {
					versionPatch = &vp
					break
				}
			}
		}
	}

	if versionPatch == nil {
		return patch, nil // No specific patch for this version/location transition
	}

	patch.Description = versionPatch.Description

	// Convert with block to a map for easier manipulation
	withMap, err := p.toMap(withBlock)
	if err != nil {
		return patch, fmt.Errorf("failed to convert with block to map: %w", err)
	}

	// Apply the patches and build the patch structure
	updatedWith, err := p.applyVersionPatch(withMap, *versionPatch, patch)
	if err != nil {
		return patch, fmt.Errorf("failed to apply patches: %w", err)
	}

	patch.Applied = len(patch.Additions) > 0 || len(patch.Removals) > 0 || len(patch.Renames) > 0 || len(patch.Modifications) > 0 || (fromRepository != toRepository)
	patch.UpdatedWith = updatedWith

	return patch, nil
}

// applyVersionPatch applies a specific version patch to a with block and populates the patch structure
func (p *Patcher) applyVersionPatch(withMap map[string]interface{}, versionPatch VersionPatch, patch *Patch) (interface{}, error) {
	// Apply each field patch and populate the appropriate patch structure
	for _, fieldPatch := range versionPatch.Patches {
		err := p.applyFieldPatch(withMap, fieldPatch, patch)
		if err != nil {
			return withMap, fmt.Errorf("failed to apply field patch: %w", err)
		}
	}

	return withMap, nil
}

// applyFieldPatch applies a single field patch to a with map and updates the patch structure
func (p *Patcher) applyFieldPatch(withMap map[string]interface{}, fieldPatch FieldPatch, patch *Patch) error {
	switch fieldPatch.Operation {
	case OperationAdd:
		return p.applyAddPatch(withMap, fieldPatch, patch)
	case OperationRemove:
		return p.applyRemovePatch(withMap, fieldPatch, patch)
	case OperationRename:
		return p.applyRenamePatch(withMap, fieldPatch, patch)
	case OperationModify:
		return p.applyModifyPatch(withMap, fieldPatch, patch)
	default:
		return fmt.Errorf("unknown operation: %s", fieldPatch.Operation)
	}
}

// applyAddPatch adds a new field to the with block
func (p *Patcher) applyAddPatch(withMap map[string]interface{}, fieldPatch FieldPatch, patch *Patch) error {
	if _, exists := withMap[fieldPatch.Field]; exists {
		warning := fmt.Sprintf("Field %s already exists, skipping add operation", fieldPatch.Field)
		patch.Warnings = append(patch.Warnings, warning)
		return nil
	}

	withMap[fieldPatch.Field] = fieldPatch.Value
	patch.Additions = append(patch.Additions, FieldAddition{
		Field:  fieldPatch.Field,
		Value:  fieldPatch.Value,
		Reason: fieldPatch.Reason,
	})
	return nil
}

// applyRemovePatch removes a field from the with block
func (p *Patcher) applyRemovePatch(withMap map[string]interface{}, fieldPatch FieldPatch, patch *Patch) error {
	if _, exists := withMap[fieldPatch.Field]; !exists {
		warning := fmt.Sprintf("Field %s does not exist, skipping remove operation", fieldPatch.Field)
		patch.Warnings = append(patch.Warnings, warning)
		return nil
	}

	delete(withMap, fieldPatch.Field)
	patch.Removals = append(patch.Removals, FieldRemoval{
		Field:  fieldPatch.Field,
		Reason: fieldPatch.Reason,
	})
	return nil
}

// applyRenamePatch renames a field in the with block
func (p *Patcher) applyRenamePatch(withMap map[string]interface{}, fieldPatch FieldPatch, patch *Patch) error {
	if fieldPatch.NewField == "" {
		return fmt.Errorf("new_field must be specified for rename operation")
	}

	value, exists := withMap[fieldPatch.Field]
	if !exists {
		warning := fmt.Sprintf("Field %s does not exist, skipping rename operation", fieldPatch.Field)
		patch.Warnings = append(patch.Warnings, warning)
		return nil
	}

	if _, newExists := withMap[fieldPatch.NewField]; newExists {
		warning := fmt.Sprintf("Target field %s already exists, skipping rename operation", fieldPatch.NewField)
		patch.Warnings = append(patch.Warnings, warning)
		return nil
	}

	withMap[fieldPatch.NewField] = value
	delete(withMap, fieldPatch.Field)
	patch.Renames = append(patch.Renames, FieldRename{
		OldField: fieldPatch.Field,
		NewField: fieldPatch.NewField,
		Reason:   fieldPatch.Reason,
	})
	return nil
}

// applyModifyPatch modifies the value of an existing field
func (p *Patcher) applyModifyPatch(withMap map[string]interface{}, fieldPatch FieldPatch, patch *Patch) error {
	oldValue, exists := withMap[fieldPatch.Field]
	if !exists {
		warning := fmt.Sprintf("Field %s does not exist, skipping modify operation", fieldPatch.Field)
		patch.Warnings = append(patch.Warnings, warning)
		return nil
	}

	withMap[fieldPatch.Field] = fieldPatch.Value
	patch.Modifications = append(patch.Modifications, FieldModification{
		Field:    fieldPatch.Field,
		OldValue: oldValue,
		NewValue: fieldPatch.Value,
		Reason:   fieldPatch.Reason,
	})
	return nil
}

// toMap converts various types to map[string]interface{} for easier manipulation
func (p *Patcher) toMap(input interface{}) (map[string]interface{}, error) {
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
func (p *Patcher) GetPatchRules() map[string]ActionPatchRule {
	return p.rules
}

// AddPatchRule adds a custom patch rule
func (p *Patcher) AddPatchRule(rule ActionPatchRule) {
	p.rules[rule.Repository] = rule
}

// HasPatch checks if a patch is available for the given repository and version transition
func (p *Patcher) HasPatch(repository, fromVersion, toVersion string) bool {
	return p.HasPatchWithLocation(repository, fromVersion, toVersion, repository)
}

// HasPatchWithLocation checks if a patch is available for the given repository and version transition with location change
func (p *Patcher) HasPatchWithLocation(fromRepository, fromVersion, toVersion, toRepository string) bool {
	// Try to find patch rules - check both source and target repositories
	rule, exists := p.rules[fromRepository]
	if !exists {
		rule, exists = p.rules[toRepository]
		if !exists {
			return false
		}
	}

	for _, patch := range rule.VersionPatches {
		if patch.FromVersion == fromVersion && patch.ToVersion == toVersion {
			// If both repositories are specified in the patch, ensure they match
			if patch.FromRepository != "" && patch.ToRepository != "" {
				if patch.FromRepository == fromRepository && patch.ToRepository == toRepository {
					return true
				}
			} else {
				// If no repository migration specified, match same-repo transitions
				if fromRepository == toRepository {
					return true
				}
			}
		}
	}
	return false
}
