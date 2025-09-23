package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

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
		Usage: `scan [--owner <owner>] [--output <file>] [--filter <regex>] [--verbose]`,
		Help:  `Scans all repositories for a GitHub owner, analyzes workflow files, and outputs JSON results.`,
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
				Short:    "O",
				Usage:    `--output <file>`,
				Help:     `Output file for scan results. Use .json extension for JSON format or .ipynb for Jupyter notebook (default: JSON to stdout)`,
				Variable: true,
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
			{
				Name:     "rules-file",
				Short:    "R",
				Usage:    `--rules-file <file>`,
				Help:     `Path to custom rules file (JSON format). Rules will be merged with defaults. Supports version rules and repository migrations`,
				Variable: true,
			},
			{
				Name:     "custom-property",
				Short:    "P",
				Usage:    `--custom-property <property>`,
				Help:     `Custom repository property to include in the report (e.g., "ProductId"). Can be specified multiple times for multiple properties`,
				Variable: true,
			},
		},
		Handle: handleScan,
	}

	cli.AddCommand(scanCmd)

	// Report command
	reportCmd := climax.Command{
		Name:  "report",
		Brief: "Generate formatted reports from scan JSON results",
		Usage: `report [--input <file>] [--output <file>] [--format <format>]`,
		Help:  `Generates formatted reports from JSON scan results. Input can be a file or stdin. Supports JSON and Jupyter notebook output formats.`,
		Flags: []climax.Flag{
			{
				Name:     "input",
				Short:    "i",
				Usage:    `--input <file>`,
				Help:     `JSON input file from scan command (default: read from stdin)`,
				Variable: true,
			},
			{
				Name:     "output",
				Short:    "o",
				Usage:    `--output <file>`,
				Help:     `Output file for formatted report. Use .json extension for JSON format or .ipynb for Jupyter notebook (default: JSON to stdout)`,
				Variable: true,
			},
		},
		Handle: handleReport,
	}

	cli.AddCommand(reportCmd)

	// Create-PR command
	createPRCmd := climax.Command{
		Name:  "create-pr",
		Brief: "Create pull requests from scan results",
		Usage: `create-pr [--input <file>] [--template <file>] [--token <token>]`,
		Help:  `Creates pull requests for action updates from scan results. Input can be a file or stdin. Supports custom Go templates for PR body generation.`,
		Flags: []climax.Flag{
			{
				Name:     "input",
				Short:    "i",
				Usage:    `--input <file>`,
				Help:     `JSON input file from scan command (default: read from stdin)`,
				Variable: true,
			},
			{
				Name:     "template",
				Short:    "T",
				Usage:    `--template <file>`,
				Help:     `Go template file for PR body generation. Template receives TemplateData with Repository, Updates, UpdateCount, and grouped update lists`,
				Variable: true,
			},
			{
				Name:     "token",
				Short:    "t",
				Usage:    `--token <token>`,
				Help:     `GitHub personal access token (or set GITHUB_TOKEN env var)`,
				Variable: true,
			},
		},
		Handle: handleCreatePR,
	}

	cli.AddCommand(createPRCmd)

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
	skipResolution := ctx.Is("skip-resolution")
	filterPattern, _ := ctx.Get("filter")
	verbose := ctx.Is("verbose")
	rulesFile, _ := ctx.Get("rules-file")
	customProperty, _ := ctx.Get("custom-property")

	// Parse custom properties (support multiple values separated by commas)
	var customProperties []string
	if customProperty != "" {
		// Split by comma and trim whitespace
		parts := strings.Split(customProperty, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				customProperties = append(customProperties, trimmed)
			}
		}
	}

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

	// Load custom rules if provided
	var customRules []actions.Rule
	if rulesFile != "" {
		if verbose {
			log.Printf("Loading custom rules from file: %s", rulesFile)
		}
		var err error
		customRules, err = loadRulesFromFile(rulesFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading rules file '%s': %v\n", rulesFile, err)
			return 1
		}
		fmt.Printf("Loaded %d custom rules from %s\n", len(customRules), rulesFile)
	}

	actionManager := actions.NewManagerWithResolverConfigAndRules(versionResolver, &actions.Config{
		Verbose: verbose,
	}, customRules)

	// Perform scan
	fmt.Printf("Fetching repositories...\n")

	// First, get basic repository list without custom properties
	repositories, err := githubClient.ListRepositories(owner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing repositories: %v\n", err)
		return 1
	}

	fmt.Printf("Found %d repositories\n", len(repositories))

	// Add helpful information about potential pagination limitations
	if len(repositories) > 0 && len(repositories)%100 == 0 {
		fmt.Printf("Note: Repository count is a multiple of 100. If you expected more repositories, check the verbose logs for any pagination errors.\n")
	}

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

	// Now fetch custom properties only for filtered repositories
	if len(customProperties) > 0 {
		fmt.Printf("Fetching custom properties for %d repositories: %v\n", len(repositories), customProperties)
		for i := range repositories {
			props, err := githubClient.GetRepositoryCustomProperties(repositories[i].Owner, repositories[i].Name, customProperties)
			if err != nil {
				if verbose {
					log.Printf("Warning: Failed to fetch custom properties for %s: %v", repositories[i].FullName, err)
				}
				// Continue with empty properties rather than failing
			}
			repositories[i].CustomProperties = props
		}
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
			Name:             repo.Name,
			FullName:         repo.FullName,
			DefaultBranch:    repo.DefaultBranch,
			WorkflowFiles:    workflowFileResults,
			Actions:          repoActions,
			Issues:           issues,
			CustomProperties: repo.CustomProperties,
		})
	}

	// Build final scan result
	scanResult := output.BuildScanResult(owner, repositoryResults)

	// Finalize scan result with timing
	output.FinalizeScanResult(scanResult)

	// Set up output writer
	var outputWriter io.Writer
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

	return 0
}

