# Files Module

The files module provides comprehensive file and directory management capabilities for dotfiles. It can create directories and manage files with content from inline text or external source files.

## Actions

The files module provides two main actions:

1. **`ensure_dir`** - Create directories with proper permissions
2. **`ensure_file`** - Create or update files with content from inline text or external files

### `ensure_dir`

Creates directories with proper permissions. Ensures the directory exists and has the correct permissions on Unix-like systems.

**Parameters:**

| Parameter | Type   | Required | Default | Description                                                             |
| --------- | ------ | -------- | ------- | ----------------------------------------------------------------------- |
| `path`    | string | Yes      | -       | The directory path to create. Supports template variables.              |
| `mode`    | string | No       | `0755`  | File permissions in octal format (Unix/Linux only). Ignored on Windows. |

**Examples:**

```yaml
ensure_dir:
  # Create a basic directory
  - path: "{{ .paths.home }}/.config/myapp"

  # Create a directory with specific permissions
  - path: "{{ .paths.home }}/.ssh"
    mode: "0700"

  # Create nested directories
  - path: "{{ .paths.home }}/.local/share/applications"
```

### `ensure_file`

Creates or updates files with optional content. Content can be provided inline or loaded from a source file with optional template rendering.

**Parameters:**

| Parameter        | Type    | Required | Default | Description                                                                                         |
| ---------------- | ------- | -------- | ------- | --------------------------------------------------------------------------------------------------- |
| `path`           | string  | Yes      | -       | The file path to create. Supports template variables.                                               |
| `content`        | string  | No       | `""`    | Inline content for the file. Supports template variables. Mutually exclusive with `content_source`. |
| `content_source` | string  | No       | -       | Path to source file (relative to dotfiles root). Mutually exclusive with `content`.                 |
| `render`         | boolean | No       | `false` | Whether to process `content_source` as a template. Only applies to `content_source`.                |
| `mode`           | string  | No       | `0644`  | File permissions in octal format (Unix/Linux only). Ignored on Windows.                             |

**Examples:**

```yaml
ensure_file:
  # Create an empty file
  - path: "{{ .paths.home }}/.config/myapp/config.txt"

  # Create a file with inline content
  - path: "{{ .paths.home }}/.gitconfig"
    content: |
      [user]
          name = {{ .user.git_name }}
          email = {{ .user.git_email }}

  # Create from template source with rendering
  - path: "{{ .paths.home }}/.ssh/config"
    content_source: "files/templates/ssh/config.tmpl"
    render: true
    mode: "0600"

  # Copy file without templating
  - path: "{{ .paths.home }}/.config/app/config.json"
    content_source: "files/config/app.json"
    render: false

  # Create executable script
  - path: "{{ .paths.home }}/bin/myscript.sh"
    content: |
      #!/bin/bash
      echo 'Hello World'
    mode: "0755"
```

**Note:** For copying files without template processing, use `ensure_file` with `content_source` and `render: false`. This provides the same functionality with better content change detection and permission control.

## Template Support

All path parameters support Go template syntax with access to your variables:

### Available Template Functions

- `pathJoin(paths...)` - Joins path components with OS-specific separators
- `pathSep()` - Returns the OS-specific path separator
- `pathClean(path)` - Cleans and normalizes a path

### Example Template Usage

```yaml
ensure_file:
  - path: '{{ pathJoin .paths.home ".config" "app" "config.yml" }}'
    content: |
      user: {{ .user.name }}
      home: {{ .paths.home }}
      projects_dir: {{ pathJoin .paths.home "Projects" }}
      platform: {{ .platform.os }}
```

## Content Management

### Inline Content vs Content Source

**Inline Content:**

- Always processed as templates
- Good for simple configuration files
- Content is embedded in the job definition

```yaml
ensure_file:
  - path: "{{ .paths.home }}/.bashrc"
    content: |
      export PATH="$PATH:{{ .paths.home }}/bin"
      alias ll="ls -la"
```

**Content Source:**

- Content loaded from external files
- Optional template rendering with `render: true`
- Better for complex configurations
- Allows reuse across multiple jobs

```yaml
ensure_file:
  - path: "{{ .paths.home }}/.vimrc"
    content_source: "files/config/vim/vimrc.tmpl"
    render: true
```

### Template Rendering for Content Source

### File Copying vs Template Rendering

For simple file copying without template processing, use `render: false`:

```yaml
ensure_file:
  # Copy file without any template processing
  - path: "{{ .paths.home }}/.config/app/config.json"
    content_source: "files/config/app.json"
    render: false
```

When using `content_source` with `render: true`, the source file is processed as a Go template:

**Source file (`files/templates/bashrc.tmpl`):**

```bash
# {{ .user.name }}'s bash configuration
export EDITOR="{{ .user.editor }}"
export PROJECTS_DIR="{{ pathJoin .paths.home "Projects" }}"

# Platform-specific settings
{{ if eq .platform.os "darwin" }}
export HOMEBREW_PREFIX="/opt/homebrew"
{{ else if eq .platform.os "linux" }}
export XDG_CONFIG_HOME="{{ pathJoin .paths.home ".config" }}"
{{ end }}
```

**Job definition:**

