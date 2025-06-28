package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/config"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/logger"
	"github.com/vleeuwenmenno/dotfiles-cp/pkg/utils"

	"github.com/spf13/cobra"
)

// createInitCommand creates the init command
func createInitCommand() *cobra.Command {
	var (
		targetDir string
		force     bool
	)

	initCmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a new dotfiles repository",
		Long: `Initialize a new dotfiles repository with sample configuration files.

Creates the following structure:
  dotfiles.yaml        - Main configuration file
  variables/
    index.yaml         - Variables entry point
    global.yaml        - Global variables
    platforms/         - Platform-specific variables
  templates/
    index.yaml         - Templates entry point
    shell/             - Shell configuration templates
    git/               - Git configuration templates
  files/               - Static files (no templating)

If no directory is specified, initializes in the current directory.`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.Get()

			// Determine target directory
			if len(args) > 0 {
				targetDir = args[0]
			} else if targetDir == "" {
				var err error
				targetDir, err = os.Getwd()
				if err != nil {
					log.Error().Err(err).Msg("Failed to get current directory")
					os.Exit(1)
				}
			}

			// Expand path
			expandedDir, err := utils.ExpandPath(targetDir)
			if err != nil {
				log.Error().Err(err).Msg("Failed to expand target directory path")
				os.Exit(1)
			}

			// Check if directory exists and is not empty
			if !force && directoryExistsAndNotEmpty(expandedDir) {
				log.Error().
					Str("directory", expandedDir).
					Msg("Directory exists and is not empty. Use --force to override")
				os.Exit(1)
			}

			// Initialize repository
			if err := initializeRepository(expandedDir); err != nil {
				log.Error().Err(err).Msg("Failed to initialize dotfiles repository")
				os.Exit(1)
			}

			log.Info().
				Str("directory", expandedDir).
				Msg("Successfully initialized dotfiles repository")

			// Show next steps
			showNextSteps(expandedDir)
		},
	}

	initCmd.Flags().StringVarP(&targetDir, "directory", "d", "", "Target directory for initialization")
	initCmd.Flags().BoolVar(&force, "force", false, "Force initialization even if directory is not empty")

	return initCmd
}

// initializeRepository creates the complete dotfiles repository structure
func initializeRepository(targetDir string) error {
	log := logger.Get()

	// Create base directory
	if err := utils.EnsureDir(targetDir); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Create directory structure
	dirs := []string{
		"variables",
		"variables/platforms",
		"variables/environments",
		"jobs",
		"files",
		"files/templates",
		"files/configs",
		"files/bin",
		"scripts",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(targetDir, dir)
		if err := utils.EnsureDir(dirPath); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		log.Debug().Str("directory", dir).Msg("Created directory")
	}

	// Create configuration files
	if err := createMainConfig(targetDir); err != nil {
		return fmt.Errorf("failed to create main configuration: %w", err)
	}

	if err := createVariablesIndex(targetDir); err != nil {
		return fmt.Errorf("failed to create variables index: %w", err)
	}

	if err := createGlobalVariables(targetDir); err != nil {
		return fmt.Errorf("failed to create global variables: %w", err)
	}

	if err := createPlatformVariables(targetDir); err != nil {
		return fmt.Errorf("failed to create platform variables: %w", err)
	}

	if err := createJobsIndex(targetDir); err != nil {
		return fmt.Errorf("failed to create jobs index: %w", err)
	}

	if err := createSampleFiles(targetDir); err != nil {
		return fmt.Errorf("failed to create sample files: %w", err)
	}

	if err := createSampleScripts(targetDir); err != nil {
		return fmt.Errorf("failed to create sample scripts: %w", err)
	}

	if err := createReadme(targetDir); err != nil {
		return fmt.Errorf("failed to create README: %w", err)
	}

	if err := createGitignore(targetDir); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	return nil
}

