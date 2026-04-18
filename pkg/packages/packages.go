package packages

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"dotgo/pkg/config"
	"dotgo/pkg/symlink"
	"dotgo/pkg/template"
)

// Manager handles package operations
type Manager struct {
	configMgr      *config.Manager
	symlinkMgr     *symlink.Manager
	templateEngine *template.Engine
	rootDir        string
	verbose        bool
	dryRun         bool
}

// NewManager creates a new package manager
func NewManager(configMgr *config.Manager, symlinkMgr *symlink.Manager, rootDir string, verbose, dryRun bool) *Manager {
	// Initialize template engine
	config := configMgr.GetConfig()
	var templateEngine *template.Engine
	if config != nil {
		templateEngine = template.NewEngine(&config.Settings.TemplateEngine, config.Settings.TemplateEngine.GlobalVariables)
	}

	return &Manager{
		configMgr:      configMgr,
		symlinkMgr:     symlinkMgr,
		templateEngine: templateEngine,
		rootDir:        rootDir,
		verbose:        verbose,
		dryRun:         dryRun,
	}
}

// Package represents a dotfiles package
type Package struct {
	Name         string
	Description  string
	Version      string
	Path         string
	Config       *config.PackageConfig
	Installed    bool
	Dependencies []string
	Files        []PackageFile
}

// PackageFile represents a file in a package
type PackageFile struct {
	Source          string
	Target          string
	Executable      bool
	Template        bool
	TemplateVars    map[string]any
	RequiredSecrets []string
	Condition       string
	LinkInfo        *symlink.LinkInfo
}

// InstallResult represents the result of a package installation
type InstallResult struct {
	Package      string
	Success      bool
	Error        error
	FilesLinked  int
	Dependencies []string
}

// List returns all available packages
func (m *Manager) List() ([]*Package, error) {
	packageNames, err := m.configMgr.ListPackages()
	if err != nil {
		return nil, err
	}

	var packages []*Package
	for _, name := range packageNames {
		pkg, err := m.LoadPackage(name)
		if err != nil {
			continue // Skip invalid packages
		}
		packages = append(packages, pkg)
	}

	// Sort by name
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Name < packages[j].Name
	})

	return packages, nil
}

// LoadPackage loads a package by name
func (m *Manager) LoadPackage(name string) (*Package, error) {
	packageConfig, err := m.configMgr.LoadPackageConfig(name)
	if err != nil {
		return nil, err
	}

	packagePath := m.configMgr.GetPackageDir(name)

	pkg := &Package{
		Name:         name,
		Description:  packageConfig.Description,
		Version:      packageConfig.Version,
		Path:         packagePath,
		Config:       packageConfig,
		Dependencies: packageConfig.Dependencies,
	}

	// Load files
	for _, fileMapping := range packageConfig.Files {
		// Process template variables in file paths
		source := filepath.Join(packagePath, fileMapping.Source)
		target := fileMapping.Target

		// Apply template processing if template engine is available
		if m.templateEngine != nil {
			// Merge package variables with global variables
			templateVars := make(map[string]any)
			for k, v := range packageConfig.Variables {
				templateVars[k] = v
			}

			// Process target path templates
			processedTarget, err := m.templateEngine.ProcessTemplate(target, templateVars)
			if err != nil {
				return nil, fmt.Errorf("failed to process template for target %s: %w", target, err)
			}
			target = processedTarget
		}

		file := PackageFile{
			Source:          source,
			Target:          m.configMgr.ExpandPath(target),
			Executable:      fileMapping.Executable,
			Template:        fileMapping.Template,
			TemplateVars:    fileMapping.TemplateVars,
			RequiredSecrets: fileMapping.RequiredSecrets,
			Condition:       fileMapping.Condition,
		}

		// Get link info
		linkInfo, err := m.symlinkMgr.GetLinkInfo(file.Source, file.Target)
		if err != nil {
			return nil, fmt.Errorf("failed to get link info for %s: %w", file.Target, err)
		}
		file.LinkInfo = linkInfo

		pkg.Files = append(pkg.Files, file)
	}

	// Check if package is installed (all files are properly linked)
	pkg.Installed = m.isPackageInstalled(pkg)

	return pkg, nil
}

