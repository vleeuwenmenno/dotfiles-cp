package files

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/config"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules"
	"github.com/vleeuwenmenno/dotfiles-cp/pkg/utils"
)

// FilesModule handles file and directory operations
type FilesModule struct{}

// New creates a new files module
func New() *FilesModule {
	return &FilesModule{}
}

// Name returns the module name
func (m *FilesModule) Name() string {
	return "files"
}

// ActionKeys returns the action keys this module handles
func (m *FilesModule) ActionKeys() []string {
	return []string{"ensure_dir", "ensure_file"}
}

// ValidateTask validates a file task configuration
func (m *FilesModule) ValidateTask(task *config.Task) error {
	switch task.Action {
	case "ensure_dir":
		return m.validateEnsureDirTask(task.Config)
	case "ensure_file":
		return m.validateEnsureFileTask(task.Config)
	default:
		return fmt.Errorf("files module does not handle action '%s'", task.Action)
	}
}

// ExecuteTask executes a file task
func (m *FilesModule) ExecuteTask(task *config.Task, ctx *modules.ExecutionContext) error {
	if ctx.DryRun {
		return nil // Plan already showed what would happen
	}

	switch task.Action {
	case "ensure_dir":
		return m.executeEnsureDir(task, ctx)
	case "ensure_file":
		return m.executeEnsureFile(task, ctx)
	default:
		return fmt.Errorf("files module does not handle action '%s'", task.Action)
	}
}

// PlanTask returns what the file task would do
func (m *FilesModule) PlanTask(task *config.Task, ctx *modules.ExecutionContext) (*modules.TaskPlan, error) {
	switch task.Action {
	case "ensure_dir":
		return m.planEnsureDir(task, ctx)
	case "ensure_file":
		return m.planEnsureFile(task, ctx)
	default:
		return nil, fmt.Errorf("files module does not handle action '%s'", task.Action)
	}
}

// validateEnsureDirTask validates ensure_dir task configuration
func (m *FilesModule) validateEnsureDirTask(config map[string]interface{}) error {
	if _, exists := config["path"]; !exists {
		return fmt.Errorf("ensure_dir task requires 'path' field")
	}
	if _, ok := config["path"].(string); !ok {
		return fmt.Errorf("ensure_dir 'path' must be a string")
	}
	return nil
}

// validateEnsureFileTask validates ensure_file task configuration
func (m *FilesModule) validateEnsureFileTask(config map[string]interface{}) error {
	if _, exists := config["path"]; !exists {
		return fmt.Errorf("ensure_file task requires 'path' field")
	}
	if _, ok := config["path"].(string); !ok {
		return fmt.Errorf("ensure_file 'path' must be a string")
	}

	// Check that content and content_source are mutually exclusive
	hasContent := false
	hasContentSource := false

	if content, exists := config["content"]; exists {
		if _, ok := content.(string); !ok {
			return fmt.Errorf("ensure_file 'content' must be a string")
		}
		hasContent = true
	}

	if contentSource, exists := config["content_source"]; exists {
		if _, ok := contentSource.(string); !ok {
			return fmt.Errorf("ensure_file 'content_source' must be a string")
		}
		hasContentSource = true
	}

	if hasContent && hasContentSource {
		return fmt.Errorf("ensure_file 'content' and 'content_source' are mutually exclusive")
	}

	// Validate render parameter if present
	if render, exists := config["render"]; exists {
		if _, ok := render.(bool); !ok {
			return fmt.Errorf("ensure_file 'render' must be a boolean")
		}
	}

	return nil
}



