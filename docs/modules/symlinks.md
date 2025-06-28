# Symlinks Module

The symlinks module provides symbolic link management capabilities for dotfiles. It creates symbolic links from files in your dotfiles repository to their target destinations, enabling you to keep your configuration files centralized while making them available in their expected locations.

## Actions

The symlinks module provides one main action:

1. **`symlink`** - Create symbolic links from source to destination

### `symlink`

Creates a symbolic link from a source file in the dotfiles repository to a destination path. This is the primary method for deploying configuration files while keeping them centralized in your dotfiles repository.

**Parameters:**

| Parameter | Type    | Required | Default | Description                                                                                                      |
| --------- | ------- | -------- | ------- | ---------------------------------------------------------------------------------------------------------------- |
| `src`     | string  | Yes      | -       | The source file path relative to the dotfiles repository root. Supports template variables.                     |
| `dst`     | string  | Yes      | -       | The destination path where the symlink will be created. Supports template variables and path expansion.         |
| `backup`  | boolean | No       | `false` | Whether to create a backup of existing files before creating the symlink. Backup files get a `.backup` suffix. |

**Examples:**

```yaml
symlink:
  # Create a basic symlink
  - src: "files/config/nvim/init.vim"
    dst: "{{ .paths.home }}/.config/nvim/init.vim"

  # Create a symlink with backup
  - src: "files/config/git/gitconfig"
    dst: "{{ .paths.home }}/.gitconfig"
    backup: true

  # Symlink an entire directory
  - src: "files/config/zsh"
    dst: "{{ .paths.home }}/.config/zsh"

  # Platform-specific symlinks
  - src: "files/config/windows/powershell"
    dst: '{{ if eq .platform.os "windows" }}{{ .paths.home }}/Documents/PowerShell{{ else }}{{ .paths.home }}/.config/powershell{{ end }}'

  # Symlink with dynamic paths
  - src: "files/config/{{ .user.editor }}/config"
    dst: "{{ .paths.home }}/.config/{{ .user.editor }}/config"
```

## How Symlinks Work

Symbolic links (symlinks) are special files that point to another file or directory. When you access a symlink, the operating system automatically redirects to the target file.

### Benefits of Using Symlinks for Dotfiles

1. **Centralized Management** - Keep all configs in one repository
2. **Version Control** - Track changes to configuration files
3. **Easy Backup** - Single location to backup all configs
4. **Synchronization** - Share configs across multiple machines
5. **Atomic Updates** - Update configs by changing the symlink target

### Symlink vs File Copying

| Aspect              | Symlinks                        | File Copying                    |
| ------------------- | ------------------------------- | ------------------------------- |
| **Storage**         | Minimal (just the link)        | Full file duplication           |
| **Updates**         | Automatic (changes propagate)  | Manual (need to copy again)     |
| **Source Control**  | Single source of truth         | Multiple copies to maintain     |
| **Portability**     | Requires source always present  | Self-contained                  |
| **Performance**     | Tiny overhead                   | No overhead                     |

## Template Support

All path parameters support Go template syntax with access to your variables:

### Available Template Functions

- `pathJoin(paths...)` - Joins path components with OS-specific separators
- `pathSep()` - Returns the OS-specific path separator
- `pathClean(path)` - Cleans and normalizes a path

### Example Template Usage

```yaml
symlink:
  # Use path functions for cross-platform compatibility
  - src: '{{ pathJoin "files" "config" "app" "config.yml" }}'
    dst: '{{ pathJoin .paths.home ".config" "app" "config.yml" }}'

  # Platform-specific destinations
  - src: "files/config/app/settings.json"
    dst: '{{ if eq .platform.os "windows" }}{{ pathJoin .paths.home "AppData" "Local" "app" "settings.json" }}{{ else }}{{ pathJoin .paths.home ".config" "app" "settings.json" }}{{ end }}'

  # Use user variables
  - src: "files/editors/{{ .user.editor }}/config"
    dst: "{{ .paths.home }}/.config/{{ .user.editor }}/config"
```

## Backup Functionality

The `backup` parameter provides safety when creating symlinks that might overwrite existing files:

### Without Backup (Default)
```yaml
symlink:
  - src: "files/config/git/gitconfig"
    dst: "{{ .paths.home }}/.gitconfig"
    # Existing .gitconfig will be removed and replaced
```

### With Backup
```yaml
symlink:
  - src: "files/config/git/gitconfig"
    dst: "{{ .paths.home }}/.gitconfig"
    backup: true
    # Existing .gitconfig moved to .gitconfig.backup
```

**Backup behavior:**
- Existing files are renamed with `.backup` suffix
- Directories are also backed up recursively
- If a backup already exists, it will be overwritten
- Backups are created before symlink creation

