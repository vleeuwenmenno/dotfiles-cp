package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/config"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules"
)

func TestCommandsModule(t *testing.T) {
	module := New()

	t.Run("ModuleName", func(t *testing.T) {
		assert.Equal(t, "commands", module.Name())
	})

	t.Run("ActionKeys", func(t *testing.T) {
		keys := module.ActionKeys()
		assert.Contains(t, keys, "run_command")
		assert.Len(t, keys, 1)
	})

	t.Run("ValidateRunCommand", func(t *testing.T) {
		// Valid configuration
		task := &config.Task{
			Action: "run_command",
			Config: map[string]interface{}{
				"name":    "Test command",
				"command": "echo hello",
			},
		}
		err := module.ValidateTask(task)
		assert.NoError(t, err)

		// Missing name
		task.Config = map[string]interface{}{
			"command": "echo hello",
		}
		err = module.ValidateTask(task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")

		// Missing command
		task.Config = map[string]interface{}{
			"name": "Test command",
		}
		err = module.ValidateTask(task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "command is required")
	})

	t.Run("ExplainAction", func(t *testing.T) {
		doc, err := module.ExplainAction("run_command")
		assert.NoError(t, err)
		assert.Equal(t, "run_command", doc.Action)
		assert.NotEmpty(t, doc.Description)
		assert.NotEmpty(t, doc.Parameters)

		// Invalid action
		_, err = module.ExplainAction("invalid_action")
		assert.Error(t, err)
	})

	t.Run("ListActions", func(t *testing.T) {
		actions := module.ListActions()
		assert.Len(t, actions, 1)
		assert.Equal(t, "run_command", actions[0].Action)
	})

	t.Run("ParseCommandConfig", func(t *testing.T) {
		config := map[string]interface{}{
			"name":    "Test command",
			"command": "echo hello",
			"when":    "test -f /nonexistent",
			"shell":   "bash",
			"workdir": "/tmp",
			"env": map[string]interface{}{
				"TEST_VAR": "test_value",
			},
		}

		cmdConfig, err := module.parseCommandConfig(config)
		assert.NoError(t, err)
		assert.Equal(t, "Test command", cmdConfig.Name)
		assert.Equal(t, "echo hello", cmdConfig.Command)
		assert.Equal(t, "test -f /nonexistent", cmdConfig.When)
		assert.Equal(t, "bash", cmdConfig.Shell)
		assert.Equal(t, "/tmp", cmdConfig.WorkDir)
		assert.Equal(t, "test_value", cmdConfig.Env["TEST_VAR"])
	})

	t.Run("DryRunExecution", func(t *testing.T) {
		task := &config.Task{
			Action: "run_command",
			Config: map[string]interface{}{
				"name":    "Test command",
				"command": "echo hello",
			},
		}

		ctx := &modules.ExecutionContext{
			DryRun: true,
		}

		err := module.ExecuteTask(task, ctx)
		assert.NoError(t, err)
	})
}

func TestGetShell(t *testing.T) {
	module := New()

	t.Run("SpecifiedShell", func(t *testing.T) {
		shell := module.getShell("bash")
		assert.Equal(t, []string{"bash", "-c"}, shell)

		shell = module.getShell("zsh")
		assert.Equal(t, []string{"zsh", "-c"}, shell)

		shell = module.getShell("powershell")
		assert.Equal(t, []string{"powershell", "-Command"}, shell)

		shell = module.getShell("cmd")
		assert.Equal(t, []string{"cmd", "/c"}, shell)
	})

	t.Run("AutoDetection", func(t *testing.T) {
		// This will test auto-detection based on current platform
		shell := module.getShell("")
		assert.NotEmpty(t, shell)
		assert.Len(t, shell, 2)
	})

	t.Run("UnknownShell", func(t *testing.T) {
		// Should fallback to auto-detection for unknown shells
		shell := module.getShell("unknown-shell")
		assert.NotEmpty(t, shell)
		assert.Len(t, shell, 2)
	})
}
