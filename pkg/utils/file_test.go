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

func TestCopyDir(t *testing.T) {
	tmpRoot, err := os.MkdirTemp("", "copy-dir-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpRoot)

	src := filepath.Join(tmpRoot, "src")
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(src, "top.txt"), []byte("top"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "child.txt"), []byte("child"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Symlink inside src that should be skipped (not followed) by CopyDir.
	external := filepath.Join(tmpRoot, "external.txt")
	if err := os.WriteFile(external, []byte("external"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(src, "link")); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(tmpRoot, "dst")
	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir error = %v", err)
	}

	// Top-level regular file copied with original mode.
	topInfo, err := os.Stat(filepath.Join(dst, "top.txt"))
	if err != nil {
		t.Fatalf("expected dst/top.txt: %v", err)
	}
	if topInfo.Mode().Perm() != 0o644 {
		t.Errorf("top.txt perm = %v, want 0644", topInfo.Mode().Perm())
	}

	// Nested file preserved with restrictive mode.
	childData, err := os.ReadFile(filepath.Join(dst, "nested", "child.txt"))
	if err != nil {
		t.Fatalf("expected dst/nested/child.txt: %v", err)
	}
	if string(childData) != "child" {
		t.Errorf("child contents = %q, want \"child\"", childData)
	}

	// Symlink should be skipped (not copied as file or symlink).
	if _, err := os.Lstat(filepath.Join(dst, "link")); !os.IsNotExist(err) {
		t.Errorf("expected symlink to be skipped, got err = %v", err)
	}
}

func TestCopyDir_NonDirectorySource(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "copy-dir-non-dir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	regular := filepath.Join(tmpDir, "regular.txt")
	if err := os.WriteFile(regular, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyDir(regular, filepath.Join(tmpDir, "dst")); err == nil {
		t.Errorf("CopyDir on regular file should fail, got nil")
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
