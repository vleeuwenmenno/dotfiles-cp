package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vleeuwenmenno/dotfiles-cp/pkg/utils"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the minimal main configuration structure (dotfiles.yaml)
type Config struct {
	Metadata *Metadata `yaml:"metadata" json:"metadata"`
	Paths    *Paths    `yaml:"paths" json:"paths"`
	Settings *Settings `yaml:"settings" json:"settings"`
}

// Metadata contains information about the dotfiles repository
type Metadata struct {
	Name        string `yaml:"name" json:"name"`
	Version     string `yaml:"version" json:"version"`
	Author      string `yaml:"author" json:"author"`
	Description string `yaml:"description" json:"description"`
	Repository  string `yaml:"repository" json:"repository"`
}

// Paths contains configurable paths for variables, jobs, etc.
type Paths struct {
	VariablesDir   string `yaml:"variables_dir" json:"variables_dir"`
	VariablesIndex string `yaml:"variables_index" json:"variables_index"`
	JobsDir        string `yaml:"jobs_dir" json:"jobs_dir"`
	JobsIndex      string `yaml:"jobs_index" json:"jobs_index"`
	FilesDir       string `yaml:"files_dir" json:"files_dir"`
	ScriptsDir     string `yaml:"scripts_dir" json:"scripts_dir"`
	BackupDir      string `yaml:"backup_dir" json:"backup_dir"`
}

// Settings contains global configuration settings
type Settings struct {
	LogLevel      string `yaml:"log_level" json:"log_level"`
	DryRun        bool   `yaml:"dry_run" json:"dry_run"`
	CreateBackups bool   `yaml:"create_backups" json:"create_backups"`
	AutoUpdate    bool   `yaml:"auto_update" json:"auto_update"`
}

// ImportContext tracks import chain and provides context for processing
type ImportContext struct {
	ImportChain []string               // Breadcrumb trail for circular detection
	BasePath    string                 // Base directory for relative imports
	Variables   map[string]interface{} // Current variable state
	Config      *Config                // Reference to main config
}

// VariableSource tracks where a variable came from for debugging
type VariableSource struct {
	Key            string      `json:"key"`
	RawValue       interface{} `json:"raw_value"`       // Original template value
	ProcessedValue interface{} `json:"processed_value"` // Rendered template value
	Source         string      `json:"source"`          // File path where this variable was defined
	Line           int         `json:"line"`            // Line number in source file
}

// ImportFile represents a file that can be imported with conditions
type ImportFile struct {
	Path      string                 `yaml:"path" json:"path"`
	Condition string                 `yaml:"condition" json:"condition"`
	Variables map[string]interface{} `yaml:"variables" json:"variables"`
}

// VariableIndex represents the structure of variables/index.yaml
type VariableIndex struct {
	Imports   []ImportFile           `yaml:"imports" json:"imports"`
	Variables map[string]interface{} `yaml:"variables" json:"variables"`
}

// JobsIndex represents the structure of jobs/index.yaml
type JobsIndex struct {
	Jobs map[string]interface{} `yaml:",inline" json:"jobs"`
}

// Task represents a single task to be executed
type Task struct {
	ID     string                 `json:"id"`
	Action string                 `json:"action"`
	Config map[string]interface{} `json:"config"`
	Order  int                    `json:"order"`
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

// Platform represents platform-specific configuration (moved from main config)
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
	OS       string   `yaml:"os" json:"os"`
	Arch     string   `yaml:"arch" json:"arch"`
	Shell    string   `yaml:"shell" json:"shell"`
	Hostname string   `yaml:"hostname" json:"hostname"`
	Env      []string `yaml:"env" json:"env"`
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
		Paths: &Paths{
			VariablesDir:   "variables",
			VariablesIndex: "index.yaml",
			JobsDir:        "jobs",
			JobsIndex:      "index.yaml",
			FilesDir:       "files",
			ScriptsDir:     "scripts",
			BackupDir:      "~/.dotfiles-backup",
		},
		Settings: &Settings{
			LogLevel:      "info",
			DryRun:        false,
			CreateBackups: true,
			AutoUpdate:    false,
		},
	}
}

