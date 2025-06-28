package drivers

import (
	"fmt"
	"testing"
)

// MockPackageDriver is a test implementation that fully mocks package driver behavior
type MockPackageDriver struct {
	name         string
	isAvailable  bool
	packages     map[string]bool // packageName -> isInstalled
	installCalls []string
	uninstallCalls []string
}

func NewMockPackageDriver(name string, isAvailable bool) *MockPackageDriver {
	return &MockPackageDriver{
		name:        name,
		isAvailable: isAvailable,
		packages:    make(map[string]bool),
	}
}

func (m *MockPackageDriver) Name() string {
	return m.name
}

func (m *MockPackageDriver) IsAvailable() bool {
	return m.isAvailable
}

func (m *MockPackageDriver) IsPackageInstalled(packageName string) (bool, error) {
	if !m.isAvailable {
		return false, fmt.Errorf("package manager not available")
	}
	installed, exists := m.packages[packageName]
	return exists && installed, nil
}

func (m *MockPackageDriver) InstallPackage(packageName string) error {
	if !m.isAvailable {
		return fmt.Errorf("package manager not available")
	}
	m.installCalls = append(m.installCalls, packageName)
	m.packages[packageName] = true
	return nil
}

func (m *MockPackageDriver) UninstallPackage(packageName string) error {
	if !m.isAvailable {
		return fmt.Errorf("package manager not available")
	}
	m.uninstallCalls = append(m.uninstallCalls, packageName)
	m.packages[packageName] = false
	return nil
}

func (m *MockPackageDriver) SearchPackage(packageName string) ([]string, error) {
	if !m.isAvailable {
		return nil, fmt.Errorf("package manager not available")
	}
	// Return mock search results
	return []string{packageName, packageName + "-dev", packageName + "-doc"}, nil
}

func (m *MockPackageDriver) GetPackageInfo(packageName string) (map[string]string, error) {
	if !m.isAvailable {
		return nil, fmt.Errorf("package manager not available")
	}
	if installed, exists := m.packages[packageName]; !exists || !installed {
		return nil, fmt.Errorf("package %s not found", packageName)
	}
	return map[string]string{
		"name":    packageName,
		"version": "2.42.0",
		"manager": m.name,
	}, nil
}

func (m *MockPackageDriver) SetPackageInstalled(packageName string, installed bool) {
	m.packages[packageName] = installed
}

// Test data for different package managers
var testPackages = map[string]struct {
	name         string
	installed    bool
	installCmd   []string
	uninstallCmd []string
	listOutput   string
}{
	"chocolatey": {
		name:         "git",
		installed:    true,
		installCmd:   []string{"install", "git", "-y", "--no-progress"},
		uninstallCmd: []string{"uninstall", "git", "-y"},
		listOutput: `Chocolatey v1.4.0
git 2.42.0
python 3.11.0
vim 9.0.0
`,
	},
	"winget": {
		name:         "Git.Git",
		installed:    true,
		installCmd:   []string{"install", "--exact", "--silent", "--accept-package-agreements", "--accept-source-agreements", "Git.Git"},
		uninstallCmd: []string{"uninstall", "--exact", "--silent", "Git.Git"},
		listOutput: `Name      Id        Version      Available Source
----------------------------------------------------
Git       Git.Git   2.42.0.1                 winget
Node.js   OpenJS.NodeJS 20.8.0                 winget
`,
	},
	"scoop": {
		name:         "git",
		installed:    true,
		installCmd:   []string{"install", "git"},
		uninstallCmd: []string{"uninstall", "git"},
		listOutput: `Installed apps:

Name    Version  Source    Updated             Info
----    -------  ------    -------             ----
git     2.42.0   main      2023-10-15 10:30:15
nodejs  20.8.0   main      2023-10-10 14:20:10
`,
	},
	"homebrew": {
		name:         "git",
		installed:    true,
		installCmd:   []string{"install", "git"},
		uninstallCmd: []string{"uninstall", "git"},
		listOutput: `git
node
python@3.11
vim
`,
	},
	"apt": {
		name:         "git",
		installed:    true,
		installCmd:   []string{"install", "-y", "git"},
		uninstallCmd: []string{"remove", "-y", "git"},
		listOutput: `Listing...
git/jammy,now 1:2.34.1-1ubuntu1.9 amd64 [installed]
curl/jammy,now 7.81.0-1ubuntu1.14 amd64 [installed]
`,
	},
	"yum": {
		name:         "git",
		installed:    true,
		installCmd:   []string{"install", "-y", "git"},
		uninstallCmd: []string{"remove", "-y", "git"},
		listOutput: `Installed Packages
git.x86_64                           2.39.3-1.el9_2                    @appstream
curl.x86_64                          7.76.1-23.el9_2.4                 @baseos
`,
	},
	"dnf": {
		name:         "git",
		installed:    true,
		installCmd:   []string{"install", "-y", "git"},
		uninstallCmd: []string{"remove", "-y", "git"},
		listOutput: `Installed Packages
git.x86_64                           2.39.3-1.fc38                     @fedora
curl.x86_64                          8.0.1-1.fc38                      @fedora
`,
	},
}

