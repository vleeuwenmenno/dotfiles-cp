//go:build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	binaryName = "dotfiles"
	cmdDir     = "cmd/dotfiles"
	buildDir   = "bin"
)

var (
	version  = flag.String("version", "dev", "Version to build")
	clean    = flag.Bool("clean", false, "Clean build directory before building")
	all      = flag.Bool("all", false, "Build for all platforms")
	verbose  = flag.Bool("v", false, "Verbose output")
	run      = flag.Bool("run", false, "Build and run the binary")
	test     = flag.Bool("test", false, "Run tests")
	coverage = flag.Bool("coverage", false, "Run tests with coverage")
	install  = flag.Bool("install", false, "Install binary to GOPATH/bin")
	format   = flag.Bool("fmt", false, "Format code")
	deps     = flag.Bool("deps", false, "Download and tidy dependencies")
	help     = flag.Bool("help", false, "Show help")
)

type BuildTarget struct {
	OS   string
	Arch string
}

var targets = []BuildTarget{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
	{"windows", "amd64"},
	{"windows", "arm64"},
}

func main() {
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Check if Go is available
	if !commandExists("go") {
		fatal("Go is not installed or not in PATH")
	}

	// Handle different operations
	switch {
	case *deps:
		downloadDeps()
	case *format:
		formatCode()
	case *test:
		runTests()
	case *coverage:
		runTestsWithCoverage()
	case *clean:
		cleanBuild()
		if !*all {
			return
		}
		fallthrough
	case *all:
		buildAll()
	case *install:
		buildCurrent()
		installBinary()
	case *run:
		buildCurrent()
		runBinary()
	default:
		buildCurrent()
	}
}

func showHelp() {
	fmt.Printf(`Cross-platform build script for dotfiles manager

Usage: go run build.go [OPTIONS]

OPTIONS:
    -help           Show this help message
    -version=X      Set version (default: dev)
    -clean          Clean build directory
    -all            Build for all platforms
    -run            Build and run the binary
    -test           Run tests
    -coverage       Run tests with coverage
    -install        Build and install binary
    -fmt            Format code
    -deps           Download and tidy dependencies
    -v              Verbose output

EXAMPLES:
    go run build.go                    # Build for current platform
    go run build.go -all               # Build for all platforms
    go run build.go -clean -all        # Clean and build for all platforms
    go run build.go -version=1.0.0     # Build with specific version
    go run build.go -run               # Build and run
    go run build.go -test              # Run tests
    go run build.go -install           # Build and install

`)
}

func buildCurrent() {
	log("Building for current platform...")

	target := BuildTarget{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	if err := build(target, ""); err != nil {
		fatal("Build failed: %v", err)
	}

	success("Build completed successfully!")
}

func buildAll() {
	log("Building for all platforms...")

	for _, target := range targets {
		suffix := fmt.Sprintf("-%s-%s", target.OS, target.Arch)

		if err := build(target, suffix); err != nil {
			fatal("Build failed for %s/%s: %v", target.OS, target.Arch, err)
		}

		success("Built %s/%s", target.OS, target.Arch)
	}

	success("All builds completed successfully!")
}

func build(target BuildTarget, suffix string) error {
	// Ensure build directory exists
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	// Determine output filename
	outputName := binaryName + suffix
	if target.OS == "windows" {
		outputName += ".exe"
	}
	outputPath := filepath.Join(buildDir, outputName)

	// Build info
	commit := getCommit()
	date := time.Now().UTC().Format(time.RFC3339)

	// Linker flags
	ldflags := fmt.Sprintf("-X main.version=%s -X main.commit=%s -X main.date=%s",
		*version, commit, date)

	// Build command
	args := []string{
		"build",
		"-ldflags", ldflags,
		"-o", outputPath,
		filepath.Join(cmdDir, "main.go"),
	}

	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(),
		"GOOS="+target.OS,
		"GOARCH="+target.Arch,
	)

	if *verbose {
		log("Running: go %s", strings.Join(args, " "))
		log("Environment: GOOS=%s GOARCH=%s", target.OS, target.Arch)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w\nOutput: %s", err, output)
	}

	return nil
}

func cleanBuild() {
	log("Cleaning build directory...")

	if err := os.RemoveAll(buildDir); err != nil {
		fatal("Failed to clean build directory: %v", err)
	}

	// Also clean test artifacts
	patterns := []string{"coverage.out", "coverage.html", "*.test"}
	for _, pattern := range patterns {
		if matches, err := filepath.Glob(pattern); err == nil {
			for _, match := range matches {
				os.Remove(match)
			}
		}
	}

	success("Build directory and artifacts cleaned")
}

