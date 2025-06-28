package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/config"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/logger"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules"
)

// CommandsModule handles running arbitrary commands with state checking
type CommandsModule struct{}

// CommandConfig represents the configuration for a command
type CommandConfig struct {
	Name    string            `json:"name"`
	When    string            `json:"when"`              // Command to check current state
	Command string            `json:"command"`           // Command to execute if when check fails
	Shell   string            `json:"shell,omitempty"`   // Shell to use (optional, auto-detected)
	WorkDir string            `json:"workdir,omitempty"` // Working directory (optional)
	Env     map[string]string `json:"env,omitempty"`     // Environment variables (optional)
}

// New creates a new commands module
func New() *CommandsModule {
	return &CommandsModule{}
}

// Name returns the module name
func (m *CommandsModule) Name() string {
	return "commands"
}

// ActionKeys returns the supported action keys
func (m *CommandsModule) ActionKeys() []string {
	return []string{"run_command"}
}

// ValidateTask validates the task configuration
func (m *CommandsModule) ValidateTask(task *config.Task) error {
	switch task.Action {
	case "run_command":
		return m.validateRunCommand(task.Config)
	default:
		return fmt.Errorf("unsupported action: %s", task.Action)
	}
}

// ExecuteTask executes the specified task
func (m *CommandsModule) ExecuteTask(task *config.Task, ctx *modules.ExecutionContext) error {
	switch task.Action {
	case "run_command":
		return m.executeRunCommand(task, ctx)
	default:
		return fmt.Errorf("unsupported action: %s", task.Action)
	}
}

// PlanTask creates an execution plan for the task
func (m *CommandsModule) PlanTask(task *config.Task, ctx *modules.ExecutionContext) (*modules.TaskPlan, error) {
	switch task.Action {
	case "run_command":
		return m.planRunCommand(task, ctx)
	default:
		return nil, fmt.Errorf("unsupported action: %s", task.Action)
	}
}

// validateRunCommand validates run_command configuration
func (m *CommandsModule) validateRunCommand(config map[string]interface{}) error {
	if name, exists := config["name"]; !exists || name == "" {
		return fmt.Errorf("name is required for run_command")
	}

	if command, exists := config["command"]; !exists || command == "" {
		return fmt.Errorf("command is required for run_command")
	}

	// when is optional - if not provided, command always runs
	// shell is optional - will be auto-detected
	// workdir is optional
	// env is optional

	return nil
}

// executeRunCommand executes a run_command task
func (m *CommandsModule) executeRunCommand(task *config.Task, ctx *modules.ExecutionContext) error {
	log := logger.Get()

	cmdConfig, err := m.parseCommandConfig(task.Config)
	if err != nil {
		return fmt.Errorf("invalid command configuration: %w", err)
	}

	// Check if we should run the command using 'when' condition
	shouldRun, err := m.shouldRunCommand(cmdConfig)
	if err != nil {
		return fmt.Errorf("failed to check when condition: %w", err)
	}

	if !shouldRun {
		log.Info().Str("command", cmdConfig.Name).Msg("Command skipped - when condition already satisfied")
		return nil
	}

	if ctx.DryRun {
		log.Info().Str("command", cmdConfig.Name).Msg("Would execute command (dry run)")
		return nil
	}

	// Execute the command
	log.Info().Str("command", cmdConfig.Name).Msg("Executing command")
	err = m.runCommand(cmdConfig)
	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	log.Info().Str("command", cmdConfig.Name).Msg("Command executed successfully")
	return nil
}

// planRunCommand creates an execution plan for run_command
func (m *CommandsModule) planRunCommand(task *config.Task, ctx *modules.ExecutionContext) (*modules.TaskPlan, error) {
	cmdConfig, err := m.parseCommandConfig(task.Config)
	if err != nil {
		return &modules.TaskPlan{
			TaskID:     task.ID,
			Action:     task.Action,
			WillSkip:   true,
			SkipReason: fmt.Sprintf("Invalid configuration: %v", err),
		}, nil
	}

	// Check if we should run the command
	shouldRun, err := m.shouldRunCommand(cmdConfig)
	if err != nil {
		return &modules.TaskPlan{
			TaskID:     task.ID,
			Action:     task.Action,
			WillSkip:   true,
			SkipReason: fmt.Sprintf("Failed to check when condition: %v", err),
		}, nil
	}

	plan := &modules.TaskPlan{
		TaskID:      task.ID,
		Action:      task.Action,
		Description: cmdConfig.Name,
	}

	if shouldRun {
		plan.Changes = []string{fmt.Sprintf("Execute: %s", cmdConfig.Command)}
	} else {
		plan.WillSkip = true
		plan.SkipReason = "Command already in desired state (when condition satisfied)"
	}

	return plan, nil
}

// parseCommandConfig parses the command configuration from task config
func (m *CommandsModule) parseCommandConfig(config map[string]interface{}) (*CommandConfig, error) {
	cmdConfig := &CommandConfig{}

	// Required fields
	if name, exists := config["name"]; exists {
		cmdConfig.Name = name.(string)
	}

	if command, exists := config["command"]; exists {
		cmdConfig.Command = command.(string)
	}

	// Optional fields
	if when, exists := config["when"]; exists {
		cmdConfig.When = when.(string)
	}

	if shell, exists := config["shell"]; exists {
		cmdConfig.Shell = shell.(string)
	}

	if workdir, exists := config["workdir"]; exists {
		cmdConfig.WorkDir = workdir.(string)
	}

	if env, exists := config["env"]; exists {
		if envMap, ok := env.(map[string]interface{}); ok {
			cmdConfig.Env = make(map[string]string)
			for k, v := range envMap {
				cmdConfig.Env[k] = v.(string)
			}
		}
	}

	return cmdConfig, nil
}