// Install installs a package and its dependencies
func (m *Manager) Install(packageName string) (*InstallResult, error) {
	result := &InstallResult{
		Package: packageName,
	}

	// Load package
	pkg, err := m.LoadPackage(packageName)
	if err != nil {
		result.Error = err
		return result, err
	}

	// Install dependencies first
	for _, dep := range pkg.Dependencies {
		_, err := m.Install(dep)
		if err != nil {
			result.Error = fmt.Errorf("failed to install dependency %s: %w", dep, err)
			return result, result.Error
		}
		result.Dependencies = append(result.Dependencies, dep)
		if m.verbose {
			fmt.Printf("  Installed dependency: %s\n", dep)
		}
	}

	// Check if already installed
	if pkg.Installed && !m.dryRun {
		if m.verbose {
			fmt.Printf("Package %s is already installed\n", packageName)
		}
		result.Success = true
		return result, nil
	}

	// Run pre-install commands
	if err := m.runCommands(pkg.Config.Commands.PreInstall, pkg.Path); err != nil {
		result.Error = fmt.Errorf("pre-install commands failed: %w", err)
		return result, result.Error
	}

	// Install files
	for _, file := range pkg.Files {
		// Check condition if specified
		if file.Condition != "" && !m.evaluateCondition(file.Condition) {
			if m.verbose {
				fmt.Printf("Skipping %s (condition not met: %s)\n", file.Target, file.Condition)
			}
			continue
		}

		// Handle template files differently
		if file.Template {
			// Process template file
			err := m.processTemplateFile(file, pkg.Config.Variables)
			if err != nil {
				result.Error = fmt.Errorf("failed to process template file %s: %w", file.Source, err)
				return result, result.Error
			}
		} else {
			// Verify source file exists
			if _, err := os.Stat(file.Source); os.IsNotExist(err) {
				result.Error = fmt.Errorf("source file does not exist: %s", file.Source)
				return result, result.Error
			}

			// Create symlink
			if err := m.symlinkMgr.CreateLink(file.Source, file.Target, file.Executable); err != nil {
				result.Error = fmt.Errorf("failed to create link %s -> %s: %w", file.Target, file.Source, err)
				return result, result.Error
			}
		}

		result.FilesLinked++
	}

	// Run post-install commands
	if err := m.runCommands(pkg.Config.Commands.PostInstall, pkg.Path); err != nil {
		result.Error = fmt.Errorf("post-install commands failed: %w", err)
		return result, result.Error
	}

	result.Success = true
	return result, nil
}

// Remove removes a package
func (m *Manager) Remove(packageName string, restoreBackup bool) error {
	// Load package
	pkg, err := m.LoadPackage(packageName)
	if err != nil {
		return err
	}

	// Check if package is installed
	if !pkg.Installed {
		if m.verbose {
			fmt.Printf("Package %s is not installed\n", packageName)
		}
		return nil
	}

	// Run pre-remove commands
	if err := m.runCommands(pkg.Config.Commands.PreRemove, pkg.Path); err != nil {
		return fmt.Errorf("pre-remove commands failed: %w", err)
	}

	// Remove symlinks
	for _, file := range pkg.Files {
		if file.LinkInfo != nil && file.LinkInfo.Exists && file.LinkInfo.IsSymlink {
			if err := m.symlinkMgr.RemoveLink(file.Target, restoreBackup); err != nil {
				return fmt.Errorf("failed to remove link %s: %w", file.Target, err)
			}
		}
	}

	// Run post-remove commands
	if err := m.runCommands(pkg.Config.Commands.PostRemove, pkg.Path); err != nil {
		return fmt.Errorf("post-remove commands failed: %w", err)
	}

	return nil
}

// Update updates a package (remove and install)
func (m *Manager) Update(packageName string) (*InstallResult, error) {
	// Remove first
	if err := m.Remove(packageName, false); err != nil {
		return nil, fmt.Errorf("failed to remove package: %w", err)
	}

	// Install again
	return m.Install(packageName)
}

// GetStatus returns the status of all packages
func (m *Manager) GetStatus() ([]*Package, error) {
	return m.List()
}

// GetInstalledPackages returns only installed packages
func (m *Manager) GetInstalledPackages() ([]*Package, error) {
	packages, err := m.List()
	if err != nil {
		return nil, err
	}

	var installed []*Package
	for _, pkg := range packages {
		if pkg.Installed {
			installed = append(installed, pkg)
		}
	}

	return installed, nil
}

