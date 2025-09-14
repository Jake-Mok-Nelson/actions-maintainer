package github

import (
	"context"
	"fmt"
	"log"

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
	Owner         string
	Name          string
	DefaultBranch string
	FullName      string
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
	if c.verbose {
		log.Printf("GitHub API: Listing repositories for owner '%s'", owner)
	}

	var allRepos []Repository

	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		if c.verbose {
			log.Printf("GitHub API: GET /users/%s/repos (page=%d, per_page=%d)", owner, opts.Page, opts.PerPage)
		}

		repos, resp, err := c.client.Repositories.List(c.ctx, owner, opts)
		if err != nil {
			if c.verbose {
				log.Printf("GitHub API: Error listing repositories - %v", err)
			}
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}

		if c.verbose {
			log.Printf("GitHub API: Response status %d, received %d repositories", resp.StatusCode, len(repos))
		}

		for _, repo := range repos {
			if repo.GetDefaultBranch() == "" {
				continue // Skip repos without default branch
			}

			allRepos = append(allRepos, Repository{
				Owner:         owner,
				Name:          repo.GetName(),
				DefaultBranch: repo.GetDefaultBranch(),
				FullName:      repo.GetFullName(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	if c.verbose {
		log.Printf("GitHub API: Total repositories found: %d", len(allRepos))
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
