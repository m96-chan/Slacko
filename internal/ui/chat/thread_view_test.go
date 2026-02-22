package chat

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/slack-go/slack"

	"github.com/m96-chan/Slacko/internal/config"
)

func newTestThreadView() *ThreadView {
	app := tview.NewApplication()
	cfg := &config.Config{}
	cfg.Keybinds.ThreadView.Up = "Rune[k]"
	cfg.Keybinds.ThreadView.Down = "Rune[j]"
	cfg.Keybinds.ThreadView.Reply = "Rune[r]"
	cfg.Keybinds.ThreadView.Close = "Escape"
	cfg.Keybinds.MessageInput.Send = "Enter"
	cfg.Keybinds.MessageInput.Newline = "Shift+Enter"
	cfg.Keybinds.MessageInput.Cancel = "Escape"
	cfg.Timestamps.Enabled = true
	cfg.Timestamps.Format = "3:04PM"
	return NewThreadView(app, cfg)
}

func makeThreadMsg(user, text, ts, threadTS string) slack.Message {
	msg := slack.Message{}
	msg.User = user
	msg.Text = text
	msg.Timestamp = ts
	msg.ThreadTimestamp = threadTS
	return msg
}

func TestThreadView_InitialState(t *testing.T) {
	tv := newTestThreadView()

	if tv.IsOpen() {
		t.Error("thread should not be open initially")
	}
	if tv.ChannelID() != "" {
		t.Errorf("channelID should be empty, got %q", tv.ChannelID())
	}
	if tv.ThreadTS() != "" {
		t.Errorf("threadTS should be empty, got %q", tv.ThreadTS())
	}
	if tv.IsInputFocused() {
		t.Error("input should not be focused initially")
	}
}

func TestThreadView_SetMessages(t *testing.T) {
	tv := newTestThreadView()

	parent := makeThreadMsg("U1", "parent message", "1000.0", "1000.0")
	reply1 := makeThreadMsg("U2", "first reply", "1001.0", "1000.0")
	reply2 := makeThreadMsg("U3", "second reply", "1002.0", "1000.0")

	users := map[string]slack.User{
		"U1": {ID: "U1", Name: "alice"},
		"U2": {ID: "U2", Name: "bob"},
		"U3": {ID: "U3", Name: "charlie"},
	}

	tv.SetMessages("C123", "1000.0", []slack.Message{parent, reply1, reply2}, users)

	if !tv.IsOpen() {
		t.Error("thread should be open after SetMessages")
	}
	if tv.ChannelID() != "C123" {
		t.Errorf("channelID should be C123, got %q", tv.ChannelID())
	}
	if tv.ThreadTS() != "1000.0" {
		t.Errorf("threadTS should be 1000.0, got %q", tv.ThreadTS())
	}
	if len(tv.messages) != 3 {
		t.Errorf("should have 3 messages, got %d", len(tv.messages))
	}
}

func TestThreadView_AppendReply(t *testing.T) {
	tv := newTestThreadView()

	parent := makeThreadMsg("U1", "parent", "1000.0", "1000.0")
	tv.SetMessages("C123", "1000.0", []slack.Message{parent}, nil)

	reply := makeThreadMsg("U2", "new reply", "1001.0", "1000.0")
	tv.AppendReply(reply)

	if len(tv.messages) != 2 {
		t.Errorf("should have 2 messages after append, got %d", len(tv.messages))
	}
	if tv.messages[1].Text != "new reply" {
		t.Errorf("appended reply text should be 'new reply', got %q", tv.messages[1].Text)
	}
}

func TestThreadView_UpdateReply(t *testing.T) {
	tv := newTestThreadView()

	parent := makeThreadMsg("U1", "parent", "1000.0", "1000.0")
	reply := makeThreadMsg("U2", "original", "1001.0", "1000.0")
	tv.SetMessages("C123", "1000.0", []slack.Message{parent, reply}, nil)

	tv.UpdateReply("1001.0", "updated text")

	if tv.messages[1].Text != "updated text" {
		t.Errorf("reply text should be 'updated text', got %q", tv.messages[1].Text)
	}
	if tv.messages[1].Edited == nil {
		t.Error("reply should have Edited set")
	}
}

func TestThreadView_RemoveReply(t *testing.T) {
	tv := newTestThreadView()

	parent := makeThreadMsg("U1", "parent", "1000.0", "1000.0")
	reply1 := makeThreadMsg("U2", "reply1", "1001.0", "1000.0")
	reply2 := makeThreadMsg("U3", "reply2", "1002.0", "1000.0")
	tv.SetMessages("C123", "1000.0", []slack.Message{parent, reply1, reply2}, nil)

	tv.RemoveReply("1001.0")

	if len(tv.messages) != 2 {
		t.Errorf("should have 2 messages after remove, got %d", len(tv.messages))
	}
	if tv.messages[1].Timestamp != "1002.0" {
		t.Errorf("remaining reply should be 1002.0, got %q", tv.messages[1].Timestamp)
	}
}

func TestThreadView_SendCallsCallback(t *testing.T) {
	tv := newTestThreadView()

	parent := makeThreadMsg("U1", "parent", "1000.0", "1000.0")
	tv.SetMessages("C123", "1000.0", []slack.Message{parent}, nil)

	var gotChannel, gotText, gotThread string
	tv.SetOnSend(func(channelID, text, threadTS string) {
		gotChannel = channelID
		gotText = text
		gotThread = threadTS
	})

	tv.replyInput.SetText("hello thread", false)
	tv.sendReply()

	if gotChannel != "C123" {
		t.Errorf("channel should be C123, got %q", gotChannel)
	}
	if gotText != "hello thread" {
		t.Errorf("text should be 'hello thread', got %q", gotText)
	}
	if gotThread != "1000.0" {
		t.Errorf("threadTS should be 1000.0, got %q", gotThread)
	}
	// Input should be cleared after send.
	if got := tv.replyInput.GetText(); got != "" {
		t.Errorf("reply input should be empty after send, got %q", got)
	}
}

func TestThreadView_SendEmptyIgnored(t *testing.T) {
	tv := newTestThreadView()

	parent := makeThreadMsg("U1", "parent", "1000.0", "1000.0")
	tv.SetMessages("C123", "1000.0", []slack.Message{parent}, nil)

	called := false
	tv.SetOnSend(func(channelID, text, threadTS string) {
		called = true
	})

	tv.sendReply() // empty text

	if called {
		t.Error("send should not be called for empty text")
	}
}

func TestThreadView_CloseCallsCallback(t *testing.T) {
	tv := newTestThreadView()

	parent := makeThreadMsg("U1", "parent", "1000.0", "1000.0")
	tv.SetMessages("C123", "1000.0", []slack.Message{parent}, nil)

	called := false
	tv.SetOnClose(func() {
		called = true
	})

	tv.close()

	if !called {
		t.Error("close callback should have been called")
	}
}

func TestThreadView_Clear(t *testing.T) {
	tv := newTestThreadView()

	parent := makeThreadMsg("U1", "parent", "1000.0", "1000.0")
	tv.SetMessages("C123", "1000.0", []slack.Message{parent}, nil)

	tv.Clear()

	if tv.IsOpen() {
		t.Error("thread should not be open after Clear")
	}
	if tv.ChannelID() != "" {
		t.Errorf("channelID should be empty after Clear, got %q", tv.ChannelID())
	}
	if len(tv.messages) != 0 {
		t.Errorf("messages should be empty after Clear, got %d", len(tv.messages))
	}
}
