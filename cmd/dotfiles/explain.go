package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/logger"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules/files"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules/packages"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules/symlinks"

	"github.com/spf13/cobra"
)

func createExplainCommand() *cobra.Command {
	var format string
	var listAll bool

	explainCmd := &cobra.Command{
		Use:   "explain [action|module]",
		Short: "Explain available actions and their parameters",
		Long: `Explain available actions and their parameters.

Examples:
  dotfiles explain                    # List all available actions
  dotfiles explain ensure_file        # Explain the ensure_file action
  dotfiles explain files              # Explain all actions in the files module
  dotfiles explain --format json     # Output in JSON format`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()

			// Create module registry
			registry := modules.NewModuleRegistry()
			if err := registry.Register(files.New()); err != nil {
				log.Error().Err(err).Msg("Failed to register files module")
				os.Exit(1)
			}
			if err := registry.Register(packages.New()); err != nil {
				log.Error().Err(err).Msg("Failed to register packages module")
				os.Exit(1)
			}
			if err := registry.Register(symlinks.New()); err != nil {
				log.Error().Err(err).Msg("Failed to register symlinks module")
				os.Exit(1)
			}

			if len(args) == 0 || listAll {
				// List all actions from all modules
				allActions := registry.ListAllActions()
				if format == "json" {
					outputJSON(allActions)
				} else {
					outputText(allActions, "")
				}
				return
			}

			target := args[0]

			// Check if it's an action
			if actionDoc, err := registry.ExplainAction(target); err == nil {
				if format == "json" {
					outputJSON(map[string][]*modules.ActionDocumentation{
						"action": {actionDoc},
					})
				} else {
					outputActionText(actionDoc)
				}
				return
			}

			// Check if it's a module
			if moduleActions, err := registry.ExplainModule(target); err == nil {
				if format == "json" {
					outputJSON(map[string][]*modules.ActionDocumentation{
						target: moduleActions,
					})
				} else {
					outputText(map[string][]*modules.ActionDocumentation{
						target: moduleActions,
					}, target)
				}
				return
			}

			// Not found
			log.Error().Str("target", target).Msg("Action or module not found")
			fmt.Printf("\nAvailable actions: %s\n", strings.Join(registry.GetSupportedActions(), ", "))
			fmt.Printf("Available modules: %s\n", strings.Join(getModuleNames(registry), ", "))
			os.Exit(1)
		},
	}

	explainCmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json)")
	explainCmd.Flags().BoolVarP(&listAll, "all", "a", false, "List all available actions and modules")

	return explainCmd
}

func outputJSON(data map[string][]*modules.ActionDocumentation) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		log := logger.Get()
		log.Error().Err(err).Msg("Failed to output JSON")
		os.Exit(1)
	}
}

func outputText(data map[string][]*modules.ActionDocumentation, specificModule string) {
	if specificModule != "" {
		fmt.Printf("Module: %s\n", specificModule)
		fmt.Println(strings.Repeat("=", len(specificModule)+8))
		fmt.Println()
	} else {
		fmt.Println("Available Actions")
		fmt.Println("=================")
		fmt.Println()
	}

	// Sort modules for consistent output
	moduleNames := make([]string, 0, len(data))
	for moduleName := range data {
		moduleNames = append(moduleNames, moduleName)
	}
	sort.Strings(moduleNames)

	for i, moduleName := range moduleNames {
		if i > 0 && specificModule == "" {
			fmt.Println()
		}

		if specificModule == "" {
			fmt.Printf("Module: %s\n", moduleName)
			fmt.Println(strings.Repeat("-", len(moduleName)+8))
		}

		actions := data[moduleName]
		for j, action := range actions {
			if j > 0 {
				fmt.Println()
			}
			outputActionText(action)
		}

		if specificModule == "" {
			fmt.Println()
		}
	}
}

func outputActionText(action *modules.ActionDocumentation) {
	fmt.Printf("Action: %s\n", action.Action)
	fmt.Printf("Description: %s\n", action.Description)
	fmt.Println()

	if len(action.Parameters) > 0 {
		fmt.Println("Parameters:")
		for _, param := range action.Parameters {
			required := ""
			if param.Required {
				required = " (required)"
			}

			defaultValue := ""
			if param.Default != "" {
				defaultValue = fmt.Sprintf(" [default: %s]", param.Default)
			}

			fmt.Printf("  %s (%s)%s%s\n", param.Name, param.Type, required, defaultValue)

			// Wrap description text
			description := wrapText(param.Description, 4, 76)
			fmt.Printf("    %s\n", description)
		}
		fmt.Println()
	}

	if len(action.Examples) > 0 {
		fmt.Println("Examples:")
		for i, example := range action.Examples {
			fmt.Printf("  %d. %s\n", i+1, example.Description)

			// Format config as proper YAML with action key
			fmt.Printf("     %s:\n", action.Action)

			// Sort config keys for consistent output
			keys := make([]string, 0, len(example.Config))
			for key := range example.Config {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			fmt.Printf("       - ")
			for j, key := range keys {
				value := example.Config[key]
				if j > 0 {
					fmt.Printf("         ")
				}
				switch v := value.(type) {
				case string:
					// Handle multiline strings
					if strings.Contains(v, "\n") {
						fmt.Printf("%s: |\n", key)
						lines := strings.Split(v, "\n")
						for _, line := range lines {
							fmt.Printf("           %s\n", line)
						}
					} else {
						fmt.Printf("%s: \"%s\"\n", key, v)
					}
				case bool:
					fmt.Printf("%s: %t\n", key, v)
				default:
					fmt.Printf("%s: %v\n", key, v)
				}
			}

			if i < len(action.Examples)-1 {
				fmt.Println()
			}
		}
	}
}

func getModuleNames(registry *modules.ModuleRegistry) []string {
	allModules := registry.GetAllModules()
	names := make([]string, 0, len(allModules))
	for name := range allModules {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func wrapText(text string, indent, maxWidth int) string {
	if len(text) <= maxWidth-indent {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var currentLine strings.Builder
	indentStr := strings.Repeat(" ", indent)

	for i, word := range words {
		// Check if adding this word would exceed the line length
		if currentLine.Len() > 0 && currentLine.Len()+1+len(word) > maxWidth-indent {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
		}

		if currentLine.Len() > 0 {
			currentLine.WriteString(" ")
		}
		currentLine.WriteString(word)

		// If this is the last word, add the current line
		if i == len(words)-1 {
			lines = append(lines, currentLine.String())
		}
	}

	// Join lines with newlines and indentation
	result := strings.Join(lines, "\n"+indentStr)
	return result
}
