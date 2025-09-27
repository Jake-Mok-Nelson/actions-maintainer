# Example Workflow Files

This directory contains example workflow files demonstrating before and after states for common action updates and migrations.

## Files Structure

- **`before/`** - Original workflow files with outdated actions
- **`after/`** - Updated workflow files showing the result of applying rules
- **`migration/`** - Examples of workflow migrations between repositories

## Common Transformation Patterns

### Action Version Updates
- `before/basic-ci.yml` → `after/basic-ci.yml` - Common actions updated to latest versions
- `before/complex-workflow.yml` → `after/complex-workflow.yml` - Complex workflow with multiple action updates

### Parameter Transformations  
- `before/node-workflow.yml` → `after/node-workflow.yml` - setup-node v2 → v4 with parameter changes
- `before/artifact-workflow.yml` → `after/artifact-workflow.yml` - upload/download-artifact v3 → v4

### Repository Migrations
- `migration/reusable-workflow-before.yml` → `migration/reusable-workflow-after.yml` - Reusable workflow migration

Each file includes comments explaining the changes and reasoning behind transformations.