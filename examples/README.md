# Examples Directory

This directory contains practical examples and rule templates for using actions-maintainer to update and migrate GitHub Actions workflows.

## Directory Structure

- **`rules/`** - Custom rule files for different scenarios
- **`workflows/`** - Example workflow files showing before/after transformations
- **`commands/`** - Example CLI commands for common use cases

## Quick Start

1. **Basic Action Updates**: Use rules from `rules/basic-updates.json` to update common actions to their latest versions
2. **Organization Migration**: Use `rules/organization-migration.json` when actions move between organizations
3. **Complete Workflow Migration**: Use `rules/workflow-migration.json` for migrating reusable workflows
4. **Custom Transformations**: See `rules/custom-transformations.json` for parameter transformation examples

## Usage Patterns

### Apply Basic Updates
```bash
./bin/actions-maintainer scan --owner myorg --rules-file examples/rules/basic-updates.json --create-prs
```

### Organization Migration
```bash
./bin/actions-maintainer scan --owner myorg --rules-file examples/rules/organization-migration.json --create-prs --verbose
```

### Workflow Migration
```bash
./bin/actions-maintainer scan --owner myorg --workflow-only --rules-file examples/rules/workflow-migration.json --create-prs
```

Each example includes detailed comments explaining the rule structure and use cases.