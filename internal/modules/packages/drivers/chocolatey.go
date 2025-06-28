package drivers

import (
	"fmt"
	"strings"
)

// ChocolateyDriver implements PackageDriver for Chocolatey package manager
type ChocolateyDriver struct {
	*BaseDriver
}

// NewChocolateyDriver creates a new Chocolatey driver
func NewChocolateyDriver() *ChocolateyDriver {
	return &ChocolateyDriver{
		BaseDriver: NewBaseDriver("chocolatey", "choco"),
	}
}

// IsPackageInstalled checks if a package is installed via Chocolatey
func (d *ChocolateyDriver) IsPackageInstalled(packageName string) (bool, error) {
	return d.IsPackageInstalledCached(packageName, d.fetchAllInstalledPackages)
}

// fetchAllInstalledPackages fetches all installed packages from Chocolatey
func (d *ChocolateyDriver) fetchAllInstalledPackages() (map[string]bool, error) {
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

		// Skip header lines and validation output
		if strings.Contains(line, "Chocolatey v") {
			continue
		}
		if strings.Contains(line, "validations performed") {
			continue
		}
		if strings.Contains(line, "Validation Warnings:") {
			continue
		}
		if strings.HasPrefix(line, " - ") {
			continue // Skip validation warning details
		}

		// Check if we've reached the package list section
		// After validation messages, actual packages are listed
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			// Verify this looks like a package line (name + version)
			packageName := parts[0]
			version := parts[1]

			// Skip if it looks like a summary line
			if strings.Contains(line, "packages installed") {
				continue
			}

			// Version should look like a version number (contains dots or numbers)
			if strings.Contains(version, ".") || strings.ContainsAny(version, "0123456789") {
				packages[packageName] = true
				packages[strings.ToLower(packageName)] = true
			}
		}
	}

	return packages, nil
}

// InstallPackage installs a package using Chocolatey
func (d *ChocolateyDriver) InstallPackage(packageName string) error {
	output, err := d.RunCommand("install", packageName, "-y", "--no-progress")
	if err != nil {
		return fmt.Errorf("failed to install package %s via Chocolatey: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// UninstallPackage uninstalls a package using Chocolatey
func (d *ChocolateyDriver) UninstallPackage(packageName string) error {
	output, err := d.RunCommand("uninstall", packageName, "-y")
	if err != nil {
		return fmt.Errorf("failed to uninstall package %s via Chocolatey: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// SearchPackage searches for packages using Chocolatey
func (d *ChocolateyDriver) SearchPackage(packageName string) ([]string, error) {
	output, err := d.RunCommand("search", packageName, "--limit-output")
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

		// Chocolatey search output format: "packagename|version"
		parts := strings.Split(line, "|")
		if len(parts) >= 1 {
			packages = append(packages, parts[0])
		}
	}

	return packages, nil
}

// GetPackageInfo gets information about an installed package
func (d *ChocolateyDriver) GetPackageInfo(packageName string) (map[string]string, error) {
	output, err := d.RunCommand("list", "--local-only", packageName, "--exact")
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

		// Look for the package in the output
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			if strings.EqualFold(parts[0], packageName) {
				info["name"] = parts[0]
				info["version"] = parts[1]
				info["manager"] = "chocolatey"
				break
			}
		}
	}

	if len(info) == 0 {
		return nil, fmt.Errorf("package %s not found", packageName)
	}

	return info, nil
}
