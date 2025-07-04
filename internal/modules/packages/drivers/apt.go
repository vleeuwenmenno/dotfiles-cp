package drivers

import (
	"fmt"
	"os/exec"
	"runtime"
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
	_, updateErr := d.RunCommandWithSudo("update")
	if updateErr != nil {
		// Log warning but continue - update might fail due to permissions
		// but installation might still work
	}

	output, err := d.RunCommandWithSudo("install", "-y", packageName)
	if err != nil {
		return fmt.Errorf("failed to install package %s via APT: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// UninstallPackage uninstalls a package using APT
func (d *AptDriver) UninstallPackage(packageName string) error {
	output, err := d.RunCommandWithSudo("remove", "-y", packageName)
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

// GetAllInstalledPackages returns a map of all installed packages
func (d *AptDriver) GetAllInstalledPackages() (map[string]bool, error) {
	return d.fetchAllInstalledPackages()
}

// RunCommandWithSudo executes an APT command with sudo privileges
func (d *AptDriver) RunCommandWithSudo(args ...string) (string, error) {
	// Prepend sudo to the command
	sudoArgs := append([]string{d.executable}, args...)
	cmd := exec.Command("sudo", sudoArgs...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// EnsureRepository ensures a PPA or repository is available
func (d *AptDriver) EnsureRepository(repoName string) error {
	// Check if add-apt-repository is available
	_, err := exec.LookPath("add-apt-repository")
	if err != nil {
		return fmt.Errorf("add-apt-repository command not found. Please install software-properties-common package")
	}

	// Check if repository is already available
	isAvailable, err := d.IsRepositoryAvailable(repoName)
	if err != nil {
		return fmt.Errorf("failed to check repository availability: %w", err)
	}

	if isAvailable {
		return nil // Repository already exists
	}

	// Add the repository
	var output string
	if strings.HasPrefix(repoName, "ppa:") {
		// Handle PPA
		output, err = d.RunCommandWithSudo("add-apt-repository", "-y", repoName)
	} else {
		// Handle regular repository (assume it's a complete sources.list line)
		output, err = d.RunCommandWithSudo("add-apt-repository", "-y", repoName)
	}

	if err != nil {
		return fmt.Errorf("failed to add repository %s: %w\nOutput: %s", repoName, err, output)
	}

	// Update package list after adding repository
	_, updateErr := d.RunCommandWithSudo("update")
	if updateErr != nil {
		// Log warning but don't fail - the repository was added successfully
		return fmt.Errorf("repository %s added but failed to update package list: %w", repoName, updateErr)
	}

	return nil
}

// IsRepositoryAvailable checks if a PPA or repository is already available
func (d *AptDriver) IsRepositoryAvailable(repoName string) (bool, error) {
	// Check if add-apt-repository is available for repository management
	_, err := exec.LookPath("add-apt-repository")
	if err != nil {
		return false, fmt.Errorf("add-apt-repository command not found. Please install software-properties-common package")
	}

	if strings.HasPrefix(repoName, "ppa:") {
		return d.isPPAAvailable(repoName)
	}
	return d.isRepositoryInSources(repoName)
}

// isPPAAvailable checks if a PPA is already added
func (d *AptDriver) isPPAAvailable(ppaName string) (bool, error) {
	// Extract PPA identifier (remove "ppa:" prefix)
	ppaId := strings.TrimPrefix(ppaName, "ppa:")

	// Check in /etc/apt/sources.list.d/ for files containing this PPA
	output, err := exec.Command("find", "/etc/apt/sources.list.d/", "-name", "*.list", "-exec", "grep", "-l", ppaId, "{}", ";").CombinedOutput()
	if err != nil {
		// If grep finds nothing, it returns exit code 1, which is normal
		if strings.Contains(err.Error(), "exit status 1") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check PPA availability: %w", err)
	}

	// If we found files containing the PPA, it's available
	return strings.TrimSpace(string(output)) != "", nil
}

// isRepositoryInSources checks if a repository line exists in sources
func (d *AptDriver) isRepositoryInSources(repoLine string) (bool, error) {
	// Check main sources.list
	output, err := exec.Command("grep", "-h", repoLine, "/etc/apt/sources.list").CombinedOutput()
	if err == nil && strings.TrimSpace(string(output)) != "" {
		return true, nil
	}

	// Check sources.list.d directory
	output, err = exec.Command("find", "/etc/apt/sources.list.d/", "-name", "*.list", "-exec", "grep", "-l", repoLine, "{}", ";").CombinedOutput()
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check repository in sources: %w", err)
	}

	return strings.TrimSpace(string(output)) != "", nil
}

// IsAvailable overrides the base implementation to check platform compatibility and sudo
func (d *AptDriver) IsAvailable() bool {
	// APT is only available on Linux
	if runtime.GOOS != "linux" {
		return false
	}

	// Check if apt is available
	if !d.BaseDriver.IsAvailable() {
		return false
	}

	// Check if sudo is available (needed for install/remove operations)
	_, err := exec.LookPath("sudo")
	if err != nil {
		return false
	}

	// Check if add-apt-repository is available (needed for repository management)
	_, err = exec.LookPath("add-apt-repository")
	if err != nil {
		return false
	}

	return true
}
