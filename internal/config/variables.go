package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/platform"
	"github.com/vleeuwenmenno/dotfiles-cp/pkg/utils"
	"gopkg.in/yaml.v3"
)

// VariableConflictError represents a variable conflict with detailed information
type VariableConflictError struct {
	Variable      string
	ExistingValue interface{}
	NewValue      interface{}
	ExistingSource string
	NewSource     string
	BasePath      string
}

// Error implements the error interface
func (e *VariableConflictError) Error() string {
	return fmt.Sprintf("variable conflict: '%s' has different values in %s and %s",
		e.Variable, e.getRelativePath(e.ExistingSource), e.getRelativePath(e.NewSource))
}

// PrettyPrint returns a formatted, user-friendly error message
func (e *VariableConflictError) PrettyPrint() string {
	var msg strings.Builder

	msg.WriteString("\n")
	msg.WriteString("🔥 VARIABLE CONFLICT DETECTED\n")
	msg.WriteString(strings.Repeat("=", 50) + "\n\n")

	msg.WriteString(fmt.Sprintf("Variable: %s\n\n", e.Variable))

	msg.WriteString("Conflicting definitions found:\n\n")

	// Show first definition
	msg.WriteString(fmt.Sprintf("📁 File: %s\n", e.getRelativePath(e.ExistingSource)))
	msg.WriteString(fmt.Sprintf("   Value: %v\n\n", e.ExistingValue))

	// Show second definition
	msg.WriteString(fmt.Sprintf("📁 File: %s\n", e.getRelativePath(e.NewSource)))
	msg.WriteString(fmt.Sprintf("   Value: %v\n\n", e.NewValue))

	msg.WriteString("💡 To fix this conflict:\n")
	msg.WriteString("   1. Remove the duplicate definition from one of the files, OR\n")
	msg.WriteString("   2. Use different variable names for different purposes, OR\n")
	msg.WriteString("   3. Move one definition to a more specific scope\n\n")

	msg.WriteString("Note: Variables must have the same value when defined in multiple files\n")

	return msg.String()
}

// getRelativePath converts absolute paths to relative paths for better readability
func (e *VariableConflictError) getRelativePath(source string) string {
	if e.BasePath != "" {
		if relPath, err := filepath.Rel(e.BasePath, source); err == nil {
			return relPath
		}
	}
	return filepath.Base(source)
}

// IsVariableConflictError checks if an error is a variable conflict error
func IsVariableConflictError(err error) (*VariableConflictError, bool) {
	var conflictErr *VariableConflictError
	if errors.As(err, &conflictErr) {
		return conflictErr, true
	}
	return nil, false
}

// VariableLoader handles loading and merging variables from multiple sources
type VariableLoader struct {
	config   *Config
	context  *ImportContext
	sources  []*VariableSource
	platform *platform.PlatformInfo
	basePath string
}

// VariableLoadOptions contains options for variable loading
type VariableLoadOptions struct {
	Platform    string            // Override platform detection
	Shell       string            // Override shell detection
	Environment map[string]string // Additional environment variables
	Hostname    string            // Override hostname
}

// NewVariableLoader creates a new variable loader
func NewVariableLoader(config *Config, basePath string) (*VariableLoader, error) {
	// Get platform information
	platformInfo, err := platform.GetPlatformInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get platform info: %w", err)
	}

	context := NewImportContext(config, basePath)

	return &VariableLoader{
		config:   config,
		context:  context,
		sources:  make([]*VariableSource, 0),
		platform: platformInfo,
		basePath: basePath,
	}, nil
}

