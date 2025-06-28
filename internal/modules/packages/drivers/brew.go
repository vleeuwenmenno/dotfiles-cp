package drivers

import (
	"fmt"
	"strings"
)

// BrewDriver implements PackageDriver for Homebrew package manager (macOS)
type BrewDriver struct {
	*BaseDriver
}

// NewBrewDriver creates a new Homebrew driver
func NewBrewDriver() *BrewDriver {
	return &BrewDriver{
		BaseDriver: NewBaseDriver("homebrew", "brew"),
	}
}

// IsPackageInstalled checks if a package is installed via Homebrew
func (d *BrewDriver) IsPackageInstalled(packageName string) (bool, error) {
	return d.IsPackageInstalledCached(packageName, d.fetchAllInstalledPackages)
}

// fetchAllInstalledPackages fetches all installed packages from Homebrew
func (d *BrewDriver) fetchAllInstalledPackages() (map[string]bool, error) {
	packages := make(map[string]bool)

	// Get formulae
	output, err := d.RunCommand("list", "--formula")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed formulae: %w", err)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		packages[line] = true
		packages[strings.ToLower(line)] = true
	}

	// Get casks
	output, err = d.RunCommand("list", "--cask")
	if err != nil {
		// Cask might not be available or no casks installed, but continue
	} else {
		lines = strings.Split(output, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			packages[line] = true
			packages[strings.ToLower(line)] = true
		}
	}

	return packages, nil
}

// InstallPackage installs a package using Homebrew
func (d *BrewDriver) InstallPackage(packageName string) error {
	// Try installing as formula first
	output, err := d.RunCommand("install", packageName)
	if err != nil {
		return fmt.Errorf("failed to install package %s via Homebrew: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// UninstallPackage uninstalls a package using Homebrew
func (d *BrewDriver) UninstallPackage(packageName string) error {
	// Check if it's a formula or cask first
	isFormula, _ := d.isFormulaInstalled(packageName)
	isCask, _ := d.isCaskInstalled(packageName)

	if isFormula {
		output, err := d.RunCommand("uninstall", packageName)
		if err != nil {
			return fmt.Errorf("failed to uninstall formula %s via Homebrew: %w\nOutput: %s", packageName, err, output)
		}
	} else if isCask {
		output, err := d.RunCommand("uninstall", "--cask", packageName)
		if err != nil {
			return fmt.Errorf("failed to uninstall cask %s via Homebrew: %w\nOutput: %s", packageName, err, output)
		}
	} else {
		return fmt.Errorf("package %s not found as formula or cask", packageName)
	}

	return nil
}

// isFormulaInstalled checks if a package is installed as a formula
func (d *BrewDriver) isFormulaInstalled(packageName string) (bool, error) {
	output, err := d.RunCommand("list", "--formula")
	if err != nil {
		return false, err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == packageName {
			return true, nil
		}
	}
	return false, nil
}

// isCaskInstalled checks if a package is installed as a cask
func (d *BrewDriver) isCaskInstalled(packageName string) (bool, error) {
	output, err := d.RunCommand("list", "--cask")
	if err != nil {
		return false, err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == packageName {
			return true, nil
		}
	}
	return false, nil
}

// SearchPackage searches for packages using Homebrew
func (d *BrewDriver) SearchPackage(packageName string) ([]string, error) {
	output, err := d.RunCommand("search", packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to search for package %s: %w", packageName, err)
	}

	var packages []string
	lines := strings.Split(output, "\n")
	inFormulae := false
	inCasks := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Section headers
		if strings.Contains(line, "==> Formulae") {
			inFormulae = true
			inCasks = false
			continue
		}
		if strings.Contains(line, "==> Casks") {
			inFormulae = false
			inCasks = true
			continue
		}

		// Skip other section headers
		if strings.HasPrefix(line, "==>") {
			inFormulae = false
			inCasks = false
			continue
		}

		// Add packages from current section
		if inFormulae || inCasks {
			// Homebrew search output can have multiple packages per line
			fields := strings.Fields(line)
			for _, field := range fields {
				if field != "" {
					packages = append(packages, field)
				}
			}
		}
	}

	return packages, nil
}

// GetPackageInfo gets information about an installed package
func (d *BrewDriver) GetPackageInfo(packageName string) (map[string]string, error) {
	info := make(map[string]string)

	// Check if it's a formula first
	output, err := d.RunCommand("list", "--formula", "--versions", packageName)
	if err == nil && strings.TrimSpace(output) != "" {
		// Parse formula info
		line := strings.TrimSpace(output)
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			info["name"] = parts[0]
			info["version"] = parts[1]
			info["type"] = "formula"
			info["manager"] = "homebrew"
			return info, nil
		}
	}

	// Check if it's a cask
	output, err = d.RunCommand("list", "--cask", "--versions", packageName)
	if err == nil && strings.TrimSpace(output) != "" {
		// Parse cask info
		line := strings.TrimSpace(output)
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			info["name"] = parts[0]
			info["version"] = parts[1]
			info["type"] = "cask"
			info["manager"] = "homebrew"
			return info, nil
		}
	}

	return nil, fmt.Errorf("package %s not found", packageName)
}
