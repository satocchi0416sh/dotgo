package ui

import (
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color palette for the UI
type Theme struct {
	Accent lipgloss.Color // Cyan for primary actions
	Subtle lipgloss.Color // Gray for secondary text
	Danger lipgloss.Color // Red only for errors
	Normal lipgloss.Color // Default terminal color
}

// DefaultTheme returns the default color theme
func DefaultTheme() Theme {
	return Theme{
		Accent: lipgloss.Color("38"),  // Cyan
		Subtle: lipgloss.Color("241"), // Gray
		Danger: lipgloss.Color("196"), // Red
		Normal: lipgloss.Color("15"),  // Default
	}
}

// MinimalFormTheme returns a minimal huh theme that matches our design
func MinimalFormTheme() *huh.Theme {
	// Start with Charm theme as base to preserve all functionality
	t := huh.ThemeCharm()

	// Base colors matching our minimal palette
	accent := lipgloss.Color("38")  // Cyan
	subtle := lipgloss.Color("241") // Gray
	normal := lipgloss.Color("15")  // Default

	// Focused style - simple cyan accent
	t.Focused.Base = lipgloss.NewStyle()
	t.Focused.Title = lipgloss.NewStyle().Bold(true).Foreground(normal)
	t.Focused.NoteTitle = lipgloss.NewStyle().Foreground(accent).Bold(true) 
	t.Focused.Directory = lipgloss.NewStyle().Foreground(accent)
	t.Focused.Description = lipgloss.NewStyle().Foreground(subtle)
	t.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	t.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	t.Focused.FocusedButton = lipgloss.NewStyle().Foreground(accent).Bold(true)
	t.Focused.BlurredButton = lipgloss.NewStyle().Foreground(subtle)
	t.Focused.Next = lipgloss.NewStyle().Foreground(accent)
	t.Focused.Option = lipgloss.NewStyle().Foreground(normal)
	
	// Multi-select styling with proper selector
	t.Focused.MultiSelectSelector = lipgloss.NewStyle().Foreground(accent).SetString("› ")
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(accent)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(accent).SetString("✓ ")
	t.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(normal)
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(subtle).SetString("· ")
	
	// Text input styling
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(accent)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(subtle)
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(accent).SetString("> ")

	// Blurred style - muted colors  
	t.Blurred.Base = lipgloss.NewStyle()
	t.Blurred.Title = lipgloss.NewStyle().Foreground(subtle)
	t.Blurred.NoteTitle = lipgloss.NewStyle().Foreground(subtle)
	t.Blurred.Directory = lipgloss.NewStyle().Foreground(subtle)
	t.Blurred.Description = lipgloss.NewStyle().Foreground(subtle)
	t.Blurred.ErrorIndicator = lipgloss.NewStyle().Foreground(subtle)
	t.Blurred.ErrorMessage = lipgloss.NewStyle().Foreground(subtle)
	t.Blurred.FocusedButton = lipgloss.NewStyle().Foreground(subtle)
	t.Blurred.Next = lipgloss.NewStyle().Foreground(subtle)
	t.Blurred.Option = lipgloss.NewStyle().Foreground(subtle)
	t.Blurred.MultiSelectSelector = lipgloss.NewStyle().Foreground(subtle)
	t.Blurred.SelectedOption = lipgloss.NewStyle().Foreground(subtle)
	t.Blurred.SelectedPrefix = lipgloss.NewStyle().Foreground(subtle).SetString("✓ ")
	t.Blurred.UnselectedPrefix = lipgloss.NewStyle().Foreground(subtle).SetString("· ")
	t.Blurred.TextInput.Cursor = lipgloss.NewStyle().Foreground(subtle)
	t.Blurred.TextInput.Placeholder = lipgloss.NewStyle().Foreground(subtle)
	t.Blurred.TextInput.Prompt = lipgloss.NewStyle().Foreground(subtle)

	return t
}

// Styles holds all the UI styles for the application
type Styles struct {
	theme Theme

	// Base styles
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Description lipgloss.Style

	// Status styles
	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style
	Muted   lipgloss.Style

	// Essential component styles
	List     lipgloss.Style
	ListItem lipgloss.Style

	// File status styles
	Path lipgloss.Style
	Tag  lipgloss.Style
	Hook lipgloss.Style

	// Progress styles
	Spinner lipgloss.Style

	// Table styles
	TableHeader lipgloss.Style
	TableCell   lipgloss.Style
}

// NewStyles creates a new set of styles with the default theme
func NewStyles() *Styles {
	theme := DefaultTheme()
	s := &Styles{theme: theme}

	// Essential styles only
	s.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Normal)

	s.Subtitle = lipgloss.NewStyle().
		Foreground(theme.Subtle)

	s.Description = lipgloss.NewStyle().
		Foreground(theme.Subtle)

	// Status styles mapped to our minimal palette
	s.Success = lipgloss.NewStyle().
		Foreground(theme.Accent)

	s.Error = lipgloss.NewStyle().
		Foreground(theme.Danger)

	s.Warning = lipgloss.NewStyle().
		Foreground(theme.Subtle)

	s.Info = lipgloss.NewStyle().
		Foreground(theme.Accent)

	s.Muted = lipgloss.NewStyle().
		Foreground(theme.Subtle)

	// Simple list styles without decorative borders
	s.List = lipgloss.NewStyle().
		MarginLeft(2)

	s.ListItem = lipgloss.NewStyle().
		PaddingLeft(2)

	// File status styles
	s.Path = lipgloss.NewStyle().
		Foreground(theme.Normal)

	s.Tag = lipgloss.NewStyle().
		Foreground(theme.Subtle)

	s.Hook = lipgloss.NewStyle().
		Foreground(theme.Subtle)

	// Minimal spinner style
	s.Spinner = lipgloss.NewStyle().
		Foreground(theme.Subtle)

	// Simple table styles without heavy borders
	s.TableHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Normal)

	s.TableCell = lipgloss.NewStyle().
		PaddingRight(2)

	return s
}

// StatusIcon returns an appropriate icon for the status
func (s *Styles) StatusIcon(status string) string {
	switch status {
	case "success", "link", "synced":
		return "✓"
	case "error", "missing", "broken":
		return "×"
	case "warning", "modified":
		return "~"
	case "skip":
		return "·"
	case "info":
		return "›"
	default:
		return "·"
	}
}

// FormatPath formats a file path with appropriate styling
func (s *Styles) FormatPath(path string, isHome bool) string {
	if isHome {
		return s.Path.Render("~/" + path)
	}
	return s.Path.Render(path)
}

// FormatTags formats a list of tags with appropriate styling
func (s *Styles) FormatTags(tags []string) string {
	if len(tags) == 0 {
		return s.Muted.Render("[no tags]")
	}
	var builder strings.Builder
	builder.WriteString("[")
	for i, tag := range tags {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(tag)
	}
	builder.WriteString("]")
	return s.Tag.Render(builder.String())
}
