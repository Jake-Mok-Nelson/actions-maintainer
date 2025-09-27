#!/bin/bash
# Migration and consolidation commands for actions-maintainer
# These commands handle complex organizational changes

export GITHUB_TOKEN="your_github_token_here"

echo "=== Migration Commands ==="

echo "1. Organization name change migration"
echo "# Use when your organization changes names (old-company -> new-company)"
echo "./bin/actions-maintainer scan --owner affected-repos --rules-file examples/rules/organization-migration.json --create-prs --verbose"
echo ""

echo "2. Workflow repository consolidation"
echo "# Consolidate multiple workflow repos into a central one"
echo "./bin/actions-maintainer scan --owner company --workflow-only --rules-file examples/rules/workflow-migration.json --create-prs"
echo ""

echo "3. Department-specific consolidation"
echo "# Migrate all department workflows to a central location"
echo "./bin/actions-maintainer scan --owner company --filter 'engineering-.*' --workflow-only --rules-file examples/rules/workflow-migration.json --create-prs"
echo ""

echo "4. Legacy system migration"
echo "# Migrate from legacy actions to modern alternatives"
echo "./bin/actions-maintainer scan --owner legacy-org --rules-file examples/rules/organization-migration.json --verbose --create-prs"
echo ""

echo "=== Step-by-Step Migration Process ==="

echo "Step 1: Discovery - Find what needs to be migrated"
echo "./bin/actions-maintainer scan --owner myorg --verbose --output migration-analysis.json"
echo ""

echo "Step 2: Dry run with migration rules"
echo "./bin/actions-maintainer scan --owner myorg --rules-file examples/rules/workflow-migration.json --verbose"
echo ""

echo "Step 3: Apply migrations with PR creation"
echo "./bin/actions-maintainer scan --owner myorg --rules-file examples/rules/workflow-migration.json --create-prs"
echo ""

echo "=== Targeted Migration Scenarios ==="

echo "Scenario A: Specific repository migration"
echo "./bin/actions-maintainer scan --owner myorg --filter 'legacy-workflows' --rules-file examples/rules/workflow-migration.json --create-prs"
echo ""

echo "Scenario B: Multi-organization consolidation" 
echo "# First scan old organizations"
echo "./bin/actions-maintainer scan --owner old-org-1 --rules-file examples/rules/organization-migration.json --output old-org-1-results.json"
echo "./bin/actions-maintainer scan --owner old-org-2 --rules-file examples/rules/organization-migration.json --output old-org-2-results.json"
echo "# Then apply migrations"
echo "./bin/actions-maintainer scan --owner old-org-1 --rules-file examples/rules/organization-migration.json --create-prs"
echo "./bin/actions-maintainer scan --owner old-org-2 --rules-file examples/rules/organization-migration.json --create-prs"
echo ""

echo "Scenario C: Gradual migration with filtering"
echo "# Migrate one team at a time"
echo "./bin/actions-maintainer scan --owner company --filter 'team-alpha-.*' --rules-file examples/rules/workflow-migration.json --create-prs"
echo "./bin/actions-maintainer scan --owner company --filter 'team-beta-.*' --rules-file examples/rules/workflow-migration.json --create-prs"
echo ""

echo "=== Validation Commands ==="

echo "Before migration: Analyze current state"
echo "./bin/actions-maintainer scan --owner myorg --output pre-migration-state.json"
echo ""

echo "After migration: Verify changes"  
echo "./bin/actions-maintainer scan --owner myorg --output post-migration-state.json"
echo ""

echo "Compare results:"
echo "# Use your preferred JSON diff tool to compare pre-migration-state.json and post-migration-state.json"