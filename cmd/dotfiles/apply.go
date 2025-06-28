package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/config"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/jobs"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/logger"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules"

	"github.com/spf13/cobra"
)

// createApplyCommand creates the apply command
func createApplyCommand() *cobra.Command {
	var (
		platform    string
		shell       string
		environment []string
		dryRun      bool
	)

	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply dotfiles configuration to the system",
		Long: `Apply all configured jobs to set up your dotfiles environment.
This command will:
- Create required directories
- Set up symlinks to configuration files
- Install packages
- Process templates
- Run any configured scripts

Use --dry-run to see what would be done without actually making changes.`,
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()

			// Find and load configuration
			configPath, err := findConfigFile()
			if err != nil {
				log.Error().Err(err).Msg("Failed to find configuration file")
				os.Exit(1)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				log.Error().Err(err).Msg("Failed to load configuration")
				os.Exit(1)
			}

			// Get base path
			basePath := filepath.Dir(configPath)

			// Load variables
			vloader, err := config.NewVariableLoader(cfg, basePath)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create variable loader")
				os.Exit(1)
			}

			// Prepare variable load options
			opts := &config.VariableLoadOptions{}
			if platform != "" {
				opts.Platform = platform
			}
			if shell != "" {
				opts.Shell = shell
			}
			if len(environment) > 0 {
				opts.Environment = make(map[string]string)
				for _, env := range environment {
					parts := strings.SplitN(env, "=", 2)
					if len(parts) == 2 {
						opts.Environment[parts[0]] = parts[1]
					}
				}
			}

			variables, err := vloader.LoadAllVariables(opts)
			if err != nil {
				log.Error().Err(err).Msg("Failed to load variables")
				os.Exit(1)
			}

			// Load jobs
			jobsIndexPath := cfg.GetJobsIndexPath(basePath)
			tasksList, err := jobs.LoadJobsFromFile(jobsIndexPath)
			if err != nil {
				log.Error().Err(err).Msg("Failed to load jobs")
				os.Exit(1)
			}

			if len(tasksList) == 0 {
				log.Info().Msg("No jobs found. Check your jobs/index.yaml file.")
				return
			}

			// Create module registry
			registry := createModuleRegistry()

			// Create execution context
			ctx := &modules.ExecutionContext{
				BasePath:  basePath,
				Variables: variables,
				DryRun:    dryRun,
				Verbose:   verbose,
			}

			// Show what we're about to do
			if dryRun {
				fmt.Printf("🧪 DRY RUN - No changes will be made\n\n")
			} else {
				fmt.Printf("🚀 Applying dotfiles configuration...\n\n")
			}

			// Execute all jobs
			successCount := 0
			skipCount := 0
			failCount := 0

			for i, task := range tasksList {
				fmt.Printf("[%d/%d] %s (%s)\n", i+1, len(tasksList), task.ID, task.Action)

				// Plan the task first
				plan, err := registry.PlanTask(task, ctx)
				if err != nil {
					log.Error().Err(err).Str("task", task.ID).Msg("Failed to plan task")
					failCount++
					continue
				}

				// Show what will be done
				if plan.WillSkip {
					fmt.Printf("   ⏭️  SKIP: %s\n", plan.SkipReason)
					skipCount++
					fmt.Println()
					continue
				}

				if dryRun {
					fmt.Printf("   📋 Would do:\n")
					for _, change := range plan.Changes {
						fmt.Printf("      - %s\n", change)
					}
				} else {
					fmt.Printf("   📋 Description: %s\n", plan.Description)
					if verbose && len(plan.Changes) > 0 {
						fmt.Printf("   Changes:\n")
						for _, change := range plan.Changes {
							fmt.Printf("      - %s\n", change)
						}
					}
				}

				// Execute the task (unless dry run)
				if !dryRun {
					result, err := registry.ExecuteTask(task, ctx)
					if err != nil {
						log.Error().Err(err).Str("task", task.ID).Msg("Failed to execute task")
						fmt.Printf("   ❌ FAILED: %v\n", err)
						failCount++
					} else if result.Success {
						fmt.Printf("   ✅ SUCCESS\n")
						successCount++
					} else {
						fmt.Printf("   ❌ FAILED: %s\n", result.Message)
						failCount++
					}
				} else {
					successCount++
				}

				fmt.Println()
			}

			// Summary
			if dryRun {
				fmt.Printf("📊 Dry Run Summary:\n")
				fmt.Printf("   Would execute: %d jobs\n", successCount)
				fmt.Printf("   Would skip: %d jobs\n", skipCount)
				if failCount > 0 {
					fmt.Printf("   Failed to plan: %d jobs\n", failCount)
				}
			} else {
				fmt.Printf("📊 Execution Summary:\n")
				fmt.Printf("   Successful: %d jobs\n", successCount)
				fmt.Printf("   Skipped: %d jobs\n", skipCount)
				if failCount > 0 {
					fmt.Printf("   Failed: %d jobs\n", failCount)
					os.Exit(1)
				}
			}

			if failCount == 0 {
				if dryRun {
					log.Info().Msg("Dry run completed successfully!")
				} else {
					log.Info().Msg("All jobs completed successfully!")
				}
			}
		},
	}

	applyCmd.Flags().StringVar(&platform, "platform", "", "Override platform detection (windows, linux, darwin)")
	applyCmd.Flags().StringVar(&shell, "shell", "", "Override shell detection (bash, zsh, powershell)")
	applyCmd.Flags().StringSliceVarP(&environment, "env", "e", []string{}, "Set environment variables (KEY=VALUE)")
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")

	return applyCmd
}
