package panel

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/afcollins/kbx/internal/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

type FilePickerPanel struct {
	Dir      string
	Files    []string // all matching files
	Cursor   int
	Scroll   int
	Selected map[string]bool
	Width    int
	Height   int

	// Search/filter
	Searching bool
	SearchBuf string
	filtered  []int // indices into Files matching search
}

func NewFilePickerPanel() *FilePickerPanel {
	return NewFilePickerPanelDir("")
}

// NewFilePickerPanelDir creates a file picker for the given directory.
// If dir is empty, the current working directory is used.
func NewFilePickerPanelDir(dir string) *FilePickerPanel {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	fp := &FilePickerPanel{
		Dir:      dir,
		Selected: make(map[string]bool),
		Width:    80,
		Height:   20,
	}
	fp.Refresh()
	return fp
}

func (fp *FilePickerPanel) Refresh() {
	fp.Files = nil
	entries, err := os.ReadDir(fp.Dir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if isPickerFile(name) {
			fp.Files = append(fp.Files, name)
		}
	}
	sort.Strings(fp.Files)
	fp.rebuildFiltered()
}

func isPickerFile(name string) bool {
	return strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".log.gz") ||
		strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".json.gz")
}

func (fp *FilePickerPanel) rebuildFiltered() {
	fp.filtered = nil
	query := strings.ToLower(fp.SearchBuf)
	for i, name := range fp.Files {
		if query == "" || strings.Contains(strings.ToLower(name), query) {
			fp.filtered = append(fp.filtered, i)
		}
	}
	if fp.Cursor >= len(fp.filtered) {
		fp.Cursor = len(fp.filtered) - 1
	}
	if fp.Cursor < 0 {
		fp.Cursor = 0
	}
	fp.Scroll = 0
}

func (fp *FilePickerPanel) visibleRows() int {
	// Height minus: title, dir, help, search bar, top/bottom padding, borders
	v := fp.Height - 8
	if v < 1 {
		v = 1
	}
	return v
}

func (fp *FilePickerPanel) MoveUp() {
	if fp.Cursor > 0 {
		fp.Cursor--
		if fp.Cursor < fp.Scroll {
			fp.Scroll = fp.Cursor
		}
	}
}

func (fp *FilePickerPanel) MoveDown() {
	if fp.Cursor < len(fp.filtered)-1 {
		fp.Cursor++
		vis := fp.visibleRows()
		if fp.Cursor >= fp.Scroll+vis {
			fp.Scroll = fp.Cursor - vis + 1
		}
	}
}

func (fp *FilePickerPanel) ToggleSelection() {
	if fp.Cursor < len(fp.filtered) {
		name := fp.Files[fp.filtered[fp.Cursor]]
		if fp.Selected[name] {
			delete(fp.Selected, name)
		} else {
			fp.Selected[name] = true
		}
	}
}

func (fp *FilePickerPanel) SelectedPaths() []string {
	var paths []string
	for name := range fp.Selected {
		paths = append(paths, filepath.Join(fp.Dir, name))
	}
	sort.Strings(paths)
	return paths
}

// HandleSearchKey processes a key during search mode.
// Returns true if the key was consumed.
func (fp *FilePickerPanel) HandleSearchKey(key string) bool {
	switch key {
	case "enter", "esc":
		fp.Searching = false
		if key == "esc" {
			fp.SearchBuf = ""
			fp.rebuildFiltered()
		}
		return true
	case "backspace":
		if len(fp.SearchBuf) > 0 {
			fp.SearchBuf = fp.SearchBuf[:len(fp.SearchBuf)-1]
			fp.rebuildFiltered()
		}
		return true
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			fp.SearchBuf += key
			fp.rebuildFiltered()
			return true
		}
	}
	return false
}

func (fp *FilePickerPanel) View() string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Select audit log or metrics files"))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(
		fmt.Sprintf("Dir: %s", fp.Dir)))
	b.WriteString("\n")

	helpParts := "[↑↓] navigate  [Space] toggle  [Enter] load  [/] filter  [q] quit"
	b.WriteString(styles.HelpStyle.Render(helpParts))
	b.WriteString("\n")

	// Search bar
	if fp.Searching {
		b.WriteString(lipgloss.NewStyle().Foreground(styles.ColorAccent).Render(
			fmt.Sprintf("/%s█", fp.SearchBuf)))
	} else if fp.SearchBuf != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(
			fmt.Sprintf("filter: %s", fp.SearchBuf)))
	}
	b.WriteString("\n")

	vis := fp.visibleRows()
	end := fp.Scroll + vis
	if end > len(fp.filtered) {
		end = len(fp.filtered)
	}

	for i := fp.Scroll; i < end; i++ {
		fileIdx := fp.filtered[i]
		name := fp.Files[fileIdx]
		prefix := "  "
		if fp.Selected[name] {
			prefix = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("✓ ")
		}

		line := prefix + name
		if i == fp.Cursor {
			line = styles.SelectedStyle.Render(line)
		}

		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	if len(fp.filtered) == 0 {
		if fp.SearchBuf != "" {
			b.WriteString(lipgloss.NewStyle().Foreground(styles.ColorDanger).Render(
				fmt.Sprintf("No files matching '%s'", fp.SearchBuf)))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(styles.ColorDanger).Render(
				"No .log, .log.gz, .json, or .json.gz files found"))
		}
	}

	sel := len(fp.Selected)
	if sel > 0 {
		b.WriteString(fmt.Sprintf("\n\n%d file(s) selected", sel))
	}

	panelStyle := styles.FocusedPanelStyle.Width(fp.Width - 2).Height(fp.Height - 2)
	return panelStyle.Render(b.String())
}
