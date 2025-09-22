package workflow

import (
	"testing"
)

func TestParseWorkflow_BothActionsAndWorkflows(t *testing.T) {
	// Sample workflow content with both regular actions and reusable workflows
	workflowContent := `
name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3  # Regular action
      - uses: actions/setup-node@v3  # Regular action
      - run: npm test
  
  deploy:
    uses: my-org/shared-workflows/.github/workflows/deploy.yml@v1  # Reusable workflow
    with:
      environment: production
`

	// Parse the workflow
	config := &Config{
		Verbose: false,
	}

	actions, err := ParseWorkflowWithConfig(workflowContent, ".github/workflows/test.yml", "test-org/test-repo", config)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	t.Logf("Found %d action references:", len(actions))
	
	regularActions := 0
	reusableWorkflows := 0
	
	for i, action := range actions {
		actionType := "action"
		if action.IsReusable {
			actionType = "reusable workflow"
			reusableWorkflows++
		} else {
			regularActions++
		}
		t.Logf("  %d. %s@%s (%s)", i+1, action.Repository, action.Version, actionType)
	}
	
	t.Logf("Summary:")
	t.Logf("- Regular actions: %d", regularActions)
	t.Logf("- Reusable workflows: %d", reusableWorkflows)
	t.Logf("- Total: %d", len(actions))
	
	// Verify we got both types
	if regularActions == 0 {
		t.Error("Expected to find regular actions, but found none")
	}
	if reusableWorkflows == 0 {
		t.Error("Expected to find reusable workflows, but found none")
	}
	
	// Verify specific actions
	if len(actions) != 3 {
		t.Errorf("Expected 3 actions total, got %d", len(actions))
	}
	
	// Expected actions:
	expectedActions := map[string]bool{
		"actions/checkout":                                      false, // regular action
		"actions/setup-node":                                   false, // regular action
		"my-org/shared-workflows":                             false, // reusable workflow
	}
	
	for _, action := range actions {
		if _, exists := expectedActions[action.Repository]; exists {
			expectedActions[action.Repository] = true
		}
	}
	
	for repo, found := range expectedActions {
		if !found {
			t.Errorf("Expected to find action/workflow %s, but it was not found", repo)
		}
	}
	
	t.Log("✅ SUCCESS: Both regular actions and reusable workflows were detected in a single scan!")
}