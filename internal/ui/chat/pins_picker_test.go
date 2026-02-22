package chat

import (
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func TestNewPinsPicker(t *testing.T) {
	pp := NewPinsPicker(&config.Config{})
	if pp == nil {
		t.Fatal("NewPinsPicker returned nil")
	}
}

func TestPinsPickerSetPins(t *testing.T) {
	pp := NewPinsPicker(&config.Config{})
	pp.SetPins([]PinnedEntry{
		{ChannelID: "C1", Timestamp: "1700000000.000000", UserName: "alice", Text: "hello"},
		{ChannelID: "C1", Timestamp: "1700000001.000000", UserName: "bob", Text: "world"},
	})

	if pp.list.GetItemCount() != 2 {
		t.Fatalf("list count = %d, want 2", pp.list.GetItemCount())
	}
	if len(pp.entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(pp.entries))
	}
}

func TestPinsPickerReset(t *testing.T) {
	pp := NewPinsPicker(&config.Config{})
	pp.SetPins([]PinnedEntry{
		{ChannelID: "C1", Timestamp: "123", UserName: "alice", Text: "hello"},
	})
	pp.Reset()

	if pp.list.GetItemCount() != 0 {
		t.Errorf("list count = %d, want 0", pp.list.GetItemCount())
	}
	if pp.entries != nil {
		t.Errorf("entries = %v, want nil", pp.entries)
	}
}

func TestPinsPickerSetStatus(t *testing.T) {
	pp := NewPinsPicker(&config.Config{})
	pp.SetStatus("2 pinned messages")
	got := pp.status.GetText(false)
	if got != " 2 pinned messages" {
		t.Errorf("status = %q, want %q", got, " 2 pinned messages")
	}
}

func TestPinsPickerCallbacks(t *testing.T) {
	pp := NewPinsPicker(&config.Config{})

	var selectCh, selectTs string
	closeCalled := false

	pp.SetOnSelect(func(ch, ts string) { selectCh = ch; selectTs = ts })
	pp.SetOnClose(func() { closeCalled = true })

	pp.onClose()
	if !closeCalled {
		t.Error("onClose not called")
	}

	pp.onSelect("C1", "123.456")
	if selectCh != "C1" || selectTs != "123.456" {
		t.Errorf("onSelect got (%q, %q), want (%q, %q)", selectCh, selectTs, "C1", "123.456")
	}
}
