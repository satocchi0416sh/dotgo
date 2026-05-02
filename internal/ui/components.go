package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// UI provides methods for rendering rich terminal output
type UI struct {
	styles  *Styles
	spinner spinner.Model
}

// New creates a new UI instance
func New() *UI {
	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	return &UI{
		styles:  NewStyles(),
		spinner: s,
	}
}

// Header renders a styled header
func (ui *UI) Header(title string, subtitle ...string) string {
	var output strings.Builder
	
	// Simple title with optional subtitle
	output.WriteString(ui.styles.Title.Render(title))
	
	if len(subtitle) > 0 && subtitle[0] != "" {
		output.WriteString(" / " + ui.styles.Subtitle.Render(subtitle[0]))
	}
	
	output.WriteString("\n")
	
	// Simple divider line
	divider := strings.Repeat("─", 48)
	output.WriteString(ui.styles.Muted.Render(divider))
	
	return output.String()
}

// StatusMessage renders a status message with an icon
func (ui *UI) StatusMessage(status, message string) string {
	icon := ui.styles.StatusIcon(status)
	
	var style lipgloss.Style
	switch status {
	case "success":
		style = ui.styles.Success
	case "error":
		style = ui.styles.Error
	case "warning":
		style = ui.styles.Warning
	case "info":
		style = ui.styles.Info
	default:
		style = ui.styles.Muted
	}
	
	return fmt.Sprintf("%s %s", style.Render(icon), message)
}

// FileStatus renders a file status line
func (ui *UI) FileStatus(icon, path, status string, tags []string) string {
	iconStr := ui.styles.StatusIcon(icon)
	
	var iconStyle lipgloss.Style
	switch icon {
	case "success", "link":
		iconStyle = ui.styles.Success
	case "error", "broken", "missing":
		iconStyle = ui.styles.Error
	case "warning", "modified":
		iconStyle = ui.styles.Warning
	case "skip":
		iconStyle = ui.styles.Muted
	default:
		iconStyle = ui.styles.Info
	}
	
	pathStr := ui.styles.Path.Render(path)
	statusStr := ui.styles.Muted.Render(status)
	tagsStr := ui.styles.FormatTags(tags)
	
	return fmt.Sprintf("  %s %s %s %s", 
		iconStyle.Render(iconStr),
		pathStr,
		statusStr,
		tagsStr,
	)
}

// Section renders a section with a title and optional content
func (ui *UI) Section(title string, items ...string) string {
	var output strings.Builder
	
	// Simple section header with uppercase
	header := ui.styles.Subtitle.
		Bold(true).
		MarginTop(1).
		Render(strings.ToUpper(title))
	output.WriteString(header + "\n")
	
	// Section items
	for _, item := range items {
		output.WriteString(item + "\n")
	}
	
	return strings.TrimRight(output.String(), "\n")
}

// Summary renders a summary box
func (ui *UI) Summary(stats map[string]int) string {
	var lines []string
	
	// Build summary lines with ASCII icons
	for key, value := range stats {
		var icon, label string
		var style lipgloss.Style
		
		switch key {
		case "total":
			icon = "›"
			label = "Total files"
			style = ui.styles.Info
		case "synced":
			icon = "✓"
			label = "Synced"
			style = ui.styles.Success
		case "missing":
			icon = "×"
			label = "Missing"
			style = ui.styles.Error
		case "broken":
			icon = "×"
			label = "Broken"
			style = ui.styles.Error
		case "modified":
			icon = "~"
			label = "Modified"
			style = ui.styles.Warning
		case "skipped":
			icon = "·"
			label = "Skipped"
			style = ui.styles.Muted
		default:
			icon = "·"
			label = key
			style = ui.styles.Muted
		}
		
		line := fmt.Sprintf("  %s %s: %s",
			style.Render(icon),
			label,
			style.Bold(true).Render(fmt.Sprintf("%d", value)),
		)
		lines = append(lines, line)
	}
	
	// Simple content without decorative box
	return strings.Join(lines, "\n")
}

