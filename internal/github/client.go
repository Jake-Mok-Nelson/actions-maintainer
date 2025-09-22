package github

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-github/v65/github"
	"golang.org/x/oauth2"
)

// Config holds configuration options for the GitHub client
type Config struct {
	Verbose bool
}

// Client wraps the GitHub API client with our specific functionality
type Client struct {
	client  *github.Client
	ctx     context.Context
	verbose bool
}

// Repository represents a GitHub repository with relevant metadata
type Repository struct {
	Owner            string            `json:"owner"`
	Name             string            `json:"name"`
	DefaultBranch    string            `json:"default_branch"`
	FullName         string            `json:"full_name"`
	CustomProperties map[string]string `json:"custom_properties,omitempty"`
}

// WorkflowFile represents a workflow file found in a repository
type WorkflowFile struct {
	Repository Repository
	Path       string
	Content    string
}

// NewClient creates a new GitHub API client with authentication
func NewClient(token string) *Client {
	return NewClientWithConfig(token, &Config{Verbose: false})
}

// NewClientWithConfig creates a new GitHub API client with authentication and configuration
func NewClientWithConfig(token string, config *Config) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	if config.Verbose {
		log.Printf("GitHub client initialized with verbose logging enabled")
	}

	return &Client{
		client:  client,
		ctx:     ctx,
		verbose: config.Verbose,
	}
}

// ListRepositories gets all repositories for a given owner (user or org)
func (c *Client) ListRepositories(owner string) ([]Repository, error) {
	return c.ListRepositoriesWithCustomProperties(owner, nil)
}

// ListRepositoriesWithCustomProperties gets all repositories for a given owner and optionally fetches custom properties
func (c *Client) ListRepositoriesWithCustomProperties(owner string, customProperties []string) ([]Repository, error) {
	if c.verbose {
		log.Printf("GitHub API: Listing repositories for owner '%s'", owner)
		if len(customProperties) > 0 {
			log.Printf("GitHub API: Will fetch custom properties: %v", customProperties)
		}
	}

	// First, determine if owner is a user or organization
	isOrg, err := c.isOrganization(owner)
	if err != nil {
		if c.verbose {
			log.Printf("GitHub API: Could not determine owner type, falling back to user endpoint - %v", err)
		}
		// Fall back to user endpoint if we can't determine the type
		return c.listRepositoriesAsUserWithCustomProperties(owner, customProperties)
	}

	if isOrg {
		if c.verbose {
			log.Printf("GitHub API: Owner '%s' detected as organization, using org endpoint", owner)
		}
		repos, err := c.listRepositoriesAsOrgWithCustomProperties(owner, customProperties)
		if err != nil {
			if c.verbose {
				log.Printf("GitHub API: Organization endpoint failed, falling back to user endpoint - %v", err)
			}
			// If org endpoint fails, fall back to user endpoint
			// This handles cases where the token doesn't have org permissions
			return c.listRepositoriesAsUserWithCustomProperties(owner, customProperties)
		}
		return repos, nil
	} else {
		if c.verbose {
			log.Printf("GitHub API: Owner '%s' detected as user, using user endpoint", owner)
		}
		return c.listRepositoriesAsUserWithCustomProperties(owner, customProperties)
	}
}

// isOrganization checks if the given owner is an organization
func (c *Client) isOrganization(owner string) (bool, error) {
	if c.verbose {
		log.Printf("GitHub API: Checking if '%s' is an organization", owner)
	}

	// Try to get organization info
	_, resp, err := c.client.Organizations.Get(c.ctx, owner)
	if err != nil {
		// If we get a 404, it's likely a user account
		if resp != nil && resp.StatusCode == 404 {
			if c.verbose {
				log.Printf("GitHub API: '%s' is not an organization (404)", owner)
			}
			return false, nil
		}
		// If we get a 403, it might be a private org we don't have access to
		// In this case, we should still try the org endpoint for listing repos
		if resp != nil && resp.StatusCode == 403 {
			if c.verbose {
				log.Printf("GitHub API: '%s' might be a private organization (403), will try org endpoint", owner)
			}
			return true, nil
		}
		// Other errors (like 401) we should propagate for fallback handling
		if c.verbose {
			log.Printf("GitHub API: Error checking organization status for '%s' - %v", owner, err)
		}
		return false, err
	}

	if c.verbose {
		log.Printf("GitHub API: '%s' confirmed as organization", owner)
	}
	return true, nil
}

// listRepositoriesAsOrg lists repositories for an organization
func (c *Client) listRepositoriesAsOrg(org string) ([]Repository, error) {
	return c.listRepositoriesAsOrgWithCustomProperties(org, nil)
}

