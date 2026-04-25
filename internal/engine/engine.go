package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"

	"github.com/satocchi0416sh/dotgo/internal/config"
	"github.com/satocchi0416sh/dotgo/internal/errors"
	"github.com/satocchi0416sh/dotgo/pkg/utils"
)

// Engine handles the core dotfile operations
type Engine struct {
	configMgr *config.Manager
	backupDir string
	dryRun    bool
	verbose   bool
}

// NewEngine creates a new engine instance
func NewEngine(rootDir string, dryRun, verbose bool) *Engine {
	configMgr := config.NewManager(rootDir)
	backupDir := filepath.Join(rootDir, ".dotgo/backups")

	return &Engine{
		configMgr: configMgr,
		backupDir: backupDir,
		dryRun:    dryRun,
		verbose:   verbose,
	}
}

// LinkStatus represents the status of a link
type LinkStatus struct {
	TargetPath   string // Home-relative path like .zshrc
	SourcePath   string // Full path to source file in dotfiles
	Exists       bool   // Target exists
	IsSymlink    bool   // Target is a symlink
	IsCorrect    bool   // Points to correct source
	LinkTarget   string // What the symlink points to
	BackupExists bool   // Backup file exists
	ShouldApply  bool   // Should be applied with current tags
}

// Add copies a file to the dotfiles directory and adds it to the manifest
func (e *Engine) Add(filePath string, tags []string) error {
	// Load manifest
	if err := e.configMgr.Load(); err != nil {
		return fmt.Errorf(errors.ErrLoadManifest, err)
	}

	// Expand and validate source file
	sourceFile, err := utils.ExpandPath(filePath)
	if err != nil {
		return fmt.Errorf(errors.ErrExpandPath, err)
	}

	exists, err := utils.FileExists(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to check file: %w", err)
	}
	if !exists {
		return fmt.Errorf("file does not exist: %s", sourceFile)
	}

	// Determine home-relative target path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf(errors.ErrHomeDir, err)
	}

	var targetPath string
	if strings.HasPrefix(sourceFile, homeDir) {
		// File is already in home, get relative path
		relPath, err := filepath.Rel(homeDir, sourceFile)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		targetPath = relPath
	} else {
		// File is outside home, use basename
		targetPath = filepath.Base(sourceFile)
	}

	// Determine destination in dotfiles directory
	rootDir := e.configMgr.GetRootDir()
	destPath := filepath.Join(rootDir, "files", targetPath)

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := utils.EnsureDir(destDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Copy file to dotfiles directory
	if e.dryRun {
		fmt.Printf("%s [DRY-RUN] Would copy: %s -> %s\n",
			color.CyanString("ℹ️"), sourceFile, destPath)
	} else {
		if err := utils.CopyFile(sourceFile, destPath); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
		fmt.Printf("%s Copied: %s -> %s\n",
			color.GreenString("✓"), sourceFile, destPath)
	}

	// Add to manifest
	linkSpec := config.LinkSpec{
		Tags:  tags,
		Hooks: make(map[string]string),
	}

	if err := e.configMgr.AddLink(targetPath, linkSpec); err != nil {
		return fmt.Errorf("failed to add link to manifest: %w", err)
	}

	// Save manifest
	if !e.dryRun {
		if err := e.configMgr.Save(); err != nil {
			return fmt.Errorf("failed to save manifest: %w", err)
		}
	}

	fmt.Printf("%s Added to dotgo.yaml: %s\n",
		color.GreenString("✓"), targetPath)

	return nil
}

// Apply creates symlinks for all applicable files
func (e *Engine) Apply(requestedTags []string) error {
	// Load manifest
	if err := e.configMgr.Load(); err != nil {
		return fmt.Errorf(errors.ErrLoadManifest, err)
	}

	manifest := e.configMgr.GetManifest()
	if manifest == nil {
		return fmt.Errorf("no manifest loaded")
	}

	// Get links that should be applied
	links := e.configMgr.ListLinks(requestedTags)
	if len(links) == 0 {
		fmt.Printf("%s No links to apply\n", color.YellowString("⏭️"))
		return nil
	}

	fmt.Printf("Applying %d link(s)...\n", len(links))

	// Apply each link
	for targetPath, linkSpec := range links {
		if err := e.applyLink(targetPath, linkSpec); err != nil {
			fmt.Printf("%s Failed to apply %s: %v\n",
				color.RedString("✗"), targetPath, err)
			continue
		}

		// Run post-apply hooks if any
		if hook, exists := linkSpec.Hooks["post_apply"]; exists {
			if err := e.runHook(hook, targetPath); err != nil {
				fmt.Printf("%s Hook failed for %s: %v\n",
					color.YellowString("⚠️"), targetPath, err)
			}
		}
	}

	return nil
}

