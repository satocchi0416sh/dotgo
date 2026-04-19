package cmd

import (
	"github.com/spf13/cobra"

	"dotgo/internal/cmdutil"
)

var (
	// applyTags are the tags to filter which links to apply
	applyTags []string
)

func init() {
	applyCmd.Flags().StringSliceVarP(&applyTags, "tags", "t", []string{}, "Tags to apply (e.g., linux,work)")
	rootCmd.AddCommand(applyCmd)
}

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply dotfiles by creating symlinks",
	Long: `Apply dotfiles by creating symlinks from your home directory to the files
in the dotfiles repository.

This command will:
1. Read the dotgo.yaml manifest
2. Create symlinks for all files that match the requested tags
3. Back up any existing files that would be overwritten
4. Run any post-apply hooks defined in the manifest

Examples:
  dotgo apply                               # Apply all files (respecting OS tags)
  dotgo apply --tags work                   # Apply only work-tagged files
  dotgo apply -t linux,personal             # Apply linux and personal tagged files
  dotgo apply --dry-run                     # Preview what would be applied`,
	RunE: runApply,
}

// runApply implements the apply command logic
func runApply(cmd *cobra.Command, args []string) error {
	// Initialize engine
	eng, err := cmdutil.InitializeEngine(dryRun, verbose)
	if err != nil {
		return err
	}

	// Clean up tags
	tags := cmdutil.ProcessTags(applyTags)

	// Apply the links
	return eng.Apply(tags)
}