// Progress renders a progress indicator
func (ui *UI) Progress(current, total int, message string) string {
	percentage := float64(current) / float64(total)
	barWidth := 30
	filled := int(percentage * float64(barWidth))
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	
	progressStyle := ui.styles.Info
	if percentage >= 1.0 {
		progressStyle = ui.styles.Success
	}
	
	return fmt.Sprintf("%s [%s] %d/%d - %s",
		ui.spinner.View(),
		progressStyle.Render(bar),
		current,
		total,
		message,
	)
}

// List renders a styled list
func (ui *UI) List(title string, items []string, ordered bool) string {
	var output strings.Builder
	
	if title != "" {
		output.WriteString(ui.styles.Subtitle.Render(title) + "\n")
	}
	
	for i, item := range items {
		var prefix string
		if ordered {
			prefix = fmt.Sprintf("%d.", i+1)
		} else {
			prefix = "•"
		}
		
		line := ui.styles.ListItem.Render(fmt.Sprintf("%s %s", 
			ui.styles.Muted.Render(prefix),
			item,
		))
		output.WriteString(line + "\n")
	}
	
	return strings.TrimRight(output.String(), "\n")
}

// CodeBlock renders a code block
func (ui *UI) CodeBlock(code string) string {
	// Simple code block without decorative background
	return ui.styles.Muted.Render(code)
}

// Badge renders a badge
func (ui *UI) Badge(text string) string {
	// Simple badge using accent color
	return ui.styles.Info.Bold(true).Render(text)
}

// Link renders a hyperlink-style text
func (ui *UI) Link(text string) string {
	return ui.styles.Info.
		Underline(true).
		Render(text)
}

// Duration formats a duration in a human-readable way
func (ui *UI) Duration(d time.Duration) string {
	if d < time.Millisecond {
		return ui.styles.Muted.Render(fmt.Sprintf("%dμs", d.Microseconds()))
	} else if d < time.Second {
		return ui.styles.Muted.Render(fmt.Sprintf("%dms", d.Milliseconds()))
	}
	return ui.styles.Muted.Render(fmt.Sprintf("%.1fs", d.Seconds()))
}

// Hint renders a hint message
func (ui *UI) Hint(message string) string {
	return ui.styles.Muted.
		Italic(true).
		Render(fmt.Sprintf("💡 %s", message))
}

// Table renders a simple table
func (ui *UI) Table(headers []string, rows [][]string) string {
	var output strings.Builder
	
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	
	// Render headers
	var headerLine strings.Builder
	for i, header := range headers {
		cell := ui.styles.TableHeader.
			Width(widths[i] + 2).
			Render(header)
		headerLine.WriteString(cell)
		if i < len(headers)-1 {
			headerLine.WriteString(" ")
		}
	}
	output.WriteString(headerLine.String() + "\n")
	
	// Render separator
	var separator strings.Builder
	for i, width := range widths {
		separator.WriteString(strings.Repeat("─", width+2))
		if i < len(widths)-1 {
			separator.WriteString(" ")
		}
	}
	output.WriteString(ui.styles.Muted.Render(separator.String()) + "\n")
	
	// Render rows
	for _, row := range rows {
		var rowLine strings.Builder
		for i, cell := range row {
			if i < len(widths) {
				cellStyle := ui.styles.TableCell.
					Width(widths[i] + 2)
				rowLine.WriteString(cellStyle.Render(cell))
				if i < len(row)-1 {
					rowLine.WriteString(" ")
				}
			}
		}
		output.WriteString(rowLine.String() + "\n")
	}
	
	return strings.TrimRight(output.String(), "\n")
}

// Divider renders a horizontal divider
func (ui *UI) Divider() string {
	width := 60
	return ui.styles.Muted.Render(strings.Repeat("─", width))
}