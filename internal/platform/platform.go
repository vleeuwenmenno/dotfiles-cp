package platform

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// PlatformInfo contains information about the current platform
type PlatformInfo struct {
	OS                      string            `json:"os"`
	Arch                    string            `json:"arch"`
	Shell                   string            `json:"shell"`
	PackageManagers         []string          `json:"package_managers"`
	AvailablePackageManagers []string         `json:"available_package_managers"`
	HomeDir                 string            `json:"home_dir"`
	ConfigDir               string            `json:"config_dir"`
	IsElevated              bool              `json:"is_elevated"`
	IsRoot                  bool              `json:"is_root"`
	Distro                  string            `json:"distro"`
	DistroVersion           string            `json:"distro_version"`
	DistroCodename          string            `json:"distro_codename"`
	KernelVersion           string            `json:"kernel_version"`
	SystemVersion           string            `json:"system_version"`
	UnameInfo               map[string]string `json:"uname_info"`
}

// GetPlatformInfo returns detailed information about the current platform
func GetPlatformInfo() (*PlatformInfo, error) {
	info := &PlatformInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	var err error

	// Get home directory
	info.HomeDir, err = os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Get config directory
	info.ConfigDir = getConfigDir(info.OS, info.HomeDir)

	// Detect shell
	info.Shell = detectShell()

	// Detect package managers
	info.PackageManagers = detectPackageManagers(info.OS)
	info.AvailablePackageManagers = info.PackageManagers // Alias for template compatibility

	// Check if running with elevated privileges
	info.IsElevated = isElevated(info.OS)
	info.IsRoot = (info.OS != "windows" && os.Geteuid() == 0)

	// Get system information
	info.KernelVersion = getKernelVersion()
	info.UnameInfo = getUnameInfo()

	// Get distro/system version information
	switch info.OS {
	case "linux":
		info.Distro, info.DistroVersion, info.DistroCodename = detectLinuxDistro()
		info.SystemVersion = fmt.Sprintf("%s %s", info.Distro, info.DistroVersion)
	case "windows":
		info.SystemVersion = getWindowsVersion()
		info.Distro = "Windows"
	case "darwin":
		info.SystemVersion = getMacOSVersion()
		info.Distro = "macOS"
	default:
		info.Distro = info.OS
		info.SystemVersion = info.OS
	}

	return info, nil
}

// detectShell attempts to detect the current shell
func detectShell() string {
	// Try SHELL environment variable first (Unix-like systems)
	if shell := os.Getenv("SHELL"); shell != "" {
		return getShellName(shell)
	}

	// Try PSModulePath for PowerShell (Windows)
	if os.Getenv("PSModulePath") != "" {
		return "powershell"
	}

	// Fallback based on OS
	switch runtime.GOOS {
	case "windows":
		return "cmd"
	case "darwin":
		return "zsh" // macOS default since Catalina
	default:
		return "bash" // Most Linux distributions
	}
}

// getShellName extracts the shell name from a full path
func getShellName(shellPath string) string {
	parts := strings.Split(shellPath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return shellPath
}

// detectPackageManagers detects available package managers on the system
func detectPackageManagers(osName string) []string {
	var managers []string

	switch osName {
	case "windows":
		if commandExists("choco") {
			managers = append(managers, "chocolatey")
		}
		if commandExists("winget") {
			managers = append(managers, "winget")
		}
		if commandExists("scoop") {
			managers = append(managers, "scoop")
		}
	case "darwin":
		if commandExists("brew") {
			managers = append(managers, "homebrew")
		}
		if commandExists("port") {
			managers = append(managers, "macports")
		}
	case "linux":
		if commandExists("apt") || commandExists("apt-get") {
			managers = append(managers, "apt")
		}
		if commandExists("yum") {
			managers = append(managers, "yum")
		}
		if commandExists("dnf") {
			managers = append(managers, "dnf")
		}
		if commandExists("pacman") {
			managers = append(managers, "pacman")
		}
		if commandExists("zypper") {
			managers = append(managers, "zypper")
		}
		if commandExists("emerge") {
			managers = append(managers, "portage")
		}
		if commandExists("xbps-install") {
			managers = append(managers, "xbps")
		}
		if commandExists("apk") {
			managers = append(managers, "apk")
		}
	}

	return managers
}

// commandExists checks if a command is available in the system PATH
func commandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// getConfigDir returns the appropriate configuration directory for the OS
func getConfigDir(osName, homeDir string) string {
	switch osName {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return appData
		}
		return homeDir + "\\AppData\\Roaming"
	case "darwin":
		return homeDir + "/Library/Application Support"
	default:
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			return xdgConfig
		}
		return homeDir + "/.config"
	}
}

// isElevated checks if the current process is running with elevated privileges
func isElevated(osName string) bool {
	switch osName {
	case "windows":
		// On Windows, check if we can write to a system directory
		tempFile := os.Getenv("WINDIR") + "\\temp\\dotfiles-test"
		file, err := os.Create(tempFile)
		if err != nil {
			return false
		}
		file.Close()
		os.Remove(tempFile)
		return true
	default:
		// On Unix-like systems, check if we're running as root
		return os.Geteuid() == 0
	}
}

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMacOS returns true if running on macOS
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// IsLinux returns true if running on Linux
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// GetShellConfigPath returns the path to the shell configuration file
func GetShellConfigPath(shell, homeDir string) string {
	switch shell {
	case "bash":
		return homeDir + "/.bashrc"
	case "zsh":
		return homeDir + "/.zshrc"
	case "fish":
		return homeDir + "/.config/fish/config.fish"
	case "powershell":
		if IsWindows() {
			return homeDir + "\\Documents\\PowerShell\\Microsoft.PowerShell_profile.ps1"
		}
		return homeDir + "/.config/powershell/Microsoft.PowerShell_profile.ps1"
	case "cmd":
		return homeDir + "\\autoexec.bat"
	default:
		return ""
	}
}

