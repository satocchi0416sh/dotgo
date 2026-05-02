// Tests for the tree-mode status renderer.
//
// Pattern: table-driven tests that build a TreeNode (or render one), then
// compare against expected slices/strings. ANSI escapes are stripped by
// installing a default renderer bound to io.Discard, which lipgloss detects
// as a non-tty and therefore renders in the Ascii profile.
package ui

import (
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/satocchi0416sh/dotgo/internal/engine"
)

func TestMain(m *testing.M) {
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(io.Discard))
	m.Run()
}

func flattenTree(root *TreeNode) []string {
	var out []string
	var walk func(n *TreeNode, depth int)
	walk = func(n *TreeNode, depth int) {
		indent := strings.Repeat("  ", depth)
		var label string
		if n.IsDir {
			suffix := "/"
			if n.Name == "~" {
				suffix = ""
			}
			if n.Status != nil {
				label = n.Name + suffix + "(" + n.Category + ")"
			} else {
				label = n.Name + suffix
			}
		} else {
			label = n.Name + "(" + n.Category + ")"
		}
		out = append(out, indent+label)
		if len(n.Children) == 0 {
			return
		}
		keys := make([]string, 0, len(n.Children))
		for k := range n.Children {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			ci := n.Children[keys[i]]
			cj := n.Children[keys[j]]
			if ci.IsDir != cj.IsDir {
				return ci.IsDir
			}
			return ci.Name < cj.Name
		})
		for _, k := range keys {
			walk(n.Children[k], depth+1)
		}
	}
	walk(root, 0)
	return out
}

