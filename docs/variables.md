# Variables System

The variables system is the heart of the dotfiles manager, providing a powerful and flexible way to manage configuration data across different platforms, environments, and use cases.

## 📖 **Table of Contents**

- [Overview](#overview)
- [Variable Sources](#variable-sources)
- [Variable Precedence](#variable-precedence)
- [Creating Variables](#creating-variables)
- [Importing Variables](#importing-variables)
- [Template Processing](#template-processing)
- [Built-in Functions](#built-in-functions)
- [CLI Commands](#cli-commands)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## 🎯 **Overview**

Variables provide dynamic data to templates and support:

- **Cross-platform configuration** - Different values for Windows/macOS/Linux
- **Environment-specific settings** - Work vs personal vs development
- **Template processing** - Go template syntax with custom functions
- **Conditional loading** - Load variables based on conditions
- **Import system** - Organize variables across multiple files

## 📁 **Variable Sources**

Variables are loaded from multiple sources in a specific order:

```
variables/
├── index.yaml           # Entry point and import definitions
├── global.yaml          # Global variables (all platforms)
├── platforms/           # Platform-specific variables
│   ├── windows.yaml     # Windows-only variables
│   ├── linux.yaml       # Linux-only variables
│   └── darwin.yaml      # macOS-only variables
└── environments/        # Environment-specific variables
    ├── work.yaml        # Work environment
    ├── personal.yaml    # Personal environment
    └── development.yaml # Development environment
```

## ⚡ **Variable Precedence**

Variables are merged in the following order (later sources override earlier ones):

1. **Global variables** (`global.yaml`)
2. **Platform-specific variables** (`platforms/{os}.yaml`)
3. **Environment-specific variables** (`environments/{env}.yaml`)
4. **Index variables** (direct variables in `index.yaml`)
5. **File-specific variables** (variables defined per template)

## 🔧 **Creating Variables**

### **Global Variables**

```yaml
# variables/global.yaml
user:
  name: "{{ .Env.USERNAME }}"
  email: "user@example.com"
  github: "username"

editor:
  default: "code"
  terminal: "vim"

directories:
  projects: '{{ pathJoin .User.Home "Projects" }}'
  downloads: '{{ pathJoin .User.Home "Downloads" }}'

colors:
  theme: "dark"
  accent: "blue"
```

### **Platform-Specific Variables**

```yaml
# variables/platforms/windows.yaml
paths:
  home: "{{ .Env.USERPROFILE }}"
  config: "{{ .Env.APPDATA }}"

shell:
  type: "powershell"
  profile: '{{ pathJoin .Env.USERPROFILE "Documents" "PowerShell" "profile.ps1" }}'
```

```yaml
# variables/platforms/linux.yaml
paths:
  home: "{{ .Env.HOME }}"
  config: '{{ pathJoin .Env.HOME ".config" }}'

shell:
  type: "bash"
  profile: '{{ pathJoin .Env.HOME ".bashrc" }}'
```

### **Environment-Specific Variables**

```yaml
# variables/environments/work.yaml
user:
  email: "name@company.com"

git:
  signing_key: "work_gpg_key_id"

proxy:
  http: "http://proxy.company.com:8080"
  https: "https://proxy.company.com:8080"

directories:
  work_projects: '{{ pathJoin .User.Home "Work" "Projects" }}'
```

## 📤 **Importing Variables**

### **Basic Import (variables/index.yaml)**

```yaml
# variables/index.yaml
imports:
  - path: "global.yaml"
  - path: "platforms/{{ .Platform.OS }}.yaml"
  - path: "environments/work.yaml"

variables:
  dotfiles:
    version: "1.0.0"
    initialized: true
```

### **Conditional Imports**

```yaml
# variables/index.yaml
imports:
  # Always load global variables
  - path: "global.yaml"

  # Load platform-specific variables if OS is detected
  - path: "platforms/{{ .Platform.OS }}.yaml"
    condition: 'ne .Platform.OS ""'

  # Load environment variables if DOTFILES_ENV is set
  - path: "environments/{{ .Env.DOTFILES_ENV }}.yaml"
    condition: 'ne .Env.DOTFILES_ENV ""'

  # Load hostname-specific variables
  - path: "hosts/{{ .Platform.Hostname }}.yaml"
    condition: 'ne .Platform.Hostname ""'

  # Load work variables during work hours
  - path: "environments/work.yaml"
    condition: 'and (ne .Env.DOTFILES_ENV "") (eq .Env.DOTFILES_ENV "work")'
```

### **Import with File-Specific Variables**

```yaml
imports:
  - path: "global.yaml"
    variables:
      import_source: "global"
      loaded_at: "{{ now }}"
```

## 🎨 **Template Processing**

Variables support Go template syntax with custom functions:

### **Basic Template Syntax**

```yaml
user:
  name: "{{ .Env.USERNAME }}"
  email: "{{ .user.name }}@example.com"

paths:
  config: "{{ .User.Home }}/.config"

shell:
  prompt: "{{ .user.name }}@{{ .Platform.Hostname }}"
```

### **Conditional Logic**

```yaml
editor:
  config_path: '{{ if eq .Platform.OS "windows" }}{{ .Env.APPDATA }}{{ else }}{{ .User.Home }}/.config{{ end }}/Code/User/settings.json'

package_manager: '{{ if eq .Platform.OS "windows" }}winget{{ else if eq .Platform.OS "darwin" }}brew{{ else }}apt{{ end }}'
```

### **Loops and Arrays**

```yaml
shell:
  aliases:
    - name: "ll"
      command: "ls -la"
    - name: "la"
      command: "ls -la"
# In templates:
# {{ range .shell.aliases }}
# alias {{ .name }}="{{ .command }}"
# {{ end }}
```

## 🛠️ **Built-in Functions**

### **Path Functions**

| Function    | Description          | Example                                |
| ----------- | -------------------- | -------------------------------------- |
| `pathJoin`  | Join path components | `{{ pathJoin .User.Home "Projects" }}` |
| `pathSep`   | Get path separator   | `{{ pathSep }}`                        |
| `pathClean` | Clean path           | `{{ pathClean .some.path }}`           |

### **Condition Functions**

| Function | Description | Example                                                         |
| -------- | ----------- | --------------------------------------------------------------- |
| `eq`     | Equal       | `{{ eq .Platform.OS "windows" }}`                               |
| `ne`     | Not equal   | `{{ ne .Env.USER "" }}`                                         |
| `and`    | Logical AND | `{{ and (eq .Platform.OS "linux") (ne .Env.DISPLAY "") }}`      |
| `or`     | Logical OR  | `{{ or (eq .Platform.OS "linux") (eq .Platform.OS "darwin") }}` |

### **String Functions**

| Function | Description | Example                    |
| -------- | ----------- | -------------------------- |
| `lower`  | Lowercase   | `{{ lower .Platform.OS }}` |
| `upper`  | Uppercase   | `{{ upper .user.name }}`   |
| `title`  | Title case  | `{{ title .user.name }}`   |

## 💻 **CLI Commands**

### **List All Variables**

```bash
# View all variables (YAML format)
dotfiles variables list

# View in JSON format
dotfiles variables list --format json

# View in table format
dotfiles variables list --format table

# Override platform detection
dotfiles variables list --platform linux --shell bash
```

### **Get Specific Variable**

```bash
# Get specific variable (dot notation)
dotfiles variables get user.name
dotfiles variables get directories.projects
dotfiles variables get shell.aliases

# Different output formats
dotfiles variables get user.name --format json
dotfiles variables get user.name --format raw
```

### **Debug Variables**

```bash
# Trace where a variable comes from (shows rendered values)
dotfiles variables trace user.name

# Show raw template syntax instead of rendered values
dotfiles variables trace user.name --raw

# Show all loaded sources
dotfiles variables sources

# Load with environment override
dotfiles variables list --env DOTFILES_ENV=work
```

### **Advanced Tracing Examples**

```bash
# Basic tracing (shows rendered values)
$ dotfiles variables trace user.name
Variable: user.name
Sources (rendered values):

1. Source: C:\Users\menno\.dotfiles\variables\global.yaml
   Value: menno

# Raw template debugging
$ dotfiles variables trace user.name --raw
Variable: user.name
Sources (raw template values):

1. Source: C:\Users\menno\.dotfiles\variables\global.yaml
   Raw Value: {{ .Env.USERNAME }}

Final processed result: menno

# Deep nested tracing
$ dotfiles variables trace user.details.location
Variable: user.details.location
Sources (rendered values):

1. Source: C:\Users\menno\.dotfiles\variables\global.yaml
   Value: Amsterdam

# Complex template tracing
$ dotfiles variables trace directories.projects --raw
Variable: directories.projects
Sources (raw template values):

1. Source: C:\Users\menno\.dotfiles\variables\global.yaml
   Raw Value: {{ pathJoin .User.Home "Projects" }}

Final processed result: C:\Users\menno\Projects
```

## 📋 **Examples**

### **Example 1: User Configuration**

```yaml
# variables/global.yaml
user:
  name: "{{ .Env.USERNAME }}"
  email: "{{ .user.name }}@example.com"
  full_name: "John Doe"

git:
  user_name: "{{ .user.full_name }}"
  user_email: "{{ .user.email }}"
  default_branch: "main"
```

### **Example 2: Cross-Platform Paths**

```yaml
# variables/global.yaml
editor:
  vscode_settings: '{{ if eq .Platform.OS "windows" }}{{ pathJoin .Env.APPDATA "Code" "User" "settings.json" }}{{ else if eq .Platform.OS "darwin" }}{{ pathJoin .User.Home "Library" "Application Support" "Code" "User" "settings.json" }}{{ else }}{{ pathJoin .User.Home ".config" "Code" "User" "settings.json" }}{{ end }}'
```

### **Example 3: Environment-Specific Configuration**

```yaml
# variables/environments/work.yaml
proxy:
  enabled: true
  http: "http://proxy.company.com:8080"
  https: "https://proxy.company.com:8080"

git:
  signing_key: "{{ .Env.WORK_GPG_KEY }}"
  email: "{{ .user.name }}@company.com"

directories:
  workspace: '{{ pathJoin .User.Home "Work" }}'
```

### **Example 4: Complex Conditional Logic**

```yaml
# variables/index.yaml
imports:
  - path: "global.yaml"

  # Load platform-specific variables
  - path: "platforms/{{ .Platform.OS }}.yaml"
    condition: 'ne .Platform.OS ""'

  # Load work config during work hours or if explicitly set
  - path: "environments/work.yaml"
    condition: 'or (eq .Env.DOTFILES_ENV "work") (and (ne .Env.WORK_MODE "") (eq .Env.WORK_MODE "on"))'

  # Load gaming config on gaming PC
  - path: "environments/gaming.yaml"
    condition: 'and (eq .Platform.OS "windows") (ne .Env.GAMING_PC "")'
```

## ✅ **Best Practices**

### **Organization**

1. **Keep global.yaml clean** - Only truly global variables
2. **Use platform-specific files** - for OS-dependent configurations
3. **Create environment files** - for work/personal/dev environments
4. **Use meaningful names** - Clear variable and file naming

### **Template Design**

1. **Use pathJoin for paths** - Ensures cross-platform compatibility
2. **Prefer conditions over complex logic** - Keep templates readable
3. **Document complex variables** - Add comments explaining usage
4. **Test across platforms** - Verify variables work on all target systems

### **Import Strategy**

1. **Use conditional imports** - Only load what's needed
2. **Maintain import order** - Consider precedence carefully
3. **Avoid circular imports** - The system will detect and error
4. **Keep imports explicit** - Don't rely on implicit loading

### **Variable Naming**

```yaml
# Good: Hierarchical and descriptive
user:
  name: "value"
  email: "value"

directories:
  projects: "value"
  downloads: "value"

# Avoid: Flat and unclear
username: "value"
user_email: "value"
proj_dir: "value"
```

## 🚨 **Troubleshooting**

### **Common Issues**

**1. Variable not found**

```bash
# Check if variable exists
dotfiles variables list | grep "variable_name"

# Trace variable source
dotfiles variables trace variable_name
```

**2. Template processing errors**

```bash
# Check for syntax errors in YAML
dotfiles validate

# View raw variables (before processing)
dotfiles variables sources
```

**3. Path issues on Windows**

```yaml
# Wrong: Mixed slashes
path: "{{ .User.Home }}/Documents"

# Right: Use pathJoin
path: "{{ pathJoin .User.Home \"Documents\" }}"
```

**4. Conditional imports not working**

```bash
# Test with explicit environment
dotfiles variables list --env DOTFILES_ENV=work

# Check condition evaluation
dotfiles variables trace import_file
```

### **Debugging Commands**

```bash
# Show all loaded sources and their order
dotfiles variables sources

# Get variable with source information (rendered values)
dotfiles variables trace user.name

# See raw template syntax for debugging
dotfiles variables trace user.name --raw

# Test with different platform settings
dotfiles variables list --platform windows --shell powershell

# Validate configuration
dotfiles validate
```

### **Error Messages**

| Error                          | Cause                 | Solution                                       |
| ------------------------------ | --------------------- | ---------------------------------------------- |
| `circular import detected`     | Import loop           | Check import chain, remove circular references |
| `variable file does not exist` | Missing file          | Check file path and spelling                   |
| `failed to parse template`     | Template syntax error | Check Go template syntax                       |
| `condition evaluation failed`  | Invalid condition     | Verify condition syntax and variables          |

---

**Next**: [Templates System](templates.md) | **Previous**: [Documentation Index](index.md)
