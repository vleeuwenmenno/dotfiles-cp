package filters

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/flosch/pongo2/v6"
)

// OnePasswordFilter provides 1Password CLI integration for Pongo2 templates
type OnePasswordFilter struct{}

// NewOnePasswordFilter creates a new 1Password filter
func NewOnePasswordFilter() *OnePasswordFilter {
	return &OnePasswordFilter{}
}

// Register registers the 1Password functions with the given template set
func (f *OnePasswordFilter) Register(templateSet *pongo2.TemplateSet) {
	// Register op_read as a global function
	templateSet.Globals["op_read"] = f.readSecret
}

// readSecret executes the 1Password CLI to read a secret
func (f *OnePasswordFilter) readSecret(reference string) (string, error) {
	if reference == "" {
		return "", fmt.Errorf("1Password reference cannot be empty")
	}

	// Check if op CLI is available
	if _, err := exec.LookPath("op"); err != nil {
		return "", fmt.Errorf("1Password CLI (op) not found in PATH: %w", err)
	}

	// Execute op read command
	cmd := exec.Command("op", "read", reference)
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("1Password CLI error for '%s': %s", reference, string(exitError.Stderr))
		}
		return "", fmt.Errorf("failed to execute 1Password CLI for '%s': %w", reference, err)
	}

	// Return the secret value, trimming any trailing whitespace
	return strings.TrimSpace(string(output)), nil
}

// GetSyntaxHelp returns help text for the 1Password filter
func (f *OnePasswordFilter) GetSyntaxHelp() string {
	return `1Password Integration:
  {{ op_read("op://Private/Signulous/password") }}
  {{ op_read("op://Vault/Item/field") }}
  {{ op_read("op://Personal/GitHub/token") }}

Examples:
  IdentityFile {{ op_read("op://Private/SSH/private_key") }}
  Password={{ op_read("op://Work/Database/password") }}
`
}
