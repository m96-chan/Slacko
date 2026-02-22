package chat

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/slack-go/slack"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/markdown"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// OnThreadReplyFunc is called when the user sends a reply in the thread view.
type OnThreadReplyFunc func(channelID, text, threadTS string)

// OnThreadRequestFunc is called when the user requests to open a thread.
type OnThreadRequestFunc func(channelID, threadTS string)

// ThreadView displays a thread panel with parent message, replies, and reply input.
type ThreadView struct {
	*tview.Flex
	app          *tview.Application
	cfg          *config.Config
	repliesView  *tview.TextView
	replyInput   *tview.TextArea
	channelID    string
	threadTS     string
	messages     []slack.Message // first is parent, rest are replies
	users        map[string]slack.User
	channelNames map[string]string // channelID → name
	selectedIdx  int
	inputFocused bool
	onSend       OnThreadReplyFunc
	onClose      func()
}

// NewThreadView creates a new thread view component.
func NewThreadView(app *tview.Application, cfg *config.Config) *ThreadView {
	tv := &ThreadView{
		app:          app,
		cfg:          cfg,
		selectedIdx:  -1,
		users:        make(map[string]slack.User),
		channelNames: make(map[string]string),
	}

	tv.repliesView = tview.NewTextView()
	tv.repliesView.SetDynamicColors(true)
	tv.repliesView.SetRegions(true)
	tv.repliesView.SetScrollable(true)
	tv.repliesView.SetWordWrap(true)
	tv.repliesView.SetBorder(true).SetTitle(" Thread ")

	tv.replyInput = tview.NewTextArea()
	tv.replyInput.SetBorder(true)
	tv.replyInput.SetPlaceholder("Reply...")

	tv.repliesView.SetInputCapture(tv.handleRepliesInput)
	tv.replyInput.SetInputCapture(tv.handleReplyInput)

	tv.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tv.repliesView, 0, 1, true).
		AddItem(tv.replyInput, 3, 0, false)

	return tv
}

// SetChannelNames sets the channel ID → name map for mention rendering.
func (tv *ThreadView) SetChannelNames(names map[string]string) {
	tv.channelNames = names
}

// SetOnSend sets the callback for sending thread replies.
func (tv *ThreadView) SetOnSend(fn OnThreadReplyFunc) {
	tv.onSend = fn
}

// SetOnClose sets the callback for closing the thread view.
func (tv *ThreadView) SetOnClose(fn func()) {
	tv.onClose = fn
}

// IsOpen returns whether a thread is currently displayed.
func (tv *ThreadView) IsOpen() bool {
	return tv.threadTS != ""
}

// ChannelID returns the channel of the open thread.
func (tv *ThreadView) ChannelID() string {
	return tv.channelID
}

// ThreadTS returns the parent timestamp of the open thread.
func (tv *ThreadView) ThreadTS() string {
	return tv.threadTS
}

// IsInputFocused returns whether the reply input has focus.
func (tv *ThreadView) IsInputFocused() bool {
	return tv.inputFocused
}

// FocusReplies sets focus to the replies view.
func (tv *ThreadView) FocusReplies() {
	tv.inputFocused = false
	tv.app.SetFocus(tv.repliesView)
}

// FocusInput sets focus to the reply input.
func (tv *ThreadView) FocusInput() {
	tv.inputFocused = true
	tv.app.SetFocus(tv.replyInput)
}

// SetMessages sets the thread messages and renders.
func (tv *ThreadView) SetMessages(channelID, threadTS string, messages []slack.Message, users map[string]slack.User) {
	tv.channelID = channelID
	tv.threadTS = threadTS
	tv.messages = messages
	tv.users = users
	tv.selectedIdx = -1
	tv.render()
	tv.repliesView.ScrollToEnd()
}

// AppendReply adds a reply to the thread.
func (tv *ThreadView) AppendReply(msg slack.Message) {
	tv.messages = append(tv.messages, msg)
	tv.render()
	if tv.selectedIdx < 0 {
		tv.repliesView.ScrollToEnd()
	}
}

// UpdateReply updates a reply's text in place.
func (tv *ThreadView) UpdateReply(timestamp, newText string) {
	for i := range tv.messages {
		if tv.messages[i].Timestamp == timestamp {
			tv.messages[i].Text = newText
			if tv.messages[i].Edited == nil {
				tv.messages[i].Edited = &slack.Edited{}
			}
			tv.messages[i].Edited.Timestamp = timestamp
			tv.render()
			return
		}
	}
}

// RemoveReply removes a reply by timestamp.
func (tv *ThreadView) RemoveReply(timestamp string) {
	for i := range tv.messages {
		if tv.messages[i].Timestamp == timestamp {
			tv.messages = append(tv.messages[:i], tv.messages[i+1:]...)
			if tv.selectedIdx >= len(tv.messages) {
				tv.selectedIdx = len(tv.messages) - 1
			}
			tv.render()
			return
		}
	}
}

// Clear resets the thread view state without triggering the close callback.
func (tv *ThreadView) Clear() {
	tv.channelID = ""
	tv.threadTS = ""
	tv.messages = nil
	tv.selectedIdx = -1
	tv.inputFocused = false
	tv.repliesView.SetText("")
	tv.replyInput.SetText("", false)
}

