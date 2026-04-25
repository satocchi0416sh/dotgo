package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"dotgo/internal/cmdutil"
	"dotgo/internal/engine"
	"dotgo/internal/ui"
)

var (
	// statusTags are the tags to filter which links to show status for
	statusTags []string
)

func init() {
	statusCmd.Flags().StringSliceVarP(&statusTags, "tags", "t", []string{}, "Tags to filter status (e.g., linux,work)")
	rootCmd.AddCommand(statusCmd)
}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current dotfiles status",
	Long: `Show the current status of your dotfiles installation.

This command displays:
• Which files are managed by dotgo
• Whether symlinks exist and are correct
• Which files would be applied with current tags
• Any broken or missing symlinks

Examples:
  dotgo status                              # Show status of all files
  dotgo status --tags work                  # Show only work-tagged files
  dotgo status -t linux,personal            # Show linux and personal tagged files`,
	RunE: runStatus,
}

// runStatus implements the status command logic
func runStatus(cmd *cobra.Command, args []string) error {
	// Initialize engine
	eng, err := cmdutil.InitializeEngine(false, verbose)
	if err != nil {
		return err
	}

	// Process tags
	tags := cmdutil.ProcessTags(statusTags)

	// Get status
	statuses, err := eng.Status(tags)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Initialize UI
	uiRenderer := ui.New()

	if len(statuses) == 0 {
		fmt.Println(uiRenderer.StatusMessage("info", "No files are managed by dotgo"))
		fmt.Println(uiRenderer.Hint("Use 'dotgo add <file>' to start managing files"))
		return nil
	}

	// Print header
	header := uiRenderer.Header("dotgo", "status")
	fmt.Println(header)

	if len(tags) > 0 {
		fmt.Printf("Filtering by tags: %s\n\n", strings.Join(tags, ", "))
	}

	// Count statistics
	stats := map[string]int{
		"total":    len(statuses),
		"synced":   0,
		"missing":  0,
		"broken":   0,
		"modified": 0,
		"skipped":  0,
	}

	// Group statuses by category
	var synced, missing, broken, modified, skipped []engine.LinkStatus

	for _, status := range statuses {
		if !status.ShouldApply {
			skipped = append(skipped, status)
			stats["skipped"]++
		} else if status.IsCorrect {
			synced = append(synced, status)
			stats["synced"]++
		} else if status.IsSymlink && !status.Exists {
			broken = append(broken, status)
			stats["broken"]++
		} else if status.Exists && !status.IsSymlink {
			modified = append(modified, status)
			stats["modified"]++
		} else if !status.Exists {
			missing = append(missing, status)
			stats["missing"]++
		}
	}

	// Print sections
	if len(synced) > 0 {
		items := make([]string, 0, len(synced))
		for _, status := range synced {
			items = append(items, uiRenderer.FileStatus("link", status.TargetPath, "→ "+status.LinkTarget, nil))
		}
		fmt.Println(uiRenderer.Section("TRACKED", items...))
		fmt.Println()
	}

	if len(modified) > 0 {
		items := make([]string, 0, len(modified))
		for _, status := range modified {
			items = append(items, uiRenderer.FileStatus("modified", status.TargetPath, "exists but not a symlink", nil))
		}
		section := uiRenderer.Section("MODIFIED", items...)
		fmt.Println(section)
		fmt.Println()
	}

	if len(missing) > 0 {
		items := make([]string, 0, len(missing))
		for _, status := range missing {
			items = append(items, uiRenderer.FileStatus("missing", status.TargetPath, "missing symlink", nil))
		}
		section := uiRenderer.Section("MISSING", items...)
		fmt.Println(section)
		fmt.Println()
	}

	if len(broken) > 0 {
		items := make([]string, 0, len(broken))
		for _, status := range broken {
			items = append(items, uiRenderer.FileStatus("broken", status.TargetPath, "→ "+status.LinkTarget, nil))
		}
		section := uiRenderer.Section("BROKEN LINKS", items...)
		fmt.Println(section)
		fmt.Println()
	}

	if len(skipped) > 0 && verbose {
		items := make([]string, 0, len(skipped))
		for _, status := range skipped {
			items = append(items, uiRenderer.FileStatus("skip", status.TargetPath, "tags don't match", nil))
		}
		fmt.Println(uiRenderer.Section("SKIPPED", items...))
		fmt.Println()
	}

	// Simple divider and action hint
	if stats["missing"] > 0 || stats["broken"] > 0 || stats["modified"] > 0 {
		divider := strings.Repeat("─", 48)
		fmt.Println(divider)
		fmt.Println(uiRenderer.StatusMessage("info", "Run `dotgo apply` to sync files."))
	}

	return nil
}

