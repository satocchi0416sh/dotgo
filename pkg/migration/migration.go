package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"

	"dotgo/pkg/config"
)

// Migrator handles migration from traditional dotfiles to dotgo
type Migrator struct {
	rootDir string
	verbose bool
	dryRun  bool
}

// NewMigrator creates a new migration handler
func NewMigrator(rootDir string, verbose, dryRun bool) *Migrator {
	return &Migrator{
		rootDir: rootDir,
		verbose: verbose,
		dryRun:  dryRun,
	}
}

// AnalysisResult contains the analysis of existing dotfiles
type AnalysisResult struct {
	RootDir           string
	DotFiles          []string
	ConfigDirs        []string
	Scripts           []string
	InstallScript     string
	BrewFile          string
	SuggestedPackages []SuggestedPackage
	BackupNeeded      []string
}

// SuggestedPackage represents a suggested package configuration
type SuggestedPackage struct {
	Name        string
	Description string
	Files       []config.FileMapping
	Commands    []string
}

// AnalyzeExisting analyzes the existing dotfiles structure
func (m *Migrator) AnalyzeExisting() (*AnalysisResult, error) {
	result := &AnalysisResult{
		RootDir: m.rootDir,
	}

	// Find dotfiles in root directory
	entries, err := os.ReadDir(m.rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read root directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(m.rootDir, name)

		// Skip hidden directories and known non-dotfile directories
		if strings.HasPrefix(name, ".") && (name == ".git" || name == ".dotgo") {
			continue
		}

		if entry.IsDir() {
			// Check for config directories
			if isConfigDirectory(name) {
				result.ConfigDirs = append(result.ConfigDirs, fullPath)
			}
		} else {
			// Check for dotfiles (start with . or common dotfile names)
			if isDotfile(name) {
				result.DotFiles = append(result.DotFiles, fullPath)
			}

			// Check for special files
			switch name {
			case "install.sh", "setup.sh", "bootstrap.sh":
				result.InstallScript = fullPath
				result.Scripts = append(result.Scripts, fullPath)
			case "Brewfile", "Brewfile.lock.json":
				result.BrewFile = fullPath
			default:
				if strings.HasSuffix(name, ".sh") {
					result.Scripts = append(result.Scripts, fullPath)
				}
			}
		}
	}

	// Generate suggested packages
	result.SuggestedPackages = m.generateSuggestedPackages(result)

	// Identify files that need backup
	result.BackupNeeded = m.identifyBackupFiles(result)

	return result, nil
}

// Migrate performs the actual migration
func (m *Migrator) Migrate(analysis *AnalysisResult, preserveOriginal bool) error {
	fmt.Printf("%s Starting migration to dotgo structure...\n", color.BlueString("🔄"))

	// Create dotgo structure if it doesn't exist
	if err := m.createDotgoStructure(); err != nil {
		return fmt.Errorf("failed to create dotgo structure: %w", err)
	}

	// Create packages based on suggestions
	for _, pkg := range analysis.SuggestedPackages {
		if err := m.createPackage(pkg, preserveOriginal); err != nil {
			fmt.Printf("%s Failed to create package '%s': %v\n",
				color.RedString("✗"), pkg.Name, err)
			continue
		}
		fmt.Printf("%s Created package: %s\n",
			color.GreenString("✓"), pkg.Name)
	}

	// Create migration script if install.sh exists
	if analysis.InstallScript != "" {
		if err := m.createMigrationScript(analysis.InstallScript); err != nil {
			fmt.Printf("%s Failed to create migration script: %v\n",
				color.YellowString("⚠️"), err)
		} else {
			fmt.Printf("%s Created migration script\n", color.GreenString("✓"))
		}
	}

	// Update main configuration
	if err := m.updateMainConfig(analysis); err != nil {
		return fmt.Errorf("failed to update main config: %w", err)
	}

	fmt.Printf("\n%s Migration completed successfully!\n", color.GreenString("🎉"))
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Review generated packages in packages/ directory")
	fmt.Println("2. Edit package configurations as needed")
	fmt.Println("3. Test with 'dotgo install --dry-run'")
	fmt.Println("4. Run 'dotgo install' to apply your dotfiles")
	if !preserveOriginal {
		fmt.Println("5. Remove old dotfiles once you're satisfied")
	}

	return nil
}

