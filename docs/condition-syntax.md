# Condition Syntax Guide

This document explains how to write conditions for imports and tasks in your dotfiles configuration.

## Overview

Conditions allow you to conditionally include imports or execute tasks based on platform information, environment variables, or other criteria. They use Go's template syntax with custom functions.

## Basic Syntax

All conditions are wrapped in template syntax and evaluated as boolean expressions:

```yaml
condition: "eq .Platform.OS \"linux\""
```

## Available Functions

### Comparison Functions

- `eq a b` - Returns true if a equals b
- `ne a b` - Returns true if a does not equal b

### Boolean Functions

- `and a b` - Returns true if both a and b are true
- `or a b` - Returns true if either a or b is true
- `not a` - Returns true if a is false

### Platform Variables

Available platform variables include:

- `.Platform.OS` - Operating system (windows, linux, darwin)
- `.Platform.Arch` - Architecture (amd64, arm64, etc.)
- `.Platform.Distro` - Distribution name (Windows, Ubuntu, Alpine Linux, etc.)
- `.Platform.Shell` - Current shell (bash, zsh, powershell, etc.)
- `.Platform.IsElevated` - Boolean: running with elevated privileges
- `.Platform.IsRoot` - Boolean: running as root (Unix-like systems)
- `.Platform.AvailablePackageManagers` - Array of available package managers

## Examples

### Simple Conditions

```yaml
# Only on Linux
condition: "eq .Platform.OS \"linux\""

# Only on Windows
condition: "eq .Platform.OS \"windows\""

# Only when elevated
condition: ".Platform.IsElevated"

# Only when NOT elevated
condition: "not .Platform.IsElevated"
```

### AND Conditions

```yaml
# Linux AND Alpine
condition: "and (eq .Platform.OS \"linux\") (eq .Platform.Distro \"Alpine Linux\")"

# Windows AND elevated
condition: "and (eq .Platform.OS \"windows\") .Platform.IsElevated"

# Linux AND NOT root
condition: "and (eq .Platform.OS \"linux\") (not .Platform.IsRoot)"
```

### OR Conditions

```yaml
# Linux OR macOS
condition: "or (eq .Platform.OS \"linux\") (eq .Platform.OS \"darwin\")"

# Ubuntu OR Debian
condition: "or (eq .Platform.Distro \"Ubuntu\") (eq .Platform.Distro \"Debian\")"

# Elevated OR root
condition: "or .Platform.IsElevated .Platform.IsRoot"
```

### Complex Nested Conditions

```yaml
# Unix-like systems with specific shells
condition: "and (or (eq .Platform.OS \"linux\") (eq .Platform.OS \"darwin\")) (or (eq .Platform.Shell \"bash\") (eq .Platform.Shell \"zsh\"))"

# Windows elevated OR Unix root
condition: "or (and (eq .Platform.OS \"windows\") .Platform.IsElevated) (and (ne .Platform.OS \"windows\") .Platform.IsRoot)"
```

## Common Patterns

### Platform-Specific Imports

```yaml
imports:
  - path: platforms/linux.yaml
    condition: "eq .Platform.OS \"linux\""

  - path: platforms/windows.yaml
    condition: "eq .Platform.OS \"windows\""

  - path: distros/alpine.yaml
    condition: "and (eq .Platform.OS \"linux\") (eq .Platform.Distro \"Alpine Linux\")"
```

### Shell-Specific Tasks

```yaml
ensure_file:
  - path: "{{ .Platform.HomeDir }}/.bashrc"
    content_source: "files/bashrc"
    condition: "eq .Platform.Shell \"bash\""

  - path: "{{ .Platform.HomeDir }}/.zshrc"
    content_source: "files/zshrc"
    condition: "eq .Platform.Shell \"zsh\""
```

### Privilege-Based Tasks

```yaml
install_package:
  - name: "docker"
    condition: "or .Platform.IsElevated .Platform.IsRoot"

  - name: "user-tool"
    condition: "not (or .Platform.IsElevated .Platform.IsRoot)"
```

## Important Syntax Rules

### ✅ Correct Syntax

```yaml
# Use parentheses around function calls in boolean operations
condition: "and (eq .Platform.OS \"linux\") (eq .Platform.Distro \"Alpine\")"

# Boolean fields can be used directly
condition: ".Platform.IsElevated"

# Negate with not
condition: "not .Platform.IsRoot"
```

### ❌ Incorrect Syntax

```yaml
# Missing parentheses - will cause parse errors
condition: "eq .Platform.OS \"linux\" and eq .Platform.Distro \"Alpine\""

# Incorrect boolean syntax
condition: ".Platform.OS == \"linux\""

# Missing quotes around string values
condition: "eq .Platform.OS linux"
```

## Error Messages

If you see a condition parse error, check for:

1. **Missing parentheses** around function calls
2. **Missing quotes** around string values
3. **Incorrect function names** (use `eq`, not `==`)
4. **Wrong number of arguments** to functions

## Testing Conditions

You can test conditions by using the variables command to see available platform information:

```bash
# See all platform variables
dotfiles variables get Platform

# Test specific values
dotfiles variables get Platform.OS
dotfiles variables get Platform.Distro
```

## Advanced Usage

### Multiple Conditions

You can combine multiple conditions in complex ways:

```yaml
condition: "and (or (eq .Platform.OS \"linux\") (eq .Platform.OS \"darwin\")) (and (eq .Platform.Arch \"amd64\") (not .Platform.IsElevated))"
```

This condition is true when:
- Platform is Linux OR macOS
- AND architecture is amd64
- AND NOT running with elevated privileges

### Environment Variables

You can also reference environment variables in conditions:

```yaml
condition: "eq .Env.USER \"developer\""
condition: "and (eq .Platform.OS \"linux\") (ne .Env.HOME \"\")"
```
