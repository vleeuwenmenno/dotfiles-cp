package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/config"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/logger"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/platform"
	"github.com/vleeuwenmenno/dotfiles-cp/pkg/utils"

	"github.com/spf13/cobra"
)

// GitStatus represents the current git repository status
type GitStatus struct {
	IsRepo           bool
	Branch           string
	IsClean          bool
	ModifiedFiles    []string
	UntrackedFiles   []string
	StagedFiles      []string
	AheadCount       int
	BehindCount      int
	RemoteCommits    []string
	HasRemote        bool
	CanFetch         bool
}

// ConfigStatus represents the current configuration status
type ConfigStatus struct {
	ConfigExists     bool
	ConfigPath       string
	ManagedFiles     int
	TemplateFiles    int
	LastApplied      time.Time
	ValidSymlinks    int
	BrokenSymlinks   int
	MissingSymlinks  int
}

// createStatusCommand creates the status command
func createStatusCommand() *cobra.Command {
	var (
		verbose  bool
		jsonOut  bool
		noFetch  bool
	)

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of dotfiles configuration",
		Long: `Show comprehensive status information about your dotfiles configuration:

- Git repository status and remote changes
- Configuration file status and health
- System integration status
- Template and symlink health check

Use --verbose for detailed output, --json for machine-readable format.`,
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()

			// Find dotfiles root directory by locating config file
			configPath, err := config.FindConfigFile()
			var dotfilesDir string
			if err != nil {
				// If no config found, fall back to current directory
				dotfilesDir, err = os.Getwd()
				if err != nil {
					log.Error().Err(err).Msg("Failed to get current directory")
					os.Exit(1)
				}
			} else {
				// Use the directory containing the config file as dotfiles root
				dotfilesDir = filepath.Dir(configPath)
			}

			// Get platform info
			platformInfo, err := platform.GetPlatformInfo()
			if err != nil {
				log.Error().Err(err).Msg("Failed to get platform information")
				os.Exit(1)
			}

			// Get Git status
			gitStatus := getGitStatus(dotfilesDir, !noFetch)

			// Get configuration status
			configStatus := getConfigStatus(dotfilesDir)

			// Output results
			if jsonOut {
				outputStatusJSON(gitStatus, configStatus, platformInfo)
			} else {
				outputStatusText(gitStatus, configStatus, platformInfo, verbose)
			}
		},
	}

	statusCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed status information")
	statusCmd.Flags().BoolVar(&jsonOut, "json", false, "Output status in JSON format")
	statusCmd.Flags().BoolVar(&noFetch, "no-fetch", false, "Skip fetching remote changes")

	return statusCmd
}

// getGitStatus analyzes the git repository status
func getGitStatus(dir string, shouldFetch bool) *GitStatus {
	status := &GitStatus{
		IsRepo: isGitRepository(dir),
	}

	if !status.IsRepo {
		return status
	}

	log := logger.Get()

	// Get current branch
	if branch := getGitBranch(dir); branch != "" {
		status.Branch = branch
	}

	// Check if we have a remote
	status.HasRemote = hasGitRemote(dir)

	// Get working directory status
	status.ModifiedFiles = getModifiedFiles(dir)
	status.UntrackedFiles = getUntrackedFiles(dir)
	status.StagedFiles = getStagedFiles(dir)
	status.IsClean = len(status.ModifiedFiles) == 0 && len(status.UntrackedFiles) == 0 && len(status.StagedFiles) == 0

	// Get ahead/behind counts
	if status.HasRemote {
		status.AheadCount, status.BehindCount = getAheadBehindCount(dir)

		// Fetch remote changes if requested
		if shouldFetch {
			status.CanFetch = true
			if err := fetchGitChanges(dir); err != nil {
				log.Warn().Err(err).Msg("Failed to fetch remote changes")
				status.CanFetch = false
			} else {
				// Get remote commits after fetch
				status.RemoteCommits = getRemoteCommits(dir, 5)
			}
		}
	}

	return status
}