// ValidateInstallation validates the installation of all packages
func (m *Manager) ValidateInstallation() error {
	packages, err := m.List()
	if err != nil {
		return err
	}

	var errors []string
	for _, pkg := range packages {
		if pkg.Installed {
			for _, file := range pkg.Files {
				if file.LinkInfo == nil {
					continue
				}

				if file.LinkInfo.IsSymlink && !file.LinkInfo.IsValid {
					errors = append(errors, fmt.Sprintf("Broken symlink: %s", file.Target))
				} else if !file.LinkInfo.Exists {
					errors = append(errors, fmt.Sprintf("Missing link: %s", file.Target))
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// ResolveDependencies resolves package dependencies and returns install order
func (m *Manager) ResolveDependencies(packages []string) ([]string, error) {
	var resolved []string
	var resolving []string

	var resolve func(string) error
	resolve = func(pkg string) error {
		// Check for circular dependency
		for _, resolving := range resolving {
			if resolving == pkg {
				return fmt.Errorf("circular dependency detected: %s", pkg)
			}
		}

		// Check if already resolved
		for _, resolved := range resolved {
			if resolved == pkg {
				return nil
			}
		}

		// Load package config to get dependencies
		packageConfig, err := m.configMgr.LoadPackageConfig(pkg)
		if err != nil {
			return fmt.Errorf("failed to load package %s: %w", pkg, err)
		}

		resolving = append(resolving, pkg)

		// Resolve dependencies first
		for _, dep := range packageConfig.Dependencies {
			if err := resolve(dep); err != nil {
				return err
			}
		}

		// Remove from resolving list
		for i, r := range resolving {
			if r == pkg {
				resolving = append(resolving[:i], resolving[i+1:]...)
				break
			}
		}

		// Add to resolved list
		resolved = append(resolved, pkg)
		return nil
	}

	// Resolve all requested packages
	for _, pkg := range packages {
		if err := resolve(pkg); err != nil {
			return nil, err
		}
	}

	return resolved, nil
}

// isPackageInstalled checks if a package is fully installed
func (m *Manager) isPackageInstalled(pkg *Package) bool {
	for _, file := range pkg.Files {
		if file.LinkInfo == nil {
			return false
		}

		// Skip files with unmet conditions
		if file.Condition != "" && !m.evaluateCondition(file.Condition) {
			continue
		}

		if !file.LinkInfo.Exists || !file.LinkInfo.IsSymlink || !file.LinkInfo.IsValid {
			return false
		}

		// Verify the link points to the correct source
		expectedSource, _ := filepath.Abs(file.Source)
		actualSource, _ := filepath.Abs(file.LinkInfo.LinkTarget)
		if expectedSource != actualSource {
			return false
		}
	}

	return true
}

// runCommands executes a list of commands in the specified directory
func (m *Manager) runCommands(commands []string, _ string) error {
	if len(commands) == 0 {
		return nil
	}

	for _, cmd := range commands {
		if m.verbose {
			fmt.Printf("Executing: %s\n", cmd)
		}

		if m.dryRun {
			fmt.Printf("[DRY-RUN] Would execute: %s\n", cmd)
			continue
		}

		// TODO: Execute command in workDir
		// This would require a proper command execution implementation
	}

	return nil
}

// evaluateCondition evaluates a condition string
// For now, this is a simple implementation that can be extended
func (m *Manager) evaluateCondition(condition string) bool {
	// Simple condition evaluation
	switch condition {
	case "darwin":
		return strings.Contains(strings.ToLower(os.Getenv("GOOS")), "darwin")
	case "linux":
		return strings.Contains(strings.ToLower(os.Getenv("GOOS")), "linux")
	case "windows":
		return strings.Contains(strings.ToLower(os.Getenv("GOOS")), "windows")
	default:
		// Enhanced condition evaluation with template engine
		if m.templateEngine != nil {
			// Try to evaluate as a template expression
			result, err := m.templateEngine.ProcessTemplate(condition, make(map[string]any))
			if err == nil && (result == "true" || result == "1") {
				return true
			}
		}
		// For now, return true for unknown conditions
		return true
	}
}

// processTemplateFile processes a template file and copies it to the target
func (m *Manager) processTemplateFile(file PackageFile, vars map[string]any) error {
	if m.templateEngine == nil {
		return fmt.Errorf("template engine not initialized")
	}

	// Check if source template file exists
	if _, err := os.Stat(file.Source); os.IsNotExist(err) {
		return fmt.Errorf("template source file does not exist: %s", file.Source)
	}

	// Validate required secrets before processing
	if len(file.RequiredSecrets) > 0 {
		missingSecrets := m.templateEngine.ValidateRequiredSecrets(file.RequiredSecrets)
		if len(missingSecrets) > 0 {
			return fmt.Errorf("missing required secrets for %s: %v", file.Target, missingSecrets)
		}
	}

	if m.dryRun {
		fmt.Printf("[DRY-RUN] Would process template: %s -> %s\n", file.Source, file.Target)
		return nil
	}

	// Load package-specific .env files if present
	packageDir := filepath.Dir(file.Source)
	if err := m.templateEngine.LoadEnvWithHierarchy(packageDir); err != nil {
		if m.verbose {
			fmt.Printf("Warning: Failed to load .env files from %s: %v\n", packageDir, err)
		}
	}

	// Merge template variables from file mapping with package variables
	allVars := make(map[string]any)
	for k, v := range vars {
		allVars[k] = v
	}
	for k, v := range file.TemplateVars {
		allVars[k] = v
	}

	// Process the template file
	if err := m.templateEngine.ProcessFileToFile(file.Source, file.Target, allVars); err != nil {
		return fmt.Errorf("failed to process template file: %w", err)
	}

	// Set executable permission if needed
	if file.Executable {
		if err := os.Chmod(file.Target, 0755); err != nil {
			return fmt.Errorf("failed to set executable permission: %w", err)
		}
	}

	if m.verbose {
		fmt.Printf("Processed template: %s -> %s\n", file.Source, file.Target)
	}

	return nil
}
