#!/bin/bash
# Basic scanning commands for actions-maintainer
# These commands demonstrate fundamental usage patterns

# Set your GitHub token (replace with your actual token)
export GITHUB_TOKEN="your_github_token_here"

echo "=== Basic Scanning Examples ==="

echo "1. Basic scan with default rules (no custom rules file needed)"
echo "./bin/actions-maintainer scan --owner myorg"
echo ""

echo "2. Scan with verbose output to see what's happening"
echo "./bin/actions-maintainer scan --owner myorg --verbose"
echo ""

echo "3. Save results to JSON file for later analysis"
echo "./bin/actions-maintainer scan --owner myorg --output results.json"
echo ""

echo "4. Save results to Jupyter notebook for analysis"
echo "./bin/actions-maintainer scan --owner myorg --output results.ipynb"
echo ""

echo "5. Apply basic action updates using example rules"
echo "./bin/actions-maintainer scan --owner myorg --rules-file examples/rules/basic-updates.json"
echo ""

echo "6. Create pull requests for basic updates (DRY RUN FIRST!)"
echo "./bin/actions-maintainer scan --owner myorg --rules-file examples/rules/basic-updates.json --verbose"
echo "# Review the output above, then run with --create-prs if it looks correct"
echo "./bin/actions-maintainer scan --owner myorg --rules-file examples/rules/basic-updates.json --create-prs"
echo ""

echo "7. Filter repositories by name pattern"
echo "./bin/actions-maintainer scan --owner myorg --filter 'my-project-.*' --verbose"
echo ""

echo "8. Scan only workflow files (skip action analysis)"
echo "./bin/actions-maintainer scan --owner myorg --workflow-only"
echo ""

echo "9. Skip version resolution for faster scanning"
echo "./bin/actions-maintainer scan --owner myorg --skip-resolution --verbose"
echo ""

echo "=== Environment Variable Usage ==="
echo "# Instead of --token flag, you can use environment variable:"
echo "export GITHUB_TOKEN=your_token_here"
echo "./bin/actions-maintainer scan --owner myorg"
echo ""

echo "=== Testing Commands (use these first) ==="
echo "# Always test with a small organization or specific repo filter first:"
echo "./bin/actions-maintainer scan --owner small-test-org --verbose"
echo "./bin/actions-maintainer scan --owner large-org --filter 'test-repo' --verbose"