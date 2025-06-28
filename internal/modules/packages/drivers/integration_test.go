//go:build integration

package drivers

import (
	"testing"
)

// Integration tests that run against real package managers
// Run with: go test -tags=integration ./internal/modules/packages/drivers

func TestRealDriversIntegration(t *testing.T) {
	registry := NewDriverRegistry()
	availableDrivers := registry.GetAvailableDrivers()

	if len(availableDrivers) == 0 {
		t.Skip("No package managers available on this system")
	}

	for _, driver := range availableDrivers {
		t.Run(driver.Name(), func(t *testing.T) {
			testRealDriver(t, driver)
		})
	}
}

func testRealDriver(t *testing.T, driver PackageDriver) {
	driverName := driver.Name()

	t.Run("IsAvailable", func(t *testing.T) {
		if !driver.IsAvailable() {
			t.Errorf("Driver %s should be available", driverName)
		}
	})

	// Test with a package that is unlikely to be installed
	testPackage := getTestPackage(driverName)

	t.Run("CheckNonExistentPackage", func(t *testing.T) {
		installed, err := driver.IsPackageInstalled("nonexistent-package-12345")
		if err != nil {
			t.Logf("Warning: Error checking non-existent package (this may be expected): %v", err)
		}
		if installed {
			t.Errorf("Non-existent package should not be installed")
		}
	})

	t.Run("SearchPackage", func(t *testing.T) {
		packages, err := driver.SearchPackage(testPackage)
		if err != nil {
			t.Logf("Warning: Search failed (this may be expected on some systems): %v", err)
		} else if len(packages) == 0 {
			t.Logf("Warning: No packages found for search term '%s'", testPackage)
		} else {
			t.Logf("Found %d packages for search term '%s'", len(packages), testPackage)
		}
	})

	// Test with commonly available packages
	commonPackages := getCommonPackages(driverName)
	for _, pkg := range commonPackages {
		t.Run("CheckCommonPackage_"+pkg, func(t *testing.T) {
			installed, err := driver.IsPackageInstalled(pkg)
			if err != nil {
				t.Logf("Note: Error checking package '%s': %v", pkg, err)
			} else {
				t.Logf("Package '%s' installed: %v", pkg, installed)
			}

			// If the package is installed, try to get its info
			if installed {
				info, err := driver.GetPackageInfo(pkg)
				if err != nil {
					t.Logf("Note: Could not get info for package '%s': %v", pkg, err)
				} else {
					t.Logf("Package info for '%s': %+v", pkg, info)
					if info["name"] == "" {
						t.Errorf("Package info should contain name")
					}
					if info["manager"] != driverName {
						t.Errorf("Expected manager %s, got %s", driverName, info["manager"])
					}
				}
			}
		})
	}
}

func getTestPackage(driverName string) string {
	switch driverName {
	case "chocolatey":
		return "git"
	case "winget":
		return "git"
	case "scoop":
		return "git"
	case "homebrew":
		return "git"
	case "apt":
		return "git"
	case "yum":
		return "git"
	case "dnf":
		return "git"
	default:
		return "git"
	}
}

func getCommonPackages(driverName string) []string {
	switch driverName {
	case "chocolatey":
		return []string{"chocolatey"}
	case "winget":
		return []string{"Microsoft.WindowsTerminal"}
	case "scoop":
		return []string{"7zip"}
	case "homebrew":
		return []string{"git"}
	case "apt":
		return []string{"base-files", "libc6"}
	case "yum":
		return []string{"bash", "glibc"}
	case "dnf":
		return []string{"bash", "glibc"}
	default:
		return []string{}
	}
}

// TestDriverRegistryIntegration tests the driver registry with real drivers
func TestDriverRegistryIntegration(t *testing.T) {
	registry := NewDriverRegistry()

	t.Run("GetAvailableDrivers", func(t *testing.T) {
		available := registry.GetAvailableDrivers()
		t.Logf("Available drivers: %v", registry.GetAvailableDriverNames())

		for _, driver := range available {
			if !driver.IsAvailable() {
				t.Errorf("Driver %s reported as available but IsAvailable() returns false", driver.Name())
			}
		}
	})

	t.Run("GetPreferredDriverWithRealPreferences", func(t *testing.T) {
		// Test with platform-appropriate preferences
		var preferences []string
		availableNames := registry.GetAvailableDriverNames()

		if contains(availableNames, "chocolatey") {
			preferences = []string{"chocolatey", "winget", "scoop"}
		} else if contains(availableNames, "homebrew") {
			preferences = []string{"homebrew"}
		} else if contains(availableNames, "apt") {
			preferences = []string{"apt", "yum", "dnf"}
		} else {
			t.Skip("No supported package managers available")
		}

		driver, err := registry.GetPreferredDriver(preferences)
		if err != nil {
			t.Errorf("Failed to get preferred driver: %v", err)
		} else {
			t.Logf("Selected preferred driver: %s", driver.Name())
			if !driver.IsAvailable() {
				t.Errorf("Preferred driver should be available")
			}
		}
	})
}

