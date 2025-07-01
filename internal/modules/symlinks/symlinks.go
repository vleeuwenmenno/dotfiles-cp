package symlinks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/config"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/templating"
	"github.com/vleeuwenmenno/dotfiles-cp/pkg/utils"
)

// SymlinksModule handles symlink creation and management
type SymlinksModule struct {
	templateEngine *templating.TemplatingEngine
}

// New creates a new symlinks module
func New() *SymlinksModule {
	return &SymlinksModule{
		templateEngine: templating.NewTemplatingEngine("."),
	}
}

// Name returns the module name
func (m *SymlinksModule) Name() string {
	return "symlinks"
}

// ActionKeys returns the action keys this module handles
func (m *SymlinksModule) ActionKeys() []string {
	return []string{"symlink"}
}

// ValidateTask validates a symlink task configuration
func (m *SymlinksModule) ValidateTask(task *config.Task) error {
	if task.Action != "symlink" {
		return fmt.Errorf("symlinks module only handles 'symlink' action, got '%s'", task.Action)
	}

	src, exists := task.Config["src"]
	if !exists {
		return fmt.Errorf("symlink task requires 'src' field")
	}
	if _, ok := src.(string); !ok {
		return fmt.Errorf("symlink 'src' must be a string")
	}

	dst, exists := task.Config["dst"]
	if !exists {
		return fmt.Errorf("symlink task requires 'dst' field")
	}
	if _, ok := dst.(string); !ok {
		return fmt.Errorf("symlink 'dst' must be a string")
	}

	return nil
}

