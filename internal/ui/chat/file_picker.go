package chat

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// fileEntry represents a single file or directory in the picker.
type fileEntry struct {
	name  string
	path  string
	size  int64
	isDir bool
}

// FilePicker is a modal popup for browsing the filesystem and selecting a file.
type FilePicker struct {
	*tview.Flex
	cfg        *config.Config
	input      *tview.InputField
	list       *tview.List
	currentDir string
	entries    []fileEntry
	onSelect   func(path string)
	onClose    func()
}

// NewFilePicker creates a new file picker component.
func NewFilePicker(cfg *config.Config) *FilePicker {
	fp := &FilePicker{
		cfg: cfg,
	}

	fp.input = tview.NewInputField()
	fp.input.SetLabel(" Path: ")
	fp.input.SetFieldBackgroundColor(tcell.ColorDefault)
	fp.input.SetDoneFunc(fp.onInputDone)
	fp.input.SetInputCapture(fp.handleInput)

	fp.list = tview.NewList()
	fp.list.SetHighlightFullLine(true)
	fp.list.ShowSecondaryText(false)
	fp.list.SetWrapAround(false)

	fp.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(fp.input, 1, 0, true).
		AddItem(fp.list, 0, 1, false)
	fp.SetBorder(true).SetTitle(" Select File ")

	return fp
}

// SetOnSelect sets the callback for file selection.
func (fp *FilePicker) SetOnSelect(fn func(path string)) {
	fp.onSelect = fn
}

// SetOnClose sets the callback for closing the picker.
func (fp *FilePicker) SetOnClose(fn func()) {
	fp.onClose = fn
}

// Reset resets the picker to the user's home directory.
func (fp *FilePicker) Reset() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}
	fp.input.SetText("")
	fp.loadDir(home)
}

// loadDir reads the given directory and populates the list.
func (fp *FilePicker) loadDir(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	fp.currentDir = absPath
	fp.input.SetText(absPath)

	dirEntries, err := os.ReadDir(absPath)
	if err != nil {
		fp.entries = nil
		fp.list.Clear()
		fp.list.AddItem(fmt.Sprintf("  Error: %s", err.Error()), "", 0, nil)
		return
	}

	fp.entries = make([]fileEntry, 0, len(dirEntries)+1)

	// Add parent directory entry if not at root.
	if absPath != "/" {
		fp.entries = append(fp.entries, fileEntry{
			name:  "..",
			path:  filepath.Dir(absPath),
			isDir: true,
		})
	}

	// Separate dirs and files, sort each alphabetically.
	var dirs, files []fileEntry
	for _, de := range dirEntries {
		// Skip hidden files.
		if strings.HasPrefix(de.Name(), ".") {
			continue
		}
		info, infoErr := de.Info()
		var size int64
		if infoErr == nil {
			size = info.Size()
		}
		entry := fileEntry{
			name:  de.Name(),
			path:  filepath.Join(absPath, de.Name()),
			size:  size,
			isDir: de.IsDir(),
		}
		if de.IsDir() {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}

	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].name) < strings.ToLower(dirs[j].name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].name) < strings.ToLower(files[j].name)
	})

	fp.entries = append(fp.entries, dirs...)
	fp.entries = append(fp.entries, files...)

	fp.rebuildList()
}

// rebuildList updates the tview.List from entries.
func (fp *FilePicker) rebuildList() {
	fp.list.Clear()
	for _, e := range fp.entries {
		var display string
		if e.name == ".." {
			display = "  \U0001F4C1 .."
		} else if e.isDir {
			display = fmt.Sprintf("  \U0001F4C1 %s/", e.name)
		} else {
			icon := fileIcon(e.name)
			display = fmt.Sprintf("  %s %s  (%s)", icon, e.name, formatFileSize(int(e.size)))
		}
		fp.list.AddItem(display, "", 0, nil)
	}
	if fp.list.GetItemCount() > 0 {
		fp.list.SetCurrentItem(0)
	}
}

// handleInput processes keybindings for the picker.
func (fp *FilePicker) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch {
	case name == fp.cfg.Keybinds.FilePicker.Close:
		fp.close()
		return nil

	case name == fp.cfg.Keybinds.FilePicker.Select:
		fp.selectCurrent()
		return nil

	case name == fp.cfg.Keybinds.FilePicker.Up || event.Key() == tcell.KeyUp:
		cur := fp.list.GetCurrentItem()
		if cur > 0 {
			fp.list.SetCurrentItem(cur - 1)
		}
		return nil

	case name == fp.cfg.Keybinds.FilePicker.Down || event.Key() == tcell.KeyDown:
		cur := fp.list.GetCurrentItem()
		if cur < fp.list.GetItemCount()-1 {
			fp.list.SetCurrentItem(cur + 1)
		}
		return nil
	}

	return event
}

// onInputDone is called when the user presses Enter in the path input.
func (fp *FilePicker) onInputDone(key tcell.Key) {
	if key == tcell.KeyEnter {
		fp.selectCurrent()
	}
}

// selectCurrent opens the selected directory or picks the selected file.
func (fp *FilePicker) selectCurrent() {
	cur := fp.list.GetCurrentItem()
	if cur < 0 || cur >= len(fp.entries) {
		// If no list items, try navigating to the typed path.
		text := strings.TrimSpace(fp.input.GetText())
		if text != "" {
			info, err := os.Stat(text)
			if err == nil && info.IsDir() {
				fp.loadDir(text)
				return
			}
		}
		return
	}

	entry := fp.entries[cur]
	if entry.isDir {
		fp.loadDir(entry.path)
	} else {
		if fp.onSelect != nil {
			fp.onSelect(entry.path)
		}
		fp.close()
	}
}

// close signals the picker should be hidden.
func (fp *FilePicker) close() {
	if fp.onClose != nil {
		fp.onClose()
	}
}

// fileIcon returns a type-specific icon for a file based on its extension.
func fileIcon(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp", ".ico", ".tiff":
		return "\U0001F5BC" // framed picture
	case ".mp3", ".wav", ".flac", ".ogg", ".aac", ".wma", ".m4a":
		return "\U0001F3B5" // musical note
	case ".mp4", ".avi", ".mkv", ".mov", ".wmv", ".flv", ".webm":
		return "\U0001F3AC" // clapper board
	case ".zip", ".tar", ".gz", ".bz2", ".xz", ".rar", ".7z", ".tgz":
		return "\U0001F4E6" // package
	case ".pdf":
		return "\U0001F4C4" // page facing up
	case ".doc", ".docx", ".odt", ".rtf":
		return "\U0001F4C4" // page facing up
	case ".xls", ".xlsx", ".ods", ".csv":
		return "\U0001F4CA" // bar chart
	case ".ppt", ".pptx", ".odp":
		return "\U0001F4CA" // bar chart
	case ".go", ".py", ".js", ".ts", ".rs", ".c", ".cpp", ".h", ".java",
		".rb", ".sh", ".bash", ".zsh", ".lua", ".php", ".swift", ".kt",
		".scala", ".hs", ".ml", ".ex", ".exs", ".clj", ".lisp", ".r",
		".sql", ".html", ".css", ".scss", ".sass", ".less",
		".yml", ".yaml", ".json", ".xml", ".toml", ".ini", ".cfg":
		return "\U0001F4DD" // memo
	case ".txt", ".md", ".rst", ".log":
		return "\U0001F4DD" // memo
	default:
		return "\U0001F4CE" // paperclip
	}
}
