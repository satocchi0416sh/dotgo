package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/satocchi0416sh/dotgo/internal/cmdutil"
	"github.com/satocchi0416sh/dotgo/internal/engine"
	"github.com/satocchi0416sh/dotgo/internal/ui"
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

// applyModel represents the model for the apply command's interactive UI
type applyModel struct {
	spinner  spinner.Model
	statuses []engine.LinkStatus
	current  int
	done     bool
	err      error
	eng      *engine.Engine
	tags     []string
	uiRender *ui.UI
	dryRun   bool
}

func initialApplyModel(eng *engine.Engine, statuses []engine.LinkStatus, tags []string, dryRun bool) applyModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return applyModel{
		spinner:  s,
		statuses: statuses,
		eng:      eng,
		tags:     tags,
		uiRender: ui.New(),
		dryRun:   dryRun,
	}
}

func (m applyModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.processNext())
}

func (m applyModel) processNext() tea.Cmd {
	return func() tea.Msg {
		if m.current >= len(m.statuses) {
			return doneMsg{}
		}

		status := m.statuses[m.current]
		if !status.ShouldApply {
			return nextMsg{}
		}

		// Simulate work with a small delay for visual effect
		time.Sleep(100 * time.Millisecond)

		if !m.dryRun {
			// Actually apply the link
			err := m.eng.Apply(m.tags)
			if err != nil {
				return errMsg{err}
			}
		}

		return nextMsg{}
	}
}

type nextMsg struct{}
type doneMsg struct{}
type errMsg struct{ err error }

func (m applyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case nextMsg:
		m.current++
		return m, m.processNext()

	case doneMsg:
		m.done = true
		return m, tea.Quit

	case errMsg:
		m.err = msg.err
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m applyModel) View() string {
	if m.err != nil {
		return m.uiRender.StatusMessage("error", fmt.Sprintf("Error: %v", m.err))
	}

	if m.done {
		return ""
	}

	if m.current >= len(m.statuses) {
		return ""
	}

	status := m.statuses[m.current]
	progress := m.uiRender.Progress(m.current+1, len(m.statuses), status.TargetPath)
	return fmt.Sprintf("%s %s", m.spinner.View(), progress)
}

// runApply implements the apply command logic
func runApply(cmd *cobra.Command, args []string) error {
	// Initialize UI
	uiRenderer := ui.New()

	// Initialize engine
	eng, err := cmdutil.InitializeEngine(dryRun, verbose)
	if err != nil {
		return err
	}

	// Clean up tags
	tags := cmdutil.ProcessTags(applyTags)

	// Get current status to know what needs to be applied
	statuses, err := eng.Status(tags)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Count files to apply
	toApply := 0
	for _, status := range statuses {
		if status.ShouldApply && (!status.IsCorrect || !status.Exists) {
			toApply++
		}
	}

	// Show header
	fmt.Println(uiRenderer.Header("Applying dotfiles", fmt.Sprintf("Repository: %s", eng.GetRootDir())))
	fmt.Println()

	if len(tags) > 0 {
		fmt.Println(uiRenderer.StatusMessage("info", fmt.Sprintf("Applying tags: %s", cmdutil.FormatTags(tags))))
	}

	if dryRun {
		fmt.Println(uiRenderer.StatusMessage("warning", "[DRY RUN] No changes will be made"))
	}

	fmt.Println(uiRenderer.StatusMessage("info", fmt.Sprintf("Files to process: %d", toApply)))
	fmt.Println()

	if toApply == 0 {
		fmt.Println(uiRenderer.StatusMessage("success", "All files are already up to date!"))
		return nil
	}

	// Run interactive progress if not in verbose mode
	if !verbose {
		p := tea.NewProgram(initialApplyModel(eng, statuses, tags, dryRun))
		if _, err := p.Run(); err != nil {
			return err
		}
	} else {
		// In verbose mode, apply without animation
		err = eng.Apply(tags)
		if err != nil {
			return err
		}
	}

	// Show summary
	fmt.Println()
	fmt.Println(uiRenderer.StatusMessage("success", fmt.Sprintf("Successfully applied %d files!", toApply)))

	// Show next steps
	fmt.Println()
	fmt.Println(uiRenderer.Hint("Run 'dotgo status' to see the current state of your dotfiles"))

	return nil
}
