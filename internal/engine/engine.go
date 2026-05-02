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

// Outcomes for ApplyResult. Strings are stable and may appear in user-facing
// output / structured logs; do not rename without auditing call sites.
const (
	OutcomeApplied = "applied"
	OutcomeSkipped = "skipped"
	OutcomeFailed  = "failed"
	OutcomeDryRun  = "dry-run"
)

// ApplyResult is the structured outcome of applying a single link. Apply
// returns one per link (plus an extra Failed entry if a post-apply hook
// errors) so orchestrators can render output and tally counts without
// parsing free-form text.
type ApplyResult struct {
	TargetPath string
	Outcome    string
	Detail     string
	Err        error
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

	// Use PathExists so that directories are accepted; FileExists would skip
	// them because os.Stat reports IsDir for directory targets.
	exists, err := utils.PathExists(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to check source path: %w", err)
	}
	if !exists {
		return fmt.Errorf("path does not exist: %s", sourceFile)
	}

	isDir, err := utils.DirExists(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to inspect source path: %w", err)
	}

	// Determine home-relative target path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf(errors.ErrHomeDir, err)
	}

	var targetPath string
	if strings.HasPrefix(sourceFile, homeDir) {
		// Source is already in home, get relative path
		relPath, err := filepath.Rel(homeDir, sourceFile)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		targetPath = relPath
	} else {
		// Source is outside home, use basename
		targetPath = filepath.Base(sourceFile)
	}

	// Determine destination in dotfiles directory
	rootDir := e.configMgr.GetRootDir()
	destPath := filepath.Join(rootDir, "files", targetPath)

	// Ensure destination's parent directory exists. For files this is the parent
	// of destPath; for directories the destPath itself acts as the parent and
	// CopyDir will create it.
	destDir := filepath.Dir(destPath)
	if err := utils.EnsureDir(destDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Copy source to dotfiles directory (file or recursive directory copy).
	if e.dryRun {
		fmt.Printf("%s [DRY-RUN] Would copy: %s -> %s\n",
			color.CyanString("ℹ️"), sourceFile, destPath)
	} else {
		if isDir {
			if err := utils.CopyDir(sourceFile, destPath); err != nil {
				return fmt.Errorf("failed to copy directory: %w", err)
			}
		} else {
			if err := utils.CopyFile(sourceFile, destPath); err != nil {
				return fmt.Errorf("failed to copy file: %w", err)
			}
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

// Apply creates symlinks for all applicable files and returns a result per
// link (plus an additional failed entry per failing post-apply hook).
func (e *Engine) Apply(requestedTags []string) ([]ApplyResult, error) {
	// Load manifest
	if err := e.configMgr.Load(); err != nil {
		return nil, fmt.Errorf(errors.ErrLoadManifest, err)
	}

	manifest := e.configMgr.GetManifest()
	if manifest == nil {
		return nil, fmt.Errorf("no manifest loaded")
	}

	links := e.configMgr.ListLinks(requestedTags)
	results := make([]ApplyResult, 0, len(links))

	for targetPath, linkSpec := range links {
		res := e.applyLink(targetPath)
		results = append(results, res)

		// Skip hooks for failed links so we don't compound failures.
		if res.Outcome == OutcomeFailed {
			continue
		}

		if hook, exists := linkSpec.Hooks["post_apply"]; exists {
			if err := e.runHook(hook, targetPath); err != nil {
				results = append(results, ApplyResult{
					TargetPath: targetPath + " (hook)",
					Outcome:    OutcomeFailed,
					Err:        err,
					Detail:     err.Error(),
				})
			}
		}
	}

	return results, nil
}

// ApplyOne applies a single link and returns its result. Hooks are NOT run
// here; orchestrators (e.g. cmd/apply.go) drive per-link calls and may
// invoke hooks separately.
func (e *Engine) ApplyOne(targetPath string) ApplyResult {
	return e.applyLink(targetPath)
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

	// Use PathExists so that directory-targeted symlinks are detected too;
	// FileExists would skip them because os.Stat dereferences and reports IsDir.
	exists, err := utils.PathExists(fullTargetPath)
	if err != nil {
		return fmt.Errorf("failed to check target path: %w", err)
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

// applyLink creates a symlink for a specific file and returns a structured
// result (no I/O side effects beyond symlink/backup creation).
func (e *Engine) applyLink(targetPath string) ApplyResult {
	failed := func(err error) ApplyResult {
		return ApplyResult{
			TargetPath: targetPath,
			Outcome:    OutcomeFailed,
			Err:        err,
			Detail:     err.Error(),
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return failed(fmt.Errorf(errors.ErrHomeDir, err))
	}

	rootDir := e.configMgr.GetRootDir()
	sourcePath := filepath.Join(rootDir, "files", targetPath)
	fullTargetPath := filepath.Join(homeDir, targetPath)

	// Source may be either a regular file or a directory; os.Symlink supports both,
	// so we accept any existing entry (PathExists rather than FileExists).
	exists, err := utils.PathExists(sourcePath)
	if err != nil {
		return failed(fmt.Errorf("failed to check source path: %w", err))
	}
	if !exists {
		return failed(fmt.Errorf("source path does not exist: %s", sourcePath))
	}

	// Check if target already exists and is correct
	if stat, err := os.Lstat(fullTargetPath); err == nil {
		if stat.Mode()&os.ModeSymlink != 0 {
			if linkTarget, err := os.Readlink(fullTargetPath); err == nil && linkTarget == sourcePath {
				return ApplyResult{
					TargetPath: targetPath,
					Outcome:    OutcomeSkipped,
					Detail:     "already linked",
				}
			}
		}

		// Backup existing file
		if err := e.backupFile(fullTargetPath); err != nil {
			return failed(fmt.Errorf("failed to backup existing file: %w", err))
		}
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(fullTargetPath)
	if err := utils.EnsureDir(targetDir); err != nil {
		return failed(fmt.Errorf("failed to create target directory: %w", err))
	}

	if e.dryRun {
		return ApplyResult{
			TargetPath: targetPath,
			Outcome:    OutcomeDryRun,
			Detail:     "→ " + sourcePath,
		}
	}

	if err := os.Symlink(sourcePath, fullTargetPath); err != nil {
		return failed(fmt.Errorf("failed to create symlink: %w", err))
	}
	return ApplyResult{
		TargetPath: targetPath,
		Outcome:    OutcomeApplied,
		Detail:     "→ " + sourcePath,
	}
}

// backupFile creates a backup of an existing file
func (e *Engine) backupFile(filePath string) error {
	if err := utils.EnsureDir(e.backupDir); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	fileName := filepath.Base(filePath)
	backupPath := filepath.Join(e.backupDir, fileName+".backup")

	if e.dryRun {
		return nil
	}

	// Move existing entry into the backup location. Rename works for regular
	// files, directories, and symlinks alike, and is atomic on the same
	// filesystem — which is what we want when the source is a managed
	// directory (e.g. ~/.claude/skills/<name>).
	if err := os.Rename(filePath, backupPath); err != nil {
		return fmt.Errorf("failed to move to backup: %w", err)
	}

	return nil
}

// restoreOriginal restores a backup file
func (e *Engine) restoreOriginal(targetPath string) error {
	fileName := filepath.Base(targetPath)
	backupPath := filepath.Join(e.backupDir, fileName+".backup")

	exists, err := utils.PathExists(backupPath)
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
		return nil
	}

	if err := os.Rename(backupPath, fullTargetPath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
}

// hasBackup checks if a backup exists for a file
func (e *Engine) hasBackup(targetPath string) bool {
	fileName := filepath.Base(targetPath)
	backupPath := filepath.Join(e.backupDir, fileName+".backup")

	exists, _ := utils.PathExists(backupPath)
	return exists
}

// runHook executes a post-apply hook
func (e *Engine) runHook(hook string, targetPath string) error {
	if e.dryRun {
		return nil
	}

	// For now, hooks are just informational. Avoid emitting log lines from
	// the engine; orchestrators surface hook results in their own UI.
	_ = hook
	_ = targetPath
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
