package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/satocchi0416sh/dotgo/internal/engine"
)

// treeMaxLineWidth bounds rendered tree lines so the aux annotation can be
// elided before terminal wrapping kicks in.
const treeMaxLineWidth = 78

// TreeNode is a single segment in the path-hierarchy view of a manifest.
// A node is either an internal directory or a leaf carrying a LinkStatus,
// but a node may legitimately be both when the manifest tracks a directory
// and a file inside it (e.g. ~/.config and ~/.config/nvim/init.vim).
type TreeNode struct {
	Name     string
	IsDir    bool
	Status   *engine.LinkStatus
	Category string
	Children map[string]*TreeNode
}

func categorize(s engine.LinkStatus) string {
	if !s.ShouldApply {
		return "skipped"
	}
	if s.IsCorrect {
		return "synced"
	}
	if s.IsSymlink && !s.Exists {
		return "broken"
	}
	if s.Exists && !s.IsSymlink {
		return "modified"
	}
	if !s.Exists {
		return "missing"
	}
	return ""
}

// BuildStatusTree groups statuses into a path-hierarchy tree rooted at "~".
// Skipped entries are included only when includeSkipped is true so callers
// can mirror the legacy --verbose gate. Collisions where the same segment is
// reached as both a leaf and an internal directory are resolved by attaching
// the leaf's status onto the shared node; rendering then surfaces both the
// status line and the descendants beneath it.
func BuildStatusTree(statuses []engine.LinkStatus, includeSkipped bool) *TreeNode {
	root := &TreeNode{Name: "~", IsDir: true, Children: map[string]*TreeNode{}}

	for _, s := range statuses {
		cat := categorize(s)
		if cat == "skipped" && !includeSkipped {
			continue
		}

		raw := strings.Split(s.TargetPath, "/")
		segments := raw[:0]
		for _, seg := range raw {
			if seg != "" {
				segments = append(segments, seg)
			}
		}
		if len(segments) == 0 {
			continue
		}

		statusCopy := s
		node := root
		for i, seg := range segments {
			isLeaf := i == len(segments)-1
			child, ok := node.Children[seg]
			if !ok {
				child = &TreeNode{
					Name:     seg,
					Children: map[string]*TreeNode{},
				}
				node.Children[seg] = child
			}
			if isLeaf {
				child.Status = &statusCopy
				child.Category = cat
				if len(child.Children) > 0 {
					child.IsDir = true
				}
			} else {
				child.IsDir = true
			}
			node = child
		}
	}

	return root
}

// StatusTree renders the path-hierarchy view starting at the synthetic root
// "~". Branch glyphs and the directory-vs-file alignment are stable so
// downstream snapshot tests remain meaningful.
func (ui *UI) StatusTree(root *TreeNode) string {
	var b strings.Builder
	b.WriteString(ui.styles.Path.Render("~"))
	b.WriteString("\n")

	keys := sortedChildKeys(root)
	for i, k := range keys {
		ui.renderTreeNode(&b, root.Children[k], nil, i == len(keys)-1)
	}

	return strings.TrimRight(b.String(), "\n")
}

func (ui *UI) renderTreeNode(b *strings.Builder, node *TreeNode, prefixContinues []bool, isLast bool) {
	var prefixBuilder strings.Builder
	for _, c := range prefixContinues {
		if c {
			prefixBuilder.WriteString("│  ")
		} else {
			prefixBuilder.WriteString("   ")
		}
	}
	if isLast {
		prefixBuilder.WriteString("└─ ")
	} else {
		prefixBuilder.WriteString("├─ ")
	}
	styledPrefix := ui.styles.Muted.Render(prefixBuilder.String())

	hasChildren := len(node.Children) > 0
	hasStatus := node.Status != nil

	if hasChildren && !hasStatus {
		// Pure directory row. Two leading spaces hold the column that a
		// leaf would use for its 1-cell status icon plus a separating space,
		// so directory names align with leaf names below.
		line := styledPrefix + "  " + ui.styles.Muted.Render(node.Name+"/")
		b.WriteString(line)
		b.WriteString("\n")

		keys := sortedChildKeys(node)
		nextContinues := append(append([]bool{}, prefixContinues...), !isLast)
		for i, k := range keys {
			ui.renderTreeNode(b, node.Children[k], nextContinues, i == len(keys)-1)
		}
		return
	}

	if hasStatus {
		iconStyle := ui.iconStyleFor(node.Category)
		icon := iconStyle.Render(ui.styles.StatusIcon(node.Category))
		nameRendered := ui.styles.Path.Render(node.Name)
		if hasChildren {
			nameRendered = ui.styles.Path.Render(node.Name + "/")
		}
		auxPlain := auxFor(node.Status, node.Category)

		prefixIconName := styledPrefix + icon + " " + nameRendered
		line := prefixIconName
		if auxPlain != "" {
			line = prefixIconName + " " + ui.styles.Muted.Render(auxPlain)
		}

		if lipgloss.Width(line) > treeMaxLineWidth && auxPlain != "" {
			base := lipgloss.Width(prefixIconName)
			allow := treeMaxLineWidth - base - 1
			if allow <= 1 {
				line = prefixIconName
			} else {
				truncated := truncateToCells(auxPlain, allow-1) + "…"
				line = prefixIconName + " " + ui.styles.Muted.Render(truncated)
			}
		}

		b.WriteString(line)
		b.WriteString("\n")

		if hasChildren {
			keys := sortedChildKeys(node)
			nextContinues := append(append([]bool{}, prefixContinues...), !isLast)
			for i, k := range keys {
				ui.renderTreeNode(b, node.Children[k], nextContinues, i == len(keys)-1)
			}
		}
		return
	}

	// Defensive fallback: a node with neither children nor status. Render as
	// a bare directory marker rather than panic.
	line := styledPrefix + "  " + ui.styles.Muted.Render(node.Name+"/")
	b.WriteString(line)
	b.WriteString("\n")
}

func (ui *UI) iconStyleFor(category string) lipgloss.Style {
	switch category {
	case "synced":
		return ui.styles.Success
	case "modified":
		return ui.styles.Warning
	case "missing", "broken":
		return ui.styles.Error
	case "skipped":
		return ui.styles.Muted
	default:
		return ui.styles.Muted
	}
}

func auxFor(status *engine.LinkStatus, cat string) string {
	if status == nil {
		return ""
	}
	switch cat {
	case "synced":
		return fmt.Sprintf("→ %s", status.LinkTarget)
	case "broken":
		return fmt.Sprintf("→ %s", status.LinkTarget)
	case "modified":
		return "exists but not a symlink"
	case "missing":
		return "missing symlink"
	case "skipped":
		return "tags don't match"
	default:
		return ""
	}
}

func sortedChildKeys(node *TreeNode) []string {
	keys := make([]string, 0, len(node.Children))
	for k := range node.Children {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		ci := node.Children[keys[i]]
		cj := node.Children[keys[j]]
		if ci.IsDir != cj.IsDir {
			return ci.IsDir
		}
		return ci.Name < cj.Name
	})
	return keys
}

// truncateToCells cuts s to fit within maxCells display columns. Width is
// measured per-rune via lipgloss so wide-cell glyphs (CJK, many emoji) are
// counted at their actual visual width.
func truncateToCells(s string, maxCells int) string {
	if maxCells <= 0 {
		return ""
	}
	var out strings.Builder
	used := 0
	for _, r := range s {
		w := lipgloss.Width(string(r))
		if used+w > maxCells {
			break
		}
		out.WriteRune(r)
		used += w
	}
	return out.String()
}
