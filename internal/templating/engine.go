package templating

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/flosch/pongo2/v6"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/templating/filters"
)

// TemplatingEngine provides hybrid templating:
// - Expr for simple conditions (fast, type-safe)
// - Pongo2 for complex templating (full Jinja2-like power)
type TemplatingEngine struct {
	// Expr for conditions and simple expressions
	exprPrograms map[string]*vm.Program

	// Pongo2 for complex templating
	pongo2Set *pongo2.TemplateSet

	// Base path for template resolution
	basePath string
}

// NewTemplatingEngine creates a new hybrid templating engine
func NewTemplatingEngine(basePath string) *TemplatingEngine {
	// Use a safe base path
	safeBasePath := basePath
	if safeBasePath == "" {
		safeBasePath = "."
	}

	// Ensure the directory exists or use current directory
	if _, err := os.Stat(safeBasePath); os.IsNotExist(err) {
		safeBasePath = "."
	}

	// Create loader safely
	loader, err := pongo2.NewLocalFileSystemLoader(safeBasePath)
	if err != nil {
		// Fallback to current directory if there's an issue
		loader = pongo2.MustNewLocalFileSystemLoader(".")
	}

	engine := &TemplatingEngine{
		exprPrograms: make(map[string]*vm.Program),
		pongo2Set:    pongo2.NewSet("dotfiles", loader),
		basePath:     safeBasePath,
	}

	// Register custom filters
	engine.registerFilters()

	return engine
}

// EvaluateCondition evaluates job conditions using Expr
// Perfect for simple, fast boolean conditions
// Examples: Platform.OS == "linux" && !Platform.IsElevated
func (e *TemplatingEngine) EvaluateCondition(condition string, variables map[string]interface{}) (bool, error) {
	if condition == "" {
		return true, nil
	}

	program, err := e.getOrCompileExpr(condition, expr.AsBool())
	if err != nil {
		return false, fmt.Errorf("failed to compile condition '%s': %w", condition, err)
	}

	result, err := expr.Run(program, variables)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate condition '%s': %w", condition, err)
	}

	if boolResult, ok := result.(bool); ok {
		return boolResult, nil
	}

	return false, fmt.Errorf("condition '%s' did not evaluate to boolean, got %T", condition, result)
}

// ProcessTemplate processes complex templates using Pongo2
// Perfect for file templating with full Jinja2-like syntax
// Examples: {% if Platform.OS == "linux" %}...{% endif %}, {% for item in list %}...{% endfor %}
func (e *TemplatingEngine) ProcessTemplate(templateContent string, variables map[string]interface{}) (string, error) {
	template, err := e.pongo2Set.FromString(templateContent)
	if err != nil {
		return "", e.enhanceTemplateError(err, templateContent, "<inline template>")
	}

	result, err := template.Execute(pongo2.Context(variables))
	if err != nil {
		return "", e.enhanceTemplateError(err, templateContent, "<inline template>")
	}

	return result, nil
}

// ProcessTemplateFile processes a template file using Pongo2
// Perfect for ensure_file and other file-based templating
func (e *TemplatingEngine) ProcessTemplateFile(templatePath string, variables map[string]interface{}) (string, error) {
	// Read template content for better error reporting
	templateContent := ""
	if contentBytes, err := os.ReadFile(templatePath); err == nil {
		templateContent = string(contentBytes)
	}

	template, err := e.pongo2Set.FromFile(templatePath)
	if err != nil {
		return "", e.enhanceTemplateError(err, templateContent, templatePath)
	}

	result, err := template.Execute(pongo2.Context(variables))
	if err != nil {
		return "", e.enhanceTemplateError(err, templateContent, templatePath)
	}

	return result, nil
}

// ProcessVariableTemplate processes variable templates using Pongo2
// Used in the pillar-like variable system for complex variable processing
func (e *TemplatingEngine) ProcessVariableTemplate(templateStr string, variables map[string]interface{}) (string, error) {
	return e.ProcessTemplate(templateStr, variables)
}

// getOrCompileExpr compiles and caches Expr programs for conditions
func (e *TemplatingEngine) getOrCompileExpr(expression string, options ...expr.Option) (*vm.Program, error) {
	cacheKey := fmt.Sprintf("%s:%d", expression, len(options))

	if program, exists := e.exprPrograms[cacheKey]; exists {
		return program, nil
	}

	// Use built-in Expr operators and options
	allOptions := append([]expr.Option{
		expr.AllowUndefinedVariables(),
	}, options...)

	program, err := expr.Compile(expression, allOptions...)
	if err != nil {
		return nil, err
	}

	e.exprPrograms[cacheKey] = program
	return program, nil
}

