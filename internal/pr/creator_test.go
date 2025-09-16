package pr

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/github"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/output"
)

// TestPlanUpdates_BatchesAllRepositoryPatches tests that all patches for a repository are batched into a single plan
func TestPlanUpdates_BatchesAllRepositoryPatches(t *testing.T) {
	repositories := []output.RepositoryResult{
		{
			Name:          "test-repo",
			FullName:      "testowner/test-repo",
			DefaultBranch: "main",
			Issues: []output.ActionIssue{
				{
					Repository:       "actions/checkout",
					CurrentVersion:   "v3",
					SuggestedVersion: "v4",
					FilePath:         ".github/workflows/ci.yml",
					IssueType:        "outdated",
					Severity:         "medium",
				},
				{
					Repository:       "actions/setup-node",
					CurrentVersion:   "v2",
					SuggestedVersion: "v4",
					FilePath:         ".github/workflows/ci.yml",
					IssueType:        "deprecated",
					Severity:         "high",
				},
				{
					Repository:       "actions/upload-artifact",
					CurrentVersion:   "v3",
					SuggestedVersion: "v4",
					FilePath:         ".github/workflows/release.yml",
					IssueType:        "outdated",
					Severity:         "low",
				},
			},
		},
		{
			Name:          "another-repo",
			FullName:      "testowner/another-repo",
			DefaultBranch: "main",
			Issues: []output.ActionIssue{
				{
					Repository:       "actions/cache",
					CurrentVersion:   "v3",
					SuggestedVersion: "v4",
					FilePath:         ".github/workflows/test.yml",
					IssueType:        "outdated",
					Severity:         "low",
				},
			},
		},
	}

	plans := PlanUpdates(repositories)

	// Should have exactly one plan per repository
	if len(plans) != 2 {
		t.Errorf("Expected 2 plans, got %d", len(plans))
	}

	// First repository should have all 3 updates in one plan
	if len(plans[0].Updates) != 3 {
		t.Errorf("Expected 3 updates in first plan, got %d", len(plans[0].Updates))
	}

	// Second repository should have 1 update in one plan
	if len(plans[1].Updates) != 1 {
		t.Errorf("Expected 1 update in second plan, got %d", len(plans[1].Updates))
	}

	// Verify that all updates for a repository are in the same plan
	expectedRepo1Updates := map[string]bool{
		"actions/checkout":        false,
		"actions/setup-node":      false,
		"actions/upload-artifact": false,
	}

	for _, update := range plans[0].Updates {
		if _, exists := expectedRepo1Updates[update.ActionRepo]; exists {
			expectedRepo1Updates[update.ActionRepo] = true
		}
	}

	for action, found := range expectedRepo1Updates {
		if !found {
			t.Errorf("Expected action %s to be in the first plan but it was not found", action)
		}
	}
}

// TestPlanUpdates_SkipsRepositoriesWithoutIssues tests that repositories without issues are skipped
func TestPlanUpdates_SkipsRepositoriesWithoutIssues(t *testing.T) {
	repositories := []output.RepositoryResult{
		{
			Name:          "repo-with-issues",
			FullName:      "testowner/repo-with-issues",
			DefaultBranch: "main",
			Issues: []output.ActionIssue{
				{
					Repository:       "actions/checkout",
					CurrentVersion:   "v3",
					SuggestedVersion: "v4",
					FilePath:         ".github/workflows/ci.yml",
					IssueType:        "outdated",
					Severity:         "medium",
				},
			},
		},
		{
			Name:          "repo-without-issues",
			FullName:      "testowner/repo-without-issues",
			DefaultBranch: "main",
			Issues:        []output.ActionIssue{}, // No issues
		},
	}

	plans := PlanUpdates(repositories)

	// Should only have one plan for the repository with issues
	if len(plans) != 1 {
		t.Errorf("Expected 1 plan, got %d", len(plans))
	}

	if plans[0].Repository.FullName != "testowner/repo-with-issues" {
		t.Errorf("Expected plan for repo-with-issues, got %s", plans[0].Repository.FullName)
	}
}