// executeEnsureDir ensures a directory exists with proper permissions
func (m *FilesModule) executeEnsureDir(task *config.Task, ctx *modules.ExecutionContext) error {
	// Process template in path
	path, err := m.processTemplate(task.Config["path"].(string), ctx.Variables)
	if err != nil {
		return fmt.Errorf("failed to process path template: %w", err)
	}

	// Expand path
	path, err = utils.ExpandPath(path)
	if err != nil {
		return fmt.Errorf("failed to expand path: %w", err)
	}

	// Get mode (default to 0755, but only used on Unix-like systems)
	mode := os.FileMode(0755)
	if modeStr, exists := task.Config["mode"]; exists {
		if modeString, ok := modeStr.(string); ok {
			if parsedMode, err := strconv.ParseUint(modeString, 8, 32); err == nil {
				mode = os.FileMode(parsedMode)
			}
		}
	}

	// Check if directory already exists
	if stat, err := os.Stat(path); err == nil {
		if stat.IsDir() {
			// On Windows, we just check if directory exists
			// On Unix, we also check permissions
			if runtime.GOOS == "windows" {
				if ctx.Verbose {
					fmt.Printf("Directory already exists: %s\n", path)
				}
				return nil // Nothing to do on Windows
			} else {
				// Unix-like systems: check permissions
				currentMode := stat.Mode().Perm()
				if currentMode == mode {
					if ctx.Verbose {
						fmt.Printf("Directory already exists with correct permissions: %s (mode: %04o)\n", path, mode)
					}
					return nil // Nothing to do
				}
			}
		}
	}

	if ctx.Verbose {
		if runtime.GOOS == "windows" {
			fmt.Printf("Ensuring directory exists: %s\n", path)
		} else {
			fmt.Printf("Ensuring directory exists: %s (mode: %04o)\n", path, mode)
		}
	}

	// Create directory with proper permissions
	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Only set permissions on Unix-like systems
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, mode); err != nil {
			return fmt.Errorf("failed to set directory permissions: %w", err)
		}
	}

	return nil
}

// executeEnsureFile ensures a file exists with optional content
func (m *FilesModule) executeEnsureFile(task *config.Task, ctx *modules.ExecutionContext) error {
	// Process template in path
	path, err := m.processTemplate(task.Config["path"].(string), ctx.Variables)
	if err != nil {
		return fmt.Errorf("failed to process path template: %w", err)
	}

	// Expand path
	path, err = utils.ExpandPath(path)
	if err != nil {
		return fmt.Errorf("failed to expand path: %w", err)
	}

	// Get mode (default to 0644)
	mode := os.FileMode(0644)
	if modeStr, exists := task.Config["mode"]; exists {
		if modeString, ok := modeStr.(string); ok {
			if parsedMode, err := strconv.ParseUint(modeString, 8, 32); err == nil {
				mode = os.FileMode(parsedMode)
			}
		}
	}

	// Ensure parent directory exists
	if err := utils.EnsureDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Get content from either inline content or content_source
	content := ""

	if contentSourceStr, exists := task.Config["content_source"]; exists {
		// Read content from source file
		contentSourcePath, err := m.processTemplate(contentSourceStr.(string), ctx.Variables)
		if err != nil {
			return fmt.Errorf("failed to process content_source template: %w", err)
		}

		// Resolve content source path relative to base path
		if !filepath.IsAbs(contentSourcePath) {
			contentSourcePath = filepath.Join(ctx.BasePath, contentSourcePath)
		}

		// Check if source file exists
		if !utils.FileExists(contentSourcePath) {
			return fmt.Errorf("content source file does not exist: %s", contentSourcePath)
		}

		// Read the source file
		contentBytes, err := os.ReadFile(contentSourcePath)
		if err != nil {
			return fmt.Errorf("failed to read content source file: %w", err)
		}
		content = string(contentBytes)

		// Check if we should render the content as a template
		if render, exists := task.Config["render"]; exists && render.(bool) {
			content, err = m.processTemplateWithPathConversion(content, ctx.Variables, false)
			if err != nil {
				return fmt.Errorf("failed to render content template: %w", err)
			}
		}
	} else if contentStr, exists := task.Config["content"]; exists {
		// Use inline content (always process as template for backward compatibility)
		if contentString, ok := contentStr.(string); ok {
			content, err = m.processTemplateWithPathConversion(contentString, ctx.Variables, false)
			if err != nil {
				return fmt.Errorf("failed to process content template: %w", err)
			}
		}
	}
	// If neither content nor content_source is specified, content remains empty

	// Check if file already exists and compare content
	fileExists := utils.FileExists(path)
	needsUpdate := true

	if fileExists {
		// Read existing content
		existingContent, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read existing file: %w", err)
		}

		// Compare content
		if string(existingContent) == content {
			needsUpdate = false
			if ctx.Verbose {
				fmt.Printf("File content unchanged: %s\n", path)
			}
			// Just ensure permissions are correct
			return os.Chmod(path, mode)
		}
	}

	if needsUpdate {
		if ctx.Verbose {
			if fileExists {
				fmt.Printf("Updating file: %s (mode: %04o)\n", path, mode)
			} else {
				if contentSourceStr, exists := task.Config["content_source"]; exists {
					fmt.Printf("Creating file from source: %s -> %s (mode: %04o)\n", contentSourceStr, path, mode)
				} else {
					fmt.Printf("Creating file: %s (mode: %04o)\n", path, mode)
				}
			}
		}

		// Create or update file with content
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	return nil
}



