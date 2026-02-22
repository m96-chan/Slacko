package app

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/m96-chan/Slacko/internal/config"
	"github.com/rivo/tview"
)

func TestNormalizeKeyName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Ctrl-C", "Ctrl+C"},
		{"Ctrl-T", "Ctrl+T"},
		{"Rune[j]", "Rune[j]"},
		{"Enter", "Enter"},
		{"Escape", "Escape"},
		{"Ctrl-Shift-A", "Ctrl+Shift-A"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeKeyName(tt.input)
			if got != tt.want {
				t.Errorf("normalizeKeyName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHandleGlobalKey_QuitConsumed(t *testing.T) {
	cfg := &config.Config{}
	cfg.Keybinds.Quit = "Ctrl+C"

	a := &App{
		Config: cfg,
		tview:  nil, // not used in this path since shutdown is overridden below
	}

	// Override shutdown so we don't need a real tview.Application.
	shutdownCalled := false
	origShutdown := a.shutdown
	_ = origShutdown

	// We can't override a method, so test via handleGlobalKey directly.
	// Create a Ctrl+C event. tcell.KeyCtrlC has Name() "Ctrl-C".
	event := tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModCtrl)

	// handleGlobalKey will call a.shutdown(), which calls a.tview.Stop().
	// We need a minimal tview to avoid nil panic.
	// Instead, set cancel to track that shutdown path was reached.
	a.cancel = func() { shutdownCalled = true }

	// We need a real tview.Application for Stop() to not panic.
	app := newTestApp()
	a.tview = app

	result := a.handleGlobalKey(event)
	if result != nil {
		t.Error("handleGlobalKey should return nil for quit keybind (event consumed)")
	}
	if !shutdownCalled {
		t.Error("shutdown should have called cancel")
	}
}

func TestHandleGlobalKey_NonQuitPassesThrough(t *testing.T) {
	cfg := &config.Config{}
	cfg.Keybinds.Quit = "Ctrl+C"

	app := newTestApp()
	a := &App{
		Config: cfg,
		tview:  app,
	}

	// Create an 'a' key event — should pass through.
	event := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)

	result := a.handleGlobalKey(event)
	if result == nil {
		t.Error("handleGlobalKey should pass through non-quit keybinds")
	}
}

func TestShutdown_CallsCancel(t *testing.T) {
	cancelCalled := false
	app := newTestApp()

	a := &App{
		tview:  app,
		cancel: func() { cancelCalled = true },
	}

	a.shutdown()

	if !cancelCalled {
		t.Error("shutdown should call cancel")
	}
}

func TestShutdown_NilCancel(t *testing.T) {
	app := newTestApp()

	a := &App{
		tview:  app,
		cancel: nil,
	}

	// Should not panic.
	a.shutdown()
}

// newTestApp creates a minimal tview.Application for testing.
func newTestApp() *tview.Application {
	return tview.NewApplication()
}
