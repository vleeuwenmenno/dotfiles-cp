package jobs

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/config"
	"github.com/vleeuwenmenno/dotfiles-cp/pkg/utils"
)

// JobParser handles parsing of jobs from YAML configuration
type JobParser struct {
	orderCounter int
	basePath     string
	importChain  []string
	currentFile  string
}

// NewJobParser creates a new job parser
func NewJobParser(basePath string) *JobParser {
	return &JobParser{
		orderCounter: 0,
		basePath:     basePath,
		importChain:  make([]string, 0),
		currentFile:  "",
	}
}

// ParseJobsConfig parses a raw configuration map into a list of tasks
func (p *JobParser) ParseJobsConfig(rawConfig map[string]interface{}) ([]*config.Task, error) {
	var tasks []*config.Task

	// Get sorted keys to maintain order from YAML
	keys := p.getSortedKeys(rawConfig)

	for _, actionKey := range keys {
		value := rawConfig[actionKey]
		actionJobs, err := p.parseActionJobs(actionKey, value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse action '%s': %w", actionKey, err)
		}
		tasks = append(tasks, actionJobs...)
	}

	return tasks, nil
}

// ParseJobsIndex parses a jobs index file with import support
func (p *JobParser) ParseJobsIndex(indexPath string, variables map[string]interface{}) ([]*config.Task, error) {
	// Add to import chain to prevent circular imports
	if err := p.addToImportChain(indexPath); err != nil {
		return nil, err
	}
	defer p.removeFromImportChain()

	// Set current file for source tracking
	oldFile := p.currentFile
	p.currentFile = indexPath
	defer func() { p.currentFile = oldFile }()

	// Load jobs index
	jobsIndex, err := config.LoadJobsIndex(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load jobs index from %s: %w", indexPath, err)
	}

	var allTasks []*config.Task

	// Normalize and process imports first
	normalizedImports, err := config.NormalizeImports(jobsIndex.Imports)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize imports: %w", err)
	}

	for _, importFile := range normalizedImports {
		importTasks, err := p.processImport(importFile, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to process import %s: %w", importFile.Path, err)
		}
		allTasks = append(allTasks, importTasks...)
	}

	// Process local jobs
	localTasks, err := p.ParseJobsConfig(jobsIndex.Jobs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse local jobs: %w", err)
	}
	allTasks = append(allTasks, localTasks...)

	return allTasks, nil
}

// parseActionJobs converts an action's configuration into tasks
func (p *JobParser) parseActionJobs(actionKey string, value interface{}) ([]*config.Task, error) {
	switch v := value.(type) {
	case string:
		// Single string value: ensure_dir: "path"
		return p.createJobsFromString(actionKey, v), nil

	case []interface{}:
		// Array of items: install: ["git", "vim"] or symlink: [{src: "...", dst: "..."}]
		return p.createJobsFromArray(actionKey, v)

	case map[string]interface{}:
		// Single object: symlink: {src: "...", dst: "..."}
		return p.createJobsFromObject(actionKey, v), nil

	default:
		return nil, fmt.Errorf("unsupported value type for action '%s': %T", actionKey, value)
	}
}

// createJobsFromString creates a task from a string value
func (p *JobParser) createJobsFromString(actionKey, value string) []*config.Task {
	p.orderCounter++
	taskConfig := p.stringToConfig(actionKey, value)
	task := &config.Task{
		ID:     p.generateTaskID(actionKey, taskConfig),
		Action: actionKey,
		Config: taskConfig,
		Source: p.getRelativeSource(),
		Order:  p.orderCounter,
	}
	p.extractCondition(task)
	return []*config.Task{task}
}