// planEnsureDir returns what ensure_dir would do
func (m *FilesModule) planEnsureDir(task *config.Task, ctx *modules.ExecutionContext) (*modules.TaskPlan, error) {
	// Process template in path
	path, err := m.processTemplate(task.Config["path"].(string), ctx.Variables)
	if err != nil {
		return nil, fmt.Errorf("failed to process path template: %w", err)
	}

	// Expand path
	path, err = utils.ExpandPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to expand path: %w", err)
	}

	// Get mode (default to 0755, but only relevant on Unix)
	mode := "0755"
	if modeStr, exists := task.Config["mode"]; exists {
		if modeString, ok := modeStr.(string); ok {
			mode = modeString
		}
	}

	var description string
	if runtime.GOOS == "windows" {
		description = fmt.Sprintf("Ensure directory exists: %s", path)
	} else {
		description = fmt.Sprintf("Ensure directory exists: %s (mode: %s)", path, mode)
	}

	plan := &modules.TaskPlan{
		TaskID:      task.ID,
		Action:      task.Action,
		Description: description,
		Changes:     []string{},
	}

	// Check if directory already exists
	if stat, err := os.Stat(path); err == nil {
		if !stat.IsDir() {
			plan.Changes = append(plan.Changes, "Remove existing file and create directory")
		} else {
			// On Windows, just check if directory exists
			if runtime.GOOS == "windows" {
				plan.WillSkip = true
				plan.SkipReason = "Directory already exists"
			} else {
				// On Unix, check permissions too
				currentMode := stat.Mode().Perm()
				if parsedMode, err := strconv.ParseUint(mode, 8, 32); err == nil {
					expectedMode := os.FileMode(parsedMode)
					if currentMode != expectedMode {
						plan.Changes = append(plan.Changes, fmt.Sprintf("Update permissions from %04o to %s", currentMode, mode))
					} else {
						plan.WillSkip = true
						plan.SkipReason = "Directory already exists with correct permissions"
					}
				}
			}
		}
	} else {
		plan.Changes = append(plan.Changes, "Create directory")
	}

	return plan, nil
}

