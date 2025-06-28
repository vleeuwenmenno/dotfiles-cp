# Dotfiles Manager Documentation

A comprehensive cross-platform dotfiles manager built in Go with templating, variable system, and a variety of modules to be used in jobs.

## 📖 **Documentation Index**

### **Getting Started**

- [Installation & Setup](installation.md)
- [Quick Start Guide](quickstart.md)
- [Configuration Overview](configuration.md)

### **Core Features**

- [Job Modules](modules.md) - Available modules for usage in jobs
  - [Package Management](modules/packages.md) - Cross-platform package installation and management
  - [File Management](modules/files.md) - File creation, modification, deletion and template management
  - [Symlinks](modules/symlinks.md) - Symlink creation, and modification
- [Import System](imports.md) - File imports and dependency management
- [Variables System](variables.md) - Variable loading, processing, and management
- [Platform Detection](platforms.md) - OS, shell, and architecture detection
- [Condition Syntax](condition-syntax.md) - Jobs and Variables can have conditional logic
- [Debugging & Troubleshooting](DEBUG.md) - Debug techniques and common issues

### **Reference**

- [CLI Commands](cli-reference.md) - Complete command reference

### **Development**

- [Architecture](architecture.md) - Project structure and design decisions
- [Contributing](../CONTRIBUTING.md) - How to contribute to the project
- [Building & Testing](building.md) - Development workflow

## 🚀 **Quick Navigation**

### **I want to...**

- **Get started quickly** → [Quick Start Guide](quickstart.md)
- **Understand variables** → [Variables System](variables.md)
- **Install packages automatically** → [Package Management](modules/packages.md)
- **Manage symlinks** → [Symbolic Links](modules/symlinks.md)
- **Manage files and/or template them** → [File Management](modules/files.md)
- **Debug my configuration** → [Debugging Guide](DEBUG.md)
- **See all CLI commands** → [CLI Reference](cli-reference.md)
- **Create conditional configurations** → [Condition Syntax](condition-syntax.md)

## 🎯 **Key Concepts**

### **Variables**

Variables provide data for templates and can be:

- **Global** - Available everywhere
- **Platform-specific** - Windows/macOS/Linux specific
- **Environment-specific** - Work/personal/development specific
- **Conditional** - Loaded based on conditions

## 🔍 **Quick Examples**

### **Variable Usage**

```yaml
# variables/global.yaml
user:
  name: "{{ .Env.USERNAME }}"
  email: "user@example.com"

directories:
  projects: '{{ pathJoin .User.Home "Projects" }}'
```

### **Template Usage**

Files managed, and loaded with the files module.

```bash
# templates/shell/.bashrc.tmpl
export PROJECTS_DIR="{{ .directories.projects }}"
alias ll="ls -la"

# User: {{ .user.name }}
# Email: {{ .user.email }}
```

### **Conditional Import**

Both for jobs and variables imports can be conditional.

```yaml
# variables/index.yaml
imports:
  - path: "global.yaml"
  - path: "platforms/{{ .Platform.OS }}.yaml"
    condition: 'ne .Platform.OS ""'
```

## 🛠️ **CLI Quick Reference**

```bash
# Initialize new dotfiles repository
dotfiles init

# View all variables
dotfiles variables list

# Get specific variable
dotfiles variables get user.name

# Debug variable sources (shows rendered values)
dotfiles variables trace user.name

# Debug raw template syntax
dotfiles variables trace user.name --raw

# Show all variable sources
dotfiles variables sources

# Validate configuration
dotfiles validate

# Apply configuration
dotfiles apply --dry-run

# Show platform info
dotfiles info
```

## 📋 **Support & Help**

- **Issues**: Found a bug? [Report it](https://github.com/vleeuwenmenno/dotfiles-cp/issues)
- **Questions**: Need help? Check [DEBUG.md](DEBUG.md) first
- **Contributing**: Want to contribute? See [Contributing Guide](../CONTRIBUTING.md)