// createJobsFromArray creates tasks from an array of values
func (p *JobParser) createJobsFromArray(actionKey string, values []interface{}) ([]*config.Task, error) {
	var tasks []*config.Task

	for i, item := range values {
		p.orderCounter++
		taskConfig, err := p.itemToConfig(actionKey, item)
		if err != nil {
			return nil, fmt.Errorf("failed to parse array item %d: %w", i, err)
		}

		task := &config.Task{
			ID:     p.generateTaskID(actionKey, taskConfig),
			Action: actionKey,
			Config: taskConfig,
			Source: p.getRelativeSource(),
			Order:  p.orderCounter,
		}
		p.extractCondition(task)
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// createJobsFromObject creates a task from an object value
func (p *JobParser) createJobsFromObject(actionKey string, value map[string]interface{}) []*config.Task {
	p.orderCounter++
	task := &config.Task{
		ID:     p.generateTaskID(actionKey, value),
		Action: actionKey,
		Config: value,
		Source: p.getRelativeSource(),
		Order:  p.orderCounter,
	}
	p.extractCondition(task)
	return []*config.Task{task}
}

// stringToConfig converts a string value to appropriate config based on action
// TODO: This should be made generic by having modules register their string conversion logic
func (p *JobParser) stringToConfig(actionKey, value string) map[string]interface{} {
	switch actionKey {
	case "ensure_dir", "ensure_file":
		return map[string]interface{}{"path": value}
	case "install_package", "uninstall_package":
		return map[string]interface{}{"name": value}
	default:
		// Generic fallback - modules should support this
		return map[string]interface{}{"value": value}
	}
}

// itemToConfig converts an array item to config map
func (p *JobParser) itemToConfig(actionKey string, item interface{}) (map[string]interface{}, error) {
	switch v := item.(type) {
	case string:
		return p.stringToConfig(actionKey, v), nil

	case map[string]interface{}:
		return v, nil

	default:
		return nil, fmt.Errorf("unsupported item type: %T", item)
	}
}

// getSortedKeys returns map keys sorted to maintain order
func (p *JobParser) getSortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// generateTaskID creates a descriptive task ID based on action and config
func (p *JobParser) generateTaskID(actionKey string, config map[string]interface{}) string {
	// Generic task ID generation - try to find a meaningful identifier
	if name, exists := config["name"]; exists {
		if nameStr, ok := name.(string); ok {
			return fmt.Sprintf("%s: %s", actionKey, nameStr)
		}
	}

	if path, exists := config["path"]; exists {
		if pathStr, ok := path.(string); ok {
			return fmt.Sprintf("%s: %s", actionKey, pathStr)
		}
	}

	if value, exists := config["value"]; exists {
		if valueStr, ok := value.(string); ok {
			return fmt.Sprintf("%s: %s", actionKey, valueStr)
		}
	}

	if src, exists := config["src"]; exists {
		if dst, dstExists := config["dst"]; dstExists {
			if srcStr, srcOk := src.(string); srcOk {
				if dstStr, dstOk := dst.(string); dstOk {
					return fmt.Sprintf("%s: %s -> %s", actionKey, srcStr, dstStr)
				}
			}
		}
	}

	if packages, exists := config["packages"]; exists {
		if pkgSlice, ok := packages.([]interface{}); ok && len(pkgSlice) > 0 {
			return fmt.Sprintf("%s: %d packages", actionKey, len(pkgSlice))
		}
	}

	// Fallback for any action
	return fmt.Sprintf("%s_%d", actionKey, p.orderCounter)
}

// extractCondition extracts the condition from task config and moves it to the Condition field
func (p *JobParser) extractCondition(task *config.Task) {
	if condition, exists := task.Config["condition"]; exists {
		if conditionStr, ok := condition.(string); ok {
			task.Condition = conditionStr
			// Remove condition from config since it's now in the Condition field
			delete(task.Config, "condition")
		}
	}
}



// LoadJobsFromFileWithConditions loads and parses jobs from a file, filtering by conditions
func LoadJobsFromFileWithConditions(filePath string, variables map[string]interface{}) ([]*config.Task, error) {

	parser := NewJobParser(filepath.Dir(filepath.Dir(filePath))) // Go up one level to get the dotfiles root
	allTasks, err := parser.ParseJobsIndex(filePath, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jobs: %w", err)
	}

	// Filter tasks based on conditions
	var filteredTasks []*config.Task
	for _, task := range allTasks {
		// Check condition
		if task.Condition != "" {
			shouldExecute, err := parser.evaluateCondition(task.Condition, variables)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate condition for task '%s': %w", task.ID, err)
			}
			if !shouldExecute {
				continue
			}
		}

		filteredTasks = append(filteredTasks, task)
	}

	return filteredTasks, nil
}

// evaluateCondition evaluates a condition string against variables
//
// Supported syntax:
//   Simple comparisons: eq .Platform.OS "linux"
//   Boolean operations: and (eq .Platform.OS "linux") (eq .Platform.Distro "Alpine Linux")
//   Negation: not .Platform.IsElevated
//   Complex nesting: and (or (eq .Platform.OS "linux") (eq .Platform.OS "darwin")) .Platform.IsRoot
//
// Available functions:
//   Comparison: eq, ne, gt, lt
//   Boolean: and, or, not
//   String: contains, hasPrefix, hasSuffix, empty
//   Lists: in (check if item is in list)
//
// IMPORTANT: Use parentheses around function calls in boolean operations!
// CORRECT:   and (eq .Platform.OS "linux") (eq .Platform.Distro "Alpine")
// INCORRECT: eq .Platform.OS "linux" and eq .Platform.Distro "Alpine"
func (p *JobParser) evaluateCondition(condition string, variables map[string]interface{}) (bool, error) {
	// Use the same template engine approach as the variable loader
	tmpl := template.New("condition").Option("missingkey=zero").Funcs(template.FuncMap{
		"eq": func(a, b interface{}) bool {
			return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
		},
		"ne": func(a, b interface{}) bool {
			return fmt.Sprintf("%v", a) != fmt.Sprintf("%v", b)
		},
		"and": func(a, b bool) bool {
			return a && b
		},
		"or": func(a, b bool) bool {
			return a || b
		},
		"not": func(a bool) bool {
			return !a
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"hasPrefix": func(s, prefix string) bool {
			return strings.HasPrefix(s, prefix)
		},
		"hasSuffix": func(s, suffix string) bool {
			return strings.HasSuffix(s, suffix)
		},
		"in": func(item interface{}, list []interface{}) bool {
			for _, v := range list {
				if fmt.Sprintf("%v", item) == fmt.Sprintf("%v", v) {
					return true
				}
			}
			return false
		},
		"gt": func(a, b interface{}) bool {
			return fmt.Sprintf("%v", a) > fmt.Sprintf("%v", b)
		},
		"lt": func(a, b interface{}) bool {
			return fmt.Sprintf("%v", a) < fmt.Sprintf("%v", b)
		},
		"empty": func(s interface{}) bool {
			return fmt.Sprintf("%v", s) == ""
		},
	})

	// Wrap condition in template syntax
	conditionTemplate := "{{" + condition + "}}"
	tmpl, err := tmpl.Parse(conditionTemplate)
	if err != nil {
		// Try to provide helpful suggestions for common syntax errors
		if syntaxErr := p.validateAndSuggestConditionFix(condition); syntaxErr != nil {
			return false, fmt.Errorf("condition syntax error: %w\n\nOriginal error: %v", syntaxErr, err)
		}
		return false, fmt.Errorf("failed to parse condition '%s': %w\n\nCondition syntax help:\n- Use parentheses around function calls: and (eq .Platform.OS \"linux\") (eq .Platform.Distro \"Alpine\")\n- Available functions: eq, ne, and, or, not", condition, err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, variables); err != nil {
		return false, fmt.Errorf("failed to evaluate condition '%s': %w\n\nCondition syntax help:\n- Use parentheses around function calls: and (eq .Platform.OS \"linux\") (eq .Platform.Distro \"Alpine\")\n- Available functions: eq, ne, and, or, not", condition, err)
	}

	resultStr := strings.TrimSpace(result.String())
	return resultStr == "true", nil
}

// validateAndSuggestConditionFix checks for common condition syntax errors and suggests fixes
func (p *JobParser) validateAndSuggestConditionFix(condition string) error {
	// Check for common mistake: missing parentheses around function calls in boolean operations
	if (strings.Contains(condition, " and ") || strings.Contains(condition, " or ")) &&
		!strings.Contains(condition, "(") {
		return fmt.Errorf("missing parentheses around function calls. Try: and (eq .Platform.OS \"linux\") (eq .Platform.Distro \"Alpine\")")
	}

	// Check for bare function calls without proper syntax
	if strings.Contains(condition, "eq ") && strings.Count(condition, "\"") < 2 && !strings.Contains(condition, "(") {
		return fmt.Errorf("eq function requires two arguments in quotes. Try: eq .Platform.OS \"linux\"")
	}

	return nil
}

// processImport processes a single import file with conditions
func (p *JobParser) processImport(importFile config.ImportFile, variables map[string]interface{}) ([]*config.Task, error) {
	// Process import path template
	importPath, err := p.processTemplate(importFile.Path, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to process import path template: %w", err)
	}

	// Check if the processed path contains unresolved template placeholders
	if strings.Contains(importPath, "<no value>") || strings.Contains(importPath, "{{") {
		// Skip imports with unresolved variables
		return []*config.Task{}, nil
	}

	// Check condition if specified
	if importFile.Condition != "" {
		shouldImport, err := p.evaluateCondition(importFile.Condition, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate import condition: %w", err)
		}
		if !shouldImport {
			return []*config.Task{}, nil // Skip this import
		}
	}

	// Resolve relative path
	var fullPath string
	if filepath.IsAbs(importPath) {
		fullPath = importPath
	} else {
		fullPath = filepath.Join(p.basePath, "jobs", importPath)
	}

	// Check if file exists
	if !utils.FileExists(fullPath) {
		return nil, fmt.Errorf("import file does not exist: %s", fullPath)
	}

	// Parse imported jobs file
	importedTasks, err := p.ParseJobsIndex(fullPath, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to parse imported jobs file %s: %w", fullPath, err)
	}

	return importedTasks, nil
}

// processTemplate processes a template string with variables
func (p *JobParser) processTemplate(templateStr string, variables map[string]interface{}) (string, error) {
	tmpl := template.New("import").Option("missingkey=zero").Funcs(template.FuncMap{
		"pathJoin":  func(paths ...string) string { return filepath.Join(paths...) },
		"pathSep":   func() string { return string(filepath.Separator) },
		"pathClean": func(path string) string { return filepath.Clean(path) },
		"eq": func(a, b interface{}) bool {
			return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
		},
		"ne": func(a, b interface{}) bool {
			return fmt.Sprintf("%v", a) != fmt.Sprintf("%v", b)
		},
	})

	tmpl, err := tmpl.Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, variables); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Ensure OS-specific path separators
	renderedResult := result.String()
	return filepath.FromSlash(renderedResult), nil
}

// addToImportChain adds a file to the import chain to detect circular imports
func (p *JobParser) addToImportChain(filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check for circular import
	for _, existing := range p.importChain {
		if existing == absPath {
			return fmt.Errorf("circular import detected: %s", absPath)
		}
	}

	p.importChain = append(p.importChain, absPath)
	return nil
}

// removeFromImportChain removes the last file from the import chain
func (p *JobParser) removeFromImportChain() {
	if len(p.importChain) > 0 {
		p.importChain = p.importChain[:len(p.importChain)-1]
	}
}

// getRelativeSource returns the relative path of the current source file
func (p *JobParser) getRelativeSource() string {
	if p.currentFile == "" {
		return ""
	}

	// Try to make it relative to the base path
	if relPath, err := filepath.Rel(p.basePath, p.currentFile); err == nil {
		return relPath
	}

	// If we can't make it relative, just return the filename
	return filepath.Base(p.currentFile)
}