## Directory Creation

Parent directories are automatically created when needed:

```yaml
symlink:
  # This will create ~/.config/nvim/ if it doesn't exist
  - src: "files/config/nvim/init.vim"
    dst: "{{ .paths.home }}/.config/nvim/init.vim"

  # This will create entire nested directory structure
  - src: "files/config/app/deeply/nested/config.yml"
    dst: "{{ .paths.home }}/.config/app/deeply/nested/config.yml"
```

## Symlink Validation and Updates

The symlinks module intelligently handles existing symlinks:

### Correct Symlink (No Action)
```
Target: ~/.gitconfig -> /path/to/dotfiles/files/config/git/gitconfig ✓
Status: Symlink already exists and points to correct target
Action: None (skipped)
```

### Incorrect Symlink (Update)
```
Target: ~/.gitconfig -> /old/path/to/gitconfig ✗
Status: Symlink exists but points to wrong target
Action: Update symlink target
```

### File Exists (Replace/Backup)
```
Target: ~/.gitconfig (regular file) ✗
Status: Regular file exists at destination
Action: Remove file and create symlink (or backup first)
```

## Platform-Specific Behavior

### Windows
- Symlinks require appropriate permissions (usually admin rights)
- Works with both files and directories
- NTFS filesystem required for symlinks
- Some Windows tools may not follow symlinks properly

### macOS/Linux
- Symlinks work without special permissions
- Universal support across applications
- Works on all common filesystems
- Standard Unix behavior

### Path Handling
All platforms automatically handle path separators correctly:

```yaml
symlink:
  # Automatically uses \ on Windows, / on Unix
  - src: "files/config/app/config.yml"
    dst: "{{ .paths.home }}/.config/app/config.yml"
```

## Common Use Cases

### 1. Configuration Files

```yaml
symlink:
  # Shell configuration
  - src: "files/shell/bashrc"
    dst: "{{ .paths.home }}/.bashrc"

  - src: "files/shell/zshrc"
    dst: "{{ .paths.home }}/.zshrc"

  # Editor configuration
  - src: "files/editors/vim/vimrc"
    dst: "{{ .paths.home }}/.vimrc"

  - src: "files/editors/vscode/settings.json"
    dst: "{{ .paths.home }}/.config/Code/User/settings.json"
```

### 2. Entire Configuration Directories

```yaml
symlink:
  # Symlink entire application config directories
  - src: "files/config/nvim"
    dst: "{{ .paths.home }}/.config/nvim"

  - src: "files/config/git"
    dst: "{{ .paths.home }}/.config/git"

  - src: "files/config/tmux"
    dst: "{{ .paths.home }}/.config/tmux"
```

### 3. Platform-Specific Configurations

```yaml
symlink:
  # Windows-specific
  - src: "files/windows/powershell/profile.ps1"
    dst: "{{ .paths.home }}/Documents/PowerShell/profile.ps1"
    condition: 'eq .platform.os "windows"'

  # macOS-specific
  - src: "files/macos/hammerspoon"
    dst: "{{ .paths.home }}/.hammerspoon"
    condition: 'eq .platform.os "darwin"'

  # Linux-specific
  - src: "files/linux/i3"
    dst: "{{ .paths.home }}/.config/i3"
    condition: 'eq .platform.os "linux"'
```

### 4. User-Specific Configurations

```yaml
symlink:
  # Different configs for different users/environments
  - src: "files/profiles/{{ .user.profile }}/gitconfig"
    dst: "{{ .paths.home }}/.gitconfig"

  - src: "files/profiles/{{ .user.profile }}/ssh/config"
    dst: "{{ .paths.home }}/.ssh/config"
    backup: true
```

## Error Handling

Common error scenarios and solutions:

### Source File Not Found
```
Error: Source file does not exist: files/config/missing.conf
```

**Solutions:**
- Verify the source file exists in your dotfiles repository
- Check the file path is relative to repository root
- Ensure no typos in the source path

### Permission Denied
```
Error: Permission denied creating symlink at /etc/config
```

**Solutions:**
- Run dotfiles manager with appropriate permissions
- Choose a user-writable destination
- Use `sudo` if targeting system directories (not recommended)

### Symlink Not Supported
```
Error: Symlinks not supported on this filesystem
```

**Solutions:**
- Use a filesystem that supports symlinks (NTFS, ext4, APFS, etc.)
- Consider using the files module instead for file copying
- Check Windows symlink permissions if on Windows

### Destination Directory Not Writable
```
Error: Cannot create parent directory /restricted/path
```

**Solutions:**
- Verify write permissions to parent directory
- Choose an accessible destination path
- Ensure the target directory can be created