// LoadAllVariables loads variables from all sources with proper precedence
func (vl *VariableLoader) LoadAllVariables(opts *VariableLoadOptions) (map[string]interface{}, error) {
	// Reset sources for fresh load
	vl.sources = make([]*VariableSource, 0)
	vl.context.Variables = make(map[string]interface{})

	// Create template context for conditional imports
	templateContext := vl.createTemplateContext(opts)

	// Load variables index
	variablesIndexPath := vl.config.GetVariablesIndexPath(vl.basePath)
	if !utils.FileExists(variablesIndexPath) {
		return vl.context.Variables, nil // No variables to load
	}

	// Process variables index
	if err := vl.processVariablesIndex(variablesIndexPath, templateContext); err != nil {
		return nil, fmt.Errorf("failed to process variables index: %w", err)
	}

	// Process all variables through template engine
	processedVariables, err := vl.processVariableTemplates(vl.context.Variables, templateContext)
	if err != nil {
		return nil, fmt.Errorf("failed to process variable templates: %w", err)
	}

	// Add platform information and other context to final variables
	// This ensures Platform, Env, etc. are available in job templates
	for key, value := range templateContext {
		if _, exists := processedVariables[key]; !exists {
			processedVariables[key] = value
		}
	}

	return processedVariables, nil
}

// GetVariableSources returns all variable sources for debugging
func (vl *VariableLoader) GetVariableSources() []*VariableSource {
	// Sort sources by precedence for display
	sources := make([]*VariableSource, len(vl.sources))
	copy(sources, vl.sources)

	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Source < sources[j].Source
	})

	return sources
}

// TraceVariable finds where a specific variable was defined
func (vl *VariableLoader) TraceVariable(key string) []*VariableSource {
	var traces []*VariableSource

	// Handle dot notation (e.g., "user.name")
	if strings.Contains(key, ".") {
		// Split the key to find the root (e.g., "user" from "user.name")
		keyParts := strings.Split(key, ".")
		rootKey := keyParts[0]

		// Find all sources that define the root key
		for _, source := range vl.sources {
			if source.Key == rootKey {
				// Create a new source entry that shows the specific nested value
				nestedSource := &VariableSource{
					Key:            key, // Use the full dot notation key
					RawValue:       vl.extractNestedValue(source.RawValue, keyParts[1:]),
					ProcessedValue: vl.extractNestedValue(source.ProcessedValue, keyParts[1:]),
					Source:         source.Source,
					Line:           source.Line,
				}
				traces = append(traces, nestedSource)
			}
		}
	} else {
		// For simple keys, use exact match
		for _, source := range vl.sources {
			if source.Key == key {
				traces = append(traces, source)
			}
		}
	}

	return traces
}

// processVariablesIndex processes the main variables index file
func (vl *VariableLoader) processVariablesIndex(indexPath string, templateContext map[string]interface{}) error {
	// Add to import chain
	if err := vl.context.AddToChain(indexPath); err != nil {
		return err
	}
	defer vl.context.RemoveFromChain()

	// Load index file
	index, err := LoadVariableIndex(indexPath)
	if err != nil {
		return fmt.Errorf("failed to load variables index from %s: %w", indexPath, err)
	}

	// Normalize and process imports first
	normalizedImports, err := NormalizeImports(index.Imports)
	if err != nil {
		return fmt.Errorf("failed to normalize imports: %w", err)
	}

	for _, importFile := range normalizedImports {
		if err := vl.processImport(importFile, templateContext); err != nil {
			return fmt.Errorf("failed to process import %s: %w", importFile.Path, err)
		}
	}

	// Process variables in index file
	if err := vl.addVariables(index.Variables, indexPath, 0); err != nil {
		return fmt.Errorf("failed to add variables from %s: %w", indexPath, err)
	}

	return nil
}

// processImport processes a single import file with conditions
func (vl *VariableLoader) processImport(importFile ImportFile, templateContext map[string]interface{}) error {
	// Process conditional imports
	importPath, err := vl.processTemplate(importFile.Path, templateContext)
	if err != nil {
		return fmt.Errorf("failed to process import path template: %w", err)
	}

	// Check if the processed path contains template placeholders that couldn't be resolved
	if strings.Contains(importPath, "<no value>") || strings.Contains(importPath, "{{") {
		// Skip imports with unresolved variables
		return nil
	}

	// Check condition if specified
	if importFile.Condition != "" {
		shouldImport, err := vl.evaluateCondition(importFile.Condition, templateContext)
		if err != nil {
			return fmt.Errorf("failed to evaluate import condition: %w", err)
		}
		if !shouldImport {
			return nil // Skip this import
		}
	}

	// Resolve relative path
	fullPath := filepath.Join(vl.config.GetVariablesPath(vl.basePath), importPath)

	// Check for circular imports
	if err := vl.context.AddToChain(fullPath); err != nil {
		return err
	}
	defer vl.context.RemoveFromChain()

	// Load imported file
	if err := vl.loadVariableFile(fullPath); err != nil {
		return fmt.Errorf("failed to load imported file %s: %w", fullPath, err)
	}

	// Add file-specific variables
	if len(importFile.Variables) > 0 {
		if err := vl.addVariables(importFile.Variables, fullPath, 0); err != nil {
			return fmt.Errorf("failed to add import variables from %s: %w", fullPath, err)
		}
	}

	return nil
}

