package drivers

import (
	"fmt"
	"testing"
	"time"
)

// SlowMockDriver simulates the old behavior without caching
type SlowMockDriver struct {
	*BaseDriver
	commandCallCount int
	packages         map[string]bool
}

func NewSlowMockDriver(name string) *SlowMockDriver {
	return &SlowMockDriver{
		BaseDriver: NewBaseDriver(name, name),
		packages:   make(map[string]bool),
	}
}

func (d *SlowMockDriver) SetPackageInstalled(packageName string, installed bool) {
	d.packages[packageName] = installed
}

func (d *SlowMockDriver) IsPackageInstalled(packageName string) (bool, error) {
	// Simulate the old behavior - make a command call every time
	d.commandCallCount++

	// Simulate command execution time
	time.Sleep(10 * time.Millisecond)

	installed, exists := d.packages[packageName]
	return exists && installed, nil
}

// FastMockDriver simulates the new behavior with caching
type FastMockDriver struct {
	*BaseDriver
	commandCallCount int
	packages         map[string]bool
}

func NewFastMockDriver(name string) *FastMockDriver {
	return &FastMockDriver{
		BaseDriver: NewBaseDriver(name, name),
		packages:   make(map[string]bool),
	}
}

func (d *FastMockDriver) SetPackageInstalled(packageName string, installed bool) {
	d.packages[packageName] = installed
}

func (d *FastMockDriver) IsPackageInstalled(packageName string) (bool, error) {
	return d.IsPackageInstalledCached(packageName, d.fetchAllPackages)
}

func (d *FastMockDriver) fetchAllPackages() (map[string]bool, error) {
	// This should only be called once due to caching
	d.commandCallCount++

	// Simulate fetching all packages (longer than individual check)
	time.Sleep(50 * time.Millisecond)

	return d.packages, nil
}

func TestCachingPerformance(t *testing.T) {
	packageList := []string{
		"git", "curl", "vim", "nodejs", "python", "docker", "kubectl", "terraform", "aws-cli", "vscode",
		"chrome", "firefox", "slack", "zoom", "teams", "discord", "spotify", "steam", "obs", "gimp",
	}

	t.Run("SlowDriver_Without_Caching", func(t *testing.T) {
		driver := NewSlowMockDriver("test")

		// Set up some packages as installed
		for i, pkg := range packageList {
			driver.SetPackageInstalled(pkg, i%2 == 0) // Every other package is installed
		}

		start := time.Now()

		// Check all packages
		for _, pkg := range packageList {
			_, err := driver.IsPackageInstalled(pkg)
			if err != nil {
				t.Errorf("Error checking package %s: %v", pkg, err)
			}
		}

		duration := time.Since(start)
		t.Logf("Slow driver: %d command calls, took %v", driver.commandCallCount, duration)

		if driver.commandCallCount != len(packageList) {
			t.Errorf("Expected %d command calls, got %d", len(packageList), driver.commandCallCount)
		}
	})

	t.Run("FastDriver_With_Caching", func(t *testing.T) {
		driver := NewFastMockDriver("test")

		// Set up some packages as installed
		for i, pkg := range packageList {
			driver.SetPackageInstalled(pkg, i%2 == 0) // Every other package is installed
		}

		start := time.Now()

		// Check all packages
		for _, pkg := range packageList {
			_, err := driver.IsPackageInstalled(pkg)
			if err != nil {
				t.Errorf("Error checking package %s: %v", pkg, err)
			}
		}

		duration := time.Since(start)
		t.Logf("Fast driver: %d command calls, took %v", driver.commandCallCount, duration)

		if driver.commandCallCount != 1 {
			t.Errorf("Expected 1 command call with caching, got %d", driver.commandCallCount)
		}
	})
}

func BenchmarkSlowDriverMultiplePackages(b *testing.B) {
	packageList := []string{"git", "curl", "vim", "nodejs", "python"}
	driver := NewSlowMockDriver("test")

	for _, pkg := range packageList {
		driver.SetPackageInstalled(pkg, true)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pkg := range packageList {
			_, _ = driver.IsPackageInstalled(pkg)
		}
	}
}

func BenchmarkFastDriverMultiplePackages(b *testing.B) {
	packageList := []string{"git", "curl", "vim", "nodejs", "python"}
	driver := NewFastMockDriver("test")

	for _, pkg := range packageList {
		driver.SetPackageInstalled(pkg, true)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pkg := range packageList {
			_, _ = driver.IsPackageInstalled(pkg)
		}
		// Clear cache between iterations to simulate real usage
		driver.GetCache().InvalidateCache()
	}
}

