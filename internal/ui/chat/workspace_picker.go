package chat

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// WorkspaceEntry represents a workspace for display in the picker.
type WorkspaceEntry struct {
	ID   string
	Name string
}

// WorkspacePicker is a modal for selecting a workspace.
type WorkspacePicker struct {
	*tview.Flex
	cfg        *config.Config
	list       *tview.List
	status     *tview.TextView
	entries    []WorkspaceEntry
	currentID  string
	onSelect   func(workspaceID string)
	onClose    func()
}

// NewWorkspacePicker creates a new workspace picker.
func NewWorkspacePicker(cfg *config.Config) *WorkspacePicker {
	wp := &WorkspacePicker{
		cfg:  cfg,
		list: tview.NewList(),
	}

	wp.list.ShowSecondaryText(false)
	wp.list.SetHighlightFullLine(true)
	wp.list.SetWrapAround(false)
	wp.list.SetBorder(true).SetTitle(" Switch Workspace ")
	wp.list.SetInputCapture(wp.handleInput)

	wp.status = tview.NewTextView().SetDynamicColors(true)

	wp.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(wp.list, 0, 1, true).
		AddItem(wp.status, 1, 0, false)

	return wp
}

// SetOnSelect sets the callback for workspace selection.
func (wp *WorkspacePicker) SetOnSelect(fn func(workspaceID string)) {
	wp.onSelect = fn
}

// SetOnClose sets the callback for closing the picker.
func (wp *WorkspacePicker) SetOnClose(fn func()) {
	wp.onClose = fn
}

// SetCurrentWorkspace marks the currently active workspace.
func (wp *WorkspacePicker) SetCurrentWorkspace(id string) {
	wp.currentID = id
}

// SetWorkspaces populates the workspace list.
func (wp *WorkspacePicker) SetWorkspaces(entries []WorkspaceEntry) {
	wp.entries = entries
	wp.list.Clear()
	for _, e := range entries {
		label := e.Name
		if e.ID == wp.currentID {
			label += " (current)"
		}
		wp.list.AddItem(label, "", 0, nil)
	}
	if len(entries) == 0 {
		wp.status.SetText(" No workspaces configured")
	} else {
		wp.status.SetText(fmt.Sprintf(" %d workspace(s) — Enter to switch, Esc to close", len(entries)))
	}
}

// SetStatus updates the status text.
func (wp *WorkspacePicker) SetStatus(s string) {
	wp.status.SetText(" " + s)
}

// Reset prepares the picker for display.
func (wp *WorkspacePicker) Reset() {
	if wp.list.GetItemCount() > 0 {
		wp.list.SetCurrentItem(0)
	}
}

func (wp *WorkspacePicker) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())
	_ = name

	switch event.Key() {
	case tcell.KeyEscape:
		if wp.onClose != nil {
			wp.onClose()
		}
		return nil
	case tcell.KeyEnter:
		idx := wp.list.GetCurrentItem()
		if idx >= 0 && idx < len(wp.entries) {
			e := wp.entries[idx]
			if e.ID != wp.currentID && wp.onSelect != nil {
				wp.onSelect(e.ID)
			} else if wp.onClose != nil {
				wp.onClose()
			}
		}
		return nil
	}

	return event
}