// IsTemplateContent checks if a string contains Pongo2 template syntax
func (e *TemplatingEngine) IsTemplateContent(content string) bool {
	return strings.Contains(content, "{{") || strings.Contains(content, "{%")
}

// enhanceTemplateError provides detailed error information with context
func (e *TemplatingEngine) enhanceTemplateError(err error, templateContent, templatePath string) error {
	errStr := err.Error()

	// Extract line and column information from Pongo2 error
	lineNum, colNum := e.extractErrorPosition(errStr)

	if lineNum > 0 && templateContent != "" {
		lines := strings.Split(templateContent, "\n")

		// Build enhanced error message
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("template error in '%s':\n", templatePath))
		errorMsg.WriteString(fmt.Sprintf("  %s\n\n", errStr))

		// Show context around the error
		if lineNum <= len(lines) {
			errorMsg.WriteString("Context:\n")

			// Show 2 lines before, the error line, and 2 lines after
			start := max(1, lineNum-2)
			end := min(len(lines), lineNum+2)

			for i := start; i <= end; i++ {
				line := lines[i-1] // Convert to 0-based index
				prefix := "  "
				if i == lineNum {
					prefix = "→ " // Arrow pointing to error line
				}
				errorMsg.WriteString(fmt.Sprintf("%s%3d: %s\n", prefix, i, line))

				// Add column indicator for error line
				if i == lineNum && colNum > 0 {
					spaces := strings.Repeat(" ", colNum+6) // Account for line number and prefix
					errorMsg.WriteString(fmt.Sprintf("     %s^\n", spaces))
				}
			}
		}

		// Add helpful suggestions
		errorMsg.WriteString("\nCommon fixes:\n")
		if strings.Contains(errStr, "'}}' expected") {
			errorMsg.WriteString("  • Check for unmatched template brackets {{ }} or {% %}\n")
			errorMsg.WriteString("  • Avoid nesting {{ }} inside {% %} expressions\n")
		}
		if strings.Contains(errStr, "items()") {
			errorMsg.WriteString("  • Use 'for key, value in dict.items()' for dictionary iteration\n")
		}
		if strings.Contains(errStr, "get(") {
			errorMsg.WriteString("  • Use 'if dict.key' instead of 'dict.get(key)' in Pongo2\n")
		}

		return fmt.Errorf(errorMsg.String())
	}

	// Fallback for cases where we can't extract detailed info
	return fmt.Errorf("template error in '%s': %w", templatePath, err)
}

// extractErrorPosition extracts line and column numbers from Pongo2 error messages
func (e *TemplatingEngine) extractErrorPosition(errStr string) (int, int) {
	// Pongo2 errors typically contain "Line X Col Y" patterns
	lineRegex := regexp.MustCompile(`Line (\d+)`)
	colRegex := regexp.MustCompile(`Col (\d+)`)

	var lineNum, colNum int

	if lineMatch := lineRegex.FindStringSubmatch(errStr); len(lineMatch) > 1 {
		lineNum, _ = strconv.Atoi(lineMatch[1])
	}

	if colMatch := colRegex.FindStringSubmatch(errStr); len(colMatch) > 1 {
		colNum, _ = strconv.Atoi(colMatch[1])
	}

	return lineNum, colNum
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// registerFilters registers all custom filters with the template set
func (e *TemplatingEngine) registerFilters() {
	// Register 1Password filter
	onePasswordFilter := filters.NewOnePasswordFilter()
	onePasswordFilter.Register(e.pongo2Set)
}

// GetSyntaxHelp returns help text for users about syntax
func (e *TemplatingEngine) GetSyntaxHelp() string {
	onePasswordFilter := filters.NewOnePasswordFilter()

	return `Templating Syntax:

Job Conditions (Expr):
  Platform.OS == "linux"
  Platform.OS == "linux" && !Platform.IsElevated
  "docker" in Platform.Tags
  Platform.Distro matches "Ubuntu"
  Platform.Version matches "^22\\."

File Templates (Pongo2/Jinja2):
  {{ Platform.OS }}
  {% if Platform.OS == "linux" %}...{% endif %}
  {% for pkg in Packages %}{{ pkg }}{% endfor %}
  {{ Platform.OS|upper }}

Variable Templates (Pongo2/Jinja2):
  {{ Platform.OS }}-config
  {% if Platform.IsElevated %}admin{% else %}user{% endif %}

` + onePasswordFilter.GetSyntaxHelp()
}
