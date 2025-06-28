package modules

import (
	"fmt"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/config"
)

// ActionParameter describes a parameter for an action
type ActionParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

// ActionDocumentation provides documentation for a module action
type ActionDocumentation struct {
	Action      string             `json:"action"`
	Description string             `json:"description"`
	Parameters  []ActionParameter  `json:"parameters"`
	Examples    []ActionExample    `json:"examples,omitempty"`
}

// ActionExample provides an example of how to use an action
type ActionExample struct {
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
}

// Module represents a dotfiles module that can execute tasks
type Module interface {
	// Name returns the module name for logging and identification
	Name() string

	// ActionKeys returns the action keys this module handles
	ActionKeys() []string

	// ValidateTask validates a task configuration for this module
	ValidateTask(task *config.Task) error

	// ExecuteTask executes a single task
	ExecuteTask(task *config.Task, ctx *ExecutionContext) error

	// PlanTask returns what the task would do without executing it
	PlanTask(task *config.Task, ctx *ExecutionContext) (*TaskPlan, error)

	// ExplainAction returns documentation for a specific action
	ExplainAction(action string) (*ActionDocumentation, error)

	// ListActions returns documentation for all actions supported by this module
	ListActions() []*ActionDocumentation
}

// ExecutionContext provides context for task execution
type ExecutionContext struct {
	BasePath    string                 // Base directory of dotfiles repo
	Variables   map[string]interface{} // Processed variables
	DryRun      bool                   // Whether this is a dry run
	Verbose     bool                   // Whether to output verbose information
	ShowDiff    bool                   // Whether to show detailed diffs of file changes
	HideSkipped bool                   // Whether to hide skipped jobs from output
}

// TaskPlan describes what a task would do
type TaskPlan struct {
	TaskID      string   `json:"task_id"`
	Action      string   `json:"action"`
	Description string   `json:"description"`
	Changes     []string `json:"changes"`
	WillSkip    bool     `json:"will_skip"`
	SkipReason  string   `json:"skip_reason"`
}

// TaskResult represents the result of executing a task
type TaskResult struct {
	TaskID  string   `json:"task_id"`
	Success bool     `json:"success"`
	Error   error    `json:"error,omitempty"`
	Changes []string `json:"changes"`
	Skipped bool     `json:"skipped"`
	Message string   `json:"message"`
}

// ModuleRegistry manages available modules
type ModuleRegistry struct {
	modules     map[string]Module
	actionIndex map[string]Module // Maps action keys to modules
}

// NewModuleRegistry creates a new module registry
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules:     make(map[string]Module),
		actionIndex: make(map[string]Module),
	}
}

// Register registers a module in the registry
func (r *ModuleRegistry) Register(module Module) error {
	name := module.Name()
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("module %s is already registered", name)
	}

	r.modules[name] = module

	// Register action keys
	for _, actionKey := range module.ActionKeys() {
		if existingModule, exists := r.actionIndex[actionKey]; exists {
			return fmt.Errorf("action key %s is already registered by module %s",
				actionKey, existingModule.Name())
		}
		r.actionIndex[actionKey] = module
	}

	return nil
}

// GetModuleByAction returns the module that handles the given action
func (r *ModuleRegistry) GetModuleByAction(action string) (Module, error) {
	module, exists := r.actionIndex[action]
	if !exists {
		return nil, fmt.Errorf("no module registered for action: %s", action)
	}
	return module, nil
}

// GetModule returns a module by name
func (r *ModuleRegistry) GetModule(name string) (Module, error) {
	module, exists := r.modules[name]
	if !exists {
		return nil, fmt.Errorf("module not found: %s", name)
	}
	return module, nil
}

// GetAllModules returns all registered modules
func (r *ModuleRegistry) GetAllModules() map[string]Module {
	result := make(map[string]Module)
	for name, module := range r.modules {
		result[name] = module
	}
	return result
}

// GetSupportedActions returns all supported action keys
func (r *ModuleRegistry) GetSupportedActions() []string {
	actions := make([]string, 0, len(r.actionIndex))
	for action := range r.actionIndex {
		actions = append(actions, action)
	}
	return actions
}

// ValidateTask validates a task using the appropriate module
func (r *ModuleRegistry) ValidateTask(task *config.Task) error {
	module, err := r.GetModuleByAction(task.Action)
	if err != nil {
		return err
	}
	return module.ValidateTask(task)
}

// ExecuteTask executes a task using the appropriate module
func (r *ModuleRegistry) ExecuteTask(task *config.Task, ctx *ExecutionContext) (*TaskResult, error) {
	module, err := r.GetModuleByAction(task.Action)
	if err != nil {
		return &TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   err,
		}, err
	}

	err = module.ExecuteTask(task, ctx)
	if err != nil {
		return &TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   err,
		}, err
	}

	return &TaskResult{
		TaskID:  task.ID,
		Success: true,
	}, nil
}

// PlanTask creates an execution plan for a task using the appropriate module
func (r *ModuleRegistry) PlanTask(task *config.Task, ctx *ExecutionContext) (*TaskPlan, error) {
	module, err := r.GetModuleByAction(task.Action)
	if err != nil {
		return nil, err
	}
	return module.PlanTask(task, ctx)
}

// ExplainAction returns documentation for a specific action
func (r *ModuleRegistry) ExplainAction(action string) (*ActionDocumentation, error) {
	module, err := r.GetModuleByAction(action)
	if err != nil {
		return nil, err
	}
	return module.ExplainAction(action)
}

// ExplainModule returns documentation for all actions in a module
func (r *ModuleRegistry) ExplainModule(moduleName string) ([]*ActionDocumentation, error) {
	module, err := r.GetModule(moduleName)
	if err != nil {
		return nil, err
	}
	return module.ListActions(), nil
}

// ListAllActions returns documentation for all actions from all modules
func (r *ModuleRegistry) ListAllActions() map[string][]*ActionDocumentation {
	result := make(map[string][]*ActionDocumentation)
	for name, module := range r.modules {
		result[name] = module.ListActions()
	}
	return result
}

// NewDefaultRegistry creates a registry with all built-in modules
func NewDefaultRegistry() *ModuleRegistry {
	registry := NewModuleRegistry()

	// Register built-in modules
	// Note: Import statements need to be added to the file
	// registry.Register(symlinks.New())
	// registry.Register(files.New())
	// registry.Register(packages.New())

	return registry
}
