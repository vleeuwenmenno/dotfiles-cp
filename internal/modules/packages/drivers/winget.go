package drivers

import (
	"fmt"
	"runtime"
	"strings"
)

// WingetDriver implements PackageDriver for Windows Package Manager (winget)
type WingetDriver struct {
	*BaseDriver
}

// NewWingetDriver creates a new Winget driver
func NewWingetDriver() *WingetDriver {
	return &WingetDriver{
		BaseDriver: NewBaseDriver("winget", "winget"),
	}
}

// IsPackageInstalled checks if a package is installed via Winget
func (d *WingetDriver) IsPackageInstalled(packageName string) (bool, error) {
	return d.IsPackageInstalledCached(packageName, d.fetchAllInstalledPackages)
}

// fetchAllInstalledPackages fetches all installed packages from Winget
func (d *WingetDriver) fetchAllInstalledPackages() (map[string]bool, error) {
	output, err := d.RunCommand("list")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	packages := make(map[string]bool)
	lines := strings.Split(output, "\n")
	headerPassed := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip until we pass the header
		if strings.Contains(line, "---") {
			headerPassed = true
			continue
		}
		if !headerPassed {
			continue
		}

		// Winget output format: "Name Id Version Available Source"
		// Extract both name and ID for matching
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			name := fields[0]
			id := fields[1]

			// Store both name and ID for lookups
			packages[name] = true
			packages[id] = true
			packages[strings.ToLower(name)] = true
			packages[strings.ToLower(id)] = true
		}
	}

	return packages, nil
}

// InstallPackage installs a package using Winget
func (d *WingetDriver) InstallPackage(packageName string) error {
	output, err := d.RunCommand("install", "--exact", "--silent", "--accept-package-agreements", "--accept-source-agreements", packageName)
	if err != nil {
		return fmt.Errorf("failed to install package %s via Winget: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// UninstallPackage uninstalls a package using Winget
func (d *WingetDriver) UninstallPackage(packageName string) error {
	output, err := d.RunCommand("uninstall", "--exact", "--silent", packageName)
	if err != nil {
		return fmt.Errorf("failed to uninstall package %s via Winget: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// SearchPackage searches for packages using Winget
func (d *WingetDriver) SearchPackage(packageName string) ([]string, error) {
	output, err := d.RunCommand("search", packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to search for package %s: %w", packageName, err)
	}

	var packages []string
	lines := strings.Split(output, "\n")
	headerPassed := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip until we pass the header
		if strings.Contains(line, "---") {
			headerPassed = true
			continue
		}
		if !headerPassed {
			continue
		}

		// Extract package ID from the line
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			// Usually the second field is the package ID
			packages = append(packages, fields[1])
		}
	}

	return packages, nil
}

// GetPackageInfo gets information about an installed package
func (d *WingetDriver) GetPackageInfo(packageName string) (map[string]string, error) {
	output, err := d.RunCommand("list", "--exact", packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to get package info for %s: %w", packageName, err)
	}

	info := make(map[string]string)
	lines := strings.Split(output, "\n")
	headerPassed := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip until we pass the header
		if strings.Contains(line, "---") {
			headerPassed = true
			continue
		}
		if !headerPassed {
			continue
		}

		// Check if this line contains our package
		if strings.Contains(strings.ToLower(line), strings.ToLower(packageName)) {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				info["name"] = fields[0]
				info["id"] = fields[1]
				info["version"] = fields[2]
				info["manager"] = "winget"
				break
			}
		}
	}

	if len(info) == 0 {
		return nil, fmt.Errorf("package %s not found", packageName)
	}

	return info, nil
}

// GetAllInstalledPackages returns a map of all installed packages
func (d *WingetDriver) GetAllInstalledPackages() (map[string]bool, error) {
	return d.fetchAllInstalledPackages()
}

// IsAvailable overrides the base implementation to check platform compatibility
func (d *WingetDriver) IsAvailable() bool {
	// Winget is only available on Windows
	if runtime.GOOS != "windows" {
		return false
	}

	return d.BaseDriver.IsAvailable()
}
