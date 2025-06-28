package drivers

import (
	"fmt"
	"strings"
)

// AptDriver implements PackageDriver for APT package manager (Debian/Ubuntu)
type AptDriver struct {
	*BaseDriver
}

// NewAptDriver creates a new APT driver
func NewAptDriver() *AptDriver {
	return &AptDriver{
		BaseDriver: NewBaseDriver("apt", "apt"),
	}
}

// IsPackageInstalled checks if a package is installed via APT
func (d *AptDriver) IsPackageInstalled(packageName string) (bool, error) {
	return d.IsPackageInstalledCached(packageName, d.fetchAllInstalledPackages)
}

// fetchAllInstalledPackages fetches all installed packages from APT
func (d *AptDriver) fetchAllInstalledPackages() (map[string]bool, error) {
	output, err := d.RunCommand("list", "--installed")
	if err != nil {
		// If apt list fails, try dpkg as fallback
		return d.fetchAllInstalledPackagesDpkg()
	}

	packages := make(map[string]bool)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip warning and informational lines
		if strings.HasPrefix(line, "WARNING:") || strings.Contains(line, "Listing...") {
			continue
		}

		// APT output format: "packagename/suite version architecture [status]"
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			// Extract package name (before the slash if present)
			fullName := parts[0]
			pkgName := strings.Split(fullName, "/")[0]
			packages[pkgName] = true
			packages[strings.ToLower(pkgName)] = true
		}
	}

	return packages, nil
}

// fetchAllInstalledPackagesDpkg uses dpkg as fallback to get all installed packages
func (d *AptDriver) fetchAllInstalledPackagesDpkg() (map[string]bool, error) {
	baseDriver := &BaseDriver{
		name:       "dpkg-query",
		executable: "dpkg-query",
	}
	output, err := baseDriver.RunCommand("-W", "-f=${Package} ${Status}\n")
	if err != nil {
		return nil, fmt.Errorf("failed to list packages with dpkg: %w", err)
	}

	packages := make(map[string]bool)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// dpkg output format: "packagename install ok installed"
		parts := strings.Fields(line)
		if len(parts) >= 4 {
			packageName := parts[0]
			status := strings.Join(parts[1:], " ")
			if strings.Contains(status, "install ok installed") {
				packages[packageName] = true
				packages[strings.ToLower(packageName)] = true
			}
		}
	}

	return packages, nil
}

// InstallPackage installs a package using APT
func (d *AptDriver) InstallPackage(packageName string) error {
	// Update package list first (only if not updated recently)
	_, updateErr := d.RunCommand("update")
	if updateErr != nil {
		// Log warning but continue - update might fail due to permissions
		// but installation might still work
	}

	output, err := d.RunCommand("install", "-y", packageName)
	if err != nil {
		return fmt.Errorf("failed to install package %s via APT: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// UninstallPackage uninstalls a package using APT
func (d *AptDriver) UninstallPackage(packageName string) error {
	output, err := d.RunCommand("remove", "-y", packageName)
	if err != nil {
		return fmt.Errorf("failed to uninstall package %s via APT: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// SearchPackage searches for packages using APT
func (d *AptDriver) SearchPackage(packageName string) ([]string, error) {
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

		// Skip warning and informational lines
		if strings.HasPrefix(line, "WARNING:") || strings.Contains(line, "Sorting...") {
			continue
		}

		// APT search output format: "packagename/suite version architecture"
		// followed by description lines that start with spaces
		if !strings.HasPrefix(line, " ") && strings.Contains(line, "/") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				// Extract package name (before the slash)
				pkgName := strings.Split(parts[0], "/")[0]
				packages = append(packages, pkgName)
			}
		}
	}

	return packages, nil
}

// GetPackageInfo gets information about an installed package
func (d *AptDriver) GetPackageInfo(packageName string) (map[string]string, error) {
	output, err := d.RunCommand("list", "--installed", packageName)
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

		// Skip warning and informational lines
		if strings.HasPrefix(line, "WARNING:") || strings.Contains(line, "Listing...") {
			continue
		}

		// Check if this line contains our package
		if strings.Contains(line, packageName) {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				// Extract package name (before the slash)
				fullName := parts[0]
				pkgName := strings.Split(fullName, "/")[0]
				if strings.EqualFold(pkgName, packageName) {
					info["name"] = pkgName
					info["version"] = parts[1]
					info["architecture"] = parts[2]
					info["manager"] = "apt"

					// Extract suite/repository info
					if strings.Contains(fullName, "/") {
						suite := strings.Split(fullName, "/")[1]
						info["suite"] = suite
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

// IsAvailable overrides the base implementation to also check for sudo
func (d *AptDriver) IsAvailable() bool {
	// Check if apt is available
	if !d.BaseDriver.IsAvailable() {
		return false
	}

	// For APT operations, we typically need sudo for install/remove
	// But we can still check package status without it
	return true
}
