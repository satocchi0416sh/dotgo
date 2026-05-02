package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/satocchi0416sh/dotgo/internal/config"
)

func TestNewEngine(t *testing.T) {
	tests := []struct {
		name    string
		rootDir string
		dryRun  bool
		verbose bool
	}{
		{
			name:    "basic initialization",
			rootDir: "/test/root",
			dryRun:  false,
			verbose: false,
		},
		{
			name:    "with dry-run",
			rootDir: "/test/root",
			dryRun:  true,
			verbose: false,
		},
		{
			name:    "with verbose",
			rootDir: "/test/root",
			dryRun:  false,
			verbose: true,
		},
		{
			name:    "with both flags",
			rootDir: "/test/root",
			dryRun:  true,
			verbose: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := NewEngine(tt.rootDir, tt.dryRun, tt.verbose)
			
			if eng == nil {
				t.Fatal("NewEngine() returned nil")
			}
			if eng.configMgr == nil {
				t.Error("NewEngine() created engine with nil configMgr")
			}
			if eng.backupDir != filepath.Join(tt.rootDir, ".dotgo/backups") {
				t.Errorf("backupDir = %s, expected %s", 
					eng.backupDir, filepath.Join(tt.rootDir, ".dotgo/backups"))
			}
			if eng.dryRun != tt.dryRun {
				t.Errorf("dryRun = %v, expected %v", eng.dryRun, tt.dryRun)
			}
			if eng.verbose != tt.verbose {
				t.Errorf("verbose = %v, expected %v", eng.verbose, tt.verbose)
			}
		})
	}
}

func TestEngine_GetRootDir(t *testing.T) {
	rootDir := "/test/root"
	eng := NewEngine(rootDir, false, false)
	
	if got := eng.GetRootDir(); got != rootDir {
		t.Errorf("GetRootDir() = %s, expected %s", got, rootDir)
	}
}

func TestEngine_Add(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles map[string]string
		filePath   string
		tags       []string
		dryRun     bool
		wantErr    bool
	}{
		{
			name: "add existing file",
			setupFiles: map[string]string{
				"test.txt": "test content",
			},
			filePath: "test.txt",
			tags:     []string{"test"},
			dryRun:   false,
			wantErr:  false,
		},
		{
			name:       "add non-existent file",
			setupFiles: map[string]string{},
			filePath:   "nonexistent.txt",
			tags:       []string{"test"},
			dryRun:     false,
			wantErr:    true,
		},
		{
			name: "add with dry-run",
			setupFiles: map[string]string{
				"test.txt": "test content",
			},
			filePath: "test.txt",
			tags:     []string{"test"},
			dryRun:   true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directories
			tmpDir, err := os.MkdirTemp("", "engine-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			srcDir, err := os.MkdirTemp("", "engine-src")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(srcDir)

			// Setup source files
			for filename, content := range tt.setupFiles {
				path := filepath.Join(srcDir, filename)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Create engine
			eng := NewEngine(tmpDir, tt.dryRun, false)

			// Initialize config
			if err := eng.configMgr.Initialize(); err != nil {
				t.Fatal(err)
			}

			// Test Add
			filePath := filepath.Join(srcDir, tt.filePath)
			err = eng.Add(filePath, tt.tags)

			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !tt.dryRun {
				// Check if file was copied
				destFile := filepath.Join(tmpDir, "files", filepath.Base(tt.filePath))
				if _, err := os.Stat(destFile); os.IsNotExist(err) {
					t.Error("Add() did not copy file to destination")
				}

				// Check if manifest was updated
				manifest := eng.configMgr.GetManifest()
				if manifest == nil || manifest.Links == nil {
					t.Error("Add() did not update manifest")
				} else if _, exists := manifest.Links[filepath.Base(tt.filePath)]; !exists {
					t.Error("Add() did not add link to manifest")
				}
			}
		})
	}
}

func TestEngine_Status(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "status-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create engine
	eng := NewEngine(tmpDir, false, false)

	// Initialize config with some links
	if err := eng.configMgr.Initialize(); err != nil {
		t.Fatal(err)
	}

	// Add some test links
	eng.configMgr.AddLink(".zshrc", config.LinkSpec{Tags: []string{"common"}})
	eng.configMgr.AddLink(".vimrc", config.LinkSpec{Tags: []string{"vim"}})
	if err := eng.configMgr.Save(); err != nil {
		t.Fatal(err)
	}

	// Get status
	statuses, err := eng.Status([]string{"common"})
	if err != nil {
		t.Errorf("Status() error = %v", err)
	}

	if len(statuses) == 0 {
		t.Error("Status() returned empty list")
	}

	// Check status fields
	for _, status := range statuses {
		if status.TargetPath == "" {
			t.Error("Status has empty TargetPath")
		}
		if status.SourcePath == "" {
			t.Error("Status has empty SourcePath")
		}
	}
}

