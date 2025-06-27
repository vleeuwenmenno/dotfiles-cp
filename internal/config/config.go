package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vleeuwenmenno/dotfiles-cp/pkg/utils"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Metadata  *Metadata              `yaml:"metadata" json:"metadata"`
	Settings  *Settings              `yaml:"settings" json:"settings"`
	Variables map[string]interface{} `yaml:"variables" json:"variables"`
	Platforms []*Platform            `yaml:"platforms" json:"platforms"`
}

// Metadata contains information about the dotfiles repository
type Metadata struct {
	Name        string `yaml:"name" json:"name"`
	Version     string `yaml:"version" json:"version"`
	Author      string `yaml:"author" json:"author"`
	Description string `yaml:"description" json:"description"`
	Repository  string `yaml:"repository" json:"repository"`
}

// Settings contains global configuration settings
type Settings struct {
	BackupDir   string `yaml:"backup_dir" json:"backup_dir"`
	TemplateDir string `yaml:"template_dir" json:"template_dir"`
	TargetDir   string `yaml:"target_dir" json:"target_dir"`
	ConfigFile  string `yaml:"config_file" json:"config_file"`
	LogLevel    string `yaml:"log_level" json:"log_level"`
	DryRun      bool   `yaml:"dry_run" json:"dry_run"`
}

// Platform represents a platform-specific configuration
type Platform struct {
	Name       string                 `yaml:"name" json:"name"`
	Conditions *Conditions            `yaml:"conditions" json:"conditions"`
	Variables  map[string]interface{} `yaml:"variables" json:"variables"`
	Packages   map[string][]string    `yaml:"packages" json:"packages"`
	Files      []*FileMapping         `yaml:"files" json:"files"`
	Scripts    *Scripts               `yaml:"scripts" json:"scripts"`
}

// Conditions define when a platform configuration should be applied
type Conditions struct {
	OS    string   `yaml:"os" json:"os"`
	Arch  string   `yaml:"arch" json:"arch"`
	Shell string   `yaml:"shell" json:"shell"`
	Env   []string `yaml:"env" json:"env"`
}

// FileMapping defines how a source file should be mapped to a target location
type FileMapping struct {
	Source     string                 `yaml:"source" json:"source"`
	Target     string                 `yaml:"target" json:"target"`
	Template   bool                   `yaml:"template" json:"template"`
	Executable bool                   `yaml:"executable" json:"executable"`
	Backup     bool                   `yaml:"backup" json:"backup"`
	Condition  string                 `yaml:"condition" json:"condition"`
	Variables  map[string]interface{} `yaml:"variables" json:"variables"`
}