func runTests() {
	log("Running tests...")

	cmd := exec.Command("go", "test", "-v", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fatal("Tests failed: %v", err)
	}

	success("All tests passed!")
}

func runTestsWithCoverage() {
	log("Running tests with coverage...")

	// Run tests with coverage
	cmd := exec.Command("go", "test", "-v", "-coverprofile=coverage.out", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fatal("Tests failed: %v", err)
	}

	// Generate HTML coverage report
	cmd = exec.Command("go", "tool", "cover", "-html=coverage.out", "-o=coverage.html")
	if err := cmd.Run(); err != nil {
		warn("Failed to generate HTML coverage report: %v", err)
	} else {
		success("Coverage report generated: coverage.html")
	}

	success("Tests completed with coverage!")
}

func formatCode() {
	log("Formatting code...")

	cmd := exec.Command("go", "fmt", "./...")
	output, err := cmd.CombinedOutput()

	if err != nil {
		fatal("Failed to format code: %v\nOutput: %s", err, output)
	}

	if len(output) > 0 {
		log("Formatted files:\n%s", output)
	}

	success("Code formatted successfully!")
}

func downloadDeps() {
	log("Downloading dependencies...")

	// Download dependencies
	cmd := exec.Command("go", "mod", "download")
	if *verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		fatal("Failed to download dependencies: %v", err)
	}

	// Tidy dependencies
	log("Tidying dependencies...")
	cmd = exec.Command("go", "mod", "tidy")
	if *verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		fatal("Failed to tidy dependencies: %v", err)
	}

	success("Dependencies updated successfully!")
}

func installBinary() {
	log("Installing binary...")

	// Determine binary path
	binaryPath := filepath.Join(buildDir, binaryName)
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	if !fileExists(binaryPath) {
		fatal("Binary not found: %s. Run build first.", binaryPath)
	}

	// Determine installation path
	var installPath string
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		installPath = filepath.Join(gopath, "bin", binaryName)
		if runtime.GOOS == "windows" {
			installPath += ".exe"
		}
	} else if gobin := os.Getenv("GOBIN"); gobin != "" {
		installPath = filepath.Join(gobin, binaryName)
		if runtime.GOOS == "windows" {
			installPath += ".exe"
		}
	} else {
		// Try to use go env GOPATH
		cmd := exec.Command("go", "env", "GOPATH")
		output, err := cmd.Output()
		if err != nil {
			fatal("Cannot determine installation path. Set GOPATH or GOBIN.")
		}
		gopath := strings.TrimSpace(string(output))
		installPath = filepath.Join(gopath, "bin", binaryName)
		if runtime.GOOS == "windows" {
			installPath += ".exe"
		}
	}

	// Ensure install directory exists
	installDir := filepath.Dir(installPath)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		fatal("Failed to create install directory: %v", err)
	}

	// Copy binary
	if err := copyFile(binaryPath, installPath); err != nil {
		fatal("Failed to install binary: %v", err)
	}

	// Make executable on Unix-like systems
	if runtime.GOOS != "windows" {
		if err := os.Chmod(installPath, 0755); err != nil {
			warn("Failed to make binary executable: %v", err)
		}
	}

	success("Installed to %s", installPath)
}

func runBinary() {
	log("Running binary...")

	binaryPath := filepath.Join(buildDir, binaryName)
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	if !fileExists(binaryPath) {
		fatal("Binary not found: %s", binaryPath)
	}

	// Run with help flag to show available commands
	cmd := exec.Command(binaryPath, "--help")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fatal("Failed to run binary: %v", err)
	}
}

// Utility functions

func getCommit() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func commandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0644)
}

// Logging functions

func log(format string, args ...interface{}) {
	fmt.Printf("\033[34m[BUILD]\033[0m "+format+"\n", args...)
}

func success(format string, args ...interface{}) {
	fmt.Printf("\033[32m[SUCCESS]\033[0m "+format+"\n", args...)
}

func warn(format string, args ...interface{}) {
	fmt.Printf("\033[33m[WARN]\033[0m "+format+"\n", args...)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "\033[31m[ERROR]\033[0m "+format+"\n", args...)
	os.Exit(1)
}
