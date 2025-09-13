# Transformer/Patch Component Documentation

## Overview

The transformer/patch component enables automatic schema transformations when upgrading GitHub Actions to newer versions. Instead of just updating version numbers, it intelligently modifies action input parameters to match the new version's requirements.

## Key Features

- **Declarative patch rules** for common GitHub Actions
- **Four transformation operations**: add, remove, rename, modify fields
- **Version-specific patches** (e.g., v2 → v4 transitions)
- **Preview functionality** for safe testing
- **Detailed reasoning** for each transformation

## Supported Actions

The transformer includes patch rules for these popular actions:

| Action | Transformations | Key Changes |
|--------|----------------|-------------|
| `actions/checkout` | v1→v4, v2→v4, v3→v4 | Token handling, fetch-depth defaults |
| `actions/setup-node` | v1→v4, v2→v4, v3→v4 | Parameter renaming (version→node-version), caching |
| `actions/setup-python` | v1→v5, v2→v5, v3→v5, v4→v5 | Enhanced caching, dependency management |
| `actions/upload-artifact` | v1→v4, v2→v4, v3→v4 | Compression, retention, path handling |
| `actions/download-artifact` | v1→v4, v2→v4, v3→v4 | Authentication, merge capabilities |
| `actions/cache` | v3→v4 | Lookup options, failure handling |
| `actions/setup-go` | v4→v5 | Dependency path specification |
| `actions/setup-java` | v3→v4 | Enhanced caching configurations |

## How It Works

### 1. Integration with Action Analysis

When the action manager detects outdated actions, it automatically:

```go
// Check if there are schema transformations for this version upgrade
if patchInfo, hasPatches := m.GetTransformationInfo(action.Repository, action.Version, rule.LatestVersion); hasPatches {
    issue.HasTransformations = true
    issue.SchemaChanges = []string{patchInfo.Description}
    // Add details about specific field changes...
}
```

### 2. Patch Operations

The transformer supports four types of operations:

- **Add**: Insert new fields with default values
- **Remove**: Delete deprecated fields 
- **Rename**: Change field names (e.g., `version` → `node-version`)
- **Modify**: Update field values

### 3. Preview Before Apply

Test transformations safely:

```go
transformer := transformer.NewWorkflowTransformer()
result, err := transformer.PreviewChanges("actions/checkout", "v1", "v4", withBlock)
if result.Applied {
    fmt.Printf("Would apply %d changes\n", len(result.Changes))
}
```

## Example Transformations

### actions/checkout v1 → v4

**Before:**
```yaml
- uses: actions/checkout@v1
  with:
    token: ${{ secrets.GITHUB_TOKEN }}
```

**After:**
```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 1
```

**Rationale**: v4 no longer requires explicit token (uses GITHUB_TOKEN automatically) and defaults to shallow clone for performance.

### actions/setup-node v2 → v4

**Before:**
```yaml
- uses: actions/setup-node@v2
  with:
    version: '16'
```

**After:**
```yaml
- uses: actions/setup-node@v4
  with:
    node-version: '16'
    cache: 'npm'
```

**Rationale**: Parameter renamed for clarity, intelligent caching added for faster builds.

## API Usage

### Basic Transformation

```go
import "github.com/Jake-Mok-Nelson/actions-maintainer/internal/transformer"

// Create transformer
wt := transformer.NewWorkflowTransformer()

// Apply patches to a workflow step
result, err := wt.TransformStep(step, "v2", "v4")
if err != nil {
    log.Fatal(err)
}

if result.Applied {
    fmt.Printf("Applied %d changes: %v\n", len(result.Changes), result.Changes)
}
```

### Workflow Content Transformation

```go
// Transform entire workflow content
updates := []transformer.ActionVersionUpdate{
    {
        ActionRepo:  "actions/checkout",
        FromVersion: "v1",
        ToVersion:   "v4",
        FilePath:    "workflow.yml",
    },
}

updatedContent, changes, err := wt.TransformWorkflowContent(content, updates)
```

### Enhanced PR Creation

The PR creator now automatically applies schema transformations:

```go
creator := pr.NewCreator(githubClient)
finalContent, changes, err := creator.UpdateWorkflowContentWithTransformations(content, updates)
```

## Adding New Patch Rules

To add rules for a new action, edit `internal/transformer/rules.go`:

```go
t.rules["my-org/my-action"] = ActionPatchRule{
    Repository: "my-org/my-action",
    VersionPatches: []VersionPatch{
        {
            FromVersion: "v1",
            ToVersion:   "v2",
            Description: "Upgrade description",
            Patches: []FieldPatch{
                {
                    Operation: OperationRename,
                    Field:     "old-param",
                    NewField:  "new-param",
                    Reason:    "Parameter renamed for clarity",
                },
            },
        },
    },
}
```

## Benefits

1. **Reduced Manual Work**: Automatic schema updates eliminate tedious parameter fixes
2. **Safer Migrations**: Preview functionality prevents breaking changes
3. **Knowledge Preservation**: Patch rules document why changes are needed
4. **Team Efficiency**: Melbourne teams can focus on features instead of maintenance
5. **Backward Compatibility**: Gracefully handles actions without transformation rules

## Testing

The transformer includes comprehensive tests covering:

- Basic patch operations
- Real-world transformations (checkout, setup-node)
- Edge cases (nil with blocks, unsupported actions)
- Integration with existing workflow parsing

Run tests: `go test ./internal/transformer -v`

## Future Enhancements

- Support for conditional patches based on existing field values
- Integration with GitHub API for fetching latest action schemas
- Custom patch rule loading from external configuration files
- Support for complex nested field transformations