# Packages Module

The packages module provides cross-platform package management for your dotfiles. It can install, uninstall, and manage packages across different operating systems using their native package managers.

## Supported Package Managers

### Windows
- **winget** - Windows Package Manager
- **chocolatey** - Community-driven package manager
- **scoop** - Command-line installer

### macOS
- **homebrew** - The missing package manager for macOS
- **macports** - Open-source community initiative

### Linux
- **apt** - Advanced Package Tool (Debian/Ubuntu)
- **yum** - Yellowdog Updater Modified (RHEL/CentOS)
- **dnf** - Dandified YUM (Fedora)
- **pacman** - Package Manager (Arch Linux)
- **zypper** - Package manager (openSUSE)
- **portage** - Package management system (Gentoo)
- **xbps** - X Binary Package System (Void Linux)
- **apk** - Alpine Package Keeper (Alpine Linux)

## Actions

### install_package

Install a single package using the system's package manager.

**Parameters:**
- `name` (string, required) - Name of the package to install
- `managers` (map[string]string, optional) - Package manager specific names
- `prefer` ([]string, optional) - Preferred package manager order

**Examples:**

```yaml
# Simple install
install_package:
  name: git

# Install with manager-specific names
install_package:
  name: nodejs
  managers:
    winget: "OpenJS.NodeJS"
    brew: "node"
    apt: "nodejs"
    choco: "nodejs"

# Install with package manager preference
install_package:
  name: curl
  prefer: ["winget", "brew", "apt"]
```

### uninstall_package

Uninstall a single package using the system's package manager.

**Parameters:**
- `name` (string, required) - Name of the package to uninstall
- `managers` (map[string]string, optional) - Package manager specific names
- `prefer` ([]string, optional) - Preferred package manager order

**Examples:**

```yaml
# Simple uninstall
uninstall_package:
  name: old-software

# Uninstall with manager-specific names
uninstall_package:
  name: nodejs
  managers:
    winget: "OpenJS.NodeJS"
    brew: "node"
    apt: "nodejs"
```

### manage_packages

Manage multiple packages with different states (install/uninstall).

**Parameters:**
- `packages` ([]object, required) - List of package configurations

Each package object supports:
- `name` (string, required) - Package name
- `state` (string, optional) - "present" (default) or "absent"
- `managers` (map[string]string, optional) - Manager-specific names
- `prefer` ([]string, optional) - Preferred package manager order

**Examples:**

```yaml
# Manage multiple packages
manage_packages:
  packages:
    # Install packages
    - name: git
      state: present
    - name: curl
      state: present
    - name: nodejs
      state: present
      managers:
        winget: "OpenJS.NodeJS"
        brew: "node"
        apt: "nodejs"

    # Remove packages
    - name: old-package
      state: absent
    - name: unwanted-tool
      state: absent

# Simple format (assumes present state)
manage_packages:
  packages:
    - name: git
    - name: curl
    - name: vim
```

## Package Manager Selection

The module automatically selects the best available package manager based on:

1. **Preference order** - If `prefer` is specified, it tries managers in that order
2. **Availability** - Only considers package managers that are installed on the system
3. **Fallback** - Uses the first available package manager if no preferences match

## Package Name Resolution

Package names can vary between different package managers. The module handles this through:

1. **Manager-specific names** - Use the `managers` map to specify different names per package manager
2. **Fallback to generic name** - If no specific name is found, uses the `name` field

Example of different package names:
```yaml
manage_packages:
  packages:
    - name: nodejs  # Generic name
      managers:
        winget: "OpenJS.NodeJS"      # Official Microsoft Store ID
        choco: "nodejs"              # Chocolatey package name
        brew: "node"                 # Homebrew formula name
        apt: "nodejs"                # Debian package name
        pacman: "nodejs"             # Arch package name
```

## Usage in Job Files

### Basic Setup

