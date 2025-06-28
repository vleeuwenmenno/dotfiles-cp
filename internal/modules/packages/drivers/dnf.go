package drivers

import (
	"fmt"
	"os/exec"
	"runtime"
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
	return d.IsPackageInstalledCached(packageName, d.fetchAllInstalledPackages)
}

// fetchAllInstalledPackages fetches all installed packages from DNF
func (d *DnfDriver) fetchAllInstalledPackages() (map[string]bool, error) {
	output, err := d.RunCommand("list", "installed")
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

		// Skip header lines and error messages
		if strings.Contains(line, "Installed Packages") || strings.Contains(line, "Error:") {
			continue
		}

		// DNF output format: "packagename.arch version repository"
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			// Extract package name (before the dot for architecture)
			fullName := parts[0]
			pkgName := strings.Split(fullName, ".")[0]
			packages[pkgName] = true
			packages[strings.ToLower(pkgName)] = true
		}
	}

	return packages, nil
}

// InstallPackage installs a package using DNF
func (d *DnfDriver) InstallPackage(packageName string) error {
	output, err := d.RunCommandWithSudo("install", "-y", packageName)
	if err != nil {
		return fmt.Errorf("failed to install package %s via DNF: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// UninstallPackage uninstalls a package using DNF
func (d *DnfDriver) UninstallPackage(packageName string) error {
	output, err := d.RunCommandWithSudo("remove", "-y", packageName)
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

// IsAvailable overrides the base implementation to check platform compatibility and sudo
func (d *DnfDriver) IsAvailable() bool {
	// DNF is only available on Linux
	if runtime.GOOS != "linux" {
		return false
	}

	// Check if dnf is available
	if !d.BaseDriver.IsAvailable() {
		return false
	}

	// Check if sudo is available (needed for install/remove operations)
	_, err := exec.LookPath("sudo")
	if err != nil {
		return false
	}

	return true
}

// GetAllInstalledPackages returns a map of all installed packages
func (d *DnfDriver) GetAllInstalledPackages() (map[string]bool, error) {
	return d.fetchAllInstalledPackages()
}

// RunCommandWithSudo executes a DNF command with sudo privileges
func (d *DnfDriver) RunCommandWithSudo(args ...string) (string, error) {
	// Prepend sudo to the command
	sudoArgs := append([]string{d.executable}, args...)
	cmd := exec.Command("sudo", sudoArgs...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}
