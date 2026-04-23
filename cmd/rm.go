package cmd

import (
	"github.com/spf13/cobra"

	"github.com/satocchi0416sh/dotgo/internal/cmdutil"
)

var (
	// rmRestore indicates whether to restore the original file from backup
	rmRestore bool
)

func init() {
	rmCmd.Flags().BoolVar(&rmRestore, "restore", false, "Restore original file from backup")
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

	// Initialize engine
	eng, err := cmdutil.InitializeEngine(dryRun, verbose)
	if err != nil {
		return err
	}

	// Remove the file
	return eng.Remove(targetPath, rmRestore)
}
