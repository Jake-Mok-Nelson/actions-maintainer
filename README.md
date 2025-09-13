# actions-maintainer

A Go CLI tool that identifies and helps resolve TOIL around migration and updating GitHub Actions workflows.

## Features

- 🔍 **Repository Scanning**: Automatically scans all repositories for a GitHub owner/organization
- 📋 **Workflow Analysis**: Parses `.github/workflows/*.yml` files to extract action dependencies
- ⚡ **Version Management**: Identifies outdated, deprecated, and vulnerable action versions
- 💾 **Smart Caching**: SQLite-based caching with TTL to avoid unnecessary API calls
- 📊 **Detailed Reporting**: Comprehensive JSON output with statistics and issue summaries
- 🔧 **Automated Updates**: Optionally creates pull requests with safe version updates

## Installation

```bash
git clone https://github.com/Jake-Mok-Nelson/actions-maintainer
cd actions-maintainer
go build -o actions-maintainer cmd/actions-maintainer/main.go
```

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

```bash
./actions-maintainer scan --owner my-org --token YOUR_GITHUB_TOKEN --create-prs
```

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
- **Security**: Action versions with known security vulnerabilities

## Architecture

```
cmd/actions-maintainer/    # CLI entry point
internal/
├── github/               # GitHub API client
├── workflow/             # Workflow parsing and analysis
├── actions/              # Action version management
├── cache/                # SQLite caching with TTL
├── output/               # JSON output formatting
└── pr/                   # Pull request creation
```

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

## Contributing

1. Follow the KISS principle - keep it simple and readable
2. Break features into focused packages within `internal/`
3. Add tests for new functionality
4. Update documentation as needed

## License

MIT License - see LICENSE file for details.