// render rebuilds the full text content from thread messages.
func (tv *ThreadView) render() {
	var b strings.Builder

	for i, msg := range tv.messages {
		// Region start.
		fmt.Fprintf(&b, `["%s"]`, tview.Escape(msg.Timestamp))

		// Author line.
		userName := resolveUserName(msg.User, msg.Username, msg.BotID, tv.users)
		if i == 0 {
			fmt.Fprintf(&b, "[green::b]%s[-::-] [gray](parent)[-]\n", tview.Escape(userName))
		} else {
			t := parseSlackTimestamp(msg.Timestamp)
			if tv.cfg.Timestamps.Enabled {
				timeStr := t.Format(tv.cfg.Timestamps.Format)
				fmt.Fprintf(&b, "[gray]%s[-] ", timeStr)
			}
			fmt.Fprintf(&b, "[green::b]%s[-::-]\n", tview.Escape(userName))
		}

		// Message text.
		if msg.Text != "" {
			rendered := markdown.Render(msg.Text, tv.users, tv.channelNames,
				tv.cfg.Markdown.Enabled, tv.cfg.Markdown.SyntaxTheme)
			for _, line := range strings.Split(rendered, "\n") {
				fmt.Fprintf(&b, "  %s\n", line)
			}
		}

		// Edited indicator.
		if msg.Edited != nil && msg.Edited.Timestamp != "" {
			b.WriteString("  [gray::d](edited)[-::-]\n")
		}

		// Reactions.
		if len(msg.Reactions) > 0 {
			b.WriteString("  ")
			for j, r := range msg.Reactions {
				if j > 0 {
					b.WriteString("  ")
				}
				fmt.Fprintf(&b, "[gray]:%s: %d[-]", tview.Escape(r.Name), r.Count)
			}
			b.WriteString("\n")
		}

		// Region end.
		b.WriteString(`[""]`)

		// Separator after parent message.
		if i == 0 {
			b.WriteString("[gray]────────────────────────────[-]\n")
		}
	}

	tv.repliesView.SetText(b.String())

	// Apply selection highlight.
	if tv.selectedIdx >= 0 && tv.selectedIdx < len(tv.messages) {
		tv.repliesView.Highlight(tv.messages[tv.selectedIdx].Timestamp)
		tv.repliesView.ScrollToHighlight()
	} else {
		tv.repliesView.Highlight()
	}
}

// handleRepliesInput processes keybindings for the replies view.
func (tv *ThreadView) handleRepliesInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch name {
	case tv.cfg.Keybinds.ThreadView.Down:
		tv.selectNext()
		return nil
	case tv.cfg.Keybinds.ThreadView.Up:
		tv.selectPrev()
		return nil
	case tv.cfg.Keybinds.ThreadView.Reply:
		tv.FocusInput()
		return nil
	case tv.cfg.Keybinds.ThreadView.Close:
		tv.close()
		return nil
	}

	return event
}

// handleReplyInput processes keybindings for the reply input.
func (tv *ThreadView) handleReplyInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch name {
	case tv.cfg.Keybinds.MessageInput.Send:
		tv.sendReply()
		return nil
	case tv.cfg.Keybinds.MessageInput.Newline:
		return tcell.NewEventKey(tcell.KeyEnter, '\n', tcell.ModNone)
	case tv.cfg.Keybinds.MessageInput.Cancel:
		tv.FocusReplies()
		return nil
	}

	return event
}

// sendReply dispatches the current reply input text.
func (tv *ThreadView) sendReply() {
	text := strings.TrimSpace(tv.replyInput.GetText())
	if text == "" {
		return
	}
	if tv.channelID == "" || tv.threadTS == "" {
		return
	}

	if tv.onSend != nil {
		tv.onSend(tv.channelID, text, tv.threadTS)
	}

	tv.replyInput.SetText("", false)
	tv.FocusReplies()
}

// close signals that the user wants to close the thread view.
func (tv *ThreadView) close() {
	if tv.onClose != nil {
		tv.onClose()
	}
}

// selectNext moves selection to the next message.
func (tv *ThreadView) selectNext() {
	if len(tv.messages) == 0 {
		return
	}
	if tv.selectedIdx < 0 {
		tv.selectedIdx = len(tv.messages) - 1
	} else if tv.selectedIdx < len(tv.messages)-1 {
		tv.selectedIdx++
	}
	tv.repliesView.Highlight(tv.messages[tv.selectedIdx].Timestamp)
	tv.repliesView.ScrollToHighlight()
}

// selectPrev moves selection to the previous message.
func (tv *ThreadView) selectPrev() {
	if len(tv.messages) == 0 {
		return
	}
	if tv.selectedIdx < 0 {
		tv.selectedIdx = len(tv.messages) - 1
	} else if tv.selectedIdx > 0 {
		tv.selectedIdx--
	}
	tv.repliesView.Highlight(tv.messages[tv.selectedIdx].Timestamp)
	tv.repliesView.ScrollToHighlight()
}