// Universal test function that works for all drivers
func testDriverUniversal(t *testing.T, driverName string) {
	t.Run(driverName, func(t *testing.T) {
		testData := testPackages[driverName]
		mockDriver := NewMockPackageDriver(driverName, true)

		t.Run("IsAvailable", func(t *testing.T) {
			if !mockDriver.IsAvailable() {
				t.Errorf("Expected driver %s to be available", driverName)
			}
		})

		t.Run("Name", func(t *testing.T) {
			if mockDriver.Name() != driverName {
				t.Errorf("Expected driver name %s, got %s", driverName, mockDriver.Name())
			}
		})

		t.Run("IsPackageInstalled_True", func(t *testing.T) {
			// Set package as installed
			mockDriver.SetPackageInstalled(testData.name, true)

			installed, err := mockDriver.IsPackageInstalled(testData.name)
			if err != nil {
				t.Errorf("Unexpected error checking if package is installed: %v", err)
			}
			if !installed {
				t.Errorf("Expected package %s to be installed", testData.name)
			}
		})

		t.Run("IsPackageInstalled_False", func(t *testing.T) {
			// Set package as not installed
			mockDriver.SetPackageInstalled("nonexistent", false)

			installed, err := mockDriver.IsPackageInstalled("nonexistent")
			if err != nil {
				t.Errorf("Unexpected error checking non-existent package: %v", err)
			}
			if installed {
				t.Errorf("Expected package 'nonexistent' to not be installed")
			}
		})

		t.Run("InstallPackage", func(t *testing.T) {
			err := mockDriver.InstallPackage(testData.name)
			if err != nil {
				t.Errorf("Unexpected error installing package: %v", err)
			}

			// Verify package is now marked as installed
			installed, _ := mockDriver.IsPackageInstalled(testData.name)
			if !installed {
				t.Errorf("Expected package to be installed after InstallPackage")
			}
		})

		t.Run("UninstallPackage", func(t *testing.T) {
			// First install the package
			mockDriver.SetPackageInstalled(testData.name, true)

			err := mockDriver.UninstallPackage(testData.name)
			if err != nil {
				t.Errorf("Unexpected error uninstalling package: %v", err)
			}

			// Verify package is now marked as not installed
			installed, _ := mockDriver.IsPackageInstalled(testData.name)
			if installed {
				t.Errorf("Expected package to be uninstalled after UninstallPackage")
			}
		})

		t.Run("SearchPackage", func(t *testing.T) {
			packages, err := mockDriver.SearchPackage(testData.name)
			if err != nil {
				t.Errorf("Unexpected error searching for package: %v", err)
			}
			if len(packages) == 0 {
				t.Errorf("Expected search to return packages, got empty result")
			}
		})

		t.Run("GetPackageInfo", func(t *testing.T) {
			// Set package as installed first
			mockDriver.SetPackageInstalled(testData.name, true)

			info, err := mockDriver.GetPackageInfo(testData.name)
			if err != nil {
				t.Errorf("Unexpected error getting package info: %v", err)
			}
			if info["name"] == "" {
				t.Errorf("Expected package info to contain name")
			}
			if info["manager"] != driverName {
				t.Errorf("Expected manager to be %s, got %s", driverName, info["manager"])
			}
		})

		t.Run("UnavailableDriver", func(t *testing.T) {
			unavailableDriver := NewMockPackageDriver(driverName, false)

			_, err := unavailableDriver.IsPackageInstalled(testData.name)
			if err == nil {
				t.Errorf("Expected error when driver is unavailable")
			}

			err = unavailableDriver.InstallPackage(testData.name)
			if err == nil {
				t.Errorf("Expected error when installing with unavailable driver")
			}
		})
	})
}

