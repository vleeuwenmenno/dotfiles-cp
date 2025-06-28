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
func (p *JobParser) stringToConfig(actionKey, value string) map[string]interface{} {
	switch actionKey {
	case "ensure_dir":
		return map[string]interface{}{"path": value}
	case "ensure_file":
		return map[string]interface{}{"path": value}
	case "install":
		return map[string]interface{}{"packages": []string{value}}
	case "remove":
		return map[string]interface{}{"packages": []string{value}}
	default:
		// Generic fallback
		return map[string]interface{}{"value": value}
	}
}

// itemToConfig converts an array item to config map
func (p *JobParser) itemToConfig(actionKey string, item interface{}) (map[string]interface{}, error) {
	switch v := item.(type) {
	case string:
		return p.stringToConfig(actionKey, v), nil

	case []interface{}:
		// Handle arrays like install: [["git", "vim"], ["curl"]]
		if actionKey == "install" || actionKey == "remove" {
			packages := make([]string, len(v))
			for i, pkg := range v {
				if pkgStr, ok := pkg.(string); ok {
					packages[i] = pkgStr
				} else {
					return nil, fmt.Errorf("expected string package name, got %T", pkg)
				}
			}
			return map[string]interface{}{"packages": packages}, nil
		}
		return nil, fmt.Errorf("arrays not supported for action '%s'", actionKey)

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
	switch actionKey {
	case "ensure_file":
		if path, exists := config["path"]; exists {
			if pathStr, ok := path.(string); ok {
				return fmt.Sprintf("ensure_file: %s", pathStr)
			}
		}
		return fmt.Sprintf("ensure_file_%d", p.orderCounter)

	case "ensure_dir":
		if path, exists := config["path"]; exists {
			if pathStr, ok := path.(string); ok {
				return fmt.Sprintf("ensure_dir: %s", pathStr)
			}
		}
		return fmt.Sprintf("ensure_dir_%d", p.orderCounter)

	case "symlink":
		src, srcExists := config["src"]
		dst, dstExists := config["dst"]
		if srcExists && dstExists {
			if srcStr, srcOk := src.(string); srcOk {
				if dstStr, dstOk := dst.(string); dstOk {
					return fmt.Sprintf("symlink: %s -> %s", srcStr, dstStr)
				}
			}
		}
		return fmt.Sprintf("symlink_%d", p.orderCounter)

	case "copy":
		src, srcExists := config["src"]
		dst, dstExists := config["dst"]
		if srcExists && dstExists {
			if srcStr, srcOk := src.(string); srcOk {
				if dstStr, dstOk := dst.(string); dstOk {
					return fmt.Sprintf("copy: %s -> %s", srcStr, dstStr)
				}
			}
		}
		return fmt.Sprintf("copy_%d", p.orderCounter)

	case "install":
		if packages, exists := config["packages"]; exists {
			if pkgSlice, ok := packages.([]string); ok && len(pkgSlice) > 0 {
				if len(pkgSlice) == 1 {
					return fmt.Sprintf("install: %s", pkgSlice[0])
				}
				return fmt.Sprintf("install: %d packages", len(pkgSlice))
			}
		}
		return fmt.Sprintf("install_%d", p.orderCounter)

	case "remove":
		if packages, exists := config["packages"]; exists {
			if pkgSlice, ok := packages.([]string); ok && len(pkgSlice) > 0 {
				if len(pkgSlice) == 1 {
					return fmt.Sprintf("remove: %s", pkgSlice[0])
				}
				return fmt.Sprintf("remove: %d packages", len(pkgSlice))
			}
		}
		return fmt.Sprintf("remove_%d", p.orderCounter)

	default:
		// Fallback for unknown actions
		return fmt.Sprintf("%s_%d", actionKey, p.orderCounter)
	}
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

// ValidateTask validates a task configuration
func ValidateTask(task *config.Task) error {
	if task.Action == "" {
		return fmt.Errorf("task action is required")
	}

	if task.Config == nil {
		return fmt.Errorf("task config is required")
	}

	// Action-specific validation
	switch task.Action {
	case "symlink":
		return validateSymlinkTask(task.Config)
	case "template":
		return validateTemplateTask(task.Config)
	case "ensure_dir":
		return validateEnsureDirTask(task.Config)
	case "install", "remove":
		return validatePackageTask(task.Config)
	}

	return nil
}

// validateSymlinkTask validates symlink task configuration
func validateSymlinkTask(config map[string]interface{}) error {
	if _, exists := config["src"]; !exists {
		return fmt.Errorf("symlink task requires 'src' field")
	}
	if _, exists := config["dst"]; !exists {
		return fmt.Errorf("symlink task requires 'dst' field")
	}
	return nil
}

// validateTemplateTask validates template task configuration
func validateTemplateTask(config map[string]interface{}) error {
	if _, exists := config["src"]; !exists {
		return fmt.Errorf("template task requires 'src' field")
	}
	if _, exists := config["dst"]; !exists {
		return fmt.Errorf("template task requires 'dst' field")
	}
	return nil
}

// validateEnsureDirTask validates ensure_dir task configuration
func validateEnsureDirTask(config map[string]interface{}) error {
	if _, exists := config["path"]; !exists {
		return fmt.Errorf("ensure_dir task requires 'path' field")
	}
	return nil
}

// validatePackageTask validates package task configuration
func validatePackageTask(config map[string]interface{}) error {
	if _, exists := config["packages"]; !exists {
		return fmt.Errorf("package task requires 'packages' field")
	}
	return nil
}

// LoadJobsFromFile loads and parses jobs from a file
func LoadJobsFromFile(filePath string) ([]*config.Task, error) {
	jobsIndex, err := config.LoadJobsIndex(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load jobs index: %w", err)
	}

	parser := NewJobParser(filepath.Dir(filepath.Dir(filePath)))
	tasks, err := parser.ParseJobsConfig(jobsIndex.Jobs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jobs: %w", err)
	}

	// Validate all tasks
	for _, task := range tasks {
		if err := ValidateTask(task); err != nil {
			return nil, fmt.Errorf("task validation failed for '%s': %w", task.ID, err)
		}
	}

	return tasks, nil
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
		// Validate task first
		if err := ValidateTask(task); err != nil {
			return nil, fmt.Errorf("task validation failed for '%s': %w", task.ID, err)
		}

		// Check condition
		if task.Condition != "" {
			shouldExecute, err := parser.evaluateCondition(task.Condition, variables)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate condition for task '%s': %w", task.ID, err)
			}
			if !shouldExecute {
				continue // Skip this task
			}
		}

		filteredTasks = append(filteredTasks, task)
	}

	return filteredTasks, nil
}

// evaluateCondition evaluates a condition string against variables
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
	})

	// Wrap condition in template syntax
	conditionTemplate := "{{" + condition + "}}"
	tmpl, err := tmpl.Parse(conditionTemplate)
	if err != nil {
		return false, fmt.Errorf("failed to parse condition template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, variables); err != nil {
		return false, fmt.Errorf("failed to execute condition template: %w", err)
	}

	resultStr := strings.TrimSpace(result.String())
	return resultStr == "true", nil
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