// TestPlanUpdates_SkipsIssuesWithoutSuggestedVersions tests that issues without suggested versions are skipped
func TestPlanUpdates_SkipsIssuesWithoutSuggestedVersions(t *testing.T) {
	repositories := []output.RepositoryResult{
		{
			Name:          "test-repo",
			FullName:      "testowner/test-repo",
			DefaultBranch: "main",
			Issues: []output.ActionIssue{
				{
					Repository:       "actions/checkout",
					CurrentVersion:   "v3",
					SuggestedVersion: "v4", // Has suggested version
					FilePath:         ".github/workflows/ci.yml",
					IssueType:        "outdated",
					Severity:         "medium",
				},
				{
					Repository:       "actions/setup-node",
					CurrentVersion:   "v2",
					SuggestedVersion: "", // No suggested version
					FilePath:         ".github/workflows/ci.yml",
					IssueType:        "unknown",
					Severity:         "low",
				},
			},
		},
	}

	plans := PlanUpdates(repositories)

	// Should have one plan with only one update (skipping the issue without suggested version)
	if len(plans) != 1 {
		t.Errorf("Expected 1 plan, got %d", len(plans))
	}

	if len(plans[0].Updates) != 1 {
		t.Errorf("Expected 1 update, got %d", len(plans[0].Updates))
	}

	if plans[0].Updates[0].ActionRepo != "actions/checkout" {
		t.Errorf("Expected checkout action, got %s", plans[0].Updates[0].ActionRepo)
	}
}

// TestPlanUpdates_HandlesDuplicateActionsAcrossFiles tests that the same action in multiple files is handled correctly
func TestPlanUpdates_HandlesDuplicateActionsAcrossFiles(t *testing.T) {
	repositories := []output.RepositoryResult{
		{
			Name:          "test-repo",
			FullName:      "testowner/test-repo",
			DefaultBranch: "main",
			Issues: []output.ActionIssue{
				{
					Repository:       "actions/checkout",
					CurrentVersion:   "v3",
					SuggestedVersion: "v4",
					FilePath:         ".github/workflows/ci.yml",
					IssueType:        "outdated",
					Severity:         "medium",
				},
				{
					Repository:       "actions/checkout",
					CurrentVersion:   "v3",
					SuggestedVersion: "v4",
					FilePath:         ".github/workflows/release.yml", // Same action, different file
					IssueType:        "outdated",
					Severity:         "medium",
				},
			},
		},
	}

	plans := PlanUpdates(repositories)

	// Should have one plan with both updates (even for the same action in different files)
	if len(plans) != 1 {
		t.Errorf("Expected 1 plan, got %d", len(plans))
	}

	if len(plans[0].Updates) != 2 {
		t.Errorf("Expected 2 updates, got %d", len(plans[0].Updates))
	}

	// Both updates should be for the same action but different files
	update1 := plans[0].Updates[0]
	update2 := plans[0].Updates[1]

	if update1.ActionRepo != "actions/checkout" || update2.ActionRepo != "actions/checkout" {
		t.Error("Expected both updates to be for actions/checkout")
	}

	if update1.FilePath == update2.FilePath {
		t.Error("Expected updates to be for different files")
	}
}