// Remove removes a link from the manifest and optionally restores the original file
func (e *Engine) Remove(targetPath string, restore bool) error {
	// Load manifest
	if err := e.configMgr.Load(); err != nil {
		return fmt.Errorf(errors.ErrLoadManifest, err)
	}

	// Check if link exists in manifest
	_, err := e.configMgr.GetLink(targetPath)
	if err != nil {
		return fmt.Errorf("link not found in manifest: %s", targetPath)
	}

	// Get target file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf(errors.ErrHomeDir, err)
	}

	fullTargetPath := filepath.Join(homeDir, targetPath)

	// Remove symlink if it exists
	exists, err := utils.FileExists(fullTargetPath)
	if err != nil {
		return fmt.Errorf("failed to check target file: %w", err)
	}

	if exists {
		isSymlink, err := utils.IsSymlink(fullTargetPath)
		if err != nil {
			return fmt.Errorf("failed to check if symlink: %w", err)
		}

		if isSymlink {
			if e.dryRun {
				fmt.Printf("%s [DRY-RUN] Would remove symlink: %s\n",
					color.CyanString("ℹ️"), fullTargetPath)
			} else {
				if err := os.Remove(fullTargetPath); err != nil {
					return fmt.Errorf("failed to remove symlink: %w", err)
				}
				fmt.Printf("%s Removed symlink: %s\n",
					color.RedString("✗"), fullTargetPath)
			}
		}
	}

	// Restore original file if requested
	if restore {
		if err := e.restoreOriginal(targetPath); err != nil {
			fmt.Printf("%s Failed to restore original: %v\n",
				color.YellowString("⚠️"), err)
		}
	}

	// Remove from manifest
	if err := e.configMgr.RemoveLink(targetPath); err != nil {
		return fmt.Errorf("failed to remove from manifest: %w", err)
	}

	// Save manifest
	if !e.dryRun {
		if err := e.configMgr.Save(); err != nil {
			return fmt.Errorf("failed to save manifest: %w", err)
		}
	}

	fmt.Printf("%s Removed from dotgo.yaml: %s\n",
		color.RedString("✗"), targetPath)

	return nil
}

// Status shows the current status of all links
func (e *Engine) Status(requestedTags []string) ([]LinkStatus, error) {
	// Load manifest
	if err := e.configMgr.Load(); err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	manifest := e.configMgr.GetManifest()
	if manifest == nil || len(manifest.Links) == 0 {
		return []LinkStatus{}, nil
	}

	var results []LinkStatus
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	rootDir := e.configMgr.GetRootDir()

	for targetPath, linkSpec := range manifest.Links {
		status := LinkStatus{
			TargetPath:  targetPath,
			SourcePath:  filepath.Join(rootDir, "files", targetPath),
			ShouldApply: linkSpec.ShouldApply(requestedTags),
		}

		fullTargetPath := filepath.Join(homeDir, targetPath)

		// Check if target exists
		if stat, err := os.Lstat(fullTargetPath); err == nil {
			status.Exists = true
			status.IsSymlink = stat.Mode()&os.ModeSymlink != 0

			if status.IsSymlink {
				if linkTarget, err := os.Readlink(fullTargetPath); err == nil {
					status.LinkTarget = linkTarget
					// Check if it points to the correct source
					if linkTarget == status.SourcePath {
						status.IsCorrect = true
					}
				}
			}
		}

		// Check if backup exists
		status.BackupExists = e.hasBackup(targetPath)

		results = append(results, status)
	}

	return results, nil
}

