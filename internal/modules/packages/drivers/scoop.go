package drivers

import (
	"fmt"
	"strings"
)

// ScoopDriver implements PackageDriver for Scoop package manager
type ScoopDriver struct {
	*BaseDriver
}

// NewScoopDriver creates a new Scoop driver
func NewScoopDriver() *ScoopDriver {
	return &ScoopDriver{
		BaseDriver: NewBaseDriver("scoop", "scoop"),
	}
}

// IsPackageInstalled checks if a package is installed via Scoop
func (d *ScoopDriver) IsPackageInstalled(packageName string) (bool, error) {
	return d.IsPackageInstalledCached(packageName, d.fetchAllInstalledPackages)
}

// fetchAllInstalledPackages fetches all installed packages from Scoop
func (d *ScoopDriver) fetchAllInstalledPackages() (map[string]bool, error) {
	output, err := d.RunCommand("list")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	packages := make(map[string]bool)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header lines
		if strings.Contains(line, "Installed apps:") || strings.Contains(line, "Name") {
			continue
		}

		// Scoop output format: "name version [bucket] *extras"
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			packageName := parts[0]
			packages[packageName] = true
			packages[strings.ToLower(packageName)] = true
		}
	}

	return packages, nil
}

// InstallPackage installs a package using Scoop
func (d *ScoopDriver) InstallPackage(packageName string) error {
	output, err := d.RunCommand("install", packageName)
	if err != nil {
		return fmt.Errorf("failed to install package %s via Scoop: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// UninstallPackage uninstalls a package using Scoop
func (d *ScoopDriver) UninstallPackage(packageName string) error {
	output, err := d.RunCommand("uninstall", packageName)
	if err != nil {
		return fmt.Errorf("failed to uninstall package %s via Scoop: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// SearchPackage searches for packages using Scoop
func (d *ScoopDriver) SearchPackage(packageName string) ([]string, error) {
	output, err := d.RunCommand("search", packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to search for package %s: %w", packageName, err)
	}

	var packages []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip informational lines
		if strings.Contains(line, "Results from") || strings.Contains(line, "Name") {
			continue
		}

		// Scoop search output format: "name (version) [bucket]"
		// Extract the package name (first part before space or parenthesis)
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			packageName := strings.Split(parts[0], "(")[0]
			packages = append(packages, packageName)
		}
	}

	return packages, nil
}

// GetPackageInfo gets information about an installed package
func (d *ScoopDriver) GetPackageInfo(packageName string) (map[string]string, error) {
	output, err := d.RunCommand("list", packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to get package info for %s: %w", packageName, err)
	}

	info := make(map[string]string)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header lines
		if strings.Contains(line, "Installed apps:") || strings.Contains(line, "Name") {
			continue
		}

		// Look for the package in the output
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			if strings.EqualFold(parts[0], packageName) {
				info["name"] = parts[0]
				info["version"] = parts[1]
				info["manager"] = "scoop"
				if len(parts) >= 3 {
					// Extract bucket name (remove brackets)
					bucket := strings.Trim(parts[2], "[]")
					info["bucket"] = bucket
				}
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
func (d *ScoopDriver) GetAllInstalledPackages() (map[string]bool, error) {
	return d.fetchAllInstalledPackages()
}