// loadVariableFile loads variables from a YAML file
func (vl *VariableLoader) loadVariableFile(filePath string) error {
	if !utils.FileExists(filePath) {
		return fmt.Errorf("variable file does not exist: %s", filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read variable file: %w", err)
	}

	var variables map[string]interface{}
	if err := yaml.Unmarshal(data, &variables); err != nil {
		return fmt.Errorf("failed to unmarshal variables: %w", err)
	}

	// Add variables with source tracking
	return vl.addVariables(variables, filePath, 0)
}

// addVariables adds variables to the context with source tracking and deep merging
func (vl *VariableLoader) addVariables(variables map[string]interface{}, source string, line int) error {
	for key, value := range variables {
		// Track variable source (store raw value, will update with processed later)
		vl.sources = append(vl.sources, &VariableSource{
			Key:            key,
			RawValue:       value,
			ProcessedValue: nil, // Will be updated after processing
			Source:         source,
			Line:           line,
		})

		// Deep merge or add to context
		if existing, exists := vl.context.Variables[key]; exists {
			merged, err := vl.deepMergeVariables(key, existing, value, source)
			if err != nil {
				return fmt.Errorf("failed to merge variable '%s' from %s: %w", key, source, err)
			}
			vl.context.Variables[key] = merged
		} else {
			vl.context.Variables[key] = value
		}
	}

	return nil
}

// deepMergeVariables performs deep merging of variable values with conflict detection
func (vl *VariableLoader) deepMergeVariables(key string, existing, new interface{}, newSource string) (interface{}, error) {
	// If both are maps, merge them recursively
	existingMap, existingIsMap := existing.(map[string]interface{})
	newMap, newIsMap := new.(map[string]interface{})

	if existingIsMap && newIsMap {
		// Deep merge maps
		result := make(map[string]interface{})

		// Copy existing values
		for k, v := range existingMap {
			result[k] = v
		}

		// Merge new values
		for k, v := range newMap {
			if existingValue, exists := result[k]; exists {
				// Check for conflicts (same key, different non-map values)
				if !vl.isMapValue(existingValue) && !vl.isMapValue(v) && !vl.valuesEqual(existingValue, v) {
					// Find source of existing value
					existingSource := vl.findVariableSource(key + "." + k)
					return nil, &VariableConflictError{
						Variable:       key + "." + k,
						ExistingValue:  existingValue,
						NewValue:       v,
						ExistingSource: existingSource,
						NewSource:      newSource,
						BasePath:       vl.basePath,
					}
				}

				// Recursively merge if both are maps
				if vl.isMapValue(existingValue) && vl.isMapValue(v) {
					merged, err := vl.deepMergeVariables(key+"."+k, existingValue, v, newSource)
					if err != nil {
						return nil, err
					}
					result[k] = merged
				} else {
					// Non-map values: use the new value (precedence rule)
					result[k] = v
				}
			} else {
				result[k] = v
			}
		}

		return result, nil
	}

	// If not both maps, check for conflict
	if !vl.valuesEqual(existing, new) {
		existingSource := vl.findVariableSource(key)
		return nil, &VariableConflictError{
			Variable:       key,
			ExistingValue:  existing,
			NewValue:       new,
			ExistingSource: existingSource,
			NewSource:      newSource,
			BasePath:       vl.basePath,
		}
	}

	// Same values, return the new one (precedence)
	return new, nil
}

// isMapValue checks if a value is a map
func (vl *VariableLoader) isMapValue(value interface{}) bool {
	_, isMap := value.(map[string]interface{})
	return isMap
}

// valuesEqual checks if two values are equal
func (vl *VariableLoader) valuesEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

// findVariableSource finds the source file for a variable
func (vl *VariableLoader) findVariableSource(key string) string {
	for _, source := range vl.sources {
		if source.Key == key || strings.HasPrefix(key, source.Key+".") {
			return source.Source
		}
	}
	return "unknown"
}

// createTemplateContext creates context for template processing
func (vl *VariableLoader) createTemplateContext(opts *VariableLoadOptions) map[string]interface{} {
	context := make(map[string]interface{})

	// Platform information
	platformInfo := map[string]interface{}{
		"OS":   vl.platform.OS,
		"Arch": vl.platform.Arch,
	}

	// Override with options if provided
	if opts != nil {
		if opts.Platform != "" {
			platformInfo["OS"] = opts.Platform
		}
		if opts.Shell != "" {
			platformInfo["Shell"] = opts.Shell
		} else {
			platformInfo["Shell"] = vl.platform.Shell
		}
		if opts.Hostname != "" {
			platformInfo["Hostname"] = opts.Hostname
		} else {
			hostname, err := os.Hostname()
			if err == nil {
				platformInfo["Hostname"] = hostname
			}
		}
	} else {
		platformInfo["Shell"] = vl.platform.Shell
		hostname, err := os.Hostname()
		if err == nil {
			platformInfo["Hostname"] = hostname
		}
	}

	context["Platform"] = platformInfo

	// Environment variables
	if opts != nil && opts.Environment != nil {
		context["Env"] = opts.Environment
	} else {
		env := make(map[string]string)
		for _, envVar := range os.Environ() {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}
		context["Env"] = env
	}

	// User information
	user := make(map[string]interface{})
	if homeDir, err := os.UserHomeDir(); err == nil {
		user["Home"] = homeDir
	}
	context["User"] = user

	return context
}

// processTemplate processes a template string with the given context
func (vl *VariableLoader) processTemplate(templateStr string, context map[string]interface{}) (string, error) {
	// Create template with custom functions
	tmpl := template.New("import").Option("missingkey=zero").Funcs(template.FuncMap{
		"pathJoin":  func(paths ...string) string { return filepath.Join(paths...) },
		"pathSep":   func() string { return string(filepath.Separator) },
		"pathClean": func(path string) string { return filepath.Clean(path) },
	})

	tmpl, err := tmpl.Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, context); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Ensure OS-specific path separators
	renderedResult := result.String()
	return filepath.FromSlash(renderedResult), nil
}