// applyLink creates a symlink for a specific file
func (e *Engine) applyLink(targetPath string, linkSpec config.LinkSpec) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf(errors.ErrHomeDir, err)
	}

	rootDir := e.configMgr.GetRootDir()
	sourcePath := filepath.Join(rootDir, "files", targetPath)
	fullTargetPath := filepath.Join(homeDir, targetPath)

	// Check if source exists
	exists, err := utils.FileExists(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to check source file: %w", err)
	}
	if !exists {
		return fmt.Errorf("source file does not exist: %s", sourcePath)
	}

	// Check if target already exists and is correct
	if stat, err := os.Lstat(fullTargetPath); err == nil {
		if stat.Mode()&os.ModeSymlink != 0 {
			if linkTarget, err := os.Readlink(fullTargetPath); err == nil && linkTarget == sourcePath {
				if e.verbose {
					fmt.Printf("%s Already linked: %s -> %s\n",
						color.YellowString("⏭️"), targetPath, sourcePath)
				}
				return nil
			}
		}

		// Backup existing file
		if err := e.backupFile(fullTargetPath); err != nil {
			return fmt.Errorf("failed to backup existing file: %w", err)
		}
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(fullTargetPath)
	if err := utils.EnsureDir(targetDir); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Create symlink
	if e.dryRun {
		fmt.Printf("%s [DRY-RUN] Would create link: %s -> %s\n",
			color.CyanString("ℹ️"), targetPath, sourcePath)
	} else {
		if err := os.Symlink(sourcePath, fullTargetPath); err != nil {
			return fmt.Errorf("failed to create symlink: %w", err)
		}
		fmt.Printf("%s Linked: %s -> %s\n",
			color.GreenString("✓"), targetPath, sourcePath)
	}

	return nil
}

// backupFile creates a backup of an existing file
func (e *Engine) backupFile(filePath string) error {
	if err := utils.EnsureDir(e.backupDir); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	fileName := filepath.Base(filePath)
	backupPath := filepath.Join(e.backupDir, fileName+".backup")

	if e.dryRun {
		fmt.Printf("%s [DRY-RUN] Would backup: %s -> %s\n",
			color.CyanString("ℹ️"), filePath, backupPath)
		return nil
	}

	// Copy to backup location
	if err := utils.CopyFile(filePath, backupPath); err != nil {
		return fmt.Errorf("failed to copy to backup: %w", err)
	}

	// Remove original
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove original: %w", err)
	}

	fmt.Printf("%s Backed up: %s -> %s\n",
		color.BlueString("💾"), filePath, backupPath)

	return nil
}

// restoreOriginal restores a backup file
func (e *Engine) restoreOriginal(targetPath string) error {
	fileName := filepath.Base(targetPath)
	backupPath := filepath.Join(e.backupDir, fileName+".backup")

	exists, err := utils.FileExists(backupPath)
	if err != nil {
		return fmt.Errorf("failed to check backup: %w", err)
	}
	if !exists {
		return fmt.Errorf("no backup found: %s", backupPath)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf(errors.ErrHomeDir, err)
	}

	fullTargetPath := filepath.Join(homeDir, targetPath)

	if e.dryRun {
		fmt.Printf("%s [DRY-RUN] Would restore: %s -> %s\n",
			color.CyanString("ℹ️"), backupPath, fullTargetPath)
		return nil
	}

	if err := utils.CopyFile(backupPath, fullTargetPath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	fmt.Printf("%s Restored: %s\n", color.GreenString("✓"), targetPath)

	return nil
}

// hasBackup checks if a backup exists for a file
func (e *Engine) hasBackup(targetPath string) bool {
	fileName := filepath.Base(targetPath)
	backupPath := filepath.Join(e.backupDir, fileName+".backup")

	exists, _ := utils.FileExists(backupPath)
	return exists
}

// runHook executes a post-apply hook
func (e *Engine) runHook(hook string, targetPath string) error {
	if e.dryRun {
		fmt.Printf("%s [DRY-RUN] Would run hook for %s: %s\n",
			color.CyanString("ℹ️"), targetPath, hook)
		return nil
	}

	if e.verbose {
		fmt.Printf("%s Running hook for %s: %s\n",
			color.BlueString("🔧"), targetPath, hook)
	}

	// For now, hooks are just informational
	// In a full implementation, you might execute shell commands
	return nil
}

// Initialize creates a new dotgo.yaml file
func (e *Engine) Initialize() error {
	return e.configMgr.Initialize()
}

// GetConfigManager returns the config manager
func (e *Engine) GetConfigManager() *config.Manager {
	return e.configMgr
}
