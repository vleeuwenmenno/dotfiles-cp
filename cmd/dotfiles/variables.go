package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/config"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/logger"

	"github.com/spf13/cobra"
)

// createVariablesCommand creates the variables command with subcommands
func createVariablesCommand() *cobra.Command {
	variablesCmd := &cobra.Command{
		Use:   "variables",
		Short: "Manage and inspect dotfiles variables",
		Long: `Manage and inspect variables used in dotfiles templates.
Variables are loaded from variables/ directory with proper precedence and inheritance.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Add subcommands
	variablesCmd.AddCommand(createVariablesListCommand())
	variablesCmd.AddCommand(createVariablesGetCommand())
	variablesCmd.AddCommand(createVariablesTraceCommand())
	variablesCmd.AddCommand(createVariablesSourcesCommand())

	return variablesCmd
}

// createVariablesListCommand creates the variables list subcommand
func createVariablesListCommand() *cobra.Command {
	var (
		platform    string
		shell       string
		hostname    string
		environment []string
		format      string
	)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all variables with their values",
		Long: `List all variables loaded from the variables directory.
Shows the final merged state after all imports and precedence rules are applied.`,
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

			// Create variable loader
			vloader, err := config.NewVariableLoader(cfg, basePath)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create variable loader")
				os.Exit(1)
			}

			// Prepare load options
			opts := &config.VariableLoadOptions{}
			if platform != "" {
				opts.Platform = platform
			}
			if shell != "" {
				opts.Shell = shell
			}
			if hostname != "" {
				opts.Hostname = hostname
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

			// Load variables
			variables, err := vloader.LoadAllVariables(opts)
			if err != nil {
				handleVariableError(err)
				os.Exit(1)
			}

			// Display variables
			if err := displayVariables(variables, format); err != nil {
				log.Error().Err(err).Msg("Failed to display variables")
				os.Exit(1)
			}
		},
	}

	listCmd.Flags().StringVar(&platform, "platform", "", "Override platform detection (windows, linux, darwin)")
	listCmd.Flags().StringVar(&shell, "shell", "", "Override shell detection (bash, zsh, powershell)")
	listCmd.Flags().StringVar(&hostname, "hostname", "", "Override hostname")
	listCmd.Flags().StringSliceVarP(&environment, "env", "e", []string{}, "Set environment variables (KEY=VALUE)")
	listCmd.Flags().StringVar(&format, "format", "yaml", "Output format (yaml, json, table)")

	return listCmd
}

// createVariablesGetCommand creates the variables get subcommand
func createVariablesGetCommand() *cobra.Command {
	var (
		platform    string
		shell       string
		hostname    string
		environment []string
		format      string
	)

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get the value of a specific variable",
		Long: `Get the value of a specific variable using dot notation.
Examples:
  dotfiles variables get user.name
  dotfiles variables get shell.theme
  dotfiles variables get platform.os`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()
			key := args[0]

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

			// Create variable loader
			vloader, err := config.NewVariableLoader(cfg, basePath)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create variable loader")
				os.Exit(1)
			}

			// Prepare load options
			opts := &config.VariableLoadOptions{}
			if platform != "" {
				opts.Platform = platform
			}
			if shell != "" {
				opts.Shell = shell
			}
			if hostname != "" {
				opts.Hostname = hostname
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

			// Load variables
			variables, err := vloader.LoadAllVariables(opts)
			if err != nil {
				handleVariableError(err)
				os.Exit(1)
			}

			// Get specific variable
			value, exists := vloader.GetVariable(key, variables)
			if !exists {
				log.Error().Str("key", key).Msg("Variable not found")
				os.Exit(1)
			}

			// Display value
			if err := displayValue(key, value, format); err != nil {
				log.Error().Err(err).Msg("Failed to display value")
				os.Exit(1)
			}
		},
	}

	getCmd.Flags().StringVar(&platform, "platform", "", "Override platform detection (windows, linux, darwin)")
	getCmd.Flags().StringVar(&shell, "shell", "", "Override shell detection (bash, zsh, powershell)")
	getCmd.Flags().StringVar(&hostname, "hostname", "", "Override hostname")
	getCmd.Flags().StringSliceVarP(&environment, "env", "e", []string{}, "Set environment variables (KEY=VALUE)")
	getCmd.Flags().StringVar(&format, "format", "yaml", "Output format (yaml, json, raw)")

	return getCmd
}

// createVariablesTraceCommand creates the variables trace subcommand
func createVariablesTraceCommand() *cobra.Command {
	var showRaw bool

	traceCmd := &cobra.Command{
		Use:   "trace <key>",
		Short: "Trace where a variable is defined and show its values",
		Long: `Trace where a specific variable is defined and show all sources that contribute to its value.

By default shows RENDERED values (the final result after template processing).
Use --raw to see the original template syntax for debugging template issues.

Supports dot notation for nested variables (e.g., user.details.location).

Examples:
  dotfiles variables trace user.name            # Shows a rendered value
  dotfiles variables trace user.name --raw      # Shows the original template syntax

This is useful for:
- Understanding variable precedence and inheritance
- Debugging template processing issues
- Finding which file defines a specific variable`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()
			key := args[0]

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

			// Create variable loader
			vloader, err := config.NewVariableLoader(cfg, basePath)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create variable loader")
				os.Exit(1)
			}

			// Load variables
			_, err = vloader.LoadAllVariables(nil)
			if err != nil {
				log.Error().Err(err).Msg("Failed to load variables")
				os.Exit(1)
			}

			// Load variables first to get processed values
			variables, err := vloader.LoadAllVariables(nil)
			if err != nil {
				handleVariableError(err)
				os.Exit(1)
			}

			// Trace variable
			traces := vloader.TraceVariable(key)
			if len(traces) == 0 {
				log.Error().Str("key", key).Msg("Variable not found in any source")
				os.Exit(1)
			}

			// Get processed value if not showing raw
			var processedValue interface{}
			if !showRaw {
				processedValue, _ = vloader.GetVariable(key, variables)
			}

			// Display trace information
			displayTrace(key, traces, showRaw, processedValue)
		},
	}

	traceCmd.Flags().BoolVar(&showRaw, "raw", false, "Show raw template syntax instead of rendered values")

	return traceCmd
}

// createVariablesSourcesCommand creates the variables sources subcommand
func createVariablesSourcesCommand() *cobra.Command {
	sourcesCmd := &cobra.Command{
		Use:   "sources",
		Short: "Show all variable sources and load order",
		Long: `Show all variable source files that were loaded and their precedence order.
This helps understand the variable loading process and file hierarchy.`,
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

			// Create variable loader
			vloader, err := config.NewVariableLoader(cfg, basePath)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create variable loader")
				os.Exit(1)
			}

			// Load variables
			_, err = vloader.LoadAllVariables(nil)
			if err != nil {
				handleVariableError(err)
				os.Exit(1)
			}

			// Display sources
			sources := vloader.GetVariableSources()
			displaySources(sources)
		},
	}

	return sourcesCmd
}



// Helper functions for display

func displayVariables(variables map[string]interface{}, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(variables)
	case "yaml":
		// Simple YAML-like output
		return displayYAMLVariables(variables, 0)
	case "table":
		return displayTableVariables(variables)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func displayYAMLVariables(variables map[string]interface{}, indent int) error {
	indentStr := strings.Repeat("  ", indent)

	for key, value := range variables {
		switch v := value.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s:\n", indentStr, key)
			if err := displayYAMLVariables(v, indent+1); err != nil {
				return err
			}
		case []interface{}:
			fmt.Printf("%s%s:\n", indentStr, key)
			for i, item := range v {
				fmt.Printf("%s  - %v\n", indentStr, item)
				_ = i
			}
		default:
			fmt.Printf("%s%s: %v\n", indentStr, key, v)
		}
	}

	return nil
}

func displayTableVariables(variables map[string]interface{}) error {
	fmt.Printf("%-30s | %s\n", "Key", "Value")
	fmt.Printf("%s\n", strings.Repeat("-", 50))

	return displayTableVariablesRecursive(variables, "")
}

func displayTableVariablesRecursive(variables map[string]interface{}, prefix string) error {
	for key, value := range variables {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			if err := displayTableVariablesRecursive(v, fullKey); err != nil {
				return err
			}
		default:
			fmt.Printf("%-30s | %v\n", fullKey, v)
		}
	}

	return nil
}

func displayValue(key string, value interface{}, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(map[string]interface{}{key: value})
	case "raw":
		fmt.Printf("%v\n", value)
		return nil
	default: // yaml
		fmt.Printf("%s: %v\n", key, value)
		return nil
	}
}

func displayTrace(key string, traces []*config.VariableSource, showRaw bool, processedValue interface{}) {
	fmt.Printf("Variable: %s\n", key)
	if showRaw {
		fmt.Printf("Sources (raw template values):\n\n")
	} else {
		fmt.Printf("Sources (rendered values):\n\n")
	}

	for i, trace := range traces {
		fmt.Printf("%d. Source: %s\n", i+1, trace.Source)
		if trace.Line > 0 {
			fmt.Printf("   Line: %d\n", trace.Line)
		}
		if showRaw {
			fmt.Printf("   Raw Value: %v\n", trace.RawValue)
		} else {
			// Show processed value if available, fallback to raw value
			if trace.ProcessedValue != nil {
				fmt.Printf("   Value: %v\n", trace.ProcessedValue)
			} else {
				fmt.Printf("   Value: %v\n", trace.RawValue)
			}
		}
		fmt.Println()
	}

	if len(traces) > 1 {
		finalTrace := traces[len(traces)-1]
		if showRaw {
			fmt.Printf("Final raw value: %v (from %s)\n", finalTrace.RawValue, finalTrace.Source)
		} else {
			if finalTrace.ProcessedValue != nil {
				fmt.Printf("Final value: %v (from %s)\n", finalTrace.ProcessedValue, finalTrace.Source)
			} else {
				fmt.Printf("Final value: %v (from %s)\n", finalTrace.RawValue, finalTrace.Source)
			}
		}
	}

	// Show additional context for raw vs processed comparison
	if showRaw && len(traces) > 0 && processedValue != nil {
		fmt.Printf("\nFinal processed result: %v\n", processedValue)
	}
}

func displaySources(sources []*config.VariableSource) {
	if len(sources) == 0 {
		fmt.Println("No variable sources found")
		return
	}

	// Group by source file
	sourceFiles := make(map[string][]*config.VariableSource)
	for _, source := range sources {
		sourceFiles[source.Source] = append(sourceFiles[source.Source], source)
	}

	fmt.Printf("Variable Sources (%d files loaded):\n\n", len(sourceFiles))

	i := 1
	for file, variables := range sourceFiles {
		fmt.Printf("%d. %s (%d variables)\n", i, file, len(variables))

		// Show first few variables as examples
		for j, variable := range variables {
			if j >= 3 {
				fmt.Printf("     ... and %d more\n", len(variables)-3)
				break
			}
			fmt.Printf("     - %s\n", variable.Key)
		}
		fmt.Println()
		i++
	}
}

func findConfigFile() (string, error) {
	return config.FindConfigFile()
}
