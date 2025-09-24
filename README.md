# actions-maintainer

A Go CLI tool that identifies and helps resolve TOIL around migration and updating GitHub Actions workflows.


---
**NOTE**

This project is in early development. The CLI and output format may change in future releases.
There may be bugs and incomplete features.

---



## Features

- 🔍 **Repository Scanning**: Automatically scans all repositories for a GitHub owner/organization
- 📋 **Workflow Analysis**: Parses `.github/workflows/*.yml` files to extract action dependencies
- ⚡ **Version Management**: Identity actions and workflows that need updating based on rules
- 🏗️ **Location Migration**: Complete migration support from detection to automated PR creation for actions moving repositories
- 📊 **Detailed Reporting**: Comprehensive JSON output with statistics and issue summaries
- 🔧 **Automated Updates**: Creates pull requests with safe version updates and repository migrations

## Installation

### Option 1: Install from source with go install (recommended)

```bash
go install github.com/Jake-Mok-Nelson/actions-maintainer@latest
```

This will install the latest released version to your `$GOPATH/bin` directory.

### Option 2: Build from source

```bash
git clone https://github.com/Jake-Mok-Nelson/actions-maintainer
cd actions-maintainer
make build
# Binary will be available at ./bin/actions-maintainer
```

### Option 3: Download pre-built binaries

