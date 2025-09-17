# Reusable Workflow Migration Guide

This guide provides comprehensive instructions for migrating reusable workflows from one repository to another using the actions-maintainer tool.

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Migration Process](#migration-process)
4. [Common Migration Scenarios](#common-migration-scenarios)
5. [Best Practices](#best-practices)
6. [Troubleshooting](#troubleshooting)
7. [Examples](#examples)

## Overview

Reusable workflows are a powerful GitHub Actions feature that allows you to share common workflows across repositories. When you need to migrate these workflows to a new location (due to repository restructuring, organization changes, or maintenance transfers), the actions-maintainer tool can help automate the process.

### What is a Reusable Workflow?

A reusable workflow is defined in `.github/workflows/` and can be called from other workflows using the `uses` keyword at the job level:

```yaml
jobs:
  call-workflow:
    uses: owner/repo/.github/workflows/reusable-workflow.yml@v1
    with:
      input-parameter: value
    secrets:
      secret-parameter: ${{ secrets.MY_SECRET }}
```

### Migration Challenges

Migrating reusable workflows involves several challenges:

- **Dependency Updates**: All repositories calling the workflow need to update their references
- **Parameter Changes**: Input/output parameters might change during migration
- **Secret Handling**: Secret passing might need adjustment
- **Version Management**: Maintaining backward compatibility during transition
- **Testing**: Ensuring the migrated workflow works correctly

## Prerequisites

### Tool Installation

1. Clone and build the actions-maintainer tool:
   ```bash
   git clone https://github.com/Jake-Mok-Nelson/actions-maintainer
   cd actions-maintainer
   make build
   ```

2. Set up authentication:
   ```bash
   export GITHUB_TOKEN=your_personal_access_token
   ```

### Required Permissions

Your GitHub token needs the following permissions:
- **Repository access**: Read access to source and target repositories
- **Pull requests**: Write access to create migration PRs
- **Actions**: Read access to analyze workflows

### Planning Phase

1. **Identify Dependencies**: Find all repositories using your reusable workflow
2. **Plan Migration**: Decide on the new location and any parameter changes
3. **Communication**: Notify teams about the upcoming migration
4. **Backup**: Ensure you have backups of critical workflows

## Migration Process

### Step 1: Analyze Current Usage

First, identify all repositories using your reusable workflows:

```bash
# Scan organization for reusable workflow usage
./bin/actions-maintainer scan \
  --owner your-org \
  --workflow-only \
  --verbose \
  --output current-usage.json
```

The `--workflow-only` flag ensures the tool focuses only on reusable workflows, filtering out regular actions.

### Step 2: Create Migration Rules

Create a custom rules file to define the migration:

```json
[
  {
    "repository": "old-org/shared-workflows",
    "workflow_path": ".github/workflows/ci.yml",
    "migrate_to_repository": "new-org/common-workflows",
    "migrate_to_version": "v2",
    "migrate_to_path": ".github/workflows/enhanced-ci.yml",
    "recommendation": "CI workflow has moved to new-org/common-workflows with enhanced features and better performance"
  },
  {
    "repository": "old-org/shared-workflows", 
    "workflow_path": ".github/workflows/deploy.yml",
    "migrate_to_repository": "new-org/common-workflows",
    "migrate_to_version": "v2",
    "migrate_to_path": ".github/workflows/enhanced-deploy.yml",
    "recommendation": "Deploy workflow has moved to new-org/common-workflows with enhanced security and deployment features"
  }
]
```

**Path-Specific Migration**: The tool now supports targeting specific workflow files within a repository using the `workflow_path` field. This allows for granular control over which workflows are migrated and where they go.

### Step 3: Set Up the Target Repository

1. **Create Target Repository**: Set up the new repository if it doesn't exist
2. **Copy Workflows**: Copy the reusable workflow files to the new location
3. **Update Documentation**: Update README and documentation in the target repository
4. **Test Workflows**: Ensure the workflows work correctly in the new location

### Step 4: Create Migration PRs

Run the tool to create pull requests for all dependent repositories:

```bash
./bin/actions-maintainer scan \
  --owner your-org \
  --workflow-only \
  --rules-file migration-rules.json \
  --create-prs \
  --verbose
```

### Step 5: Monitor and Update

1. **Review PRs**: Check all generated pull requests for accuracy
2. **Test Changes**: Ensure workflows still function correctly
3. **Merge Gradually**: Merge PRs in batches to monitor for issues
4. **Monitor Usage**: Track usage of old vs new workflows

## Common Migration Scenarios

### Path-Specific Targeting

The tool now supports targeting specific workflow files within a repository, enabling precise control over which workflows are migrated:

```yaml
# Source workflow reference
jobs:
  test:
    uses: old-org/workflows/.github/workflows/ci.yml@v1

# After migration with path-specific rule
jobs:
  test:
    uses: new-org/workflows/.github/workflows/enhanced-ci.yml@v2
```

**Path-Specific Rule Format:**
```json
[
  {
    "repository": "old-org/workflows",
    "workflow_path": ".github/workflows/ci.yml", 
    "migrate_to_repository": "new-org/workflows",
    "migrate_to_version": "v2",
    "migrate_to_path": ".github/workflows/enhanced-ci.yml",
    "recommendation": "CI workflow enhanced with better caching and security"
  }
]
```

### Rule Priority

When multiple rules match the same repository:
1. **Exact path match** takes highest priority
2. **Generic repository rule** (no `workflow_path`) serves as fallback

This allows you to have specific rules for certain workflows and general rules for others in the same repository.

### Scenario 1: Organization Migration

Moving workflows when changing organizations:

```yaml
# Before
jobs:
  test:
    uses: old-company/workflows/.github/workflows/test.yml@v1

# After  
jobs:
  test:
    uses: new-company/workflows/.github/workflows/test.yml@v1
```

**Migration Rule:**
```json
[
  {
    "repository": "old-company/workflows",
    "migrate_to_repository": "new-company/workflows",
    "migrate_to_version": "v1"
  }
]
```

For path-specific migrations:
```json
[
  {
    "repository": "old-company/workflows",
    "workflow_path": ".github/workflows/ci.yml",
    "migrate_to_repository": "new-company/workflows", 
    "migrate_to_version": "v1",
    "migrate_to_path": ".github/workflows/enhanced-ci.yml"
  }
]
```

### Scenario 2: Repository Restructuring

Consolidating workflows into a dedicated repository:

```yaml
# Before
jobs:
  build:
    uses: project-a/repo/.github/workflows/build.yml@main
  deploy:
    uses: project-b/repo/.github/workflows/deploy.yml@main

# After
jobs:
  build:
    uses: company/shared-workflows/.github/workflows/build.yml@v1
  deploy:
    uses: company/shared-workflows/.github/workflows/deploy.yml@v1
```

### Scenario 3: Workflow Enhancement

Migrating to an improved version with parameter changes:

```yaml
# Before
jobs:
  analyze:
    uses: old-org/tools/.github/workflows/security-scan.yml@v1
    with:
      scan-type: basic

# After
jobs:
  analyze:
    uses: new-org/security/.github/workflows/advanced-scan.yml@v2
    with:
      scan-level: comprehensive  # Parameter renamed
      include-deps: true         # New parameter
```

### Scenario 4: Technology Migration

Moving from one CI/CD platform integration to another:

```yaml
# Before
jobs:
  deploy:
    uses: legacy-org/jenkins-workflows/.github/workflows/deploy.yml@v1

# After
jobs:
  deploy:
    uses: modern-org/github-actions/.github/workflows/deploy.yml@v1
```

## Best Practices

### 1. Versioning Strategy

- **Use Semantic Versioning**: Tag your reusable workflows with semantic versions
- **Maintain Compatibility**: Keep old versions available during transition
- **Document Changes**: Clearly document what changes between versions

```bash
# Tag the new workflow version
git tag -a v2.0.0 -m "Enhanced CI workflow with new features"
git push origin v2.0.0
```

### 2. Gradual Migration

- **Pilot Testing**: Start with non-critical repositories
- **Phased Rollout**: Migrate in batches rather than all at once
- **Rollback Plan**: Keep old workflows available for rollback if needed

### 3. Communication

- **Advance Notice**: Notify teams at least 2 weeks before migration
- **Migration Timeline**: Provide clear timelines and milestones
- **Support Channels**: Establish channels for migration support

### 4. Testing Strategy

- **Dry Run**: Test migrations in a staging environment first
- **Validation**: Ensure all inputs/outputs work as expected
- **Monitoring**: Monitor workflow executions after migration

### 5. Documentation

- **Migration Guide**: Create repository-specific migration guides
- **Change Log**: Document all parameter and behavior changes
- **Examples**: Provide before/after examples for common use cases

## Troubleshooting

### Common Issues

#### Issue: "Workflow not found" errors

**Cause**: The new workflow path doesn't exist or isn't accessible.

**Solution**:
1. Verify the target repository exists and is public (or accessible)
2. Check that the workflow file exists at the specified path
3. Ensure the branch/tag reference is correct

```bash
# Verify workflow exists
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://api.github.com/repos/new-org/workflows/contents/.github/workflows/ci.yml
```

#### Issue: "Input parameter not defined" errors

**Cause**: Required input parameters were renamed or removed.

**Solution**:
1. Check the workflow's input definitions
2. Update the calling workflow's `with` section
3. Use migration rules to handle parameter transformations

#### Issue: Permission denied errors

**Cause**: The new workflow requires different permissions.

**Solution**:
1. Review the workflow's permission requirements
2. Update the calling workflow's permissions
3. Ensure secrets are properly configured

#### Issue: Workflow runs but fails

**Cause**: Environment or context differences between old and new locations.

**Solution**:
1. Compare environment variables between old and new workflows
2. Check secret availability in the new repository
3. Verify runner labels and requirements

### Debugging Tips

1. **Use Verbose Mode**: Run with `--verbose` to see detailed processing information
2. **Check API Limits**: Monitor GitHub API rate limits during large migrations
3. **Validate Rules**: Test migration rules on a small subset first
4. **Monitor Logs**: Watch workflow execution logs for runtime issues

### Getting Help

1. **Tool Issues**: Check the actions-maintainer repository for known issues
2. **GitHub Actions**: Consult GitHub's official documentation
3. **Community**: Use GitHub Community forums for workflow-specific questions

## Examples

### Example 1: Simple Organization Migration

**Scenario**: Moving workflows from `old-company` to `new-company` organization.

**Step 1**: Create migration rules file `org-migration.json`:
```json
[
  {
    "repository": "old-company/shared-workflows",
    "migrate_to_repository": "new-company/shared-workflows",
    "migrate_to_version": "v1",
    "recommendation": "Workflow moved to new-company organization"
  }
]
```

For targeting specific workflows:
```json
[
  {
    "repository": "old-company/shared-workflows",
    "workflow_path": ".github/workflows/ci.yml",
    "migrate_to_repository": "new-company/shared-workflows",
    "migrate_to_version": "v1",
    "migrate_to_path": ".github/workflows/ci.yml",
    "recommendation": "CI workflow moved to new-company organization"
  },
  {
    "repository": "old-company/shared-workflows",
    "workflow_path": ".github/workflows/deploy.yml", 
    "migrate_to_repository": "new-company/shared-workflows",
    "migrate_to_version": "v1",
    "migrate_to_path": ".github/workflows/deploy.yml",
    "recommendation": "Deploy workflow moved to new-company organization"
  }
]
```

**Step 2**: Run migration:
```bash
./bin/actions-maintainer scan \
  --owner old-company \
  --workflow-only \
  --rules-file org-migration.json \
  --create-prs
```

### Example 2: Workflow Consolidation

**Scenario**: Consolidating multiple project workflows into a central repository.

**Step 1**: Create rules for multiple sources:
```json
[
  {
    "repository": "project-a/ci-workflows",
    "migrate_to_repository": "company/central-workflows",
    "migrate_to_version": "v1"
  },
  {
    "repository": "project-b/build-workflows", 
    "migrate_to_repository": "company/central-workflows",
    "migrate_to_version": "v1"
  }
]
```

For path-specific consolidation:
```json
[
  {
    "repository": "project-a/ci-workflows",
    "workflow_path": ".github/workflows/test.yml",
    "migrate_to_repository": "company/central-workflows",
    "migrate_to_version": "v1",
    "migrate_to_path": ".github/workflows/project-a-test.yml"
  },
  {
    "repository": "project-b/build-workflows",
    "workflow_path": ".github/workflows/build.yml", 
    "migrate_to_repository": "company/central-workflows",
    "migrate_to_version": "v1",
    "migrate_to_path": ".github/workflows/project-b-build.yml"
  }
]
```

**Step 2**: Scan all organization repositories:
```bash
./bin/actions-maintainer scan \
  --owner company \
  --workflow-only \
  --rules-file consolidation.json \
  --filter "project-.*" \
  --create-prs
```

### Example 3: Workflow Enhancement Migration

**Scenario**: Migrating to an enhanced workflow with new parameters.

**Step 1**: Set up parameter transformation rules in the patcher system (requires code modification):
```go
// Add to internal/patcher/rules.go
p.rules["old-org/basic-ci"] = ActionPatchRule{
    Repository: "old-org/basic-ci", 
    VersionPatches: []VersionPatch{
        {
            FromVersion:    "v1",
            ToVersion:      "v2",
            FromRepository: "old-org/basic-ci",
            ToRepository:   "new-org/enhanced-ci",
            Description:    "Migration to enhanced CI with new features",
            Patches: []FieldPatch{
                {
                    Operation: OperationAdd,
                    Field:     "enable-caching",
                    Value:     true,
                    Reason:    "New caching feature enabled by default",
                },
                {
                    Operation: OperationRename,
                    Field:     "test-command",
                    NewField:  "test-script", 
                    Reason:    "Parameter renamed for clarity",
                },
            },
        },
    },
}
```

**Step 2**: Run migration with transformations:
```bash
./bin/actions-maintainer scan \
  --owner company \
  --workflow-only \
  --create-prs \
  --verbose
```

This guide provides a comprehensive approach to migrating reusable workflows using the actions-maintainer tool. Remember to always test migrations in a controlled environment before applying them to production workflows.