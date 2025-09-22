package patcher

import (
	"strings"
	"testing"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
)

// createWorkflowPatcherWithMigrationRules creates a WorkflowPatcher with migration rules for testing
func createWorkflowPatcherWithMigrationRules() *WorkflowPatcher {
	wp := NewWorkflowPatcher()

	// Add migration rules needed for tests
	migrationRule := ActionPatchRule{
		Repository: "legacy-org/deprecated-action",
		VersionPatches: []VersionPatch{
			{
				FromVersion:    "v1",
				ToVersion:      "v2",
				FromRepository: "legacy-org/deprecated-action",
				ToRepository:   "modern-org/recommended-action",
				Description:    "Repository migration from legacy-org to modern-org",
				Patches: []FieldPatch{
					{
						Operation: OperationRename,
						Field:     "old-param",
						NewField:  "new-param",
						Reason:    "Parameter renamed during migration",
					},
					{
						Operation: OperationAdd,
						Field:     "migrate-notice",
						Value:     "This action has been migrated to modern-org/recommended-action for better maintenance and support",
						Reason:    "Migration tracking notice",
					},
				},
			},
		},
	}
	wp.patcher.AddPatchRule(migrationRule)

	// Add organization migration rule
	orgMigrationRule := ActionPatchRule{
		Repository: "old-org/standard-action",
		VersionPatches: []VersionPatch{
			{
				FromVersion:    "v3",
				ToVersion:      "v3",
				FromRepository: "old-org/standard-action",
				ToRepository:   "new-org/standard-action",
				Description:    "Organization migration from old-org to new-org with same functionality",
				Patches:        []FieldPatch{},
			},
		},
	}
	wp.patcher.AddPatchRule(orgMigrationRule)

	return wp
}

// createWorkflowPatcherWithCheckoutRules creates a WorkflowPatcher with checkout rules for testing
func createWorkflowPatcherWithCheckoutRules() *WorkflowPatcher {
	wp := NewWorkflowPatcher()

	// Add checkout rules needed for tests
	checkoutRule := ActionPatchRule{
		Repository: "actions/checkout",
		VersionPatches: []VersionPatch{
			{
				FromVersion: "v1",
				ToVersion:   "v4",
				Description: "Major upgrade from v1 to v4 with token handling and fetch behavior changes",
				Patches: []FieldPatch{
					{
						Operation: OperationRemove,
						Field:     "token",
						Reason:    "In v4, the token parameter is no longer required as it automatically uses GITHUB_TOKEN with appropriate permissions",
					},
					{
						Operation: OperationAdd,
						Field:     "fetch-depth",
						Value:     1,
						Reason:    "v4 defaults to shallow clone (fetch-depth: 1) for better performance. Explicitly set if full history needed",
					},
				},
			},
		},
	}
	wp.patcher.AddPatchRule(checkoutRule)

	return wp
}

// TestWorkflowPatcherLocationMigration tests WorkflowPatcher with location changes
func TestWorkflowPatcherLocationMigration(t *testing.T) {
	wp := createWorkflowPatcherWithMigrationRules()

	// Create a test step using legacy action
	step := &workflow.Step{
		Name: "Test step",
		Uses: "legacy-org/deprecated-action@v1",
		With: map[string]interface{}{
			"old-param": "test-value",
		},
	}

	// Apply location migration patch
	patch, err := wp.PatchStepWithLocation(step, "v1", "v2", "modern-org/recommended-action")
	if err != nil {
		t.Fatalf("Failed to patch step with location: %v", err)
	}

	if !patch.Applied {
		t.Fatal("Expected patch to be applied for location migration")
	}

	// Verify the step's uses field was updated
	expectedUses := "modern-org/recommended-action@v2"
	if step.Uses != expectedUses {
		t.Errorf("Expected step.Uses to be '%s', got '%s'", expectedUses, step.Uses)
	}

	// Verify the with block was transformed
	withMap, ok := step.With.(map[string]interface{})
	if !ok {
		t.Fatal("Expected step.With to be a map")
	}

	if _, hasOldParam := withMap["old-param"]; hasOldParam {
		t.Error("Expected old-param to be renamed")
	}

	if newParam, hasNewParam := withMap["new-param"]; !hasNewParam {
		t.Error("Expected new-param to be added")
	} else if newParam != "test-value" {
		t.Errorf("Expected new-param to be 'test-value', got %v", newParam)
	}

	t.Logf("Location migration successful: %s -> %s", patch.FromRepository, patch.ToRepository)
}

