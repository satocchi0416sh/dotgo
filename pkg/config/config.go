package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the main dotgo configuration
type Config struct {
	Version    string                   `yaml:"version"`
	Repository RepositoryConfig         `yaml:"repository"`
	Profiles   map[string]ProfileConfig `yaml:"profiles"`
	Settings   SettingsConfig           `yaml:"settings"`
}

// RepositoryConfig defines repository settings
type RepositoryConfig struct {
	Type string `yaml:"type"` // local, git, etc.
	URL  string `yaml:"url,omitempty"`
}

// ProfileConfig defines a profile configuration
type ProfileConfig struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Packages    []string          `yaml:"packages"`
	Variables   map[string]any    `yaml:"variables"`
	Inherit     []string          `yaml:"inherit,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
}

// SettingsConfig defines global settings
type SettingsConfig struct {
	DefaultProfile   string               `yaml:"default_profile"`
	BackupDir        string               `yaml:"backup_dir"`
	SymlinkMode      string               `yaml:"symlink_mode"`  // auto, force, ask
	ConflictMode     string               `yaml:"conflict_mode"` // ask, overwrite, skip
	PackagesDir      string               `yaml:"packages_dir"`
	ProfilesDir      string               `yaml:"profiles_dir"`
	TemplatesDir     string               `yaml:"templates_dir"`
	ShellIntegration bool                 `yaml:"shell_integration"`
	XDGConfig        XDGConfigSettings    `yaml:"xdg_config"`
	ModernConfig     ModernConfigSettings `yaml:"modern_config"`
	TemplateEngine   TemplateSettings     `yaml:"template_engine"`
}

// XDGConfigSettings defines XDG Base Directory Specification settings
type XDGConfigSettings struct {
	Enabled    bool   `yaml:"enabled"`
	ConfigHome string `yaml:"config_home"` // Default: ~/.config
	DataHome   string `yaml:"data_home"`   // Default: ~/.local/share
	CacheHome  string `yaml:"cache_home"`  // Default: ~/.cache
	StateHome  string `yaml:"state_home"`  // Default: ~/.local/state
	RuntimeDir string `yaml:"runtime_dir"` // Default: /run/user/$UID
}

// ModernConfigSettings defines settings for modern development tools
type ModernConfigSettings struct {
	VSCodeSettings   bool `yaml:"vscode_settings"`
	ClaudeSettings   bool `yaml:"claude_settings"`
	CursorSettings   bool `yaml:"cursor_settings"`
	ContinueSettings bool `yaml:"continue_settings"`
	AIToolsSettings  bool `yaml:"ai_tools_settings"`
	DevToolsSettings bool `yaml:"dev_tools_settings"`
}

// TemplateSettings defines template engine configuration
type TemplateSettings struct {
	Enabled         bool           `yaml:"enabled"`
	Engine          string         `yaml:"engine"` // "go", "handlebars", "mustache"
	TemplateDelims  TemplateDelims `yaml:"template_delims"`
	GlobalVariables map[string]any `yaml:"global_variables"`
	Functions       []string       `yaml:"functions"`
}

// TemplateDelims defines custom template delimiters
type TemplateDelims struct {
	Left  string `yaml:"left"`
	Right string `yaml:"right"`
}

// PackageConfig represents a package configuration
type PackageConfig struct {
	Name         string            `yaml:"name"`
	Description  string            `yaml:"description"`
	Version      string            `yaml:"version,omitempty"`
	Dependencies []string          `yaml:"dependencies"`
	Files        []FileMapping     `yaml:"files"`
	Templates    []TemplateMapping `yaml:"templates,omitempty"`
	Commands     CommandsConfig    `yaml:"commands"`
	Variables    map[string]any    `yaml:"variables,omitempty"`
	Conditions   map[string]string `yaml:"conditions,omitempty"`
}

// FileMapping represents a file mapping
type FileMapping struct {
	Source     string `yaml:"source"`
	Target     string `yaml:"target"`
	Executable bool   `yaml:"executable,omitempty"`
	Template   bool   `yaml:"template,omitempty"`
	Condition  string `yaml:"condition,omitempty"`
}

// TemplateMapping represents a template file mapping
type TemplateMapping struct {
	Source    string         `yaml:"source"`
	Target    string         `yaml:"target"`
	Variables map[string]any `yaml:"variables,omitempty"`
	Condition string         `yaml:"condition,omitempty"`
}

// CommandsConfig defines pre/post install/remove commands
type CommandsConfig struct {
	PreInstall  []string `yaml:"pre_install"`
	PostInstall []string `yaml:"post_install"`
	PreRemove   []string `yaml:"pre_remove"`
	PostRemove  []string `yaml:"post_remove"`
}

// Manager handles configuration loading and management
type Manager struct {
	config    *Config
	rootDir   string
	configDir string
}

// NewManager creates a new configuration manager
func NewManager(rootDir string) *Manager {
	configDir := filepath.Join(rootDir, ".dotgo")
	return &Manager{
		rootDir:   rootDir,
		configDir: configDir,
	}
}

// Load loads the configuration from the config file
func (m *Manager) Load() error {
	configPath := filepath.Join(m.configDir, "config.yaml")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("dotgo not initialized in %s (run 'dotgo init' first)", m.rootDir)
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	m.setDefaults(&config)

	m.config = &config
	return nil
}

// GetConfig returns the loaded configuration
func (m *Manager) GetConfig() *Config {
	return m.config
}

// GetProfile returns a specific profile by name
func (m *Manager) GetProfile(name string) (*ProfileConfig, error) {
	if m.config == nil {
		return nil, fmt.Errorf("configuration not loaded")
	}

	profile, exists := m.config.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}

	return &profile, nil
}

// GetDefaultProfile returns the default profile
func (m *Manager) GetDefaultProfile() (*ProfileConfig, error) {
	if m.config == nil {
		return nil, fmt.Errorf("configuration not loaded")
	}

	return m.GetProfile(m.config.Settings.DefaultProfile)
}

// LoadPackageConfig loads a package configuration from file
func (m *Manager) LoadPackageConfig(packageName string) (*PackageConfig, error) {
	if m.config == nil {
		return nil, fmt.Errorf("configuration not loaded")
	}

	packageDir := filepath.Join(m.rootDir, m.config.Settings.PackagesDir, packageName)
	configPath := filepath.Join(packageDir, "package.yaml")

	// Check if package config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("package '%s' not found", packageName)
	}

	// Read package config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package config: %w", err)
	}

	// Parse YAML
	var packageConfig PackageConfig
	if err := yaml.Unmarshal(data, &packageConfig); err != nil {
		return nil, fmt.Errorf("failed to parse package config: %w", err)
	}

	return &packageConfig, nil
}

// ExpandPath expands ~ and environment variables in paths
func (m *Manager) ExpandPath(path string) string {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(homeDir, path[2:])
		}
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	return path
}

// GetPackageDir returns the full path to a package directory
func (m *Manager) GetPackageDir(packageName string) string {
	if m.config == nil {
		return ""
	}
	return filepath.Join(m.rootDir, m.config.Settings.PackagesDir, packageName)
}

// GetBackupDir returns the backup directory path
func (m *Manager) GetBackupDir() string {
	if m.config == nil {
		return ""
	}
	return filepath.Join(m.rootDir, m.config.Settings.BackupDir)
}

// ListPackages returns a list of available packages
func (m *Manager) ListPackages() ([]string, error) {
	if m.config == nil {
		return nil, fmt.Errorf("configuration not loaded")
	}

	packagesDir := filepath.Join(m.rootDir, m.config.Settings.PackagesDir)
	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read packages directory: %w", err)
	}

	var packages []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if package.yaml exists
			configPath := filepath.Join(packagesDir, entry.Name(), "package.yaml")
			if _, err := os.Stat(configPath); err == nil {
				packages = append(packages, entry.Name())
			}
		}
	}

	return packages, nil
}

// setDefaults sets default values for configuration
func (m *Manager) setDefaults(config *Config) {
	// Set default settings
	if config.Settings.DefaultProfile == "" {
		config.Settings.DefaultProfile = "default"
	}
	if config.Settings.BackupDir == "" {
		config.Settings.BackupDir = ".dotgo/backups"
	}
	if config.Settings.SymlinkMode == "" {
		config.Settings.SymlinkMode = "auto"
	}
	if config.Settings.ConflictMode == "" {
		config.Settings.ConflictMode = "ask"
	}
	if config.Settings.PackagesDir == "" {
		config.Settings.PackagesDir = "packages"
	}
	if config.Settings.ProfilesDir == "" {
		config.Settings.ProfilesDir = "profiles"
	}
	if config.Settings.TemplatesDir == "" {
		config.Settings.TemplatesDir = "templates"
	}

	// Set XDG defaults
	m.setXDGDefaults(&config.Settings.XDGConfig)
	
	// Set modern config defaults
	m.setModernConfigDefaults(&config.Settings.ModernConfig)
	
	// Set template engine defaults
	m.setTemplateDefaults(&config.Settings.TemplateEngine)

	// Initialize profiles if empty
	if config.Profiles == nil {
		config.Profiles = make(map[string]ProfileConfig)
	}

	// Ensure default profile exists
	if _, exists := config.Profiles["default"]; !exists {
		config.Profiles["default"] = ProfileConfig{
			Name:        "default",
			Description: "Default profile",
			Packages:    []string{},
			Variables:   make(map[string]any),
		}
	}
}

// setXDGDefaults sets default XDG Base Directory values
func (m *Manager) setXDGDefaults(xdg *XDGConfigSettings) {
	if xdg.ConfigHome == "" {
		xdg.ConfigHome = "${HOME}/.config"
	}
	if xdg.DataHome == "" {
		xdg.DataHome = "${HOME}/.local/share"
	}
	if xdg.CacheHome == "" {
		xdg.CacheHome = "${HOME}/.cache"
	}
	if xdg.StateHome == "" {
		xdg.StateHome = "${HOME}/.local/state"
	}
	if xdg.RuntimeDir == "" {
		xdg.RuntimeDir = "/run/user/${UID}"
	}
}

// setModernConfigDefaults sets defaults for modern config tools
func (m *Manager) setModernConfigDefaults(modern *ModernConfigSettings) {
	// Enable all modern config tools by default
	modern.VSCodeSettings = true
	modern.ClaudeSettings = true
	modern.CursorSettings = true
	modern.ContinueSettings = true
	modern.AIToolsSettings = true
	modern.DevToolsSettings = true
}

// setTemplateDefaults sets template engine defaults
func (m *Manager) setTemplateDefaults(tmpl *TemplateSettings) {
	tmpl.Enabled = true
	if tmpl.Engine == "" {
		tmpl.Engine = "go"
	}
	if tmpl.TemplateDelims.Left == "" {
		tmpl.TemplateDelims.Left = "{{"
	}
	if tmpl.TemplateDelims.Right == "" {
		tmpl.TemplateDelims.Right = "}}"
	}
	if tmpl.GlobalVariables == nil {
		tmpl.GlobalVariables = make(map[string]any)
	}
	if len(tmpl.Functions) == 0 {
		tmpl.Functions = []string{"env", "now", "default", "required"}
	}
}

// Save saves the current configuration to file
func (m *Manager) Save() error {
	if m.config == nil {
		return fmt.Errorf("no configuration to save")
	}

	configPath := filepath.Join(m.configDir, "config.yaml")

	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
