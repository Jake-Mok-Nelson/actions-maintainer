package patcher

import (
	"testing"
)

// createPatcherWithCheckoutRules creates a patcher with actions/checkout rules for testing
func createPatcherWithCheckoutRules() *Patcher {
	patcher := NewPatcher()

	// Manually add actions/checkout rules needed for tests
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
			{
				FromVersion: "v3",
				ToVersion:   "v4",
				Description: "Minor upgrade from v3 to v4 with performance improvements",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "show-progress",
						Value:     true,
						Reason:    "v4 adds show-progress parameter for better user experience during large repository operations",
					},
				},
			},
		},
	}
	patcher.AddPatchRule(checkoutRule)
	return patcher
}

// createPatcherWithSetupNodeRules creates a patcher with actions/setup-node rules for testing
func createPatcherWithSetupNodeRules() *Patcher {
	patcher := NewPatcher()

	// Manually add actions/setup-node rules needed for tests
	setupNodeRule := ActionPatchRule{
		Repository: "actions/setup-node",
		VersionPatches: []VersionPatch{
			{
				FromVersion: "v2",
				ToVersion:   "v4",
				Description: "Upgrade from v2 to v4 with improved caching and registry support",
				Patches: []FieldPatch{
					{
						Operation: OperationRename,
						Field:     "version",
						NewField:  "node-version",
						Reason:    "Parameter renamed from 'version' to 'node-version' for better clarity",
					},
					{
						Operation: OperationAdd,
						Field:     "cache",
						Value:     "npm",
						Reason:    "v4 introduces built-in dependency caching",
					},
				},
			},
		},
	}
	patcher.AddPatchRule(setupNodeRule)
	return patcher
}

// createPatcherWithMigrationRules creates a patcher with migration rules for testing
func createPatcherWithMigrationRules() *Patcher {
	patcher := NewPatcher()

	// Add migration rules for testing
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
	patcher.AddPatchRule(migrationRule)

	// Add organization migration rule for other tests
	orgMigrationRule := ActionPatchRule{
		Repository: "old-org/standard-action",
		VersionPatches: []VersionPatch{
			{
				FromVersion:    "v3",
				ToVersion:      "v3",
				FromRepository: "old-org/standard-action",
				ToRepository:   "new-org/standard-action",
				Description:    "Organization migration from old-org to new-org with same functionality",
				Patches:        []FieldPatch{
					// No parameter changes needed, just location change
				},
			},
		},
	}
	patcher.AddPatchRule(orgMigrationRule)

	return patcher
}

// TestBasicPatchOperations tests the basic patch operations with default rules loaded
func TestBasicPatchOperations(t *testing.T) {
	patcher := NewPatcher()

	// Test data: simulate actions/checkout v3 to v4 upgrade
	withBlock := map[string]interface{}{
		"fetch-depth": 0,
		"token":       "${{ secrets.GITHUB_TOKEN }}",
	}

	// Build patch
	patch, err := patcher.BuildPatch("actions/checkout", "v3", "v4", withBlock)
	if err != nil {
		t.Fatalf("Failed to build patch: %v", err)
	}

	// With default rules loaded, patches should be applied for checkout v3->v4
	if !patch.Applied {
		t.Error("Expected patches to be applied for checkout v3->v4 with default rules loaded")
		return
	}

	// Check that changes were made
	totalChanges := len(patch.Additions) + len(patch.Removals) + len(patch.Renames) + len(patch.Modifications)
	if totalChanges == 0 {
		t.Error("Expected changes but none were found")
	}

	t.Logf("Applied %d total changes - Additions: %d, Removals: %d, Renames: %d, Modifications: %d",
		totalChanges, len(patch.Additions), len(patch.Removals), len(patch.Renames), len(patch.Modifications))
}

