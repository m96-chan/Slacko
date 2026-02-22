package chat

import (
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func TestNewStarredPicker(t *testing.T) {
	sp := NewStarredPicker(&config.Config{})
	if sp == nil {
		t.Fatal("NewStarredPicker returned nil")
	}
}

func TestStarredPickerSetStarred(t *testing.T) {
	sp := NewStarredPicker(&config.Config{})
	sp.SetStarred([]StarredEntry{
		{ChannelID: "C1", ChannelName: "general", Timestamp: "1700000000.000000", UserName: "alice", Text: "hello"},
		{ChannelID: "C2", ChannelName: "", Timestamp: "1700000001.000000", UserName: "bob", Text: "world"},
	})

	if sp.list.GetItemCount() != 2 {
		t.Fatalf("list count = %d, want 2", sp.list.GetItemCount())
	}

	// Second entry should fall back to ChannelID when ChannelName is empty.
	main, _ := sp.list.GetItemText(1)
	if main == "" {
		t.Error("second item text should not be empty")
	}
}

func TestStarredPickerReset(t *testing.T) {
	sp := NewStarredPicker(&config.Config{})
	sp.SetStarred([]StarredEntry{
		{ChannelID: "C1", ChannelName: "general", Timestamp: "123", UserName: "alice", Text: "hello"},
	})
	sp.Reset()

	if sp.list.GetItemCount() != 0 {
		t.Errorf("list count = %d, want 0", sp.list.GetItemCount())
	}
	if sp.entries != nil {
		t.Errorf("entries = %v, want nil", sp.entries)
	}
}

func TestStarredPickerSetStatus(t *testing.T) {
	sp := NewStarredPicker(&config.Config{})
	sp.SetStatus("3 starred messages")
	got := sp.status.GetText(false)
	if got != " 3 starred messages" {
		t.Errorf("status = %q, want %q", got, " 3 starred messages")
	}
}

func TestStarredPickerCallbacks(t *testing.T) {
	sp := NewStarredPicker(&config.Config{})

	var selectCh, selectTs string
	var unstarCh, unstarTs string
	closeCalled := false

	sp.SetOnSelect(func(ch, ts string) { selectCh = ch; selectTs = ts })
	sp.SetOnUnstar(func(ch, ts string) { unstarCh = ch; unstarTs = ts })
	sp.SetOnClose(func() { closeCalled = true })

	sp.onClose()
	if !closeCalled {
		t.Error("onClose not called")
	}

	sp.onSelect("C1", "123.456")
	if selectCh != "C1" || selectTs != "123.456" {
		t.Errorf("onSelect got (%q, %q), want (%q, %q)", selectCh, selectTs, "C1", "123.456")
	}

	sp.onUnstar("C2", "789.012")
	if unstarCh != "C2" || unstarTs != "789.012" {
		t.Errorf("onUnstar got (%q, %q), want (%q, %q)", unstarCh, unstarTs, "C2", "789.012")
	}
}
