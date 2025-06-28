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
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules/files"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules/symlinks"

	"github.com/spf13/cobra"
)

// createPlanCommand creates the plan command
func createPlanCommand() *cobra.Command {
	var (
		platform    string
		shell       string
		environment []string
	)

	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Show execution plan for jobs",
		Long: `Show what would be executed without actually running the jobs.
This helps you understand what changes would be made to your system.

The plan shows:
- Job execution order
- What files would be created/modified
- What packages would be installed
- Any jobs that would be skipped and why`,
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

			// Create module registry
			registry := createModuleRegistry()

			// Create execution context
			ctx := &modules.ExecutionContext{
				BasePath:  basePath,
				Variables: variables,
				DryRun:    true,
				Verbose:   verbose,
			}

			// Generate plans for all jobs
			fmt.Printf("📋 Execution Plan (%d jobs):\n\n", len(tasksList))

			for i, task := range tasksList {
				plan, err := registry.PlanTask(task, ctx)
				if err != nil {
					log.Error().Err(err).Str("task", task.ID).Msg("Failed to plan task")
					continue
				}

				// Display plan
				fmt.Printf("%d. %s (%s)\n", i+1, plan.TaskID, plan.Action)
				fmt.Printf("   Description: %s\n", plan.Description)

				if plan.WillSkip {
					fmt.Printf("   ⚠️  SKIP: %s\n", plan.SkipReason)
				} else if len(plan.Changes) > 0 {
					fmt.Printf("   Changes:\n")
					for _, change := range plan.Changes {
						fmt.Printf("     - %s\n", change)
					}
				} else {
					fmt.Printf("   ✅ No changes needed\n")
				}
				fmt.Println()
			}

			if len(tasksList) == 0 {
				fmt.Println("No jobs found. Check your jobs/index.yaml file.")
			}
		},
	}

	planCmd.Flags().StringVar(&platform, "platform", "", "Override platform detection (windows, linux, darwin)")
	planCmd.Flags().StringVar(&shell, "shell", "", "Override shell detection (bash, zsh, powershell)")
	planCmd.Flags().StringSliceVarP(&environment, "env", "e", []string{}, "Set environment variables (KEY=VALUE)")

	return planCmd
}

// createModuleRegistry creates a registry with available modules
func createModuleRegistry() *modules.ModuleRegistry {
	registry := modules.NewModuleRegistry()

	// Register available modules
	if err := registry.Register(symlinks.New()); err != nil {
		fmt.Printf("Warning: Failed to register symlinks module: %v\n", err)
	}

	if err := registry.Register(files.New()); err != nil {
		fmt.Printf("Warning: Failed to register files module: %v\n", err)
	}

	return registry
}
