package transformer

import (
	"testing"
)

// TestBasicPatchOperations tests the basic patch operations
func TestBasicPatchOperations(t *testing.T) {
	transformer := NewTransformer()

	// Test data: simulate actions/checkout v3 to v4 upgrade
	withBlock := map[string]interface{}{
		"fetch-depth": 0,
		"token":       "${{ secrets.GITHUB_TOKEN }}",
	}

	// Apply patches
	result, err := transformer.ApplyPatches("actions/checkout", "v3", "v4", withBlock)
	if err != nil {
		t.Fatalf("Failed to apply patches: %v", err)
	}

	if !result.Applied {
		t.Log("No patches were applied for checkout v3->v4, which is expected")
		return
	}

	// Check that changes were made
	if len(result.Changes) == 0 {
		t.Error("Expected changes but none were reported")
	}

	t.Logf("Applied %d changes: %v", len(result.Changes), result.Changes)
}

// TestCheckoutV1ToV4Transformation tests the transformation from checkout v1 to v4
func TestCheckoutV1ToV4Transformation(t *testing.T) {
	transformer := NewTransformer()

	// Test data: simulate actions/checkout v1 with token
	withBlock := map[string]interface{}{
		"token": "${{ secrets.GITHUB_TOKEN }}",
	}

	// Apply patches
	result, err := transformer.ApplyPatches("actions/checkout", "v1", "v4", withBlock)
	if err != nil {
		t.Fatalf("Failed to apply patches: %v", err)
	}

	if !result.Applied {
		t.Fatal("Expected patches to be applied for checkout v1->v4")
	}

	// Check that changes were made
	if len(result.Changes) == 0 {
		t.Error("Expected changes but none were reported")
	}

	// Verify token was removed and fetch-depth was added
	updatedWith := result.UpdatedWith.(map[string]interface{})

	if _, hasToken := updatedWith["token"]; hasToken {
		t.Error("Expected token field to be removed")
	}

	if fetchDepth, hasFetchDepth := updatedWith["fetch-depth"]; !hasFetchDepth {
		t.Error("Expected fetch-depth field to be added")
	} else if fetchDepth != 1 {
		t.Errorf("Expected fetch-depth to be 1, got %v", fetchDepth)
	}

	t.Logf("Successfully applied %d changes: %v", len(result.Changes), result.Changes)
}

// TestSetupNodeV2ToV4Transformation tests setup-node parameter renaming
func TestSetupNodeV2ToV4Transformation(t *testing.T) {
	transformer := NewTransformer()

	// Test data: simulate actions/setup-node v2 with version parameter
	withBlock := map[string]interface{}{
		"version": "16",
	}

	// Apply patches
	result, err := transformer.ApplyPatches("actions/setup-node", "v2", "v4", withBlock)
	if err != nil {
		t.Fatalf("Failed to apply patches: %v", err)
	}

	if !result.Applied {
		t.Fatal("Expected patches to be applied for setup-node v2->v4")
	}

	// Verify version was renamed to node-version and cache was added
	updatedWith := result.UpdatedWith.(map[string]interface{})

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

	t.Logf("Successfully applied %d changes: %v", len(result.Changes), result.Changes)
}

// TestNoTransformationForUnsupportedAction tests behavior when no transformation rules exist
func TestNoTransformationForUnsupportedAction(t *testing.T) {
	transformer := NewTransformer()

	// Test with an action that doesn't have transformation rules
	withBlock := map[string]interface{}{
		"some-param": "value",
	}

	result, err := transformer.ApplyPatches("unsupported/action", "v1", "v2", withBlock)
	if err != nil {
		t.Fatalf("Failed to apply patches: %v", err)
	}

	if result.Applied {
		t.Error("Expected no patches to be applied for unsupported action")
	}

	if len(result.Changes) > 0 {
		t.Error("Expected no changes for unsupported action")
	}

	t.Log("Correctly handled unsupported action with no transformation")
}

// TestNilWithBlock tests handling of nil with blocks
func TestNilWithBlock(t *testing.T) {
	transformer := NewTransformer()

	// Test with nil with block
	result, err := transformer.ApplyPatches("actions/checkout", "v1", "v4", nil)
	if err != nil {
		t.Fatalf("Failed to apply patches to nil with block: %v", err)
	}

	if !result.Applied {
		t.Fatal("Expected patches to be applied even with nil with block")
	}

	// Should add fetch-depth field to empty map
	updatedWith := result.UpdatedWith.(map[string]interface{})
	if fetchDepth, hasFetchDepth := updatedWith["fetch-depth"]; !hasFetchDepth {
		t.Error("Expected fetch-depth field to be added to nil with block")
	} else if fetchDepth != 1 {
		t.Errorf("Expected fetch-depth to be 1, got %v", fetchDepth)
	}

	t.Logf("Successfully handled nil with block: %v", result.Changes)
}

// TestGetSupportedActions tests that we can retrieve supported actions
func TestGetSupportedActions(t *testing.T) {
	wt := NewWorkflowTransformer()
	actions := wt.GetSupportedActions()

	if len(actions) == 0 {
		t.Error("Expected at least some supported actions")
	}

	// Check for some expected actions
	expectedActions := []string{
		"actions/checkout",
		"actions/setup-node", 
		"actions/setup-python",
		"actions/upload-artifact",
	}

	for _, expected := range expectedActions {
		found := false
		for _, action := range actions {
			if action == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected action %s to be supported", expected)
		}
	}

	t.Logf("Found %d supported actions: %v", len(actions), actions)
}