## Best Practices

### 1. Organize Source Files Logically

```
dotfiles/
├── files/
│   ├── config/
│   │   ├── git/
│   │   │   ├── gitconfig
│   │   │   └── gitignore_global
│   │   ├── nvim/
│   │   │   ├── init.vim
│   │   │   └── plugins.vim
│   │   └── zsh/
│   │       ├── zshrc
│   │       └── aliases
│   └── scripts/
│       ├── backup.sh
│       └── sync.sh
└── jobs/
    └── symlinks.yaml
```

### 2. Use Consistent Naming

```yaml
symlink:
  # Good: Clear, consistent source organization
  - src: "files/config/git/gitconfig"
    dst: "{{ .paths.home }}/.gitconfig"

  - src: "files/config/git/gitignore_global"
    dst: "{{ .paths.home }}/.gitignore_global"

  # Avoid: Inconsistent or unclear organization
  - src: "git-config-file"
    dst: "{{ .paths.home }}/.gitconfig"
```

### 3. Handle Sensitive Files Carefully

```yaml
symlink:
  # Use backup for sensitive files
  - src: "files/ssh/config"
    dst: "{{ .paths.home }}/.ssh/config"
    backup: true

  # Be careful with SSH keys (consider using files module with proper permissions)
  - src: "files/ssh/id_rsa"
    dst: "{{ .paths.home }}/.ssh/id_rsa"
    backup: true
    condition: 'not (fileExists (pathJoin .paths.home ".ssh" "id_rsa"))'
```

### 4. Group Related Symlinks

```yaml
symlink:
  # Shell configuration
  - src: "files/shell/bashrc"
    dst: "{{ .paths.home }}/.bashrc"

  - src: "files/shell/zshrc"
    dst: "{{ .paths.home }}/.zshrc"

  - src: "files/shell/aliases"
    dst: "{{ .paths.home }}/.aliases"

  # Development tools
  - src: "files/dev/gitconfig"
    dst: "{{ .paths.home }}/.gitconfig"

  - src: "files/dev/editorconfig"
    dst: "{{ .paths.home }}/.editorconfig"
```

### 5. Test Symlinks Work Correctly

After creating symlinks, verify they work:

```bash
# Check symlink target
ls -la ~/.gitconfig

# Verify content is accessible
cat ~/.gitconfig

# Test that changes to source reflect in symlink
echo "# test" >> dotfiles/files/config/git/gitconfig
tail ~/.gitconfig  # Should show the test comment
```

## Integration with Other Modules

The symlinks module works well with other modules:

### With Files Module

```yaml
# Create files with proper permissions first
ensure_file:
  - path: "{{ .paths.home }}/.ssh/config"
    content_source: "files/templates/ssh/config.tmpl"
    render: true
    mode: "0600"

# Then symlink other configs
symlink:
  - src: "files/config/git/gitconfig"
    dst: "{{ .paths.home }}/.gitconfig"
```

### With Packages Module

```yaml
# Install applications first
install_package:
  - name: "nvim"

# Then symlink their configurations
symlink:
  - src: "files/config/nvim"
    dst: "{{ .paths.home }}/.config/nvim"
```

### Mixed Approach for Complex Setups

```yaml
# Use files module for templates that need rendering
ensure_file:
  - path: "{{ .paths.home }}/.gitconfig"
    content_source: "files/templates/git/gitconfig.tmpl"
    render: true

# Use symlinks for static files
symlink:
  - src: "files/config/git/gitignore_global"
    dst: "{{ .paths.home }}/.gitignore_global"

  - src: "files/config/nvim"
    dst: "{{ .paths.home }}/.config/nvim"
```

## Troubleshooting

### Debug Symlink Issues

Use verbose mode to see detailed symlink operations:

```bash
dotfiles apply --verbose --dry-run
```

### Check Symlink Status

```bash
# Check if a symlink exists and where it points
ls -la ~/.gitconfig

# Use readlink to see target
readlink ~/.gitconfig

# Check if target exists
ls -la $(readlink ~/.gitconfig)
```

### Fix Broken Symlinks

```bash
# Find broken symlinks
find ~ -type l -exec test ! -e {} \; -print

# Remove broken symlink and recreate
rm ~/.broken-config
dotfiles apply  # Will recreate the symlink
```

### Symlink vs Hard Link Confusion

Symlinks are different from hard links:

- **Symlink**: Points to a path (can break if target moves)
- **Hard link**: Points to file content (survives target moves)

The symlinks module creates symbolic links, not hard links.

This module provides a robust foundation for managing configuration files through symbolic links, enabling centralized dotfiles management while maintaining the expected file locations for applications.
