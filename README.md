# Dotfiles Manager

A powerful cross-platform dotfiles manager with templating support, built in Go. Works seamlessly across Windows, macOS, and Linux with support for multiple shells and package managers.

## Features

- 🚀 **Cross-platform**: Windows 11, macOS, and Linux support
- 🐚 **Multi-shell**: PowerShell, Bash, ZSH support  
- 📦 **Package managers**: Chocolatey, Winget, Homebrew, APT, YUM/DNF support
- 🎨 **Templating**: Go templates with conditional logic and variables
- ⚙️ **Flexible configuration**: YAML-based with platform-specific overrides
- 🔗 **Smart linking**: Automatic symlink management with backups
- 📊 **Rich logging**: Beautiful console output with zerolog
- 🛠️ **Easy installation**: Single binary with no dependencies

## Quick Start

### Installation

#### Via Go (Recommended)
```bash
go install github.com/vleeuwenmenno/dotfiles-cp/cmd/dotfiles@latest
```

**Note**: Make sure `$GOPATH/bin` (or `$GOBIN`) is in your PATH:
```bash
# Add to your shell profile (.bashrc, .zshrc, etc.)
export PATH=$PATH:$(go env GOPATH)/bin

# Or for current session only
export PATH=$PATH:$(go env GOPATH)/bin
```

#### Build from Source
```bash
git clone https://github.com/vleeuwenmenno/dotfiles-cp.git
cd dotfiles-cp

# Option 1: Use our build script (installs to GOPATH/bin)
go run build.go -install

# Option 2: Use Go install directly
go install ./cmd/dotfiles
```

