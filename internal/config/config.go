package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Manifest represents the single source of truth (dotgo.yaml)
type Manifest struct {
	Version  int                 `yaml:"version"`
	Settings Settings            `yaml:"settings"`
	Links    map[string]LinkSpec `yaml:"links"` // Key is relative path from HOME
}

type Settings struct {
	DefaultTags []string `yaml:"default_tags"`
}

type LinkSpec struct {
	Tags  []string          `yaml:"tags,omitempty"`
	Hooks map[string]string `yaml:"hooks,omitempty"` // e.g., {"post_apply": "source ~/.zshrc"}
}

// ShouldApply determines if a link should be created based on current OS and requested tags
func (l *LinkSpec) ShouldApply(requestedTags []string) bool {
	// If no tags specified, apply by default
	if len(l.Tags) == 0 {
		return true
	}

	// Check if current OS is in tags (implicit OS filtering)
	currentOS := runtime.GOOS
	hasOSTag := false
	for _, tag := range l.Tags {
		if tag == currentOS {
			hasOSTag = true
			break
		}
	}

	// If OS-specific tags exist but current OS isn't included, don't apply
	osTagsExist := false
	for _, tag := range l.Tags {
		if tag == "darwin" || tag == "linux" || tag == "windows" {
			osTagsExist = true
			break
		}
	}

	if osTagsExist && !hasOSTag {
		return false
	}

	// If no requested tags, apply if no non-OS tags or current OS matches
	if len(requestedTags) == 0 {
		// Apply if no non-OS tags exist, or if OS tag matches
		for _, tag := range l.Tags {
			if tag != "darwin" && tag != "linux" && tag != "windows" {
				return false // Has non-OS tags, but none requested
			}
		}
		return true
	}

	// Check if any requested tag matches
	tagMap := make(map[string]bool)
	for _, tag := range requestedTags {
		tagMap[tag] = true
	}

	for _, tag := range l.Tags {
		if tagMap[tag] {
			return true
		}
	}

	return false
}

// Manager handles the dotgo.yaml manifest
type Manager struct {
	rootDir      string
	manifestPath string
	manifest     *Manifest
}

// NewManager creates a new configuration manager
func NewManager(rootDir string) *Manager {
	return &Manager{
		rootDir:      rootDir,
		manifestPath: filepath.Join(rootDir, "dotgo.yaml"),
	}
}

// Load loads the manifest from dotgo.yaml
func (m *Manager) Load() error {
	// Check if manifest file exists
	if _, err := os.Stat(m.manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("dotgo.yaml not found in %s", m.rootDir)
	}

	// Read manifest file
	data, err := os.ReadFile(m.manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read dotgo.yaml: %w", err)
	}

	// Parse YAML
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse dotgo.yaml: %w", err)
	}

	// Set defaults
	m.setDefaults(&manifest)

	m.manifest = &manifest
	return nil
}

// Save saves the current manifest to dotgo.yaml
func (m *Manager) Save() error {
	if m.manifest == nil {
		return fmt.Errorf("no manifest to save")
	}

	data, err := yaml.Marshal(m.manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(m.manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write dotgo.yaml: %w", err)
	}

	return nil
}

// GetManifest returns the loaded manifest
func (m *Manager) GetManifest() *Manifest {
	return m.manifest
}

// AddLink adds a new link to the manifest
func (m *Manager) AddLink(targetPath string, spec LinkSpec) error {
	if m.manifest == nil {
		return fmt.Errorf("manifest not loaded")
	}

	if m.manifest.Links == nil {
		m.manifest.Links = make(map[string]LinkSpec)
	}

	m.manifest.Links[targetPath] = spec
	return nil
}

// RemoveLink removes a link from the manifest
func (m *Manager) RemoveLink(targetPath string) error {
	if m.manifest == nil {
		return fmt.Errorf("manifest not loaded")
	}

	if m.manifest.Links == nil {
		return fmt.Errorf("link not found: %s", targetPath)
	}

	if _, exists := m.manifest.Links[targetPath]; !exists {
		return fmt.Errorf("link not found: %s", targetPath)
	}

	delete(m.manifest.Links, targetPath)
	return nil
}

// GetLink returns a specific link spec
func (m *Manager) GetLink(targetPath string) (LinkSpec, error) {
	if m.manifest == nil {
		return LinkSpec{}, fmt.Errorf("manifest not loaded")
	}

	if m.manifest.Links == nil {
		return LinkSpec{}, fmt.Errorf("link not found: %s", targetPath)
	}

	spec, exists := m.manifest.Links[targetPath]
	if !exists {
		return LinkSpec{}, fmt.Errorf("link not found: %s", targetPath)
	}

	return spec, nil
}

// ListLinks returns all links, optionally filtered by tags
func (m *Manager) ListLinks(requestedTags []string) map[string]LinkSpec {
	if m.manifest == nil || m.manifest.Links == nil {
		return make(map[string]LinkSpec)
	}

	result := make(map[string]LinkSpec)
	for path, spec := range m.manifest.Links {
		if spec.ShouldApply(requestedTags) {
			result[path] = spec
		}
	}

	return result
}

// Initialize creates a new dotgo.yaml with default settings
func (m *Manager) Initialize() error {
	// Check if manifest already exists
	if _, err := os.Stat(m.manifestPath); err == nil {
		return fmt.Errorf("dotgo.yaml already exists in %s", m.rootDir)
	}

	manifest := Manifest{
		Version: 1,
		Settings: Settings{
			DefaultTags: []string{},
		},
		Links: make(map[string]LinkSpec),
	}

	m.setDefaults(&manifest)
	m.manifest = &manifest

	return m.Save()
}

// setDefaults sets default values for the manifest
func (m *Manager) setDefaults(manifest *Manifest) {
	if manifest.Version == 0 {
		manifest.Version = 1
	}

	if manifest.Links == nil {
		manifest.Links = make(map[string]LinkSpec)
	}

	if manifest.Settings.DefaultTags == nil {
		manifest.Settings.DefaultTags = []string{}
	}
}

// GetRootDir returns the root directory
func (m *Manager) GetRootDir() string {
	return m.rootDir
}

// GetManifestPath returns the path to the manifest file
func (m *Manager) GetManifestPath() string {
	return m.manifestPath
}
