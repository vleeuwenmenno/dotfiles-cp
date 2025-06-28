# Packages Module

The packages module provides cross-platform package management capabilities for dotfiles. It automatically detects available package managers on your system and can install, uninstall, and manage packages using the most appropriate package manager for your platform.

## Actions

The packages module provides three main actions:

1. **`install_package`** - Install a single package
2. **`uninstall_package`** - Uninstall a single package
3. **`manage_packages`** - Manage multiple packages with different states

### `install_package`

Installs a single package using the system's package manager. The module automatically selects the best available package manager based on your platform and preferences.

**Parameters:**

| Parameter           | Type              | Required | Default | Description                                                                                          |
| ------------------- | ----------------- | -------- | ------- | ---------------------------------------------------------------------------------------------------- |
| `name`              | string            | Yes      | -       | Name of the package to install                                                                       |
| `managers`          | map[string]string | No       | -       | Package manager specific names (e.g., {"winget": "Git.Git", "brew": "git"})                        |
| `prefer`            | []string          | No       | -       | Preferred package manager order (e.g., ["winget", "brew"])                                          |
| `check_system_wide` | boolean           | No       | `false` | Check if command is available system-wide before installing. Skips installation if command exists. |

**Examples:**

```yaml
install_package:
  # Install git package
  - name: "git"

  # Install Node.js with manager-specific names
  - name: "nodejs"
    managers:
      winget: "OpenJS.NodeJS"
      brew: "node"
      apt: "nodejs"
      chocolatey: "nodejs"

  # Install with preferred package manager order
  - name: "python"
    prefer: ["winget", "brew", "apt"]

  # Skip installation if command exists system-wide
  - name: "curl"
    check_system_wide: true
```

### `uninstall_package`

Uninstalls a single package using the system's package manager.

**Parameters:**

| Parameter  | Type              | Required | Default | Description                                                       |
| ---------- | ----------------- | -------- | ------- | ----------------------------------------------------------------- |
| `name`     | string            | Yes      | -       | Name of the package to uninstall                                 |
| `managers` | map[string]string | No       | -       | Package manager specific names                                    |
| `prefer`   | []string          | No       | -       | Preferred package manager order                                   |

**Examples:**

```yaml
uninstall_package:
  # Uninstall git package
  - name: "git"

  # Uninstall with specific manager names
  - name: "old-software"
    managers:
      winget: "Publisher.OldSoftware"
      brew: "old-software"
```

### `manage_packages`

Manages multiple packages with different states (install/uninstall). This is the most flexible action for handling complex package scenarios.

**Parameters:**

| Parameter  | Type     | Required | Default | Description                    |
| ---------- | -------- | -------- | ------- | ------------------------------ |
| `packages` | []object | Yes      | -       | List of package configurations |

Each package object supports:

| Parameter           | Type              | Required | Default     | Description                                                                              |
| ------------------- | ----------------- | -------- | ----------- | ---------------------------------------------------------------------------------------- |
| `name`              | string            | Yes      | -           | Package name                                                                             |
| `state`             | string            | No       | `"present"` | Desired state: "present" (install) or "absent" (uninstall)                              |
| `managers`          | map[string]string | No       | -           | Package manager specific names                                                           |
| `prefer`            | []string          | No       | -           | Preferred package manager order                                                          |
| `check_system_wide` | boolean           | No       | `false`     | Check if command is available system-wide before installing                             |

**Examples:**

```yaml
manage_packages:
  - packages:
      # Install essential development tools
      - name: "git"
        state: "present"

      - name: "nodejs"
        state: "present"
        managers:
          winget: "OpenJS.NodeJS"
          brew: "node"
          apt: "nodejs"

      - name: "python"
        state: "present"
        prefer: ["winget", "brew", "apt"]

      # Remove unwanted software
      - name: "old-editor"
        state: "absent"

      # Install with system-wide check
      - name: "curl"
        state: "present"
        check_system_wide: true
```

## Supported Package Managers

The packages module automatically detects and uses available package managers on your system:

### Windows
- **winget** - Windows Package Manager (recommended)
- **chocolatey** - Community-driven package manager
- **scoop** - Command-line installer

### macOS
- **homebrew** - The missing package manager for macOS

### Linux
- **apt** - Debian/Ubuntu package manager
- **yum** - Red Hat/CentOS package manager (legacy)
- **dnf** - Fedora package manager
- **apk** - Alpine Linux package manager

### Cross-Platform
- **cargo** - Rust package manager (available on all platforms)

## Package Manager Selection

The module uses intelligent package manager selection:

1. **Platform Detection** - Automatically detects your operating system
2. **Availability Check** - Verifies which package managers are installed
3. **Preference Order** - Respects your specified preferences
4. **Fallback Logic** - Uses the best available manager if preferences aren't available

### Default Priority Order

**Windows:**
1. winget
2. chocolatey
3. scoop
4. cargo

**macOS:**
1. homebrew
2. cargo

**Linux:**
1. apt (Debian/Ubuntu)
2. apk (Alpine)
3. dnf (Fedora)
4. yum (RHEL/CentOS)
5. cargo

## Manager-Specific Package Names

Different package managers often use different names for the same software. Use the `managers` parameter to specify the correct name for each manager:

```yaml
install_package:
  - name: "git"  # fallback name
    managers:
      winget: "Git.Git"                    # Full publisher.product format
      chocolatey: "git"                    # Simple name
      brew: "git"                          # Simple name
      apt: "git"                           # Simple name

  - name: "vscode"
    managers:
      winget: "Microsoft.VisualStudioCode"
      chocolatey: "vscode"
      brew: "visual-studio-code"
      apt: "code"
```

