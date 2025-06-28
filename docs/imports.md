# Import System Guide

The import system allows you to organize your dotfiles configuration across multiple files with support for conditional imports, relative paths, and circular dependency detection.

## Overview

Imports enable you to:
- Split large configurations into manageable files
- Create platform-specific configurations
- Share common configurations across different setups
- Conditionally include files based on environment

## Basic Import Syntax

### Simple Import

```yaml
imports:
  - path: "common.yaml"
```

### Multiple Imports

```yaml
imports:
  - path: "variables/global.yaml"
  - path: "variables/user.yaml"
  - path: "jobs/common.yaml"
```

### Import with Variables

```yaml
imports:
  - path: "platforms/{{ .Platform.OS }}.yaml"
  - path: "shells/{{ .Platform.Shell }}.yaml"
```

## Conditional Imports

Imports can be conditionally included based on platform, environment, or other criteria:

### Platform-Specific Imports

```yaml
imports:
  - path: "platforms/linux.yaml"
    condition: "eq .Platform.OS \"linux\""

  - path: "platforms/windows.yaml"
    condition: "eq .Platform.OS \"windows\""

  - path: "platforms/macos.yaml"
    condition: "eq .Platform.OS \"darwin\""
```

### Environment-Specific Imports

```yaml
imports:
  - path: "environments/work.yaml"
    condition: "eq .Env.ENVIRONMENT \"work\""

  - path: "environments/personal.yaml"
    condition: "eq .Env.ENVIRONMENT \"personal\""
```

### Complex Conditional Imports

```yaml
imports:
  # Alpine Linux specific
  - path: "distros/alpine.yaml"
    condition: "and (eq .Platform.OS \"linux\") (eq .Platform.Distro \"Alpine Linux\")"

  # Unix-like systems
  - path: "unix-common.yaml"
    condition: "or (eq .Platform.OS \"linux\") (eq .Platform.OS \"darwin\")"

  # Elevated/Admin users only
  - path: "admin-tools.yaml"
    condition: "or .Platform.IsElevated .Platform.IsRoot"
```

## Import Types

### Variable Imports

Variable files can import other variable files:

```yaml
# variables/index.yaml
imports:
  - path: "global.yaml"
  - path: "user.yaml"
  - path: "platform-specific/{{ .Platform.OS }}.yaml"

user:
  theme: "dark"
```

### Job Imports

Job files can import other job files:

```yaml
# jobs/index.yaml
imports:
  - path: "packages.yaml"
  - path: "dotfiles.yaml"
  - path: "platform-specific/{{ .Platform.OS }}.yaml"

# Common jobs for all platforms
ensure_dir:
  - "{{ .paths.config }}"
```

## Path Resolution

### Relative Paths

Imports are resolved relative to the importing file's directory:

```
.dotfiles/
├── variables/
│   ├── index.yaml          # imports: ["global.yaml", "user.yaml"]
│   ├── global.yaml
│   ├── user.yaml
│   └── platforms/
│       ├── linux.yaml
│       └── windows.yaml
└── jobs/
    ├── index.yaml          # imports: ["common.yaml"]
    └── common.yaml
```

### Absolute Paths

You can use absolute paths from the dotfiles root:

```yaml
imports:
  - path: "/variables/global.yaml"      # From dotfiles root
  - path: "relative/path.yaml"          # Relative to current file
```

### Template Path Resolution

Paths can use template variables:

```yaml
imports:
  - path: "platforms/{{ .Platform.OS }}.yaml"
  - path: "shells/{{ .Platform.Shell }}/config.yaml"
  - path: "environments/{{ .Env.ENVIRONMENT | default \"default\" }}.yaml"
```

## File Organization Patterns

### Platform-Based Organization

```
.dotfiles/
├── variables/
│   ├── index.yaml
│   ├── global.yaml
│   └── platforms/
│       ├── linux.yaml
│       ├── windows.yaml
│       └── darwin.yaml
└── jobs/
    ├── index.yaml
    ├── common.yaml
    └── platforms/
        ├── linux.yaml
        ├── windows.yaml
        └── darwin.yaml
```

```yaml
# variables/index.yaml
imports:
  - path: "global.yaml"
  - path: "platforms/{{ .Platform.OS }}.yaml"
    condition: "ne .Platform.OS \"\""

# jobs/index.yaml
imports:
  - path: "common.yaml"
  - path: "platforms/{{ .Platform.OS }}.yaml"
    condition: "ne .Platform.OS \"\""
```

### Environment-Based Organization

```
.dotfiles/
├── variables/
│   ├── index.yaml
│   ├── base.yaml
│   └── environments/
│       ├── work.yaml
│       ├── personal.yaml
│       └── development.yaml
```

```yaml
# variables/index.yaml
imports:
  - path: "base.yaml"
  - path: "environments/{{ .Env.PROFILE | default \"personal\" }}.yaml"
```

### Feature-Based Organization

```
.dotfiles/
├── variables/index.yaml
├── jobs/index.yaml
├── features/
│   ├── development/
│   │   ├── variables.yaml
│   │   └── jobs.yaml
│   ├── gaming/
│   │   ├── variables.yaml
│   │   └── jobs.yaml
│   └── work/
│       ├── variables.yaml
│       └── jobs.yaml
```

```yaml
# variables/index.yaml
imports:
  - path: "../features/development/variables.yaml"
    condition: "eq .Env.PROFILE \"developer\""
  - path: "../features/gaming/variables.yaml"
    condition: "eq .Env.ENABLE_GAMING \"true\""

# jobs/index.yaml
imports:
  - path: "../features/development/jobs.yaml"
    condition: "eq .Env.PROFILE \"developer\""
```