// Scripts define pre/post installation scripts
type Scripts struct {
	PreInstall  []string `yaml:"pre_install" json:"pre_install"`
	PostInstall []string `yaml:"post_install" json:"post_install"`
	PreUpdate   []string `yaml:"pre_update" json:"pre_update"`
	PostUpdate  []string `yaml:"post_update" json:"post_update"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Metadata: &Metadata{
			Name:        "My Dotfiles",
			Version:     "1.0.0",
			Author:      "User",
			Description: "Personal dotfiles configuration",
		},
		Settings: &Settings{
			BackupDir:   "~/.dotfiles-backup",
			TemplateDir: "templates",
			TargetDir:   "~",
			ConfigFile:  "dotfiles.yaml",
			LogLevel:    "info",
			DryRun:      false,
		},
		Variables: make(map[string]interface{}),
		Platforms: []*Platform{},
	}
}

// Load loads configuration from a file
func Load(configPath string) (*Config, error) {
	// Expand path
	expandedPath, err := utils.ExpandPath(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to expand config path: %w", err)
	}

	// Check if config file exists
	if !utils.FileExists(expandedPath) {
		return nil, fmt.Errorf("config file does not exist: %s", expandedPath)
	}

	// Initialize viper
	v := viper.New()
	v.SetConfigFile(expandedPath)
	v.SetConfigType("yaml")

	// Read config
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into config struct
	config := DefaultConfig()
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Save saves the configuration to a file
func (c *Config) Save(configPath string) error {
	// Expand path
	expandedPath, err := utils.ExpandPath(configPath)
	if err != nil {
		return fmt.Errorf("failed to expand config path: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(expandedPath)
	if err := utils.EnsureDir(dir); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(expandedPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Metadata == nil {
		return fmt.Errorf("metadata section is required")
	}

	if c.Settings == nil {
		return fmt.Errorf("settings section is required")
	}

	if c.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	if c.Settings.TemplateDir == "" {
		return fmt.Errorf("settings.template_dir is required")
	}

	if c.Settings.TargetDir == "" {
		return fmt.Errorf("settings.target_dir is required")
	}

	// Validate platforms
	for i, platform := range c.Platforms {
		if err := platform.Validate(); err != nil {
			return fmt.Errorf("platform[%d]: %w", i, err)
		}
	}

	return nil
}

// Validate validates a platform configuration
func (p *Platform) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("platform name is required")
	}

	// Validate file mappings
	for i, file := range p.Files {
		if err := file.Validate(); err != nil {
			return fmt.Errorf("file[%d]: %w", i, err)
		}
	}

	return nil
}

// Validate validates a file mapping
func (f *FileMapping) Validate() error {
	if f.Source == "" {
		return fmt.Errorf("source is required")
	}

	if f.Target == "" {
		return fmt.Errorf("target is required")
	}

	return nil
}

// GetPlatformForConditions returns the first platform that matches the given conditions
func (c *Config) GetPlatformForConditions(os, arch, shell string, env map[string]string) *Platform {
	for _, platform := range c.Platforms {
		if platform.MatchesConditions(os, arch, shell, env) {
			return platform
		}
	}
	return nil
}

// MatchesConditions checks if the platform matches the given conditions
func (p *Platform) MatchesConditions(os, arch, shell string, env map[string]string) bool {
	if p.Conditions == nil {
		return true // No conditions means it applies to all platforms
	}

	// Check OS condition
	if p.Conditions.OS != "" && p.Conditions.OS != os {
		return false
	}

	// Check architecture condition
	if p.Conditions.Arch != "" && p.Conditions.Arch != arch {
		return false
	}

	// Check shell condition
	if p.Conditions.Shell != "" && p.Conditions.Shell != shell {
		return false
	}

	// Check environment variable conditions
	for _, envVar := range p.Conditions.Env {
		if _, exists := env[envVar]; !exists {
			return false
		}
	}

	return true
}

// GetAllVariables returns all variables merged from global and platform-specific
func (c *Config) GetAllVariables(platform *Platform) map[string]interface{} {
	variables := make(map[string]interface{})

	// Start with global variables
	for k, v := range c.Variables {
		variables[k] = v
	}

	// Override with platform-specific variables
	if platform != nil {
		for k, v := range platform.Variables {
			variables[k] = v
		}
	}

	return variables
}

// FindConfigFile searches for a configuration file in common locations
func FindConfigFile() (string, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Search locations in order of preference
	searchPaths := []string{
		filepath.Join(cwd, "dotfiles.yaml"),
		filepath.Join(cwd, "dotfiles.yml"),
		filepath.Join(cwd, ".dotfiles.yaml"),
		filepath.Join(cwd, ".dotfiles.yml"),
		filepath.Join(homeDir, ".dotfiles", "dotfiles.yaml"),
		filepath.Join(homeDir, ".dotfiles", "dotfiles.yml"),
		filepath.Join(homeDir, ".config", "dotfiles", "dotfiles.yaml"),
		filepath.Join(homeDir, ".config", "dotfiles", "dotfiles.yml"),
	}

	for _, path := range searchPaths {
		if utils.FileExists(path) {
			return path, nil
		}
	}

	return "", fmt.Errorf("no configuration file found in common locations")
}
