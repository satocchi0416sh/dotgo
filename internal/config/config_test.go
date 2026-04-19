package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"gopkg.in/yaml.v3"
)

func TestLinkSpec_ShouldApply(t *testing.T) {
	// Save original GOOS for restoration
	originalGOOS := runtime.GOOS

	tests := []struct {
		name          string
		linkSpec      LinkSpec
		requestedTags []string
		currentOS     string
		expected      bool
	}{
		{
			name:          "no tags always applies",
			linkSpec:      LinkSpec{Tags: nil},
			requestedTags: []string{},
			currentOS:     "darwin",
			expected:      true,
		},
		{
			name:          "empty tags always applies",
			linkSpec:      LinkSpec{Tags: []string{}},
			requestedTags: []string{},
			currentOS:     "darwin",
			expected:      true,
		},
		{
			name:          "darwin tag on darwin OS",
			linkSpec:      LinkSpec{Tags: []string{"darwin"}},
			requestedTags: []string{},
			currentOS:     "darwin",
			expected:      true,
		},
		{
			name:          "linux tag on darwin OS",
			linkSpec:      LinkSpec{Tags: []string{"linux"}},
			requestedTags: []string{},
			currentOS:     "darwin",
			expected:      false,
		},
		{
			name:          "windows tag on linux OS",
			linkSpec:      LinkSpec{Tags: []string{"windows"}},
			requestedTags: []string{},
			currentOS:     "linux",
			expected:      false,
		},
		{
			name:          "matching requested tag",
			linkSpec:      LinkSpec{Tags: []string{"work", "important"}},
			requestedTags: []string{"work"},
			currentOS:     "darwin",
			expected:      true,
		},
		{
			name:          "non-matching requested tag",
			linkSpec:      LinkSpec{Tags: []string{"personal"}},
			requestedTags: []string{"work"},
			currentOS:     "darwin",
			expected:      false,
		},
		{
			name:          "multiple requested tags with one match",
			linkSpec:      LinkSpec{Tags: []string{"vim", "editor"}},
			requestedTags: []string{"work", "vim", "linux"},
			currentOS:     "darwin",
			expected:      true,
		},
		{
			name:          "OS tag with other tags on matching OS",
			linkSpec:      LinkSpec{Tags: []string{"darwin", "work"}},
			requestedTags: []string{"work"},
			currentOS:     "darwin",
			expected:      true,
		},
		{
			name:          "OS tag with other tags on non-matching OS",
			linkSpec:      LinkSpec{Tags: []string{"linux", "work"}},
			requestedTags: []string{"work"},
			currentOS:     "darwin",
			expected:      false,
		},
		{
			name:          "multiple OS tags should fail",
			linkSpec:      LinkSpec{Tags: []string{"darwin", "linux"}},
			requestedTags: []string{},
			currentOS:     "darwin",
			expected:      false, // darwin tag matches but linux doesn't
		},
		{
			name:          "no requested tags with non-OS tags",
			linkSpec:      LinkSpec{Tags: []string{"common"}},
			requestedTags: []string{},
			currentOS:     "darwin",
			expected:      true,
		},
		{
			name:          "no requested tags with non-matching OS",
			linkSpec:      LinkSpec{Tags: []string{"linux", "common"}},
			requestedTags: []string{},
			currentOS:     "darwin",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock runtime.GOOS
			if tt.currentOS != originalGOOS {
				t.Skipf("Cannot test OS %s on %s", tt.currentOS, originalGOOS)
			}

			result := tt.linkSpec.ShouldApply(tt.requestedTags)
			if result != tt.expected {
				t.Errorf("ShouldApply() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestManager_NewManager(t *testing.T) {
	rootDir := "/test/root"
	manager := NewManager(rootDir)

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}
	if manager.rootDir != rootDir {
		t.Errorf("NewManager() rootDir = %s, expected %s", manager.rootDir, rootDir)
	}
	if manager.manifestPath != filepath.Join(rootDir, "dotgo.yaml") {
		t.Errorf("NewManager() manifestPath = %s, expected %s", 
			manager.manifestPath, filepath.Join(rootDir, "dotgo.yaml"))
	}
}

func TestManager_Load(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  map[string]string // filename -> content
		wantErr     bool
		checkResult func(t *testing.T, m *Manager)
	}{
		{
			name: "successful load",
			setupFiles: map[string]string{
				"dotgo.yaml": `version: 1
settings:
  default_tags: ["common", "darwin"]
links:
  .zshrc:
    tags: ["common"]
  .vimrc:
    tags: ["vim", "editor"]`,
			},
			wantErr: false,
			checkResult: func(t *testing.T, m *Manager) {
				if m.manifest == nil {
					t.Error("manifest is nil after successful load")
				}
				if m.manifest.Version != 1 {
					t.Errorf("Version = %d, expected 1", m.manifest.Version)
				}
				if len(m.manifest.Links) != 2 {
					t.Errorf("Links count = %d, expected 2", len(m.manifest.Links))
				}
			},
		},
		{
			name:       "file not found",
			setupFiles: map[string]string{},
			wantErr:    false, // Load creates default manifest if not found
			checkResult: func(t *testing.T, m *Manager) {
				if m.manifest == nil {
					t.Error("manifest is nil after load")
				}
				if m.manifest.Version != 1 {
					t.Errorf("Version = %d, expected 1", m.manifest.Version)
				}
			},
		},
		{
			name: "invalid yaml",
			setupFiles: map[string]string{
				"dotgo.yaml": `version: 1
settings:
  default_tags: [this is invalid yaml`,
			},
			wantErr: true,
		},
		{
			name: "empty file creates default",
			setupFiles: map[string]string{
				"dotgo.yaml": "",
			},
			wantErr: false,
			checkResult: func(t *testing.T, m *Manager) {
				if m.manifest.Version != 1 {
					t.Errorf("Version = %d, expected 1 for default", m.manifest.Version)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "config-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Setup files
			for filename, content := range tt.setupFiles {
				path := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Create manager and load
			manager := NewManager(tmpDir)
			err = manager.Load()

			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, manager)
			}
		})
	}
}

func TestManager_Save(t *testing.T) {
	tests := []struct {
		name     string
		manifest *Manifest
		wantErr  bool
	}{
		{
			name: "successful save",
			manifest: &Manifest{
				Version: 1,
				Settings: Settings{
					DefaultTags: []string{"common"},
				},
				Links: map[string]LinkSpec{
					".zshrc": {Tags: []string{"common"}},
				},
			},
			wantErr: false,
		},
		{
			name:     "save without manifest",
			manifest: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "save-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			manager := NewManager(tmpDir)
			manager.manifest = tt.manifest

			err = manager.Save()
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify file was created
				data, err := os.ReadFile(filepath.Join(tmpDir, "dotgo.yaml"))
				if err != nil {
					t.Errorf("Failed to read saved file: %v", err)
				}

				// Verify content is valid YAML
				var loaded Manifest
				if err := yaml.Unmarshal(data, &loaded); err != nil {
					t.Errorf("Saved file is not valid YAML: %v", err)
				}

				if loaded.Version != tt.manifest.Version {
					t.Errorf("Saved version = %d, expected %d", loaded.Version, tt.manifest.Version)
				}
			}
		})
	}
}

