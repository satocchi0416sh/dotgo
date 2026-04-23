package cmd

import (
	"github.com/spf13/cobra"

	"github.com/satocchi0416sh/dotgo/internal/cmdutil"
)

var (
	// addTags are the tags to apply to the added file
	addTags []string
)

func init() {
	addCmd.Flags().StringSliceVarP(&addTags, "tags", "t", []string{}, "Tags for the file (e.g., linux,work)")
	rootCmd.AddCommand(addCmd)
}

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <file-path>",
	Short: "Add a file to dotfiles management",
	Long: `Add a file to dotfiles management by copying it to the dotfiles directory
and adding it to dotgo.yaml.

This command will:
1. Copy the file to the files/ directory in the dotfiles repository
2. Add an entry to dotgo.yaml with the specified tags
3. The file can then be linked using 'dotgo apply'

Examples:
  dotgo add ~/.zshrc                         # Add without tags
  dotgo add ~/.config/starship.toml -t linux,work  # Add with tags
  dotgo add ~/.vimrc --tags personal         # Add with single tag
  dotgo add ~/.gitconfig --dry-run           # Preview without executing`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

// runAdd implements the add command logic
func runAdd(cmd *cobra.Command, args []string) error {
	sourcePath := args[0]

	// Initialize engine
	eng, err := cmdutil.InitializeEngine(dryRun, verbose)
	if err != nil {
		return err
	}

	// Clean up tags
	tags := cmdutil.ProcessTags(addTags)

	// Add the file
	return eng.Add(sourcePath, tags)
}
