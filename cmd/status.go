package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"dotgo/internal/cmdutil"
	"dotgo/internal/engine"
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

	if len(statuses) == 0 {
		fmt.Printf("%s No files are managed by dotgo\n", color.YellowString("ℹ️"))
		fmt.Println("Use 'dotgo add <file>' to start managing files")
		return nil
	}

	// Print header
	fmt.Printf("%s Dotfiles Status\n", color.BlueString("📊"))
	fmt.Printf("Repository: %s\n", eng.GetRootDir())
	if len(tags) > 0 {
		fmt.Printf("Filtering by tags: %s\n", strings.Join(tags, ", "))
	}
	fmt.Println()

	// Count statistics
	var total, linked, broken, shouldApply int
	for _, status := range statuses {
		total++
		if status.ShouldApply {
			shouldApply++
		}
		if status.Exists {
			if status.IsSymlink && status.IsCorrect {
				linked++
			} else if status.IsSymlink && !status.IsCorrect {
				broken++
			}
		}
	}

	// Print summary
	fmt.Printf("%s Summary:\n", color.CyanString("📋"))
	fmt.Printf("  Total files: %d\n", total)
	fmt.Printf("  Should apply: %s\n", getCountColor(shouldApply, total, shouldApply))
	fmt.Printf("  Correctly linked: %s\n", getCountColor(linked, shouldApply, linked))
	if broken > 0 {
		fmt.Printf("  Broken links: %s\n", color.RedString("%d", broken))
	}
	fmt.Println()

	// Print detailed status
	fmt.Printf("%s File Status:\n", color.CyanString("📁"))
	for _, status := range statuses {
		printFileStatus(status)
	}

	// Print recommendations
	if shouldApply > linked || broken > 0 {
		fmt.Printf("\n%s Recommendations:\n", color.BlueString("💡"))
		if shouldApply > linked {
			fmt.Printf("  • Run '%s' to create missing symlinks\n",
				color.GreenString("dotgo apply"))
		}
		if broken > 0 {
			fmt.Printf("  • Check and fix broken symlinks manually\n")
		}
	}

	return nil
}

// printFileStatus prints the status of a single file
func printFileStatus(status engine.LinkStatus) {
	icon := "❓"
	statusText := ""
	colorFunc := color.WhiteString

	if !status.ShouldApply {
		icon = "⏭️"
		statusText = "skipped (tags don't match)"
		colorFunc = color.YellowString
	} else if !status.Exists {
		icon = "❌"
		statusText = "missing symlink"
		colorFunc = color.RedString
	} else if !status.IsSymlink {
		icon = "⚠️"
		statusText = "exists but not a symlink"
		colorFunc = color.YellowString
	} else if !status.IsCorrect {
		icon = "💔"
		statusText = fmt.Sprintf("broken (points to %s)", status.LinkTarget)
		colorFunc = color.RedString
	} else {
		icon = "✅"
		statusText = "correctly linked"
		colorFunc = color.GreenString
	}

	fmt.Printf("  %s %s - %s\n",
		icon,
		colorFunc(status.TargetPath),
		statusText)

	if verbose && status.Exists {
		fmt.Printf("      Source: %s\n", status.SourcePath)
		if status.IsSymlink {
			fmt.Printf("      Target: %s\n", status.LinkTarget)
		}
		if status.BackupExists {
			fmt.Printf("      Backup: available\n")
		}
	}
}

// getCountColor returns a colored count string based on current vs total
func getCountColor(current, total int, value int) string {
	if current == total && total > 0 {
		return color.GreenString("%d", value)
	} else if current > 0 {
		return color.YellowString("%d", value)
	} else {
		return color.RedString("%d", value)
	}
}
