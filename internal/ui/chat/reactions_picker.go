package chat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/markdown"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// frequentEmoji lists commonly used emoji shown at the top of the picker.
var frequentEmoji = []string{
	"thumbsup", "thumbsdown", "heart", "smile", "tada",
	"eyes", "fire", "rocket", "white_check_mark", "pray",
	"+1", "-1", "100", "clap", "raised_hands",
	"thinking_face", "joy", "wave", "star", "sparkles",
}

// emojiEntry holds a single emoji for the picker list.
type emojiEntry struct {
	name    string // shortcode (e.g. "thumbsup")
	unicode string // unicode char (e.g. "👍")
}

// OnReactionSelectedFunc is called when the user selects an emoji from the picker.
type OnReactionSelectedFunc func(emojiName string)

// ReactionsPicker is a modal popup for searching and selecting emoji reactions.
type ReactionsPicker struct {
	*tview.Flex
	cfg      *config.Config
	input    *tview.InputField
	list     *tview.List
	entries  []emojiEntry
	filtered []int // indices into entries for current filter
	onSelect OnReactionSelectedFunc
	onClose  func()
}

// NewReactionsPicker creates a new reaction picker component.
func NewReactionsPicker(cfg *config.Config) *ReactionsPicker {
	rp := &ReactionsPicker{
		cfg: cfg,
	}

	rp.input = tview.NewInputField()
	rp.input.SetLabel(" Emoji: ")
	rp.input.SetFieldBackgroundColor(cfg.Theme.Modal.InputBackground.Background())
	rp.input.SetChangedFunc(rp.onInputChanged)
	rp.input.SetInputCapture(rp.handleInput)

	rp.list = tview.NewList()
	rp.list.SetHighlightFullLine(true)
	rp.list.ShowSecondaryText(false)
	rp.list.SetWrapAround(false)

	rp.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(rp.input, 1, 0, true).
		AddItem(rp.list, 0, 1, false)
	rp.SetBorder(true).SetTitle(" Add Reaction ")

	rp.buildEntries()

	return rp
}

// SetOnSelect sets the callback for emoji selection.
func (rp *ReactionsPicker) SetOnSelect(fn OnReactionSelectedFunc) {
	rp.onSelect = fn
}

// SetOnClose sets the callback for closing the picker.
func (rp *ReactionsPicker) SetOnClose(fn func()) {
	rp.onClose = fn
}

// Reset clears the input and shows frequent emoji.
func (rp *ReactionsPicker) Reset() {
	rp.input.SetText("")
	rp.showFrequent()
}

// buildEntries populates the emoji list from the markdown emoji map.
func (rp *ReactionsPicker) buildEntries() {
	emojiMap := markdown.EmojiEntries()

	// Collect all entries sorted by name.
	rp.entries = make([]emojiEntry, 0, len(emojiMap))
	for name, unicode := range emojiMap {
		rp.entries = append(rp.entries, emojiEntry{
			name:    name,
			unicode: unicode,
		})
	}
	sort.Slice(rp.entries, func(i, j int) bool {
		return rp.entries[i].name < rp.entries[j].name
	})
}

// handleInput processes keybindings for the picker input field.
func (rp *ReactionsPicker) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch {
	case name == "Escape":
		rp.close()
		return nil

	case name == "Enter":
		rp.selectCurrent()
		return nil

	case event.Key() == tcell.KeyUp:
		cur := rp.list.GetCurrentItem()
		if cur > 0 {
			rp.list.SetCurrentItem(cur - 1)
		}
		return nil

	case event.Key() == tcell.KeyDown:
		cur := rp.list.GetCurrentItem()
		if cur < rp.list.GetItemCount()-1 {
			rp.list.SetCurrentItem(cur + 1)
		}
		return nil
	}

	return event
}

// onInputChanged filters the emoji list based on search text.
func (rp *ReactionsPicker) onInputChanged(text string) {
	if text == "" {
		rp.showFrequent()
		return
	}

	// Build search targets from all entries.
	targets := make([]string, len(rp.entries))
	for i, e := range rp.entries {
		targets[i] = e.name
	}

	matches := fuzzy.Find(strings.ToLower(text), targets)

	const maxResults = 50
	n := len(matches)
	if n > maxResults {
		n = maxResults
	}
	rp.filtered = make([]int, n)
	for i := 0; i < n; i++ {
		rp.filtered[i] = matches[i].Index
	}

	rp.rebuildList()
}

// showFrequent displays the frequently used emoji.
func (rp *ReactionsPicker) showFrequent() {
	rp.filtered = make([]int, 0, len(frequentEmoji))

	// Build index map for O(1) lookup.
	nameToIdx := make(map[string]int, len(rp.entries))
	for i, e := range rp.entries {
		nameToIdx[e.name] = i
	}

	for _, name := range frequentEmoji {
		if idx, ok := nameToIdx[name]; ok {
			rp.filtered = append(rp.filtered, idx)
		}
	}

	rp.rebuildList()
}

// rebuildList updates the tview.List from filtered entries.
func (rp *ReactionsPicker) rebuildList() {
	rp.list.Clear()
	for _, idx := range rp.filtered {
		e := rp.entries[idx]
		display := fmt.Sprintf("%s  :%s:", e.unicode, e.name)
		rp.list.AddItem(display, "", 0, nil)
	}
	if rp.list.GetItemCount() > 0 {
		rp.list.SetCurrentItem(0)
	}
}

// selectCurrent selects the currently highlighted emoji.
func (rp *ReactionsPicker) selectCurrent() {
	cur := rp.list.GetCurrentItem()
	if cur < 0 || cur >= len(rp.filtered) {
		return
	}

	entry := rp.entries[rp.filtered[cur]]
	if rp.onSelect != nil {
		rp.onSelect(entry.name)
	}
	rp.close()
}

// close signals the picker should be hidden.
func (rp *ReactionsPicker) close() {
	if rp.onClose != nil {
		rp.onClose()
	}
}

// FilteredCount returns the number of currently visible entries.
func (rp *ReactionsPicker) FilteredCount() int {
	return len(rp.filtered)
}
