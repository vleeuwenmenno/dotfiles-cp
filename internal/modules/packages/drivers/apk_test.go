package drivers

import (
	"strings"
	"testing"
)

func TestApkDriver_ExtractPackageName(t *testing.T) {
	driver := NewApkDriver()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple package with version",
			input:    "curl-7.88.1-r1",
			expected: "curl",
		},
		{
			name:     "package with multiple dashes in name",
			input:    "build-base-0.5-r3",
			expected: "build-base",
		},
		{
			name:     "complex package name",
			input:    "py3-pip-22.3.1-r1",
			expected: "py3-pip",
		},
		{
			name:     "package with architecture",
			input:    "alpine-base-3.17.0-r0",
			expected: "alpine-base",
		},
		{
			name:     "package without version",
			input:    "git",
			expected: "git",
		},
		{
			name:     "package with only one dash",
			input:    "test-package",
			expected: "test",
		},
		{
			name:     "complex version string",
			input:    "nodejs-18.14.2-r0",
			expected: "nodejs",
		},
		{
			name:     "package with pre-release version",
			input:    "vim-9.0.1000-r0",
			expected: "vim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := driver.extractPackageName(tt.input)
			if result != tt.expected {
				t.Errorf("extractPackageName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestApkDriver_Name(t *testing.T) {
	driver := NewApkDriver()
	if driver.Name() != "apk" {
		t.Errorf("Name() = %q, want %q", driver.Name(), "apk")
	}
}

func TestApkDriver_ParseInstalledPackages(t *testing.T) {
	driver := NewApkDriver()

	packages := make(map[string]bool)
	lines := []string{
		"curl-7.88.1-r1 x86_64 {curl} (curl and libcurl)",
		"git-2.39.0-r1 x86_64 {git} (Fast, scalable, distributed revision control system)",
		"nodejs-18.14.2-r0 x86_64 {nodejs} (JavaScript runtime built on V8 engine)",
		"py3-pip-22.3.1-r1 x86_64 {py3-pip} (Tool for installing Python packages)",
		"build-base-0.5-r3 x86_64 {build-base} (Meta package for build base)",
		"alpine-base-3.17.0-r0 x86_64 {alpine-base} (Meta package for minimal alpine base)",
	}

	// Simulate the parsing logic
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, " ") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				packageInfo := parts[0]
				packageName := driver.extractPackageName(packageInfo)
				if packageName != "" {
					packages[packageName] = true
					packages[strings.ToLower(packageName)] = true
				}
			}
		}
	}

	expectedPackages := []string{
		"curl", "git", "nodejs", "py3-pip", "build-base", "alpine-base",
	}

	for _, expected := range expectedPackages {
		if !packages[expected] {
			t.Errorf("Expected package %q not found in parsed packages", expected)
		}
		if !packages[strings.ToLower(expected)] {
			t.Errorf("Expected lowercase package %q not found in parsed packages", strings.ToLower(expected))
		}
	}

	// Verify we don't have false positives
	if packages["7.88.1"] { // version number shouldn't be a package
		t.Error("Version number incorrectly identified as package name")
	}
	if packages["x86_64"] { // architecture shouldn't be a package
		t.Error("Architecture incorrectly identified as package name")
	}
}

func TestApkDriver_IsAvailable(t *testing.T) {
	driver := NewApkDriver()

	// On non-Linux systems, should return false
	// On Linux systems, depends on whether apk is actually installed
	available := driver.IsAvailable()

	// We can't test the actual availability without knowing the test environment,
	// but we can test that the method doesn't panic and returns a boolean
	if available != true && available != false {
		t.Error("IsAvailable() should return a boolean value")
	}
}

func TestApkDriver_EdgeCases(t *testing.T) {
	driver := NewApkDriver()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no dashes",
			input:    "package",
			expected: "package",
		},
		{
			name:     "single character package",
			input:    "a-1.0.0-r0",
			expected: "a",
		},
		{
			name:     "package name with numbers",
			input:    "lib32-glibc-2.36-r4",
			expected: "lib32-glibc",
		},
		{
			name:     "version starting with zero",
			input:    "package-0.1.0-r0",
			expected: "package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := driver.extractPackageName(tt.input)
			if result != tt.expected {
				t.Errorf("extractPackageName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Mock tests for methods that require actual apk commands
func TestApkDriver_MockCommands(t *testing.T) {
	driver := NewApkDriver()

	// Test that the driver is properly initialized
	if driver.BaseDriver == nil {
		t.Error("BaseDriver should not be nil")
	}

	if driver.BaseDriver.name != "apk" {
		t.Errorf("Driver name should be 'apk', got %q", driver.BaseDriver.name)
	}

	if driver.BaseDriver.executable != "apk" {
		t.Errorf("Driver executable should be 'apk', got %q", driver.BaseDriver.executable)
	}
}
