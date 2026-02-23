package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// SearchResultEntry holds a single search result for display.
type SearchResultEntry struct {
	ChannelID   string
	ChannelName string
	UserName    string
	Timestamp   string
	Text        string
}

// SearchPicker is a modal popup for searching Slack messages.
type SearchPicker struct {
	*tview.Flex
	cfg      *config.Config
	input    *tview.InputField
	list     *tview.List
	status   *tview.TextView
	results  []SearchResultEntry
	onSelect func(channelID, timestamp string)
	onSearch func(query string)
	onClose  func()
	debounce *time.Timer
}

// NewSearchPicker creates a new search picker component.
func NewSearchPicker(cfg *config.Config) *SearchPicker {
	sp := &SearchPicker{
		cfg: cfg,
	}

	sp.input = tview.NewInputField()
	sp.input.SetLabel(" Search: ")
	sp.input.SetFieldBackgroundColor(cfg.Theme.Modal.InputBackground.Background())
	sp.input.SetChangedFunc(sp.onInputChanged)
	sp.input.SetInputCapture(sp.handleInput)

	sp.list = tview.NewList()
	sp.list.SetHighlightFullLine(true)
	sp.list.ShowSecondaryText(true)
	sp.list.SetWrapAround(false)
	sp.list.SetSecondaryTextColor(cfg.Theme.Modal.SecondaryText.Foreground())

	sp.status = tview.NewTextView()
	sp.status.SetTextAlign(tview.AlignLeft)
	sp.status.SetDynamicColors(true)

	sp.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(sp.input, 1, 0, true).
		AddItem(sp.list, 0, 1, false).
		AddItem(sp.status, 1, 0, false)
	sp.SetBorder(true).SetTitle(" Search Messages ")

	return sp
}

// SetOnSelect sets the callback for result selection.
func (sp *SearchPicker) SetOnSelect(fn func(channelID, timestamp string)) {
	sp.onSelect = fn
}

// SetOnSearch sets the callback for triggering a search query.
func (sp *SearchPicker) SetOnSearch(fn func(query string)) {
	sp.onSearch = fn
}

// SetOnClose sets the callback for closing the picker.
func (sp *SearchPicker) SetOnClose(fn func()) {
	sp.onClose = fn
}

// Reset clears the input, results, and shows filter hints.
func (sp *SearchPicker) Reset() {
	sp.input.SetText("")
	sp.list.Clear()
	sp.results = nil
	sp.SetStatus(filterHelpText())
}

// SetResults populates the list with search results.
func (sp *SearchPicker) SetResults(results []SearchResultEntry) {
	sp.results = results
	sp.list.Clear()
	for _, r := range results {
		timeStr := r.Timestamp
		if t := parseSlackTimestamp(r.Timestamp); !t.IsZero() {
			timeStr = t.Format(sp.cfg.Timestamps.Format)
		}
		main := fmt.Sprintf("#%s  @%s  %s", r.ChannelName, r.UserName, timeStr)
		secondary := truncateText(r.Text, 70)
		sp.list.AddItem(main, secondary, 0, nil)
	}
	if sp.list.GetItemCount() > 0 {
		sp.list.SetCurrentItem(0)
	}
}

// SetStatus updates the status text at the bottom of the picker.
func (sp *SearchPicker) SetStatus(text string) {
	sp.status.SetText(" " + text)
}

// handleInput processes keybindings for the search input field.
func (sp *SearchPicker) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch {
	case name == sp.cfg.Keybinds.SearchPicker.Close:
		sp.close()
		return nil

	case name == sp.cfg.Keybinds.SearchPicker.Select:
		sp.selectCurrent()
		return nil

	case name == sp.cfg.Keybinds.SearchPicker.Up || event.Key() == tcell.KeyUp:
		cur := sp.list.GetCurrentItem()
		if cur > 0 {
			sp.list.SetCurrentItem(cur - 1)
		}
		return nil

	case name == sp.cfg.Keybinds.SearchPicker.Down || event.Key() == tcell.KeyDown:
		cur := sp.list.GetCurrentItem()
		if cur < sp.list.GetItemCount()-1 {
			sp.list.SetCurrentItem(cur + 1)
		}
		return nil

	case name == sp.cfg.Keybinds.Search:
		// Ctrl+S while picker is open -> close it.
		sp.close()
		return nil
	}

	return event
}

// onInputChanged debounces input changes and triggers the search callback.
func (sp *SearchPicker) onInputChanged(text string) {
	if sp.debounce != nil {
		sp.debounce.Stop()
	}

	if text == "" {
		sp.list.Clear()
		sp.results = nil
		sp.SetStatus(filterHelpText())
		return
	}

	// Show contextual filter hint if the user is typing a known prefix.
	if hint := getFilterHint(text); hint != "" {
		sp.SetStatus(hint)
	} else {
		sp.SetStatus("Searching...")
	}

	sp.debounce = time.AfterFunc(300*time.Millisecond, func() {
		if sp.onSearch != nil {
			sp.onSearch(text)
		}
	})
}

// selectCurrent selects the currently highlighted result.
func (sp *SearchPicker) selectCurrent() {
	cur := sp.list.GetCurrentItem()
	if cur < 0 || cur >= len(sp.results) {
		return
	}

	entry := sp.results[cur]
	if sp.onSelect != nil {
		sp.onSelect(entry.ChannelID, entry.Timestamp)
	}
	sp.close()
}

// close signals the picker should be hidden.
func (sp *SearchPicker) close() {
	if sp.debounce != nil {
		sp.debounce.Stop()
	}
	if sp.onClose != nil {
		sp.onClose()
	}
}

// filterHelpText returns the full help text listing available search filters.
func filterHelpText() string {
	return "Filters: from:@user  in:#channel  has:reaction/link/pin  before:YYYY-MM-DD  after:YYYY-MM-DD"
}

// filterHints maps each recognized filter prefix to its hint text.
var filterHints = map[string]string{
	"from:":   "from:@user - search messages from a specific user",
	"in:":     "in:#channel - search within a specific channel",
	"has:":    "has:reaction / has:link / has:pin - filter by attachment type",
	"before:": "before:YYYY-MM-DD - messages before a date",
	"after:":  "after:YYYY-MM-DD - messages after a date",
}

// getFilterHint returns a contextual hint for the current query.
// If the query is empty it returns the full help text.
// If the query ends with a known filter prefix it returns the hint for that prefix.
// Otherwise it returns an empty string.
func getFilterHint(query string) string {
	if query == "" {
		return filterHelpText()
	}

	lower := strings.ToLower(query)
	for prefix, hint := range filterHints {
		if strings.HasSuffix(lower, prefix) {
			return hint
		}
	}

	return ""
}

// truncateText shortens text to maxLen characters, adding ellipsis if needed.
func truncateText(text string, maxLen int) string {
	// Replace newlines with spaces for single-line display.
	text = strings.ReplaceAll(text, "\n", " ")
	if len(text) > maxLen {
		return text[:maxLen-1] + "…"
	}
	return text
}