// TestWorkflowPatcherOrganizationMigration tests migration with only organization change
func TestWorkflowPatcherOrganizationMigration(t *testing.T) {
	wp := NewWorkflowPatcher()

	// Create a test step using old organization
	step := &workflow.Step{
		Name: "Test step",
		Uses: "old-org/standard-action@v3",
		With: map[string]interface{}{
			"some-param": "value",
		},
	}

	// Apply organization migration patch
	patch, err := wp.PatchStepWithLocation(step, "v3", "v3", "new-org/standard-action")
	if err != nil {
		t.Fatalf("Failed to patch step with organization migration: %v", err)
	}

	// Verify the step's uses field was updated
	expectedUses := "new-org/standard-action@v3"
	if step.Uses != expectedUses {
		t.Errorf("Expected step.Uses to be '%s', got '%s'", expectedUses, step.Uses)
	}

	t.Logf("Organization migration successful: %s -> %s", patch.FromRepository, patch.ToRepository)
}

// TestWorkflowContentPatching tests patching of complete workflow content with location changes
func TestWorkflowContentPatching(t *testing.T) {
	wp := createWorkflowPatcherWithMigrationRules()

	// Sample workflow content with legacy action
	workflowContent := `
name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Legacy action step
        uses: legacy-org/deprecated-action@v1
        with:
          old-param: test-value
      - name: Standard step
        uses: actions/checkout@v4
`

	// Define updates including location migration
	updates := []ActionVersionUpdate{
		{
			ActionRepo:   "legacy-org/deprecated-action",
			FromVersion:  "v1",
			ToVersion:    "v2",
			ToActionRepo: "modern-org/recommended-action",
			FilePath:     "test.yml",
		},
	}

	// Apply patches
	updatedContent, changes, err := wp.PatchWorkflowContent(workflowContent, updates)
	if err != nil {
		t.Fatalf("Failed to patch workflow content: %v", err)
	}

	if len(changes) == 0 {
		t.Fatal("Expected changes to be made")
	}

	// Verify the action reference was updated
	if !strings.Contains(updatedContent, "modern-org/recommended-action@v2") {
		t.Error("Expected updated content to contain new action reference")
	}

	if strings.Contains(updatedContent, "legacy-org/deprecated-action@v1") {
		t.Error("Expected old action reference to be removed")
	}

	// Verify parameter changes are reflected in the changes list
	hasRename := false
	hasAddition := false
	for _, change := range changes {
		if strings.Contains(change, "Renamed 'old-param' to 'new-param'") {
			hasRename = true
		}
		if strings.Contains(change, "Added 'migrate-notice'") {
			hasAddition = true
		}
	}

	if !hasRename {
		t.Error("Expected rename change to be recorded")
	}
	if !hasAddition {
		t.Error("Expected addition change to be recorded")
	}

	t.Logf("Workflow content patching successful with %d changes", len(changes))
	for i, change := range changes {
		t.Logf("Change %d: %s", i+1, change)
	}
}

// TestPreservingActionPaths tests that action paths are preserved during migration
func TestPreservingActionPaths(t *testing.T) {
	wp := createWorkflowPatcherWithMigrationRules()

	// Create a test step with action path
	step := &workflow.Step{
		Name: "Test step with path",
		Uses: "legacy-org/deprecated-action/subpath@v1",
		With: map[string]interface{}{
			"old-param": "test-value",
		},
	}

	// Apply location migration patch
	patch, err := wp.PatchStepWithLocation(step, "v1", "v2", "modern-org/recommended-action")
	if err != nil {
		t.Fatalf("Failed to patch step with location: %v", err)
	}

	if !patch.Applied {
		t.Fatal("Expected patch to be applied for location migration")
	}

	// Verify the step's uses field preserves the path
	expectedUses := "modern-org/recommended-action/subpath@v2"
	if step.Uses != expectedUses {
		t.Errorf("Expected step.Uses to be '%s', got '%s'", expectedUses, step.Uses)
	}

	t.Logf("Action path preservation successful: %s", step.Uses)
}

// TestHasPatchWithLocationIntegration tests the integration layer HasPatchWithLocation
func TestHasPatchWithLocationIntegration(t *testing.T) {
	wp := createWorkflowPatcherWithMigrationRules()

	// Test that location migration patches are detected
	if !wp.HasPatchWithLocation("legacy-org/deprecated-action", "v1", "v2", "modern-org/recommended-action") {
		t.Error("Expected HasPatchWithLocation to return true for legacy-org migration")
	}

	if !wp.HasPatchWithLocation("old-org/standard-action", "v3", "v3", "new-org/standard-action") {
		t.Error("Expected HasPatchWithLocation to return true for org migration")
	}

	// Test that non-existent migrations are not detected
	if wp.HasPatchWithLocation("non-existent/action", "v1", "v2", "another/action") {
		t.Error("Expected HasPatchWithLocation to return false for non-existent migration")
	}

	t.Log("HasPatchWithLocation integration tests completed successfully")
}
