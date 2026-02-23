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

// InviteUserEntry holds a workspace user's data for the invite picker.
type InviteUserEntry struct {
	UserID      string
	DisplayName string
	RealName    string
}

// InvitePicker is a modal popup for fuzzy-searching and selecting a user to invite.
type InvitePicker struct {
	*tview.Flex
	cfg      *config.Config
	input    *tview.InputField
	list     *tview.List
	status   *tview.TextView
	users    []InviteUserEntry
	filtered []int // indices into users for current filter
	onSelect func(userID string)
	onClose  func()
}

// NewInvitePicker creates a new invite picker component.
func NewInvitePicker(cfg *config.Config) *InvitePicker {
	ip := &InvitePicker{
		cfg: cfg,
	}

	ip.input = tview.NewInputField()
	ip.input.SetLabel(" Search: ")
	ip.input.SetFieldBackgroundColor(cfg.Theme.Modal.InputBackground.Background())
	ip.input.SetChangedFunc(ip.onInputChanged)
	ip.input.SetInputCapture(ip.handleInput)

	ip.list = tview.NewList()
	ip.list.SetHighlightFullLine(true)
	ip.list.ShowSecondaryText(true)
	ip.list.SetWrapAround(false)
	ip.list.SetSecondaryTextColor(tcell.ColorGray)

	ip.status = tview.NewTextView()
	ip.status.SetTextAlign(tview.AlignLeft)
	ip.status.SetDynamicColors(true)

	ip.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ip.input, 1, 0, true).
		AddItem(ip.list, 0, 1, false).
		AddItem(ip.status, 1, 0, false)
	ip.SetBorder(true).SetTitle(" Invite User ")

	return ip
}

// SetOnSelect sets the callback for user selection.
func (ip *InvitePicker) SetOnSelect(fn func(userID string)) {
	ip.onSelect = fn
}

// SetOnClose sets the callback for closing the picker.
func (ip *InvitePicker) SetOnClose(fn func()) {
	ip.onClose = fn
}

// SetUsers populates the picker with user entries and shows all.
func (ip *InvitePicker) SetUsers(users []InviteUserEntry) {
	if users == nil {
		ip.users = []InviteUserEntry{}
	} else {
		ip.users = users
	}
	ip.showAll()
	ip.updateStatus()
}

// SetStatus sets the status bar text.
func (ip *InvitePicker) SetStatus(text string) {
	ip.status.SetText(text)
}

// Reset clears the input and shows all users.
func (ip *InvitePicker) Reset() {
	ip.input.SetText("")
	ip.showAll()
}

// FilteredCount returns the number of currently visible entries.
func (ip *InvitePicker) FilteredCount() int {
	return len(ip.filtered)
}

// handleInput processes keybindings for the picker input field.
func (ip *InvitePicker) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch {
	case name == ip.cfg.Keybinds.InvitePicker.Close:
		ip.close()
		return nil

	case name == ip.cfg.Keybinds.InvitePicker.Select:
		ip.selectCurrent()
		return nil

	case name == ip.cfg.Keybinds.InvitePicker.Up || event.Key() == tcell.KeyUp:
		cur := ip.list.GetCurrentItem()
		if cur > 0 {
			ip.list.SetCurrentItem(cur - 1)
		}
		return nil

	case name == ip.cfg.Keybinds.InvitePicker.Down || event.Key() == tcell.KeyDown:
		cur := ip.list.GetCurrentItem()
		if cur < ip.list.GetItemCount()-1 {
			ip.list.SetCurrentItem(cur + 1)
		}
		return nil
	}

	return event
}

// onInputChanged filters the list based on the current search text.
func (ip *InvitePicker) onInputChanged(text string) {
	if text == "" {
		ip.showAll()
		ip.updateStatus()
		return
	}

	// Build search targets.
	targets := make([]string, len(ip.users))
	for i, u := range ip.users {
		targets[i] = inviteUserSearchText(u)
	}

	matches := fuzzy.Find(text, targets)

	ip.filtered = make([]int, len(matches))
	for i, m := range matches {
		ip.filtered[i] = m.Index
	}

	ip.rebuildList()
	ip.updateStatus()
}

// showAll displays all users (no filter).
func (ip *InvitePicker) showAll() {
	ip.filtered = make([]int, len(ip.users))
	for i := range ip.users {
		ip.filtered[i] = i
	}
	ip.rebuildList()
}

// rebuildList updates the tview.List from the filtered entries.
func (ip *InvitePicker) rebuildList() {
	ip.list.Clear()
	for _, idx := range ip.filtered {
		u := ip.users[idx]
		mainText := inviteDisplayText(u)
		secondaryText := inviteSecondaryText(u)
		ip.list.AddItem(mainText, secondaryText, 0, nil)
	}
	if ip.list.GetItemCount() > 0 {
		ip.list.SetCurrentItem(0)
	}
}

// selectCurrent selects the currently highlighted user.
func (ip *InvitePicker) selectCurrent() {
	cur := ip.list.GetCurrentItem()
	if cur < 0 || cur >= len(ip.filtered) {
		return
	}

	entry := ip.users[ip.filtered[cur]]
	if ip.onSelect != nil {
		ip.onSelect(entry.UserID)
	}
	ip.close()
}

// close signals the picker should be hidden.
func (ip *InvitePicker) close() {
	if ip.onClose != nil {
		ip.onClose()
	}
}

// updateStatus updates the status text with user count.
func (ip *InvitePicker) updateStatus() {
	total := len(ip.users)
	shown := len(ip.filtered)
	if total == 0 {
		ip.status.SetText(" No users")
		return
	}
	if shown == total {
		if total == 1 {
			ip.status.SetText(" 1 user")
		} else {
			ip.status.SetText(fmt.Sprintf(" %d users", total))
		}
	} else {
		ip.status.SetText(fmt.Sprintf(" %d / %d users", shown, total))
	}
}

// inviteDisplayText returns the main display text for a user entry.
func inviteDisplayText(u InviteUserEntry) string {
	name := u.DisplayName
	if name == "" {
		name = u.RealName
	}
	if name == "" {
		name = u.UserID
	}
	return tview.Escape(name)
}

// inviteSecondaryText returns the secondary text for a user entry.
func inviteSecondaryText(u InviteUserEntry) string {
	if u.DisplayName != "" && u.RealName != "" && u.DisplayName != u.RealName {
		return "  " + tview.Escape(u.RealName)
	}
	return ""
}

// inviteUserSearchText returns a lowercased search string for fuzzy matching.
func inviteUserSearchText(u InviteUserEntry) string {
	parts := []string{
		strings.ToLower(u.DisplayName),
		strings.ToLower(u.RealName),
	}
	return strings.Join(parts, " ")
}