// TestCreateUpdatePRs_NeverCreatesSeparatePRsPerRepository tests that only one PR is created per repository
func TestCreateUpdatePRs_NeverCreatesSeparatePRsPerRepository(t *testing.T) {
	// Create a mock github client
	mockClient := &github.Client{}
	creator := NewCreator(mockClient)

	plans := []UpdatePlan{
		{
			Repository: github.Repository{
				Owner:         "testowner",
				Name:          "test-repo",
				FullName:      "testowner/test-repo",
				DefaultBranch: "main",
			},
			Updates: []ActionUpdate{
				{
					FilePath:       ".github/workflows/ci.yml",
					ActionRepo:     "actions/checkout",
					CurrentVersion: "v3",
					TargetVersion:  "v4",
					Issue: output.ActionIssue{
						Repository:       "actions/checkout",
						CurrentVersion:   "v3",
						SuggestedVersion: "v4",
						FilePath:         ".github/workflows/ci.yml",
						IssueType:        "outdated",
						Severity:         "medium",
					},
				},
				{
					FilePath:       ".github/workflows/ci.yml",
					ActionRepo:     "actions/setup-node",
					CurrentVersion: "v2",
					TargetVersion:  "v4",
					Issue: output.ActionIssue{
						Repository:       "actions/setup-node",
						CurrentVersion:   "v2",
						SuggestedVersion: "v4",
						FilePath:         ".github/workflows/ci.yml",
						IssueType:        "deprecated",
						Severity:         "high",
					},
				},
			},
		},
	}

	createdPRs, err := creator.CreateUpdatePRs(plans)
	if err != nil {
		t.Fatalf("CreateUpdatePRs failed: %v", err)
	}

	// Should create exactly one PR for the one repository plan
	if len(createdPRs) != 1 {
		t.Errorf("Expected 1 PR, got %d", len(createdPRs))
	}

	// The PR should indicate it contains multiple updates
	if createdPRs[0].UpdateCount != 2 {
		t.Errorf("Expected PR to contain 2 updates, got %d", createdPRs[0].UpdateCount)
	}
}

// TestCreateUpdatePRs_HandlesMultipleRepositories tests that separate PRs are created for different repositories
func TestCreateUpdatePRs_HandlesMultipleRepositories(t *testing.T) {
	mockClient := &github.Client{}
	creator := NewCreator(mockClient)

	plans := []UpdatePlan{
		{
			Repository: github.Repository{
				Owner:         "testowner",
				Name:          "repo1",
				FullName:      "testowner/repo1",
				DefaultBranch: "main",
			},
			Updates: []ActionUpdate{
				{
					FilePath:       ".github/workflows/ci.yml",
					ActionRepo:     "actions/checkout",
					CurrentVersion: "v3",
					TargetVersion:  "v4",
					Issue:          output.ActionIssue{Repository: "actions/checkout"},
				},
			},
		},
		{
			Repository: github.Repository{
				Owner:         "testowner",
				Name:          "repo2",
				FullName:      "testowner/repo2",
				DefaultBranch: "main",
			},
			Updates: []ActionUpdate{
				{
					FilePath:       ".github/workflows/test.yml",
					ActionRepo:     "actions/cache",
					CurrentVersion: "v3",
					TargetVersion:  "v4",
					Issue:          output.ActionIssue{Repository: "actions/cache"},
				},
			},
		},
	}

	createdPRs, err := creator.CreateUpdatePRs(plans)
	if err != nil {
		t.Fatalf("CreateUpdatePRs failed: %v", err)
	}

	// Should create exactly one PR per repository
	if len(createdPRs) != 2 {
		t.Errorf("Expected 2 PRs, got %d", len(createdPRs))
	}

	// Verify that PRs are for different repositories
	repos := make(map[string]bool)
	for _, pr := range createdPRs {
		repos[pr.Repository] = true
	}

	if len(repos) != 2 {
		t.Error("Expected PRs to be for 2 different repositories")
	}
}