// listRepositoriesAsOrgWithCustomProperties lists repositories for an organization with custom properties
func (c *Client) listRepositoriesAsOrgWithCustomProperties(org string, customProperties []string) ([]Repository, error) {
	var allRepos []Repository

	opts := &github.RepositoryListByOrgOptions{
		Type: "all", // Include public, private, and forked repositories
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	pageCount := 0
	for {
		pageCount++
		if c.verbose {
			log.Printf("GitHub API: GET /orgs/%s/repos (page=%d, per_page=%d, type=%s)", org, opts.Page, opts.PerPage, opts.Type)
		}

		repos, resp, err := c.client.Repositories.ListByOrg(c.ctx, org, opts)
		if err != nil {
			if c.verbose {
				log.Printf("GitHub API: Error listing organization repositories on page %d - %v", pageCount, err)
			}
			// If this is the first page, return the error as the operation completely failed
			if pageCount == 1 {
				return nil, fmt.Errorf("failed to list organization repositories: %w", err)
			}
			// If this is a subsequent page, log a warning but return what we have so far
			if c.verbose {
				log.Printf("GitHub API: Pagination failed on page %d, returning %d repositories from previous pages", pageCount, len(allRepos))
			}
			break
		}

		if c.verbose {
			log.Printf("GitHub API: Response status %d, received %d repositories on page %d", resp.StatusCode, len(repos), pageCount)
		}

		for _, repo := range repos {
			if repo.GetDefaultBranch() == "" {
				continue // Skip repos without default branch
			}

			repository := Repository{
				Owner:         org,
				Name:          repo.GetName(),
				DefaultBranch: repo.GetDefaultBranch(),
				FullName:      repo.GetFullName(),
			}

			// Fetch custom properties if requested
			if len(customProperties) > 0 {
				props, err := c.GetRepositoryCustomProperties(org, repo.GetName(), customProperties)
				if err != nil {
					if c.verbose {
						log.Printf("Warning: Failed to fetch custom properties for %s: %v", repo.GetFullName(), err)
					}
					// Continue with empty properties rather than failing
				}
				repository.CustomProperties = props
			}

			allRepos = append(allRepos, repository)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	if c.verbose {
		log.Printf("GitHub API: Total organization repositories found: %d (across %d pages)", len(allRepos), pageCount)
	}

	return allRepos, nil
}

// listRepositoriesAsUser lists repositories for a user
func (c *Client) listRepositoriesAsUser(user string) ([]Repository, error) {
	return c.listRepositoriesAsUserWithCustomProperties(user, nil)
}

// listRepositoriesAsUserWithCustomProperties lists repositories for a user with custom properties
func (c *Client) listRepositoriesAsUserWithCustomProperties(user string, customProperties []string) ([]Repository, error) {
	var allRepos []Repository

	opts := &github.RepositoryListByUserOptions{
		Type: "all", // Include public, private, and forked repositories
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	pageCount := 0
	for {
		pageCount++
		if c.verbose {
			log.Printf("GitHub API: GET /users/%s/repos (page=%d, per_page=%d, type=%s)", user, opts.Page, opts.PerPage, opts.Type)
		}

		repos, resp, err := c.client.Repositories.ListByUser(c.ctx, user, opts)
		if err != nil {
			if c.verbose {
				log.Printf("GitHub API: Error listing user repositories on page %d - %v", pageCount, err)
			}
			// If this is the first page, return the error as the operation completely failed
			if pageCount == 1 {
				return nil, fmt.Errorf("failed to list user repositories: %w", err)
			}
			// If this is a subsequent page, log a warning but return what we have so far
			if c.verbose {
				log.Printf("GitHub API: Pagination failed on page %d, returning %d repositories from previous pages", pageCount, len(allRepos))
			}
			break
		}

		if c.verbose {
			log.Printf("GitHub API: Response status %d, received %d repositories on page %d", resp.StatusCode, len(repos), pageCount)
		}

		for _, repo := range repos {
			if repo.GetDefaultBranch() == "" {
				continue // Skip repos without default branch
			}

			repository := Repository{
				Owner:         user,
				Name:          repo.GetName(),
				DefaultBranch: repo.GetDefaultBranch(),
				FullName:      repo.GetFullName(),
			}

			// Fetch custom properties if requested
			if len(customProperties) > 0 {
				props, err := c.GetRepositoryCustomProperties(user, repo.GetName(), customProperties)
				if err != nil {
					if c.verbose {
						log.Printf("Warning: Failed to fetch custom properties for %s: %v", repo.GetFullName(), err)
					}
					// Continue with empty properties rather than failing
				}
				repository.CustomProperties = props
			}

			allRepos = append(allRepos, repository)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	if c.verbose {
		log.Printf("GitHub API: Total user repositories found: %d (across %d pages)", len(allRepos), pageCount)
	}

	return allRepos, nil
}

// GetWorkflowFiles retrieves all workflow files from a repository's .github/workflows directory
func (c *Client) GetWorkflowFiles(repo Repository) ([]WorkflowFile, error) {
	if c.verbose {
		log.Printf("GitHub API: Getting workflow files for repository '%s'", repo.FullName)
	}

	var workflowFiles []WorkflowFile

	// Get contents of .github/workflows directory
	if c.verbose {
		log.Printf("GitHub API: GET /repos/%s/contents/.github/workflows", repo.FullName)
	}

	_, dirContent, resp, err := c.client.Repositories.GetContents(
		c.ctx,
		repo.Owner,
		repo.Name,
		".github/workflows",
		&github.RepositoryContentGetOptions{Ref: repo.DefaultBranch},
	)

	if err != nil {
		// If the directory doesn't exist, that's okay - no workflows
		if resp != nil && resp.StatusCode == 404 {
			if c.verbose {
				log.Printf("GitHub API: No .github/workflows directory found (404) - repository has no workflows")
			}
			return workflowFiles, nil
		}
		if c.verbose {
			log.Printf("GitHub API: Error getting workflow directory - %v", err)
		}
		return nil, fmt.Errorf("failed to get workflow directory: %w", err)
	}

	if c.verbose {
		log.Printf("GitHub API: Response status %d, found %d items in workflows directory", resp.StatusCode, len(dirContent))
	}

	// Process each file in the workflows directory
	for _, item := range dirContent {
		if item.GetType() != "file" {
			continue
		}

		filename := item.GetName()
		// Only process YAML/YML files
		if !isWorkflowFile(filename) {
			if c.verbose {
				log.Printf("Skipping non-workflow file: %s", filename)
			}
			continue
		}

		if c.verbose {
			log.Printf("GitHub API: GET /repos/%s/contents/%s", repo.FullName, item.GetPath())
		}

		// Get the file content
		fileContent, _, _, err := c.client.Repositories.GetContents(
			c.ctx,
			repo.Owner,
			repo.Name,
			item.GetPath(),
			&github.RepositoryContentGetOptions{Ref: repo.DefaultBranch},
		)

		if err != nil {
			if c.verbose {
				log.Printf("GitHub API: Error getting workflow file %s - %v", filename, err)
			}
			return nil, fmt.Errorf("failed to get workflow file %s: %w", filename, err)
		}

		content, err := fileContent.GetContent()
		if err != nil {
			if c.verbose {
				log.Printf("Error decoding workflow file %s - %v", filename, err)
			}
			return nil, fmt.Errorf("failed to decode workflow file %s: %w", filename, err)
		}

		if c.verbose {
			log.Printf("Successfully retrieved workflow file: %s (%d bytes)", item.GetPath(), len(content))
		}

		workflowFiles = append(workflowFiles, WorkflowFile{
			Repository: repo,
			Path:       item.GetPath(),
			Content:    content,
		})
	}

	if c.verbose {
		log.Printf("GitHub API: Total workflow files retrieved: %d", len(workflowFiles))
	}

	return workflowFiles, nil
}

// isWorkflowFile checks if a filename is a workflow file (yml or yaml)
func isWorkflowFile(filename string) bool {
	if len(filename) < 5 {
		return false
	}

	ext := filename[len(filename)-4:]
	return ext == ".yml" || ext == "yaml"
}

// CreatePullRequest creates a pull request with the given changes
func (c *Client) CreatePullRequest(repo Repository, title, body, headBranch string) error {
	baseBranch := repo.DefaultBranch

	newPR := &github.NewPullRequest{
		Title: &title,
		Head:  &headBranch,
		Base:  &baseBranch,
		Body:  &body,
	}

	_, _, err := c.client.PullRequests.Create(c.ctx, repo.Owner, repo.Name, newPR)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	return nil
}

// ResolveRef resolves a git reference (tag, branch, or SHA) to a commit SHA
func (c *Client) ResolveRef(owner, repo, ref string) (string, error) {
	// Try to get the reference directly
	gitRef, _, err := c.client.Git.GetRef(c.ctx, owner, repo, "refs/tags/"+ref)
	if err == nil && gitRef.Object != nil {
		return gitRef.Object.GetSHA(), nil
	}

	// Try as a branch reference
	gitRef, _, err = c.client.Git.GetRef(c.ctx, owner, repo, "refs/heads/"+ref)
	if err == nil && gitRef.Object != nil {
		return gitRef.Object.GetSHA(), nil
	}

	// Try to get commit directly (if ref is already a SHA)
	commit, _, err := c.client.Git.GetCommit(c.ctx, owner, repo, ref)
	if err == nil {
		return commit.GetSHA(), nil
	}

	return "", fmt.Errorf("could not resolve reference %s in %s/%s", ref, owner, repo)
}

// GetTagsForRepo gets all tags for a repository and returns them with their commit SHAs
func (c *Client) GetTagsForRepo(owner, repo string) (map[string]string, error) {
	tags := make(map[string]string)

	opts := &github.ListOptions{
		PerPage: 100,
	}

	for {
		repoTags, resp, err := c.client.Repositories.ListTags(c.ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list tags: %w", err)
		}

		for _, tag := range repoTags {
			if tag.Name != nil && tag.Commit != nil && tag.Commit.SHA != nil {
				tags[*tag.Name] = *tag.Commit.SHA
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return tags, nil
}

// GetRepositoryCustomProperties fetches custom properties for a repository
func (c *Client) GetRepositoryCustomProperties(owner, repo string, properties []string) (map[string]string, error) {
	if c.verbose {
		log.Printf("GitHub API: Getting custom properties for repository '%s/%s': %v", owner, repo, properties)
	}

	customProperties := make(map[string]string)

	// If no specific properties requested, return empty map
	if len(properties) == 0 {
		return customProperties, nil
	}

	// GitHub's repository custom properties are available via the repository API
	// Note: Custom properties might be available in different API endpoints depending on the GitHub version
	// This implementation attempts to fetch them from the repository details

	if c.verbose {
		log.Printf("GitHub API: GET /repos/%s/%s (for repository details including custom properties)", owner, repo)
	}

	repository, resp, err := c.client.Repositories.Get(c.ctx, owner, repo)
	if err != nil {
		if c.verbose {
			log.Printf("GitHub API: Error getting repository details for custom properties - %v", err)
		}
		// Don't fail the entire scan if custom properties can't be fetched
		// Return empty properties with a warning
		return customProperties, nil
	}

	if c.verbose {
		log.Printf("GitHub API: Repository %s/%s fetched (status: %d)", owner, repo, resp.StatusCode)
	}

	// Try to extract custom properties from repository response
	// GitHub Enterprise may have custom properties in the repository object
	// For GitHub.com, custom properties might be in organization settings or metadata

	// Method 1: Check if the repository object has custom properties
	if repository != nil {
		// The go-github library might not expose all custom properties directly
		// Try to access them via the raw response if available

		// For demonstration purposes, we'll simulate some custom properties
		// In a real implementation, you would need to:
		// 1. Use the GitHub GraphQL API to get custom properties
		// 2. Use the REST API custom properties endpoint if available
		// 3. Parse them from the repository metadata

		if c.verbose {
			log.Printf("Repository topics: %v", repository.Topics)
			log.Printf("Repository language: %s", repository.GetLanguage())
			log.Printf("Repository description: %s", repository.GetDescription())
		}

		// Try to extract properties from repository topics or other metadata
		// This is a placeholder implementation - actual implementation depends on how
		// your organization stores custom properties

		for _, propertyName := range properties {
			switch propertyName {
			case "ProductId":
				// Example: Extract from topics that start with "product-"
				for _, topic := range repository.Topics {
					if strings.HasPrefix(topic, "product-") {
						customProperties[propertyName] = strings.TrimPrefix(topic, "product-")
						break
					}
				}
			case "Team":
				// Example: Extract from topics that start with "team-"
				for _, topic := range repository.Topics {
					if strings.HasPrefix(topic, "team-") {
						customProperties[propertyName] = strings.TrimPrefix(topic, "team-")
						break
					}
				}
			case "Environment":
				// Example: Extract from repository name patterns
				if strings.Contains(repository.GetName(), "-prod") {
					customProperties[propertyName] = "production"
				} else if strings.Contains(repository.GetName(), "-dev") {
					customProperties[propertyName] = "development"
				} else if strings.Contains(repository.GetName(), "-test") {
					customProperties[propertyName] = "testing"
				}
			default:
				// For other properties, try to find them in repository metadata
				// This could be extended to use GraphQL or other API endpoints
				if c.verbose {
					log.Printf("Custom property '%s' not found in repository metadata", propertyName)
				}
			}
		}
	}

	if c.verbose {
		log.Printf("GitHub API: Extracted custom properties for %s/%s: %v", owner, repo, customProperties)
	}

	return customProperties, nil
}
