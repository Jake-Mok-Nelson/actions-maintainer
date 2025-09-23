# Custom Properties Feature Guide

## Overview

The actions-maintainer tool supports extracting and displaying custom repository properties in both JSON and Jupyter notebook outputs. This feature helps you organize and filter repositories based on custom metadata.

## How Custom Properties Work

Custom properties are extracted from repository metadata using multiple detection methods:

### 1. Repository Topics (Recommended)

Use GitHub repository topics with specific prefixes:

- **ProductId**: Use topic `product-{value}` (e.g., `product-shopping-cart`)
- **Team**: Use topic `team-{value}` (e.g., `team-backend`)
- **Environment**: Use topic `env-{value}` (e.g., `env-production`)
- **Custom Properties**: Use topic `{property}-{value}` (e.g., `criticality-high`, `owner-platform-team`)

### 2. Repository Names (Environment Only)

Environment is automatically detected from repository name patterns:
- Names containing `-prod` → `production`
- Names containing `-dev` → `development`
- Names containing `-test` or `-testing` → `testing`
- Names containing `-stage` or `-staging` → `staging`

### 3. Repository Descriptions

Use key-value patterns in repository descriptions:
- `Product: user-analytics` → ProductId = `user-analytics`
- `Team: data-science` → Team = `data-science`
- `{PropertyName}: {value}` → Custom property (case-insensitive)

## Usage Examples

### Basic Usage

```bash
# Scan repositories with custom properties
./bin/actions-maintainer scan --owner myorg --token $GITHUB_TOKEN \
  --custom-property ProductId \
  --custom-property Team \
  --custom-property Environment

# Output to Jupyter notebook with filtering interface
./bin/actions-maintainer scan --owner myorg --token $GITHUB_TOKEN \
  --custom-property ProductId \
  --custom-property Team \
  --output results.ipynb
```

### Setting Up Repository Metadata

#### Option 1: Using Topics (Recommended)

Add topics to your repositories:
```
product-user-management
team-backend
env-production
criticality-high
owner-platform-team
```

#### Option 2: Using Repository Descriptions

Include structured information in repository descriptions:
```
User management microservice for handling authentication and authorization.
Product: user-management Team: backend Environment: production
```

#### Option 3: Using Repository Names

Use consistent naming patterns:
```
user-service-prod     → Environment: production
payment-api-dev       → Environment: development
analytics-service-test → Environment: testing
```

### Multiple Properties

```bash
# Scan with multiple custom properties
./bin/actions-maintainer scan --owner myorg --token $GITHUB_TOKEN \
  --custom-property ProductId \
  --custom-property Team \
  --custom-property Environment \
  --custom-property Criticality \
  --custom-property Owner
```

## Output Formats

### JSON Output

Custom properties are included in the `custom_properties` field of each repository:

```json
{
  "repositories": [
    {
      "name": "user-service",
      "full_name": "myorg/user-service",
      "custom_properties": {
        "ProductId": "user-management",
        "Team": "backend",
        "Environment": "production"
      }
    }
  ]
}
```

### Jupyter Notebook Output

The notebook output includes:

1. **Interactive Filtering Interface**: Dropdown filters for each custom property
2. **Enhanced Table**: Columns for each custom property
3. **Repository Details**: Custom properties displayed in detailed breakdowns

Example table output:
```
| Repository | Workflows | Actions | Issues | Environment | ProductId | Team |
|------------|-----------|---------|--------|-------------|-----------|------|
| user-service | 3 | 8 | 2 | production | user-management | backend |
| payment-api | 2 | 5 | 0 | development | payments | backend |
```

## Best Practices

### 1. Consistent Naming Conventions

Use consistent prefixes and naming across your organization:
- `product-{product-name}`
- `team-{team-name}`
- `env-{environment}`
- `criticality-{level}`

### 2. Repository Topics

Topics are the most reliable method because they're:
- Easily searchable in GitHub
- Programmatically accessible
- Visible in the GitHub UI
- Standard GitHub feature

### 3. Multiple Detection Methods

The tool tries multiple methods in order:
1. Repository topics (most reliable)
2. Repository descriptions (flexible)
3. Repository names (automatic for environments)

### 4. Property Naming

Use clear, consistent property names:
- `ProductId` instead of `Product` or `Prod`
- `Team` instead of `TeamName` or `Owner`
- `Environment` instead of `Env` or `Stage`

## Troubleshooting

### Custom Properties Not Appearing

1. **Check Repository Access**: Ensure your GitHub token has access to the repositories
2. **Verify Property Names**: Make sure the property names match exactly (case-sensitive)
3. **Check Metadata Patterns**: Verify your repositories have the expected topics, descriptions, or name patterns
4. **Use Verbose Mode**: Run with `--verbose` to see detailed extraction logs

```bash
./bin/actions-maintainer scan --owner myorg --token $GITHUB_TOKEN \
  --custom-property ProductId --verbose
```

### Example Verbose Output

```
2025/09/23 01:05:13 GitHub API: Found ProductId 'user-analytics' from description
2025/09/23 01:05:13 GitHub API: Found Team 'backend' from topic 'team-backend'
2025/09/23 01:05:13 GitHub API: Found Environment 'production' from repository name pattern
```

### No Custom Properties Found

If no custom properties are found:
1. The filtering interface won't appear in notebook output
2. The table will use standard columns only
3. JSON output will omit the `custom_properties` field or show empty objects

### Missing Values

Missing custom properties are displayed as:
- `-` in notebook tables
- Empty string or omitted in JSON

## API Limitations

The current implementation uses GitHub's public repository API and extracts custom properties from metadata. It does not yet use the official GitHub custom properties API, which is still evolving.

For organizations with GitHub Enterprise Server, the implementation can be extended to use the official custom properties endpoints when available.

## Future Enhancements

- Support for official GitHub custom properties API
- GraphQL integration for more efficient property retrieval
- Additional detection patterns and metadata sources
- Bulk property management capabilities