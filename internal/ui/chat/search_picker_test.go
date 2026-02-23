package chat

import (
	"strings"
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

func TestFilterHelpText(t *testing.T) {
	text := filterHelpText()
	// Should mention all five supported filter prefixes.
	for _, prefix := range []string{"from:", "in:", "has:", "before:", "after:"} {
		if !strings.Contains(text, prefix) {
			t.Errorf("filterHelpText() missing %q", prefix)
		}
	}
}

func TestGetFilterHint_EmptyQuery(t *testing.T) {
	hint := getFilterHint("")
	// Empty query should return the full help text.
	if hint != filterHelpText() {
		t.Errorf("getFilterHint(\"\") = %q, want filterHelpText()", hint)
	}
}

func TestGetFilterHint_KnownPrefixes(t *testing.T) {
	tests := []struct {
		query    string
		contains string
	}{
		{"from:", "from:@user"},
		{"hello from:", "from:@user"},
		{"in:", "in:#channel"},
		{"has:", "has:reaction"},
		{"before:", "before:YYYY-MM-DD"},
		{"after:", "after:YYYY-MM-DD"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			hint := getFilterHint(tt.query)
			if !strings.Contains(hint, tt.contains) {
				t.Errorf("getFilterHint(%q) = %q, does not contain %q", tt.query, hint, tt.contains)
			}
		})
	}
}

func TestGetFilterHint_NoMatch(t *testing.T) {
	hint := getFilterHint("hello world")
	if hint != "" {
		t.Errorf("getFilterHint(\"hello world\") = %q, want empty", hint)
	}
}

func TestSearchPickerResetShowsFilterHints(t *testing.T) {
	sp := NewSearchPicker(&config.Config{})
	sp.SetStatus("3 results")
	sp.Reset()

	got := sp.status.GetText(false)
	expected := " " + filterHelpText()
	if got != expected {
		t.Errorf("after Reset(), status = %q, want %q", got, expected)
	}
}

func TestSearchPickerOnInputChangedShowsHints(t *testing.T) {
	sp := NewSearchPicker(&config.Config{})

	// When input is cleared, filter hints should appear.
	sp.onInputChanged("")
	got := sp.status.GetText(false)
	expected := " " + filterHelpText()
	if got != expected {
		t.Errorf("after empty input, status = %q, want %q", got, expected)
	}
}

func TestSearchPickerOnInputChangedShowsPrefixHint(t *testing.T) {
	sp := NewSearchPicker(&config.Config{})
	// Prevent the debounce timer from firing during test.
	sp.onSearch = nil

	sp.onInputChanged("from:")
	got := sp.status.GetText(false)
	if !strings.Contains(got, "from:@user") {
		t.Errorf("after typing 'from:', status = %q, want it to contain 'from:@user'", got)
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
