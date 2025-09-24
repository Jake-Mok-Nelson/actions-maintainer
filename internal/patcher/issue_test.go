package patcher

import (
	"testing"
)

// TestCurrentPatchBehaviorFixed verifies the fix - NewPatcher() now loads default rules
func TestCurrentPatchBehaviorFixed(t *testing.T) {
	// Create a patcher using NewPatcher (should now have default rules loaded)
	patcher := NewPatcher()

	// Test data: simulate actions/checkout v3 to v4 upgrade
	// This is a common upgrade that should have transformation rules
	withBlock := map[string]interface{}{
		"fetch-depth": 0,
		"token":       "${{ secrets.GITHUB_TOKEN }}",
	}

	// Try to build patch for a supported action (actions/checkout v3 -> v4)
	patch, err := patcher.BuildPatch("actions/checkout", "v3", "v4", withBlock)
	if err != nil {
		t.Fatalf("Failed to build patch: %v", err)
	}

	// Fixed behavior: Patches should be applied because default rules are now loaded
	if !patch.Applied {
		t.Error("Expected patches to be applied for checkout v3->v4 with default rules loaded")
		return
	}

	totalChanges := len(patch.Additions) + len(patch.Removals) + len(patch.Renames) + len(patch.Modifications)
	if totalChanges == 0 {
		t.Error("Expected changes to be applied but none were found")
	}

	t.Logf("SUCCESS: Applied %d total changes for checkout v3->v4", totalChanges)
}

// TestNewPatcherWithDefaultRules verifies that default rules are loaded automatically
func TestNewPatcherWithDefaultRules(t *testing.T) {
	// Create a patcher using NewPatcher
	patcher := NewPatcher()

	// Verify default rules are loaded
	rules := patcher.GetPatchRules()
	if len(rules) == 0 {
		t.Error("Expected default rules to be loaded automatically by NewPatcher()")
		return
	}

	// Check for common actions
	expectedActions := []string{"actions/checkout", "actions/setup-node"}
	for _, expected := range expectedActions {
		if _, exists := rules[expected]; !exists {
			t.Errorf("Expected %s to be in default rules", expected)
		}
	}

	t.Logf("SUCCESS: NewPatcher() loaded %d default rules", len(rules))
}
