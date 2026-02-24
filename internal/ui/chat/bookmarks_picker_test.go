package chat

import (
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func TestNewBookmarksPicker(t *testing.T) {
	bp := NewBookmarksPicker(&config.Config{})
	if bp == nil {
		t.Fatal("NewBookmarksPicker returned nil")
	}
}

func TestBookmarksPickerSetBookmarks(t *testing.T) {
	bp := NewBookmarksPicker(&config.Config{})
	bp.SetBookmarks([]BookmarkEntry{
		{ID: "B1", Title: "Design Doc", Link: "https://example.com/design", Type: "link"},
		{ID: "B2", Title: "Sprint Board", Link: "https://example.com/board", Type: "link"},
		{ID: "B3", Title: "notes.txt", Link: "https://files.slack.com/notes.txt", Type: "file"},
	})

	if bp.list.GetItemCount() != 3 {
		t.Fatalf("list count = %d, want 3", bp.list.GetItemCount())
	}
	if len(bp.bookmarks) != 3 {
		t.Fatalf("bookmarks len = %d, want 3", len(bp.bookmarks))
	}
}

func TestBookmarksPickerSetBookmarksDisplayText(t *testing.T) {
	bp := NewBookmarksPicker(&config.Config{})
	bp.SetBookmarks([]BookmarkEntry{
		{ID: "B1", Title: "Design Doc", Link: "https://example.com/design", Type: "link"},
		{ID: "B2", Title: "notes.txt", Link: "https://files.slack.com/notes.txt", Type: "file"},
	})

	// Verify link bookmark main text contains the title.
	main, secondary := bp.list.GetItemText(0)
	if main == "" {
		t.Error("link bookmark main text is empty")
	}
	if secondary == "" {
		t.Error("link bookmark secondary text is empty")
	}

	// Verify file bookmark main text contains the title.
	main, secondary = bp.list.GetItemText(1)
	if main == "" {
		t.Error("file bookmark main text is empty")
	}
	if secondary == "" {
		t.Error("file bookmark secondary text is empty")
	}
}

func TestBookmarksPickerEmptyBookmarks(t *testing.T) {
	bp := NewBookmarksPicker(&config.Config{})
	bp.SetBookmarks([]BookmarkEntry{})

	if bp.list.GetItemCount() != 0 {
		t.Fatalf("list count = %d, want 0", bp.list.GetItemCount())
	}
}

func TestBookmarksPickerReset(t *testing.T) {
	bp := NewBookmarksPicker(&config.Config{})
	bp.SetBookmarks([]BookmarkEntry{
		{ID: "B1", Title: "Design Doc", Link: "https://example.com/design", Type: "link"},
	})
	bp.Reset()

	if bp.list.GetItemCount() != 0 {
		t.Errorf("list count = %d, want 0", bp.list.GetItemCount())
	}
	if bp.bookmarks != nil {
		t.Errorf("bookmarks = %v, want nil", bp.bookmarks)
	}
}

func TestBookmarksPickerSetStatus(t *testing.T) {
	bp := NewBookmarksPicker(&config.Config{})
	bp.SetStatus("3 bookmarks")
	got := bp.status.GetText(false)
	if got != " 3 bookmarks" {
		t.Errorf("status = %q, want %q", got, " 3 bookmarks")
	}
}

func TestBookmarksPickerCallbacks(t *testing.T) {
	bp := NewBookmarksPicker(&config.Config{})

	var selectLink string
	closeCalled := false

	bp.SetOnSelect(func(link string) { selectLink = link })
	bp.SetOnClose(func() { closeCalled = true })

	bp.onClose()
	if !closeCalled {
		t.Error("onClose not called")
	}

	bp.onSelect("https://example.com/design")
	if selectLink != "https://example.com/design" {
		t.Errorf("onSelect got %q, want %q", selectLink, "https://example.com/design")
	}
}

func TestBookmarksPickerSelectCurrent(t *testing.T) {
	bp := NewBookmarksPicker(&config.Config{})
	bp.SetBookmarks([]BookmarkEntry{
		{ID: "B1", Title: "Design Doc", Link: "https://example.com/design", Type: "link"},
		{ID: "B2", Title: "Sprint Board", Link: "https://example.com/board", Type: "link"},
	})

	var selectedLink string
	closeCalled := false
	bp.SetOnSelect(func(link string) { selectedLink = link })
	bp.SetOnClose(func() { closeCalled = true })

	// Select the first item (index 0).
	bp.list.SetCurrentItem(0)
	bp.selectCurrent()

	if selectedLink != "https://example.com/design" {
		t.Errorf("selected link = %q, want %q", selectedLink, "https://example.com/design")
	}
	if !closeCalled {
		t.Error("close not called after select")
	}
}

func TestBookmarksPickerSelectCurrentOutOfBounds(t *testing.T) {
	bp := NewBookmarksPicker(&config.Config{})
	// Empty list, should not panic.
	bp.selectCurrent()
}

func TestBookmarksPickerClose(t *testing.T) {
	bp := NewBookmarksPicker(&config.Config{})
	closeCalled := false
	bp.SetOnClose(func() { closeCalled = true })

	bp.close()
	if !closeCalled {
		t.Error("onClose not called")
	}
}

func TestBookmarksPickerCloseNoCallback(t *testing.T) {
	bp := NewBookmarksPicker(&config.Config{})
	// Should not panic even without a callback set.
	bp.close()
}
