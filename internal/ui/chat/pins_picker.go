package chat

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// PinnedEntry holds a single pinned message for display.
type PinnedEntry struct {
	ChannelID string
	Timestamp string
	UserName  string
	Text      string
}

// PinsPicker is a modal popup for viewing pinned messages in a channel.
type PinsPicker struct {
	*tview.Flex
	cfg      *config.Config
	list     *tview.List
	status   *tview.TextView
	entries  []PinnedEntry
	onSelect func(channelID, timestamp string)
	onClose  func()
}

// NewPinsPicker creates a new pinned messages picker component.
func NewPinsPicker(cfg *config.Config) *PinsPicker {
	pp := &PinsPicker{
		cfg: cfg,
	}

	pp.list = tview.NewList()
	pp.list.SetHighlightFullLine(true)
	pp.list.ShowSecondaryText(true)
	pp.list.SetWrapAround(false)
	pp.list.SetSecondaryTextColor(cfg.Theme.Modal.SecondaryText.Foreground())
	pp.list.SetInputCapture(pp.handleInput)

	pp.status = tview.NewTextView()
	pp.status.SetTextAlign(tview.AlignLeft)
	pp.status.SetDynamicColors(true)

	pp.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(pp.list, 0, 1, true).
		AddItem(pp.status, 1, 0, false)
	pp.SetBorder(true).SetTitle(" Pinned Messages ")
	pp.SetInputCapture(pp.handleInput)

	return pp
}

// SetOnSelect sets the callback for selecting a pinned message.
func (pp *PinsPicker) SetOnSelect(fn func(channelID, timestamp string)) {
	pp.onSelect = fn
}

// SetOnClose sets the callback for closing the picker.
func (pp *PinsPicker) SetOnClose(fn func()) {
	pp.onClose = fn
}

// Reset clears the list and status.
func (pp *PinsPicker) Reset() {
	pp.list.Clear()
	pp.entries = nil
	pp.status.SetText("")
}

// SetPins populates the list with pinned messages.
func (pp *PinsPicker) SetPins(entries []PinnedEntry) {
	pp.entries = entries
	pp.list.Clear()
	for _, e := range entries {
		timeStr := e.Timestamp
		if t := parseSlackTimestamp(e.Timestamp); !t.IsZero() {
			timeStr = t.Format(time.DateTime)
		}
		main := fmt.Sprintf("@%s  %s", e.UserName, timeStr)
		secondary := truncateText(e.Text, 70)
		pp.list.AddItem(main, secondary, 0, nil)
	}
	if pp.list.GetItemCount() > 0 {
		pp.list.SetCurrentItem(0)
	}
}

// SetStatus updates the status text at the bottom of the picker.
func (pp *PinsPicker) SetStatus(text string) {
	pp.status.SetText(" " + text)
}

// handleInput processes keybindings for the pins picker.
func (pp *PinsPicker) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch {
	case name == pp.cfg.Keybinds.PinsPicker.Close:
		pp.close()
		return nil

	case name == pp.cfg.Keybinds.PinsPicker.Select:
		pp.selectCurrent()
		return nil

	case name == pp.cfg.Keybinds.PinsPicker.Up || event.Key() == tcell.KeyUp:
		cur := pp.list.GetCurrentItem()
		if cur > 0 {
			pp.list.SetCurrentItem(cur - 1)
		}
		return nil

	case name == pp.cfg.Keybinds.PinsPicker.Down || event.Key() == tcell.KeyDown:
		cur := pp.list.GetCurrentItem()
		if cur < pp.list.GetItemCount()-1 {
			pp.list.SetCurrentItem(cur + 1)
		}
		return nil

	case name == pp.cfg.Keybinds.PinnedMessages:
		// Toggle: pressing the keybind again closes the picker.
		pp.close()
		return nil
	}

	return event
}

// selectCurrent selects the currently highlighted pinned message.
func (pp *PinsPicker) selectCurrent() {
	cur := pp.list.GetCurrentItem()
	if cur < 0 || cur >= len(pp.entries) {
		return
	}

	entry := pp.entries[cur]
	if pp.onSelect != nil {
		pp.onSelect(entry.ChannelID, entry.Timestamp)
	}
	pp.close()
}

// close signals the picker should be hidden.
func (pp *PinsPicker) close() {
	if pp.onClose != nil {
		pp.onClose()
	}
}
