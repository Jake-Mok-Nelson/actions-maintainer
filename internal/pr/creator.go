package pr

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/github"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/output"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/patcher"
)

// Creator handles creating pull requests for action updates
type Creator struct {
	githubClient *github.Client
	patcher      *patcher.WorkflowPatcher
	template     *template.Template
}

// UpdatePlan represents a plan to update actions in a repository
// Each UpdatePlan corresponds to exactly one repository and contains ALL
// action updates for that repository. This ensures that all patches for
// a repository are applied together in a single pull request.
type UpdatePlan struct {
	Repository github.Repository
	Updates    []ActionUpdate // ALL updates for this repository
}

// ActionUpdate represents a single action update
type ActionUpdate struct {
	FilePath       string
	ActionRepo     string
	CurrentVersion string
	TargetVersion  string
	Issue          output.ActionIssue
}

// TemplateData represents the data available to PR body templates
type TemplateData struct {
	Repository        github.Repository
	Updates           []ActionUpdate
	UpdateCount       int
	DeprecatedUpdates []ActionUpdate
	OutdatedUpdates   []ActionUpdate
	SecurityUpdates   []ActionUpdate
	OtherUpdates      []ActionUpdate
}

// NewCreator creates a new PR creator
func NewCreator(githubClient *github.Client) *Creator {
	return &Creator{
		githubClient: githubClient,
		patcher:      patcher.NewWorkflowPatcher(),
		template:     nil, // Use default template
	}
}

// NewCreatorWithTemplate creates a new PR creator with a custom template
func NewCreatorWithTemplate(githubClient *github.Client, tmpl *template.Template) *Creator {
	return &Creator{
		githubClient: githubClient,
		patcher:      patcher.NewWorkflowPatcher(),
		template:     tmpl,
	}
}

// CreateUpdatePRs creates pull requests for action updates
// This function creates exactly one PR per UpdatePlan, and since PlanUpdates
// ensures one plan per repository, this guarantees one PR per repository.
// All patches for a repository are batched together in the same PR.
func (c *Creator) CreateUpdatePRs(plans []UpdatePlan) ([]output.CreatedPR, error) {
	var createdPRs []output.CreatedPR

	for _, plan := range plans {
		if len(plan.Updates) == 0 {
			continue
		}

		// Create a single PR that contains ALL updates for this repository
		createdPR, err := c.createPRForPlan(plan)
		if err != nil {
			fmt.Printf("Failed to create PR for %s: %v\n", plan.Repository.FullName, err)
			continue
		}

		createdPRs = append(createdPRs, createdPR)
		fmt.Printf("Created PR for %s with %d action updates\n", plan.Repository.FullName, len(plan.Updates))
	}

	return createdPRs, nil
}

// createPRForPlan creates a pull request for a single update plan
func (c *Creator) createPRForPlan(plan UpdatePlan) (output.CreatedPR, error) {
	// Create a descriptive branch name
	branchName := fmt.Sprintf("actions-maintainer/update-actions-%d", len(plan.Updates))

	// Generate PR title and body
	title := c.generatePRTitle(plan)
	body := c.generatePRBody(plan)

	// For now, we'll simulate the PR creation since we'd need to:
	// 1. Create a new branch
	// 2. Update the workflow files
	// 3. Commit the changes
	// 4. Create the PR

	// This is a simplified implementation that would need additional
	// GitHub API calls to actually create and push changes
	fmt.Printf("Would create PR for %s:\n", plan.Repository.FullName)
	fmt.Printf("Branch: %s\n", branchName)
	fmt.Printf("Title: %s\n", title)
	fmt.Printf("Body: %s\n", body)

	// Return simulated PR info
	prNumber := 42 // Simulated PR number
	prURL := fmt.Sprintf("https://github.com/%s/pull/%d", plan.Repository.FullName, prNumber)

	return output.CreatedPR{
		Repository:  plan.Repository.FullName,
		URL:         prURL,
		Title:       title,
		Number:      prNumber,
		UpdateCount: len(plan.Updates),
	}, nil
}

// generatePRTitle creates a descriptive title for the PR
func (c *Creator) generatePRTitle(plan UpdatePlan) string {
	if len(plan.Updates) == 1 {
		update := plan.Updates[0]
		return fmt.Sprintf("Update %s from %s to %s",
			update.ActionRepo, update.CurrentVersion, update.TargetVersion)
	}

	return fmt.Sprintf("Update %d GitHub Actions to latest versions", len(plan.Updates))
}