func TestCacheExpiration(t *testing.T) {
	driver := NewFastMockDriver("test")
	driver.SetPackageInstalled("git", true)

	// Set a very short cache duration for testing
	driver.GetCache().cacheDuration = 100 * time.Millisecond

	// First call should fetch
	_, err := driver.IsPackageInstalled("git")
	if err != nil {
		t.Errorf("Error checking package: %v", err)
	}

	if driver.commandCallCount != 1 {
		t.Errorf("Expected 1 command call, got %d", driver.commandCallCount)
	}

	// Second call should use cache
	_, err = driver.IsPackageInstalled("git")
	if err != nil {
		t.Errorf("Error checking package: %v", err)
	}

	if driver.commandCallCount != 1 {
		t.Errorf("Expected 1 command call (cached), got %d", driver.commandCallCount)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call should fetch again
	_, err = driver.IsPackageInstalled("git")
	if err != nil {
		t.Errorf("Error checking package: %v", err)
	}

	if driver.commandCallCount != 2 {
		t.Errorf("Expected 2 command calls (cache expired), got %d", driver.commandCallCount)
	}
}

func TestConcurrentAccess(t *testing.T) {
	driver := NewFastMockDriver("test")
	driver.SetPackageInstalled("git", true)

	// Test concurrent access to the cache
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, err := driver.IsPackageInstalled("git")
			if err != nil {
				t.Errorf("Error in concurrent access: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should only have made one command call due to caching
	if driver.commandCallCount != 1 {
		t.Errorf("Expected 1 command call with concurrent access, got %d", driver.commandCallCount)
	}
}

func ExampleCachingBenefit() {
	// Create a fast driver with caching
	driver := NewFastMockDriver("chocolatey")

	// Set up some packages
	packages := []string{"git", "nodejs", "python", "docker", "vscode"}
	for _, pkg := range packages {
		driver.SetPackageInstalled(pkg, true)
	}

	fmt.Println("Checking multiple packages with caching:")

	start := time.Now()
	for _, pkg := range packages {
		installed, _ := driver.IsPackageInstalled(pkg)
		fmt.Printf("Package %s: installed=%v\n", pkg, installed)
	}
	duration := time.Since(start)

	fmt.Printf("Total time: %v\n", duration)
	fmt.Printf("Command calls made: %d\n", driver.commandCallCount)
	fmt.Println("Note: Only 1 command call needed to check all packages!")

	// Output:
	// Checking multiple packages with caching:
	// Package git: installed=true
	// Package nodejs: installed=true
	// Package python: installed=true
	// Package docker: installed=true
	// Package vscode: installed=true
	// Total time: 50ms
	// Command calls made: 1
	// Note: Only 1 command call needed to check all packages!
}

func TestRealWorldScenario(t *testing.T) {
	// Simulate a real dotfiles scenario with many packages
	packages := []string{
		// Development tools
		"git", "nodejs", "python", "golang", "rust", "java", "dotnet",
		// Editors and IDEs
		"vscode", "vim", "emacs", "sublime-text", "intellij",
		// Browsers
		"chrome", "firefox", "edge", "safari",
		// Communication
		"slack", "discord", "teams", "zoom", "skype",
		// Media
		"vlc", "spotify", "obs", "gimp", "blender",
		// System tools
		"curl", "wget", "htop", "tree", "jq", "docker", "kubectl",
		// Utilities
		"7zip", "notepad++", "winrar", "ccleaner",
	}

	t.Run("WithoutCaching", func(t *testing.T) {
		driver := NewSlowMockDriver("chocolatey")

		// Set up packages (mix of installed and not installed)
		for i, pkg := range packages {
			driver.SetPackageInstalled(pkg, i%3 == 0) // Every 3rd package is installed
		}

		start := time.Now()
		installedCount := 0

		for _, pkg := range packages {
			installed, err := driver.IsPackageInstalled(pkg)
			if err != nil {
				t.Errorf("Error checking package %s: %v", pkg, err)
			}
			if installed {
				installedCount++
			}
		}

		duration := time.Since(start)
		t.Logf("Without caching: %d packages, %d installed, %d commands, %v total time",
			len(packages), installedCount, driver.commandCallCount, duration)

		// Should make one command per package
		if driver.commandCallCount != len(packages) {
			t.Errorf("Expected %d commands, got %d", len(packages), driver.commandCallCount)
		}
	})

	t.Run("WithCaching", func(t *testing.T) {
		driver := NewFastMockDriver("chocolatey")

		// Set up packages (mix of installed and not installed)
		for i, pkg := range packages {
			driver.SetPackageInstalled(pkg, i%3 == 0) // Every 3rd package is installed
		}

		start := time.Now()
		installedCount := 0

		for _, pkg := range packages {
			installed, err := driver.IsPackageInstalled(pkg)
			if err != nil {
				t.Errorf("Error checking package %s: %v", pkg, err)
			}
			if installed {
				installedCount++
			}
		}

		duration := time.Since(start)
		t.Logf("With caching: %d packages, %d installed, %d commands, %v total time",
			len(packages), installedCount, driver.commandCallCount, duration)

		// Should make only one command total
		if driver.commandCallCount != 1 {
			t.Errorf("Expected 1 command with caching, got %d", driver.commandCallCount)
		}

		// Performance improvement should be significant
		expectedMinTime := time.Duration(len(packages)) * 10 * time.Millisecond // Old way
		if duration >= expectedMinTime {
			t.Errorf("Caching should be much faster. Expected < %v, got %v", expectedMinTime, duration)
		}
	})
}
