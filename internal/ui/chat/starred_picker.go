package chat

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// StarredEntry holds a single starred message for display.
type StarredEntry struct {
	ChannelID   string
	ChannelName string
	Timestamp   string
	UserName    string
	Text        string
}

// StarredPicker is a modal popup for viewing starred (bookmarked) messages.
type StarredPicker struct {
	*tview.Flex
	cfg      *config.Config
	list     *tview.List
	status   *tview.TextView
	entries  []StarredEntry
	onSelect func(channelID, timestamp string)
	onUnstar func(channelID, timestamp string)
	onClose  func()
}

// NewStarredPicker creates a new starred items picker component.
func NewStarredPicker(cfg *config.Config) *StarredPicker {
	sp := &StarredPicker{
		cfg: cfg,
	}

	sp.list = tview.NewList()
	sp.list.SetHighlightFullLine(true)
	sp.list.ShowSecondaryText(true)
	sp.list.SetWrapAround(false)
	sp.list.SetSecondaryTextColor(cfg.Theme.Modal.SecondaryText.Foreground())
	sp.list.SetInputCapture(sp.handleInput)

	sp.status = tview.NewTextView()
	sp.status.SetTextAlign(tview.AlignLeft)
	sp.status.SetDynamicColors(true)

	sp.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(sp.list, 0, 1, true).
		AddItem(sp.status, 1, 0, false)
	sp.SetBorder(true).SetTitle(" Starred Messages ")

	return sp
}

// SetOnSelect sets the callback for selecting a starred message.
func (sp *StarredPicker) SetOnSelect(fn func(channelID, timestamp string)) {
	sp.onSelect = fn
}

// SetOnUnstar sets the callback for removing a star from a message.
func (sp *StarredPicker) SetOnUnstar(fn func(channelID, timestamp string)) {
	sp.onUnstar = fn
}

// SetOnClose sets the callback for closing the picker.
func (sp *StarredPicker) SetOnClose(fn func()) {
	sp.onClose = fn
}

// Reset clears the list and status.
func (sp *StarredPicker) Reset() {
	sp.list.Clear()
	sp.entries = nil
	sp.status.SetText("")
}

// SetStarred populates the list with starred messages.
func (sp *StarredPicker) SetStarred(entries []StarredEntry) {
	sp.entries = entries
	sp.list.Clear()
	for _, e := range entries {
		timeStr := e.Timestamp
		if t := parseSlackTimestamp(e.Timestamp); !t.IsZero() {
			timeStr = t.Format(time.DateTime)
		}
		channelLabel := e.ChannelName
		if channelLabel == "" {
			channelLabel = e.ChannelID
		}
		main := fmt.Sprintf("#%s  @%s  %s", channelLabel, e.UserName, timeStr)
		secondary := truncateText(e.Text, 70)
		sp.list.AddItem(main, secondary, 0, nil)
	}
	if sp.list.GetItemCount() > 0 {
		sp.list.SetCurrentItem(0)
	}
}

// SetStatus updates the status text at the bottom of the picker.
func (sp *StarredPicker) SetStatus(text string) {
	sp.status.SetText(" " + text)
}

// handleInput processes keybindings for the starred picker.
func (sp *StarredPicker) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch {
	case name == sp.cfg.Keybinds.StarredPicker.Close:
		sp.close()
		return nil

	case name == sp.cfg.Keybinds.StarredPicker.Select:
		sp.selectCurrent()
		return nil

	case name == sp.cfg.Keybinds.StarredPicker.Unstar:
		sp.unstarCurrent()
		return nil

	case name == sp.cfg.Keybinds.StarredPicker.Up || event.Key() == tcell.KeyUp:
		cur := sp.list.GetCurrentItem()
		if cur > 0 {
			sp.list.SetCurrentItem(cur - 1)
		}
		return nil

	case name == sp.cfg.Keybinds.StarredPicker.Down || event.Key() == tcell.KeyDown:
		cur := sp.list.GetCurrentItem()
		if cur < sp.list.GetItemCount()-1 {
			sp.list.SetCurrentItem(cur + 1)
		}
		return nil

	case name == sp.cfg.Keybinds.StarredItems:
		// Toggle: pressing the keybind again closes the picker.
		sp.close()
		return nil
	}

	return event
}

// selectCurrent selects the currently highlighted starred message.
func (sp *StarredPicker) selectCurrent() {
	cur := sp.list.GetCurrentItem()
	if cur < 0 || cur >= len(sp.entries) {
		return
	}

	entry := sp.entries[cur]
	if sp.onSelect != nil {
		sp.onSelect(entry.ChannelID, entry.Timestamp)
	}
	sp.close()
}

// unstarCurrent removes the star from the currently highlighted message.
func (sp *StarredPicker) unstarCurrent() {
	cur := sp.list.GetCurrentItem()
	if cur < 0 || cur >= len(sp.entries) {
		return
	}

	entry := sp.entries[cur]
	if sp.onUnstar != nil {
		sp.onUnstar(entry.ChannelID, entry.Timestamp)
	}

	// Remove from local list.
	sp.entries = append(sp.entries[:cur], sp.entries[cur+1:]...)
	sp.list.RemoveItem(cur)
	if cur >= sp.list.GetItemCount() && sp.list.GetItemCount() > 0 {
		sp.list.SetCurrentItem(sp.list.GetItemCount() - 1)
	}

	// Update status.
	count := len(sp.entries)
	if count == 1 {
		sp.SetStatus("1 starred message")
	} else {
		sp.SetStatus(fmt.Sprintf("%d starred messages", count))
	}
}

// close signals the picker should be hidden.
func (sp *StarredPicker) close() {
	if sp.onClose != nil {
		sp.onClose()
	}
}
