package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/logger"
)

// createFetchCommand creates the fetch command
func createFetchCommand() *cobra.Command {
	var (
		rebase bool
		force  bool
	)

	fetchCmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch latest changes from the dotfiles repository",
		Long: `Fetch and pull the latest changes from the remote dotfiles repository.
This command will:
- Check if the current directory is a git repository
- Fetch the latest changes from the remote
- Pull changes using git pull (or git pull --rebase if --rebase is specified)

Use --rebase to rebase your local changes on top of the remote changes.
Use --force to force pull even if there are uncommitted changes (stashes them first).`,
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()

			// Find the dotfiles directory
			dotfilesDir, err := findDotfilesDirectory()
			if err != nil {
				log.Error().Err(err).Msg("Failed to find dotfiles directory")
				os.Exit(1)
			}

			log.Info().Str("directory", dotfilesDir).Msg("Found dotfiles directory")

			// Check if it's a git repository
			if !isGitRepository(dotfilesDir) {
				log.Error().Msg("Current dotfiles directory is not a git repository")
				log.Info().Msg("Initialize git repository with: git init && git remote add origin <your-repo-url>")
				os.Exit(1)
			}

			// Check for uncommitted changes
			if hasUncommittedChanges(dotfilesDir) {
				if force {
					log.Warn().Msg("Uncommitted changes detected. Stashing them before pull...")
					if err := stashChanges(dotfilesDir); err != nil {
						log.Error().Err(err).Msg("Failed to stash changes")
						os.Exit(1)
					}
				} else {
					log.Error().Msg("Uncommitted changes detected in dotfiles directory")
					log.Info().Msg("Commit your changes or use --force to stash them automatically")
					os.Exit(1)
				}
			}

			// Fetch latest changes
			log.Info().Msg("Fetching latest changes from remote...")
			if err := fetchChanges(dotfilesDir); err != nil {
				log.Error().Err(err).Msg("Failed to fetch changes")
				os.Exit(1)
			}

			// Pull changes
			log.Info().Msg("Pulling latest changes...")
			if err := pullChanges(dotfilesDir, rebase); err != nil {
				log.Error().Err(err).Msg("Failed to pull changes")
				os.Exit(1)
			}

			log.Info().Msg("Successfully updated dotfiles repository!")

			// Show latest commits
			if err := showRecentCommits(dotfilesDir); err != nil {
				log.Warn().Err(err).Msg("Failed to show recent commits")
			}
		},
	}

	fetchCmd.Flags().BoolVar(&rebase, "rebase", false, "Use rebase instead of merge when pulling")
	fetchCmd.Flags().BoolVar(&force, "force", false, "Force pull by stashing uncommitted changes")

	return fetchCmd
}

// findDotfilesDirectory finds the dotfiles directory by looking for the config file
func findDotfilesDirectory() (string, error) {
	// First try to find config file which will give us the dotfiles directory
	configPath, err := findConfigFile()
	if err != nil {
		return "", fmt.Errorf("could not find dotfiles configuration: %w", err)
	}

	// Return the directory containing the config file
	return filepath.Dir(configPath), nil
}

// isGitRepository checks if the given directory is a git repository
func isGitRepository(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return false
	}
	return true
}

// hasUncommittedChanges checks if there are uncommitted changes in the repository
func hasUncommittedChanges(dir string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return len(strings.TrimSpace(string(output))) > 0
}

// stashChanges stashes uncommitted changes
func stashChanges(dir string) error {
	cmd := exec.Command("git", "stash", "push", "-m", "dotfiles fetch: auto-stash before pull")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// fetchChanges fetches the latest changes from remote
func fetchChanges(dir string) error {
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// pullChanges pulls the latest changes from remote
func pullChanges(dir string, useRebase bool) error {
	var cmd *exec.Cmd

	if useRebase {
		cmd = exec.Command("git", "pull", "--rebase", "origin")
	} else {
		cmd = exec.Command("git", "pull", "origin")
	}

	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// showRecentCommits shows the most recent commits
func showRecentCommits(dir string) error {
	fmt.Println("\n📋 Recent commits:")

	cmd := exec.Command("git", "log", "--oneline", "-5", "--color=always")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
