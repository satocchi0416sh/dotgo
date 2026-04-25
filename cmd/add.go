package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/satocchi0416sh/dotgo/internal/cmdutil"
	"github.com/satocchi0416sh/dotgo/internal/ui"
)

var (
	// addTags are the tags to apply to the added file
	addTags []string
	// addInteractive enables interactive mode
	addInteractive bool
)

func init() {
	addCmd.Flags().StringSliceVarP(&addTags, "tags", "t", []string{}, "Tags for the file (e.g., linux,work)")
	addCmd.Flags().BoolVarP(&addInteractive, "interactive", "i", false, "Interactive mode for tag selection")
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

	// Initialize UI
	uiRenderer := ui.New()

	// Initialize engine
	eng, err := cmdutil.InitializeEngine(dryRun, verbose)
	if err != nil {
		return err
	}

	// Clean up tags
	tags := cmdutil.ProcessTags(addTags)

	// Interactive mode for tag selection
	if addInteractive || (len(tags) == 0 && os.Getenv("DOTGO_NO_INTERACTIVE") != "true") {
		tags, err = interactiveTagSelection(uiRenderer)
		if err != nil {
			return err
		}
	}

	// Show what we're about to do
	fmt.Println(uiRenderer.Header("Adding file to dotgo"))
	fmt.Println()
	fmt.Println(uiRenderer.StatusMessage("info", fmt.Sprintf("File: %s", sourcePath)))
	if len(tags) > 0 {
		fmt.Println(uiRenderer.StatusMessage("info", fmt.Sprintf("Tags: %s", strings.Join(tags, ", "))))
	} else {
		fmt.Println(uiRenderer.StatusMessage("warning", "No tags specified"))
	}
	fmt.Println()

	if dryRun {
		fmt.Println(uiRenderer.StatusMessage("info", "[DRY RUN] No changes will be made"))
		fmt.Println()
	}

	// Add the file
	err = eng.Add(sourcePath, tags)
	if err != nil {
		fmt.Println(uiRenderer.StatusMessage("error", fmt.Sprintf("Failed to add file: %v", err)))
		return err
	}

	fmt.Println(uiRenderer.StatusMessage("success", "File added successfully!"))
	fmt.Println()
	fmt.Println(uiRenderer.Hint("Run 'dotgo apply' to create the symlink"))

	return nil
}

// interactiveTagSelection presents an interactive form for tag selection
func interactiveTagSelection(_ *ui.UI) ([]string, error) {
	// Common tag presets
	presetTags := []string{
		"common",
		"darwin",
		"linux",
		"windows",
		"work",
		"personal",
		"laptop",
		"desktop",
		"server",
	}

	var selectedPresets []string
	var customTags string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select tags").
				Description("Choose from common tags or add custom ones below").
				Options(
					huh.NewOptions(presetTags...)...,
				).
				Value(&selectedPresets),

			huh.NewInput().
				Title("Custom tags").
				Description("Enter additional tags separated by commas").
				Placeholder("e.g., vim,neovim,editor").
				Value(&customTags),
		),
	).WithTheme(ui.MinimalFormTheme())

	err := form.Run()
	if err != nil {
		return nil, err
	}

	// Combine selected presets and custom tags
	var allTags []string
	allTags = append(allTags, selectedPresets...)

	if customTags != "" {
		for _, tag := range strings.Split(customTags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				allTags = append(allTags, tag)
			}
		}
	}

	return allTags, nil
}
