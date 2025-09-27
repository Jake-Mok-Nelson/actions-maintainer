#!/bin/bash
# Advanced usage patterns and filtering for actions-maintainer
# These commands demonstrate sophisticated filtering and analysis capabilities

export GITHUB_TOKEN="your_github_token_here"

echo "=== Advanced Filtering and Analysis ==="

echo "1. Multi-pattern repository filtering"
echo "./bin/actions-maintainer scan --owner company --filter '(frontend|backend|api)-.*' --verbose"
echo ""

echo "2. Exclude specific repositories from scanning"
echo "# Note: Use negative lookahead regex patterns"
echo "./bin/actions-maintainer scan --owner company --filter '^(?!archived-).*' --verbose"
echo ""

echo "3. Custom property tracking"
echo "./bin/actions-maintainer scan --owner company --custom-property ProductId --custom-property TeamOwner --output detailed-report.json"
echo ""

echo "4. Performance optimized scanning"
echo "./bin/actions-maintainer scan --owner large-org --skip-resolution --filter 'critical-.*' --output quick-scan.json"
echo ""

echo "=== Complex Analysis Scenarios ==="

echo "Scenario 1: Security audit across organization"
echo "# Focus on security-critical actions and deprecated versions"
echo "./bin/actions-maintainer scan --owner company --rules-file examples/rules/basic-updates.json --output security-audit.json --verbose"
echo ""

echo "Scenario 2: Performance optimization analysis"
echo "# Identify actions that can benefit from caching improvements"
echo "./bin/actions-maintainer scan --owner company --filter 'build-.*' --verbose --output performance-analysis.json"
echo ""

echo "Scenario 3: Compliance reporting"
echo "# Generate comprehensive report for compliance review"
echo "./bin/actions-maintainer scan --owner company --custom-property ComplianceLevel --custom-property DataClassification --output compliance-report.ipynb"
echo ""

echo "=== Batch Processing Patterns ==="

echo "Pattern A: Multi-organization scanning"
echo "organizations=('org1' 'org2' 'org3')"
echo "for org in \"\${organizations[@]}\"; do"
echo "  echo \"Scanning \$org...\""
echo "  ./bin/actions-maintainer scan --owner \$org --rules-file examples/rules/basic-updates.json --output \"\${org}-results.json\""
echo "done"
echo ""

echo "Pattern B: Team-based analysis"
echo "teams=('frontend' 'backend' 'platform' 'mobile')"
echo "for team in \"\${teams[@]}\"; do"
echo "  echo \"Analyzing \$team repositories...\""
echo "  ./bin/actions-maintainer scan --owner company --filter \"\${team}-.*\" --output \"\${team}-analysis.json\" --verbose"
echo "done"
echo ""

echo "=== Result Analysis and Reporting ==="

echo "1. Generate Jupyter notebook for interactive analysis"
echo "./bin/actions-maintainer scan --owner company --output interactive-analysis.ipynb"
echo ""

echo "2. Create multiple output formats for different audiences"
echo "./bin/actions-maintainer scan --owner company --rules-file examples/rules/basic-updates.json --output technical-report.json"
echo "./bin/actions-maintainer scan --owner company --rules-file examples/rules/basic-updates.json --output executive-summary.ipynb"
echo ""

echo "3. Comparative analysis over time"
echo "# Create baseline"
echo "./bin/actions-maintainer scan --owner company --output baseline-$(date +%Y%m%d).json"
echo "# After changes, create new scan"
echo "./bin/actions-maintainer scan --owner company --output current-$(date +%Y%m%d).json"
echo "# Compare using your preferred JSON diff tool"
echo ""

echo "=== Debugging and Troubleshooting ==="

echo "1. Maximum verbosity for debugging"
echo "./bin/actions-maintainer scan --owner problematic-org --verbose --skip-resolution --filter 'single-repo'"
echo ""

echo "2. Test rules on specific repository"
echo "./bin/actions-maintainer scan --owner test-org --filter 'test-repo-name' --rules-file examples/rules/custom-transformations.json --verbose"
echo ""

echo "3. Cache debugging (if using external cache)"
echo "./bin/actions-maintainer scan --owner company --cache memory --verbose --output cache-debug.json"