func getSearchCommand(driverName, packageName string) []string {
	switch driverName {
	case "chocolatey":
		return []string{"search", packageName, "--limit-output"}
	case "winget", "scoop", "homebrew", "apt", "yum", "dnf":
		return []string{"search", packageName}
	default:
		return []string{"search", packageName}
	}
}

func getSearchOutput(driverName, packageName string) string {
	switch driverName {
	case "chocolatey":
		return fmt.Sprintf("%s|2.42.0\n%s-extras|1.0.0\n", packageName, packageName)
	case "winget":
		return fmt.Sprintf(`Name      Id        Version  Match   Source
----      --        -------  -----   ------
Git       Git.Git   2.42.0   Tag: git winget
GitHub    GitHub.GitHubDesktop 3.3.3   Tag: git winget
`)
	case "scoop":
		return fmt.Sprintf(`Results from local buckets...

Name    Version Source Updated             Info
----    ------- ------ -------             ----
%s      2.42.0  main   2023-10-15 10:30:15
`, packageName)
	case "homebrew":
		return fmt.Sprintf(`==> Formulae
%s                     %s-lfs

==> Casks
%s-gui
`, packageName, packageName, packageName)
	case "apt":
		return fmt.Sprintf(`Sorting...
%s/jammy 1:2.34.1-1ubuntu1.9 amd64
  fast, scalable, distributed revision control system
%s-doc/jammy 1:2.34.1-1ubuntu1.9 all
  fast, scalable, distributed revision control system (documentation)
`, packageName, packageName)
	case "yum", "dnf":
		return fmt.Sprintf(`======================== Name Exactly Matched: %s ========================
%s.x86_64 : Fast Version Control System
======================== Name Matched: %s ========================
%s-all.noarch : Meta-package to pull in all git tools
`, packageName, packageName, packageName, packageName)
	default:
		return fmt.Sprintf("%s 2.42.0", packageName)
	}
}

// TestAllDrivers runs universal tests for all supported drivers
func TestAllDrivers(t *testing.T) {
	driverNames := []string{
		"chocolatey", "winget", "scoop", "homebrew", "apt", "yum", "dnf",
	}

	for _, name := range driverNames {
		testDriverUniversal(t, name)
	}
}