// NewImportContext creates a new import context
func NewImportContext(config *Config, basePath string) *ImportContext {
	return &ImportContext{
		ImportChain: make([]string, 0),
		BasePath:    basePath,
		Variables:   make(map[string]interface{}),
		Config:      config,
	}
}

// AddToChain adds a file to the import chain for circular detection
func (ctx *ImportContext) AddToChain(filePath string) error {
	// Check for circular imports
	for _, existing := range ctx.ImportChain {
		if existing == filePath {
			return fmt.Errorf("circular import detected: %s -> %s",
				strings.Join(ctx.ImportChain, " -> "), filePath)
		}
	}

	ctx.ImportChain = append(ctx.ImportChain, filePath)
	return nil
}

// RemoveFromChain removes the last file from the import chain
func (ctx *ImportContext) RemoveFromChain() {
	if len(ctx.ImportChain) > 0 {
		ctx.ImportChain = ctx.ImportChain[:len(ctx.ImportChain)-1]
	}
}

// GetImportChainString returns a formatted string of the current import chain
func (ctx *ImportContext) GetImportChainString() string {
	return strings.Join(ctx.ImportChain, " -> ")
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

	if c.Paths == nil {
		return fmt.Errorf("paths section is required")
	}

	if c.Settings == nil {
		return fmt.Errorf("settings section is required")
	}

	if c.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	if c.Paths.VariablesDir == "" {
		return fmt.Errorf("paths.variables_dir is required")
	}

	if c.Paths.JobsDir == "" {
		return fmt.Errorf("paths.jobs_dir is required")
	}

	if c.Paths.VariablesIndex == "" {
		return fmt.Errorf("paths.variables_index is required")
	}

	if c.Paths.JobsIndex == "" {
		return fmt.Errorf("paths.jobs_index is required")
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

// GetVariablesPath returns the full path to the variables directory
func (c *Config) GetVariablesPath(basePath string) string {
	return filepath.Join(basePath, c.Paths.VariablesDir)
}

// GetVariablesIndexPath returns the full path to the variables index file
func (c *Config) GetVariablesIndexPath(basePath string) string {
	return filepath.Join(basePath, c.Paths.VariablesDir, c.Paths.VariablesIndex)
}

// GetJobsPath returns the full path to the jobs directory
func (c *Config) GetJobsPath(basePath string) string {
	return filepath.Join(basePath, c.Paths.JobsDir)
}

// GetJobsIndexPath returns the full path to the jobs index file
func (c *Config) GetJobsIndexPath(basePath string) string {
	return filepath.Join(basePath, c.Paths.JobsDir, c.Paths.JobsIndex)
}

// GetFilesPath returns the full path to the files directory
func (c *Config) GetFilesPath(basePath string) string {
	return filepath.Join(basePath, c.Paths.FilesDir)
}

// GetScriptsPath returns the full path to the scripts directory
func (c *Config) GetScriptsPath(basePath string) string {
	return filepath.Join(basePath, c.Paths.ScriptsDir)
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

// LoadVariableIndex loads and parses a variables index file
func LoadVariableIndex(indexPath string) (*VariableIndex, error) {
	if !utils.FileExists(indexPath) {
		return nil, fmt.Errorf("variables index file does not exist: %s", indexPath)
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read variables index file: %w", err)
	}

	var index VariableIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variables index: %w", err)
	}

	return &index, nil
}

// LoadJobsIndex loads and parses a jobs index file
func LoadJobsIndex(indexPath string) (*JobsIndex, error) {
	if !utils.FileExists(indexPath) {
		return nil, fmt.Errorf("jobs index file does not exist: %s", indexPath)
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read jobs index file: %w", err)
	}

	var index JobsIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse jobs index: %w", err)
	}

	return &index, nil
}
