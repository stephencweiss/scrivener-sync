// Package config manages YAML configuration for Scrivener sync.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// ConfigDir returns the path to the global config directory (~/.scriv-sync/).
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".scriv-sync"), nil
}

// ConfigPath returns the path to the global config file.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// StatePath returns the path to a project's state file.
func StatePath(alias string) (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state", alias+".json"), nil
}

// GlobalConfig represents the global configuration with all project aliases.
type GlobalConfig struct {
	Version  string                    `yaml:"version"`
	Projects map[string]*ProjectConfig `yaml:"projects"`

	configPath string
}

// ProjectConfig represents a single project's sync configuration.
type ProjectConfig struct {
	LocalPath      string          `yaml:"local_path"`
	ScrivPath      string          `yaml:"scriv_path"`
	FolderMappings []FolderMapping `yaml:"folder_mappings"`
	Options        Options         `yaml:"options"`

	alias string
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

// LoadGlobal loads the global config from ~/.scriv-sync/config.yaml.
func LoadGlobal() (*GlobalConfig, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &GlobalConfig{
				Version:    "1.0",
				Projects:   make(map[string]*ProjectConfig),
				configPath: configPath,
			}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &GlobalConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.configPath = configPath

	// Initialize projects map if nil
	if cfg.Projects == nil {
		cfg.Projects = make(map[string]*ProjectConfig)
	}

	// Set alias on each project and apply defaults
	for alias, proj := range cfg.Projects {
		proj.alias = alias
		if proj.Options.DefaultConflictResolution == "" {
			proj.Options.DefaultConflictResolution = "prompt"
		}
		if proj.Options.DefaultDeletionAction == "" {
			proj.Options.DefaultDeletionAction = "prompt"
		}
	}

	return cfg, nil
}

// Save writes the global config to its file.
func (g *GlobalConfig) Save() error {
	if g.configPath == "" {
		path, err := ConfigPath()
		if err != nil {
			return err
		}
		g.configPath = path
	}

	// Ensure directory exists
	dir := filepath.Dir(g.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Also ensure state directory exists
	stateDir := filepath.Join(dir, "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := yaml.Marshal(g)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(g.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetProject returns the config for a specific project alias.
func (g *GlobalConfig) GetProject(alias string) (*ProjectConfig, error) {
	proj, exists := g.Projects[alias]
	if !exists {
		return nil, fmt.Errorf("project '%s' not found. Run 'scriv-sync list' to see available projects", alias)
	}
	proj.alias = alias
	return proj, nil
}

// AddProject adds a new project to the global config.
func (g *GlobalConfig) AddProject(alias, localPath, scrivPath string) *ProjectConfig {
	proj := &ProjectConfig{
		LocalPath:      localPath,
		ScrivPath:      scrivPath,
		FolderMappings: []FolderMapping{},
		Options:        DefaultOptions(),
		alias:          alias,
	}
	g.Projects[alias] = proj
	return proj
}

// ListProjects returns all project aliases sorted alphabetically.
func (g *GlobalConfig) ListProjects() []string {
	aliases := make([]string, 0, len(g.Projects))
	for alias := range g.Projects {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

// HasProject checks if a project alias exists.
func (g *GlobalConfig) HasProject(alias string) bool {
	_, exists := g.Projects[alias]
	return exists
}

// Validate checks the project config for errors.
func (p *ProjectConfig) Validate() []error {
	var errs []error

	if p.ScrivPath == "" {
		errs = append(errs, fmt.Errorf("scriv_path is required"))
	}

	if p.LocalPath == "" {
		errs = append(errs, fmt.Errorf("local_path is required"))
	}

	// Validate conflict resolution
	validConflict := map[string]bool{
		"prompt": true, "markdown": true, "scrivener": true, "skip": true,
	}
	if !validConflict[p.Options.DefaultConflictResolution] {
		errs = append(errs, fmt.Errorf("invalid default_conflict_resolution: %s", p.Options.DefaultConflictResolution))
	}

	// Validate deletion action
	validDeletion := map[string]bool{
		"prompt": true, "delete": true, "recreate": true, "skip": true,
	}
	if !validDeletion[p.Options.DefaultDeletionAction] {
		errs = append(errs, fmt.Errorf("invalid default_deletion_action: %s", p.Options.DefaultDeletionAction))
	}

	return errs
}

// ScrivenerPath returns the absolute path to the Scrivener project.
func (p *ProjectConfig) ScrivenerPath() (string, error) {
	if filepath.IsAbs(p.ScrivPath) {
		return p.ScrivPath, nil
	}

	// Resolve relative to local path
	absPath := filepath.Join(p.LocalPath, p.ScrivPath)

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
func (p *ProjectConfig) MarkdownPath() string {
	return p.LocalPath
}

// EnabledMappings returns only the folder mappings that have sync enabled.
func (p *ProjectConfig) EnabledMappings() []FolderMapping {
	var enabled []FolderMapping
	for _, mapping := range p.FolderMappings {
		if mapping.SyncEnabled {
			enabled = append(enabled, mapping)
		}
	}
	return enabled
}

// Alias returns the project's alias.
func (p *ProjectConfig) Alias() string {
	return p.alias
}

// AddMapping adds a folder mapping to the project config.
func (p *ProjectConfig) AddMapping(markdownDir, scrivenerFolder string, enabled bool) {
	p.FolderMappings = append(p.FolderMappings, FolderMapping{
		MarkdownDir:     markdownDir,
		ScrivenerFolder: scrivenerFolder,
		SyncEnabled:     enabled,
	})
}

// DefaultOptions returns the default option values.
func DefaultOptions() Options {
	return Options{
		CreateMissingFolders:      true,
		DefaultConflictResolution: "prompt",
		DefaultDeletionAction:     "prompt",
	}
}