#### Download Binary
Download the latest binary from the [releases page](https://github.com/vleeuwenmenno/dotfiles-cp/releases).

### Initialize Your Dotfiles

```bash
# Create a new dotfiles repository
dotfiles init

# Check platform information
dotfiles info

# Apply dotfiles configuration
dotfiles apply

# Update to latest version
dotfiles update
```

## Usage

### Commands

- `dotfiles init` - Initialize a new dotfiles repository
- `dotfiles apply` - Apply dotfiles configuration (symlinks, packages, scripts)
- `dotfiles update` - Update dotfiles manager to latest version
- `dotfiles update --check` - Check for updates without installing
- `dotfiles info` - Show platform and environment information
- `dotfiles version` - Show version information

### Global Flags

- `-v, --verbose` - Enable verbose logging
- `-q, --quiet` - Enable quiet mode (errors only)

## Configuration

The dotfiles manager uses a `dotfiles.yaml` configuration file that defines what happens when you run `dotfiles apply`:

```yaml
metadata:
  name: "My Dotfiles"
  version: "1.0.0"
  author: "Your Name"
  description: "Personal dotfiles configuration"

settings:
  backup_dir: "~/.dotfiles-backup"    # Where to backup existing files
  template_dir: "templates"           # Directory containing template files
  target_dir: "~"                     # Base directory for file placement
  log_level: "info"

variables:
  git_user: "Your Name"               # Variables available in templates
  git_email: "your.email@example.com"

platforms:
  - name: "windows"
    conditions:
      os: "windows"
    packages:                         # Installed via package managers
      chocolatey:
        - git
        - vscode
      winget:
        - Microsoft.PowerShell
    files:                           # Files to create/symlink
      - source: "templates/powershell/profile.ps1.tmpl"
        target: "~/Documents/PowerShell/Microsoft.PowerShell_profile.ps1"
        template: true               # Process as Go template
        
  - name: "macos"
    conditions:
      os: "darwin"
    packages:
      homebrew:
        - git
        - neovim
    files:
      - source: "templates/zsh/zshrc.tmpl"
        target: "~/.zshrc"
        template: true
        
  - name: "linux"
    conditions:
      os: "linux"
    packages:
      apt:
        - git
        - neovim
    files:
      - source: "templates/bash/bashrc.tmpl"
        target: "~/.bashrc"
        template: true
```

When you run `dotfiles apply`, the manager will:
1. **Detect your platform** (OS, shell, available package managers)
2. **Install packages** using the appropriate package manager
3. **Process templates** and create configuration files
4. **Create symlinks** or copy files to target locations
5. **Backup existing files** before making changes

## Templating

Templates use Go's template syntax with additional functions:

```bash
# ~/.zshrc template
{{- if eq .OS "darwin" }}
export PATH="/opt/homebrew/bin:$PATH"
{{- end }}

{{- if .Git.Enabled }}
alias gs="git status"
alias ga="git add"
{{- end }}

# User variables
export GIT_USER="{{ .Variables.git_user }}"
export GIT_EMAIL="{{ .Variables.git_email }}"
```

## Platform Support

### Windows 11
- **Shells**: PowerShell 7+, CMD
- **Package Managers**: Chocolatey, Winget, Scoop
- **Configs**: PowerShell profiles, Windows Terminal

### macOS
- **Shells**: ZSH (default), Bash
- **Package Managers**: Homebrew, MacPorts
- **Configs**: Shell profiles, app preferences

### Linux
- **Shells**: Bash, ZSH, Fish
- **Package Managers**: APT, YUM/DNF, Pacman, Zypper
- **Configs**: Shell profiles, desktop environments

## Development

### Prerequisites
- Go 1.22.2 or later

### Building
```bash
# Clone the repository
git clone https://github.com/vleeuwenmenno/dotfiles-cp.git
cd dotfiles-cp

# Install dependencies
go run build.go -deps

# Build for current platform
go run build.go

# Build for all platforms
go run build.go -all

# Build and run
go run build.go -run

# Build with specific version
go run build.go -version=1.0.0

# Show all build options
go run build.go -help
```

### Testing
```bash
# Run tests
go run build.go -test

# Run tests with coverage
go run build.go -coverage

# Format code
go run build.go -fmt

# Clean build artifacts
go run build.go -clean
```

## Architecture

```
dotfiles-manager/
├── cmd/dotfiles/          # CLI entry point
├── internal/
│   ├── config/           # Configuration management
│   ├── logger/           # Logging setup (zerolog)
│   └── platform/         # Platform detection
├── pkg/utils/            # Utility functions
└── build.go              # Cross-platform build script
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Build System

The project uses a custom Go-based build system that works across all platforms:

```bash
# Available build commands
go run build.go -help        # Show all options
go run build.go              # Build for current platform
go run build.go -all         # Build for all platforms
go run build.go -run         # Build and run
go run build.go -test        # Run tests
go run build.go -coverage    # Run tests with coverage
go run build.go -fmt         # Format code
go run build.go -clean       # Clean build artifacts
go run build.go -install     # Build and install to GOPATH/bin
go run build.go -deps        # Download dependencies
```

**Installation & Updates:**
- **Install**: `go install github.com/vleeuwenmenno/dotfiles-cp/cmd/dotfiles@latest`
- **Update**: `dotfiles update` (or re-run the install command)
- **Development**: `go run build.go -install` (from source)
- **Binary**: Download from releases page

**Update Methods:**
```bash
# Method 1: Use built-in update command (requires Go)
dotfiles update

# Method 2: Check for updates without installing
dotfiles update --check

# Method 3: Re-run install command
go install github.com/vleeuwenmenno/dotfiles-cp/cmd/dotfiles@latest

# Method 4: Install specific version
go install github.com/vleeuwenmenno/dotfiles-cp/cmd/dotfiles@v1.0.0
```

**Note**: The `dotfiles update` command requires Go to be installed. If you don't have Go installed, you can:
- Download the latest binary from the releases page
- Install Go and then use the update command
- Use a package manager like Homebrew, Chocolatey, etc. (when available)

## Roadmap

- [x] Cross-platform build system
- [x] Platform detection and logging
- [x] Basic CLI structure with Cobra
- [ ] Templating engine implementation
- [ ] Package installation logic
- [ ] File management and symlinking
- [ ] Backup and restore functionality
- [ ] Configuration validation
- [ ] Shell integration scripts
- [ ] CI/CD pipeline
- [ ] Documentation website

## Support

- 📖 [Documentation](https://github.com/vleeuwenmenno/dotfiles-cp/wiki)
- 🐛 [Issue Tracker](https://github.com/vleeuwenmenno/dotfiles-cp/issues)
- 💬 [Discussions](https://github.com/vleeuwenmenno/dotfiles-cp/discussions)