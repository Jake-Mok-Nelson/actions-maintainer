# actions-maintainer

The actions-maintainer is a Go CLI tool that identifies and helps resolve TOIL around migration and updating GitHub Actions workflows. It scans GitHub repositories, analyzes workflow files, and identifies outdated, deprecated, or vulnerable action versions.

Always reference these instructions first and only fallback to search or bash commands when you encounter unexpected information that does not match what is documented here.

## Working Effectively

### Bootstrap and Build the Repository
- Ensure Go 1.24.7+ is installed (check with `go version`)
- Clone the repository and navigate to the project root
- Download dependencies: `make deps` -- takes 18 seconds. NEVER CANCEL. Set timeout to 60+ seconds.
- Format code: `make fmt` -- takes 0.3 seconds
- Run linter: `make lint` -- takes 8 seconds. NEVER CANCEL. Set timeout to 30+ seconds.
- Run tests: `make test` -- takes 8 seconds (no test files exist yet, this is expected)
- Build binary: `make build` -- takes 39 seconds. NEVER CANCEL. Set timeout to 90+ seconds.

### Complete Development Workflow
- Run full pipeline: `make all` -- takes 1.4 seconds when dependencies already downloaded. NEVER CANCEL. Set timeout to 120+ seconds for first run.
- Build for all platforms: `make build-all` -- takes 98 seconds on first run, ~3 seconds on subsequent runs due to Go build cache. NEVER CANCEL. Set timeout to 180+ seconds.

### Run the Application
- After building, the binary is available at `./bin/actions-maintainer`
- Show help: `./bin/actions-maintainer`
- Show scan command help: `./bin/actions-maintainer help scan`
- Basic scan (requires GitHub token): `./bin/actions-maintainer scan --owner <owner> --token <token>`
- Save output to file: `./bin/actions-maintainer scan --owner <owner> --token <token> --output results.json`
- Save as Jupyter notebook: `./bin/actions-maintainer scan --owner <owner> --token <token> --output results.ipynb`
- Create PRs for updates: `./bin/actions-maintainer scan --owner <owner> --token <token> --create-prs`
- Filter repositories: `./bin/actions-maintainer scan --owner <owner> --token <token> --filter "my-repos-.*"`
- Use custom rules: `./bin/actions-maintainer scan --owner <owner> --token <token> --rules-file custom-rules.json`
- Enable verbose logging: `./bin/actions-maintainer scan --owner <owner> --token <token> --verbose`
- Target only workflows: `./bin/actions-maintainer scan --owner <owner> --token <token> --workflow-only`
- Skip version resolution: `./bin/actions-maintainer scan --owner <owner> --token <token> --skip-resolution`

### Install Binary
- Set GOPATH if not set: `export GOPATH=/home/runner/go`
- Create bin directory: `mkdir -p $GOPATH/bin`
- Install: `GOPATH=/home/runner/go make install`
- Verify installation: `ls -la $GOPATH/bin/actions-maintainer`

## Authentication Requirements
- The tool requires a GitHub personal access token for operation
- Pass token via `--token` flag OR set `GITHUB_TOKEN` environment variable
- Without a valid token, the tool will fail with error: "GitHub token is required"
- Invalid tokens result in HTTP 403 errors from GitHub API

## Validation and Testing

### Manual Validation Scenarios
Since automated tests exist but may not cover all functionality, ALWAYS manually validate your changes with these scenarios:

1. **Help System Test**:
   ```bash
   ./bin/actions-maintainer
   ./bin/actions-maintainer help scan
   ```

2. **Error Handling Test**:
   ```bash
   # Test missing owner
   ./bin/actions-maintainer scan --token fake_token
   
   # Test missing token  
   ./bin/actions-maintainer scan --owner actions
   
   # Test invalid token
   ./bin/actions-maintainer scan --owner actions --token fake_token
   ```

3. **Build Validation**:
   ```bash
   make clean
   make all
   ls -la bin/actions-maintainer  # Verify ~15MB binary created
   ```

4. **Environment Variable Token Test**:
   ```bash
   GITHUB_TOKEN=fake_token ./bin/actions-maintainer scan --owner actions
   ```

5. **Output Format Tests**:
   ```bash
   # Test JSON output (default)
   ./bin/actions-maintainer scan --owner actions --token fake_token --output results.json
   
   # Test Jupyter notebook output
   ./bin/actions-maintainer scan --owner actions --token fake_token --output results.ipynb
   ```

