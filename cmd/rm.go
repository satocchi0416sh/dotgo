package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/satocchi0416sh/dotgo/internal/cmdutil"
	"github.com/satocchi0416sh/dotgo/internal/ui"
)

var (
	// rmRestore indicates whether to restore the original file from backup
	rmRestore bool
	// rmForce skips confirmation prompt
	rmForce bool
)

func init() {
	rmCmd.Flags().BoolVar(&rmRestore, "restore", false, "Restore original file from backup")
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Skip confirmation prompt")
	rootCmd.AddCommand(rmCmd)
}

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:   "rm <target-path>",
	Short: "Remove a file from dotfiles management",
	Long: `Remove a file from dotfiles management by removing it from dotgo.yaml
and optionally restoring the original file.

This command will:
1. Remove the entry from dotgo.yaml
2. Remove any existing symlink
3. Optionally restore the original file from backup

The target-path should be the home-relative path (e.g., .zshrc, .config/starship.toml).

Examples:
  dotgo rm .zshrc                           # Remove .zshrc from management
  dotgo rm .config/starship.toml            # Remove nested config file
  dotgo rm .vimrc --restore                 # Remove and restore original
  dotgo rm .gitconfig --dry-run             # Preview without executing`,
	Args: cobra.ExactArgs(1),
	RunE: runRm,
}

// runRm implements the rm command logic
func runRm(cmd *cobra.Command, args []string) error {
	targetPath := args[0]

	// Initialize UI
	uiRenderer := ui.New()

	// Initialize engine
	eng, err := cmdutil.InitializeEngine(dryRun, verbose)
	if err != nil {
		return err
	}

	// Show header
	fmt.Println(uiRenderer.Header("Remove from dotgo"))
	fmt.Println()

	// Show what we're about to remove
	fmt.Println(uiRenderer.StatusMessage("info", fmt.Sprintf("File: %s", targetPath)))
	if rmRestore {
		fmt.Println(uiRenderer.StatusMessage("info", "Will restore original file from backup"))
	}
	fmt.Println()

	if dryRun {
		fmt.Println(uiRenderer.StatusMessage("warning", "[DRY RUN] No changes will be made"))
		fmt.Println()
	}

	// Confirmation prompt unless force flag is set
	if !rmForce && !dryRun {
		confirmed, err := confirmRemoval(targetPath, rmRestore)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println(uiRenderer.StatusMessage("info", "Operation cancelled"))
			return nil
		}
	}

	// Remove the file
	err = eng.Remove(targetPath, rmRestore)
	if err != nil {
		if strings.Contains(err.Error(), "not found in manifest") {
			fmt.Println(uiRenderer.StatusMessage("warning", "File is not managed by dotgo"))
			fmt.Println(uiRenderer.Hint("Use 'dotgo status' to see managed files"))
			return nil
		}
		fmt.Println(uiRenderer.StatusMessage("error", fmt.Sprintf("Failed to remove: %v", err)))
		return err
	}

	fmt.Println(uiRenderer.StatusMessage("success", "File removed from dotgo management!"))
	if rmRestore {
		fmt.Println(uiRenderer.StatusMessage("success", "Original file has been restored"))
	}
	fmt.Println()
	fmt.Println(uiRenderer.Hint("Run 'dotgo status' to see current state"))

	return nil
}

// confirmRemoval shows an interactive confirmation prompt
func confirmRemoval(targetPath string, restore bool) (bool, error) {
	var message string
	if restore {
		message = fmt.Sprintf("Remove %s from dotgo and restore original?", targetPath)
	} else {
		message = fmt.Sprintf("Remove %s from dotgo management?", targetPath)
	}

	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(message).
				Description("This action cannot be undone").
				Affirmative("Yes").
				Negative("No").
				Value(&confirmed),
		),
	).WithTheme(ui.MinimalFormTheme())

	err := form.Run()
	return confirmed, err
}