```yaml
# jobs/development.yaml
name: "Development Tools"
description: "Install essential development packages"

tasks:
  - action: manage_packages
    config:
      packages:
        - name: git
        - name: curl
        - name: vim
        - name: nodejs
          managers:
            winget: "OpenJS.NodeJS"
            brew: "node"
```

### Platform-Specific Packages

```yaml
# jobs/platform-tools.yaml
name: "Platform-specific Tools"
description: "Install tools specific to each platform"

tasks:
  # Windows-specific tools
  - action: manage_packages
    when: "{{ eq .Platform.OS \"windows\" }}"
    config:
      packages:
        - name: powertoys
          managers:
            winget: "Microsoft.PowerToys"
            choco: "powertoys"
        - name: windows-terminal
          managers:
            winget: "Microsoft.WindowsTerminal"

  # macOS-specific tools
  - action: manage_packages
    when: "{{ eq .Platform.OS \"darwin\" }}"
    config:
      packages:
        - name: rectangle
          managers:
            brew: "rectangle"
        - name: iterm2
          managers:
            brew: "iterm2"

  # Linux-specific tools
  - action: manage_packages
    when: "{{ eq .Platform.OS \"linux\" }}"
    config:
      packages:
        - name: htop
        - name: tmux
        - name: zsh
```

### Cleanup Jobs

```yaml
# jobs/cleanup.yaml
name: "Package Cleanup"
description: "Remove unwanted packages"

tasks:
  - action: manage_packages
    config:
      packages:
        - name: unwanted-bloatware
          state: absent
        - name: old-version-software
          state: absent
```

## Dry Run Support

The packages module fully supports dry-run mode. Use `--dry-run` to see what packages would be installed or removed:

```bash
dotfiles apply --dry-run
```

Example output:
```
[1/3] manage_packages: Development Tools (manage_packages)
   📋 Would do:
      - Install package git using winget
      - Install package nodejs using winget
      - Package already installed: curl
```

## Error Handling

The module provides detailed error messages for common issues:

- **No package managers available** - When no supported package managers are found
- **Package not found** - When a package doesn't exist in the selected manager
- **Permission errors** - When elevated privileges are required
- **Network errors** - When package repositories are unreachable

## Best Practices

### 1. Use Manager-Specific Names
Always specify manager-specific names for packages that have different identifiers:

```yaml
- name: docker
  managers:
    winget: "Docker.DockerDesktop"
    brew: "docker"
    apt: "docker.io"  # Note: not just "docker"
```

### 2. Set Package Manager Preferences
Specify preferred package managers for consistent behavior:

```yaml
- name: git
  prefer: ["winget", "brew", "apt"]  # Prefer official/primary managers
```

### 3. Group Related Packages
Use `manage_packages` to group related packages together:

```yaml
# Development tools
manage_packages:
  packages:
    - name: git
    - name: nodejs
    - name: python

# Design tools
manage_packages:
  packages:
    - name: figma
    - name: sketch
```

### 4. Use Conditional Installation
Install platform-specific packages using template conditions:

```yaml
- action: manage_packages
  when: "{{ eq .Platform.OS \"windows\" }}"
  config:
    packages:
      - name: powershell-core
```

### 5. Handle Package Updates
The module focuses on installation/removal. For updates, create separate jobs:

```yaml
# Note: This is conceptual - update actions would need separate implementation
- action: update_packages
  when: "{{ .update_packages }}"
```

## Troubleshooting

### Package Manager Not Detected
Ensure the package manager is installed and available in your PATH:

```bash
# Windows
winget --version
choco --version

# macOS
brew --version

# Linux
apt --version
yum --version
```

### Permission Issues
Some package managers require elevated privileges:

- **Windows**: Run as Administrator for some operations
- **Linux/macOS**: Commands automatically use `sudo` where needed

### Package Name Issues
If a package fails to install, check the correct package name:

```bash
# Search for correct package names
winget search nodejs
brew search node
apt search nodejs
```

### Network Issues
Ensure you have internet connectivity and package repositories are accessible. Some corporate networks may block package manager repositories.