func TestEngine_Apply(t *testing.T) {
	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "apply-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	homeDir, err := os.MkdirTemp("", "apply-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(homeDir)

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", originalHome)

	// Create engine
	eng := NewEngine(tmpDir, false, false)

	// Initialize config
	if err := eng.configMgr.Initialize(); err != nil {
		t.Fatal(err)
	}

	// Create a test file in files directory
	filesDir := filepath.Join(tmpDir, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(filesDir, ".testrc")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Add link to manifest
	eng.configMgr.AddLink(".testrc", config.LinkSpec{Tags: []string{"test"}})
	if err := eng.configMgr.Save(); err != nil {
		t.Fatal(err)
	}

	// Test dry-run first
	eng.dryRun = true
	if err := eng.Apply([]string{"test"}); err != nil {
		t.Errorf("Apply() with dry-run error = %v", err)
	}

	// Verify no symlink created in dry-run
	targetPath := filepath.Join(homeDir, ".testrc")
	if _, err := os.Lstat(targetPath); !os.IsNotExist(err) {
		t.Error("Apply() created symlink in dry-run mode")
	}

	// Test actual apply
	eng.dryRun = false
	if err := eng.Apply([]string{"test"}); err != nil {
		t.Errorf("Apply() error = %v", err)
	}

	// Verify symlink was created
	info, err := os.Lstat(targetPath)
	if os.IsNotExist(err) {
		t.Error("Apply() did not create symlink")
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Error("Apply() created file instead of symlink")
	}
}

// TestEngine_applyLink_DirectorySource verifies that applyLink creates a
// symlink for a directory source. Previously dotgo rejected directory sources
// because the existence check used FileExists, which returns false for IsDir
// paths. This test calls applyLink directly to isolate the source-check fix
// from unrelated manifest persistence behavior covered by other tests.
func TestEngine_applyLink_DirectorySource(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "apply-dir-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	homeDir, err := os.MkdirTemp("", "apply-dir-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(homeDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", originalHome)

	eng := NewEngine(tmpDir, false, false)

	// Create a directory under files/ with a couple of nested entries.
	srcDir := filepath.Join(tmpDir, "files", ".config", "myapp")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "config.toml"), []byte("k = 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "nested"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := eng.applyLink(".config/myapp", config.LinkSpec{}); err != nil {
		t.Fatalf("applyLink() error = %v", err)
	}

	targetPath := filepath.Join(homeDir, ".config", "myapp")
	info, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatalf("expected symlink at %s, got error: %v", targetPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink, got mode %v", info.Mode())
	}

	resolved, err := os.Readlink(targetPath)
	if err != nil {
		t.Fatalf("Readlink error: %v", err)
	}
	if resolved != srcDir {
		t.Errorf("symlink target = %s, want %s", resolved, srcDir)
	}

	if _, err := os.Stat(filepath.Join(targetPath, "config.toml")); err != nil {
		t.Errorf("nested file unreachable through symlink: %v", err)
	}
}

func TestEngine_Remove(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "remove-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create engine
	eng := NewEngine(tmpDir, false, false)

	// Initialize config
	if err := eng.configMgr.Initialize(); err != nil {
		t.Fatal(err)
	}

	// Add a test link
	eng.configMgr.AddLink(".testrc", config.LinkSpec{Tags: []string{"test"}})
	if err := eng.configMgr.Save(); err != nil {
		t.Fatal(err)
	}

	// Test Remove
	err = eng.Remove(".testrc", false)
	if err != nil {
		t.Errorf("Remove() error = %v", err)
	}

	// Verify link was removed from manifest
	if _, err := eng.configMgr.GetLink(".testrc"); err == nil {
		t.Error("Remove() did not remove link from manifest")
	}

	// Test removing non-existent link
	err = eng.Remove(".nonexistent", false)
	if err == nil {
		t.Error("Remove() should error on non-existent link")
	}
}

// Benchmark tests
func BenchmarkEngine_Status(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	eng := NewEngine(tmpDir, false, false)
	eng.configMgr.Initialize()

	// Add many links
	for i := 0; i < 100; i++ {
		eng.configMgr.AddLink(filepath.Join(".config", string(rune(i))), 
			config.LinkSpec{Tags: []string{"test"}})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.Status([]string{"test"})
	}
}