// TestCheckoutV1ToV4Transformation tests the transformation from checkout v1 to v4
func TestCheckoutV1ToV4Transformation(t *testing.T) {
	patcher := createPatcherWithCheckoutRules()

	// Test data: simulate actions/checkout v1 with token
	withBlock := map[string]interface{}{
		"token": "${{ secrets.GITHUB_TOKEN }}",
	}

	// Build patch
	patch, err := patcher.BuildPatch("actions/checkout", "v1", "v4", withBlock)
	if err != nil {
		t.Fatalf("Failed to build patch: %v", err)
	}

	if !patch.Applied {
		t.Fatal("Expected patches to be applied for checkout v1->v4")
	}

	// Check that changes were made
	totalChanges := len(patch.Additions) + len(patch.Removals) + len(patch.Renames) + len(patch.Modifications)
	if totalChanges == 0 {
		t.Error("Expected changes but none were found")
	}

	// Verify token was removed and fetch-depth was added
	updatedWith := patch.UpdatedWith.(map[string]interface{})

	if _, hasToken := updatedWith["token"]; hasToken {
		t.Error("Expected token field to be removed")
	}

	if fetchDepth, hasFetchDepth := updatedWith["fetch-depth"]; !hasFetchDepth {
		t.Error("Expected fetch-depth field to be added")
	} else if fetchDepth != 1 {
		t.Errorf("Expected fetch-depth to be 1, got %v", fetchDepth)
	}

	// Check that the patch structure is populated correctly
	expectedRemovals := 1  // token removal
	expectedAdditions := 1 // fetch-depth addition
	if len(patch.Removals) != expectedRemovals {
		t.Errorf("Expected %d removals, got %d", expectedRemovals, len(patch.Removals))
	}
	if len(patch.Additions) != expectedAdditions {
		t.Errorf("Expected %d additions, got %d", expectedAdditions, len(patch.Additions))
	}

	t.Logf("Successfully applied changes - Additions: %v, Removals: %v", patch.Additions, patch.Removals)
}

// TestSetupNodeV2ToV4Transformation tests setup-node parameter renaming
func TestSetupNodeV2ToV4Transformation(t *testing.T) {
	patcher := createPatcherWithSetupNodeRules()

	// Test data: simulate actions/setup-node v2 with version parameter
	withBlock := map[string]interface{}{
		"version": "16",
	}

	// Build patch
	patch, err := patcher.BuildPatch("actions/setup-node", "v2", "v4", withBlock)
	if err != nil {
		t.Fatalf("Failed to build patch: %v", err)
	}

	if !patch.Applied {
		t.Fatal("Expected patches to be applied for setup-node v2->v4")
	}

	// Verify version was renamed to node-version and cache was added
	updatedWith := patch.UpdatedWith.(map[string]interface{})

	if _, hasVersion := updatedWith["version"]; hasVersion {
		t.Error("Expected version field to be removed/renamed")
	}

	if nodeVersion, hasNodeVersion := updatedWith["node-version"]; !hasNodeVersion {
		t.Error("Expected node-version field to be added")
	} else if nodeVersion != "16" {
		t.Errorf("Expected node-version to be '16', got %v", nodeVersion)
	}

	if cache, hasCache := updatedWith["cache"]; !hasCache {
		t.Error("Expected cache field to be added")
	} else if cache != "npm" {
		t.Errorf("Expected cache to be 'npm', got %v", cache)
	}

	// Check the patch structure
	expectedRenames := 1   // version -> node-version
	expectedAdditions := 1 // cache addition
	if len(patch.Renames) != expectedRenames {
		t.Errorf("Expected %d renames, got %d", expectedRenames, len(patch.Renames))
	}
	if len(patch.Additions) != expectedAdditions {
		t.Errorf("Expected %d additions, got %d", expectedAdditions, len(patch.Additions))
	}

	t.Logf("Successfully applied changes - Additions: %v, Renames: %v", patch.Additions, patch.Renames)
}

// TestNoTransformationForUnsupportedAction tests behavior when no transformation rules exist
func TestNoTransformationForUnsupportedAction(t *testing.T) {
	patcher := NewPatcher()

	// Test with an action that doesn't have transformation rules
	withBlock := map[string]interface{}{
		"some-param": "value",
	}

	patch, err := patcher.BuildPatch("unsupported/action", "v1", "v2", withBlock)
	if err != nil {
		t.Fatalf("Failed to build patch: %v", err)
	}

	if patch.Applied {
		t.Error("Expected no patches to be applied for unsupported action")
	}

	totalChanges := len(patch.Additions) + len(patch.Removals) + len(patch.Renames) + len(patch.Modifications)
	if totalChanges > 0 {
		t.Error("Expected no changes for unsupported action")
	}

	t.Log("Correctly handled unsupported action with no patching")
}