// TestSpecificDriverBehavior tests specific behaviors for available drivers
func TestSpecificDriverBehavior(t *testing.T) {
	registry := NewDriverRegistry()

	// Test Chocolatey specifics if available
	if choco, err := registry.GetDriver("chocolatey"); err == nil && choco.IsAvailable() {
		t.Run("ChocolateySpecific", func(t *testing.T) {
			testChocolateyBehavior(t, choco)
		})
	}

	// Test Winget specifics if available
	if winget, err := registry.GetDriver("winget"); err == nil && winget.IsAvailable() {
		t.Run("WingetSpecific", func(t *testing.T) {
			testWingetBehavior(t, winget)
		})
	}

	// Test Homebrew specifics if available
	if brew, err := registry.GetDriver("homebrew"); err == nil && brew.IsAvailable() {
		t.Run("HomebrewSpecific", func(t *testing.T) {
			testHomebrewBehavior(t, brew)
		})
	}

	// Test APT specifics if available
	if apt, err := registry.GetDriver("apt"); err == nil && apt.IsAvailable() {
		t.Run("AptSpecific", func(t *testing.T) {
			testAptBehavior(t, apt)
		})
	}
}

func testChocolateyBehavior(t *testing.T, driver PackageDriver) {
	// Test case-insensitive behavior with chocolatey itself
	installed1, err1 := driver.IsPackageInstalled("chocolatey")
	installed2, err2 := driver.IsPackageInstalled("Chocolatey")

	if err1 == nil && err2 == nil {
		if installed1 != installed2 {
			t.Logf("Note: Case sensitivity difference - 'chocolatey': %v, 'Chocolatey': %v", installed1, installed2)
		}
	}
}

func testWingetBehavior(t *testing.T, driver PackageDriver) {
	// Test exact matching behavior
	packages, err := driver.SearchPackage("Microsoft")
	if err == nil && len(packages) > 0 {
		t.Logf("Found %d Microsoft packages", len(packages))
	}
}

func testHomebrewBehavior(t *testing.T, driver PackageDriver) {
	// Test formula vs cask detection
	// Most systems should have git as a formula
	installed, err := driver.IsPackageInstalled("git")
	if err == nil {
		t.Logf("Git installed via Homebrew: %v", installed)
	}
}

func testAptBehavior(t *testing.T, driver PackageDriver) {
	// Test with a package that should always be present
	installed, err := driver.IsPackageInstalled("base-files")
	if err == nil {
		t.Logf("base-files package installed: %v", installed)
		if !installed {
			t.Logf("Note: base-files not found, this might indicate parsing issues")
		}
	} else {
		t.Logf("Note: Error checking base-files: %v", err)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestDriverErrorHandling tests how drivers handle various error conditions
func TestDriverErrorHandling(t *testing.T) {
	registry := NewDriverRegistry()
	availableDrivers := registry.GetAvailableDrivers()

	if len(availableDrivers) == 0 {
		t.Skip("No package managers available")
	}

	for _, driver := range availableDrivers {
		t.Run(driver.Name()+"_ErrorHandling", func(t *testing.T) {
			// Test with empty package name
			_, err := driver.IsPackageInstalled("")
			if err != nil {
				t.Logf("Empty package name error (expected): %v", err)
			}

			// Test with package name containing special characters
			_, err = driver.IsPackageInstalled("test/package@version")
			if err != nil {
				t.Logf("Special characters package name error: %v", err)
			}

			// Test search with empty string
			_, err = driver.SearchPackage("")
			if err != nil {
				t.Logf("Empty search term error: %v", err)
			}
		})
	}
}

// BenchmarkRealDrivers benchmarks real driver operations
func BenchmarkRealDrivers(b *testing.B) {
	registry := NewDriverRegistry()
	availableDrivers := registry.GetAvailableDrivers()

	if len(availableDrivers) == 0 {
		b.Skip("No package managers available")
	}

	for _, driver := range availableDrivers {
		b.Run(driver.Name()+"_IsPackageInstalled", func(b *testing.B) {
			testPackage := getTestPackage(driver.Name())
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = driver.IsPackageInstalled(testPackage)
			}
		})
	}
}
