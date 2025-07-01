package files

import (
	"path/filepath"
	"testing"
)

func TestCleanupTemplateArtifacts(t *testing.T) {
	m := &FilesModule{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "removes consecutive empty lines from template conditionals",
			input: `[gpg "ssh"]

    program = C:/Users/menno/AppData/Local/1Password/app/8/op-ssh-sign.exe




`,
			expected: `[gpg "ssh"]
    program = C:/Users/menno/AppData/Local/1Password/app/8/op-ssh-sign.exe`,
		},
		{
			name: "preserves single empty lines",
			input: `[section1]
config1 = value1

[section2]
config2 = value2`,
			expected: `[section1]
config1 = value1

[section2]
config2 = value2`,
		},
		{
			name: "removes multiple consecutive empty lines",
			input: `line1



line2`,
			expected: `line1

line2`,
		},
		{
			name: "removes leading and trailing empty lines",
			input: `

content line


`,
			expected: `content line`,
		},
		{
			name: "handles complex template conditional artifacts",
			input: `[core]
    editor = vim

    autocrlf = input


    fileMode = false

`,
			expected: `[core]
    editor = vim

    autocrlf = input

    fileMode = false`,
		},
		{
			name: "preserves content with no empty lines",
			input: `line1
line2
line3`,
			expected: `line1
line2
line3`,
		},
		{
			name: "handles empty input",
			input: ``,
			expected: ``,
		},
		{
			name: "handles only empty lines",
			input: `



`,
			expected: ``,
		},
		{
			name: "removes empty line between section header and content from template conditionals",
			input: `[gpg "ssh"]

    program = C:/Users/menno/AppData/Local/1Password/app/8/op-ssh-sign.exe

`,
			expected: `[gpg "ssh"]
    program = C:/Users/menno/AppData/Local/1Password/app/8/op-ssh-sign.exe`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.cleanupTemplateArtifacts(tt.input)
			if result != tt.expected {
				t.Errorf("cleanupTemplateArtifacts() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestProcessTemplateWithPathConversion(t *testing.T) {
	m := &FilesModule{}

	tests := []struct {
		name         string
		template     string
		variables    map[string]interface{}
		convertPaths bool
		expected     string
	}{
		{
			name: "cleans up template conditionals in file content",
			template: `[gpg "ssh"]
{{ if eq .Platform.OS "windows" }}
    program = C:/Users/menno/AppData/Local/1Password/app/8/op-ssh-sign.exe
{{ end }}
{{ if eq .Platform.OS "linux" }}
    program = "/usr/bin/op-ssh-sign"
{{ end }}`,
			variables: map[string]interface{}{
				"Platform": map[string]interface{}{
					"OS": "windows",
				},
			},
			convertPaths: false, // This triggers cleanup
			expected: `[gpg "ssh"]
    program = C:/Users/menno/AppData/Local/1Password/app/8/op-ssh-sign.exe`,
		},
		{
			name: "processes path templates with conversion",
			template: `{{.Home}}/config/{{.App}}/config.conf`,
			variables: map[string]interface{}{
				"Home": "C:/Users/test",
				"App":  "myapp",
			},
			convertPaths: true, // This skips cleanup but converts paths
			expected:     filepath.FromSlash("C:/Users/test/config/myapp/config.conf"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := m.processTemplateWithPathConversion(tt.template, tt.variables, tt.convertPaths)
			if err != nil {
				t.Errorf("processTemplateWithPathConversion() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("processTemplateWithPathConversion() = %q, want %q", result, tt.expected)
			}
		})
	}
}
