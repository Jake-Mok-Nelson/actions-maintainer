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
Since no automated tests exist, ALWAYS manually validate your changes with these scenarios:

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
   ls -la bin/actions-maintainer  # Verify ~18MB binary created
   ```

4. **Environment Variable Token Test**:
   ```bash
   GITHUB_TOKEN=fake_token ./bin/actions-maintainer scan --owner actions
   ```

### Expected Results
- Help commands show usage information and exit with code 0
- Missing owner/token errors show clear error messages and exit with code 1  
- Invalid token attempts API call but fails with HTTP 403 error, exits with code 0
- Build produces binary of exactly 18,800,992 bytes on Linux
- No output files are created when API calls fail

### Required Validation Steps
- ALWAYS run `make fmt` and `make lint` before committing changes
- ALWAYS rebuild and test basic functionality after making code changes
- ALWAYS test error cases (missing token, invalid owner, etc.)
- ALWAYS verify binary size is reasonable (~18MB for Linux build)

## Codebase Navigation

### Project Structure
```
cmd/actions-maintainer/    # Main CLI entry point
├── main.go               # CLI parsing and main logic

internal/                 # Internal packages
├── actions/              # Action version management and rules
│   └── manager.go        # Core business logic for issue detection
├── cache/                # SQLite caching with TTL
│   └── sqlite.go         # Database operations
├── github/               # GitHub API client wrapper
│   └── client.go         # Repository and workflow file fetching
├── output/               # JSON output formatting
│   └── json.go           # Result structuring and serialization
├── pr/                   # Pull request creation logic
│   └── creator.go        # Automated PR generation
└── workflow/             # YAML workflow parsing
    └── parser.go         # GitHub Actions workflow analysis
```

### Key Files to Know
- `cmd/actions-maintainer/main.go` - CLI interface and command handling
- `internal/actions/manager.go` - Core rules engine for action analysis
- `internal/workflow/parser.go` - YAML parsing and action extraction
- `internal/github/client.go` - GitHub API integration
- `Makefile` - All build, test, and development commands

### Common Development Patterns
- Action rules are defined in `internal/actions/manager.go` in the `getDefaultRules()` function
- Workflow parsing handles both regular actions (`uses: actions/checkout@v4`) and reusable workflows
- Error handling follows Go conventions with explicit error returns
- JSON output structure is defined in `internal/output/`

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
- Linux binary: ~18.8MB (actions-maintainer)
- macOS Intel: ~18.7MB (actions-maintainer-darwin-amd64)
- macOS ARM: ~18.1MB (actions-maintainer-darwin-arm64)  
- Windows: ~19.2MB (actions-maintainer-windows-amd64.exe)

## Common Issues and Limitations

### Known Working Conditions
- Builds successfully on Linux with Go 1.24.7+
- Requires internet access for GitHub API calls
- SQLite database is created in system temp directory for caching
- Cache TTL is set to 1 hour

### Does Not Work
- No automated tests exist - all validation must be manual
- Cannot function without valid GitHub token
- Rate limited by GitHub API when using invalid/missing tokens
- Install target fails without proper GOPATH set

### Development Workflow
1. Make code changes
2. Run `make fmt` to format code
3. Run `make lint` to check for issues  
4. Run `make build` to compile
5. Test manually with scenarios above
6. If making changes to action rules, test with a real GitHub token and repository

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
- SQLite (included via modernc.org/sqlite)
- GitHub API client (github.com/google/go-github/v65)
- YAML parser (gopkg.in/yaml.v3)
- CLI framework (github.com/tucnak/climax)