// shouldRunCommand checks if the command should be executed based on 'when' condition
func (m *CommandsModule) shouldRunCommand(cmdConfig *CommandConfig) (bool, error) {
	// If no 'when' condition is specified, always run the command
	if cmdConfig.When == "" {
		return true, nil
	}

	// Execute the 'when' command
	shell := m.getShell(cmdConfig.Shell)
	cmd := m.createCommand(shell, cmdConfig.When, cmdConfig.WorkDir, cmdConfig.Env)

	err := cmd.Run()
	if err != nil {
		// 'when' command failed (non-zero exit), so we should run the main command
		return true, nil
	}

	// 'when' command succeeded (exit code 0), so we should skip the main command
	return false, nil
}

// runCommand executes the main command
func (m *CommandsModule) runCommand(cmdConfig *CommandConfig) error {
	shell := m.getShell(cmdConfig.Shell)
	cmd := m.createCommand(shell, cmdConfig.Command, cmdConfig.WorkDir, cmdConfig.Env)

	// Set up output handling
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// getShell returns the appropriate shell command based on platform and preference
func (m *CommandsModule) getShell(preferredShell string) []string {
	if preferredShell != "" {
		// Use specified shell
		switch preferredShell {
		case "bash":
			return []string{"bash", "-c"}
		case "zsh":
			return []string{"zsh", "-c"}
		case "sh":
			return []string{"sh", "-c"}
		case "powershell":
			return []string{"powershell", "-Command"}
		case "cmd":
			return []string{"cmd", "/c"}
		default:
			// Fallback to auto-detection if unknown shell specified
		}
	}

	// Auto-detect based on platform
	switch runtime.GOOS {
	case "windows":
		// Check if PowerShell is available, fallback to cmd
		if _, err := exec.LookPath("powershell"); err == nil {
			return []string{"powershell", "-Command"}
		}
		return []string{"cmd", "/c"}
	case "darwin", "linux":
		// Check for preferred shells in order: bash, zsh, sh
		shells := []string{"bash", "zsh", "sh"}
		for _, shell := range shells {
			if _, err := exec.LookPath(shell); err == nil {
				return []string{shell, "-c"}
			}
		}
		// Fallback to sh (should always exist on Unix systems)
		return []string{"sh", "-c"}
	default:
		// Unknown platform, try bash
		return []string{"bash", "-c"}
	}
}

// createCommand creates an exec.Cmd with the specified parameters
func (m *CommandsModule) createCommand(shell []string, command, workDir string, env map[string]string) *exec.Cmd {
	// Create command with shell
	args := append(shell, command)
	cmd := exec.Command(args[0], args[1:]...)

	// Set working directory if specified
	if workDir != "" {
		// Handle tilde expansion
		if strings.HasPrefix(workDir, "~") {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				workDir = filepath.Join(homeDir, workDir[1:])
			}
		}
		cmd.Dir = workDir
	}

	// Set environment variables
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return cmd
}

// ExplainAction returns documentation for a specific action
func (m *CommandsModule) ExplainAction(action string) (*modules.ActionDocumentation, error) {
	switch action {
	case "run_command":
		return &modules.ActionDocumentation{
			Action:      "run_command",
			Description: "Execute arbitrary commands with state checking",
			Parameters: []modules.ActionParameter{
				{
					Name:        "name",
					Type:        "string",
					Required:    true,
					Description: "Description of what the command does",
				},
				{
					Name:        "command",
					Type:        "string",
					Required:    true,
					Description: "Command to execute",
				},
				{
					Name:        "when",
					Type:        "string",
					Required:    false,
					Description: "Command to check current state (execute main command only if this fails)",
				},
				{
					Name:        "shell",
					Type:        "string",
					Required:    false,
					Description: "Shell to use (bash, zsh, sh, powershell, cmd) - auto-detected if not specified",
				},
				{
					Name:        "workdir",
					Type:        "string",
					Required:    false,
					Description: "Working directory for command execution",
				},
				{
					Name:        "env",
					Type:        "map[string]string",
					Required:    false,
					Description: "Environment variables to set",
				},
			},
			Examples: []modules.ActionExample{
				{
					Description: "Install rustup on Unix systems",
					Config: map[string]interface{}{
						"name":    "Install rustup",
						"when":    "command -v rustup",
						"command": "curl --proto '=https' --tlsv1.3 https://sh.rustup.rs -sSf | sh",
					},
				},
				{
					Description: "Install Oh My Zsh with custom shell",
					Config: map[string]interface{}{
						"name":    "Install Oh My Zsh",
						"when":    "test -d ~/.oh-my-zsh",
						"command": "sh -c \"$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)\"",
						"shell":   "bash",
					},
				},
				{
					Description: "Run command with environment variables",
					Config: map[string]interface{}{
						"name":    "Setup development environment",
						"command": "make setup",
						"workdir": "~/projects/myapp",
						"env": map[string]string{
							"NODE_ENV": "development",
							"DEBUG":    "true",
						},
					},
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
	}
}

// ListActions returns documentation for all actions supported by this module
func (m *CommandsModule) ListActions() []*modules.ActionDocumentation {
	docs := []*modules.ActionDocumentation{}
	for _, action := range m.ActionKeys() {
		if doc, err := m.ExplainAction(action); err == nil {
			docs = append(docs, doc)
		}
	}
	return docs
}
