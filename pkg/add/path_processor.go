package add

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProcessedPath represents a processed file path with validation
type ProcessedPath struct {
	Source   string // Expanded absolute path
	Original string // Original input path
	IsValid  bool   // Whether the path is valid and accessible
}

// PathProcessor handles path processing and package name inference
type PathProcessor struct{}

// NewPathProcessor creates a new PathProcessor instance
func NewPathProcessor() *PathProcessor {
	return &PathProcessor{}
}

// ProcessPath expands and validates a file path
func (p *PathProcessor) ProcessPath(inputPath string) (*ProcessedPath, error) {
	if inputPath == "" {
		return nil, fmt.Errorf("input path cannot be empty")
	}

	// Expand tilde and environment variables
	expandedPath, err := p.expandPath(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to expand path '%s': %w", inputPath, err)
	}

	// Convert to absolute path
	absolutePath, err := filepath.Abs(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for '%s': %w", expandedPath, err)
	}

	// Check if file exists and is accessible
	isValid := p.isValidFile(absolutePath)

	return &ProcessedPath{
		Source:   absolutePath,
		Original: inputPath,
		IsValid:  isValid,
	}, nil
}

// expandPath expands tilde and environment variables in a path
func (p *PathProcessor) expandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Expand tilde
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[2:])
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	return path, nil
}

// isValidFile checks if a file exists and is readable
func (p *PathProcessor) isValidFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// InferPackageName infers a package name from a file path
func (p *PathProcessor) InferPackageName(filePath string) string {
	// Get the base name of the file
	fileName := filepath.Base(filePath)
	
	// Remove common dotfile prefixes and extensions
	packageName := p.extractPackageNameFromFile(fileName)
	
	// Try to infer from parent directory if filename doesn't give good result
	if p.isGenericName(packageName) {
		if dirName := p.inferFromDirectory(filePath); dirName != "" {
			packageName = dirName
		}
	}
	
	// Fallback to cleaned filename
	if packageName == "" {
		packageName = p.cleanPackageName(fileName)
	}
	
	return packageName
}

// extractPackageNameFromFile extracts package name from filename
func (p *PathProcessor) extractPackageNameFromFile(fileName string) string {
	// Handle common dotfile patterns
	patterns := map[string]string{
		".vimrc":           "vim",
		".zshrc":           "zsh",
		".bashrc":          "bash",
		".gitconfig":       "git",
		".tmux.conf":       "tmux",
		".eslintrc":        "eslint",
		".prettierrc":      "prettier",
		"starship.toml":    "starship",
		"alacritty.yml":    "alacritty",
		"alacritty.yaml":   "alacritty",
		"kitty.conf":       "kitty",
		"init.vim":         "vim",
		"init.lua":         "nvim",
		".ideavimrc":       "ideavim",
	}
	
	// Check exact matches first
	if pkg, exists := patterns[fileName]; exists {
		return pkg
	}
	
	// Check patterns with extensions
	for pattern, pkg := range patterns {
		if strings.HasSuffix(fileName, pattern) {
			return pkg
		}
	}
	
	// Remove dots and extract base name
	name := strings.TrimPrefix(fileName, ".")
	
	// Remove common extensions
	extensions := []string{".conf", ".config", ".toml", ".yaml", ".yml", ".json", ".rc"}
	for _, ext := range extensions {
		if strings.HasSuffix(name, ext) {
			name = strings.TrimSuffix(name, ext)
			break
		}
	}
	
	return name
}

// inferFromDirectory tries to infer package name from parent directory
func (p *PathProcessor) inferFromDirectory(filePath string) string {
	dir := filepath.Dir(filePath)
	
	// Common config directory patterns
	configDirs := []string{".config", "config"}
	
	for _, configDir := range configDirs {
		if strings.Contains(dir, configDir) {
			// Extract directory after config directory
			parts := strings.Split(dir, string(filepath.Separator))
			for i, part := range parts {
				if part == configDir && i+1 < len(parts) {
					return parts[i+1]
				}
			}
		}
	}
	
	// Check if in a hidden directory that might indicate package name
	parts := strings.Split(dir, string(filepath.Separator))
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if strings.HasPrefix(part, ".") && len(part) > 1 {
			// Remove dot prefix and return as package name
			return part[1:]
		}
	}
	
	return ""
}

// isGenericName checks if the extracted name is too generic
func (p *PathProcessor) isGenericName(name string) bool {
	genericNames := []string{"config", "conf", "rc", "init", "settings", "dot", "file"}
	name = strings.ToLower(name)
	
	for _, generic := range genericNames {
		if name == generic {
			return true
		}
	}
	
	return len(name) < 2 // Very short names are likely generic
}

// cleanPackageName cleans up a package name to be valid
func (p *PathProcessor) cleanPackageName(name string) string {
	// Remove dots, spaces, and other problematic characters
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ToLower(name)
	
	// Remove leading/trailing dashes
	name = strings.Trim(name, "-")
	
	// Ensure it's not empty
	if name == "" {
		name = "config"
	}
	
	return name
}

// GetRelativeToHome returns path relative to home directory if possible
func (p *PathProcessor) GetRelativeToHome(absolutePath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	relPath, err := filepath.Rel(homeDir, absolutePath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}
	
	// If the relative path goes outside home directory, return absolute path
	if strings.HasPrefix(relPath, "..") {
		return absolutePath, nil
	}
	
	return relPath, nil
}