func TestBuildStatusTree(t *testing.T) {
	tests := []struct {
		name           string
		statuses       []engine.LinkStatus
		includeSkipped bool
		want           []string
	}{
		{
			name:     "empty input",
			statuses: nil,
			want:     []string{"~"},
		},
		{
			name: "single root file",
			statuses: []engine.LinkStatus{
				{TargetPath: ".zshrc", IsCorrect: true, ShouldApply: true},
			},
			want: []string{
				"~",
				"  .zshrc(synced)",
			},
		},
		{
			name: "nested config files",
			statuses: []engine.LinkStatus{
				{TargetPath: ".config/nvim/init.vim", IsCorrect: true, ShouldApply: true},
				{TargetPath: ".config/nvim/lua/x.lua", IsCorrect: true, ShouldApply: true},
			},
			want: []string{
				"~",
				"  .config/",
				"    nvim/",
				"      lua/",
				"        x.lua(synced)",
				"      init.vim(synced)",
			},
		},
		{
			name: "skipped excluded by default",
			statuses: []engine.LinkStatus{
				{TargetPath: ".bashrc", IsCorrect: true, ShouldApply: true},
				{TargetPath: ".workrc", ShouldApply: false},
			},
			includeSkipped: false,
			want: []string{
				"~",
				"  .bashrc(synced)",
			},
		},
		{
			name: "skipped re-included with verbose",
			statuses: []engine.LinkStatus{
				{TargetPath: ".bashrc", IsCorrect: true, ShouldApply: true},
				{TargetPath: ".workrc", ShouldApply: false},
			},
			includeSkipped: true,
			want: []string{
				"~",
				"  .bashrc(synced)",
				"  .workrc(skipped)",
			},
		},
		{
			name: "dirs sort before files at same level",
			statuses: []engine.LinkStatus{
				{TargetPath: ".zshrc", IsCorrect: true, ShouldApply: true},
				{TargetPath: ".config/nvim/init.vim", IsCorrect: true, ShouldApply: true},
				{TargetPath: ".bashrc", IsCorrect: true, ShouldApply: true},
			},
			want: []string{
				"~",
				"  .config/",
				"    nvim/",
				"      init.vim(synced)",
				"  .bashrc(synced)",
				"  .zshrc(synced)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := BuildStatusTree(tt.statuses, tt.includeSkipped)
			got := flattenTree(root)
			if len(got) != len(tt.want) {
				t.Fatalf("flatten length = %d, want %d\n got: %v\nwant: %v", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildStatusTree_FileDirCollisionDoesNotPanic(t *testing.T) {
	statuses := []engine.LinkStatus{
		{TargetPath: ".config", IsCorrect: true, ShouldApply: true, LinkTarget: "/dot/.config"},
		{TargetPath: ".config/nvim/init.vim", IsCorrect: true, ShouldApply: true, LinkTarget: "/dot/init.vim"},
	}

	root := BuildStatusTree(statuses, false)
	got := flattenTree(root)

	want := []string{
		"~",
		"  .config/(synced)",
		"    nvim/",
		"      init.vim(synced)",
	}
	if len(got) != len(want) {
		t.Fatalf("flatten length = %d, want %d\n got: %v\nwant: %v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}

	statusesReversed := []engine.LinkStatus{
		{TargetPath: ".config/nvim/init.vim", IsCorrect: true, ShouldApply: true, LinkTarget: "/dot/init.vim"},
		{TargetPath: ".config", IsCorrect: true, ShouldApply: true, LinkTarget: "/dot/.config"},
	}
	rootReversed := BuildStatusTree(statusesReversed, false)
	gotReversed := flattenTree(rootReversed)
	if len(gotReversed) != len(want) {
		t.Fatalf("reversed flatten length = %d, want %d\n got: %v", len(gotReversed), len(want), gotReversed)
	}
	for i := range gotReversed {
		if gotReversed[i] != want[i] {
			t.Errorf("reversed line %d: got %q, want %q", i, gotReversed[i], want[i])
		}
	}
}

func TestStatusTree(t *testing.T) {
	tests := []struct {
		name           string
		statuses       []engine.LinkStatus
		includeSkipped bool
		want           string
	}{
		{
			name: "single synced root file",
			statuses: []engine.LinkStatus{
				{TargetPath: ".zshrc", IsCorrect: true, ShouldApply: true, LinkTarget: "/dot/.zshrc"},
			},
			want: "~\n" +
				"└─ ✓ .zshrc → /dot/.zshrc",
		},
		{
			name: "synced root file plus broken nested file",
			statuses: []engine.LinkStatus{
				{TargetPath: ".zshrc", IsCorrect: true, ShouldApply: true, LinkTarget: "/dot/.zshrc"},
				{TargetPath: ".config/nvim/init.vim", IsSymlink: true, Exists: false, ShouldApply: true, LinkTarget: "/dot/init.vim"},
			},
			want: "~\n" +
				"├─   .config/\n" +
				"│  └─   nvim/\n" +
				"│     └─ × init.vim → /dot/init.vim\n" +
				"└─ ✓ .zshrc → /dot/.zshrc",
		},
		{
			name: "modified and missing leaves",
			statuses: []engine.LinkStatus{
				{TargetPath: ".vimrc", Exists: true, IsSymlink: false, ShouldApply: true},
				{TargetPath: ".tmux.conf", Exists: false, ShouldApply: true},
			},
			want: "~\n" +
				"├─ × .tmux.conf missing symlink\n" +
				"└─ ~ .vimrc exists but not a symlink",
		},
	}

	ui := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := BuildStatusTree(tt.statuses, tt.includeSkipped)
			got := ui.StatusTree(root)
			if got != tt.want {
				t.Errorf("StatusTree mismatch\n got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestStatusTree_WideCharRespectsCellWidth(t *testing.T) {
	ui := New()
	statuses := []engine.LinkStatus{
		{
			TargetPath:  ".gitconfig",
			IsCorrect:   true,
			ShouldApply: true,
			LinkTarget:  strings.Repeat("あ", 60),
		},
	}
	root := BuildStatusTree(statuses, false)
	out := ui.StatusTree(root)
	for line := range strings.SplitSeq(out, "\n") {
		if w := lipgloss.Width(line); w > treeMaxLineWidth {
			t.Errorf("line exceeds treeMaxLineWidth=%d (got %d): %q", treeMaxLineWidth, w, line)
		}
	}
}

// TestUISectionAndFileStatusRendering_StableSnapshot guards the renderer
// primitives that printFlatSections invokes. It does NOT execute
// printFlatSections itself; that helper is byte-identical to the legacy
// runStatus L87-156 block by construction (verbatim extraction confirmed by
// `git diff main`). The snapshot here protects the rendering side so that
// changes to Section/FileStatus surface as test failures rather than silent
// drift in --flat output.
func TestUISectionAndFileStatusRendering_StableSnapshot(t *testing.T) {
	ui := New()
	statuses := []engine.LinkStatus{
		{TargetPath: ".zshrc", IsCorrect: true, ShouldApply: true, LinkTarget: "/dot/.zshrc"},
		{TargetPath: ".config/nvim/init.vim", IsSymlink: true, Exists: false, ShouldApply: true, LinkTarget: "/dot/init.vim"},
	}

	var b strings.Builder

	syncedItems := []string{
		ui.FileStatus("link", statuses[0].TargetPath, "→ "+statuses[0].LinkTarget, nil),
	}
	b.WriteString(ui.Section("TRACKED", syncedItems...))
	b.WriteString("\n")

	brokenItems := []string{
		ui.FileStatus("broken", statuses[1].TargetPath, "→ "+statuses[1].LinkTarget, nil),
	}
	b.WriteString(ui.Section("BROKEN LINKS", brokenItems...))

	got := b.String()

	const want = "       \nTRACKED\n  ✓ .zshrc → /dot/.zshrc [no tags]\n            \nBROKEN LINKS\n  × .config/nvim/init.vim → /dot/init.vim [no tags]"

	if got != want {
		t.Errorf("legacy renderer output drifted\n got: %q\nwant: %q", got, want)
	}
}
