package jobs

import (
	"fmt"
	"sort"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/config"
)

// JobParser handles parsing of jobs from YAML configuration
type JobParser struct {
	orderCounter int
}

// NewJobParser creates a new job parser
func NewJobParser() *JobParser {
	return &JobParser{
		orderCounter: 0,
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
	return []*config.Task{
		{
			ID:     fmt.Sprintf("%s_%d", actionKey, p.orderCounter),
			Action: actionKey,
			Config: p.stringToConfig(actionKey, value),
			Order:  p.orderCounter,
		},
	}
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
			ID:     fmt.Sprintf("%s_%d", actionKey, p.orderCounter),
			Action: actionKey,
			Config: taskConfig,
			Order:  p.orderCounter,
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// createJobsFromObject creates a task from an object value
func (p *JobParser) createJobsFromObject(actionKey string, value map[string]interface{}) []*config.Task {
	p.orderCounter++
	return []*config.Task{
		{
			ID:     fmt.Sprintf("%s_%d", actionKey, p.orderCounter),
			Action: actionKey,
			Config: value,
			Order:  p.orderCounter,
		},
	}
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

	parser := NewJobParser()
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