## System-Wide Command Check

The `check_system_wide` option allows you to skip package installation if the command is already available system-wide:

```yaml
install_package:
  # Only install git if 'git' command is not found in PATH
  - name: "git"
    check_system_wide: true

  # Always install Node.js through package manager
  - name: "nodejs"
    check_system_wide: false
```

This is useful for:
- **Pre-installed software** - Skip installation of tools that might be pre-installed
- **Manual installations** - Don't reinstall software installed manually
- **System packages** - Avoid conflicts with system-provided packages

## Package States

### Present (Install)
Ensures a package is installed:

```yaml
manage_packages:
  - packages:
      - name: "git"
        state: "present"  # Will install if not present
```

### Absent (Uninstall)
Ensures a package is not installed:

```yaml
manage_packages:
  - packages:
      - name: "unwanted-software"
        state: "absent"  # Will uninstall if present
```

## Template Support

All string parameters support Go template syntax with access to your variables:

```yaml
install_package:
  # Use variables in package selection
  - name: "{{ .user.preferred_editor }}"

  # Platform-specific packages
  - name: "{{ if eq .platform.os \"darwin\" }}brew{{ else }}curl{{ end }}"

  # Dynamic manager preferences based on platform
  - name: "git"
    prefer: ["{{ .package_manager.preferred }}"]
```

## Error Handling

The packages module provides robust error handling:

### Package Not Found
```
Error: failed to install package 'nonexistent-package': package not found
```

**Solutions:**
- Check package name spelling
- Verify package exists in the selected package manager
- Use `managers` parameter with correct names for each manager

### Permission Denied
```
Error: failed to install package 'git': permission denied
```

**Solutions:**
- Run dotfiles manager with appropriate permissions
- Ensure package manager has necessary privileges
- Some package managers (apt, yum, dnf) require sudo access

### Package Manager Not Available
```
Error: no package managers available on this system
```

**Solutions:**
- Install a supported package manager for your platform
- Check if package managers are in your PATH
- Verify package manager installations

## Best Practices

### 1. Use Manager-Specific Names

```yaml
install_package:
  # Good: Specify exact package names for each manager
  - name: "nodejs"
    managers:
      winget: "OpenJS.NodeJS"
      brew: "node"
      apt: "nodejs"
      chocolatey: "nodejs"
```

### 2. Group Related Packages

```yaml
manage_packages:
  - packages:
      # Development tools
      - name: "git"
        state: "present"
      - name: "nodejs"
        state: "present"
      - name: "python"
        state: "present"

      # Cleanup old packages
      - name: "old-tool"
        state: "absent"
```

### 3. Set Reasonable Preferences

```yaml
install_package:
  # Windows: prefer winget, fallback to chocolatey
  - name: "git"
    prefer: ["winget", "chocolatey"]

  # macOS: use homebrew
  - name: "git"
    prefer: ["homebrew"]
```

### 4. Use System-Wide Checks Judiciously

```yaml
install_package:
  # Good: Check for commonly pre-installed tools
  - name: "curl"
    check_system_wide: true

  # Bad: Don't check for tools you want managed by package manager
  - name: "nodejs"
    check_system_wide: false  # Explicit is better
```

### 5. Handle Platform Differences

```yaml
manage_packages:
  - packages:
      # Cross-platform packages
      - name: "git"
        state: "present"

      # Platform-specific packages
      - name: "{{ if eq .platform.os \"darwin\" }}mas{{ else if eq .platform.os \"linux\" }}snapd{{ end }}"
        state: "present"
        condition: 'ne .platform.os "windows"'
```

## Integration with Other Modules

The packages module works well with other modules:

### With Files Module

```yaml
# Install application first
install_package:
  - name: "vim"

# Then configure it
ensure_file:
  - path: "{{ .paths.home }}/.vimrc"
    content_source: "files/config/vim/vimrc.tmpl"
    render: true
```

### With Symlinks Module

```yaml
# Install package manager
install_package:
  - name: "nodejs"

# Install global packages
install_package:
  - name: "typescript"
    managers:
      cargo: "typescript"  # Example: install via npm/cargo

# Symlink configurations
symlink:
  - src: "files/config/npm/npmrc"
    dst: "{{ .paths.home }}/.npmrc"
```

## Wildcard Package Support

The packages module supports wildcard patterns for package names (useful for cleanup):

```yaml
uninstall_package:
  # Remove all packages matching pattern
  - name: "old-app-*"

manage_packages:
  - packages:
      # Remove all development tools matching pattern
      - name: "dev-tool-*"
        state: "absent"
```

**Note:** Wildcard support varies by package manager and is primarily useful for uninstallation.

## Caching and Performance

The packages module includes intelligent caching:

- **Package List Caching** - Installed package lists are cached for 5 minutes
- **Batch Operations** - Multiple package checks use a single command when possible
- **Platform Optimization** - Uses the most efficient commands for each package manager

This makes checking large numbers of packages much faster than individual commands.

## Troubleshooting

### Debug Package Detection

Use the `--verbose` flag to see detailed package manager selection:

```bash
dotfiles apply --verbose --dry-run
```

### Check Available Package Managers

```bash
# Check what package managers are detected
dotfiles info
```

### Force Specific Package Manager

```yaml
install_package:
  - name: "git"
    prefer: ["chocolatey"]  # Force chocolatey even if winget is available
```

### Package Manager Issues

1. **Update package manager databases** before running dotfiles
2. **Check package manager configuration** (repositories, sources)
3. **Verify network connectivity** for package downloads
4. **Check available disk space** for installations

This module provides a robust foundation for managing software installations across different platforms in your dotfiles setup.
