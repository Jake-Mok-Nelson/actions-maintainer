package patcher

// loadDefaultRules loads the default patch rules for common GitHub Actions
// These rules define how to transform action configurations when upgrading versions
func (p *Patcher) loadDefaultRules() {
	// Actions/checkout has significant changes between versions
	// Key migration patterns:
	// - v1 -> v4: token parameter behavior changed
	// - v2 -> v4: fetch-depth default changed, new parameters added
	// - v3 -> v4: minimal changes, mostly performance improvements
	p.rules["actions/checkout"] = ActionPatchRule{
		Repository: "actions/checkout",
		VersionPatches: []VersionPatch{
			{
				FromVersion: "v1",
				ToVersion:   "v4",
				Description: "Major upgrade from v1 to v4 with token handling and fetch behavior changes",
				Patches: []FieldPatch{
					{
						Operation: OperationRemove,
						Field:     "token",
						Reason:    "In v4, the token parameter is no longer required as it automatically uses GITHUB_TOKEN with appropriate permissions",
					},
					{
						Operation: OperationAdd,
						Field:     "fetch-depth",
						Value:     1,
						Reason:    "v4 defaults to shallow clone (fetch-depth: 1) for better performance. Explicitly set if full history needed",
					},
				},
			},
			{
				FromVersion: "v2",
				ToVersion:   "v4",
				Description: "Upgrade from v2 to v4 with improved defaults and new capabilities",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "fetch-tags",
						Value:     false,
						Reason:    "v4 introduces fetch-tags parameter to control tag fetching behavior. Default is false for performance",
					},
				},
			},
			{
				FromVersion: "v3",
				ToVersion:   "v4",
				Description: "Minor upgrade from v3 to v4 with performance improvements",
				Patches: []FieldPatch{
					// v3 to v4 is mostly backwards compatible, minimal patches needed
					{
						Operation: OperationAdd,
						Field:     "show-progress",
						Value:     true,
						Reason:    "v4 adds show-progress parameter for better user experience during large repository operations",
					},
				},
			},
		},
	}

	// Actions/setup-node has breaking changes in input parameter names
	// Key migration patterns:
	// - v1/v2 -> v4: 'version' becomes 'node-version', registry handling improved
	// - v3 -> v4: cache parameter changes, architecture support added
	p.rules["actions/setup-node"] = ActionPatchRule{
		Repository: "actions/setup-node",
		VersionPatches: []VersionPatch{
			{
				FromVersion: "v1",
				ToVersion:   "v4",
				Description: "Major upgrade from v1 to v4 with parameter name changes and enhanced caching",
				Patches: []FieldPatch{
					{
						Operation: OperationRename,
						Field:     "version",
						NewField:  "node-version",
						Reason:    "In v4, the 'version' parameter was renamed to 'node-version' for clarity and consistency",
					},
					{
						Operation: OperationAdd,
						Field:     "cache",
						Value:     "npm",
						Reason:    "v4 introduces built-in dependency caching. 'npm' is the most common package manager",
					},
				},
			},
			{
				FromVersion: "v2",
				ToVersion:   "v4",
				Description: "Upgrade from v2 to v4 with improved caching and registry support",
				Patches: []FieldPatch{
					{
						Operation: OperationRename,
						Field:     "version",
						NewField:  "node-version",
						Reason:    "Parameter renamed from 'version' to 'node-version' for better clarity",
					},
					{
						Operation: OperationAdd,
						Field:     "cache",
						Value:     "npm",
						Reason:    "v4 introduces intelligent dependency caching for faster builds",
					},
				},
			},
			{
				FromVersion: "v3",
				ToVersion:   "v4",
				Description: "Minor upgrade from v3 to v4 with enhanced features",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "check-latest",
						Value:     false,
						Reason:    "v4 adds check-latest parameter to control whether to check for the latest available version matching the version spec",
					},
				},
			},
		},
	}

	// Actions/setup-python has significant input parameter evolution
	// Key migration patterns:
	// - v1/v2 -> v5: 'python-version' parameter handling improved, cache support added
	// - v3/v4 -> v5: architecture and cache enhancements
	p.rules["actions/setup-python"] = ActionPatchRule{
		Repository: "actions/setup-python",
		VersionPatches: []VersionPatch{
			{
				FromVersion: "v1",
				ToVersion:   "v5",
				Description: "Major upgrade from v1 to v5 with enhanced version handling and caching",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "cache",
						Value:     "pip",
						Reason:    "v5 introduces dependency caching support. 'pip' is the standard Python package manager",
					},
					{
						Operation: OperationAdd,
						Field:     "check-latest",
						Value:     false,
						Reason:    "v5 adds check-latest to control version checking behavior for better build reliability",
					},
				},
			},
			{
				FromVersion: "v2",
				ToVersion:   "v5",
				Description: "Upgrade from v2 to v5 with caching and architecture improvements",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "cache",
						Value:     "pip",
						Reason:    "Built-in dependency caching significantly improves build performance",
					},
				},
			},
			{
				FromVersion: "v3",
				ToVersion:   "v5",
				Description: "Upgrade from v3 to v5 with enhanced caching options",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "cache-dependency-path",
						Value:     "requirements.txt",
						Reason:    "v5 allows specifying custom dependency file paths for more accurate cache invalidation",
					},
				},
			},
			{
				FromVersion: "v4",
				ToVersion:   "v5",
				Description: "Minor upgrade from v4 to v5 with performance improvements",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "allow-prereleases",
						Value:     false,
						Reason:    "v5 adds allow-prereleases parameter for better control over Python version selection",
					},
				},
			},
		},
	}

	// Actions/upload-artifact has major breaking changes in v4
	// Key migration patterns:
	// - v1/v2/v3 -> v4: completely new API design, different parameter structure
	p.rules["actions/upload-artifact"] = ActionPatchRule{
		Repository: "actions/upload-artifact",
		VersionPatches: []VersionPatch{
			{
				FromVersion: "v1",
				ToVersion:   "v4",
				Description: "Major breaking change from v1 to v4 with new artifact API",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "compression-level",
						Value:     6,
						Reason:    "v4 introduces compression-level parameter for controlling artifact size vs speed tradeoff",
					},
					{
						Operation: OperationAdd,
						Field:     "overwrite",
						Value:     false,
						Reason:    "v4 requires explicit overwrite setting to replace existing artifacts with the same name",
					},
				},
			},
			{
				FromVersion: "v2",
				ToVersion:   "v4",
				Description: "Breaking upgrade from v2 to v4 with enhanced artifact handling",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "compression-level",
						Value:     6,
						Reason:    "Configurable compression for optimal upload performance",
					},
					{
						Operation: OperationAdd,
						Field:     "retention-days",
						Value:     90,
						Reason:    "v4 allows explicit retention period control (default 90 days)",
					},
				},
			},
			{
				FromVersion: "v3",
				ToVersion:   "v4",
				Description: "Breaking upgrade from v3 to v4 with new artifact backend",
				Patches: []FieldPatch{
					{
						Operation: OperationRemove,
						Field:     "path-separator",
						Reason:    "v4 removes path-separator parameter as path handling is now automatic",
					},
					{
						Operation: OperationAdd,
						Field:     "include-hidden-files",
						Value:     false,
						Reason:    "v4 adds explicit control over hidden file inclusion for security",
					},
				},
			},
		},
	}

	// Actions/download-artifact also has breaking changes in v4
	p.rules["actions/download-artifact"] = ActionPatchRule{
		Repository: "actions/download-artifact",
		VersionPatches: []VersionPatch{
			{
				FromVersion: "v1",
				ToVersion:   "v4",
				Description: "Major upgrade from v1 to v4 with new download API",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "github-token",
						Value:     "${{ github.token }}",
						Reason:    "v4 requires explicit GitHub token for authentication in some scenarios",
					},
				},
			},
			{
				FromVersion: "v2",
				ToVersion:   "v4",
				Description: "Breaking upgrade from v2 to v4 with enhanced download capabilities",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "merge-multiple",
						Value:     false,
						Reason:    "v4 introduces merge-multiple parameter for handling multiple artifacts with same name",
					},
				},
			},
			{
				FromVersion: "v3",
				ToVersion:   "v4",
				Description: "Breaking upgrade from v3 to v4 with new artifact backend",
				Patches: []FieldPatch{
					{
						Operation: OperationRemove,
						Field:     "workflow",
						Reason:    "v4 removes workflow parameter as artifact resolution is now automatic",
					},
					{
						Operation: OperationAdd,
						Field:     "run-id",
						Value:     "${{ github.run_id }}",
						Reason:    "v4 uses run-id for more precise artifact identification",
					},
				},
			},
		},
	}

	// Actions/cache has incremental improvements across versions
	p.rules["actions/cache"] = ActionPatchRule{
		Repository: "actions/cache",
		VersionPatches: []VersionPatch{
			{
				FromVersion: "v3",
				ToVersion:   "v4",
				Description: "Upgrade from v3 to v4 with performance and reliability improvements",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "lookup-only",
						Value:     false,
						Reason:    "v4 adds lookup-only parameter for cache existence checking without downloading",
					},
					{
						Operation: OperationAdd,
						Field:     "fail-on-cache-miss",
						Value:     false,
						Reason:    "v4 introduces fail-on-cache-miss for stricter cache dependency workflows",
					},
				},
			},
		},
	}

	// Actions/setup-go has version parameter improvements
	p.rules["actions/setup-go"] = ActionPatchRule{
		Repository: "actions/setup-go",
		VersionPatches: []VersionPatch{
			{
				FromVersion: "v4",
				ToVersion:   "v5",
				Description: "Upgrade from v4 to v5 with enhanced Go version handling",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "cache-dependency-path",
						Value:     "go.sum",
						Reason:    "v5 improves dependency caching by allowing custom dependency file specification",
					},
					{
						Operation: OperationAdd,
						Field:     "check-latest",
						Value:     false,
						Reason:    "v5 adds check-latest for controlling Go version update behavior",
					},
				},
			},
		},
	}

	// Actions/setup-java parameter enhancements
	p.rules["actions/setup-java"] = ActionPatchRule{
		Repository: "actions/setup-java",
		VersionPatches: []VersionPatch{
			{
				FromVersion: "v3",
				ToVersion:   "v4",
				Description: "Upgrade from v3 to v4 with improved Java distribution support",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "cache-dependency-path",
						Value:     "pom.xml",
						Reason:    "v4 enhances dependency caching with configurable dependency file paths",
					},
					{
						Operation: OperationAdd,
						Field:     "job-status",
						Value:     "success",
						Reason:    "v4 adds job-status parameter for conditional cache operations based on job outcome",
					},
				},
			},
		},
	}

	// Example: Legacy action migration to new location
	// This demonstrates how to handle actions that have moved to new repositories
	p.rules["legacy-org/deprecated-action"] = ActionPatchRule{
		Repository: "legacy-org/deprecated-action",
		VersionPatches: []VersionPatch{
			{
				FromVersion:    "v1",
				ToVersion:      "v2",
				FromRepository: "legacy-org/deprecated-action",
				ToRepository:   "modern-org/recommended-action",
				Description:    "Migration from deprecated legacy action to new recommended location with enhanced features",
				Patches: []FieldPatch{
					{
						Operation: OperationAdd,
						Field:     "migrate-notice",
						Value:     "This action has been migrated to modern-org/recommended-action for better maintenance and support",
						Reason:    "Informational field to track migration source",
					},
					{
						Operation: OperationRename,
						Field:     "old-param",
						NewField:  "new-param",
						Reason:    "Parameter renamed during migration to new action",
					},
				},
			},
		},
	}

	// Example: Organization migration
	// Actions that moved from one organization to another
	p.rules["old-org/standard-action"] = ActionPatchRule{
		Repository: "old-org/standard-action",
		VersionPatches: []VersionPatch{
			{
				FromVersion:    "v3",
				ToVersion:      "v3",
				FromRepository: "old-org/standard-action",
				ToRepository:   "new-org/standard-action",
				Description:    "Organization migration from old-org to new-org with same functionality",
				Patches: []FieldPatch{
					// No parameter changes needed, just location change
				},
			},
		},
	}
}
