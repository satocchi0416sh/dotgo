package symlink

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Manager handles symlink creation and management
type Manager struct {
	backupDir string
	dryRun    bool
	verbose   bool
}

// NewManager creates a new symlink manager
func NewManager(backupDir string, dryRun, verbose bool) *Manager {
	return &Manager{
		backupDir: backupDir,
		dryRun:    dryRun,
		verbose:   verbose,
	}
}

// LinkInfo represents information about a symlink operation
type LinkInfo struct {
	Source     string
	Target     string
	Executable bool
	Exists     bool
	IsSymlink  bool
	IsValid    bool
	LinkTarget string
	BackupPath string
}

// CreateLink creates a symlink from source to target
func (m *Manager) CreateLink(source, target string, executable bool) error {
	// Expand paths
	source = expandPath(source)
	target = expandPath(target)

	// Validate source exists
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", source)
	}

	// Get link info
	info, err := m.GetLinkInfo(source, target)
	if err != nil {
		return fmt.Errorf("failed to get link info: %w", err)
	}

	// Create target directory if needed
	targetDir := filepath.Dir(target)
	if err := m.ensureDir(targetDir); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Handle existing file/link
	if info.Exists {
		if info.IsSymlink && info.IsValid && info.LinkTarget == source {
			// Link already exists and points to correct source
			if m.verbose {
				fmt.Printf("%s Link already exists: %s -> %s\n",
					color.YellowString("⏭️"), target, source)
			}
			return nil
		}

		// Backup existing file/link
		if err := m.backupFile(target, info); err != nil {
			return fmt.Errorf("failed to backup existing file: %w", err)
		}
	}

	// Create the symlink
	if err := m.createSymlink(source, target); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Set executable if needed
	if executable {
		if err := m.setExecutable(target); err != nil {
			return fmt.Errorf("failed to set executable: %w", err)
		}
	}

	fmt.Printf("%s Linked: %s -> %s\n",
		color.GreenString("✓"), target, source)

	return nil
}

// RemoveLink removes a symlink and optionally restores backup
func (m *Manager) RemoveLink(target string, restoreBackup bool) error {
	target = expandPath(target)

	info, err := m.GetLinkInfo("", target)
	if err != nil {
		return fmt.Errorf("failed to get link info: %w", err)
	}

	if !info.Exists {
		if m.verbose {
			fmt.Printf("%s Link does not exist: %s\n",
				color.YellowString("⏭️"), target)
		}
		return nil
	}

	// Remove the link
	if !m.dryRun {
		if err := os.Remove(target); err != nil {
			return fmt.Errorf("failed to remove link: %w", err)
		}
	}

	fmt.Printf("%s Removed link: %s\n",
		color.RedString("✗"), target)

	// Restore backup if requested and available
	if restoreBackup && info.BackupPath != "" {
		if err := m.restoreBackup(info.BackupPath, target); err != nil {
			return fmt.Errorf("failed to restore backup: %w", err)
		}
	}

	return nil
}

// GetLinkInfo returns information about a link
func (m *Manager) GetLinkInfo(source, target string) (*LinkInfo, error) {
	target = expandPath(target)
	if source != "" {
		source = expandPath(source)
	}

	info := &LinkInfo{
		Source: source,
		Target: target,
	}

	// Check if target exists
	stat, err := os.Lstat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return info, nil
		}
		return nil, err
	}

	info.Exists = true
	info.IsSymlink = stat.Mode()&os.ModeSymlink != 0

	// If it's a symlink, get the link target
	if info.IsSymlink {
		linkTarget, err := os.Readlink(target)
		if err != nil {
			return nil, fmt.Errorf("failed to read symlink: %w", err)
		}

		info.LinkTarget = linkTarget

		// Check if link is valid
		if _, err := os.Stat(target); err == nil {
			info.IsValid = true
		}
	}

	// Find backup path
	info.BackupPath = m.findBackupPath(target)

	return info, nil
}