// evaluateCondition evaluates a condition string
func (vl *VariableLoader) evaluateCondition(condition string, context map[string]interface{}) (bool, error) {
	// Simple condition evaluation for now
	// This could be extended to support more complex expressions

	tmpl := template.New("condition").Option("missingkey=zero").Funcs(template.FuncMap{
		"pathJoin":  func(paths ...string) string { return filepath.Join(paths...) },
		"pathSep":   func() string { return string(filepath.Separator) },
		"pathClean": func(path string) string { return filepath.Clean(path) },
	})

	tmpl, err := tmpl.Parse("{{" + condition + "}}")
	if err != nil {
		return false, fmt.Errorf("failed to parse condition template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, context); err != nil {
		return false, fmt.Errorf("failed to execute condition template: %w", err)
	}

	// Convert result to boolean
	resultStr := strings.TrimSpace(result.String())
	return resultStr == "true" || resultStr == "1", nil
}

// GetVariable gets a specific variable by key (supports dot notation)
func (vl *VariableLoader) GetVariable(key string, variables map[string]interface{}) (interface{}, bool) {
	// Support dot notation like "user.name"
	keys := strings.Split(key, ".")
	current := variables

	for i, k := range keys {
		if i == len(keys)-1 {
			// Last key - return the value
			value, exists := current[k]
			return value, exists
		}

		// Intermediate key - must be a map
		if next, exists := current[k]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return nil, false
			}
		} else {
			return nil, false
		}
	}

	return nil, false
}