// detectLinuxDistro detects the Linux distribution and version
func detectLinuxDistro() (distro, version, codename string) {
	// Try /etc/os-release first (standard)
	if distro, version, codename = parseOSRelease("/etc/os-release"); distro != "" {
		return
	}

	// Try /usr/lib/os-release as fallback
	if distro, version, codename = parseOSRelease("/usr/lib/os-release"); distro != "" {
		return
	}

	// Try legacy files
	if distro = readFirstLine("/etc/debian_version"); distro != "" {
		return "Debian", distro, ""
	}
	if distro = readFirstLine("/etc/redhat-release"); distro != "" {
		return parseRedhatRelease(distro)
	}
	if distro = readFirstLine("/etc/arch-release"); distro != "" {
		return "Arch Linux", "rolling", ""
	}

	return "Unknown", "", ""
}

// parseOSRelease parses /etc/os-release or /usr/lib/os-release
func parseOSRelease(filepath string) (distro, version, codename string) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", "", ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "NAME=") {
			distro = strings.Trim(strings.TrimPrefix(line, "NAME="), "\"")
		} else if strings.HasPrefix(line, "VERSION=") {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION="), "\"")
		} else if strings.HasPrefix(line, "VERSION_CODENAME=") {
			codename = strings.Trim(strings.TrimPrefix(line, "VERSION_CODENAME="), "\"")
		} else if strings.HasPrefix(line, "VERSION_ID=") && version == "" {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
	}
	return
}

// parseRedhatRelease parses Red Hat style release files
func parseRedhatRelease(content string) (distro, version, codename string) {
	parts := strings.Fields(content)
	if len(parts) >= 3 {
		distro = parts[0]
		for i, part := range parts {
			if strings.Contains(part, ".") && len(part) > 1 {
				version = part
				break
			}
			if part == "release" && i+1 < len(parts) {
				version = parts[i+1]
				break
			}
		}
	}
	return distro, version, ""
}

// readFirstLine reads the first line of a file
func readFirstLine(filepath string) string {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return ""
}

// getWindowsVersion gets Windows version information
func getWindowsVersion() string {
	cmd := exec.Command("cmd", "/c", "ver")
	output, err := cmd.Output()
	if err != nil {
		return "Windows (unknown version)"
	}

	version := strings.TrimSpace(string(output))
	// Clean up the output (remove "Microsoft Windows [Version " and "]")
	if strings.Contains(version, "[Version ") {
		start := strings.Index(version, "[Version ") + 9
		end := strings.Index(version[start:], "]")
		if end > 0 {
			version = version[start : start+end]
		}
	}

	return version
}

// getMacOSVersion gets macOS version information
func getMacOSVersion() string {
	cmd := exec.Command("sw_vers", "-productVersion")
	output, err := cmd.Output()
	if err != nil {
		return "macOS (unknown version)"
	}
	return strings.TrimSpace(string(output))
}

// getKernelVersion gets the kernel version
func getKernelVersion() string {
	if runtime.GOOS == "windows" {
		return getWindowsVersion()
	}

	cmd := exec.Command("uname", "-r")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getUnameInfo gets detailed uname information
func getUnameInfo() map[string]string {
	info := make(map[string]string)

	if runtime.GOOS == "windows" {
		// Windows equivalent information
		info["system"] = "Windows"
		info["version"] = getWindowsVersion()
		info["machine"] = runtime.GOARCH
		return info
	}

	// Unix-like systems
	fields := []struct {
		flag string
		key  string
	}{
		{"-s", "system"},     // System name
		{"-n", "nodename"},   // Network node hostname
		{"-r", "release"},    // System release
		{"-v", "version"},    // System version
		{"-m", "machine"},    // Machine hardware name
		{"-p", "processor"},  // Processor type
		{"-i", "platform"},   // Hardware platform
		{"-o", "operating"},  // Operating system
	}

	for _, field := range fields {
		cmd := exec.Command("uname", field.flag)
		output, err := cmd.Output()
		if err == nil {
			info[field.key] = strings.TrimSpace(string(output))
		}
	}

	return info
}

// GetPackageManagerInstallCommand returns the install command for a package manager
func GetPackageManagerInstallCommand(manager string) string {
	switch manager {
	case "chocolatey":
		return "choco install"
	case "winget":
		return "winget install"
	case "scoop":
		return "scoop install"
	case "homebrew":
		return "brew install"
	case "macports":
		return "port install"
	case "apt":
		return "apt install"
	case "yum":
		return "yum install"
	case "dnf":
		return "dnf install"
	case "pacman":
		return "pacman -S"
	case "zypper":
		return "zypper install"
	case "portage":
		return "emerge"
	case "xbps":
		return "xbps-install"
	case "apk":
		return "apk add"
	default:
		return ""
	}
}
