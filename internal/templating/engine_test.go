package templating

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplatingEngine_EvaluateCondition(t *testing.T) {
	engine := NewTemplatingEngine(t.TempDir())

	variables := map[string]interface{}{
		"Platform": map[string]interface{}{
			"OS":         "linux",
			"Distro":     "Alpine Linux",
			"IsElevated": false,
			"IsRoot":     true,
			"Tags":       []string{"docker", "alpine"},
			"Version":    "3.18",
		},
		"User": map[string]interface{}{
			"Name": "testuser",
		},
	}

	tests := []struct {
		name        string
		condition   string
		expected    bool
		shouldError bool
	}{
		{
			name:      "simple equality",
			condition: `Platform.OS == "linux"`,
			expected:  true,
		},
		{
			name:      "simple inequality",
			condition: `Platform.OS != "windows"`,
			expected:  true,
		},
		{
			name:      "boolean negation",
			condition: `!Platform.IsElevated`,
			expected:  true,
		},
		{
			name:      "complex AND condition",
			condition: `Platform.OS == "linux" && Platform.Distro == "Alpine Linux"`,
			expected:  true,
		},
		{
			name:      "complex OR condition",
			condition: `Platform.OS == "windows" || Platform.OS == "linux"`,
			expected:  true,
		},
		{
			name:      "nested map access",
			condition: `User.Name == "testuser"`,
			expected:  true,
		},
		{
			name:      "array contains using in operator",
			condition: `"docker" in Platform.Tags`,
			expected:  true,
		},
		{
			name:      "built-in matches operator",
			condition: `Platform.Distro matches "Alpine"`,
			expected:  true,
		},
		{
			name:      "built-in matches with regex",
			condition: `Platform.Version matches "^3\\."`,
			expected:  true,
		},
		{
			name:      "empty condition should return true",
			condition: "",
			expected:  true,
		},
		{
			name:        "invalid condition should error",
			condition:   "Platform.OS ==",
			shouldError: true,
		},
		{
			name:        "non-boolean result should error",
			condition:   "Platform.OS",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.EvaluateCondition(tt.condition, variables)

			if tt.shouldError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplatingEngine_ProcessTemplate(t *testing.T) {
	engine := NewTemplatingEngine(t.TempDir())

	variables := map[string]interface{}{
		"Platform": map[string]interface{}{
			"OS":   "linux",
			"Arch": "amd64",
		},
		"User": map[string]interface{}{
			"Name":    "testuser",
			"IsAdmin": false,
		},
		"Packages": []string{"git", "vim", "curl"},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "simple variable substitution",
			template: "Hello {{ User.Name }}!",
			expected: "Hello testuser!",
		},
		{
			name:     "conditional block",
			template: "{% if Platform.OS == 'linux' %}Linux detected{% else %}Not Linux{% endif %}",
			expected: "Linux detected",
		},
		{
			name:     "loop over array",
			template: "Packages:{% for pkg in Packages %} {{ pkg }}{% if not forloop.last %},{% endif %}{% endfor %}",
			expected: "Packages: git, vim, curl,",
		},
		{
			name:     "nested conditions",
			template: "{% if Platform.OS == 'linux' %}{% if User.IsAdmin %}Admin on Linux{% else %}User on Linux{% endif %}{% endif %}",
			expected: "User on Linux",
		},
		{
			name:     "filters",
			template: "OS: {{ Platform.OS|upper }}",
			expected: "OS: LINUX",
		},
		{
			name:     "string concatenation",
			template: "{{ Platform.OS }}-{{ Platform.Arch }}",
			expected: "linux-amd64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.ProcessTemplate(tt.template, variables)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplatingEngine_ProcessTemplateFile(t *testing.T) {
	// Create temporary directory and file
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "test.template")

	templateContent := `# Configuration for {{ Platform.OS }}
user: {{ User.Name }}
{% if Platform.OS == "linux" -%}
shell: /bin/bash
{% else -%}
shell: /bin/sh
{% endif -%}
packages:
{% for pkg in Packages -%}
  - {{ pkg }}
{% endfor %}`

	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	engine := NewTemplatingEngine(tempDir)

	variables := map[string]interface{}{
		"Platform": map[string]interface{}{
			"OS": "linux",
		},
		"User": map[string]interface{}{
			"Name": "testuser",
		},
		"Packages": []string{"git", "vim"},
	}

	result, err := engine.ProcessTemplateFile("test.template", variables)
	require.NoError(t, err)

	expected := `# Configuration for linux
user: testuser
shell: /bin/bash
packages:
- git
- vim
`

	assert.Equal(t, expected, result)
}

func TestTemplatingEngine_ProcessVariableTemplate(t *testing.T) {
	engine := NewTemplatingEngine(t.TempDir())

	variables := map[string]interface{}{
		"Platform": map[string]interface{}{
			"OS": "linux",
		},
		"Version": "1.0.0",
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "simple variable substitution",
			template: "{{ Platform.OS }}",
			expected: "linux",
		},
		{
			name:     "string concatenation",
			template: "{{ Platform.OS }}-config",
			expected: "linux-config",
		},
		{
			name:     "complex conditional",
			template: "{% if Platform.OS == 'linux' %}linux-config{% else %}default-config{% endif %}",
			expected: "linux-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.ProcessVariableTemplate(tt.template, variables)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplatingEngine_IsTemplateContent(t *testing.T) {
	engine := NewTemplatingEngine(t.TempDir())

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "pongo2 variable",
			content:  "{{ variable }}",
			expected: true,
		},
		{
			name:     "pongo2 tag",
			content:  "{% if condition %}",
			expected: true,
		},
		{
			name:     "plain text",
			content:  "just plain text",
			expected: false,
		},
		{
			name:     "mixed content",
			content:  "Hello {{ name }}, welcome!",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.IsTemplateContent(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplatingEngine_Caching(t *testing.T) {
	engine := NewTemplatingEngine(t.TempDir())

	variables := map[string]interface{}{
		"Platform": map[string]interface{}{
			"OS": "linux",
		},
	}

	// Evaluate the same condition multiple times
	condition := `Platform.OS == "linux"`

	for i := 0; i < 3; i++ {
		result, err := engine.EvaluateCondition(condition, variables)
		require.NoError(t, err)
		assert.True(t, result)
	}

	// Verify that the program was cached (should have 1 entry)
	assert.Len(t, engine.exprPrograms, 1)
}

func TestTemplatingEngine_GetSyntaxHelp(t *testing.T) {
	engine := NewTemplatingEngine(t.TempDir())
	help := engine.GetSyntaxHelp()

	// Just verify it returns non-empty help text
	assert.NotEmpty(t, help)
	assert.Contains(t, help, "Job Conditions")
	assert.Contains(t, help, "File Templates")
	assert.Contains(t, help, "Variable Templates")
}
