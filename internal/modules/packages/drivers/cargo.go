package drivers

import (
	"fmt"
	"strings"
)

// CargoDriver implements PackageDriver for Cargo package manager (Rust)
type CargoDriver struct {
	*BaseDriver
}

// NewCargoDriver creates a new Cargo driver
func NewCargoDriver() *CargoDriver {
	return &CargoDriver{
		BaseDriver: NewBaseDriver("cargo", "cargo"),
	}
}

// IsPackageInstalled checks if a package is installed via Cargo
func (d *CargoDriver) IsPackageInstalled(packageName string) (bool, error) {
	return d.IsPackageInstalledCached(packageName, d.fetchAllInstalledPackages)
}

// fetchAllInstalledPackages fetches all installed packages from Cargo
func (d *CargoDriver) fetchAllInstalledPackages() (map[string]bool, error) {
	output, err := d.RunCommand("install", "--list")
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

		// Cargo output format: "packagename v1.2.3:"
		// followed by indented binary paths
		if !strings.HasPrefix(line, " ") && strings.Contains(line, " v") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				packageName := parts[0]
				// Remove trailing colon if present
				packageName = strings.TrimSuffix(packageName, ":")
				packages[packageName] = true
				packages[strings.ToLower(packageName)] = true
			}
		}
	}

	return packages, nil
}

// InstallPackage installs a package using Cargo
func (d *CargoDriver) InstallPackage(packageName string) error {
	output, err := d.RunCommand("install", packageName)
	if err != nil {
		return fmt.Errorf("failed to install package %s via Cargo: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// UninstallPackage uninstalls a package using Cargo
func (d *CargoDriver) UninstallPackage(packageName string) error {
	output, err := d.RunCommand("uninstall", packageName)
	if err != nil {
		return fmt.Errorf("failed to uninstall package %s via Cargo: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// SearchPackage searches for packages using Cargo
func (d *CargoDriver) SearchPackage(packageName string) ([]string, error) {
	output, err := d.RunCommand("search", packageName, "--limit", "20")
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

		// Cargo search output format: "packagename = "version" # description"
		parts := strings.Split(line, " = ")
		if len(parts) >= 2 {
			packageName := strings.TrimSpace(parts[0])
			packages = append(packages, packageName)
		}
	}

	return packages, nil
}

// GetPackageInfo gets information about an installed package
func (d *CargoDriver) GetPackageInfo(packageName string) (map[string]string, error) {
	output, err := d.RunCommand("install", "--list")
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
		if !strings.HasPrefix(line, " ") && strings.Contains(line, " v") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				pkgName := strings.TrimSuffix(parts[0], ":")
				if strings.EqualFold(pkgName, packageName) {
					info["name"] = pkgName
					version := parts[1]
					// Remove 'v' prefix if present
					if strings.HasPrefix(version, "v") {
						version = version[1:]
					}
					// Remove trailing colon if present
					version = strings.TrimSuffix(version, ":")
					info["version"] = version
					info["manager"] = "cargo"
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

// GetAllInstalledPackages returns a map of all installed packages
func (d *CargoDriver) GetAllInstalledPackages() (map[string]bool, error) {
	return d.fetchAllInstalledPackages()
}

// IsAvailable checks if Cargo is available on the system
func (d *CargoDriver) IsAvailable() bool {
	// Check if cargo is available
	return d.BaseDriver.IsAvailable()
}