// SetVariable sets a variable with dot notation support
func (vl *VariableLoader) SetVariable(key string, value interface{}, variables map[string]interface{}) {
	keys := strings.Split(key, ".")
	current := variables

	for i, k := range keys {
		if i == len(keys)-1 {
			// Last key - set the value
			current[k] = value
			return
		}

		// Intermediate key - ensure it's a map
		if next, exists := current[k]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				// Replace non-map with map
				newMap := make(map[string]interface{})
				current[k] = newMap
				current = newMap
			}
		} else {
			// Create new map
			newMap := make(map[string]interface{})
			current[k] = newMap
			current = newMap
		}
	}
}

// ValidateVariables validates all loaded variables
func (vl *VariableLoader) ValidateVariables(variables map[string]interface{}) error {
	// Check for required variables (could be configurable)
	required := []string{
		// Add any required variables here
	}

	for _, req := range required {
		if _, exists := vl.GetVariable(req, variables); !exists {
			return fmt.Errorf("required variable missing: %s", req)
		}
	}

	return nil
}

// processVariableTemplates processes all variables through the template engine
func (vl *VariableLoader) processVariableTemplates(variables map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error) {
	// Create a copy of the context and add the current variables to it
	// This allows variables to reference other variables
	enrichedContext := make(map[string]interface{})
	for k, v := range context {
		enrichedContext[k] = v
	}

	// Add current variables to context for cross-referencing
	for k, v := range variables {
		enrichedContext[k] = v
	}

	processedVars, err := vl.processVariablesRecursive(variables, enrichedContext)
	if err != nil {
		return nil, err
	}

	// Update sources with processed values
	vl.updateSourcesWithProcessedValues(processedVars)

	return processedVars, nil
}

// processVariablesRecursive recursively processes variables through templates
func (vl *VariableLoader) processVariablesRecursive(variables map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range variables {
		switch v := value.(type) {
		case string:
			// Process string through template engine
			processed, err := vl.processTemplate(v, context)
			if err != nil {
				return nil, fmt.Errorf("failed to process template for variable %s: %w", key, err)
			}
			result[key] = processed
		case map[string]interface{}:
			// Recursively process nested maps
			nestedResult, err := vl.processVariablesRecursive(v, context)
			if err != nil {
				return nil, fmt.Errorf("failed to process nested variables for %s: %w", key, err)
			}
			result[key] = nestedResult
		case []interface{}:
			// Process arrays
			processedArray := make([]interface{}, len(v))
			for i, item := range v {
				switch itemVal := item.(type) {
				case string:
					processed, err := vl.processTemplate(itemVal, context)
					if err != nil {
						return nil, fmt.Errorf("failed to process template for array item in %s[%d]: %w", key, i, err)
					}
					processedArray[i] = processed
				case map[string]interface{}:
					nestedResult, err := vl.processVariablesRecursive(itemVal, context)
					if err != nil {
						return nil, fmt.Errorf("failed to process nested array item in %s[%d]: %w", key, i, err)
					}
					processedArray[i] = nestedResult
				default:
					processedArray[i] = item
				}
			}
			result[key] = processedArray
		default:
			// For non-string types, keep as-is
			result[key] = value
		}
	}

	return result, nil
}

// extractNestedValue extracts a nested value from a variable using the remaining key parts
func (vl *VariableLoader) extractNestedValue(value interface{}, keyParts []string) interface{} {
	if len(keyParts) == 0 {
		return value
	}

	switch v := value.(type) {
	case map[string]interface{}:
		if nextValue, exists := v[keyParts[0]]; exists {
			if len(keyParts) == 1 {
				return nextValue
			}
			return vl.extractNestedValue(nextValue, keyParts[1:])
		}
	}

	return "<not found>"
}

// updateSourcesWithProcessedValues updates the variable sources with their processed values
func (vl *VariableLoader) updateSourcesWithProcessedValues(processedVariables map[string]interface{}) {
	for _, source := range vl.sources {
		if processedValue, exists := vl.GetVariable(source.Key, processedVariables); exists {
			source.ProcessedValue = processedValue
		}
	}
}