```yaml
ensure_file:
  - path: "{{ .paths.home }}/.bashrc"
    content_source: "files/templates/bashrc.tmpl"
    render: true
```

## File Permissions

On Unix-like systems (Linux, macOS), you can specify file permissions using octal notation:

```yaml
ensure_file:
  # Private SSH key (read/write for owner only)
  - path: "{{ .paths.home }}/.ssh/id_rsa"
    content_source: "files/ssh/id_rsa"
    mode: "0600"

  # Executable script
  - path: "{{ .paths.home }}/bin/backup.sh"
    content_source: "files/scripts/backup.sh"
    mode: "0755"

  # World-readable config
  - path: "{{ .paths.home }}/.profile"
    content_source: "files/shell/profile"
    mode: "0644"
```

**Note:** File permissions are ignored on Windows systems.

## Directory Creation

Parent directories are automatically created when needed:

```yaml
ensure_file:
  # This will create ~/.config/app/ if it doesn't exist
  - path: "{{ .paths.home }}/.config/app/settings.json"
    content: "{}"
```

For explicit directory creation with specific permissions:

```yaml
ensure_dir:
  - path: "{{ .paths.home }}/.config/app"
    mode: "0755"
```

## Platform-Specific Behavior

### Windows

- File permissions (`mode`) are ignored
- Path separators are automatically converted
- Home directory expansion works with Windows paths

### Unix-like (Linux, macOS)

- File permissions are applied as specified
- Default directory permissions: `0755`
- Default file permissions: `0644`

## Content Change Detection

The files module intelligently detects when files need updates:

- **Content comparison**: Files are only updated if content differs
- **Permission changes**: On Unix systems, permissions are updated if they differ
- **Backup support**: Files can be backed up before modification (not implemented in this version)

## Error Handling

Common error scenarios and solutions:

### Source File Not Found

```yaml
ensure_file:
  # Error: content source file not found
  - path: "{{ .paths.home }}/.vimrc"
    content_source: "files/config/vim/missing.vim" # File doesn't exist
```

**Solution:** Ensure the source file exists in your dotfiles repository.

### Permission Denied

```yaml
ensure_file:
  # May fail if destination requires elevated permissions
  - path: "/etc/hosts"
    content: "127.0.0.1 localhost"
```

**Solution:** Run dotfiles manager with appropriate permissions or choose a user-writable location.

### Template Syntax Errors

```yaml
ensure_file:
  # Error: invalid template syntax
  - path: "{{ .paths.home }}/.bashrc"
    content: "export PATH={{ .invalid_variable }" # Missing closing brace
```

**Solution:** Fix template syntax and ensure variables exist.

## Best Practices

### 1. Use Template Files for Complex Configurations

Instead of inline content:

```yaml
ensure_file:
  # Good: Use template files for complex configs
  - path: "{{ .paths.home }}/.vimrc"
    content_source: "files/config/vim/vimrc.tmpl"
    render: true
```

### 2. Organize Source Files Logically

```
dotfiles/
├── files/
│   ├── config/
│   │   ├── git/
│   │   │   └── gitconfig.tmpl
│   │   ├── vim/
│   │   │   └── vimrc.tmpl
│   │   └── zsh/
│   │       └── zshrc.tmpl
│   └── templates/
│       └── ssh/
│           └── config.tmpl
└── jobs/
    └── files.yaml
```

### 3. Set Appropriate Permissions

```yaml
ensure_file:
  # SSH files should be private
  - path: "{{ .paths.home }}/.ssh/config"
    content_source: "files/templates/ssh/config.tmpl"
    render: true
    mode: "0600"

  # Scripts should be executable
  - path: "{{ .paths.home }}/bin/update.sh"
    content_source: "files/scripts/update.sh"
    mode: "0755"
```

### 4. Use Variables for Reusability

```yaml
# Define in variables
user:
  name: "John Doe"
  git_email: "john@example.com"

# Use in jobs
ensure_file:
  - path: "{{ .paths.home }}/.gitconfig"
    content: |
      [user]
          name = {{ .user.name }}
          email = {{ .user.git_email }}
```

### 5. Handle Platform Differences

```yaml
ensure_file:
  # Platform-specific paths
  - path: '{{ if eq .platform.os "windows" }}{{ .paths.home }}/AppData/Local/app/config.yml{{ else }}{{ .paths.home }}/.config/app/config.yml{{ end }}'
    content_source: "files/config/app.yml.tmpl"
    render: true
```

## Integration with Other Modules

The files module works well with other modules:

### With Symlinks Module

```yaml
ensure_file:
  # Create config file
  - path: "{{ .paths.home }}/.local/share/app/config.yml"
    content_source: "files/config/app.yml.tmpl"
    render: true

symlink:
  # Symlink to standard location
  - src: "{{ .paths.home }}/.local/share/app/config.yml"
    dst: "{{ .paths.home }}/.config/app/config.yml"
```

### With Packages Module

```yaml
install_package:
  # Install application
  - name: "vim"

ensure_file:
  # Configure application
  - path: "{{ .paths.home }}/.vimrc"
    content_source: "files/config/vim/vimrc.tmpl"
    render: true
```

This module provides a solid foundation for managing your dotfiles with flexible content management and platform awareness.