// getConfigStatus analyzes the dotfiles configuration status
func getConfigStatus(dotfilesDir string) *ConfigStatus {
	status := &ConfigStatus{}

	// Look for config file in the dotfiles directory
	configPaths := []string{
		filepath.Join(dotfilesDir, "dotfiles.yaml"),
		filepath.Join(dotfilesDir, "dotfiles.yml"),
		filepath.Join(dotfilesDir, ".dotfiles.yaml"),
		filepath.Join(dotfilesDir, ".dotfiles.yml"),
	}

	var configPath string
	for _, path := range configPaths {
		if utils.FileExists(path) {
			configPath = path
			break
		}
	}

	if configPath == "" {
		return status
	}

	status.ConfigExists = true
	status.ConfigPath = configPath

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		return status
	}

	// Use dotfiles directory as base directory
	baseDir := dotfilesDir

	// Count managed files and templates
	status.ManagedFiles, status.TemplateFiles = countManagedFiles(cfg, baseDir)

	// Check symlink health
	status.ValidSymlinks, status.BrokenSymlinks, status.MissingSymlinks = checkSymlinkHealth(cfg, baseDir)

	// Get last applied time (check for .dotfiles-last-applied file)
	lastAppliedPath := filepath.Join(dotfilesDir, ".dotfiles-last-applied")
	if stat, err := os.Stat(lastAppliedPath); err == nil {
		status.LastApplied = stat.ModTime()
	}

	return status
}

// outputStatusText outputs status in human-readable format
func outputStatusText(git *GitStatus, cfg *ConfigStatus, platform *platform.PlatformInfo, verbose bool) {
	if verbose {
		fmt.Println("Dotfiles Status (Detailed)")
		fmt.Println("─────────────────────────")
	} else {
		fmt.Println("Dotfiles Status")
		fmt.Println("──────────────")
	}

	// Configuration Status
	if cfg.ConfigExists {
		filesStr := fmt.Sprintf("%d files managed", cfg.ManagedFiles)
		if cfg.TemplateFiles > 0 {
			filesStr += fmt.Sprintf(" (%d templates)", cfg.TemplateFiles)
		}

		symlinkStr := ""
		if cfg.ValidSymlinks > 0 || cfg.BrokenSymlinks > 0 {
			symlinkStr = fmt.Sprintf(" (%d symlinks", cfg.ValidSymlinks)
			if cfg.BrokenSymlinks > 0 {
				symlinkStr += fmt.Sprintf(", %d broken", cfg.BrokenSymlinks)
			}
			symlinkStr += ")"
		}

		fmt.Printf("📁 Configuration: %s%s\n", filesStr, symlinkStr)

		if verbose && !cfg.LastApplied.IsZero() {
			fmt.Printf("  └── Last applied: %s\n", cfg.LastApplied.Format("2006-01-02 15:04 MST"))
		}
	} else {
		fmt.Println("📁 Configuration: No configuration found")
	}

	// Git Status
	if git.IsRepo {
		gitDetails := fmt.Sprintf("%s branch", git.Branch)
		if !git.IsClean {
			modCount := len(git.ModifiedFiles) + len(git.UntrackedFiles) + len(git.StagedFiles)
			gitDetails += fmt.Sprintf(", %d files modified", modCount)
		} else {
			gitDetails += ", clean"
		}

		if git.AheadCount > 0 {
			gitDetails += fmt.Sprintf(", %d commit(s) ahead", git.AheadCount)
		}

		fmt.Printf("🔄 Git: %s\n", gitDetails)

		// Remote status
		if git.HasRemote {
			if git.BehindCount > 0 {
				fmt.Printf("⚡ Remote: %d new commits available\n", git.BehindCount)
			} else if git.CanFetch {
				fmt.Println("⚡ Remote: Up to date")
			}
		}
	} else {
		fmt.Println("🔄 Git: Not a git repository")
	}

	// System Integration
	integrationStr := platform.Shell
	if len(platform.PackageManagers) > 0 {
		integrationStr += fmt.Sprintf(", %s", strings.Join(platform.PackageManagers, ", "))
	}
	fmt.Printf("🔧 Integration: %s\n", integrationStr)

	// Show modified files if any
	if !git.IsClean && len(git.ModifiedFiles) > 0 {
		fmt.Println()
		fmt.Println("Modified Files:")
		for _, file := range git.ModifiedFiles {
			fileType := getFileType(file)
			if fileType != "" {
				fmt.Printf("  M %s\n    └── %s\n", file, fileType)
			} else {
				fmt.Printf("  M %s\n", file)
			}
		}
	}

	// Show untracked files if any
	if len(git.UntrackedFiles) > 0 {
		if len(git.ModifiedFiles) == 0 {
			fmt.Println()
		}
		fmt.Println("Untracked Files:")
		for _, file := range git.UntrackedFiles {
			fmt.Printf("  ? %s\n", file)
		}
	}

	// Show remote commits if available (verbose mode)
	if verbose && len(git.RemoteCommits) > 0 {
		fmt.Println()
		fmt.Println("🔄 Recent Remote Commits:")
		for _, commit := range git.RemoteCommits {
			fmt.Printf("  ├── %s\n", commit)
		}
	}

	// Show broken symlinks if any
	if cfg.BrokenSymlinks > 0 {
		fmt.Println()
		fmt.Printf("⚠️  %d broken symlinks detected\n", cfg.BrokenSymlinks)
	}
}