// TestPlanUpdates_BatchingInvariant tests the core invariant that one repository always results in at most one plan
func TestPlanUpdates_BatchingInvariant(t *testing.T) {
	// Test with various combinations to ensure batching invariant holds
	testCases := []struct {
		name        string
		repo        output.RepositoryResult
		expectedLen int
	}{
		{
			name: "Multiple actions same file",
			repo: output.RepositoryResult{
				Name:          "test-repo",
				FullName:      "owner/test-repo",
				DefaultBranch: "main",
				Issues: []output.ActionIssue{
					{Repository: "actions/checkout", CurrentVersion: "v3", SuggestedVersion: "v4", FilePath: ".github/workflows/ci.yml"},
					{Repository: "actions/setup-node", CurrentVersion: "v2", SuggestedVersion: "v4", FilePath: ".github/workflows/ci.yml"},
					{Repository: "actions/cache", CurrentVersion: "v3", SuggestedVersion: "v4", FilePath: ".github/workflows/ci.yml"},
				},
			},
			expectedLen: 1,
		},
		{
			name: "Multiple actions different files",
			repo: output.RepositoryResult{
				Name:          "test-repo",
				FullName:      "owner/test-repo",
				DefaultBranch: "main",
				Issues: []output.ActionIssue{
					{Repository: "actions/checkout", CurrentVersion: "v3", SuggestedVersion: "v4", FilePath: ".github/workflows/ci.yml"},
					{Repository: "actions/setup-node", CurrentVersion: "v2", SuggestedVersion: "v4", FilePath: ".github/workflows/release.yml"},
					{Repository: "actions/cache", CurrentVersion: "v3", SuggestedVersion: "v4", FilePath: ".github/workflows/test.yml"},
				},
			},
			expectedLen: 1,
		},
		{
			name: "Same action different versions different files",
			repo: output.RepositoryResult{
				Name:          "test-repo",
				FullName:      "owner/test-repo",
				DefaultBranch: "main",
				Issues: []output.ActionIssue{
					{Repository: "actions/checkout", CurrentVersion: "v2", SuggestedVersion: "v4", FilePath: ".github/workflows/ci.yml"},
					{Repository: "actions/checkout", CurrentVersion: "v3", SuggestedVersion: "v4", FilePath: ".github/workflows/release.yml"},
				},
			},
			expectedLen: 1,
		},
		{
			name: "Many actions many files",
			repo: output.RepositoryResult{
				Name:          "test-repo",
				FullName:      "owner/test-repo",
				DefaultBranch: "main",
				Issues: make([]output.ActionIssue, 20), // 20 different action issues
			},
			expectedLen: 1,
		},
	}

	// Fill the many actions test case
	for i := 0; i < 20; i++ {
		testCases[3].repo.Issues[i] = output.ActionIssue{
			Repository:       fmt.Sprintf("actions/action-%d", i),
			CurrentVersion:   "v1",
			SuggestedVersion: "v2",
			FilePath:         fmt.Sprintf(".github/workflows/workflow-%d.yml", i%5), // 5 different files
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plans := PlanUpdates([]output.RepositoryResult{tc.repo})
			
			if len(plans) != tc.expectedLen {
				t.Errorf("Expected %d plan(s), got %d", tc.expectedLen, len(plans))
			}

			if len(plans) > 0 {
				// All updates should be in the single plan
				totalIssues := len(tc.repo.Issues)
				actualUpdates := len(plans[0].Updates)
				
				if actualUpdates != totalIssues {
					t.Errorf("Expected all %d issues to be converted to updates, got %d", totalIssues, actualUpdates)
				}
			}
		})
	}
}

// TestPlanUpdates_PreventsSplitting tests that there's no scenario where a repository's patches could be split
func TestPlanUpdates_PreventsSplitting(t *testing.T) {
	// This is a regression test for any future changes that might accidentally split patches
	repositories := []output.RepositoryResult{
		{
			Name:          "repo-with-many-issues",
			FullName:      "owner/repo-with-many-issues",
			DefaultBranch: "main",
			Issues: []output.ActionIssue{
				// Mix of different action types, versions, files, and severities
				{Repository: "actions/checkout", CurrentVersion: "v1", SuggestedVersion: "v4", FilePath: ".github/workflows/ci.yml", IssueType: "deprecated", Severity: "high"},
				{Repository: "actions/checkout", CurrentVersion: "v3", SuggestedVersion: "v4", FilePath: ".github/workflows/release.yml", IssueType: "outdated", Severity: "medium"},
				{Repository: "actions/setup-node", CurrentVersion: "v2", SuggestedVersion: "v4", FilePath: ".github/workflows/ci.yml", IssueType: "deprecated", Severity: "high"},
				{Repository: "actions/cache", CurrentVersion: "v3", SuggestedVersion: "v4", FilePath: ".github/workflows/test.yml", IssueType: "outdated", Severity: "low"},
				{Repository: "custom/action", CurrentVersion: "v1.0.0", SuggestedVersion: "v2.0.0", FilePath: ".github/workflows/custom.yml", IssueType: "migration", Severity: "medium"},
			},
		},
	}

	plans := PlanUpdates(repositories)

	// Fundamental invariant: exactly one plan per repository with issues
	if len(plans) != 1 {
		t.Fatalf("CRITICAL: Patches were split! Expected 1 plan, got %d", len(plans))
	}

	// All issues should be in the single plan
	if len(plans[0].Updates) != 5 {
		t.Errorf("Expected all 5 issues to be in single plan, got %d updates", len(plans[0].Updates))
	}

	// Verify all different action types are present
	actionsSeen := make(map[string]bool)
	for _, update := range plans[0].Updates {
		actionsSeen[update.ActionRepo] = true
	}

	expectedActions := []string{"actions/checkout", "actions/setup-node", "actions/cache", "custom/action"}
	for _, expected := range expectedActions {
		if !actionsSeen[expected] {
			t.Errorf("Expected action %s to be in plan but it was missing", expected)
		}
	}
}

