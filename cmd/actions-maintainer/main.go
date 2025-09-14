package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/tucnak/climax"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/actions"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/cache"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/github"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/output"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/pr"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
)

func main() {
	cli := climax.New("actions-maintainer")
	cli.Brief = "GitHub Actions maintenance tool"
	cli.Version = "0.1.0"

	// Main scan command
	scanCmd := climax.Command{
		Name:  "scan",
		Brief: "Scan GitHub repositories for action dependencies",
		Usage: `scan [--owner <owner>] [--output <file>] [--create-prs] [--filter <regex>] [--verbose]`,
		Help:  `Scans all repositories for a GitHub owner, analyzes workflow files, and reports on action dependencies.`,
		Flags: []climax.Flag{
			{
				Name:     "owner",
				Short:    "o",
				Usage:    `--owner <owner>`,
				Help:     `GitHub owner (user or organization) to scan`,
				Variable: true,
			},
			{
				Name:     "output",
				Short:    "f",
				Usage:    `--output <file>`,
				Help:     `Output file for results. Use .json extension for JSON format or .ipynb for Jupyter notebook (default: JSON to stdout)`,
				Variable: true,
			},
			{
				Name:     "create-prs",
				Short:    "p",
				Usage:    `--create-prs`,
				Help:     `Create pull requests for outdated actions`,
				Variable: false,
			},
			{
				Name:     "token",
				Short:    "t",
				Usage:    `--token <token>`,
				Help:     `GitHub personal access token (or set GITHUB_TOKEN env var)`,
				Variable: true,
			},
			{
				Name:     "cache",
				Short:    "c",
				Usage:    `--cache <provider>`,
				Help:     `Cache provider to use (default: memory)`,
				Variable: true,
			},
			{
				Name:     "skip-resolution",
				Short:    "s",
				Usage:    `--skip-resolution`,
				Help:     `Skip version alias resolution and use string matching only`,
				Variable: false,
			},
			{
				Name:     "filter",
				Short:    "r",
				Usage:    `--filter <regex>`,
				Help:     `Regular expression to filter repositories by name (e.g., "jakes-repos-.*")`,
				Variable: true,
			},
			{
				Name:     "verbose",
				Short:    "v",
				Usage:    `--verbose`,
				Help:     `Enable verbose logging for debugging (shows API calls, parsing steps, rule evaluations, and cache operations)`,
				Variable: false,
			},
		},
		Handle: handleScan,
	}

	cli.AddCommand(scanCmd)

	cli.Run()
}