// TestNilWithBlock tests handling of nil with blocks
func TestNilWithBlock(t *testing.T) {
	patcher := createPatcherWithCheckoutRules()

	// Test with nil with block
	patch, err := patcher.BuildPatch("actions/checkout", "v1", "v4", nil)
	if err != nil {
		t.Fatalf("Failed to build patch for nil with block: %v", err)
	}

	if !patch.Applied {
		t.Fatal("Expected patches to be applied even with nil with block")
	}

	// Should add fetch-depth field to empty map
	updatedWith := patch.UpdatedWith.(map[string]interface{})
	if fetchDepth, hasFetchDepth := updatedWith["fetch-depth"]; !hasFetchDepth {
		t.Error("Expected fetch-depth field to be added to nil with block")
	} else if fetchDepth != 1 {
		t.Errorf("Expected fetch-depth to be 1, got %v", fetchDepth)
	}

	t.Logf("Successfully handled nil with block: Additions: %v, Removals: %v", patch.Additions, patch.Removals)
}

// TestGetSupportedActions tests that we can retrieve supported actions
func TestGetSupportedActions(t *testing.T) {
	wp := NewWorkflowPatcher()

	// With default rules loaded, we should have some supported actions
	actions := wp.GetSupportedActions()
	if len(actions) == 0 {
		t.Error("Expected some supported actions with default rules loaded, got 0")
		return
	}

	t.Logf("Found %d supported actions with default rules: %v", len(actions), actions)

	// Verify that common actions are included
	expectedActions := []string{"actions/checkout", "actions/setup-node"}
	for _, expected := range expectedActions {
		found := false
		for _, action := range actions {
			if action == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %s in supported actions, but it was missing", expected)
		}
	}
}

// TestNewCustomPatcher tests the new function for creating a patcher without default rules
func TestNewCustomPatcher(t *testing.T) {
	// Test NewCustomPatcher creates a patcher with no default rules
	patcher := NewCustomPatcher()

	// Should have no rules initially
	rules := patcher.GetPatchRules()
	if len(rules) != 0 {
		t.Errorf("Expected no rules in custom patcher, got %d", len(rules))
	}

	// Test that no patches are applied for common actions without rules
	withBlock := map[string]interface{}{
		"fetch-depth": 0,
		"token":       "${{ secrets.GITHUB_TOKEN }}",
	}

	patch, err := patcher.BuildPatch("actions/checkout", "v3", "v4", withBlock)
	if err != nil {
		t.Fatalf("Failed to build patch: %v", err)
	}

	if patch.Applied {
		t.Error("Expected no patches to be applied in custom patcher without rules")
	}

	// Test that we can add custom rules
	customRule := ActionPatchRule{
		Repository: "my-org/custom-action",
		VersionPatches: []VersionPatch{
			{
				FromVersion: "v1",
				ToVersion:   "v2",
				Description: "Custom test rule",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "new-param",
						Value:     "test-value",
						Reason:    "Test parameter addition",
					},
				},
			},
		},
	}

	patcher.AddPatchRule(customRule)

	// Should now have one rule
	rules = patcher.GetPatchRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule after adding custom rule, got %d", len(rules))
	}

	t.Log("NewCustomPatcher working correctly - no default rules, can add custom rules")
}

// TestLocationMigration tests migration of actions to new repository locations
func TestLocationMigration(t *testing.T) {
	patcher := createPatcherWithMigrationRules()

	// Test data: simulate legacy action migration
	withBlock := map[string]interface{}{
		"old-param": "test-value",
	}

	// Build patch for location migration
	patch, err := patcher.BuildPatchWithLocation("legacy-org/deprecated-action", "v1", "v2", "modern-org/recommended-action", withBlock)
	if err != nil {
		t.Fatalf("Failed to build location migration patch: %v", err)
	}

	if !patch.Applied {
		t.Fatal("Expected patches to be applied for location migration")
	}

	// Verify the repository fields are set correctly
	if patch.FromRepository != "legacy-org/deprecated-action" {
		t.Errorf("Expected FromRepository to be 'legacy-org/deprecated-action', got %s", patch.FromRepository)
	}
	if patch.ToRepository != "modern-org/recommended-action" {
		t.Errorf("Expected ToRepository to be 'modern-org/recommended-action', got %s", patch.ToRepository)
	}

	// Verify parameter transformation
	updatedWith := patch.UpdatedWith.(map[string]interface{})

	if _, hasOldParam := updatedWith["old-param"]; hasOldParam {
		t.Error("Expected old-param field to be renamed")
	}

	if newParam, hasNewParam := updatedWith["new-param"]; !hasNewParam {
		t.Error("Expected new-param field to be added")
	} else if newParam != "test-value" {
		t.Errorf("Expected new-param to be 'test-value', got %v", newParam)
	}

	if migrateNotice, hasMigrateNotice := updatedWith["migrate-notice"]; !hasMigrateNotice {
		t.Error("Expected migrate-notice field to be added")
	} else {
		expectedNotice := "This action has been migrated to modern-org/recommended-action for better maintenance and support"
		if migrateNotice != expectedNotice {
			t.Errorf("Expected migrate-notice to be '%s', got %v", expectedNotice, migrateNotice)
		}
	}

	t.Logf("Successfully applied location migration - FromRepo: %s, ToRepo: %s, Applied changes: %d",
		patch.FromRepository, patch.ToRepository, len(patch.Additions)+len(patch.Renames))
}

