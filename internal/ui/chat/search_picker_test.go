package chat

import (
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func TestTruncateText(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world this is long", 10, "hello wor…"},
		{"short", 5, "short"},
		{"newline\nin text", 20, "newline in text"},
		{"", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncateText(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateText(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestNewSearchPicker(t *testing.T) {
	sp := NewSearchPicker(&config.Config{})
	if sp == nil {
		t.Fatal("NewSearchPicker returned nil")
	}
}

func TestSearchPickerSetResults(t *testing.T) {
	sp := NewSearchPicker(&config.Config{})
	sp.SetResults([]SearchResultEntry{
		{ChannelID: "C1", ChannelName: "general", UserName: "alice", Timestamp: "1700000000.000000", Text: "hello"},
		{ChannelID: "C2", ChannelName: "random", UserName: "bob", Timestamp: "1700000001.000000", Text: "world"},
	})

	if sp.list.GetItemCount() != 2 {
		t.Fatalf("list count = %d, want 2", sp.list.GetItemCount())
	}
	if len(sp.results) != 2 {
		t.Fatalf("results len = %d, want 2", len(sp.results))
	}
}

func TestSearchPickerReset(t *testing.T) {
	sp := NewSearchPicker(&config.Config{})
	sp.SetResults([]SearchResultEntry{
		{ChannelID: "C1", ChannelName: "general", UserName: "alice", Text: "hello"},
	})
	sp.Reset()

	if sp.list.GetItemCount() != 0 {
		t.Errorf("list count = %d, want 0", sp.list.GetItemCount())
	}
	if sp.results != nil {
		t.Errorf("results = %v, want nil", sp.results)
	}
}

func TestSearchPickerSetStatus(t *testing.T) {
	sp := NewSearchPicker(&config.Config{})
	sp.SetStatus("3 results")
	got := sp.status.GetText(false)
	if got != " 3 results" {
		t.Errorf("status = %q, want %q", got, " 3 results")
	}
}

func TestSearchPickerCallbacks(t *testing.T) {
	sp := NewSearchPicker(&config.Config{})

	var selectCh, selectTs string
	var searchQuery string
	closeCalled := false

	sp.SetOnSelect(func(ch, ts string) { selectCh = ch; selectTs = ts })
	sp.SetOnSearch(func(q string) { searchQuery = q })
	sp.SetOnClose(func() { closeCalled = true })

	sp.onClose()
	if !closeCalled {
		t.Error("onClose not called")
	}

	sp.onSelect("C1", "123.456")
	if selectCh != "C1" || selectTs != "123.456" {
		t.Errorf("onSelect got (%q, %q), want (%q, %q)", selectCh, selectTs, "C1", "123.456")
	}

	sp.onSearch("test query")
	if searchQuery != "test query" {
		t.Errorf("onSearch got %q, want %q", searchQuery, "test query")
	}
}
