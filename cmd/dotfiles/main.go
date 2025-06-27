package main

import (
	"os"

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

	// Add install command placeholder
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install dotfiles and packages",
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()
			log.Info().Msg("Installing dotfiles...")
			log.Warn().Msg("Command not yet implemented")
		},
	}

	// Add commands to root
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(installCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		log := zerolog.New(os.Stderr).With().Timestamp().Logger()
		log.Error().Err(err).Msg("Command execution failed")
		os.Exit(1)
	}
}
