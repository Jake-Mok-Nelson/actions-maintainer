package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v65/github"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API client with our specific functionality
type Client struct {
	client *github.Client
	ctx    context.Context
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
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	return &Client{
		client: client,
		ctx:    ctx,
	}
}

// ListRepositories gets all repositories for a given owner (user or org)
func (c *Client) ListRepositories(owner string) ([]Repository, error) {
	var allRepos []Repository

	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		repos, resp, err := c.client.Repositories.List(c.ctx, owner, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories: %w", err)
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

	return allRepos, nil
}

// GetWorkflowFiles retrieves all workflow files from a repository's .github/workflows directory
func (c *Client) GetWorkflowFiles(repo Repository) ([]WorkflowFile, error) {
	var workflowFiles []WorkflowFile

	// Get contents of .github/workflows directory
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
			return workflowFiles, nil
		}
		return nil, fmt.Errorf("failed to get workflow directory: %w", err)
	}

	// Process each file in the workflows directory
	for _, item := range dirContent {
		if item.GetType() != "file" {
			continue
		}

		filename := item.GetName()
		// Only process YAML/YML files
		if !isWorkflowFile(filename) {
			continue
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
			return nil, fmt.Errorf("failed to get workflow file %s: %w", filename, err)
		}

		content, err := fileContent.GetContent()
		if err != nil {
			return nil, fmt.Errorf("failed to decode workflow file %s: %w", filename, err)
		}

		workflowFiles = append(workflowFiles, WorkflowFile{
			Repository: repo,
			Path:       item.GetPath(),
			Content:    content,
		})
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