// planEnsureFile returns what ensure_file would do
func (m *FilesModule) planEnsureFile(task *config.Task, ctx *modules.ExecutionContext) (*modules.TaskPlan, error) {
	// Process template in path
	path, err := m.processTemplate(task.Config["path"].(string), ctx.Variables)
	if err != nil {
		return nil, fmt.Errorf("failed to process path template: %w", err)
	}

	// Expand path
	path, err = utils.ExpandPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to expand path: %w", err)
	}

	description := fmt.Sprintf("Ensure file exists: %s", path)
	if contentSourceStr, exists := task.Config["content_source"]; exists {
		description = fmt.Sprintf("Ensure file exists from source: %s -> %s", contentSourceStr, path)
	}

	plan := &modules.TaskPlan{
		TaskID:      task.ID,
		Action:      task.Action,
		Description: description,
		Changes:     []string{},
	}

	// Check if content source exists (if specified)
	if contentSourceStr, exists := task.Config["content_source"]; exists {
		contentSourcePath, err := m.processTemplate(contentSourceStr.(string), ctx.Variables)
		if err != nil {
			plan.WillSkip = true
			plan.SkipReason = fmt.Sprintf("Failed to process content_source template: %v", err)
			return plan, nil
		}

		if !filepath.IsAbs(contentSourcePath) {
			contentSourcePath = filepath.Join(ctx.BasePath, contentSourcePath)
		}

		if !utils.FileExists(contentSourcePath) {
			plan.WillSkip = true
			plan.SkipReason = fmt.Sprintf("Content source file does not exist: %s", contentSourcePath)
			return plan, nil
		}
	}

	// Get content to compare (same logic as execution)
	desiredContent := ""

	if contentSourceStr, exists := task.Config["content_source"]; exists {
		contentSourcePath, err := m.processTemplate(contentSourceStr.(string), ctx.Variables)
		if err != nil {
			return nil, fmt.Errorf("failed to process content_source template: %w", err)
		}

		if !filepath.IsAbs(contentSourcePath) {
			contentSourcePath = filepath.Join(ctx.BasePath, contentSourcePath)
		}

		if !utils.FileExists(contentSourcePath) {
			plan.WillSkip = true
			plan.SkipReason = fmt.Sprintf("Content source file does not exist: %s", contentSourcePath)
			return plan, nil
		}

		contentBytes, err := os.ReadFile(contentSourcePath)
		if err != nil {
			plan.WillSkip = true
			plan.SkipReason = fmt.Sprintf("Failed to read content source: %v", err)
			return plan, nil
		}
		desiredContent = string(contentBytes)

		if render, exists := task.Config["render"]; exists && render.(bool) {
			desiredContent, err = m.processTemplateWithPathConversion(desiredContent, ctx.Variables, false)
			if err != nil {
				return nil, fmt.Errorf("failed to render content template: %w", err)
			}
		}
	} else if contentStr, exists := task.Config["content"]; exists {
		if contentString, ok := contentStr.(string); ok {
			desiredContent, err = m.processTemplateWithPathConversion(contentString, ctx.Variables, false)
			if err != nil {
				return nil, fmt.Errorf("failed to process content template: %w", err)
			}
		}
	}

	// Check if file already exists and compare content
	if utils.FileExists(path) {
		existingContent, err := os.ReadFile(path)
		if err != nil {
			plan.Changes = append(plan.Changes, fmt.Sprintf("Failed to read existing file, will recreate: %v", err))
		} else if string(existingContent) == desiredContent {
			plan.WillSkip = true
			plan.SkipReason = "File exists with correct content"
			return plan, nil
		} else {
			plan.Changes = append(plan.Changes, "Update file content")

			if ctx.ShowDiff {
				// Show detailed diff
				diff := utils.GetDetailedDiff(string(existingContent), desiredContent, 20)
				if len(diff) > 0 {
					plan.Changes = append(plan.Changes, "  Content diff:")
					for _, line := range diff {
						plan.Changes = append(plan.Changes, fmt.Sprintf("    %s", line))
					}
				}
			} else {
				// Show diff summary
				diffSummary := utils.GetContentDiffSummary(string(existingContent), desiredContent)
				for _, change := range diffSummary {
					plan.Changes = append(plan.Changes, fmt.Sprintf("  %s", change))
				}
			}
		}
	} else {
		plan.Changes = append(plan.Changes, "Create file")
		// Check if parent directory needs to be created
		parentDir := filepath.Dir(path)
		if !utils.FileExists(parentDir) {
			plan.Changes = append(plan.Changes, fmt.Sprintf("Create parent directory %s", parentDir))
		}
	}

	return plan, nil
}



// ExplainAction returns documentation for a specific action
func (m *FilesModule) ExplainAction(action string) (*modules.ActionDocumentation, error) {
	docs := m.ListActions()
	for _, doc := range docs {
		if doc.Action == action {
			return doc, nil
		}
	}
	return nil, fmt.Errorf("action '%s' not supported by files module", action)
}