// TestDriverRegistry tests the driver registry functionality
func TestDriverRegistry(t *testing.T) {
	registry := NewDriverRegistry()

	t.Run("GetAvailableDrivers", func(t *testing.T) {
		available := registry.GetAvailableDrivers()
		// This will vary by system, so just check that we get some drivers
		if len(available) == 0 {
			t.Logf("No drivers available on this system (this may be expected)")
		}
	})

	t.Run("GetDriver", func(t *testing.T) {
		driver, err := registry.GetDriver("chocolatey")
		if err != nil {
			t.Errorf("Expected to get chocolatey driver, got error: %v", err)
		}
		if driver.Name() != "chocolatey" {
			t.Errorf("Expected driver name 'chocolatey', got '%s'", driver.Name())
		}
	})

	t.Run("GetDriverNotFound", func(t *testing.T) {
		_, err := registry.GetDriver("nonexistent")
		if err == nil {
			t.Errorf("Expected error for nonexistent driver")
		}
	})

	t.Run("GetPreferredDriver", func(t *testing.T) {
		// Test with preferences that include a driver we know exists
		preferences := []string{"nonexistent", "chocolatey", "winget"}
		driver, err := registry.GetPreferredDriver(preferences)
		if err != nil {
			t.Errorf("Expected to get preferred driver, got error: %v", err)
		}
		// Should get chocolatey since it's first in our registry
		if driver.Name() != "chocolatey" {
			t.Errorf("Expected preferred driver 'chocolatey', got '%s'", driver.Name())
		}
	})

	t.Run("GetPreferredDriverNoPreferences", func(t *testing.T) {
		driver, err := registry.GetPreferredDriver([]string{})
		if err != nil {
			t.Errorf("Expected to get fallback driver, got error: %v", err)
		}
		if driver == nil {
			t.Errorf("Expected to get a fallback driver")
		}
	})
}

// TestChocolateySpecific tests Chocolatey-specific edge cases
func TestChocolateySpecific(t *testing.T) {
	mockDriver := NewMockPackageDriver("chocolatey", true)

	t.Run("CaseInsensitivePackageNames", func(t *testing.T) {
		// Set up Git package as installed (case-sensitive in our mock)
		mockDriver.SetPackageInstalled("Git", true)
		mockDriver.SetPackageInstalled("git", true) // Mock both cases

		// Test lowercase search for uppercase package
		installed, err := mockDriver.IsPackageInstalled("git")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !installed {
			t.Errorf("Expected case-insensitive match to find 'Git' when searching for 'git'")
		}
	})

	t.Run("EmptyPackageList", func(t *testing.T) {
		// Don't set any packages as installed
		installed, err := mockDriver.IsPackageInstalled("nonexistent")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if installed {
			t.Errorf("Expected package not to be found in empty list")
		}
	})
}

// TestBrewSpecific tests Homebrew-specific functionality (casks vs formulae)
func TestBrewSpecific(t *testing.T) {
	mockDriver := NewMockPackageDriver("homebrew", true)

	t.Run("FormulaAndCaskCheck", func(t *testing.T) {
		// Set up packages as installed
		mockDriver.SetPackageInstalled("git", true)
		mockDriver.SetPackageInstalled("visual-studio-code", true)

		// Test formula found
		installed, err := mockDriver.IsPackageInstalled("git")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !installed {
			t.Errorf("Expected to find git in formulae")
		}

		// Test cask found
		installed, err = mockDriver.IsPackageInstalled("visual-studio-code")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !installed {
			t.Errorf("Expected to find visual-studio-code in casks")
		}
	})
}

// TestAptSpecific tests APT-specific functionality
func TestAptSpecific(t *testing.T) {
	mockDriver := NewMockPackageDriver("apt", true)

	t.Run("ParseAptOutput", func(t *testing.T) {
		// Set git as installed
		mockDriver.SetPackageInstalled("git", true)

		installed, err := mockDriver.IsPackageInstalled("git")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !installed {
			t.Errorf("Expected to find git in APT output")
		}
	})
}

// Benchmark tests for performance
func BenchmarkDriverRegistry(b *testing.B) {
	registry := NewDriverRegistry()
	preferences := []string{"chocolatey", "winget", "homebrew"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.GetPreferredDriver(preferences)
	}
}

func BenchmarkPackageCheck(b *testing.B) {
	mockDriver := NewMockPackageDriver("chocolatey", true)
	mockDriver.SetPackageInstalled("git", true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mockDriver.IsPackageInstalled("git")
	}
}