// createMainConfig creates the main dotfiles.yaml configuration file
func createMainConfig(targetDir string) error {
	config := config.DefaultConfig()

	// Customize metadata
	config.Metadata.Name = "My Dotfiles"
	config.Metadata.Version = "1.0.0"
	config.Metadata.Author = os.Getenv("USER")
	if config.Metadata.Author == "" {
		config.Metadata.Author = os.Getenv("USERNAME") // Windows
	}
	if config.Metadata.Author == "" {
		config.Metadata.Author = "User"
	}
	config.Metadata.Description = "Personal dotfiles configuration"

	configPath := filepath.Join(targetDir, "dotfiles.yaml")
	return config.Save(configPath)
}

// createVariablesIndex creates the variables/index.yaml file
func createVariablesIndex(targetDir string) error {
	content := `# Variables Index
# This file defines which variable files to load and in what order
# Variables are merged with later files overriding earlier ones

imports:
  - path: "global.yaml"
  - path: "platforms/{{ .Platform.OS }}.yaml"
    condition: "ne .Platform.OS \"\""
  - path: "environments/{{ .Env.DOTFILES_ENV }}.yaml"
    condition: "ne .Env.DOTFILES_ENV \"\""

# Direct variables can also be defined here
variables:
  dotfiles:
    version: "1.0.0"
    initialized: true
`

	indexPath := filepath.Join(targetDir, "variables", "index.yaml")
	return os.WriteFile(indexPath, []byte(content), 0644)
}

// createGlobalVariables creates the variables/global.yaml file
func createGlobalVariables(targetDir string) error {
	content := `# Global Variables
# These variables are available across all platforms and templates

user:
  name: "{{ .Env.USER }}{{ .Env.USERNAME }}"
  email: "user@example.com"
  github: "username"

editor:
  default: "code"
  terminal: "vim"

git:
  default_branch: "main"
  signing_key: ""

shell:
  aliases:
    ll: "ls -la"
    la: "ls -la"
    l: "ls -l"
    ...: "cd ../.."
    ....: "cd ../../.."

colors:
  theme: "dark"
  accent: "blue"

directories:
  projects: "{{ pathJoin .User.Home \"Projects\" }}"
  downloads: "{{ pathJoin .User.Home \"Downloads\" }}"
  documents: "{{ pathJoin .User.Home \"Documents\" }}"
`

	globalPath := filepath.Join(targetDir, "variables", "global.yaml")
	return os.WriteFile(globalPath, []byte(content), 0644)
}