// ValidateLinks checks the status of multiple links
func (m *Manager) ValidateLinks(links []LinkInfo) ([]LinkInfo, error) {
	var results []LinkInfo

	for _, link := range links {
		info, err := m.GetLinkInfo(link.Source, link.Target)
		if err != nil {
			return nil, fmt.Errorf("failed to validate link %s: %w", link.Target, err)
		}
		results = append(results, *info)
	}

	return results, nil
}

// ListBrokenLinks finds all broken symlinks in a directory
func (m *Manager) ListBrokenLinks(dir string) ([]string, error) {
	var brokenLinks []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.Mode()&os.ModeSymlink != 0 {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				brokenLinks = append(brokenLinks, path)
			}
		}

		return nil
	})

	return brokenLinks, err
}

// ensureDir creates a directory if it doesn't exist
func (m *Manager) ensureDir(dir string) error {
	if m.dryRun {
		if m.verbose {
			fmt.Printf("%s [DRY-RUN] Would create directory: %s\n",
				color.CyanString("ℹ️"), dir)
		}
		return nil
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		if m.verbose {
			fmt.Printf("%s Created directory: %s\n",
				color.GreenString("✓"), dir)
		}
	}

	return nil
}

// backupFile creates a backup of an existing file
func (m *Manager) backupFile(path string, info *LinkInfo) error {
	if m.backupDir == "" {
		return fmt.Errorf("backup directory not configured")
	}

	// Create backup directory
	if err := m.ensureDir(m.backupDir); err != nil {
		return err
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	fileName := filepath.Base(path)
	backupName := fmt.Sprintf("%s_%s", fileName, timestamp)
	backupPath := filepath.Join(m.backupDir, backupName)

	if m.dryRun {
		if m.verbose {
			fmt.Printf("%s [DRY-RUN] Would backup: %s -> %s\n",
				color.CyanString("ℹ️"), path, backupPath)
		}
		return nil
	}

	// Copy or move the file
	if err := copyFile(path, backupPath); err != nil {
		return err
	}

	// Remove original
	if err := os.Remove(path); err != nil {
		return err
	}

	fmt.Printf("%s Backed up: %s -> %s\n",
		color.BlueString("💾"), path, backupPath)

	return nil
}

// createSymlink creates the actual symlink
func (m *Manager) createSymlink(source, target string) error {
	if m.dryRun {
		fmt.Printf("%s [DRY-RUN] Would create link: %s -> %s\n",
			color.CyanString("ℹ️"), target, source)
		return nil
	}

	return os.Symlink(source, target)
}

// setExecutable sets executable permissions on a file
func (m *Manager) setExecutable(path string) error {
	if m.dryRun {
		if m.verbose {
			fmt.Printf("%s [DRY-RUN] Would set executable: %s\n",
				color.CyanString("ℹ️"), path)
		}
		return nil
	}

	return os.Chmod(path, 0755)
}

// findBackupPath finds the most recent backup for a file
func (m *Manager) findBackupPath(originalPath string) string {
	if m.backupDir == "" {
		return ""
	}

	fileName := filepath.Base(originalPath)
	prefix := fileName + "_"

	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		return ""
	}

	var latestBackup string
	var latestTime time.Time

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestBackup = filepath.Join(m.backupDir, name)
			}
		}
	}

	return latestBackup
}

// restoreBackup restores a backup file
func (m *Manager) restoreBackup(backupPath, targetPath string) error {
	if m.dryRun {
		fmt.Printf("%s [DRY-RUN] Would restore backup: %s -> %s\n",
			color.CyanString("ℹ️"), backupPath, targetPath)
		return nil
	}

	if err := copyFile(backupPath, targetPath); err != nil {
		return err
	}

	fmt.Printf("%s Restored backup: %s -> %s\n",
		color.GreenString("✓"), backupPath, targetPath)

	return nil
}

// expandPath expands ~ and environment variables in paths
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(homeDir, path[2:])
		}
	}
	return os.ExpandEnv(path)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	buf := make([]byte, 64*1024) // 64KB buffer
	for {
		n, err := srcFile.Read(buf)
		if n > 0 {
			if _, writeErr := dstFile.Write(buf[:n]); writeErr != nil {
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
	return dstFile.Chmod(srcInfo.Mode())
}