## Import Processing

### Processing Order

1. **Parse main file** (e.g., `variables/index.yaml`)
2. **Process imports** in order they appear
3. **Recursively process** imported files
4. **Merge results** with later imports overriding earlier ones

### Variable Precedence

When the same variable is defined in multiple files:

```yaml
# variables/global.yaml
user:
  name: "default"
  theme: "light"

# variables/personal.yaml (imported after global.yaml)
user:
  name: "john"      # Overrides global.yaml
  # theme: "light"  # Inherited from global.yaml
```

Result:
```yaml
user:
  name: "john"      # From personal.yaml
  theme: "light"    # From global.yaml
```

## Error Handling

### Circular Dependency Detection

The import system detects and prevents circular dependencies:

```yaml
# file-a.yaml
imports:
  - path: "file-b.yaml"

# file-b.yaml
imports:
  - path: "file-a.yaml"    # ERROR: Circular dependency
```

### Missing File Handling

```yaml
imports:
  - path: "optional-file.yaml"    # If file doesn't exist, import is skipped
```

### Conditional Import Failures

If a condition evaluation fails, the import is skipped with a warning:

```yaml
imports:
  - path: "platform-specific.yaml"
    condition: "eq .NonExistent.Field \"value\""    # Skipped with warning
```

## Advanced Usage

### Conditional Variables in Imports

```yaml
imports:
  - path: "base.yaml"
  - path: "docker.yaml"
    condition: "contains .Platform.AvailablePackageManagers \"docker\""
  - path: "kubernetes.yaml"
    condition: "and (contains .Platform.AvailablePackageManagers \"kubectl\") (eq .Env.ROLE \"admin\")"
```

### Import with Custom Variables

```yaml
imports:
  - path: "template.yaml"
    variables:
      custom_var: "value"
      override_var: "{{ .Platform.OS }}"
```

### Dynamic Import Paths

```yaml
imports:
  - path: "configs/{{ .user.profile }}/{{ .Platform.OS }}.yaml"
    condition: "and (ne .user.profile \"\") (ne .Platform.OS \"\")"
```

## Best Practices

### 1. Keep Imports at the Top

```yaml
# Good: Imports first
imports:
  - path: "base.yaml"

variables:
  custom: "value"

# Bad: Mixed imports and content
variables:
  custom: "value"
imports:
  - path: "base.yaml"
```

### 2. Use Descriptive Conditions

```yaml
# Good: Clear intent
imports:
  - path: "development-tools.yaml"
    condition: "eq .Env.PROFILE \"developer\""

# Less clear
imports:
  - path: "dev.yaml"
    condition: ".Env.DEV"
```

### 3. Organize by Specificity

```yaml
imports:
  - path: "global.yaml"           # Most general
  - path: "platform.yaml"        # Platform-specific
  - path: "environment.yaml"     # Environment-specific
  - path: "local-overrides.yaml" # Most specific
```

### 4. Handle Missing Files Gracefully

```yaml
imports:
  - path: "required-config.yaml"
  - path: "optional-local.yaml"    # OK if missing
    condition: "ne .Env.SKIP_LOCAL \"true\""
```

## Debugging Imports

### View Import Chain

```bash
# See which files were imported
dotfiles variables sources
```

### Trace Variable Sources

```bash
# See where a variable comes from
dotfiles variables trace user.name
```

### Validate Configuration

```bash
# Check for import errors
dotfiles validate
```

## Common Patterns

### Progressive Enhancement

```yaml
# Start with base, add features progressively
imports:
  - path: "base.yaml"                           # Essential config
  - path: "shell-{{ .Platform.Shell }}.yaml"   # Shell-specific
  - path: "gui.yaml"                            # GUI tools
    condition: "ne .Env.DISPLAY \"\""
  - path: "development.yaml"                    # Dev tools
    condition: "eq .Env.PROFILE \"developer\""
```

### Platform Abstraction

```yaml
# Abstract platform differences
imports:
  - path: "common.yaml"
  - path: "package-managers/{{ index .Platform.AvailablePackageManagers 0 }}.yaml"
    condition: "gt (len .Platform.AvailablePackageManagers) 0"
```

### Environment Inheritance

```yaml
# Base → Environment → Local overrides
imports:
  - path: "base.yaml"
  - path: "environments/{{ .Env.ENVIRONMENT | default \"default\" }}.yaml"
  - path: "local.yaml"
    condition: "ne .Env.SKIP_LOCAL \"true\""
```

## Error Examples and Solutions

### Common Import Errors

```bash
# Error: File not found
Error: failed to process import: file not found: missing-file.yaml
# Solution: Check file path or add condition to make optional

# Error: Circular dependency
Error: circular import detected: file-a.yaml -> file-b.yaml -> file-a.yaml
# Solution: Restructure imports to avoid circular references

# Error: Condition syntax
Error: failed to evaluate import condition: template syntax error
# Solution: Check condition syntax - use parentheses: and (eq .Platform.OS "linux") (eq .Platform.Distro "Alpine")
```

## Related Documentation

- [Condition Syntax](condition-syntax.md) - Complete condition reference
- [Variables System](variables.md) - Variable loading and processing
- [Configuration Reference](config-reference.md) - All configuration options
- [Debugging Guide](DEBUG.md) - Troubleshooting imports
