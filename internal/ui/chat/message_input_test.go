package chat

import (
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func newTestInput() *MessageInput {
	cfg := &config.Config{}
	cfg.Keybinds.MessageInput.Send = "Enter"
	cfg.Keybinds.MessageInput.Newline = "Shift+Enter"
	cfg.Keybinds.MessageInput.Cancel = "Escape"
	return NewMessageInput(cfg)
}

func TestMessageInput_InitialState(t *testing.T) {
	mi := newTestInput()

	if mi.Mode() != inputModeNormal {
		t.Errorf("initial mode should be normal, got %d", mi.Mode())
	}
	if mi.channelID != "" {
		t.Errorf("initial channelID should be empty, got %q", mi.channelID)
	}
}

func TestMessageInput_SetChannel(t *testing.T) {
	mi := newTestInput()
	mi.SetChannel("C123")

	if mi.channelID != "C123" {
		t.Errorf("channelID should be C123, got %q", mi.channelID)
	}
}

func TestMessageInput_SetReplyContext(t *testing.T) {
	mi := newTestInput()
	mi.SetChannel("C123")
	mi.SetReplyContext("1234.5678", "alice")

	if mi.Mode() != inputModeReply {
		t.Errorf("mode should be reply, got %d", mi.Mode())
	}
	if mi.threadTS != "1234.5678" {
		t.Errorf("threadTS should be 1234.5678, got %q", mi.threadTS)
	}
}

func TestMessageInput_SetEditMode(t *testing.T) {
	mi := newTestInput()
	mi.SetChannel("C123")
	mi.SetEditMode("1234.5678", "original text")

	if mi.Mode() != inputModeEdit {
		t.Errorf("mode should be edit, got %d", mi.Mode())
	}
	if mi.editTS != "1234.5678" {
		t.Errorf("editTS should be 1234.5678, got %q", mi.editTS)
	}
	if got := mi.GetText(); got != "original text" {
		t.Errorf("text should be 'original text', got %q", got)
	}
}

func TestMessageInput_CancelEditClearsText(t *testing.T) {
	mi := newTestInput()
	mi.SetChannel("C123")
	mi.SetEditMode("1234.5678", "original text")

	mi.cancelMode()

	if mi.Mode() != inputModeNormal {
		t.Errorf("mode should be normal after cancel, got %d", mi.Mode())
	}
	if got := mi.GetText(); got != "" {
		t.Errorf("text should be empty after cancelling edit, got %q", got)
	}
}

func TestMessageInput_CancelReplyKeepsText(t *testing.T) {
	mi := newTestInput()
	mi.SetChannel("C123")
	mi.SetText("some text", false)
	mi.SetReplyContext("1234.5678", "alice")

	mi.cancelMode()

	if mi.Mode() != inputModeNormal {
		t.Errorf("mode should be normal after cancel, got %d", mi.Mode())
	}
	// Text should be preserved when cancelling reply mode.
	if got := mi.GetText(); got != "some text" {
		t.Errorf("text should be preserved after cancelling reply, got %q", got)
	}
}

func TestMessageInput_SendCallsCallback(t *testing.T) {
	mi := newTestInput()
	mi.SetChannel("C123")

	var gotChannel, gotText, gotThread string
	mi.SetOnSend(func(channelID, text, threadTS string) {
		gotChannel = channelID
		gotText = text
		gotThread = threadTS
	})

	mi.SetText("hello world", false)
	mi.send()

	if gotChannel != "C123" {
		t.Errorf("channel should be C123, got %q", gotChannel)
	}
	if gotText != "hello world" {
		t.Errorf("text should be 'hello world', got %q", gotText)
	}
	if gotThread != "" {
		t.Errorf("threadTS should be empty for normal send, got %q", gotThread)
	}
	// Input should be cleared after send.
	if got := mi.GetText(); got != "" {
		t.Errorf("text should be empty after send, got %q", got)
	}
}

func TestMessageInput_SendInReplyMode(t *testing.T) {
	mi := newTestInput()
	mi.SetChannel("C123")
	mi.SetReplyContext("1234.5678", "alice")

	var gotThread string
	mi.SetOnSend(func(channelID, text, threadTS string) {
		gotThread = threadTS
	})

	mi.SetText("reply text", false)
	mi.send()

	if gotThread != "1234.5678" {
		t.Errorf("threadTS should be 1234.5678, got %q", gotThread)
	}
	// Mode should reset after send.
	if mi.Mode() != inputModeNormal {
		t.Errorf("mode should be normal after send, got %d", mi.Mode())
	}
}

func TestMessageInput_SendInEditMode(t *testing.T) {
	mi := newTestInput()
	mi.SetChannel("C123")
	mi.SetEditMode("1234.5678", "original")

	var gotChannel, gotTS, gotText string
	mi.SetOnEdit(func(channelID, timestamp, text string) {
		gotChannel = channelID
		gotTS = timestamp
		gotText = text
	})

	mi.SetText("updated text", true)
	mi.send()

	if gotChannel != "C123" {
		t.Errorf("channel should be C123, got %q", gotChannel)
	}
	if gotTS != "1234.5678" {
		t.Errorf("timestamp should be 1234.5678, got %q", gotTS)
	}
	if gotText != "updated text" {
		t.Errorf("text should be 'updated text', got %q", gotText)
	}
}

func TestMessageInput_SendEmptyIgnored(t *testing.T) {
	mi := newTestInput()
	mi.SetChannel("C123")

	called := false
	mi.SetOnSend(func(channelID, text, threadTS string) {
		called = true
	})

	mi.send() // empty text

	if called {
		t.Error("send should not be called for empty text")
	}
}

func TestMessageInput_SendNoChannelIgnored(t *testing.T) {
	mi := newTestInput()
	// No channel set.

	called := false
	mi.SetOnSend(func(channelID, text, threadTS string) {
		called = true
	})

	mi.SetText("hello", false)
	mi.send()

	if called {
		t.Error("send should not be called without channel")
	}
}

func TestMessageInput_ChannelSwitchCancelsMode(t *testing.T) {
	mi := newTestInput()
	mi.SetChannel("C123")
	mi.SetReplyContext("1234.5678", "alice")

	mi.SetChannel("C456")

	if mi.Mode() != inputModeNormal {
		t.Errorf("mode should be normal after channel switch, got %d", mi.Mode())
	}
	if mi.channelID != "C456" {
		t.Errorf("channelID should be C456, got %q", mi.channelID)
	}
}
