package jobs

import (
	"testing"
)

func TestEvaluateCondition(t *testing.T) {
	parser := NewJobParser("/test/path")

	// Test data - simulating platform variables
	variables := map[string]interface{}{
		"Platform": map[string]interface{}{
			"OS":         "linux",
			"Distro":     "Alpine Linux",
			"Arch":       "amd64",
			"IsElevated": false,
			"IsRoot":     true,
			"EmptyField": "",
		},
	}

	tests := []struct {
		name      string
		condition string
		expected  bool
		shouldErr bool
	}{
		// Basic equality tests
		{
			name:      "simple eq - true",
			condition: "eq .Platform.OS \"linux\"",
			expected:  true,
		},
		{
			name:      "simple eq - false",
			condition: "eq .Platform.OS \"windows\"",
			expected:  false,
		},
		{
			name:      "simple ne - true",
			condition: "ne .Platform.OS \"windows\"",
			expected:  true,
		},
		{
			name:      "simple ne - false",
			condition: "ne .Platform.OS \"linux\"",
			expected:  false,
		},

		// AND conditions
		{
			name:      "and - both true",
			condition: "and (eq .Platform.OS \"linux\") (eq .Platform.Distro \"Alpine Linux\")",
			expected:  true,
		},
		{
			name:      "and - first false",
			condition: "and (eq .Platform.OS \"windows\") (eq .Platform.Distro \"Alpine Linux\")",
			expected:  false,
		},
		{
			name:      "and - second false",
			condition: "and (eq .Platform.OS \"linux\") (eq .Platform.Distro \"Ubuntu\")",
			expected:  false,
		},
		{
			name:      "and - both false",
			condition: "and (eq .Platform.OS \"windows\") (eq .Platform.Distro \"Ubuntu\")",
			expected:  false,
		},

		// OR conditions
		{
			name:      "or - both true",
			condition: "or (eq .Platform.OS \"linux\") (eq .Platform.Distro \"Alpine Linux\")",
			expected:  true,
		},
		{
			name:      "or - first true",
			condition: "or (eq .Platform.OS \"linux\") (eq .Platform.Distro \"Ubuntu\")",
			expected:  true,
		},
		{
			name:      "or - second true",
			condition: "or (eq .Platform.OS \"windows\") (eq .Platform.Distro \"Alpine Linux\")",
			expected:  true,
		},
		{
			name:      "or - both false",
			condition: "or (eq .Platform.OS \"windows\") (eq .Platform.Distro \"Ubuntu\")",
			expected:  false,
		},

		// NOT conditions
		{
			name:      "not - true input",
			condition: "not (eq .Platform.OS \"windows\")",
			expected:  true,
		},
		{
			name:      "not - false input",
			condition: "not (eq .Platform.OS \"linux\")",
			expected:  false,
		},
		{
			name:      "not with boolean field",
			condition: "not .Platform.IsElevated",
			expected:  true,
		},

		// Complex nested conditions
		{
			name:      "complex - and with or",
			condition: "and (or (eq .Platform.OS \"linux\") (eq .Platform.OS \"darwin\")) (eq .Platform.Distro \"Alpine Linux\")",
			expected:  true,
		},
		{
			name:      "complex - or with and",
			condition: "or (and (eq .Platform.OS \"windows\") .Platform.IsElevated) (and (eq .Platform.OS \"linux\") .Platform.IsRoot)",
			expected:  true,
		},
		{
			name:      "complex - multiple ands",
			condition: "and (eq .Platform.OS \"linux\") (and (eq .Platform.Arch \"amd64\") (not .Platform.IsElevated))",
			expected:  true,
		},

		// Boolean field direct access
		{
			name:      "boolean field - true",
			condition: ".Platform.IsRoot",
			expected:  true,
		},
		{
			name:      "boolean field - false",
			condition: ".Platform.IsElevated",
			expected:  false,
		},

		// Error cases
		{
			name:      "invalid template syntax",
			condition: "eq .Platform.OS",
			shouldErr: true,
		},
		{
			name:      "missing field",
			condition: "eq .Platform.NonExistent \"value\"",
			expected:  false, // missingkey=zero should make this false
		},

		// String function tests
		{
			name:      "contains - true",
			condition: "contains .Platform.Distro \"Alpine\"",
			expected:  true,
		},
		{
			name:      "contains - false",
			condition: "contains .Platform.Distro \"Ubuntu\"",
			expected:  false,
		},
		{
			name:      "hasPrefix - true",
			condition: "hasPrefix .Platform.Distro \"Alpine\"",
			expected:  true,
		},
		{
			name:      "hasPrefix - false",
			condition: "hasPrefix .Platform.Distro \"Ubuntu\"",
			expected:  false,
		},
		{
			name:      "hasSuffix - true",
			condition: "hasSuffix .Platform.Distro \"Linux\"",
			expected:  true,
		},
		{
			name:      "hasSuffix - false",
			condition: "hasSuffix .Platform.Distro \"Windows\"",
			expected:  false,
		},
		{
			name:      "empty - false for non-empty",
			condition: "empty .Platform.OS",
			expected:  false,
		},
		{
			name:      "empty - true for empty",
			condition: "empty .Platform.EmptyField",
			expected:  true,
		},

		// Comparison function tests
		{
			name:      "gt - true",
			condition: "gt .Platform.Arch \"a\"", // "amd64" > "a"
			expected:  true,
		},
		{
			name:      "gt - false",
			condition: "gt .Platform.Arch \"z\"", // "amd64" < "z"
			expected:  false,
		},
		{
			name:      "lt - true",
			condition: "lt .Platform.Arch \"z\"", // "amd64" < "z"
			expected:  true,
		},
		{
			name:      "lt - false",
			condition: "lt .Platform.Arch \"a\"", // "amd64" > "a"
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.evaluateCondition(tt.condition, variables)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("evaluateCondition(%q) = %v, want %v", tt.condition, result, tt.expected)
			}
		})
	}
}

