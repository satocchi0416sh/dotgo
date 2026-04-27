package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "path-exists-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	regularFile := filepath.Join(tmpDir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	symToFile := filepath.Join(tmpDir, "sym-file")
	if err := os.Symlink(regularFile, symToFile); err != nil {
		t.Fatal(err)
	}

	symToDir := filepath.Join(tmpDir, "sym-dir")
	if err := os.Symlink(subDir, symToDir); err != nil {
		t.Fatal(err)
	}

	dangling := filepath.Join(tmpDir, "dangling")
	if err := os.Symlink(filepath.Join(tmpDir, "missing"), dangling); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"regular file", regularFile, true},
		{"directory", subDir, true},
		{"symlink to file", symToFile, true},
		{"symlink to directory", symToDir, true},
		{"dangling symlink", dangling, true},
		{"missing path", filepath.Join(tmpDir, "missing"), false},
		{"missing under missing parent", filepath.Join(tmpDir, "missing", "x"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PathExists(tt.path)
			if err != nil {
				t.Fatalf("PathExists(%s) error = %v", tt.path, err)
			}
			if got != tt.want {
				t.Errorf("PathExists(%s) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestFileExists_DirectoryReturnsFalse(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file-exists-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	got, err := FileExists(tmpDir)
	if err != nil {
		t.Fatalf("FileExists(%s) error = %v", tmpDir, err)
	}
	if got {
		t.Errorf("FileExists(%s) on directory = true, want false", tmpDir)
	}
}

func TestDirExists_RegularFileReturnsFalse(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dir-exists-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	regularFile := filepath.Join(tmpDir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := DirExists(regularFile)
	if err != nil {
		t.Fatalf("DirExists(%s) error = %v", regularFile, err)
	}
	if got {
		t.Errorf("DirExists(%s) on regular file = true, want false", regularFile)
	}
}
