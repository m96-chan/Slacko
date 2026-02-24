package chat

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// GroupDMUserEntry holds a user's data for the group DM picker.
type GroupDMUserEntry struct {
	UserID      string
	DisplayName string
}

// GroupDMPicker is a modal popup for selecting multiple users to create a group DM.
type GroupDMPicker struct {
	*tview.Flex
	cfg      *config.Config
	input    *tview.InputField
	list     *tview.List
	selected *tview.TextView // Shows selected users
	status   *tview.TextView
	users    []GroupDMUserEntry
	filtered []int              // indices into users for current filter (excludes chosen)
	chosen   []GroupDMUserEntry // Users already selected
	onCreate func(userIDs []string)
	onClose  func()
}

// NewGroupDMPicker creates a new group DM picker component.
func NewGroupDMPicker(cfg *config.Config) *GroupDMPicker {
	gp := &GroupDMPicker{
		cfg: cfg,
	}

	gp.input = tview.NewInputField()
	gp.input.SetLabel(" Search: ")
	gp.input.SetFieldBackgroundColor(cfg.Theme.Modal.InputBackground.Background())
	gp.input.SetChangedFunc(gp.onInputChanged)
	gp.input.SetInputCapture(gp.handleInput)

	gp.list = tview.NewList()
	gp.list.SetHighlightFullLine(true)
	gp.list.ShowSecondaryText(false)
	gp.list.SetWrapAround(false)

	gp.selected = tview.NewTextView()
	gp.selected.SetDynamicColors(true)
	gp.selected.SetTextAlign(tview.AlignLeft)

	gp.status = tview.NewTextView()
	gp.status.SetTextAlign(tview.AlignLeft)
	gp.status.SetDynamicColors(true)

	gp.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(gp.selected, 1, 0, false).
		AddItem(gp.input, 1, 0, true).
		AddItem(gp.list, 0, 1, false).
		AddItem(gp.status, 1, 0, false)
	gp.SetBorder(true).SetTitle(" Create Group DM ")

	return gp
}

// SetOnCreate sets the callback invoked when the user confirms the group DM creation.
func (gp *GroupDMPicker) SetOnCreate(fn func(userIDs []string)) {
	gp.onCreate = fn
}

// SetOnClose sets the callback for closing the picker.
func (gp *GroupDMPicker) SetOnClose(fn func()) {
	gp.onClose = fn
}

// SetUsers populates the picker with user entries and shows all.
func (gp *GroupDMPicker) SetUsers(users []GroupDMUserEntry) {
	if users == nil {
		gp.users = []GroupDMUserEntry{}
	} else {
		gp.users = users
	}
	gp.showAll()
	gp.updateStatus()
}

// SetStatus sets the status bar text.
func (gp *GroupDMPicker) SetStatus(text string) {
	gp.status.SetText(text)
}

// Reset clears the input, chosen users, and shows all users.
func (gp *GroupDMPicker) Reset() {
	gp.input.SetText("")
	gp.chosen = nil
	gp.updateSelectedDisplay()
	gp.showAll()
	gp.updateStatus()
}

// SelectedCount returns the number of currently chosen users.
func (gp *GroupDMPicker) SelectedCount() int {
	return len(gp.chosen)
}

// FilteredCount returns the number of currently visible entries.
func (gp *GroupDMPicker) FilteredCount() int {
	return len(gp.filtered)
}

// ChosenUserIDs returns the user IDs of all chosen users.
func (gp *GroupDMPicker) ChosenUserIDs() []string {
	ids := make([]string, len(gp.chosen))
	for i, u := range gp.chosen {
		ids[i] = u.UserID
	}
	return ids
}

// handleInput processes keybindings for the picker input field.
func (gp *GroupDMPicker) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch {
	case name == gp.cfg.Keybinds.GroupDMPicker.Close:
		gp.close()
		return nil

	case name == gp.cfg.Keybinds.GroupDMPicker.Confirm:
		gp.confirm()
		return nil

	case name == gp.cfg.Keybinds.GroupDMPicker.Add:
		gp.addCurrent()
		return nil

	case name == gp.cfg.Keybinds.GroupDMPicker.Remove:
		gp.removeLastChosen()
		return nil

	case name == gp.cfg.Keybinds.GroupDMPicker.Up || event.Key() == tcell.KeyUp:
		cur := gp.list.GetCurrentItem()
		if cur > 0 {
			gp.list.SetCurrentItem(cur - 1)
		}
		return nil

	case name == gp.cfg.Keybinds.GroupDMPicker.Down || event.Key() == tcell.KeyDown:
		cur := gp.list.GetCurrentItem()
		if cur < gp.list.GetItemCount()-1 {
			gp.list.SetCurrentItem(cur + 1)
		}
		return nil
	}

	return event
}

