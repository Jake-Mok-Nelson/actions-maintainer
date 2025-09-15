package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v65/github"
)

// TestListRepositories_Organization verifies that the client correctly identifies
// and handles organization repositories
func TestListRepositories_Organization(t *testing.T) {
	// Mock server that simulates GitHub API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/testorg":
			// Return organization info - this identifies it as an org
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"login": "testorg", "type": "Organization"}`))

		case "/orgs/testorg/repos":
			// Return mock organization repositories
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{
					"name": "private-repo",
					"full_name": "testorg/private-repo",
					"default_branch": "main",
					"private": true
				},
				{
					"name": "public-repo", 
					"full_name": "testorg/public-repo",
					"default_branch": "master",
					"private": false
				}
			]`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Create client with custom base URL
	client := github.NewClient(nil)
	client.BaseURL, _ = url.Parse(server.URL + "/")

	githubClient := &Client{
		client:  client,
		ctx:     context.Background(),
		verbose: true,
	}

	// Test listing repositories for organization
	repos, err := githubClient.ListRepositories("testorg")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(repos))
	}

	// Verify both private and public repos are returned
	repoNames := make(map[string]bool)
	for _, repo := range repos {
		repoNames[repo.Name] = true
		if repo.Owner != "testorg" {
			t.Errorf("Expected owner 'testorg', got '%s'", repo.Owner)
		}
	}

	if !repoNames["private-repo"] {
		t.Error("Expected to find private-repo")
	}
	if !repoNames["public-repo"] {
		t.Error("Expected to find public-repo")
	}
}

// TestListRepositories_User verifies that the client correctly identifies
// and handles user repositories
func TestListRepositories_User(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/testuser":
			// Return 404 for org check - this identifies it as a user
			w.WriteHeader(http.StatusNotFound)

		case "/users/testuser/repos":
			// Return mock user repositories
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{
					"name": "user-repo",
					"full_name": "testuser/user-repo",
					"default_branch": "main",
					"private": false
				}
			]`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := github.NewClient(nil)
	client.BaseURL, _ = url.Parse(server.URL + "/")

	githubClient := &Client{
		client:  client,
		ctx:     context.Background(),
		verbose: true,
	}

	repos, err := githubClient.ListRepositories("testuser")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("Expected 1 repository, got %d", len(repos))
	}

	if repos[0].Name != "user-repo" {
		t.Errorf("Expected repo name 'user-repo', got '%s'", repos[0].Name)
	}
}

// TestListRepositories_PrivateOrgFallback verifies that when org endpoint fails,
// it falls back to user endpoint
func TestListRepositories_PrivateOrgFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/privateorg":
			// Return 403 for private org - we detect it's an org but don't have access
			w.WriteHeader(http.StatusForbidden)

		case "/orgs/privateorg/repos":
			// Org repos endpoint fails with 403
			w.WriteHeader(http.StatusForbidden)

		case "/users/privateorg/repos":
			// Fall back to user endpoint - might get some public repos
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{
					"name": "public-repo",
					"full_name": "privateorg/public-repo", 
					"default_branch": "main",
					"private": false
				}
			]`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := github.NewClient(nil)
	client.BaseURL, _ = url.Parse(server.URL + "/")

	githubClient := &Client{
		client:  client,
		ctx:     context.Background(),
		verbose: true,
	}

	repos, err := githubClient.ListRepositories("privateorg")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("Expected 1 repository, got %d", len(repos))
	}

	if repos[0].Name != "public-repo" {
		t.Errorf("Expected repo name 'public-repo', got '%s'", repos[0].Name)
	}
}
