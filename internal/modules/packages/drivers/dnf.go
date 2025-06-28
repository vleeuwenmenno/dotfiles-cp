package drivers

import (
	"fmt"
	"strings"
)

// DnfDriver implements PackageDriver for DNF package manager (Fedora)
type DnfDriver struct {
	*BaseDriver
}

// NewDnfDriver creates a new DNF driver
func NewDnfDriver() *DnfDriver {
	return &DnfDriver{
		BaseDriver: NewBaseDriver("dnf", "dnf"),
	}
}

// IsPackageInstalled checks if a package is installed via DNF
func (d *DnfDriver) IsPackageInstalled(packageName string) (bool, error) {
	// Use 'dnf list installed' to check if package is installed
	output, err := d.RunCommand("list", "installed", packageName)
	if err != nil {
		// If command fails, package is likely not installed
		return false, nil
	}

	// Parse the output to see if we found the package
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header lines and error messages
		if strings.Contains(line, "Installed Packages") || strings.Contains(line, "Error:") {
			continue
		}

		// DNF output format: "packagename.arch version repository"
		if strings.Contains(line, packageName) {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				// Extract package name (before the dot for architecture)
				installedPkg := strings.Split(parts[0], ".")[0]
				if strings.EqualFold(installedPkg, packageName) {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// InstallPackage installs a package using DNF
func (d *DnfDriver) InstallPackage(packageName string) error {
	output, err := d.RunCommand("install", "-y", packageName)
	if err != nil {
		return fmt.Errorf("failed to install package %s via DNF: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// UninstallPackage uninstalls a package using DNF
func (d *DnfDriver) UninstallPackage(packageName string) error {
	output, err := d.RunCommand("remove", "-y", packageName)
	if err != nil {
		return fmt.Errorf("failed to uninstall package %s via DNF: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// SearchPackage searches for packages using DNF
func (d *DnfDriver) SearchPackage(packageName string) ([]string, error) {
	output, err := d.RunCommand("search", packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to search for package %s: %w", packageName, err)
	}

	var packages []string
	lines := strings.Split(output, "\n")
	inResults := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for the results section
		if strings.Contains(line, "========================") {
			inResults = true
			continue
		}

		// Skip informational lines
		if !inResults || strings.Contains(line, "Search Results") || strings.Contains(line, "Name") {
			continue
		}

		// DNF search output format: "packagename.arch : description"
		if strings.Contains(line, " : ") {
			parts := strings.Split(line, " : ")
			if len(parts) >= 1 {
				// Extract package name (before the dot for architecture)
				fullName := strings.TrimSpace(parts[0])
				pkgName := strings.Split(fullName, ".")[0]
				packages = append(packages, pkgName)
			}
		}
	}

	return packages, nil
}

// GetPackageInfo gets information about an installed package
func (d *DnfDriver) GetPackageInfo(packageName string) (map[string]string, error) {
	output, err := d.RunCommand("list", "installed", packageName)
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
		if strings.Contains(line, "Installed Packages") {
			continue
		}

		// Check if this line contains our package
		if strings.Contains(line, packageName) {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				// Extract package name (before the dot for architecture)
				fullName := parts[0]
				pkgName := strings.Split(fullName, ".")[0]
				if strings.EqualFold(pkgName, packageName) {
					info["name"] = pkgName
					info["version"] = parts[1]
					info["repository"] = parts[2]
					info["manager"] = "dnf"

					// Extract architecture if present
					if strings.Contains(fullName, ".") {
						archParts := strings.Split(fullName, ".")
						if len(archParts) >= 2 {
							info["architecture"] = archParts[len(archParts)-1]
						}
					}
					break
				}
			}
		}
	}

	if len(info) == 0 {
		return nil, fmt.Errorf("package %s not found", packageName)
	}

	return info, nil
}

// IsAvailable overrides the base implementation to check for sudo
func (d *DnfDriver) IsAvailable() bool {
	// Check if dnf is available
	if !d.BaseDriver.IsAvailable() {
		return false
	}

	// For DNF operations, we typically need sudo for install/remove
	// But we can still check package status without it
	return true
}