// generateSuggestedPackages creates package suggestions based on analysis
func (m *Migrator) generateSuggestedPackages(analysis *AnalysisResult) []SuggestedPackage {
	var packages []SuggestedPackage

	// Group files by logical packages
	packageGroups := make(map[string][]string)

	// Analyze dotfiles
	for _, dotfile := range analysis.DotFiles {
		name := filepath.Base(dotfile)
		packageName := m.inferPackageName(name)
		packageGroups[packageName] = append(packageGroups[packageName], dotfile)
	}

	// Analyze config directories
	for _, configDir := range analysis.ConfigDirs {
		name := filepath.Base(configDir)
		packageName := name

		// Find files in config directory
		files, err := m.findFilesInDir(configDir, "")
		if err != nil {
			continue
		}

		packageGroups[packageName] = append(packageGroups[packageName], files...)
	}

	// Create package suggestions
	for packageName, files := range packageGroups {
		pkg := SuggestedPackage{
			Name:        packageName,
			Description: fmt.Sprintf("Configuration for %s", packageName),
		}

		// Create file mappings
		for _, file := range files {
			relPath, err := filepath.Rel(m.rootDir, file)
			if err != nil {
				continue
			}

			target := m.inferTargetPath(file)
			if target != "" {
				pkg.Files = append(pkg.Files, config.FileMapping{
					Source: relPath,
					Target: target,
				})
			}
		}

		// Add commands based on package type
		pkg.Commands = m.inferCommands(packageName, files)

		if len(pkg.Files) > 0 {
			packages = append(packages, pkg)
		}
	}

	return packages
}

// createDotgoStructure creates the basic dotgo directory structure
func (m *Migrator) createDotgoStructure() error {
	dirs := []string{
		".dotgo",
		"packages",
		"profiles",
		"templates",
	}

	for _, dir := range dirs {
		path := filepath.Join(m.rootDir, dir)
		if m.dryRun {
			fmt.Printf("[DRY-RUN] Would create directory: %s\n", path)
			continue
		}

		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	return nil
}

// createPackage creates a package directory and configuration
func (m *Migrator) createPackage(pkg SuggestedPackage, preserveOriginal bool) error {
	packageDir := filepath.Join(m.rootDir, "packages", pkg.Name)

	if m.dryRun {
		fmt.Printf("[DRY-RUN] Would create package: %s\n", packageDir)
		return nil
	}

	// Create package directory
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return err
	}

	// Copy or move files
	for _, fileMapping := range pkg.Files {
		sourcePath := filepath.Join(m.rootDir, fileMapping.Source)
		targetPath := filepath.Join(packageDir, filepath.Base(fileMapping.Source))

		if preserveOriginal {
			if err := m.copyFile(sourcePath, targetPath); err != nil {
				return fmt.Errorf("failed to copy %s: %w", fileMapping.Source, err)
			}
		} else {
			if err := os.Rename(sourcePath, targetPath); err != nil {
				return fmt.Errorf("failed to move %s: %w", fileMapping.Source, err)
			}
		}

		// Update source path in mapping
		fileMapping.Source = filepath.Base(fileMapping.Source)
	}

	// Create package config
	packageConfig := config.PackageConfig{
		Name:        pkg.Name,
		Description: pkg.Description,
		Files:       pkg.Files,
		Commands: config.CommandsConfig{
			PostInstall: pkg.Commands,
		},
	}

	configPath := filepath.Join(packageDir, "package.yaml")
	return m.writePackageConfig(configPath, packageConfig)
}

// Helper methods

func (m *Migrator) inferPackageName(filename string) string {
	// Remove leading dots
	name := strings.TrimPrefix(filename, ".")

	// Map common patterns
	patterns := map[string]string{
		"zshrc":        "zsh",
		"bashrc":       "bash",
		"bash_profile": "bash",
		"vimrc":        "vim",
		"tmux.conf":    "tmux",
		"gitconfig":    "git",
		"gitignore":    "git",
	}

	if packageName, exists := patterns[name]; exists {
		return packageName
	}

	// Extract base name
	if idx := strings.Index(name, "."); idx > 0 {
		return name[:idx]
	}

	return name
}

func (m *Migrator) inferTargetPath(filePath string) string {
	filename := filepath.Base(filePath)

	// Handle config directory files
	if strings.Contains(filePath, "/.config/") || strings.HasPrefix(filepath.Base(filepath.Dir(filePath)), ".config") {
		relPath, _ := filepath.Rel(filepath.Join(m.rootDir, ".config"), filePath)
		return filepath.Join("~/.config", relPath)
	}

	// Handle dotfiles in root
	if strings.HasPrefix(filename, ".") {
		return filepath.Join("~", filename)
	}

	// Default to home directory
	return filepath.Join("~", "."+filename)
}

func (m *Migrator) inferCommands(packageName string, files []string) []string {
	var commands []string

	// Add common post-install commands based on package type
	switch packageName {
	case "zsh":
		commands = append(commands, "echo 'Restart your shell or run: source ~/.zshrc'")
	case "bash":
		commands = append(commands, "echo 'Restart your shell or run: source ~/.bashrc'")
	case "vim":
		commands = append(commands, "echo 'Run :PlugInstall in vim if using plugins'")
	case "tmux":
		commands = append(commands, "echo 'Restart tmux sessions to apply new config'")
	}

	return commands
}

