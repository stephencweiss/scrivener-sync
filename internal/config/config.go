// Package config manages YAML configuration for Scrivener sync.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the sync configuration.
type Config struct {
	Version          string          `yaml:"version"`
	ScrivenerProject string          `yaml:"scrivener_project"`
	MarkdownRoot     string          `yaml:"markdown_root"`
	FolderMappings   []FolderMapping `yaml:"folder_mappings"`
	Options          Options         `yaml:"options"`

	configPath string
}

// FolderMapping defines a mapping between markdown directory and Scrivener folder.
type FolderMapping struct {
	MarkdownDir     string `yaml:"markdown_dir"`
	ScrivenerFolder string `yaml:"scrivener_folder"`
	SyncEnabled     bool   `yaml:"sync_enabled"`
}

// Options contains sync behavior options.
type Options struct {
	CreateMissingFolders      bool   `yaml:"create_missing_folders"`
	DefaultConflictResolution string `yaml:"default_conflict_resolution"` // prompt | markdown | scrivener | skip
	DefaultDeletionAction     string `yaml:"default_deletion_action"`     // prompt | delete | recreate | skip
}

// Load reads and parses a config file from the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.configPath = path

	// Apply defaults for missing options
	if cfg.Options.DefaultConflictResolution == "" {
		cfg.Options.DefaultConflictResolution = "prompt"
	}
	if cfg.Options.DefaultDeletionAction == "" {
		cfg.Options.DefaultDeletionAction = "prompt"
	}

	return cfg, nil
}

// Save writes the config back to its file.
func (c *Config) Save() error {
	if c.configPath == "" {
		return fmt.Errorf("config path not set")
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(c.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks the config for errors and returns a list of validation errors.
func (c *Config) Validate() []error {
	var errs []error

	if c.ScrivenerProject == "" {
		errs = append(errs, fmt.Errorf("scrivener_project is required"))
	}

	if c.MarkdownRoot == "" {
		errs = append(errs, fmt.Errorf("markdown_root is required"))
	}

	// Validate conflict resolution
	validConflict := map[string]bool{
		"prompt": true, "markdown": true, "scrivener": true, "skip": true,
	}
	if !validConflict[c.Options.DefaultConflictResolution] {
		errs = append(errs, fmt.Errorf("invalid default_conflict_resolution: %s (must be prompt|markdown|scrivener|skip)", c.Options.DefaultConflictResolution))
	}

	// Validate deletion action
	validDeletion := map[string]bool{
		"prompt": true, "delete": true, "recreate": true, "skip": true,
	}
	if !validDeletion[c.Options.DefaultDeletionAction] {
		errs = append(errs, fmt.Errorf("invalid default_deletion_action: %s (must be prompt|delete|recreate|skip)", c.Options.DefaultDeletionAction))
	}

	// Validate folder mappings
	for i, mapping := range c.FolderMappings {
		if mapping.MarkdownDir == "" {
			errs = append(errs, fmt.Errorf("folder_mappings[%d]: markdown_dir is required", i))
		}
		if mapping.ScrivenerFolder == "" {
			errs = append(errs, fmt.Errorf("folder_mappings[%d]: scrivener_folder is required", i))
		}
	}

	return errs
}

// ScrivenerPath returns the absolute path to the Scrivener project.
func (c *Config) ScrivenerPath() (string, error) {
	if filepath.IsAbs(c.ScrivenerProject) {
		return c.ScrivenerProject, nil
	}

	// Resolve relative to config file directory
	configDir := filepath.Dir(c.configPath)
	absPath := filepath.Join(configDir, c.ScrivenerProject)

	// Verify it exists
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("scrivener project not found: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("scrivener project must be a directory: %s", absPath)
	}

	return absPath, nil
}

// MarkdownPath returns the absolute path to the markdown root.
func (c *Config) MarkdownPath() (string, error) {
	if filepath.IsAbs(c.MarkdownRoot) {
		return c.MarkdownRoot, nil
	}

	// Resolve relative to config file directory
	configDir := filepath.Dir(c.configPath)
	absPath := filepath.Join(configDir, c.MarkdownRoot)

	return absPath, nil
}

// EnabledMappings returns only the folder mappings that have sync enabled.
func (c *Config) EnabledMappings() []FolderMapping {
	var enabled []FolderMapping
	for _, mapping := range c.FolderMappings {
		if mapping.SyncEnabled {
			enabled = append(enabled, mapping)
		}
	}
	return enabled
}

// CreateDefault creates a new config with default values.
func CreateDefault(scrivPath, configPath string) *Config {
	return &Config{
		Version:          "1.0",
		ScrivenerProject: scrivPath,
		MarkdownRoot:     ".",
		FolderMappings:   []FolderMapping{},
		Options:          DefaultOptions(),
		configPath:       configPath,
	}
}

// DefaultOptions returns the default option values.
func DefaultOptions() Options {
	return Options{
		CreateMissingFolders:      true,
		DefaultConflictResolution: "prompt",
		DefaultDeletionAction:     "prompt",
	}
}

// SetPath sets the config file path for saving.
func (c *Config) SetPath(path string) {
	c.configPath = path
}

// Path returns the config file path.
func (c *Config) Path() string {
	return c.configPath
}

// AddMapping adds a folder mapping to the config.
func (c *Config) AddMapping(markdownDir, scrivenerFolder string, enabled bool) {
	c.FolderMappings = append(c.FolderMappings, FolderMapping{
		MarkdownDir:     markdownDir,
		ScrivenerFolder: scrivenerFolder,
		SyncEnabled:     enabled,
	})
}
