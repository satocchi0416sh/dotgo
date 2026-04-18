package add

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"dotgo/pkg/config"
	"dotgo/pkg/symlink"
)

// AddOperation represents a file add operation
type AddOperation struct {
	SourcePath  string // Absolute path to source file
	TargetPath  string // Target path where symlink will be created
	PackageName string // Package name
}

// AddResult represents the result of an add operation
type AddResult struct {
	PackageFilePath string // Path where file was moved in packages directory
	ConfigPath      string // Path to package config file that was updated
	TargetPath      string // Target path where symlink was created
	SymlinkCreated  bool   // Whether symlink was successfully created
}

// RollbackOperation tracks operations that can be rolled back
type RollbackOperation struct {
	Type       string // "file_move", "config_update", "symlink_create"
	SourcePath string
	TargetPath string
	BackupPath string
	Timestamp  time.Time
}

// FileManager handles file operations for the add command
type FileManager struct {
	configMgr   *config.Manager
	symlinkMgr  *symlink.Manager
	rootDir     string
	verbose     bool
	dryRun      bool
	rollbackOps []RollbackOperation
}

// NewFileManager creates a new FileManager instance
func NewFileManager(configMgr *config.Manager, rootDir string, verbose, dryRun bool) *FileManager {
	symlinkMgr := symlink.NewManager(
		configMgr.GetBackupDir(),
		dryRun,
		verbose,
	)

	return &FileManager{
		configMgr:   configMgr,
		symlinkMgr:  symlinkMgr,
		rootDir:     rootDir,
		verbose:     verbose,
		dryRun:      dryRun,
		rollbackOps: make([]RollbackOperation, 0),
	}
}

// AddFile executes the complete add file operation
func (fm *FileManager) AddFile(operation *AddOperation) (*AddResult, error) {
	if fm.verbose {
		fmt.Printf("Starting add operation for: %s\n", operation.SourcePath)
	}

	// Validate source file exists
	if err := fm.validateSourceFile(operation.SourcePath); err != nil {
		return nil, fmt.Errorf("source validation failed: %w", err)
	}

	// Determine package directory and file paths
	packageDir := filepath.Join(fm.rootDir, "packages", operation.PackageName)
	filesDir := filepath.Join(packageDir, "files")

	// Get relative path for the target structure
	var relPath string
	var err error

	// Check if the source file is within the dotfiles root directory
	if strings.HasPrefix(operation.SourcePath, fm.rootDir) {
		// File is within dotfiles directory - use relative path from dotfiles root
		relPath, err = filepath.Rel(fm.rootDir, operation.SourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path from dotfiles root: %w", err)
		}
		if fm.verbose {
			fmt.Printf("File is within dotfiles directory, using relative path: %s\n", relPath)
		}
	} else {
		// File is outside dotfiles directory - use relative path from home
		processor := NewPathProcessor()
		relPath, err = processor.GetRelativeToHome(operation.SourcePath)
		if err != nil {
			// If can't get relative path, use just the filename
			relPath = filepath.Base(operation.SourcePath)
		}
		if fm.verbose {
			fmt.Printf("File is outside dotfiles directory, using home-relative path: %s\n", relPath)
		}
	}

	packageFilePath := filepath.Join(filesDir, relPath)
	configFilePath := filepath.Join(packageDir, "dotgo.yaml")

	// Create package directory structure
	if err := fm.createPackageStructure(packageDir, filesDir); err != nil {
		return nil, fmt.Errorf("failed to create package structure: %w", err)
	}

	// Move file to package directory
	if err := fm.moveFileToPackage(operation.SourcePath, packageFilePath); err != nil {
		return nil, fmt.Errorf("failed to move file to package: %w", err)
	}

	// Update or create package configuration
	if err := fm.updatePackageConfig(configFilePath, relPath, operation.TargetPath); err != nil {
		return nil, fmt.Errorf("failed to update package config: %w", err)
	}

	// Create symlink
	symlinkCreated := false
	if !fm.dryRun {
		if err := fm.symlinkMgr.CreateLink(packageFilePath, operation.TargetPath, false); err != nil {
			return nil, fmt.Errorf("failed to create symlink: %w", err)
		}
		symlinkCreated = true
	} else {
		fmt.Printf("[DRY RUN] Would create symlink: %s → %s\n",
			operation.TargetPath, packageFilePath)
	}

	return &AddResult{
		PackageFilePath: packageFilePath,
		ConfigPath:      configFilePath,
		TargetPath:      operation.TargetPath,
		SymlinkCreated:  symlinkCreated,
	}, nil
}

// validateSourceFile checks if source file exists and is readable
func (fm *FileManager) validateSourceFile(sourcePath string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source file does not exist: %s", sourcePath)
		}
		return fmt.Errorf("cannot access source file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("source path is a directory, not a file: %s", sourcePath)
	}

	// Check if file is readable
	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("source file is not readable: %w", err)
	}
	file.Close()

	return nil
}

// createPackageStructure creates the package directory and files subdirectory
func (fm *FileManager) createPackageStructure(packageDir, filesDir string) error {
	if fm.dryRun {
		fmt.Printf("[DRY RUN] Would create directories: %s\n", filesDir)
		return nil
	}

	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return fmt.Errorf("failed to create package directories: %w", err)
	}

	fm.rollbackOps = append(fm.rollbackOps, RollbackOperation{
		Type:       "directory_create",
		TargetPath: packageDir,
		Timestamp:  time.Now(),
	})

	if fm.verbose {
		fmt.Printf("Created package directory: %s\n", packageDir)
	}

	return nil
}

