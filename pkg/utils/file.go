package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies a file from src to dst, preserving permissions.
// If dst already exists, it will be overwritten.
func CopyFile(src, dst string) error {
	// Expand paths
	srcExpanded, err := ExpandPath(src)
	if err != nil {
		return fmt.Errorf("failed to expand source path: %w", err)
	}

	dstExpanded, err := ExpandPath(dst)
	if err != nil {
		return fmt.Errorf("failed to expand destination path: %w", err)
	}

	// Get source file info
	srcInfo, err := os.Stat(srcExpanded)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("source is not a regular file: %s", srcExpanded)
	}

	// Open source file
	srcFile, err := os.Open(srcExpanded)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Ensure destination directory exists
	dstDir := filepath.Dir(dstExpanded)
	if err := EnsureDir(dstDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create destination file
	dstFile, err := os.Create(dstExpanded)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	// Set the same permissions as source
	if err := os.Chmod(dstExpanded, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set permissions on destination file: %w", err)
	}

	return nil
}

// CopyFileWithBackup copies a file from src to dst, creating a backup of dst if it exists.
// The backup file will have a .backup extension.
func CopyFileWithBackup(src, dst string) (string, error) {
	dstExpanded, err := ExpandPath(dst)
	if err != nil {
		return "", fmt.Errorf("failed to expand destination path: %w", err)
	}

	var backupPath string

	// Check if destination exists
	if _, err := os.Stat(dstExpanded); err == nil {
		// Create backup
		backupPath = dstExpanded + ".backup"
		if err := CopyFile(dstExpanded, backupPath); err != nil {
			return "", fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Copy file
	if err := CopyFile(src, dst); err != nil {
		// Restore backup if copy failed and backup was created
		if backupPath != "" {
			_ = CopyFile(backupPath, dstExpanded)
			_ = os.Remove(backupPath)
		}
		return "", err
	}

	return backupPath, nil
}

// FileExists checks if a file exists and is not a directory
func FileExists(path string) (bool, error) {
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return false, fmt.Errorf("failed to expand path: %w", err)
	}

	info, err := os.Stat(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return !info.IsDir(), nil
}

// DirExists checks if a directory exists
func DirExists(path string) (bool, error) {
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return false, fmt.Errorf("failed to expand path: %w", err)
	}

	info, err := os.Stat(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return info.IsDir(), nil
}

// ReadFile reads the entire contents of a file
func ReadFile(path string) ([]byte, error) {
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to expand path: %w", err)
	}

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// WriteFile writes data to a file, creating it if necessary
func WriteFile(path string, data []byte, perm os.FileMode) error {
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return fmt.Errorf("failed to expand path: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(expandedPath)
	if err := EnsureDir(dir); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(expandedPath, data, perm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// RemoveFile removes a file if it exists
func RemoveFile(path string) error {
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return fmt.Errorf("failed to expand path: %w", err)
	}

	if err := os.Remove(expandedPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove file: %w", err)
		}
	}

	return nil
}

// IsSymlink checks if a path is a symbolic link
func IsSymlink(path string) (bool, error) {
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return false, fmt.Errorf("failed to expand path: %w", err)
	}

	info, err := os.Lstat(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return info.Mode()&os.ModeSymlink != 0, nil
}
