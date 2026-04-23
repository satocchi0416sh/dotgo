package cmd

import (
	"github.com/spf13/cobra"

	"github.com/satocchi0416sh/dotgo/internal/cmdutil"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new dotgo repository",
	Long: `Initialize a new dotgo repository by creating a dotgo.yaml manifest file.

This command will:
1. Create a new dotgo.yaml file in the current directory
2. Set up the basic structure for managing dotfiles

Examples:
  dotgo init                                # Initialize in current directory`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// runInit implements the init command logic
func runInit(cmd *cobra.Command, args []string) error {
	// Initialize engine
	eng, err := cmdutil.InitializeEngine(dryRun, verbose)
	if err != nil {
		return err
	}

	// Initialize the repository
	return eng.Initialize()
}