// TestValidateBatchingInvariant tests the batching validation function
func TestValidateBatchingInvariant(t *testing.T) {
	testCases := []struct {
		name          string
		repositories  []output.RepositoryResult
		plans         []UpdatePlan
		shouldFail    bool
		expectedError string
	}{
		{
			name: "Valid batching - one repo one plan",
			repositories: []output.RepositoryResult{
				{
					FullName: "owner/repo1",
					Issues: []output.ActionIssue{
						{Repository: "actions/checkout", SuggestedVersion: "v4"},
					},
				},
			},
			plans: []UpdatePlan{
				{
					Repository: github.Repository{FullName: "owner/repo1"},
					Updates:    []ActionUpdate{{ActionRepo: "actions/checkout"}},
				},
			},
			shouldFail: false,
		},
		{
			name: "Valid batching - multiple repos multiple plans",
			repositories: []output.RepositoryResult{
				{
					FullName: "owner/repo1",
					Issues: []output.ActionIssue{
						{Repository: "actions/checkout", SuggestedVersion: "v4"},
						{Repository: "actions/setup-node", SuggestedVersion: "v4"},
					},
				},
				{
					FullName: "owner/repo2",
					Issues: []output.ActionIssue{
						{Repository: "actions/cache", SuggestedVersion: "v4"},
					},
				},
			},
			plans: []UpdatePlan{
				{
					Repository: github.Repository{FullName: "owner/repo1"},
					Updates: []ActionUpdate{
						{ActionRepo: "actions/checkout"},
						{ActionRepo: "actions/setup-node"},
					},
				},
				{
					Repository: github.Repository{FullName: "owner/repo2"},
					Updates:    []ActionUpdate{{ActionRepo: "actions/cache"}},
				},
			},
			shouldFail: false,
		},
		{
			name: "Invalid - missing plan for repository",
			repositories: []output.RepositoryResult{
				{
					FullName: "owner/repo1",
					Issues: []output.ActionIssue{
						{Repository: "actions/checkout", SuggestedVersion: "v4"},
					},
				},
			},
			plans:         []UpdatePlan{}, // No plans but should have one
			shouldFail:    true,
			expectedError: "expected 1 plans for 1 repositories",
		},
		{
			name: "Invalid - repository appears in multiple plans",
			repositories: []output.RepositoryResult{
				{
					FullName: "owner/repo1",
					Issues: []output.ActionIssue{
						{Repository: "actions/checkout", SuggestedVersion: "v4"},
						{Repository: "actions/setup-node", SuggestedVersion: "v4"},
					},
				},
			},
			plans: []UpdatePlan{
				{
					Repository: github.Repository{FullName: "owner/repo1"},
					Updates:    []ActionUpdate{{ActionRepo: "actions/checkout"}},
				},
				{
					Repository: github.Repository{FullName: "owner/repo1"}, // Same repo again!
					Updates:    []ActionUpdate{{ActionRepo: "actions/setup-node"}},
				},
			},
			shouldFail:    true,
			expectedError: "repository owner/repo1 appears in 2 plans",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBatchingInvariant(tc.repositories, tc.plans)
			
			if tc.shouldFail {
				if err == nil {
					t.Error("Expected validation to fail but it passed")
				} else if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to pass but got error: %v", err)
				}
			}
		})
	}
}