// createPlatformVariables creates platform-specific variable files
func createPlatformVariables(targetDir string) error {
	platforms := map[string]string{
		"windows": `# Windows-specific variables

paths:
  home: "{{ .Env.USERPROFILE }}"
  config: "{{ .Env.APPDATA }}"

shell:
  type: "powershell"
  profile: "{{ pathJoin .Env.USERPROFILE \"Documents\" \"PowerShell\" \"profile.ps1\" }}"

editor:
  vscode_settings: "{{ pathJoin .Env.APPDATA \"Code\" \"User\" \"settings.json\" }}"
`,
		"linux": `# Linux-specific variables

paths:
  home: "{{ .Env.HOME }}"
  config: "{{ pathJoin .Env.HOME \".config\" }}"

shell:
  type: "bash"
  profile: "{{ pathJoin .Env.HOME \".bashrc\" }}"

editor:
  vscode_settings: "{{ pathJoin .Env.HOME \".config\" \"Code\" \"User\" \"settings.json\" }}"
`,
		"darwin": `# macOS-specific variables

paths:
  home: "{{ .Env.HOME }}"
  config: "{{ pathJoin .Env.HOME \".config\" }}"

shell:
  type: "zsh"
  profile: "{{ pathJoin .Env.HOME \".zshrc\" }}"

package_managers:
  - "brew"

editor:
  vscode_settings: "{{ pathJoin .Env.HOME \"Library\" \"Application Support\" \"Code\" \"User\" \"settings.json\" }}"
`,
	}

	for platform, content := range platforms {
		platformPath := filepath.Join(targetDir, "variables", "platforms", platform+".yaml")
		if err := os.WriteFile(platformPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

// createJobsIndex creates the jobs/index.yaml file
func createJobsIndex(targetDir string) error {
	content := `# Jobs Configuration
# Define what operations to perform for your dotfiles setup

# Ensure directories exist
ensure_dir:
  - "{{ .paths.home }}/.ssh"
  - path: "{{ .paths.config }}/git"
    mode: "0755"

# Process and deploy templates using ensure_file
ensure_file:
  - path: "{{ .paths.home }}/.gitconfig"
    content_source: "files/templates/git/config.tmpl"
    render: true
  - path: "{{ .paths.home }}/.ssh/config"
    content_source: "files/templates/ssh/config.tmpl"
    render: true
    mode: "0600"
    condition: "ne .Platform.OS \"windows\""
  - path: "{{ .paths.home }}/.bashrc"
    content_source: "files/templates/shell/bashrc.tmpl"
    render: true
    condition: "eq .Platform.OS \"linux\""
  - path: "{{ .paths.home }}/.zshrc"
    content_source: "files/templates/shell/zshrc.tmpl"
    render: true
    condition: "eq .Platform.OS \"darwin\""
  - path: "{{ pathJoin .paths.home \"Documents\" \"PowerShell\" \"profile.ps1\" }}"
    content_source: "files/templates/shell/profile.ps1.tmpl"
    render: true
    condition: "eq .Platform.OS \"windows\""
  - path: "{{ .paths.config }}/code/settings.json"
    content_source: "files/templates/vscode/settings.json.tmpl"
    render: true
    condition: "eq .Env.INSTALL_VSCODE \"true\""
  # Copy static configuration files without rendering
  - path: "{{ .paths.home }}/.vimrc"
    content_source: "files/configs/.vimrc"
    render: false
  - path: "{{ .paths.home }}/.editorconfig"
    content_source: "files/configs/.editorconfig"
    render: false

# Create symlinks for files that should be linked rather than copied
symlink:
  - src: "files/bin/custom-script.sh"
    dst: "{{ .paths.home }}/bin/custom-script"
    condition: "ne .Platform.OS \"windows\""
  - src: "files/configs/tmux.conf"
    dst: "{{ .paths.home }}/.tmux.conf"
    condition: "ne .Platform.OS \"windows\""

# Install packages
install:
  - ["git", "vim", "curl"]
  - packages: ["code"]
    condition: "eq .Env.INSTALL_VSCODE \"true\""
`

	indexPath := filepath.Join(targetDir, "jobs", "index.yaml")
	return os.WriteFile(indexPath, []byte(content), 0644)
}

// createSampleFiles creates sample template and config files
func createSampleFiles(targetDir string) error {
	templates := map[string]string{
		"files/templates/git/config.tmpl": `[user]
    name = {{ .user.name }}
    email = {{ .user.email }}

[init]
    defaultBranch = {{ .git.default_branch }}

[core]
    editor = {{ .editor.default }}
    autocrlf = {{ if eq .Platform.OS "windows" }}true{{ else }}input{{ end }}

[push]
    default = simple

[pull]
    rebase = false

[alias]
    st = status
    co = checkout
    br = branch
    ci = commit
    df = diff
    lg = log --oneline --graph --decorate --all
`,
		"files/templates/ssh/config.tmpl": `# SSH Configuration for {{ .user.name }}
# Generated by dotfiles manager

Host github.com
    HostName github.com
    User git
    Port 22
    IdentityFile {{ .paths.home }}/.ssh/id_{{ .ssh.key_type | default "ed25519" }}

Host *.example.com
    User {{ .user.name }}
    Port 22
    ForwardAgent yes
`,
		"files/templates/shell/bashrc.tmpl": `# {{ .user.name }}'s Bash Configuration
# Generated by dotfiles manager

# Aliases
{{ range $alias, $command := .shell.aliases }}alias {{ $alias }}="{{ $command }}"
{{ end }}

# Environment
export EDITOR="{{ .editor.default }}"
export PROJECTS_DIR="{{ .directories.projects }}"

# Prompt
PS1='\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ '

# Platform-specific settings
{{ if eq .Platform.OS "linux" }}
# Linux-specific bash settings
export PATH="$PATH:/usr/local/bin"
{{ end }}
`,
		"files/templates/shell/zshrc.tmpl": `# {{ .user.name }}'s Zsh Configuration
# Generated by dotfiles manager

# Aliases
{{ range $alias, $command := .shell.aliases }}alias {{ $alias }}="{{ $command }}"
{{ end }}

# Environment
export EDITOR="{{ .editor.default }}"
export PROJECTS_DIR="{{ .directories.projects }}"

# Oh My Zsh (if installed)
if [[ -d "$HOME/.oh-my-zsh" ]]; then
    export ZSH="$HOME/.oh-my-zsh"
    ZSH_THEME="robbyrussell"
    plugins=(git)
    source $ZSH/oh-my-zsh.sh
fi
`,
		"files/templates/shell/profile.ps1.tmpl": `# {{ .user.name }}'s PowerShell Profile
# Generated by dotfiles manager

# Aliases
{{ range $alias, $command := .shell.aliases }}Set-Alias {{ $alias }} "{{ $command }}"
{{ end }}

# Environment
$env:EDITOR = "{{ .editor.default }}"
$env:PROJECTS_DIR = "{{ .directories.projects }}"

# Functions
function Get-GitStatus { git status $args }
Set-Alias gs Get-GitStatus

# Prompt
function prompt {
    $currentPath = (Get-Location).Path.Replace($env:USERPROFILE, "~")
    Write-Host "PS " -NoNewline -ForegroundColor Green
    Write-Host $currentPath -NoNewline -ForegroundColor Blue
    Write-Host ">" -NoNewline -ForegroundColor Green
    return " "
}
`,
		"files/templates/editors/vimrc.tmpl": `" {{ .user.name }}'s Vim Configuration
" Generated by dotfiles manager

set number
set relativenumber
set tabstop=4
set shiftwidth=4
set expandtab
set autoindent
set smartindent
set hlsearch
set incsearch
set ignorecase
set smartcase

" Color scheme
syntax on
set background={{ .colors.theme }}

" Leader key
let mapleader = ","

" Basic mappings
nnoremap <leader>w :w<CR>
nnoremap <leader>q :q<CR>
nnoremap <leader>wq :wq<CR>
`,
		"files/templates/editors/vscode-settings.json.tmpl": `{
    "editor.fontSize": 14,
    "editor.fontFamily": "Fira Code, Consolas, 'Courier New', monospace",
    "editor.fontLigatures": true,
    "editor.tabSize": 4,
    "editor.insertSpaces": true,
    "editor.rulers": [80, 120],
    "workbench.colorTheme": "{{ if eq .colors.theme \"dark\" }}Dark+ (default dark){{ else }}Default Light+{{ end }}",
    "terminal.integrated.shell.{{ if eq .Platform.OS \"windows\" }}windows{{ else }}linux{{ end }}": "{{ if eq .Platform.OS \"windows\" }}powershell.exe{{ else }}/bin/bash{{ end }}",
    "git.enableSmartCommit": true,
    "git.confirmSync": false,
    "files.autoSave": "onFocusChange"
}`,
	}

	// Create template files
	for templatePath, content := range templates {
		fullPath := filepath.Join(targetDir, templatePath)
		if err := utils.EnsureDir(filepath.Dir(fullPath)); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	// Create sample config files
	configs := map[string]string{
		"files/configs/.vimrc": `" Basic Vim Configuration
set number
set tabstop=4
set shiftwidth=4
set expandtab
syntax on
`,
		"files/configs/.editorconfig": `# EditorConfig
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true
indent_style = space
indent_size = 4
`,
		"files/configs/tmux.conf": `# Tmux Configuration
# Basic tmux settings

# Set prefix to Ctrl-a
unbind C-b
set-option -g prefix C-a
bind-key C-a send-prefix

# Split panes using | and -
bind | split-window -h
bind - split-window -v
unbind '"'
unbind %

# Enable mouse mode
set -g mouse on

# Start windows and panes at 1, not 0
set -g base-index 1
setw -g pane-base-index 1
`,
		"files/bin/custom-script.sh": `#!/bin/bash
# Custom script example

show_help() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -v, --version  Show version"
}

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -v|--version)
            echo "custom-script v1.0.0"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
    shift
done

echo "Hello from custom script!"
`,
	}

	for configPath, content := range configs {
		fullPath := filepath.Join(targetDir, configPath)
		if err := utils.EnsureDir(filepath.Dir(fullPath)); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

// createSampleScripts creates sample scripts
func createSampleScripts(targetDir string) error {
	scripts := map[string]string{
		"setup.sh": `#!/bin/bash
# Setup script for Unix-like systems

echo "Setting up dotfiles..."

# Install common packages
if command -v apt-get &> /dev/null; then
    sudo apt-get update
    sudo apt-get install -y git vim curl wget
elif command -v brew &> /dev/null; then
    brew install git vim curl wget
fi

echo "Setup complete!"
`,
		"setup.ps1": `# Setup script for Windows

Write-Host "Setting up dotfiles..." -ForegroundColor Green

# Check if winget is available
if (Get-Command winget -ErrorAction SilentlyContinue) {
    Write-Host "Installing packages with winget..." -ForegroundColor Blue
    winget install Git.Git
    winget install vim.vim
    winget install Microsoft.VisualStudioCode
} else {
    Write-Host "winget not available, please install packages manually" -ForegroundColor Yellow
}

Write-Host "Setup complete!" -ForegroundColor Green
`,
	}

	for scriptName, content := range scripts {
		scriptPath := filepath.Join(targetDir, "scripts", scriptName)
		if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
			return err
		}
	}

	return nil
}

// createReadme creates a README.md file
func createReadme(targetDir string) error {
	content := `# My Dotfiles

Personal dotfiles configuration managed with [dotfiles-cp](https://github.com/vleeuwenmenno/dotfiles-cp).

## Structure
### **Structure**

- **dotfiles.yaml** - Main configuration file
- **variables/** - Variable definitions with platform-specific overrides
- **jobs/** - Job definitions using built-in modules
- **files/** - Template and static files
- **scripts/** - Setup and maintenance scripts

## Usage

### Apply Configuration
` + "```bash" + `
dotfiles apply
` + "```" + `

### Check Status
` + "```bash" + `
dotfiles status
` + "```" + `

### View Variables
` + "```bash" + `
dotfiles variables list
dotfiles variables get user.name
` + "```" + `

### Backup Current Config
` + "```bash" + `
dotfiles backup
` + "```" + `

## Customization

1. Edit variables in **variables/global.yaml** for global settings
2. Add platform-specific variables in **variables/platforms/**
3. Create new templates in **templates/**
4. Update **jobs/index.yaml** to include new jobs

## Platform Support

- ✅ Windows (PowerShell)
- ✅ macOS (Zsh)
- ✅ Linux (Bash)
`

	readmePath := filepath.Join(targetDir, "README.md")
	return os.WriteFile(readmePath, []byte(content), 0644)
}

// createGitignore creates a .gitignore file
func createGitignore(targetDir string) error {
	content := `# Backup files
*.backup
*.bak

# Temporary files
*.tmp
*.temp

# OS-specific files
.DS_Store
Thumbs.db
desktop.ini

# Editor files
.vscode/
.idea/
*.swp
*.swo

# Local environment overrides
variables/local.yaml
.env.local
`

	gitignorePath := filepath.Join(targetDir, ".gitignore")
	return os.WriteFile(gitignorePath, []byte(content), 0644)
}

// directoryExistsAndNotEmpty checks if a directory exists and is not empty
func directoryExistsAndNotEmpty(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	return len(entries) > 0
}

// showNextSteps displays helpful next steps after initialization
func showNextSteps(targetDir string) {
	log := logger.Get()

	log.Info().Msg("Next steps:")
	log.Info().Msg("1. Customize variables in variables/global.yaml")
	log.Info().Msg("2. Edit templates to match your preferences")
	log.Info().Msg("3. Run 'dotfiles validate' to check configuration")
	log.Info().Msg("4. Run 'dotfiles variables list' to see all variables")
	log.Info().Msg("5. Run 'dotfiles apply' to deploy your dotfiles")

	if targetDir != "." {
		log.Info().Str("directory", targetDir).Msg("Don't forget to cd into your dotfiles directory")
	}
}
