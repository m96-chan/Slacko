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

// MemberEntry holds a channel member's data for the picker.
type MemberEntry struct {
	UserID      string
	DisplayName string
	RealName    string
	Presence    string // "active", "away", ""
	IsBot       bool
}

// MembersPicker is a modal popup for fuzzy-searching and viewing channel members.
type MembersPicker struct {
	*tview.Flex
	cfg      *config.Config
	input    *tview.InputField
	list     *tview.List
	status   *tview.TextView
	members  []MemberEntry
	filtered []int // indices into members for current filter
	onSelect func(userID string)
	onClose  func()
}

// NewMembersPicker creates a new members picker component.
func NewMembersPicker(cfg *config.Config) *MembersPicker {
	mp := &MembersPicker{
		cfg: cfg,
	}

	mp.input = tview.NewInputField()
	mp.input.SetLabel(" Search: ")
	mp.input.SetFieldBackgroundColor(cfg.Theme.Modal.InputBackground.Background())
	mp.input.SetChangedFunc(mp.onInputChanged)
	mp.input.SetInputCapture(mp.handleInput)

	mp.list = tview.NewList()
	mp.list.SetHighlightFullLine(true)
	mp.list.ShowSecondaryText(true)
	mp.list.SetWrapAround(false)
	mp.list.SetSecondaryTextColor(tcell.ColorGray)

	mp.status = tview.NewTextView()
	mp.status.SetTextAlign(tview.AlignLeft)
	mp.status.SetDynamicColors(true)

	mp.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mp.input, 1, 0, true).
		AddItem(mp.list, 0, 1, false).
		AddItem(mp.status, 1, 0, false)
	mp.SetBorder(true).SetTitle(" Channel Members ")

	return mp
}

// SetOnSelect sets the callback for member selection.
func (mp *MembersPicker) SetOnSelect(fn func(userID string)) {
	mp.onSelect = fn
}

// SetOnClose sets the callback for closing the picker.
func (mp *MembersPicker) SetOnClose(fn func()) {
	mp.onClose = fn
}

// SetMembers populates the picker with member entries and shows all.
func (mp *MembersPicker) SetMembers(members []MemberEntry) {
	if members == nil {
		mp.members = []MemberEntry{}
	} else {
		mp.members = members
	}
	mp.showAll()
	mp.updateStatus()
}

// SetStatus sets the status bar text.
func (mp *MembersPicker) SetStatus(text string) {
	mp.status.SetText(text)
}

// Reset clears the input and shows all members.
func (mp *MembersPicker) Reset() {
	mp.input.SetText("")
	mp.showAll()
}

// FilteredCount returns the number of currently visible entries.
func (mp *MembersPicker) FilteredCount() int {
	return len(mp.filtered)
}

// handleInput processes keybindings for the picker input field.
func (mp *MembersPicker) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch {
	case name == mp.cfg.Keybinds.MembersPicker.Close:
		mp.close()
		return nil

	case name == mp.cfg.Keybinds.MembersPicker.Select:
		mp.selectCurrent()
		return nil

	case name == mp.cfg.Keybinds.MembersPicker.Up || event.Key() == tcell.KeyUp:
		cur := mp.list.GetCurrentItem()
		if cur > 0 {
			mp.list.SetCurrentItem(cur - 1)
		}
		return nil

	case name == mp.cfg.Keybinds.MembersPicker.Down || event.Key() == tcell.KeyDown:
		cur := mp.list.GetCurrentItem()
		if cur < mp.list.GetItemCount()-1 {
			mp.list.SetCurrentItem(cur + 1)
		}
		return nil
	}

	return event
}

// onInputChanged filters the list based on the current search text.
func (mp *MembersPicker) onInputChanged(text string) {
	if text == "" {
		mp.showAll()
		mp.updateStatus()
		return
	}

	// Build search targets.
	targets := make([]string, len(mp.members))
	for i, m := range mp.members {
		targets[i] = memberSearchText(m)
	}

	matches := fuzzy.Find(text, targets)

	mp.filtered = make([]int, len(matches))
	for i, m := range matches {
		mp.filtered[i] = m.Index
	}

	mp.rebuildList()
	mp.updateStatus()
}

// showAll displays all members (no filter).
func (mp *MembersPicker) showAll() {
	mp.filtered = make([]int, len(mp.members))
	for i := range mp.members {
		mp.filtered[i] = i
	}
	mp.rebuildList()
}

// rebuildList updates the tview.List from the filtered entries.
func (mp *MembersPicker) rebuildList() {
	mp.list.Clear()
	for _, idx := range mp.filtered {
		m := mp.members[idx]
		mainText := memberDisplayText(m)
		secondaryText := memberSecondaryText(m)
		mp.list.AddItem(mainText, secondaryText, 0, nil)
	}
	if mp.list.GetItemCount() > 0 {
		mp.list.SetCurrentItem(0)
	}
}

// selectCurrent selects the currently highlighted member.
func (mp *MembersPicker) selectCurrent() {
	cur := mp.list.GetCurrentItem()
	if cur < 0 || cur >= len(mp.filtered) {
		return
	}

	entry := mp.members[mp.filtered[cur]]
	if mp.onSelect != nil {
		mp.onSelect(entry.UserID)
	}
	mp.close()
}

// close signals the picker should be hidden.
func (mp *MembersPicker) close() {
	if mp.onClose != nil {
		mp.onClose()
	}
}

// updateStatus updates the status text with member count.
func (mp *MembersPicker) updateStatus() {
	total := len(mp.members)
	shown := len(mp.filtered)
	if total == 0 {
		mp.status.SetText(" No members")
		return
	}
	if shown == total {
		if total == 1 {
			mp.status.SetText(" 1 member")
		} else {
			mp.status.SetText(fmt.Sprintf(" %d members", total))
		}
	} else {
		mp.status.SetText(fmt.Sprintf(" %d / %d members", shown, total))
	}
}

// memberDisplayText returns the main display text for a member entry.
func memberDisplayText(m MemberEntry) string {
	var icon string
	switch {
	case m.IsBot:
		icon = "[::d]BOT[::-] "
	case m.Presence == "active":
		icon = "[green]\u25cf[-] "
	case m.Presence == "away":
		icon = "[gray]\u25cb[-] "
	default:
		icon = "[gray]\u25cb[-] "
	}

	name := m.DisplayName
	if name == "" {
		name = m.RealName
	}
	if name == "" {
		name = m.UserID
	}

	return icon + tview.Escape(name)
}

// memberSecondaryText returns the secondary (smaller) text for a member entry.
func memberSecondaryText(m MemberEntry) string {
	if m.DisplayName != "" && m.RealName != "" && m.DisplayName != m.RealName {
		return "  " + tview.Escape(m.RealName)
	}
	return ""
}

// memberSearchText returns a lowercased search string for fuzzy matching.
func memberSearchText(m MemberEntry) string {
	parts := []string{
		strings.ToLower(m.DisplayName),
		strings.ToLower(m.RealName),
	}
	return strings.Join(parts, " ")
}
