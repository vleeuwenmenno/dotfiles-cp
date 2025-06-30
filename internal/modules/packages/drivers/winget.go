package drivers

import (
	"fmt"
	"os/exec"
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
	// First try the cached approach
	installed, err := d.IsPackageInstalledCached(packageName, d.fetchAllInstalledPackages)
	if err == nil && installed {
		return true, nil
	}

	// If not found in cache, try direct lookup using winget show
	// This handles cases where package names/monikers don't appear in winget list
	return d.isPackageInstalledDirect(packageName)
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

		// Skip header lines and empty lines
		if strings.Contains(line, "Name") && strings.Contains(line, "Id") && strings.Contains(line, "Version") {
			continue
		}
		if strings.Contains(line, "---") {
			headerPassed = true
			continue
		}
		if !headerPassed {
			continue
		}

		// Skip lines that don't look like package entries
		if strings.HasPrefix(line, "The following packages") ||
		   strings.HasPrefix(line, "No installed packages") ||
		   strings.Contains(line, "packages found") {
			continue
		}

		// Winget output format: "Name Id Version Available Source"
		// Split by multiple whitespace to handle varying spacing
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			name := fields[0]
			id := fields[1]

			// Store both name and ID for lookups, including case variations
			packages[name] = true
			packages[id] = true
			packages[strings.ToLower(name)] = true
			packages[strings.ToLower(id)] = true

			// Extract base name from ID for better matching (e.g. "nepnep.neofetch-win" -> "neofetch")
			if strings.Contains(id, ".") {
				idParts := strings.Split(id, ".")
				if len(idParts) > 1 {
					baseName := idParts[len(idParts)-1] // Take the last part
					// Remove common suffixes like "-win", "-cli", etc.
					baseName = strings.TrimSuffix(baseName, "-win")
					baseName = strings.TrimSuffix(baseName, "-cli")
					baseName = strings.TrimSuffix(baseName, "-windows")
					packages[baseName] = true
					packages[strings.ToLower(baseName)] = true
				}
			}

			// Also check if name contains common patterns (remove suffixes)
			cleanName := strings.TrimSuffix(strings.ToLower(name), "-win")
			cleanName = strings.TrimSuffix(cleanName, "-cli")
			cleanName = strings.TrimSuffix(cleanName, "-windows")
			if cleanName != strings.ToLower(name) {
				packages[cleanName] = true
			}
		}
	}

	return packages, nil
}

// isPackageInstalledDirect checks if a package is installed by trying to get its info
func (d *WingetDriver) isPackageInstalledDirect(packageName string) (bool, error) {
	// Try to get package info - if it succeeds and shows as installed, package exists
	output, err := d.RunCommand("show", packageName)
	if err != nil {
		// If winget show fails, package is not available/installed
		return false, nil
	}

	// Check if the output indicates the package is installed
	outputLower := strings.ToLower(output)
	if strings.Contains(outputLower, "no package found") ||
	   strings.Contains(outputLower, "no packages found") {
		return false, nil
	}

	// If we can show the package info, try to check if it's in the installed list
	// by running a more specific list command
	listOutput, listErr := d.RunCommand("list", "--exact", packageName)
	if listErr != nil {
		return false, nil
	}

	// If the list command returns the package, it's installed
	listLower := strings.ToLower(listOutput)
	return !strings.Contains(listLower, "no installed packages") &&
	       !strings.Contains(listLower, "no packages found") &&
	       strings.Contains(listLower, strings.ToLower(packageName)), nil
}

// InstallPackage installs a package using Winget
func (d *WingetDriver) InstallPackage(packageName string) error {
	// First check if the package is already installed
	isInstalled, err := d.IsPackageInstalled(packageName)
	if err != nil {
		// If we can't check, proceed with installation attempt
	} else if isInstalled {
		// Package is already installed, this is not an error
		return nil
	}

	output, err := d.RunCommand("install", "--silent", "--accept-package-agreements", "--accept-source-agreements", packageName)
	if err != nil {
		// Check if this is the "already installed" error
		if exitError, ok := err.(*exec.ExitError); ok {
			// Winget exit code 0x8a15002b means package is already installed
			if exitError.ExitCode() == 0x8a15002b || exitError.ExitCode() == -1961967829 {
				return nil // Not an error, package is already installed
			}
		}

		// Check output for already installed messages
		outputLower := strings.ToLower(output)
		if strings.Contains(outputLower, "already installed") ||
		   strings.Contains(outputLower, "existing package already installed") ||
		   strings.Contains(outputLower, "no available upgrade found") ||
		   strings.Contains(outputLower, "trying to upgrade the installed package") ||
		   strings.Contains(outputLower, "no newer package versions are available") {
			return nil // Not an error, package is already installed
		}

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
