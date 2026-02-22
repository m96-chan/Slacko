package chat

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"
	"github.com/slack-go/slack"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// pickerEntry holds a channel's data for the picker.
type pickerEntry struct {
	channelID   string
	displayText string // display text with icon (e.g. "# general")
	searchText  string // lowercased name for fuzzy matching
}

// ChannelsPicker is a modal popup for fuzzy-searching and selecting channels.
type ChannelsPicker struct {
	*tview.Flex
	cfg      *config.Config
	input    *tview.InputField
	list     *tview.List
	entries  []pickerEntry
	filtered []int // indices into entries for current filter
	onSelect OnChannelSelectedFunc
	onClose  func()
}

// NewChannelsPicker creates a new channel picker component.
func NewChannelsPicker(cfg *config.Config) *ChannelsPicker {
	cp := &ChannelsPicker{
		cfg: cfg,
	}

	cp.input = tview.NewInputField()
	cp.input.SetLabel(" Search: ")
	cp.input.SetFieldBackgroundColor(cfg.Theme.Modal.InputBackground.Background())
	cp.input.SetChangedFunc(cp.onInputChanged)
	cp.input.SetInputCapture(cp.handleInput)

	cp.list = tview.NewList()
	cp.list.SetHighlightFullLine(true)
	cp.list.ShowSecondaryText(false)
	cp.list.SetWrapAround(false)

	cp.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(cp.input, 1, 0, true).
		AddItem(cp.list, 0, 1, false)
	cp.SetBorder(true).SetTitle(" Switch Channel ")

	return cp
}

// SetOnSelect sets the callback for channel selection.
func (cp *ChannelsPicker) SetOnSelect(fn OnChannelSelectedFunc) {
	cp.onSelect = fn
}

// SetOnClose sets the callback for closing the picker.
func (cp *ChannelsPicker) SetOnClose(fn func()) {
	cp.onClose = fn
}

// SetData populates the picker with channels.
func (cp *ChannelsPicker) SetData(channels []slack.Channel, users map[string]slack.User, selfUserID string) {
	cp.entries = make([]pickerEntry, 0, len(channels))

	for _, ch := range channels {
		chType := classifyChannel(ch)
		display := channelDisplayText(ch, chType, users, selfUserID)
		search := pickerSearchText(ch, chType, users)

		cp.entries = append(cp.entries, pickerEntry{
			channelID:   ch.ID,
			displayText: display,
			searchText:  search,
		})
	}
}

// Reset clears the input and shows all channels.
func (cp *ChannelsPicker) Reset() {
	cp.input.SetText("")
	cp.showAll()
}

// handleInput processes keybindings for the picker input field.
func (cp *ChannelsPicker) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch {
	case name == cp.cfg.Keybinds.ChannelsPicker.Close:
		cp.close()
		return nil

	case name == cp.cfg.Keybinds.ChannelsPicker.Select:
		cp.selectCurrent()
		return nil

	case name == cp.cfg.Keybinds.ChannelsPicker.Up || event.Key() == tcell.KeyUp:
		cur := cp.list.GetCurrentItem()
		if cur > 0 {
			cp.list.SetCurrentItem(cur - 1)
		}
		return nil

	case name == cp.cfg.Keybinds.ChannelsPicker.Down || event.Key() == tcell.KeyDown:
		cur := cp.list.GetCurrentItem()
		if cur < cp.list.GetItemCount()-1 {
			cp.list.SetCurrentItem(cur + 1)
		}
		return nil

	case name == cp.cfg.Keybinds.ChannelPicker:
		// Ctrl+K while picker is open → close it.
		cp.close()
		return nil
	}

	return event
}

// onInputChanged filters the list based on the current search text.
func (cp *ChannelsPicker) onInputChanged(text string) {
	if text == "" {
		cp.showAll()
		return
	}

	// Build search targets.
	targets := make([]string, len(cp.entries))
	for i, e := range cp.entries {
		targets[i] = e.searchText
	}

	matches := fuzzy.Find(text, targets)

	cp.filtered = make([]int, len(matches))
	for i, m := range matches {
		cp.filtered[i] = m.Index
	}

	cp.rebuildList()
}

// showAll displays all channels (no filter).
func (cp *ChannelsPicker) showAll() {
	cp.filtered = make([]int, len(cp.entries))
	for i := range cp.entries {
		cp.filtered[i] = i
	}
	cp.rebuildList()
}

// rebuildList updates the tview.List from the filtered entries.
func (cp *ChannelsPicker) rebuildList() {
	cp.list.Clear()
	for _, idx := range cp.filtered {
		e := cp.entries[idx]
		cp.list.AddItem(e.displayText, "", 0, nil)
	}
	if cp.list.GetItemCount() > 0 {
		cp.list.SetCurrentItem(0)
	}
}

// selectCurrent selects the currently highlighted channel.
func (cp *ChannelsPicker) selectCurrent() {
	cur := cp.list.GetCurrentItem()
	if cur < 0 || cur >= len(cp.filtered) {
		return
	}

	entry := cp.entries[cp.filtered[cur]]
	if cp.onSelect != nil {
		cp.onSelect(entry.channelID)
	}
	cp.close()
}

// close signals the picker should be hidden.
func (cp *ChannelsPicker) close() {
	if cp.onClose != nil {
		cp.onClose()
	}
}

// FilteredCount returns the number of currently visible entries.
func (cp *ChannelsPicker) FilteredCount() int {
	return len(cp.filtered)
}

// pickerSearchText returns a lowercased search string for fuzzy matching.
func pickerSearchText(ch slack.Channel, chType ChannelType, users map[string]slack.User) string {
	switch chType {
	case ChannelTypeDM:
		user, ok := users[ch.User]
		if !ok {
			return strings.ToLower(ch.User)
		}
		// Include display name, real name, and username for matching.
		parts := []string{
			strings.ToLower(user.Profile.DisplayName),
			strings.ToLower(user.RealName),
			strings.ToLower(user.Name),
		}
		return strings.Join(parts, " ")

	case ChannelTypeGroupDM:
		parts := []string{strings.ToLower(ch.Name)}
		if ch.Purpose.Value != "" {
			parts = append(parts, strings.ToLower(ch.Purpose.Value))
		}
		return strings.Join(parts, " ")

	default:
		parts := []string{strings.ToLower(ch.Name)}
		if ch.Topic.Value != "" {
			parts = append(parts, strings.ToLower(ch.Topic.Value))
		}
		return strings.Join(parts, " ")
	}
}