// moveFileToPackage moves the source file to the package directory
func (fm *FileManager) moveFileToPackage(sourcePath, packageFilePath string) error {
	if fm.dryRun {
		fmt.Printf("[DRY RUN] Would move file: %s → %s\n", sourcePath, packageFilePath)
		return nil
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(packageFilePath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Create backup of source file info for rollback
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	// Move the file
	if err := os.Rename(sourcePath, packageFilePath); err != nil {
		// If rename fails (e.g., cross-device), copy and remove
		if err := fm.copyFile(sourcePath, packageFilePath); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
		if err := os.Remove(sourcePath); err != nil {
			return fmt.Errorf("failed to remove original file: %w", err)
		}
	}

	// Preserve file permissions
	if err := os.Chmod(packageFilePath, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("failed to preserve file permissions: %w", err)
	}

	fm.rollbackOps = append(fm.rollbackOps, RollbackOperation{
		Type:       "file_move",
		SourcePath: packageFilePath,
		TargetPath: sourcePath,
		Timestamp:  time.Now(),
	})

	if fm.verbose {
		fmt.Printf("Moved file: %s → %s\n", sourcePath, packageFilePath)
	}

	return nil
}

// copyFile copies a file from source to destination
func (fm *FileManager) copyFile(src, dst string) error {
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

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

// shouldProcessAsTemplate checks if a file should be processed as a template
func (fm *FileManager) shouldProcessAsTemplate(filePath string) bool {
	return strings.HasSuffix(filePath, ".tmpl")
}

// stripTmplExtension removes .tmpl extension from file path
func (fm *FileManager) stripTmplExtension(filePath string) string {
	if strings.HasSuffix(filePath, ".tmpl") {
		return filePath[:len(filePath)-5] // Remove ".tmpl"
	}
	return filePath
}

// updatePackageConfig updates or creates the package configuration file
func (fm *FileManager) updatePackageConfig(configPath, relativeSource, targetPath string) error {
	if fm.dryRun {
		fmt.Printf("[DRY RUN] Would update config: %s\n", configPath)
		return nil
	}

	var packageConfig config.PackageConfig

	// Try to load existing config
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, &packageConfig); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing config: %w", err)
	}

	// Initialize files slice if nil
	if packageConfig.Files == nil {
		packageConfig.Files = make([]config.FileMapping, 0)
	}

	// Check if this is a template file
	isTemplate := fm.shouldProcessAsTemplate(relativeSource)
	
	// For .tmpl files, strip the extension from the target path
	finalTargetPath := targetPath
	if isTemplate {
		finalTargetPath = fm.stripTmplExtension(targetPath)
	}

	// Check if mapping already exists
	var existingIndex = -1
	for i, mapping := range packageConfig.Files {
		if mapping.Target == finalTargetPath {
			existingIndex = i
			break
		}
	}

	// Create new file mapping
	newMapping := config.FileMapping{
		Source:   relativeSource,
		Target:   finalTargetPath,
		Template: isTemplate,
	}

	if existingIndex >= 0 {
		// Update existing mapping
		packageConfig.Files[existingIndex] = newMapping
	} else {
		// Add new mapping
		packageConfig.Files = append(packageConfig.Files, newMapping)
	}

	// Write updated config
	data, err := yaml.Marshal(&packageConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fm.rollbackOps = append(fm.rollbackOps, RollbackOperation{
		Type:       "config_update",
		TargetPath: configPath,
		Timestamp:  time.Now(),
	})

	if fm.verbose {
		fmt.Printf("Updated package config: %s\n", configPath)
	}

	return nil
}

// Rollback attempts to rollback all operations performed during add
func (fm *FileManager) Rollback() error {
	if fm.dryRun {
		fmt.Println("[DRY RUN] Would rollback operations")
		return nil
	}

	var errors []error

	// Rollback in reverse order
	for i := len(fm.rollbackOps) - 1; i >= 0; i-- {
		op := fm.rollbackOps[i]

		if err := fm.rollbackOperation(op); err != nil {
			errors = append(errors, fmt.Errorf("failed to rollback %s: %w", op.Type, err))
		}
	}

	if len(errors) > 0 {
		// Return first error, log others if verbose
		if fm.verbose {
			for _, err := range errors[1:] {
				fmt.Printf("Additional rollback error: %v\n", err)
			}
		}
		return errors[0]
	}

	return nil
}

// rollbackOperation rolls back a single operation
func (fm *FileManager) rollbackOperation(op RollbackOperation) error {
	switch op.Type {
	case "file_move":
		// Move file back to original location
		return os.Rename(op.SourcePath, op.TargetPath)

	case "config_update":
		// For config updates, we would need to store the original config
		// For now, just remove the config file if it was newly created
		if _, err := os.Stat(op.TargetPath); err == nil {
			return os.Remove(op.TargetPath)
		}
		return nil

	case "directory_create":
		// Remove empty directory if possible
		return os.Remove(op.TargetPath)

	default:
		return fmt.Errorf("unknown rollback operation type: %s", op.Type)
	}
}