func handleScan(ctx climax.Context) int {
	owner, _ := ctx.Get("owner")
	if owner == "" {
		fmt.Fprintf(os.Stderr, "Error: --owner is required\n")
		return 1
	}

	token, _ := ctx.Get("token")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		fmt.Fprintf(os.Stderr, "Error: GitHub token is required. Use --token or set GITHUB_TOKEN environment variable\n")
		return 1
	}

	outputFile, _ := ctx.Get("output")
	createPRs := ctx.Is("create-prs")
	skipResolution := ctx.Is("skip-resolution")
	filterPattern, _ := ctx.Get("filter")
	verbose := ctx.Is("verbose")

	if verbose {
		log.Printf("Verbose logging enabled")
		log.Printf("Scanning repositories for owner: %s", owner)
	}

	fmt.Printf("Scanning repositories for owner: %s\n", owner)

	// Initialize cache for version resolution (only memory cache is supported)
	cacheProvider, _ := ctx.Get("cache")
	if cacheProvider == "" {
		cacheProvider = "memory"
	}

	if cacheProvider != "memory" {
		fmt.Fprintf(os.Stderr, "Error: Unsupported cache provider '%s'. Only 'memory' is supported.\n", cacheProvider)
		return 1
	}

	cacheInstance := cache.NewMemoryCacheWithConfig(&cache.Config{
		Verbose: verbose,
	})
	fmt.Printf("Using in-memory cache for version resolution\n")
	defer cacheInstance.Close()

	// Clean expired cache entries
	cacheInstance.CleanExpired()

	// Initialize components
	githubClient := github.NewClientWithConfig(token, &github.Config{
		Verbose: verbose,
	})

	// Create version resolver with shared cache
	versionResolver := workflow.NewVersionResolverWithCache(githubClient, skipResolution, cacheInstance)
	actionManager := actions.NewManagerWithResolverAndConfig(versionResolver, &actions.Config{
		Verbose: verbose,
	})

	// Perform scan
	fmt.Printf("Fetching repositories...\n")
	repositories, err := githubClient.ListRepositories(owner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing repositories: %v\n", err)
		return 1
	}

	fmt.Printf("Found %d repositories\n", len(repositories))

	// Apply repository filter if provided
	if filterPattern != "" {
		fmt.Printf("Applying filter pattern: %s\n", filterPattern)
		filterRegex, err := regexp.Compile(filterPattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid filter regex pattern '%s': %v\n", filterPattern, err)
			return 1
		}

		var filteredRepositories []github.Repository
		for _, repo := range repositories {
			if filterRegex.MatchString(repo.Name) {
				filteredRepositories = append(filteredRepositories, repo)
			}
		}

		fmt.Printf("Filtered repositories: %d/%d match pattern\n", len(filteredRepositories), len(repositories))
		repositories = filteredRepositories
	}

	var repositoryResults []output.RepositoryResult

	// Scan each repository
	for i, repo := range repositories {
		fmt.Printf("Scanning repository %d/%d: %s\n", i+1, len(repositories), repo.FullName)

		// Get workflow files
		workflowFiles, err := githubClient.GetWorkflowFiles(repo)
		if err != nil {
			fmt.Printf("Warning: Failed to get workflow files for %s: %v\n", repo.FullName, err)
			continue
		}

		if len(workflowFiles) == 0 {
			fmt.Printf("  No workflow files found\n")
			continue
		}

		fmt.Printf("  Found %d workflow files\n", len(workflowFiles))

		var repoActions []workflow.ActionReference
		var workflowFileResults []output.WorkflowFileResult

		// Parse each workflow file
		for _, wf := range workflowFiles {
			if verbose {
				log.Printf("Parsing workflow file: %s", wf.Path)
			}
			actions, err := workflow.ParseWorkflowWithConfig(wf.Content, wf.Path, repo.FullName, &workflow.Config{
				Verbose: verbose,
			})
			if err != nil {
				fmt.Printf("  Warning: Failed to parse %s: %v\n", wf.Path, err)
				continue
			}

			fmt.Printf("    %s: %d actions\n", wf.Path, len(actions))

			repoActions = append(repoActions, actions...)
			workflowFileResults = append(workflowFileResults, output.WorkflowFileResult{
				Path:        wf.Path,
				ActionCount: len(actions),
				Actions:     actions,
			})
		}

		// Analyze actions for issues
		if verbose {
			log.Printf("Starting analysis of %d total actions for repository %s", len(repoActions), repo.FullName)
		}
		issues := actionManager.AnalyzeActions(repoActions)

		if len(issues) > 0 {
			fmt.Printf("  Found %d issues\n", len(issues))
			if verbose {
				for _, issue := range issues {
					log.Printf("Issue found: %s@%s - %s (severity: %s)", issue.Repository, issue.CurrentVersion, issue.IssueType, issue.Severity)
				}
			}
		}

		repositoryResults = append(repositoryResults, output.RepositoryResult{
			Name:          repo.Name,
			FullName:      repo.FullName,
			DefaultBranch: repo.DefaultBranch,
			WorkflowFiles: workflowFileResults,
			Actions:       repoActions,
			Issues:        issues,
		})
	}

	// Build final scan result
	scanResult := output.BuildScanResult(owner, repositoryResults)

	// Output results
	var outputWriter *os.File
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			return 1
		}
		defer file.Close()
		outputWriter = file
	} else {
		outputWriter = os.Stdout
	}

	// Determine output format based on file extension
	isNotebook := strings.HasSuffix(strings.ToLower(outputFile), ".ipynb")

	if isNotebook {
		if err := output.FormatNotebook(scanResult, outputWriter); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting notebook output: %v\n", err)
			return 1
		}
	} else {
		if err := output.FormatJSON(scanResult, outputWriter, true); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON output: %v\n", err)
			return 1
		}
	}

	// Create PRs if requested
	if createPRs {
		fmt.Printf("\nCreating pull requests for updates...\n")
		prCreator := pr.NewCreator(githubClient)
		updatePlans := pr.PlanUpdates(repositoryResults)

		if len(updatePlans) == 0 {
			fmt.Printf("No updates needed - all actions are up to date!\n")
		} else {
			fmt.Printf("Planning updates for %d repositories\n", len(updatePlans))
			createdPRs, err := prCreator.CreateUpdatePRs(updatePlans)
			if err != nil {
				fmt.Printf("Warning: Some PRs failed to create: %v\n", err)
			}

			// Add created PRs to scan result
			for _, createdPR := range createdPRs {
				output.AddCreatedPR(scanResult, createdPR)
			}
		}
	}

	// Finalize scan result with timing
	output.FinalizeScanResult(scanResult)

	// Print summary
	fmt.Printf("\nScan complete!\n")
	fmt.Printf("- Repositories scanned: %d\n", scanResult.Summary.TotalRepositories)
	fmt.Printf("- Workflow files found: %d\n", scanResult.Summary.TotalWorkflowFiles)
	fmt.Printf("- Actions analyzed: %d\n", scanResult.Summary.TotalActions)
	fmt.Printf("- Unique actions: %d\n", len(scanResult.Summary.UniqueActions))

	totalIssues := 0
	for _, count := range scanResult.Summary.IssuesByType {
		totalIssues += count
	}
	fmt.Printf("- Issues found: %d\n", totalIssues)

	if outputFile != "" {
		fmt.Printf("- Results saved to: %s\n", outputFile)
	}

	// Print cache statistics
	if stats, err := cacheInstance.GetStats(); err == nil {
		fmt.Printf("- Cache entries: %v total, %v valid\n", stats["total_entries"], stats["valid_entries"])
	}

	return 0
}
