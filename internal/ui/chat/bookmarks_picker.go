package chat

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// BookmarkEntry holds a single channel bookmark for display.
type BookmarkEntry struct {
	ID    string
	Title string
	Link  string
	Type  string // "link" or "file"
}

// BookmarksPicker is a modal popup for viewing channel bookmarks.
type BookmarksPicker struct {
	*tview.Flex
	cfg       *config.Config
	list      *tview.List
	status    *tview.TextView
	bookmarks []BookmarkEntry
	onSelect  func(link string) // Opens URL in browser
	onClose   func()
}

// NewBookmarksPicker creates a new channel bookmarks picker component.
func NewBookmarksPicker(cfg *config.Config) *BookmarksPicker {
	bp := &BookmarksPicker{
		cfg: cfg,
	}

	bp.list = tview.NewList()
	bp.list.SetHighlightFullLine(true)
	bp.list.ShowSecondaryText(true)
	bp.list.SetWrapAround(false)
	bp.list.SetSecondaryTextColor(cfg.Theme.Modal.SecondaryText.Foreground())
	bp.list.SetInputCapture(bp.handleInput)

	bp.status = tview.NewTextView()
	bp.status.SetTextAlign(tview.AlignLeft)
	bp.status.SetDynamicColors(true)

	bp.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(bp.list, 0, 1, true).
		AddItem(bp.status, 1, 0, false)
	bp.SetBorder(true).SetTitle(" Channel Bookmarks ")

	return bp
}

// SetOnSelect sets the callback for selecting a bookmark (opens URL).
func (bp *BookmarksPicker) SetOnSelect(fn func(link string)) {
	bp.onSelect = fn
}

// SetOnClose sets the callback for closing the picker.
func (bp *BookmarksPicker) SetOnClose(fn func()) {
	bp.onClose = fn
}

// Reset clears the list and status.
func (bp *BookmarksPicker) Reset() {
	bp.list.Clear()
	bp.bookmarks = nil
	bp.status.SetText("")
}

// SetBookmarks populates the list with channel bookmarks.
func (bp *BookmarksPicker) SetBookmarks(bookmarks []BookmarkEntry) {
	bp.bookmarks = bookmarks
	bp.list.Clear()
	for _, b := range bookmarks {
		icon := "🔗"
		if b.Type == "file" {
			icon = "📎"
		}
		main := fmt.Sprintf("%s %s", icon, b.Title)
		secondary := truncateText(b.Link, 70)
		bp.list.AddItem(main, secondary, 0, nil)
	}
	if bp.list.GetItemCount() > 0 {
		bp.list.SetCurrentItem(0)
	}
}

// SetStatus updates the status text at the bottom of the picker.
func (bp *BookmarksPicker) SetStatus(text string) {
	bp.status.SetText(" " + text)
}

// handleInput processes keybindings for the bookmarks picker.
func (bp *BookmarksPicker) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch {
	case name == bp.cfg.Keybinds.BookmarksPicker.Close:
		bp.close()
		return nil

	case name == bp.cfg.Keybinds.BookmarksPicker.Select:
		bp.selectCurrent()
		return nil

	case name == bp.cfg.Keybinds.BookmarksPicker.Up || event.Key() == tcell.KeyUp:
		cur := bp.list.GetCurrentItem()
		if cur > 0 {
			bp.list.SetCurrentItem(cur - 1)
		}
		return nil

	case name == bp.cfg.Keybinds.BookmarksPicker.Down || event.Key() == tcell.KeyDown:
		cur := bp.list.GetCurrentItem()
		if cur < bp.list.GetItemCount()-1 {
			bp.list.SetCurrentItem(cur + 1)
		}
		return nil
	}

	return event
}

// selectCurrent selects the currently highlighted bookmark and opens its URL.
func (bp *BookmarksPicker) selectCurrent() {
	cur := bp.list.GetCurrentItem()
	if cur < 0 || cur >= len(bp.bookmarks) {
		return
	}

	entry := bp.bookmarks[cur]
	if bp.onSelect != nil {
		bp.onSelect(entry.Link)
	}
	bp.close()
}

// close signals the picker should be hidden.
func (bp *BookmarksPicker) close() {
	if bp.onClose != nil {
		bp.onClose()
	}
}
