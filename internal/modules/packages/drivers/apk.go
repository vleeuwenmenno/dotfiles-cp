package drivers

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ApkDriver implements PackageDriver for APK package manager (Alpine Linux)
type ApkDriver struct {
	*BaseDriver
}

// NewApkDriver creates a new APK driver
func NewApkDriver() *ApkDriver {
	return &ApkDriver{
		BaseDriver: NewBaseDriver("apk", "apk"),
	}
}

// IsPackageInstalled checks if a package is installed via APK
func (d *ApkDriver) IsPackageInstalled(packageName string) (bool, error) {
	return d.IsPackageInstalledCached(packageName, d.fetchAllInstalledPackages)
}

// fetchAllInstalledPackages fetches all installed packages from APK
func (d *ApkDriver) fetchAllInstalledPackages() (map[string]bool, error) {
	output, err := d.RunCommand("list", "--installed")
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

		// Skip warning lines
		if strings.HasPrefix(line, "WARNING:") || strings.HasPrefix(line, "fetch") {
			continue
		}

		// APK output format: "package-name-version arch {repo} (description)"
		// We need to extract just the package name part
		if strings.Contains(line, " ") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				packageInfo := parts[0]
				// Extract package name by removing version and architecture
				// Format is typically: package-name-version-r0
				packageName := d.extractPackageName(packageInfo)
				if packageName != "" {
					packages[packageName] = true
					packages[strings.ToLower(packageName)] = true
				}
			}
		}
	}

	return packages, nil
}

// extractPackageName extracts the package name from APK's package info string
func (d *ApkDriver) extractPackageName(packageInfo string) string {
	// APK packages follow format: name-version-release
	// We need to find where the version starts (first digit after a dash)
	parts := strings.Split(packageInfo, "-")
	if len(parts) < 2 {
		return packageInfo
	}

	// Find the first part that starts with a digit (this is likely the version)
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 && (parts[i][0] >= '0' && parts[i][0] <= '9') {
			// Everything before this part is the package name
			return strings.Join(parts[:i], "-")
		}
	}

	// If no version found, return the whole string minus the last part
	if len(parts) > 1 {
		return strings.Join(parts[:len(parts)-1], "-")
	}

	return packageInfo
}

// InstallPackage installs a package using APK
func (d *ApkDriver) InstallPackage(packageName string) error {
	// Update package index first
	_, updateErr := d.RunCommandWithSudo("update")
	if updateErr != nil {
		// Log warning but continue - update might fail due to permissions
		// but installation might still work
	}

	output, err := d.RunCommandWithSudo("add", packageName)
	if err != nil {
		return fmt.Errorf("failed to install package %s via APK: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// UninstallPackage uninstalls a package using APK
func (d *ApkDriver) UninstallPackage(packageName string) error {
	output, err := d.RunCommandWithSudo("del", packageName)
	if err != nil {
		return fmt.Errorf("failed to uninstall package %s via APK: %w\nOutput: %s", packageName, err, output)
	}
	return nil
}

// SearchPackage searches for packages using APK
func (d *ApkDriver) SearchPackage(packageName string) ([]string, error) {
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
		if strings.HasPrefix(line, "WARNING:") || strings.HasPrefix(line, "fetch") {
			continue
		}

		// APK search output format: "package-name-version description"
		if strings.Contains(line, " ") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				packageInfo := parts[0]
				packageName := d.extractPackageName(packageInfo)
				if packageName != "" {
					packages = append(packages, packageName)
				}
			}
		}
	}

	return packages, nil
}

// GetPackageInfo gets information about an installed package
func (d *ApkDriver) GetPackageInfo(packageName string) (map[string]string, error) {
	output, err := d.RunCommand("info", packageName)
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

		// Parse APK info output
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch strings.ToLower(key) {
				case "package":
					info["name"] = value
				case "version":
					info["version"] = value
				case "architecture":
					info["architecture"] = value
				case "description":
					info["description"] = value
				}
			}
		}
	}

	if len(info) == 0 {
		return nil, fmt.Errorf("package %s not found", packageName)
	}

	info["manager"] = "apk"
	return info, nil
}

// GetAllInstalledPackages returns a map of all installed packages
func (d *ApkDriver) GetAllInstalledPackages() (map[string]bool, error) {
	return d.fetchAllInstalledPackages()
}

// RunCommandWithSudo executes an APK command with sudo privileges
func (d *ApkDriver) RunCommandWithSudo(args ...string) (string, error) {
	// Check if we're already running as root
	if d.isRunningAsRoot() {
		return d.RunCommand(args...)
	}

	// Prepend sudo to the command
	sudoArgs := append([]string{d.executable}, args...)
	cmd := exec.Command("sudo", sudoArgs...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// isRunningAsRoot checks if the current process is running as root
func (d *ApkDriver) isRunningAsRoot() bool {
	output, err := exec.Command("id", "-u").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "0"
}

// IsAvailable overrides the base implementation to check platform compatibility and sudo
func (d *ApkDriver) IsAvailable() bool {
	// APK is only available on Linux (specifically Alpine Linux)
	if runtime.GOOS != "linux" {
		return false
	}

	// Check if apk is available
	if !d.BaseDriver.IsAvailable() {
		return false
	}

	// Check if we can run apk commands (either as root or with sudo)
	if d.isRunningAsRoot() {
		return true
	}

	// Check if sudo is available (needed for install/remove operations)
	_, err := exec.LookPath("sudo")
	return err == nil
}