6. **CLI Option Tests**:
   ```bash
   # Test repository filtering
   ./bin/actions-maintainer scan --owner actions --token fake_token --filter "checkout.*"
   
   # Test verbose mode
   ./bin/actions-maintainer scan --owner actions --token fake_token --verbose
   
   # Test workflow-only mode
   ./bin/actions-maintainer scan --owner actions --token fake_token --workflow-only
   
   # Test skip resolution
   ./bin/actions-maintainer scan --owner actions --token fake_token --skip-resolution
   ```

7. **Transformation Testing** (requires valid token):
   ```bash
   # Test with custom rules file
   echo '{"version_rules": []}' > test-rules.json
   ./bin/actions-maintainer scan --owner actions --token real_token --rules-file test-rules.json
   
   # Test PR creation (dry run simulation)
   ./bin/actions-maintainer scan --owner actions --token fake_token --create-prs
   ```

### Expected Results
- Help commands show usage information and exit with code 0
- Missing owner/token errors show clear error messages and exit with code 1  
- Invalid token attempts API call but fails with HTTP 403 error, exits with code 0
- Build produces binary of approximately 15,486,463 bytes on Linux (~15MB)
- No output files are created when API calls fail
- Tests pass: `make test` should show passing tests for actions, github, patcher, and workflow packages
- JSON output format works when .json extension specified
- Jupyter notebook format works when .ipynb extension specified  
- CLI options are properly validated (missing required parameters show errors)
- Verbose mode provides additional logging output
- Repository filtering with regex patterns works as expected

### Required Validation Steps
- ALWAYS run `make fmt` and `make lint` before committing changes
- ALWAYS run `make test` to ensure all tests pass after code changes
- ALWAYS rebuild and test basic functionality after making code changes
- ALWAYS test error cases (missing token, invalid owner, etc.)
- ALWAYS verify binary size is reasonable (~15MB for Linux build)

## Codebase Navigation

### Project Structure
```
main.go                    # Main CLI entry point and command handling

internal/                 # Internal packages
├── actions/              # Action version management and rules
│   └── manager.go        # Core business logic for issue detection
├── cache/                # In-memory caching with TTL
│   ├── interface.go      # Cache interface definition
│   └── memory.go         # Memory-based cache implementation
├── github/               # GitHub API client wrapper
│   └── client.go         # Repository and workflow file fetching
├── output/               # Multiple output format support
│   ├── json.go           # JSON output formatting
│   └── notebook.go       # Jupyter notebook output formatting
├── patcher/              # Action transformation and migration
│   ├── patch.go          # Core patching logic and data structures
│   ├── rules.go          # Default transformation rules
│   └── integration.go    # Workflow content patching integration
├── pr/                   # Pull request creation logic
│   └── creator.go        # Automated PR generation
└── workflow/             # YAML workflow parsing and version resolution
    ├── parser.go         # GitHub Actions workflow analysis
    └── resolver.go       # Version alias resolution and caching
```

### Key Files to Know
- `main.go` - CLI interface and command handling (at project root)
- `internal/actions/manager.go` - Core rules engine for action analysis
- `internal/workflow/parser.go` - YAML parsing and action extraction
- `internal/workflow/resolver.go` - Version alias resolution and caching
- `internal/patcher/patch.go` - Action transformation and migration logic
- `internal/patcher/rules.go` - Default transformation rules for common actions
- `internal/github/client.go` - GitHub API integration
- `internal/output/json.go` - JSON output formatting
- `internal/output/notebook.go` - Jupyter notebook output formatting
- `Makefile` - All build, test, and development commands

### Common Development Patterns
- Action rules are defined in `internal/actions/manager.go` in the `getDefaultRules()` function
- Action transformation rules are defined in `internal/patcher/rules.go` in the `loadDefaultRules()` function
- Workflow parsing handles both regular actions (`uses: actions/checkout@v4`) and reusable workflows
- Version resolution with caching is handled by `internal/workflow/resolver.go` 
- Multiple output formats are supported: JSON (default) and Jupyter notebook (.ipynb)
- Error handling follows Go conventions with explicit error returns
- JSON output structure is defined in `internal/output/json.go`
- Pull request creation logic is in `internal/pr/creator.go`

## Build System Details

### Make Targets
- `make help` - Show all available commands
- `make deps` - Download and tidy Go dependencies
- `make build` - Build single Linux binary to `./bin/actions-maintainer`
- `make build-all` - Cross-compile for Linux, macOS (Intel/ARM), and Windows
- `make test` - Run Go tests (currently no test files exist)
- `make fmt` - Format all Go source files
- `make lint` - Run `go vet` linter
- `make clean` - Remove build artifacts and binaries
- `make install` - Copy binary to `$GOPATH/bin` (requires GOPATH to be set)
- `make run` - Build and run with no arguments (shows help)
- `make all` - Complete pipeline: clean, deps, fmt, lint, test, build