func (m *Migrator) findFilesInDir(dir, prefix string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			// Recursively find files in subdirectories
			subFiles, err := m.findFilesInDir(fullPath, filepath.Join(prefix, entry.Name()))
			if err == nil {
				files = append(files, subFiles...)
			}
		} else {
			files = append(files, fullPath)
		}
	}

	return files, nil
}

func (m *Migrator) identifyBackupFiles(analysis *AnalysisResult) []string {
	var backupNeeded []string

	// Files that typically exist in home directory
	for _, dotfile := range analysis.DotFiles {
		filename := filepath.Base(dotfile)
		homePath := filepath.Join(os.Getenv("HOME"), filename)

		if _, err := os.Stat(homePath); err == nil {
			backupNeeded = append(backupNeeded, homePath)
		}
	}

	return backupNeeded
}

func (m *Migrator) createMigrationScript(installScript string) error {
	// Read the original install script
	content, err := os.ReadFile(installScript)
	if err != nil {
		return err
	}

	// Create a simple migration notice
	migrationScript := fmt.Sprintf(`#!/bin/bash
# Migration script - original functionality preserved for reference
# This script has been migrated to dotgo packages
#
# To use the new dotgo system:
#   dotgo install
#
# Original script content below:
# ================================

%s`, string(content))

	scriptPath := filepath.Join(m.rootDir, "legacy-install.sh")
	if m.dryRun {
		fmt.Printf("[DRY-RUN] Would create migration script: %s\n", scriptPath)
		return nil
	}

	return os.WriteFile(scriptPath, []byte(migrationScript), 0755)
}

func (m *Migrator) updateMainConfig(analysis *AnalysisResult) error {
	configPath := filepath.Join(m.rootDir, ".dotgo", "config.yaml")

	// Create default config if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := config.Config{
			Version: "1.0",
			Repository: config.RepositoryConfig{
				Type: "local",
			},
			Profiles: map[string]config.ProfileConfig{
				"default": {
					Name:        "default",
					Description: "Default profile (migrated)",
					Packages:    m.getPackageNames(analysis.SuggestedPackages),
					Variables:   make(map[string]any),
				},
			},
			Settings: config.SettingsConfig{
				DefaultProfile: "default",
				BackupDir:      ".dotgo/backups",
				SymlinkMode:    "auto",
				ConflictMode:   "ask",
				PackagesDir:    "packages",
				ProfilesDir:    "profiles",
				TemplatesDir:   "templates",
			},
		}

		if m.dryRun {
			fmt.Printf("[DRY-RUN] Would create config: %s\n", configPath)
			return nil
		}

		data, err := yaml.Marshal(defaultConfig)
		if err != nil {
			return err
		}

		return os.WriteFile(configPath, data, 0644)
	}

	return nil
}

func (m *Migrator) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	buf := make([]byte, 64*1024)
	for {
		n, err := sourceFile.Read(buf)
		if n > 0 {
			if _, writeErr := destFile.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
	}

	// Copy permissions
	srcInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}
	return destFile.Chmod(srcInfo.Mode())
}

func (m *Migrator) writePackageConfig(path string, config config.PackageConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m *Migrator) getPackageNames(packages []SuggestedPackage) []string {
	var names []string
	for _, pkg := range packages {
		names = append(names, pkg.Name)
	}
	return names
}

// isDotfile checks if a filename represents a dotfile
func isDotfile(name string) bool {
	if strings.HasPrefix(name, ".") && name != "." && name != ".." {
		return true
	}

	// Common dotfile names that don't start with .
	dotfileNames := []string{
		"gitconfig", "vimrc", "tmux.conf", "bashrc", "zshrc",
	}

	for _, dotfileName := range dotfileNames {
		if name == dotfileName {
			return true
		}
	}

	return false
}

// isConfigDirectory checks if a directory name represents a config directory
func isConfigDirectory(name string) bool {
	configDirs := []string{
		// XDG and standard config directories
		".config", ".local", ".cache", ".state",

		// Editor configurations
		".vim", ".nvim", ".vscode", ".cursor", ".continue", ".claude",

		// Development tools
		".ssh", ".aws", ".docker", ".k8s", ".terraform",

		// Shell and terminal
		".zsh", ".bash", ".fish", ".tmux",

		// Version control
		".git", ".gitconfig",

		// Language-specific
		".npm", ".node", ".python", ".go", ".rust", ".cargo",
		".rbenv", ".rvm", ".pyenv", ".nvm",

		// Other common config directories
		"config", "vim", "vscode", "ssh", "aws", "docker",
	}

	for _, configDir := range configDirs {
		if name == configDir {
			return true
		}
	}

	// Check for modern AI tool directories
	modernAITools := []string{
		".claude", ".cursor", ".continue", ".codeium", ".copilot",
		".aider", ".tabby", ".cody", ".sourcegraph",
	}

	for _, aiTool := range modernAITools {
		if name == aiTool {
			return true
		}
	}

	return false
}
