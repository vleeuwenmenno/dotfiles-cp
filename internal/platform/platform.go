package platform

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// PlatformInfo contains information about the current platform
type PlatformInfo struct {
	OS              string   `json:"os"`
	Arch            string   `json:"arch"`
	Shell           string   `json:"shell"`
	PackageManagers []string `json:"package_managers"`
	HomeDir         string   `json:"home_dir"`
	ConfigDir       string   `json:"config_dir"`
	IsElevated      bool     `json:"is_elevated"`
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

	// Check if running with elevated privileges
	info.IsElevated = isElevated(info.OS)

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