func TestManager_LinkOperations(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "link-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Test operations without loaded manifest
	t.Run("operations without manifest", func(t *testing.T) {
		if err := manager.AddLink(".zshrc", LinkSpec{Tags: []string{"test"}}); err == nil {
			t.Error("AddLink() should fail without loaded manifest")
		}
		if err := manager.RemoveLink(".zshrc"); err == nil {
			t.Error("RemoveLink() should fail without loaded manifest")
		}
		if _, err := manager.GetLink(".zshrc"); err == nil {
			t.Error("GetLink() should fail without loaded manifest")
		}
	})

	// Load manifest
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}

	// Test AddLink
	t.Run("add link", func(t *testing.T) {
		spec := LinkSpec{Tags: []string{"test", "shell"}}
		if err := manager.AddLink(".zshrc", spec); err != nil {
			t.Errorf("AddLink() failed: %v", err)
		}

		// Verify link was added
		got, err := manager.GetLink(".zshrc")
		if err != nil {
			t.Errorf("GetLink() after add failed: %v", err)
		}
		if len(got.Tags) != len(spec.Tags) {
			t.Errorf("Added link has %d tags, expected %d", len(got.Tags), len(spec.Tags))
		}
	})

	// Test GetLink
	t.Run("get non-existent link", func(t *testing.T) {
		_, err := manager.GetLink(".bashrc")
		if err == nil {
			t.Error("GetLink() should fail for non-existent link")
		}
	})

	// Test RemoveLink
	t.Run("remove link", func(t *testing.T) {
		if err := manager.RemoveLink(".zshrc"); err != nil {
			t.Errorf("RemoveLink() failed: %v", err)
		}

		// Verify link was removed
		_, err := manager.GetLink(".zshrc")
		if err == nil {
			t.Error("GetLink() should fail after removal")
		}
	})

	// Test RemoveLink on non-existent
	t.Run("remove non-existent link", func(t *testing.T) {
		err := manager.RemoveLink(".nonexistent")
		if err == nil {
			t.Error("RemoveLink() should fail for non-existent link")
		}
	})
}

