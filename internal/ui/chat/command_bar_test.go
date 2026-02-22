package chat

import (
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func TestResolveAlias(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"q", "q"},
		{"quit", "q"},
		{"ws", "workspace"},
		{"workspace", "workspace"},
		{"leave", "leave"},
		{"unknown", "unknown"},
		{"theme", "theme"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveAlias(tt.input)
			if got != tt.want {
				t.Errorf("resolveAlias(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCommandBarAutocomplete(t *testing.T) {
	cb := NewCommandBar(&config.Config{})

	tests := []struct {
		input     string
		wantEmpty bool
		wantHas   string
	}{
		{"", true, ""},
		{"q", false, "q"},
		{"qu", false, "quit"},
		{"the", false, "theme"},
		{"le", false, "leave"},
		{"command with space", true, ""},
		{"xyz", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			matches := cb.autocomplete(tt.input)
			if tt.wantEmpty && len(matches) > 0 {
				t.Errorf("autocomplete(%q) returned %v, want empty", tt.input, matches)
			}
			if !tt.wantEmpty && len(matches) == 0 {
				t.Errorf("autocomplete(%q) returned empty, want matches", tt.input)
			}
			if tt.wantHas != "" {
				found := false
				for _, m := range matches {
					if m == tt.wantHas {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("autocomplete(%q) = %v, missing %q", tt.input, matches, tt.wantHas)
				}
			}
		})
	}
}

func TestCommandBarExecute(t *testing.T) {
	cb := NewCommandBar(&config.Config{})

	var gotCmd, gotArgs string
	cb.SetOnExecute(func(command, args string) {
		gotCmd = command
		gotArgs = args
	})

	// Execute a command with args.
	cb.SetText("join #general")
	cb.execute()

	if gotCmd != "join" {
		t.Errorf("command = %q, want %q", gotCmd, "join")
	}
	if gotArgs != "#general" {
		t.Errorf("args = %q, want %q", gotArgs, "#general")
	}
}

func TestCommandBarExecuteAlias(t *testing.T) {
	cb := NewCommandBar(&config.Config{})

	var gotCmd string
	cb.SetOnExecute(func(command, args string) {
		gotCmd = command
	})

	cb.SetText("quit")
	cb.execute()

	if gotCmd != "q" {
		t.Errorf("command = %q, want %q (alias resolved)", gotCmd, "q")
	}
}

func TestCommandBarExecuteEmpty(t *testing.T) {
	cb := NewCommandBar(&config.Config{})

	closed := false
	cb.SetOnClose(func() {
		closed = true
	})

	cb.SetText("")
	cb.execute()

	if !closed {
		t.Error("expected onClose to be called for empty input")
	}
}

func TestCommandBarHistory(t *testing.T) {
	cb := NewCommandBar(&config.Config{})

	// Execute some commands to build history.
	cb.SetOnExecute(func(string, string) {})
	cb.SetText("join #general")
	cb.execute()
	cb.SetText("leave")
	cb.execute()
	cb.SetText("search foo")
	cb.execute()

	if len(cb.history) != 3 {
		t.Fatalf("history length = %d, want 3", len(cb.history))
	}

	// Navigate backward.
	cb.historyPrev()
	if cb.GetText() != "search foo" {
		t.Errorf("after historyPrev = %q, want %q", cb.GetText(), "search foo")
	}

	cb.historyPrev()
	if cb.GetText() != "leave" {
		t.Errorf("after 2nd historyPrev = %q, want %q", cb.GetText(), "leave")
	}

	cb.historyPrev()
	if cb.GetText() != "join #general" {
		t.Errorf("after 3rd historyPrev = %q, want %q", cb.GetText(), "join #general")
	}

	// Can't go further back.
	cb.historyPrev()
	if cb.GetText() != "join #general" {
		t.Errorf("historyPrev at start = %q, want %q", cb.GetText(), "join #general")
	}

	// Navigate forward.
	cb.historyNext()
	if cb.GetText() != "leave" {
		t.Errorf("after historyNext = %q, want %q", cb.GetText(), "leave")
	}

	cb.historyNext()
	if cb.GetText() != "search foo" {
		t.Errorf("after 2nd historyNext = %q, want %q", cb.GetText(), "search foo")
	}

	// Past the end clears.
	cb.historyNext()
	if cb.GetText() != "" {
		t.Errorf("historyNext past end = %q, want empty", cb.GetText())
	}
}

func TestCommandBarHistoryDedup(t *testing.T) {
	cb := NewCommandBar(&config.Config{})
	cb.SetOnExecute(func(string, string) {})

	cb.SetText("leave")
	cb.execute()
	cb.SetText("leave")
	cb.execute()

	if len(cb.history) != 1 {
		t.Errorf("history length = %d, want 1 (dedup)", len(cb.history))
	}
}

func TestCommandBarReset(t *testing.T) {
	cb := NewCommandBar(&config.Config{})
	cb.SetText("something")
	cb.histIdx = 2

	cb.Reset()
	if cb.GetText() != "" {
		t.Errorf("after Reset, text = %q, want empty", cb.GetText())
	}
	if cb.histIdx != -1 {
		t.Errorf("after Reset, histIdx = %d, want -1", cb.histIdx)
	}
}