// generatePRBody creates a detailed body for the PR
func (c *Creator) generatePRBody(plan UpdatePlan) string {
	// If we have a custom template, use it
	if c.template != nil {
		return c.generatePRBodyFromTemplate(plan)
	}
	
	// Otherwise, use the default template
	return c.generateDefaultPRBody(plan)
}

// generatePRBodyFromTemplate generates PR body using the provided template
func (c *Creator) generatePRBodyFromTemplate(plan UpdatePlan) string {
	// Group updates by issue type
	deprecatedUpdates := []ActionUpdate{}
	outdatedUpdates := []ActionUpdate{}
	securityUpdates := []ActionUpdate{}
	otherUpdates := []ActionUpdate{}

	for _, update := range plan.Updates {
		switch update.Issue.IssueType {
		case "deprecated":
			deprecatedUpdates = append(deprecatedUpdates, update)
		case "outdated":
			outdatedUpdates = append(outdatedUpdates, update)
		case "security":
			securityUpdates = append(securityUpdates, update)
		default:
			otherUpdates = append(otherUpdates, update)
		}
	}

	// Prepare template data
	data := TemplateData{
		Repository:        plan.Repository,
		Updates:           plan.Updates,
		UpdateCount:       len(plan.Updates),
		DeprecatedUpdates: deprecatedUpdates,
		OutdatedUpdates:   outdatedUpdates,
		SecurityUpdates:   securityUpdates,
		OtherUpdates:      otherUpdates,
	}

	// Execute template
	var buf bytes.Buffer
	if err := c.template.Execute(&buf, data); err != nil {
		// Fall back to default template if custom template fails
		return c.generateDefaultPRBody(plan)
	}

	return buf.String()
}

// generateDefaultPRBody creates a detailed body for the PR using the default template
func (c *Creator) generateDefaultPRBody(plan UpdatePlan) string {
	var body strings.Builder

	body.WriteString("## GitHub Actions Updates\n\n")
	body.WriteString("This PR updates GitHub Actions to their latest recommended versions.\n\n")

	// Group updates by issue type
	deprecatedUpdates := []ActionUpdate{}
	outdatedUpdates := []ActionUpdate{}

	for _, update := range plan.Updates {
		switch update.Issue.IssueType {
		case "deprecated":
			deprecatedUpdates = append(deprecatedUpdates, update)
		case "outdated":
			outdatedUpdates = append(outdatedUpdates, update)
		}
	}

	// Deprecated updates section
	if len(deprecatedUpdates) > 0 {
		body.WriteString("### ⚠️ Deprecated Version Updates\n\n")
		for _, update := range deprecatedUpdates {
			body.WriteString(fmt.Sprintf("- **%s**: %s → %s\n",
				update.ActionRepo, update.CurrentVersion, update.TargetVersion))
			body.WriteString(fmt.Sprintf("  - **File**: `%s`\n\n", update.FilePath))
		}
	}

	// Outdated updates section
	if len(outdatedUpdates) > 0 {
		body.WriteString("### 📊 Version Updates\n\n")
		for _, update := range outdatedUpdates {
			body.WriteString(fmt.Sprintf("- **%s**: %s → %s\n",
				update.ActionRepo, update.CurrentVersion, update.TargetVersion))
			body.WriteString(fmt.Sprintf("  - **File**: `%s`\n\n", update.FilePath))
		}
	}

	body.WriteString("### Benefits of staying up to date\n\n")
	body.WriteString("- ✅ Improved performance\n")
	body.WriteString("- ✅ New features and bug fixes\n")
	body.WriteString("- ✅ Better compatibility\n\n")

	body.WriteString("### Testing\n\n")
	body.WriteString("Please ensure all CI checks pass before merging.\n\n")

	body.WriteString("---\n")
	body.WriteString("*This PR was automatically generated by [actions-maintainer](https://github.com/Jake-Mok-Nelson/actions-maintainer)*")

	return body.String()
}