func TestEvaluateConditionWithDifferentPlatforms(t *testing.T) {
	parser := NewJobParser("/test/path")

	// Test with Windows platform
	windowsVars := map[string]interface{}{
		"Platform": map[string]interface{}{
			"OS":         "windows",
			"Distro":     "Windows",
			"IsElevated": true,
			"IsRoot":     false,
		},
	}

	// Test with macOS platform
	macVars := map[string]interface{}{
		"Platform": map[string]interface{}{
			"OS":         "darwin",
			"Distro":     "macOS",
			"IsElevated": false,
			"IsRoot":     false,
		},
	}

	tests := []struct {
		name      string
		condition string
		vars      map[string]interface{}
		expected  bool
	}{
		{
			name:      "windows specific condition",
			condition: "and (eq .Platform.OS \"windows\") .Platform.IsElevated",
			vars:      windowsVars,
			expected:  true,
		},
		{
			name:      "unix-like condition on windows",
			condition: "or (eq .Platform.OS \"linux\") (eq .Platform.OS \"darwin\")",
			vars:      windowsVars,
			expected:  false,
		},
		{
			name:      "unix-like condition on mac",
			condition: "or (eq .Platform.OS \"linux\") (eq .Platform.OS \"darwin\")",
			vars:      macVars,
			expected:  true,
		},
		{
			name:      "non-windows condition",
			condition: "not (eq .Platform.OS \"windows\")",
			vars:      macVars,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.evaluateCondition(tt.condition, tt.vars)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("evaluateCondition(%q) with %s platform = %v, want %v",
					tt.condition, tt.vars["Platform"].(map[string]interface{})["OS"], result, tt.expected)
			}
		})
	}
}

func TestEvaluateConditionRealWorldExamples(t *testing.T) {
	parser := NewJobParser("/test/path")

	// Alpine Linux variables
	alpineVars := map[string]interface{}{
		"Platform": map[string]interface{}{
			"OS":                      "linux",
			"Distro":                  "Alpine Linux",
			"AvailablePackageManagers": []string{"apk"},
		},
	}

	// Ubuntu variables
	ubuntuVars := map[string]interface{}{
		"Platform": map[string]interface{}{
			"OS":                      "linux",
			"Distro":                  "Ubuntu",
			"AvailablePackageManagers": []string{"apt"},
		},
	}

	tests := []struct {
		name      string
		condition string
		vars      map[string]interface{}
		expected  bool
	}{
		{
			name:      "alpine linux import condition",
			condition: "and (eq .Platform.OS \"linux\") (eq .Platform.Distro \"Alpine Linux\")",
			vars:      alpineVars,
			expected:  true,
		},
		{
			name:      "alpine linux import condition on ubuntu",
			condition: "and (eq .Platform.OS \"linux\") (eq .Platform.Distro \"Alpine Linux\")",
			vars:      ubuntuVars,
			expected:  false,
		},
		{
			name:      "any linux condition",
			condition: "eq .Platform.OS \"linux\"",
			vars:      alpineVars,
			expected:  true,
		},
		{
			name:      "debian-based condition on ubuntu",
			condition: "and (eq .Platform.OS \"linux\") (or (eq .Platform.Distro \"Ubuntu\") (eq .Platform.Distro \"Debian\"))",
			vars:      ubuntuVars,
			expected:  true,
		},
		{
			name:      "debian-based condition on alpine",
			condition: "and (eq .Platform.OS \"linux\") (or (eq .Platform.Distro \"Ubuntu\") (eq .Platform.Distro \"Debian\"))",
			vars:      alpineVars,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.evaluateCondition(tt.condition, tt.vars)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("evaluateCondition(%q) = %v, want %v", tt.condition, result, tt.expected)
			}
		})
	}
}
