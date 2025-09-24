package main

import "testing"

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "with version set",
			version:  "1.2.3",
			expected: "1.2.3",
		},
		{
			name:     "with empty version",
			version:  "",
			expected: "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original version
			original := version
			defer func() { version = original }()

			// Set test version
			version = tt.version

			// Test getVersion
			result := getVersion()
			if result != tt.expected {
				t.Errorf("getVersion() = %v, want %v", result, tt.expected)
			}
		})
	}
}