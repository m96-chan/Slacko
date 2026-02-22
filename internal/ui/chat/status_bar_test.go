package chat

import (
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func TestNewStatusBar(t *testing.T) {
	sb := NewStatusBar(&config.Config{})
	if sb == nil {
		t.Fatal("NewStatusBar returned nil")
	}
}

func TestStatusBarSetConnectionStatus(t *testing.T) {
	sb := NewStatusBar(&config.Config{})
	sb.SetConnectionStatus("Connected")
	got := sb.GetText(false)
	if got != " Connected" {
		t.Errorf("text = %q, want %q", got, " Connected")
	}
}

func TestStatusBarSetTypingIndicator(t *testing.T) {
	sb := NewStatusBar(&config.Config{})
	sb.SetConnectionStatus("Online")
	sb.SetTypingIndicator("alice is typing...")

	got := sb.GetText(false)
	want := " Online  |  alice is typing..."
	if got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
}

func TestStatusBarSetChannelPresence(t *testing.T) {
	sb := NewStatusBar(&config.Config{})
	sb.SetConnectionStatus("Online")
	sb.SetChannelPresence(5, 10)

	got := sb.GetText(false)
	want := " Online  |  5/10 online"
	if got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
}

func TestStatusBarSetChannelPresenceZeroTotal(t *testing.T) {
	sb := NewStatusBar(&config.Config{})
	sb.SetConnectionStatus("Online")
	sb.SetChannelPresence(0, 0)

	got := sb.GetText(false)
	want := " Online"
	if got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
}

func TestStatusBarRenderAll(t *testing.T) {
	sb := NewStatusBar(&config.Config{})
	sb.SetConnectionStatus("OK")
	sb.SetChannelPresence(3, 8)
	sb.SetTypingIndicator("bob is typing...")

	got := sb.GetText(false)
	want := " OK  |  3/8 online  |  bob is typing..."
	if got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
}
