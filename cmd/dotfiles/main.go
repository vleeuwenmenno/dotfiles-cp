package main

import (
	"os"
	"os/exec"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/logger"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/platform"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	verbose bool
	quiet   bool
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "dotfiles",
		Short: "Cross-platform dotfiles manager with templating support",
		Long: `A powerful cross-platform dotfiles manager that supports templating,
package management integration, and works seamlessly across Windows, macOS, and Linux.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Initialize logger based on flags
			logger.Init(verbose, quiet)
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Enable quiet mode (errors only)")

	// Add version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()
			log.Info().
				Str("version", version).
				Str("commit", commit).
				Str("date", date).
				Msg("dotfiles manager")
		},
	}

	// Add info command to show platform details
	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Show platform and environment information",
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()

			info, err := platform.GetPlatformInfo()
			if err != nil {
				log.Error().Err(err).Msg("Failed to get platform information")
				os.Exit(1)
			}

			log.Info().
				Str("os", info.OS).
				Str("arch", info.Arch).
				Str("shell", info.Shell).
				Strs("package_managers", info.PackageManagers).
				Str("home_dir", info.HomeDir).
				Msg("Platform information")
		},
	}

	// Add init command placeholder
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new dotfiles repository",
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()
			log.Info().Msg("Initializing dotfiles repository...")
			log.Warn().Msg("Command not yet implemented")
		},
	}

	// Add apply command placeholder
	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply dotfiles configuration (symlinks, packages, scripts)",
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()
			log.Info().Msg("Applying dotfiles configuration...")
			log.Warn().Msg("Command not yet implemented")
		},
	}

	// Add backup command
	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup current configuration files",
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()
			log.Info().Msg("Backing up current configuration...")
			log.Warn().Msg("Command not yet implemented")
		},
	}

	// Add restore command
	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore configuration files from backup",
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()
			log.Info().Msg("Restoring configuration from backup...")
			log.Warn().Msg("Command not yet implemented")
		},
	}

	// Add status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of dotfiles configuration",
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()
			log.Info().Msg("Checking dotfiles status...")
			log.Warn().Msg("Command not yet implemented")
		},
	}

	// Add validate command
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate dotfiles configuration file",
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()
			log.Info().Msg("Validating configuration...")
			log.Warn().Msg("Command not yet implemented")
		},
	}

	// Add update command
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update dotfiles manager to the latest version",
		Long: `Update the dotfiles manager by running 'go install' with the latest version.
This requires Go to be installed on your system.

Use --check to only check for updates without installing.`,
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()

			checkOnly, _ := cmd.Flags().GetBool("check")

			if checkOnly {
				log.Info().Msg("Checking for updates...")
				log.Info().Str("current_version", version).Str("current_commit", commit).Msg("Current version")
				log.Info().Msg("To check for the latest version, visit: https://github.com/vleeuwenmenno/dotfiles-cp/releases")
				log.Info().Msg("To update, run: dotfiles update")
				return
			}

			log.Info().Msg("Updating dotfiles manager...")

			updateCmd := exec.Command("go", "install", "github.com/vleeuwenmenno/dotfiles-cp/cmd/dotfiles@latest")
			updateCmd.Stdout = os.Stdout
			updateCmd.Stderr = os.Stderr

			if err := updateCmd.Run(); err != nil {
				log.Error().Err(err).Msg("Failed to update dotfiles manager")
				log.Info().Msg("Make sure Go is installed and you have internet connectivity")
				os.Exit(1)
			}

			log.Info().Msg("Update completed successfully!")
			log.Info().Msg("You may need to restart your terminal or run 'hash -r' to refresh the binary")
		},
	}
	updateCmd.Flags().Bool("check", false, "Only check for updates without installing")

	// Add commands to root
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(updateCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		log := zerolog.New(os.Stderr).With().Timestamp().Logger()
		log.Error().Err(err).Msg("Command execution failed")
		os.Exit(1)
	}
}