### Binary Output Details
- Linux binary: ~15MB (actions-maintainer)
- macOS Intel: ~15MB (actions-maintainer-darwin-amd64)
- macOS ARM: ~14.5MB (actions-maintainer-darwin-arm64)  
- Windows: ~15.5MB (actions-maintainer-windows-amd64.exe)

## Action Transformation and Migration

The `patcher/` package provides sophisticated transformation capabilities for GitHub Actions upgrades and migrations:

### Core Capabilities
- **Version Transformations**: Automated parameter changes during action version upgrades
- **Repository Migrations**: Actions moving between organizations or repository names
- **Parameter Transformations**: Adding, removing, renaming, or modifying action parameters
- **Workflow Content Patching**: Safe modification of YAML workflow files with change tracking

### Transformation Types

1. **Parameter Addition**: Add new required or recommended parameters during upgrades
   ```yaml
   # Example: actions/checkout v3 → v4 adds fetch-depth parameter
   - uses: actions/checkout@v4
     with:
       fetch-depth: 1  # Added automatically
   ```

2. **Parameter Removal**: Remove deprecated parameters no longer supported
   ```yaml
   # Example: actions/upload-artifact v3 → v4 removes path-separator
   - uses: actions/upload-artifact@v4
     with:
       name: artifacts
       # path-separator: removed automatically
   ```

3. **Parameter Renaming**: Rename parameters that changed between versions
   ```yaml
   # Example: actions/setup-node v2 → v4 renames version to node-version
   - uses: actions/setup-node@v4
     with:
       node-version: '18'  # Renamed from 'version'
   ```

4. **Repository Migration**: Move actions to new repository locations
   ```yaml
   # Before migration
   - uses: legacy-org/deprecated-action@v1
     with:
       old-param: value
   
   # After migration
   - uses: modern-org/recommended-action@v2
     with:
       new-param: value
       migrate-notice: "Migration tracking comment"
   ```

### Built-in Transformation Rules
The patcher includes default rules for popular GitHub Actions:
- `actions/checkout`: v1/v2/v3 → v4 transformations
- `actions/setup-node`: v2 → v4 parameter renaming and caching
- `actions/upload-artifact`: v2/v3 → v4 with new artifact backend
- `actions/download-artifact`: v2/v3 → v4 with enhanced capabilities
- `actions/cache`: v2/v3 → v4 with improved dependency handling

### Custom Transformation Rules
You can define custom transformation rules using the `--rules-file` option:
```json
{
  "version_rules": [
    {
      "repository": "my-org/custom-action",
      "current_version": "v1.*",
      "recommended_version": "v2.0.0",
      "severity": "high"
    }
  ],
  "migration_rules": [
    {
      "from_repository": "old-org/action",
      "to_repository": "new-org/action",
      "to_version": "v2.0.0"
    }
  ]
}
```

### Integration with Pull Request Creation
The patcher integrates with the PR creation system to:
- Apply transformations automatically when creating PRs
- Document all changes made in PR descriptions
- Preserve action subpaths during migrations
- Track transformation rationale for review

## Common Issues and Limitations

### Known Working Conditions
- Builds successfully on Linux with Go 1.24.7+
- Requires internet access for GitHub API calls
- In-memory cache is used for version resolution with TTL (default 1 hour)
- Cache TTL is configurable and supports verbose logging

### Does Not Work
- Cannot function without valid GitHub token
- Rate limited by GitHub API when using invalid/missing tokens
- Install target fails without proper GOPATH set

### Development Workflow
1. Make code changes
2. Run `make fmt` to format code
3. Run `make lint` to check for issues  
4. Run `make test` to run automated tests
5. Run `make build` to compile
6. Test manually with scenarios above
7. If making changes to action rules, test with a real GitHub token and repository

### Expected Timing (NEVER CANCEL - Use Specified Timeouts)
- Initial `make deps`: 18 seconds (60+ second timeout)
- `make build`: 39 seconds (90+ second timeout) 
- `make build-all`: 98 seconds first run, ~3 seconds subsequent runs (180+ second timeout)
- `make lint`: 8 seconds (30+ second timeout)
- `make all` (after deps): 1.4 seconds (120+ second timeout for safety)

## External Dependencies
- Go 1.24.7+ (verified working version)
- Internet access for GitHub API
- GitHub personal access token for functionality
- GitHub API client (github.com/google/go-github/v65)
- YAML parser (gopkg.in/yaml.v3)
- CLI framework (github.com/tucnak/climax)