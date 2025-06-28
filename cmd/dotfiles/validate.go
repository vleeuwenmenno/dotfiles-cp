package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/config"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/jobs"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules/files"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules/symlinks"

	"github.com/spf13/cobra"
)

// createValidateCommand creates the validate command
func createValidateCommand() *cobra.Command {
	var (
		platform    string
		shell       string
		environment []string
		verbose     bool
	)

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate dotfiles configuration",
		Long: `Validate the dotfiles configuration by checking:
- Configuration file syntax and structure
- Variable definitions and merging (including conflict detection)
- Job definitions and imports
- Template syntax and variable references
- Module action validation

This command performs all validation checks without making any changes to your system.`,
		Run: func(cmd *cobra.Command, args []string) {
			errorCount := 0
			checkCount := 0

			fmt.Printf("🔍 Validating dotfiles configuration...\n\n")

			// 1. Validate main configuration file
			fmt.Printf("📋 Checking main configuration file...\n")
			checkCount++

			configPath, err := findConfigFile()
			if err != nil {
				fmt.Printf("   ❌ Failed to find configuration file: %v\n", err)
				errorCount++
			} else {
				cfg, err := config.Load(configPath)
				if err != nil {
					fmt.Printf("   ❌ Failed to load configuration: %v\n", err)
					errorCount++
				} else {
					fmt.Printf("   ✅ Configuration file loaded successfully\n")

					// Validate configuration structure
					if err := cfg.Validate(); err != nil {
						fmt.Printf("   ❌ Configuration validation failed: %v\n", err)
						errorCount++
					} else {
						fmt.Printf("   ✅ Configuration structure is valid\n")
					}
				}
			}

			if errorCount > 0 {
				fmt.Printf("\n❌ Cannot continue validation due to configuration errors\n")
				os.Exit(1)
			}

			basePath := filepath.Dir(configPath)
			cfg, _ := config.Load(configPath) // We know this works from above

			// 2. Validate variables
			fmt.Printf("\n📊 Checking variables...\n")
			checkCount++

			vloader, err := config.NewVariableLoader(cfg, basePath)
			if err != nil {
				fmt.Printf("   ❌ Failed to create variable loader: %v\n", err)
				errorCount++
			} else {
				// Create variable load options
				opts := &config.VariableLoadOptions{
					Platform:    platform,
					Shell:       shell,
					Environment: parseEnvironmentVariables(environment),
				}

				variables, err := vloader.LoadAllVariables(opts)
				if err != nil {
					// Check if it's a variable conflict error for special handling
					if conflictErr, isConflict := config.IsVariableConflictError(err); isConflict {
						fmt.Printf("   ❌ Variable validation failed:\n")
						fmt.Print(conflictErr.PrettyPrint())
					} else {
						fmt.Printf("   ❌ Variable validation failed: %v\n", err)
					}
					errorCount++
				} else {
					fmt.Printf("   ✅ Variables loaded and merged successfully\n")
					fmt.Printf("   ℹ️  Loaded %d variables\n", len(variables))

					// Show variable sources summary
					sources := vloader.GetVariableSources()
					sourceFiles := make(map[string]int)
					for _, source := range sources {
						sourceFiles[source.Source]++
					}
					fmt.Printf("   ℹ️  Variable sources: %d files\n", len(sourceFiles))
					if verbose {
						for file, count := range sourceFiles {
							fmt.Printf("      - %s: %d variables\n", file, count)
						}
					}
				}
			}

			// 3. Validate jobs
			fmt.Printf("\n🔧 Checking jobs...\n")
			checkCount++

			if errorCount == 0 { // Only check jobs if variables are valid
				variables, _ := vloader.LoadAllVariables(&config.VariableLoadOptions{
					Platform:    platform,
					Shell:       shell,
					Environment: parseEnvironmentVariables(environment),
				})

				jobsIndexPath := cfg.GetJobsIndexPath(basePath)
				tasksList, err := jobs.LoadJobsFromFileWithConditions(jobsIndexPath, variables)
				if err != nil {
					fmt.Printf("   ❌ Job validation failed: %v\n", err)
					errorCount++
				} else {
					fmt.Printf("   ✅ Jobs loaded successfully\n")
					fmt.Printf("   ℹ️  Loaded %d jobs (after condition filtering)\n", len(tasksList))

					// Group jobs by source
					jobSources := make(map[string]int)
					for _, task := range tasksList {
						if task.Source != "" {
							jobSources[task.Source]++
						} else {
							jobSources["main"]++
						}
					}
					fmt.Printf("   ℹ️  Job sources: %d files\n", len(jobSources))
					if verbose {
						for file, count := range jobSources {
							fmt.Printf("      - %s: %d jobs\n", file, count)
						}
					}

					// 4. Validate individual job configurations
					fmt.Printf("\n🎯 Checking job configurations...\n")
					checkCount++

					// Create module registry for validation
					registry := modules.NewModuleRegistry()
					if err := registry.Register(files.New()); err != nil {
						fmt.Printf("   ❌ Failed to register files module: %v\n", err)
						errorCount++
					}
					if err := registry.Register(symlinks.New()); err != nil {
						fmt.Printf("   ❌ Failed to register symlinks module: %v\n", err)
						errorCount++
					}

					validJobs := 0
					invalidJobs := 0

					for _, task := range tasksList {
						if err := registry.ValidateTask(task); err != nil {
							fmt.Printf("   ❌ Job '%s' validation failed: %v\n", task.ID, err)
							invalidJobs++
							errorCount++
						} else {
							validJobs++
						}
					}

					if invalidJobs == 0 {
						fmt.Printf("   ✅ All %d jobs are valid\n", validJobs)
					} else {
						fmt.Printf("   ⚠️  %d valid jobs, %d invalid jobs\n", validJobs, invalidJobs)
					}

					// 5. Test job planning (template validation)
					if invalidJobs == 0 {
						fmt.Printf("\n🎨 Checking templates and planning...\n")
						checkCount++

						ctx := &modules.ExecutionContext{
							BasePath:    basePath,
							Variables:   variables,
							DryRun:      true,
							Verbose:     false,
							ShowDiff:    false,
							HideSkipped: true,
						}

						planningErrors := 0
						for _, task := range tasksList {
							_, err := registry.PlanTask(task, ctx)
							if err != nil {
								fmt.Printf("   ❌ Template/planning error in '%s': %v\n", task.ID, err)
								planningErrors++
								errorCount++
							}
						}

						if planningErrors == 0 {
							fmt.Printf("   ✅ All job templates and planning successful\n")
						} else {
							fmt.Printf("   ❌ %d template/planning errors found\n", planningErrors)
						}
					}
				}
			}

			// Summary
			fmt.Printf("\n" + strings.Repeat("=", 50) + "\n")
			fmt.Printf("📊 Validation Summary\n")
			fmt.Printf(strings.Repeat("=", 50) + "\n")
			fmt.Printf("Checks performed: %d\n", checkCount)

			if errorCount == 0 {
				fmt.Printf("Status: ✅ All validations passed\n")
				fmt.Printf("\n🎉 Your dotfiles configuration is valid and ready to use!\n")
			} else {
				fmt.Printf("Status: ❌ %d error(s) found\n", errorCount)
				fmt.Printf("\n🔧 Please fix the errors above before applying your configuration.\n")
				os.Exit(1)
			}
		},
	}

	validateCmd.Flags().StringVar(&platform, "platform", "", "Override platform detection (windows, linux, darwin)")
	validateCmd.Flags().StringVar(&shell, "shell", "", "Override shell detection (bash, zsh, powershell)")
	validateCmd.Flags().StringSliceVarP(&environment, "env", "e", []string{}, "Set environment variables (KEY=VALUE)")
	validateCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information about sources and jobs")

	return validateCmd
}

// parseEnvironmentVariables parses environment variables from string slice
func parseEnvironmentVariables(envVars []string) map[string]string {
	result := make(map[string]string)
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}
