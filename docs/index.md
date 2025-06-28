# Dotfiles Manager Documentation

A comprehensive cross-platform dotfiles manager built in Go with templating, variable system, and package management integration.

## 📖 **Documentation Index**

### **Getting Started**

- [Installation & Setup](installation.md)
- [Quick Start Guide](quickstart.md)
- [Configuration Overview](configuration.md)

### **Core Features**

- [Variables System](variables.md) - Variable loading, processing, and management
- [Templates System](templates.md) - Template processing and deployment
- [Package Management](packages.md) - Cross-platform package installation and management
- [Platform Detection](platforms.md) - OS, shell, and architecture detection

### **Advanced Usage**

- [Conditional Logic](conditionals.md) - Platform and environment-based conditions
- [Custom Functions](functions.md) - Template functions and helpers
- [Import System](imports.md) - File imports and dependency management
- [Debugging & Troubleshooting](DEBUG.md) - Debug techniques and common issues

### **Reference**

- [CLI Commands](cli-reference.md) - Complete command reference
- [Configuration Reference](config-reference.md) - All configuration options
- [Template Functions](template-functions.md) - Available template functions
- [Error Codes](error-codes.md) - Error messages and solutions

### **Development**

- [Architecture](architecture.md) - Project structure and design decisions
- [Contributing](../CONTRIBUTING.md) - How to contribute to the project
- [Building & Testing](building.md) - Development workflow

## 🚀 **Quick Navigation**

### **I want to...**

- **Get started quickly** → [Quick Start Guide](quickstart.md)
- **Understand variables** → [Variables System](variables.md)
- **Install packages automatically** → [Package Management](packages.md)
- **Debug my configuration** → [Debugging Guide](DEBUG.md)
- **See all CLI commands** → [CLI Reference](cli-reference.md)
- **Create conditional configurations** → [Conditional Logic](conditionals.md)

### **Common Tasks**

| Task                            | Documentation                                       |
| ------------------------------- | --------------------------------------------------- |
| Initialize a new dotfiles repo  | [Quick Start Guide](quickstart.md#initialization)   |
| Add platform-specific variables | [Variables System](variables.md#platform-variables) |
| Create templates                | [Templates System](templates.md#creating-templates) |
| Debug variable loading          | [DEBUG.md](DEBUG.md#variable-debugging)             |
| Set up conditional imports      | [Import System](imports.md#conditional-imports)     |
| Install packages automatically  | [Package Management](packages.md)                   |

## 🎯 **Key Concepts**

### **Variables**

Variables provide data for templates and can be:

- **Global** - Available everywhere
- **Platform-specific** - Windows/macOS/Linux specific
- **Environment-specific** - Work/personal/development specific
- **Conditional** - Loaded based on conditions

### **Templates**

Templates are files that get processed with variables and deployed to target locations. They support:

- Go template syntax
- Custom functions for paths, conditions, etc.
- Cross-platform path handling
- Conditional deployment

### **Import System**

Files can import other files with:

- Relative path support
- Conditional imports
- Circular dependency detection
- Breadcrumb error tracking

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

```bash
# templates/shell/.bashrc.tmpl
export PROJECTS_DIR="{{ .directories.projects }}"
alias ll="ls -la"

# User: {{ .user.name }}
# Email: {{ .user.email }}
```

### **Conditional Import**

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

# Apply configuration
dotfiles apply

# Validate configuration
dotfiles validate

# Show platform info
dotfiles info
```

## 📋 **Support & Help**

- **Issues**: Found a bug? [Report it](https://github.com/vleeuwenmenno/dotfiles-cp/issues)
- **Questions**: Need help? Check [DEBUG.md](DEBUG.md) first
- **Contributing**: Want to contribute? See [Contributing Guide](../CONTRIBUTING.md)

---

**Version**: 1.0.0
**Last Updated**: 2025-06-28
**Project**: [github.com/vleeuwenmenno/dotfiles-cp](https://github.com/vleeuwenmenno/dotfiles-cp)