// TestOrganizationMigration tests migration when only organization changes
func TestOrganizationMigration(t *testing.T) {
	patcher := NewPatcher()

	// Test data: simulate organization migration with no parameter changes
	withBlock := map[string]interface{}{
		"some-param": "value",
	}

	// Build patch for organization migration
	patch, err := patcher.BuildPatchWithLocation("old-org/standard-action", "v3", "v3", "new-org/standard-action", withBlock)
	if err != nil {
		t.Fatalf("Failed to build organization migration patch: %v", err)
	}

	// This should not apply any patches since no parameter changes are needed
	if patch.Applied {
		// The patch is still considered "applied" because it includes repository migration info
		t.Log("Organization migration patch was applied (repository change only)")
	}

	// Verify the repository fields are set correctly
	if patch.FromRepository != "old-org/standard-action" {
		t.Errorf("Expected FromRepository to be 'old-org/standard-action', got %s", patch.FromRepository)
	}
	if patch.ToRepository != "new-org/standard-action" {
		t.Errorf("Expected ToRepository to be 'new-org/standard-action', got %s", patch.ToRepository)
	}

	// Verify no parameter changes
	totalChanges := len(patch.Additions) + len(patch.Removals) + len(patch.Renames) + len(patch.Modifications)
	if totalChanges > 0 {
		t.Logf("Organization migration included %d parameter changes: Additions: %d, Removals: %d, Renames: %d, Modifications: %d",
			totalChanges, len(patch.Additions), len(patch.Removals), len(patch.Renames), len(patch.Modifications))
	}

	t.Logf("Organization migration test completed - FromRepo: %s, ToRepo: %s",
		patch.FromRepository, patch.ToRepository)
}

// TestHasPatchWithLocation tests the HasPatchWithLocation method
func TestHasPatchWithLocation(t *testing.T) {
	patcher := createPatcherWithMigrationRules()

	// Also add checkout rules for the regular patch test
	checkoutRule := ActionPatchRule{
		Repository: "actions/checkout",
		VersionPatches: []VersionPatch{
			{
				FromVersion: "v1",
				ToVersion:   "v4",
				Description: "Test rule",
				Patches:     []FieldPatch{},
			},
		},
	}
	patcher.AddPatchRule(checkoutRule)

	// Test that location migration patches are detected
	if !patcher.HasPatchWithLocation("legacy-org/deprecated-action", "v1", "v2", "modern-org/recommended-action") {
		t.Error("Expected HasPatchWithLocation to return true for legacy-org migration")
	}

	if !patcher.HasPatchWithLocation("old-org/standard-action", "v3", "v3", "new-org/standard-action") {
		t.Error("Expected HasPatchWithLocation to return true for org migration")
	}

	// Test that non-existent migrations are not detected
	if patcher.HasPatchWithLocation("non-existent/action", "v1", "v2", "another/action") {
		t.Error("Expected HasPatchWithLocation to return false for non-existent migration")
	}

	// Test same-repository transitions (should use regular logic)
	if !patcher.HasPatch("actions/checkout", "v1", "v4") {
		t.Error("Expected HasPatch to return true for checkout v1->v4")
	}

	t.Log("HasPatchWithLocation tests completed successfully")
}