Download the latest binaries from the [releases page](https://github.com/Jake-Mok-Nelson/actions-maintainer/releases).

## Usage

### Basic Scanning

Scan all repositories for a GitHub user or organization:

```bash
./actions-maintainer scan --owner github --token YOUR_GITHUB_TOKEN
```

### Save Results to File

```bash
./actions-maintainer scan --owner my-org --token YOUR_GITHUB_TOKEN --output results.json
```

### Create Pull Requests for Updates

Create automated pull requests for all detected action updates and migrations:

```bash
./actions-maintainer scan --owner my-org --token YOUR_GITHUB_TOKEN --create-prs
```

This will:
- Create separate PRs for each repository with action updates
- Handle both version updates and repository migrations in the same PR
- Include detailed descriptions with migration reasoning
- Apply any necessary parameter transformations during migrations

### Using Environment Variable for Token

```bash
export GITHUB_TOKEN=your_token_here
./actions-maintainer scan --owner my-org
```

## Authentication

You need a GitHub personal access token with the following permissions:
- `repo` (for accessing private repositories)
- `public_repo` (for accessing public repositories)

Set the token using:
- `--token` flag
- `GITHUB_TOKEN` environment variable

### Organization vs User Repository Access

The tool automatically detects whether the target is a GitHub user or organization and uses the appropriate API endpoints:

- **Organizations**: Uses `/orgs/{org}/repos` endpoint with `type=all` to include private repositories
- **Users**: Uses `/users/{user}/repos` endpoint with `type=all` to include private repositories
- **Fallback behavior**: If organization endpoint fails due to permissions, automatically falls back to user endpoint

For **private organizations**, ensure your token has:
- Organization membership or appropriate repository access permissions
- If you only have access to specific repositories within an organization, the tool will gracefully fall back to the user endpoint which may show fewer repositories

Use `--verbose` flag to see which endpoints are being used and troubleshoot access issues.

## Output Format

The tool outputs detailed JSON with the following structure:

```json
{
  "owner": "github",
  "scan_time": "2024-01-15T10:30:00Z",
  "repositories": [
    {
      "name": "repo-name",
      "full_name": "github/repo-name",
      "workflow_files": [...],
      "actions": [...],
      "issues": [...]
    }
  ],
  "summary": {
    "total_repositories": 50,
    "total_workflow_files": 125,
    "total_actions": 300,
    "unique_actions": {...},
    "issues_by_type": {...},
    "top_issues": [...]
  }
}
```

## Supported Issue Types

- **Outdated**: Action versions that are behind the latest release
- **Deprecated**: Action versions that are no longer supported
- **Migration**: Actions that have moved to new repository locations
- **Security**: Action versions with known security vulnerabilities

## Version Alias Resolution

actions-maintainer supports intelligent version alias resolution to handle scenarios where different version references point to the same underlying commit:

### Example Scenarios
- `v1` tag and `v1.2.4` tag point to the same commit SHA
- `v4` tag and commit SHA `abc123def456` reference the same commit
- When `v1` is updated to point to a new release, aliases are automatically detected

### Resolution Modes

**Default (with resolution):**
```bash
actions-maintainer scan --owner myorg --token $GITHUB_TOKEN
```
Uses GitHub API to resolve version references to commit SHAs for accurate comparison.

**String matching only:**
```bash
actions-maintainer scan --owner myorg --token $GITHUB_TOKEN --skip-resolution
```
Uses traditional string-based version comparison for faster execution or when API access is limited.

### Benefits
- **Accuracy**: Detects equivalent versions even with different reference formats
- **Flexibility**: Supports tags, commit SHAs, and branch references
- **Performance**: 1-hour caching reduces GitHub API calls
- **Resilience**: Graceful fallback to string matching on API failures

## Architecture

```
cmd/actions-maintainer/    # CLI entry point
internal/
├── github/               # GitHub API client
├── workflow/             # Workflow parsing and analysis
├── actions/              # Action version management
├── patcher/              # Action transformation and location migration
├── cache/                # SQLite caching with TTL
├── output/               # JSON output formatting
└── pr/                   # Pull request creation
```

The `patcher/` package provides sophisticated transformation capabilities including:
- Parameter transformations during version upgrades
- Repository location migration support  
- Workflow content patching with change tracking
- Rule-based transformation logic for common actions

## Examples

### Example Workflow Analysis

Given a workflow file with:

```yaml
name: CI
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v2
      - uses: actions/cache@v3
```

The tool will identify:
- `actions/checkout@v3` → should be updated to `v4`
- `actions/setup-node@v2` → should be updated to `v4` (high severity)
- `actions/cache@v3` → should be updated to `v4`

### Sample Output

```json
{
  "summary": {
    "total_repositories": 1,
    "total_actions": 3,
    "issues_by_type": {
      "outdated": 3
    },
    "issues_by_severity": {
      "high": 1,
      "low": 2
    }
  }
}
```

## Built-in Action Rules

The tool includes rules for popular GitHub Actions:

- `actions/checkout` (latest: v4)
- `actions/setup-node` (latest: v4)
- `actions/setup-python` (latest: v5)
- `actions/upload-artifact` (latest: v4)
- `actions/download-artifact` (latest: v4)
- `actions/cache` (latest: v4)
- `actions/setup-go` (latest: v5)
- `actions/setup-java` (latest: v4)

## Action Location Migration

The patcher system now supports migration of actions to new repository locations, allowing for seamless transitions when actions move or are reorganized. This feature supports:

### Migration Types

1. **Organization Migration**: Actions moving between organizations
   ```yaml
   # Before
   - uses: old-org/standard-action@v3
   
   # After
   - uses: new-org/standard-action@v3
   ```

2. **Repository Migration**: Actions moving to completely new repository names
   ```yaml
   # Before
   - uses: legacy-org/deprecated-action@v1
     with:
       old-param: value
   
   # After  
   - uses: modern-org/recommended-action@v2
     with:
       new-param: value
       migrate-notice: "This action has been migrated..."
   ```

### Automatic Parameter Transformation

During location migration, the patcher can also transform action parameters:

- **Parameter Renaming**: `old-param` → `new-param`
- **Parameter Addition**: Adding new required/recommended parameters
- **Parameter Removal**: Removing deprecated parameters
- **Value Modification**: Updating parameter values for compatibility

### Configuration

Location migration rules are defined in the patcher's rule system with both source and target repositories:

```go
{
    FromVersion:    "v1",
    ToVersion:      "v2", 
    FromRepository: "legacy-org/deprecated-action",
    ToRepository:   "modern-org/recommended-action",
    Description:    "Migration to new maintainer with enhanced features",
    Patches: []FieldPatch{
        // Parameter transformations...
    },
}
```

This enables the tool to handle complex migration scenarios where actions not only change versions but also move to new locations with different parameter schemas.

## Automated Migration Pull Requests

When repository migrations are detected, the PR creation system automatically handles the complete migration process:

### Migration Detection to PR Creation

1. **Detection**: The action analysis system identifies when actions have migrated to new repositories
2. **Planning**: Migration targets are parsed and validated from the detection results
3. **Workflow Updates**: YAML workflow files are updated to use the new repository locations
4. **PR Creation**: Dedicated pull requests are created with detailed migration information

### PR Content for Migrations

Pull requests that include migrations feature a dedicated "🚀 Action Migrations" section:

```markdown
### 🚀 Action Migrations

- **legacy-org/deprecated-action**: `legacy-org/deprecated-action@v1` → `modern-org/recommended-action@v2`
  - **File**: `.github/workflows/ci.yml`
  - **Reason**: Action has migrated to modern-org/recommended-action for better maintenance

### 📊 Version Updates

- **actions/checkout**: v3 → v4
  - **File**: `.github/workflows/ci.yml`
```

### Example Migration Workflow

Before migration:
```yaml
- uses: legacy-org/deprecated-action@v1
  with:
    token: ${{ secrets.TOKEN }}
```

After migration:
```yaml
- uses: modern-org/recommended-action@v2
  with:
    token: ${{ secrets.TOKEN }}
```

The system ensures that both repository changes and version updates are applied correctly, with parameter transformations when needed.

## Documentation

For detailed guides and advanced usage:

- **[Reusable Workflow Migration Guide](docs/reusable-workflow-migration-guide.md)** - Comprehensive guide for migrating reusable workflows from one repository to another

## Releases

This project uses automated releases through GitHub Actions. To create a new release:

1. Go to the **Actions** tab in the GitHub repository
2. Select the **Release** workflow
3. Click **Run workflow**
4. Choose the version increment type:
   - **patch** - for bug fixes (e.g., 1.0.0 → 1.0.1)
   - **minor** - for new features (e.g., 1.0.0 → 1.1.0)
   - **major** - for breaking changes (e.g., 1.0.0 → 2.0.0)
5. Click **Run workflow**

The workflow will:
- Calculate the next semantic version automatically
- Run all tests and build binaries for multiple platforms
- Create a Git tag and GitHub release
- Upload pre-built binaries for Linux, macOS, and Windows

## Contributing

1. Follow the KISS principle - keep it simple and readable
2. Break features into focused packages within `internal/`
3. Add tests for new functionality
4. Update documentation as needed

## License

MIT License - see LICENSE file for details.