func handleReport(ctx climax.Context) int {
	inputFile, _ := ctx.Get("input")
	outputFile, _ := ctx.Get("output")

	// Read JSON input
	var inputReader io.Reader
	if inputFile != "" {
		file, err := os.Open(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
			return 1
		}
		defer file.Close()
		inputReader = file
	} else {
		inputReader = os.Stdin
	}

	// Parse JSON input
	data, err := io.ReadAll(inputReader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return 1
	}

	var scanResult output.ScanResult
	if err := json.Unmarshal(data, &scanResult); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON input: %v\n", err)
		return 1
	}

	// Set up output writer
	var outputWriter io.Writer
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
		if err := output.FormatNotebook(&scanResult, outputWriter); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting notebook output: %v\n", err)
			return 1
		}
	} else {
		if err := output.FormatJSON(&scanResult, outputWriter, true); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON output: %v\n", err)
			return 1
		}
	}

	return 0
}

func handleCreatePR(ctx climax.Context) int {
	inputFile, _ := ctx.Get("input")
	templateFile, _ := ctx.Get("template")

	token, _ := ctx.Get("token")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		fmt.Fprintf(os.Stderr, "Error: GitHub token is required. Use --token or set GITHUB_TOKEN environment variable\n")
		return 1
	}

	// Read JSON input
	var inputReader io.Reader
	if inputFile != "" {
		file, err := os.Open(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
			return 1
		}
		defer file.Close()
		inputReader = file
	} else {
		inputReader = os.Stdin
	}

	// Parse JSON input
	data, err := io.ReadAll(inputReader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return 1
	}

	var scanResult output.ScanResult
	if err := json.Unmarshal(data, &scanResult); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON input: %v\n", err)
		return 1
	}

	// Create GitHub client
	githubClient := github.NewClient(token)

	// Load custom template if provided
	var prCreator *pr.Creator
	if templateFile != "" {
		tmpl, err := loadTemplateFromFile(templateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading template file: %v\n", err)
			return 1
		}
		prCreator = pr.NewCreatorWithTemplate(githubClient, tmpl)
	} else {
		prCreator = pr.NewCreator(githubClient)
	}

	// Plan updates from scan result
	updatePlans := pr.PlanUpdates(scanResult.Repositories)

	if len(updatePlans) == 0 {
		fmt.Printf("No updates needed - all actions are up to date!\n")
		return 0
	}

	fmt.Printf("Creating pull requests for updates...\n")
	fmt.Printf("Planning updates for %d repositories\n", len(updatePlans))

	createdPRs, err := prCreator.CreateUpdatePRs(updatePlans)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating PRs: %v\n", err)
		return 1
	}

	// Output created PRs information
	for _, createdPR := range createdPRs {
		fmt.Printf("Created PR for %s: %s\n", createdPR.Repository, createdPR.URL)
	}

	fmt.Printf("Successfully created %d pull requests\n", len(createdPRs))
	return 0
}

// loadTemplateFromFile loads a Go template from a file
func loadTemplateFromFile(filename string) (*template.Template, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	tmpl, err := template.New("pr-body").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return tmpl, nil
}

// loadRulesFromFile loads custom rules from a JSON file
func loadRulesFromFile(filename string) ([]actions.Rule, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open rules file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read rules file: %w", err)
	}

	var rules []actions.Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("unable to parse rules file as JSON: %w", err)
	}

	// Validate rules
	for i, rule := range rules {
		if rule.Repository == "" {
			return nil, fmt.Errorf("rule %d: repository field is required", i+1)
		}

		// Check if this is a migration rule or a standard version rule
		isMigrationRule := rule.MigrateToRepository != "" || rule.MigrateToVersion != ""

		if isMigrationRule {
			// Migration rule validation
			if rule.MigrateToRepository == "" {
				return nil, fmt.Errorf("rule %d: migrate_to_repository field is required when migration is specified for repository %s", i+1, rule.Repository)
			}
			if rule.MigrateToVersion == "" {
				return nil, fmt.Errorf("rule %d: migrate_to_version field is required when migration is specified for repository %s", i+1, rule.Repository)
			}
			// For migration rules, latest_version is optional (defaults to current behavior)
		} else {
			// Standard version rule validation
			if rule.LatestVersion == "" {
				return nil, fmt.Errorf("rule %d: latest_version field is required for repository %s", i+1, rule.Repository)
			}
		}
	}

	return rules, nil
}