// outputStatusJSON outputs status in JSON format
func outputStatusJSON(git *GitStatus, cfg *ConfigStatus, platform *platform.PlatformInfo) {
	status := map[string]interface{}{
		"git": map[string]interface{}{
			"is_repo":         git.IsRepo,
			"branch":          git.Branch,
			"is_clean":        git.IsClean,
			"modified_files":  git.ModifiedFiles,
			"untracked_files": git.UntrackedFiles,
			"staged_files":    git.StagedFiles,
			"ahead_count":     git.AheadCount,
			"behind_count":    git.BehindCount,
			"remote_commits":  git.RemoteCommits,
			"has_remote":      git.HasRemote,
		},
		"config": map[string]interface{}{
			"exists":           cfg.ConfigExists,
			"path":             cfg.ConfigPath,
			"managed_files":    cfg.ManagedFiles,
			"template_files":   cfg.TemplateFiles,
			"last_applied":     cfg.LastApplied,
			"valid_symlinks":   cfg.ValidSymlinks,
			"broken_symlinks":  cfg.BrokenSymlinks,
			"missing_symlinks": cfg.MissingSymlinks,
		},
		"platform": map[string]interface{}{
			"os":               platform.OS,
			"arch":             platform.Arch,
			"shell":            platform.Shell,
			"package_managers": platform.PackageManagers,
			"home_dir":         platform.HomeDir,
		},
	}

	fmt.Println(utils.ToJSONString(status))
}

// Git helper functions
func getGitBranch(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func hasGitRemote(dir string) bool {
	cmd := exec.Command("git", "remote")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

func getModifiedFiles(dir string) []string {
	cmd := exec.Command("git", "diff", "--name-only")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}
	}
	return files
}

func getUntrackedFiles(dir string) []string {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}
	}
	return files
}

func getStagedFiles(dir string) []string {
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}
	}
	return files
}

func getAheadBehindCount(dir string) (ahead, behind int) {
	cmd := exec.Command("git", "rev-list", "--count", "--left-right", "HEAD...@{upstream}")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) == 2 {
		fmt.Sscanf(parts[0], "%d", &ahead)
		fmt.Sscanf(parts[1], "%d", &behind)
	}
	return
}

func fetchGitChanges(dir string) error {
	cmd := exec.Command("git", "fetch", "--quiet")
	cmd.Dir = dir
	return cmd.Run()
}

func getRemoteCommits(dir string, count int) []string {
	cmd := exec.Command("git", "log", "--oneline", fmt.Sprintf("-%d", count), "@{upstream}", "--not", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	commits := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(commits) == 1 && commits[0] == "" {
		return []string{}
	}
	return commits
}

func getFileType(filename string) string {
	if strings.HasPrefix(filename, ".") {
		return "Dotfile changes"
	}
	if strings.Contains(filename, "template") || strings.HasSuffix(filename, ".tmpl") {
		return "Template variables changed"
	}
	if strings.Contains(filename, "config") {
		return "Configuration changes"
	}
	if strings.HasSuffix(filename, ".go") {
		return "Source code changes"
	}
	return "Local changes"
}

// Configuration helper functions
func countManagedFiles(cfg *config.Config, baseDir string) (managedFiles, templateFiles int) {
	// Count files in the files directory
	filesDir := cfg.GetFilesPath(baseDir)
	if utils.DirExists(filesDir) {
		err := filepath.Walk(filesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() {
				managedFiles++
				if strings.Contains(path, "template") || strings.HasSuffix(path, ".tmpl") {
					templateFiles++
				}
			}
			return nil
		})
		if err != nil {
			return 0, 0
		}
	}

	return managedFiles, templateFiles
}

func checkSymlinkHealth(cfg *config.Config, baseDir string) (valid, broken, missing int) {
	// This is a simplified check - in a real implementation, you'd want to
	// parse the jobs configuration to find actual symlink targets

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return 0, 0, 0
	}

	// Check common dotfile locations for symlinks
	commonDotfiles := []string{".gitconfig", ".zshrc", ".bashrc", ".vimrc", ".tmux.conf"}

	for _, dotfile := range commonDotfiles {
		path := filepath.Join(homeDir, dotfile)
		if info, err := os.Lstat(path); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				// It's a symlink, check if it's valid
				if _, err := os.Stat(path); err == nil {
					valid++
				} else {
					broken++
				}
			}
		}
	}

	return valid, broken, missing
}