// ListActions returns documentation for all actions supported by this module
func (m *FilesModule) ListActions() []*modules.ActionDocumentation {
	return []*modules.ActionDocumentation{
		{
			Action:      "ensure_dir",
			Description: "Ensures a directory exists with the specified permissions. Creates the directory and any necessary parent directories if they don't exist.",
			Parameters: []modules.ActionParameter{
				{
					Name:        "path",
					Type:        "string",
					Required:    true,
					Description: "The path to the directory to create. Supports template variables.",
				},
				{
					Name:        "mode",
					Type:        "string",
					Required:    false,
					Default:     "0755",
					Description: "The file permissions in octal format (Unix/Linux only). On Windows, this parameter is ignored.",
				},
			},
			Examples: []modules.ActionExample{
				{
					Description: "Create a basic directory",
					Config: map[string]interface{}{
						"path": "{{ .paths.home }}/.config/myapp",
					},
				},
				{
					Description: "Create a directory with specific permissions",
					Config: map[string]interface{}{
						"path": "{{ .paths.home }}/.ssh",
						"mode": "0700",
					},
				},
			},
		},
		{
			Action:      "ensure_file",
			Description: "Ensures a file exists with optional content. Creates the file and any necessary parent directories if they don't exist. Content can be provided inline or from a source file, with optional template rendering.",
			Parameters: []modules.ActionParameter{
				{
					Name:        "path",
					Type:        "string",
					Required:    true,
					Description: "The path to the file to create. Supports template variables.",
				},
				{
					Name:        "content",
					Type:        "string",
					Required:    false,
					Default:     "",
					Description: "The content to write to the file. Supports template variables. Mutually exclusive with content_source.",
				},
				{
					Name:        "content_source",
					Type:        "string",
					Required:    false,
					Description: "Path to a file containing the content to write. Relative to dotfiles repository root. Supports template variables. Mutually exclusive with content.",
				},
				{
					Name:        "render",
					Type:        "boolean",
					Required:    false,
					Default:     "false",
					Description: "Whether to process the content from content_source as a template. Only applicable when using content_source. Inline content is always rendered as a template for backward compatibility.",
				},
				{
					Name:        "mode",
					Type:        "string",
					Required:    false,
					Default:     "0644",
					Description: "The file permissions in octal format (Unix/Linux only). On Windows, this parameter is ignored.",
				},
			},
			Examples: []modules.ActionExample{
				{
					Description: "Create an empty file",
					Config: map[string]interface{}{
						"path": "{{ .paths.home }}/.config/myapp/config.txt",
					},
				},
				{
					Description: "Create a file with inline content",
					Config: map[string]interface{}{
						"path":    "{{ .paths.home }}/.gitconfig",
						"content": "[user]\n    name = {{ .user.git_name }}\n    email = {{ .user.git_email }}",
					},
				},
				{
					Description: "Create a file from a template source",
					Config: map[string]interface{}{
						"path":           "{{ .paths.home }}/.ssh/config",
						"content_source": "files/templates/ssh/config.tmpl",
						"render":         true,
						"mode":           "0600",
					},
				},
				{
					Description: "Copy a file without templating",
					Config: map[string]interface{}{
						"path":           "{{ .paths.home }}/.config/app/config.json",
						"content_source": "files/config/app.json",
						"render":         false,
					},
				},
				{
					Description: "Create an executable script",
					Config: map[string]interface{}{
						"path":    "{{ .paths.home }}/bin/myscript.sh",
						"content": "#!/bin/bash\necho 'Hello World'",
						"mode":    "0755",
					},
				},
			},
		},

	}
}

// processTemplate processes a template string with variables
func (m *FilesModule) processTemplate(templateStr string, variables map[string]interface{}) (string, error) {
	return m.processTemplateWithPathConversion(templateStr, variables, true)
}

func (m *FilesModule) processTemplateWithPathConversion(templateStr string, variables map[string]interface{}, convertPaths bool) (string, error) {
	tmpl := template.New("files").Option("missingkey=zero").Funcs(template.FuncMap{
		"pathJoin":  func(paths ...string) string { return filepath.Join(paths...) },
		"pathSep":   func() string { return string(filepath.Separator) },
		"pathClean": func(path string) string { return filepath.Clean(path) },
	})

	tmpl, err := tmpl.Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, variables); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	renderedResult := result.String()

	// Clean up empty lines from template conditionals when processing file content
	if !convertPaths {
		renderedResult = m.cleanupTemplateArtifacts(renderedResult)
	}

	if convertPaths {
		// Ensure OS-specific path separators for file paths
		return filepath.FromSlash(renderedResult), nil
	}
	// Return as-is for content (don't convert path separators)
	return renderedResult, nil
}

// cleanupTemplateArtifacts removes empty lines that are artifacts from template conditionals
func (m *FilesModule) cleanupTemplateArtifacts(content string) string {
	// First, normalize line endings to Unix (LF) to avoid Windows CRLF issues in WSL/Linux
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := strings.Split(content, "\n")
	var cleaned []string

	// Track the context to better identify template artifacts
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			// Check if this empty line is likely a template artifact
			isArtifact := false

			// Look at surrounding lines for template patterns
			if i > 0 && i < len(lines)-1 {
				prevLine := strings.TrimSpace(lines[i-1])
				nextLine := strings.TrimSpace(lines[i+1])

				// If the empty line is between a section header and content, it's likely an artifact
				// Pattern: [section] -> empty line -> content
				if strings.HasPrefix(prevLine, "[") && strings.HasSuffix(prevLine, "]") &&
				   nextLine != "" && !strings.HasPrefix(nextLine, "[") {
					isArtifact = true
				}

				// If there are multiple consecutive empty lines, keep only one
				if prevLine == "" {
					isArtifact = true
				}
			}

			// Remove leading empty lines
			if i == 0 {
				isArtifact = true
			}

			// Remove trailing empty lines (handled at the end)
			if !isArtifact {
				cleaned = append(cleaned, line)
			}
		} else {
			cleaned = append(cleaned, line)
		}
	}

	// Remove trailing empty lines
	for len(cleaned) > 0 && strings.TrimSpace(cleaned[len(cleaned)-1]) == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}

	return strings.Join(cleaned, "\n")
}