// onInputChanged filters the list based on the current search text.
func (gp *GroupDMPicker) onInputChanged(text string) {
	if text == "" {
		gp.showAll()
		gp.updateStatus()
		return
	}

	// Build search targets (excluding chosen users).
	chosenSet := gp.chosenSet()
	var targets []string
	var indices []int

	for i, u := range gp.users {
		if chosenSet[u.UserID] {
			continue
		}
		targets = append(targets, strings.ToLower(u.DisplayName))
		indices = append(indices, i)
	}

	matches := fuzzy.Find(text, targets)

	gp.filtered = make([]int, len(matches))
	for i, m := range matches {
		gp.filtered[i] = indices[m.Index]
	}

	gp.rebuildList()
	gp.updateStatus()
}

// showAll displays all users not already chosen.
func (gp *GroupDMPicker) showAll() {
	chosenSet := gp.chosenSet()
	gp.filtered = make([]int, 0, len(gp.users))
	for i, u := range gp.users {
		if !chosenSet[u.UserID] {
			gp.filtered = append(gp.filtered, i)
		}
	}
	gp.rebuildList()
}

// rebuildList updates the tview.List from the filtered entries.
func (gp *GroupDMPicker) rebuildList() {
	gp.list.Clear()
	for _, idx := range gp.filtered {
		u := gp.users[idx]
		gp.list.AddItem(tview.Escape(u.DisplayName), "", 0, nil)
	}
	if gp.list.GetItemCount() > 0 {
		gp.list.SetCurrentItem(0)
	}
}

// addCurrent adds the currently highlighted user to the chosen list.
func (gp *GroupDMPicker) addCurrent() {
	cur := gp.list.GetCurrentItem()
	if cur < 0 || cur >= len(gp.filtered) {
		return
	}

	entry := gp.users[gp.filtered[cur]]

	// Verify not already chosen (safety check).
	for _, c := range gp.chosen {
		if c.UserID == entry.UserID {
			return
		}
	}

	gp.chosen = append(gp.chosen, entry)
	gp.input.SetText("")
	gp.updateSelectedDisplay()
	gp.showAll()
	gp.updateStatus()
}

// removeLastChosen removes the last chosen user from the selection.
func (gp *GroupDMPicker) removeLastChosen() {
	if len(gp.chosen) == 0 {
		return
	}
	gp.chosen = gp.chosen[:len(gp.chosen)-1]
	gp.updateSelectedDisplay()
	gp.showAll()
	gp.updateStatus()
}

// confirm triggers the onCreate callback with the chosen user IDs.
func (gp *GroupDMPicker) confirm() {
	if len(gp.chosen) == 0 {
		return
	}
	if gp.onCreate != nil {
		gp.onCreate(gp.ChosenUserIDs())
	}
}

// close signals the picker should be hidden.
func (gp *GroupDMPicker) close() {
	if gp.onClose != nil {
		gp.onClose()
	}
}

// chosenSet builds a set of chosen user IDs for fast lookup.
func (gp *GroupDMPicker) chosenSet() map[string]bool {
	set := make(map[string]bool, len(gp.chosen))
	for _, c := range gp.chosen {
		set[c.UserID] = true
	}
	return set
}

// updateSelectedDisplay updates the text area showing currently chosen users.
func (gp *GroupDMPicker) updateSelectedDisplay() {
	if len(gp.chosen) == 0 {
		gp.selected.SetText(" [gray]Select users to create group DM[-]")
		return
	}

	names := make([]string, len(gp.chosen))
	for i, c := range gp.chosen {
		names[i] = c.DisplayName
	}
	gp.selected.SetText(" [green]Selected:[-] " + strings.Join(names, ", "))
}

// updateStatus updates the status text with user/selection counts.
func (gp *GroupDMPicker) updateStatus() {
	total := len(gp.users)
	chosen := len(gp.chosen)
	shown := len(gp.filtered)

	if total == 0 {
		gp.status.SetText(" No users available")
		return
	}

	parts := []string{
		fmt.Sprintf(" %d/%d users", shown, total-chosen),
	}
	if chosen > 0 {
		parts = append(parts, fmt.Sprintf("%d selected", chosen))
	}
	parts = append(parts, "[Enter]add [Ctrl+D]remove [Ctrl+Enter]create [Esc]cancel")
	gp.status.SetText(strings.Join(parts, "  "))
}
