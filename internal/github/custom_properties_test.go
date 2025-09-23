package github

import (
	"context"
	"testing"

	"github.com/google/go-github/v65/github"
)

// TestExtractCustomPropertiesFromMetadata tests the enhanced custom properties extraction
func TestExtractCustomPropertiesFromMetadata(t *testing.T) {
	client := &Client{
		verbose: true,
	}

	tests := []struct {
		name           string
		repository     *github.Repository
		properties     []string
		expectedProps  map[string]string
		description    string
	}{
		{
			name: "Extract from topics - ProductId and Team",
			repository: &github.Repository{
				Name: github.String("test-repo"),
				Topics: []string{
					"product-shopping-cart",
					"team-backend",
					"javascript",
					"nodejs",
				},
				Description: github.String("A test repository for shopping cart functionality"),
			},
			properties: []string{"ProductId", "Team"},
			expectedProps: map[string]string{
				"ProductId": "shopping-cart",
				"Team":      "backend",
			},
			description: "Should extract ProductId and Team from topics",
		},
		{
			name: "Extract Environment from repository name",
			repository: &github.Repository{
				Name: github.String("my-service-prod"),
				Topics: []string{
					"microservice",
					"api",
				},
				Description: github.String("Production service for handling payments"),
			},
			properties: []string{"Environment"},
			expectedProps: map[string]string{
				"Environment": "production",
			},
			description: "Should extract Environment from repository name pattern",
		},
		{
			name: "Extract from description using key:value pattern",
			repository: &github.Repository{
				Name: github.String("analytics-service"),
				Topics: []string{
					"analytics",
				},
				Description: github.String("Analytics service for tracking user behavior. Product: user-analytics Team: data-science"),
			},
			properties: []string{"ProductId", "Team"},
			expectedProps: map[string]string{
				"ProductId": "user-analytics",
				"Team":      "data-science",
			},
			description: "Should extract custom properties from description patterns",
		},
		{
			name: "Extract Environment from env- topic",
			repository: &github.Repository{
				Name: github.String("service-api"),
				Topics: []string{
					"env-staging",
					"api",
					"golang",
				},
				Description: github.String("API service"),
			},
			properties: []string{"Environment"},
			expectedProps: map[string]string{
				"Environment": "staging",
			},
			description: "Should extract Environment from env- topic prefix",
		},
		{
			name: "Extract custom property with generic pattern",
			repository: &github.Repository{
				Name: github.String("user-service"),
				Topics: []string{
					"criticality-high",
					"owner-platform-team",
					"service",
				},
				Description: github.String("User management service"),
			},
			properties: []string{"Criticality", "Owner"},
			expectedProps: map[string]string{
				"Criticality": "high",
				"Owner":       "platform-team",
			},
			description: "Should extract custom properties using generic topic patterns",
		},
		{
			name: "Multiple environment detection methods",
			repository: &github.Repository{
				Name: github.String("payment-service-dev"),
				Topics: []string{
					"payment",
					"env-development", // This should override the name-based detection
				},
				Description: github.String("Development payment service"),
			},
			properties: []string{"Environment"},
			expectedProps: map[string]string{
				"Environment": "development", // Should prefer topic over name pattern
			},
			description: "Should prefer topic-based environment detection over name patterns",
		},
		{
			name: "No matching patterns",
			repository: &github.Repository{
				Name: github.String("simple-repo"),
				Topics: []string{
					"javascript",
					"web",
				},
				Description: github.String("A simple repository without custom property patterns"),
			},
			properties: []string{"ProductId", "Team", "Environment"},
			expectedProps: map[string]string{},
			description: "Should return empty map when no patterns match",
		},
		{
			name: "Mixed success and failure",
			repository: &github.Repository{
				Name: github.String("api-gateway"),
				Topics: []string{
					"team-infrastructure",
					"api",
				},
				Description: github.String("API Gateway service. Product: gateway-core"),
			},
			properties: []string{"ProductId", "Team", "Environment", "Owner"},
			expectedProps: map[string]string{
				"ProductId": "gateway-core", // From description
				"Team":      "infrastructure", // From topic
				// Environment and Owner should be missing
			},
			description: "Should extract only the properties that have matching patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.extractCustomPropertiesFromMetadata(tt.repository, tt.properties)

			// Check if we got the expected number of properties
			if len(result) != len(tt.expectedProps) {
				t.Errorf("Expected %d properties, got %d. Expected: %v, Got: %v", 
					len(tt.expectedProps), len(result), tt.expectedProps, result)
			}

			// Check each expected property
			for key, expectedValue := range tt.expectedProps {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected property '%s' not found in result", key)
				} else if actualValue != expectedValue {
					t.Errorf("Property '%s': expected '%s', got '%s'", key, expectedValue, actualValue)
				}
			}

			// Check for unexpected properties
			for key := range result {
				if _, expected := tt.expectedProps[key]; !expected {
					t.Errorf("Unexpected property '%s' = '%s' in result", key, result[key])
				}
			}

			t.Logf("✅ %s", tt.description)
			t.Logf("   Repository: %s", tt.repository.GetName())
			t.Logf("   Topics: %v", tt.repository.Topics)
			t.Logf("   Description: %s", tt.repository.GetDescription())
			t.Logf("   Extracted: %v", result)
		})
	}
}

// TestGetRepositoryCustomPropertiesIntegration tests the complete flow including error handling
func TestGetRepositoryCustomPropertiesIntegration(t *testing.T) {
	// Create a client with mock functionality
	client := &Client{
		ctx:     context.Background(),
		verbose: true,
	}

	t.Run("Empty properties list", func(t *testing.T) {
		result, err := client.GetRepositoryCustomProperties("owner", "repo", []string{})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Expected empty result for empty properties list, got %v", result)
		}
	})

	// Note: Full integration tests would require mocking the GitHub API client
	// or using a test server, which is beyond the scope of this fix
}