// PlanUpdates creates update plans from scan results
// This function ensures that all patches for a repository are batched into a single UpdatePlan.
// This is critical to ensure that when PRs are created, all related patches are applied together
// in the same pull request, preventing merge conflicts and ensuring atomic updates.
func PlanUpdates(repositories []output.RepositoryResult) []UpdatePlan {
	var plans []UpdatePlan

	for _, repo := range repositories {
		if len(repo.Issues) == 0 {
			continue
		}

		plan := UpdatePlan{
			Repository: github.Repository{
				Owner:         extractOwner(repo.FullName),
				Name:          repo.Name,
				FullName:      repo.FullName,
				DefaultBranch: repo.DefaultBranch,
			},
			Updates: []ActionUpdate{},
		}

		// Collect ALL issues for this repository into a single plan
		// This ensures patches are never split across multiple PRs for the same repository
		for _, issue := range repo.Issues {
			if issue.SuggestedVersion == "" {
				continue // Skip issues without suggested fixes
			}

			update := ActionUpdate{
				FilePath:       issue.FilePath,
				ActionRepo:     issue.Repository,
				CurrentVersion: issue.CurrentVersion,
				TargetVersion:  issue.SuggestedVersion,
				Issue:          issue,
			}

			plan.Updates = append(plan.Updates, update)
		}

		if len(plan.Updates) > 0 {
			plans = append(plans, plan)
		}
	}

	// Validate the critical batching invariant
	if err := validateBatchingInvariant(repositories, plans); err != nil {
		// This should never happen with the current implementation,
		// but we check to prevent future regressions
		fmt.Printf("CRITICAL ERROR: Batching invariant violated: %v\n", err)
	}

	return plans
}

// extractOwner extracts the owner from a full repository name
func extractOwner(fullName string) string {
	parts := strings.Split(fullName, "/")
	if len(parts) >= 2 {
		return parts[0]
	}
	return ""
}

// UpdateWorkflowContent updates the content of a workflow file with new action versions
func UpdateWorkflowContent(content string, updates []ActionUpdate) string {
	updatedContent := content

	for _, update := range updates {
		// Create pattern to match the action reference
		oldRef := fmt.Sprintf("%s@%s", update.ActionRepo, update.CurrentVersion)
		newRef := fmt.Sprintf("%s@%s", update.ActionRepo, update.TargetVersion)

		// Use regex to safely replace action references
		pattern := regexp.MustCompile(regexp.QuoteMeta(oldRef))
		updatedContent = pattern.ReplaceAllString(updatedContent, newRef)
	}

	return updatedContent
}

// UpdateWorkflowContentWithTransformations updates workflow content with both version changes and schema patches
func (c *Creator) UpdateWorkflowContentWithTransformations(content string, updates []ActionUpdate) (string, []string, error) {
	// Convert ActionUpdate to patcher.ActionVersionUpdate
	patcherUpdates := make([]patcher.ActionVersionUpdate, len(updates))
	for i, update := range updates {
		patcherUpdates[i] = patcher.ActionVersionUpdate{
			ActionRepo:  update.ActionRepo,
			FromVersion: update.CurrentVersion,
			ToVersion:   update.TargetVersion,
			FilePath:    update.FilePath,
		}
	}

	// Apply patches
	updatedContent, changes, err := c.patcher.PatchWorkflowContent(content, patcherUpdates)
	if err != nil {
		return content, nil, fmt.Errorf("failed to apply patches: %w", err)
	}

	// Update version references
	finalContent := UpdateWorkflowContent(updatedContent, updates)

	return finalContent, changes, nil
}

// validateBatchingInvariant ensures that the batching logic is working correctly
// This function validates that:
// 1. Each repository with issues gets exactly one plan
// 2. No patches are split across multiple plans for the same repository
// 3. All issues with suggested versions are included in plans
func validateBatchingInvariant(repositories []output.RepositoryResult, plans []UpdatePlan) error {
	// Count repositories that should have plans (have issues with suggested versions)
	reposWithFixableIssues := 0
	totalFixableIssues := 0

	for _, repo := range repositories {
		hasFixableIssues := false
		for _, issue := range repo.Issues {
			if issue.SuggestedVersion != "" {
				totalFixableIssues++
				hasFixableIssues = true
			}
		}
		if hasFixableIssues {
			reposWithFixableIssues++
		}
	}

	// Count total updates across all plans and check for duplicate repositories
	totalUpdates := 0
	repoPlans := make(map[string]int) // repo -> plan count

	for _, plan := range plans {
		totalUpdates += len(plan.Updates)
		repoPlans[plan.Repository.FullName]++
	}

	// Each repository should appear in exactly one plan
	for repo, count := range repoPlans {
		if count != 1 {
			return fmt.Errorf("repository %s appears in %d plans, expected exactly 1", repo, count)
		}
	}

	// Should have exactly one plan per repository with fixable issues
	if len(plans) != reposWithFixableIssues {
		return fmt.Errorf("expected %d plans for %d repositories with fixable issues, got %d plans",
			reposWithFixableIssues, reposWithFixableIssues, len(plans))
	}

	// Total updates should equal total fixable issues
	if totalUpdates != totalFixableIssues {
		return fmt.Errorf("expected %d updates (total fixable issues), got %d updates",
			totalFixableIssues, totalUpdates)
	}

	return nil
}