// ExecuteTask executes a symlink task
func (m *SymlinksModule) ExecuteTask(task *config.Task, ctx *modules.ExecutionContext) error {
	if ctx.DryRun {
		return nil // Plan already showed what would happen
	}

	// Process templates in src and dst paths
	src, err := m.processTemplate(task.Config["src"].(string), ctx.Variables)
	if err != nil {
		return fmt.Errorf("failed to process src template: %w", err)
	}

	dst, err := m.processTemplate(task.Config["dst"].(string), ctx.Variables)
	if err != nil {
		return fmt.Errorf("failed to process dst template: %w", err)
	}

	// Resolve source path relative to base path
	if !filepath.IsAbs(src) {
		src = filepath.Join(ctx.BasePath, src)
	}

	// Expand destination path
	dst, err = utils.ExpandPath(dst)
	if err != nil {
		return fmt.Errorf("failed to expand destination path: %w", err)
	}

	// Check if source exists
	if !utils.FileExists(src) {
		return fmt.Errorf("source file does not exist: %s", src)
	}

	// Handle backup if requested
	backup, _ := task.Config["backup"].(bool)
	if backup && utils.FileExists(dst) {
		backupPath := dst + ".backup"
		if ctx.Verbose {
			fmt.Printf("Creating backup: %s -> %s\n", dst, backupPath)
		}
		if err := os.Rename(dst, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Remove existing file/link if it exists
	if utils.FileExists(dst) {
		if err := os.Remove(dst); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := utils.EnsureDir(dstDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create the symlink
	if ctx.Verbose {
		fmt.Printf("Creating symlink: %s -> %s\n", src, dst)
	}

	if err := os.Symlink(src, dst); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// PlanTask returns what the symlink task would do
func (m *SymlinksModule) PlanTask(task *config.Task, ctx *modules.ExecutionContext) (*modules.TaskPlan, error) {
	// Process templates in src and dst paths
	src, err := m.processTemplate(task.Config["src"].(string), ctx.Variables)
	if err != nil {
		return nil, fmt.Errorf("failed to process src template: %w", err)
	}

	dst, err := m.processTemplate(task.Config["dst"].(string), ctx.Variables)
	if err != nil {
		return nil, fmt.Errorf("failed to process dst template: %w", err)
	}

	// Resolve source path relative to base path
	if !filepath.IsAbs(src) {
		src = filepath.Join(ctx.BasePath, src)
	}

	// Expand destination path
	dst, err = utils.ExpandPath(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to expand destination path: %w", err)
	}

	plan := &modules.TaskPlan{
		TaskID:      task.ID,
		Action:      task.Action,
		Description: fmt.Sprintf("Create symlink %s -> %s", src, dst),
		Changes:     []string{},
	}

	// Check if source exists
	if !utils.FileExists(src) {
		plan.WillSkip = true
		plan.SkipReason = fmt.Sprintf("Source file does not exist: %s", src)
		return plan, nil
	}

	// Check what changes would be made
	if utils.FileExists(dst) {
		// Check if it's already a symlink to the right target
		if linkTarget, err := os.Readlink(dst); err == nil {
			if linkTarget == src {
				plan.WillSkip = true
				plan.SkipReason = "Symlink already exists and points to correct target"
				return plan, nil
			} else {
				plan.Changes = append(plan.Changes, fmt.Sprintf("Update symlink target from %s to %s", linkTarget, src))
			}
		} else {
			// Existing file/directory
			backup, _ := task.Config["backup"].(bool)
			if backup {
				plan.Changes = append(plan.Changes, fmt.Sprintf("Backup existing file to %s.backup", dst))
			}
			plan.Changes = append(plan.Changes, "Replace existing file with symlink")
		}
	} else {
		plan.Changes = append(plan.Changes, "Create new symlink")
	}

	// Check if destination directory needs to be created
	dstDir := filepath.Dir(dst)
	if !utils.FileExists(dstDir) {
		plan.Changes = append(plan.Changes, fmt.Sprintf("Create directory %s", dstDir))
	}

	return plan, nil
}

// processTemplate processes a template string with variables using the new templating engine
func (m *SymlinksModule) processTemplate(templateStr string, variables map[string]interface{}) (string, error) {
	result, err := m.templateEngine.ProcessVariableTemplate(templateStr, variables)
	if err != nil {
		return "", err
	}

	// Ensure OS-specific path separators
	return filepath.FromSlash(result), nil
}

// ExplainAction returns documentation for a specific action
func (m *SymlinksModule) ExplainAction(action string) (*modules.ActionDocumentation, error) {
	docs := m.ListActions()
	for _, doc := range docs {
		if doc.Action == action {
			return doc, nil
		}
	}
	return nil, fmt.Errorf("action '%s' not supported by symlinks module", action)
}

// ListActions returns documentation for all actions supported by this module
func (m *SymlinksModule) ListActions() []*modules.ActionDocumentation {
	return []*modules.ActionDocumentation{
		{
			Action:      "symlink",
			Description: "Creates a symbolic link from a source file in the dotfiles repository to a destination path. Handles backup of existing files if requested.",
			Parameters: []modules.ActionParameter{
				{
					Name:        "src",
					Type:        "string",
					Required:    true,
					Description: "The source file path relative to the dotfiles repository root. Supports template variables.",
				},
				{
					Name:        "dst",
					Type:        "string",
					Required:    true,
					Description: "The destination path where the symlink will be created. Supports template variables and path expansion (e.g., ~ for home directory).",
				},
				{
					Name:        "backup",
					Type:        "boolean",
					Required:    false,
					Default:     "false",
					Description: "Whether to create a backup of existing files before creating the symlink. Backup files are named with a .backup suffix.",
				},
			},
			Examples: []modules.ActionExample{
				{
					Description: "Create a basic symlink",
					Config: map[string]interface{}{
						"src": "files/config/nvim/init.vim",
						"dst": "{{ .paths.home }}/.config/nvim/init.vim",
					},
				},
				{
					Description: "Create a symlink with backup",
					Config: map[string]interface{}{
						"src":    "files/config/git/gitconfig",
						"dst":    "{{ .paths.home }}/.gitconfig",
						"backup": true,
					},
				},
				{
					Description: "Symlink an entire directory",
					Config: map[string]interface{}{
						"src": "files/config/zsh",
						"dst": "{{ .paths.home }}/.config/zsh",
					},
				},
			},
		},
	}
}
