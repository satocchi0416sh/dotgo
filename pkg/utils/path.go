package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/satocchi0416sh/dotgo/internal/errors"
)

// ExpandPath expands tilde (~) to home directory and environment variables in a path.
// It returns an error if the home directory cannot be determined when needed.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Expand tilde to home directory
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf(errors.ErrHomeDir, err)
		}
		path = filepath.Join(homeDir, path[2:])
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	// Clean the path to remove redundant separators and relative paths
	return filepath.Clean(path), nil
}

// MustExpandPath is like ExpandPath but panics if an error occurs.
// Use this only when you're certain the path expansion will succeed.
func MustExpandPath(path string) string {
	expanded, err := ExpandPath(path)
	if err != nil {
		panic(fmt.Sprintf("path expansion failed: %v", err))
	}
	return expanded
}

// GetHomeDir returns the user's home directory with consistent error handling
func GetHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf(errors.ErrHomeDir, err)
	}
	return homeDir, nil
}

// GetHomeRelativePath converts an absolute path to a home-relative path if possible.
// For example, /home/user/file becomes ~/file
func GetHomeRelativePath(path string) (string, error) {
	homeDir, err := GetHomeDir()
	if err != nil {
		return path, nil // Return original path if home dir cannot be determined
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return path, nil // Return original path if absolute path cannot be determined
	}

	// Check if the path is under the home directory
	if strings.HasPrefix(absPath, homeDir+string(filepath.Separator)) {
		relPath, err := filepath.Rel(homeDir, absPath)
		if err != nil {
			return path, nil
		}
		return "~/" + relPath, nil
	}

	return path, nil
}

// IsHomeRelativePath checks if a path starts with ~/
func IsHomeRelativePath(path string) bool {
	return strings.HasPrefix(path, "~/")
}

// GetWorkingDirectory gets the current working directory with proper error handling
func GetWorkingDirectory() (string, error) {
	rootDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf(errors.ErrCurrentDir, err)
	}
	return rootDir, nil
}

// EnsureDir creates a directory if it doesn't exist, including all parent directories
func EnsureDir(path string) error {
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return fmt.Errorf(errors.ErrExpandPath, err)
	}

	if err := os.MkdirAll(expandedPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", expandedPath, err)
	}

	return nil
}

// JoinAndExpand joins path elements and expands the result
func JoinAndExpand(elem ...string) (string, error) {
	joined := filepath.Join(elem...)
	return ExpandPath(joined)
}