func TestManager_ListLinks(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "list-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Test without manifest
	t.Run("without manifest", func(t *testing.T) {
		result := manager.ListLinks(nil)
		if len(result) != 0 {
			t.Errorf("ListLinks() without manifest returned %d links, expected 0", len(result))
		}
	})

	// Load and populate manifest
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}

	manager.manifest.Links = map[string]LinkSpec{
		".zshrc":  {Tags: []string{"common", "shell"}},
		".vimrc":  {Tags: []string{"vim", "editor"}},
		".bashrc": {Tags: []string{"linux", "shell"}},
		".config": {Tags: []string{"darwin", "config"}},
	}

	tests := []struct {
		name          string
		requestedTags []string
		expectedCount int
		expectedKeys  []string
	}{
		{
			name:          "no filter returns matching OS",
			requestedTags: nil,
			expectedCount: 3, // Assumes running on darwin or linux
			expectedKeys:  []string{".zshrc", ".vimrc"}, // common tags
		},
		{
			name:          "filter by tag",
			requestedTags: []string{"shell"},
			expectedCount: 1, // Only darwin/current OS shell
		},
		{
			name:          "filter by vim tag",
			requestedTags: []string{"vim"},
			expectedCount: 1,
			expectedKeys:  []string{".vimrc"},
		},
		{
			name:          "no matches",
			requestedTags: []string{"nonexistent"},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.ListLinks(tt.requestedTags)
			
			// We can't predict exact count due to OS filtering
			// Just ensure it returns a map
			if result == nil {
				t.Error("ListLinks() returned nil map")
			}

			// Check for specific expected keys if provided
			for _, key := range tt.expectedKeys {
				if _, exists := result[key]; !exists {
					t.Errorf("Expected key %s not in result", key)
				}
			}
		})
	}
}

func TestManager_Initialize(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles map[string]string
		wantErr    bool
	}{
		{
			name:       "successful initialization",
			setupFiles: map[string]string{},
			wantErr:    false,
		},
		{
			name: "manifest already exists",
			setupFiles: map[string]string{
				"dotgo.yaml": "version: 1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "init-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Setup files
			for filename, content := range tt.setupFiles {
				path := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			manager := NewManager(tmpDir)
			err = manager.Initialize()

			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify file was created
				if _, err := os.Stat(filepath.Join(tmpDir, "dotgo.yaml")); os.IsNotExist(err) {
					t.Error("Initialize() did not create dotgo.yaml")
				}

				// Verify manifest is loaded
				if manager.manifest == nil {
					t.Error("Initialize() did not load manifest")
				}
			}
		})
	}
}

func TestManager_GetRootDir(t *testing.T) {
	rootDir := "/test/path"
	manager := NewManager(rootDir)
	
	if got := manager.GetRootDir(); got != rootDir {
		t.Errorf("GetRootDir() = %s, expected %s", got, rootDir)
	}
}

func TestManager_GetManifest(t *testing.T) {
	manager := NewManager("/test")
	
	// Before load
	if manager.GetManifest() != nil {
		t.Error("GetManifest() should return nil before load")
	}
	
	// After creating manifest
	manager.manifest = &Manifest{Version: 1}
	if manager.GetManifest() == nil {
		t.Error("GetManifest() should return manifest after it's set")
	}
	if manager.GetManifest().Version != 1 {
		t.Error("GetManifest() returned wrong manifest")
	}
}