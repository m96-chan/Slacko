package chat

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/slack-go/slack"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/markdown"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

const messageGroupingWindow = 5 * time.Minute

// OnReplyRequestFunc is called when the user requests to reply to a message.
type OnReplyRequestFunc func(channelID, threadTS, userName string)

// OnEditRequestFunc is called when the user requests to edit a message.
type OnEditRequestFunc func(channelID, timestamp, text string)

// MessagesList displays conversation messages with selection and scrolling.
type MessagesList struct {
	*tview.TextView
	cfg              *config.Config
	messages         []slack.Message          // oldest first
	users            map[string]slack.User
	channelNames     map[string]string        // channelID → name
	selectedIdx      int                      // -1 = no selection
	channelID        string
	selfUserID       string
	onReplyRequest   OnReplyRequestFunc
	onEditRequest    OnEditRequestFunc
	onThreadRequest  OnThreadRequestFunc
}

// NewMessagesList creates a new messages list component.
func NewMessagesList(cfg *config.Config) *MessagesList {
	ml := &MessagesList{
		TextView:     tview.NewTextView(),
		cfg:          cfg,
		selectedIdx:  -1,
		users:        make(map[string]slack.User),
		channelNames: make(map[string]string),
	}

	ml.SetDynamicColors(true)
	ml.SetRegions(true)
	ml.SetScrollable(true)
	ml.SetWordWrap(true)
	ml.SetBorder(true).SetTitle(" Messages ")

	ml.SetInputCapture(ml.handleInput)

	return ml
}

// SetSelfUserID sets the current user's ID for edit permission checks.
func (ml *MessagesList) SetSelfUserID(id string) {
	ml.selfUserID = id
}

// SetChannelNames sets the channel ID → name map for mention rendering.
func (ml *MessagesList) SetChannelNames(names map[string]string) {
	ml.channelNames = names
}

// SetOnReplyRequest sets the callback for reply requests.
func (ml *MessagesList) SetOnReplyRequest(fn OnReplyRequestFunc) {
	ml.onReplyRequest = fn
}

// SetOnEditRequest sets the callback for edit requests.
func (ml *MessagesList) SetOnEditRequest(fn OnEditRequestFunc) {
	ml.onEditRequest = fn
}

// SetOnThreadRequest sets the callback for thread open requests.
func (ml *MessagesList) SetOnThreadRequest(fn OnThreadRequestFunc) {
	ml.onThreadRequest = fn
}

// IncrementReplyCount increments the reply count on a parent message.
func (ml *MessagesList) IncrementReplyCount(channelID, threadTS string) {
	if channelID != ml.channelID {
		return
	}

	for i := range ml.messages {
		if ml.messages[i].Timestamp == threadTS {
			ml.messages[i].ReplyCount++
			ml.render()
			return
		}
	}
}

// SetMessages replaces the message list and renders.
func (ml *MessagesList) SetMessages(channelID string, messages []slack.Message, users map[string]slack.User) {
	ml.channelID = channelID
	ml.users = users
	ml.selectedIdx = -1

	// History returns newest-first; reverse to oldest-first.
	ml.messages = make([]slack.Message, len(messages))
	for i, msg := range messages {
		ml.messages[len(messages)-1-i] = msg
	}

	ml.render()
	ml.ScrollToEnd()
}

// AppendMessage adds a new message to the bottom.
func (ml *MessagesList) AppendMessage(channelID string, msg slack.Message) {
	if channelID != ml.channelID {
		return
	}

	ml.messages = append(ml.messages, msg)
	ml.render()

	// Auto-scroll if no selection active or at the bottom.
	if ml.selectedIdx < 0 {
		ml.ScrollToEnd()
	}
}

// UpdateMessage updates a message's text in place.
func (ml *MessagesList) UpdateMessage(channelID, timestamp, newText string) {
	if channelID != ml.channelID {
		return
	}

	for i := range ml.messages {
		if ml.messages[i].Timestamp == timestamp {
			ml.messages[i].Text = newText
			if ml.messages[i].Edited == nil {
				ml.messages[i].Edited = &slack.Edited{}
			}
			ml.messages[i].Edited.Timestamp = timestamp
			ml.render()
			return
		}
	}
}

// RemoveMessage removes a message by timestamp.
func (ml *MessagesList) RemoveMessage(channelID, timestamp string) {
	if channelID != ml.channelID {
		return
	}

	for i := range ml.messages {
		if ml.messages[i].Timestamp == timestamp {
			ml.messages = append(ml.messages[:i], ml.messages[i+1:]...)
			// Adjust selection.
			if ml.selectedIdx >= len(ml.messages) {
				ml.selectedIdx = len(ml.messages) - 1
			}
			ml.render()
			return
		}
	}
}

// AddReaction adds or increments a reaction on a message.
func (ml *MessagesList) AddReaction(channelID, timestamp, reaction string) {
	if channelID != ml.channelID {
		return
	}

	for i := range ml.messages {
		if ml.messages[i].Timestamp == timestamp {
			// Check if reaction already exists.
			for j := range ml.messages[i].Reactions {
				if ml.messages[i].Reactions[j].Name == reaction {
					ml.messages[i].Reactions[j].Count++
					ml.render()
					return
				}
			}
			// New reaction.
			ml.messages[i].Reactions = append(ml.messages[i].Reactions, slack.ItemReaction{
				Name:  reaction,
				Count: 1,
			})
			ml.render()
			return
		}
	}
}

// RemoveReaction decrements or removes a reaction on a message.
func (ml *MessagesList) RemoveReaction(channelID, timestamp, reaction string) {
	if channelID != ml.channelID {
		return
	}

	for i := range ml.messages {
		if ml.messages[i].Timestamp == timestamp {
			for j := range ml.messages[i].Reactions {
				if ml.messages[i].Reactions[j].Name == reaction {
					ml.messages[i].Reactions[j].Count--
					if ml.messages[i].Reactions[j].Count <= 0 {
						ml.messages[i].Reactions = append(
							ml.messages[i].Reactions[:j],
							ml.messages[i].Reactions[j+1:]...,
						)
					}
					ml.render()
					return
				}
			}
			return
		}
	}
}

// render rebuilds the full text content from messages.
func (ml *MessagesList) render() {
	var b strings.Builder

	var prevDate string
	var prevUser string
	var prevTime time.Time

	for i, msg := range ml.messages {
		// Skip thread replies that aren't the parent message.
		if msg.ThreadTimestamp != "" && msg.ThreadTimestamp != msg.Timestamp {
			continue
		}

		t := parseSlackTimestamp(msg.Timestamp)
		dateStr := t.Format("January 2, 2006")

		// Date separator.
		if ml.cfg.DateSeparator.Enabled && dateStr != prevDate {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(formatDateSeparator(dateStr, ml.cfg.DateSeparator.Character))
			b.WriteString("\n")
			prevDate = dateStr
			prevUser = ""
		}

		// Region start.
		fmt.Fprintf(&b, `["%s"]`, tview.Escape(msg.Timestamp))

		// Message grouping: skip header if same user within window.
		grouped := msg.User != "" && msg.User == prevUser &&
			t.Sub(prevTime) < messageGroupingWindow &&
			msg.SubType == ""

		if !grouped {
			// Timestamp + author line.
			if ml.cfg.Timestamps.Enabled {
				timeStr := t.Format(ml.cfg.Timestamps.Format)
				fmt.Fprintf(&b, "[gray]%s[-] ", timeStr)
			}
			userName := resolveUserName(msg.User, msg.Username, msg.BotID, ml.users)
			fmt.Fprintf(&b, "[green::b]%s[-::-]\n", tview.Escape(userName))
		}

		// System message subtypes.
		if text := systemMessageText(msg, ml.users); text != "" {
			fmt.Fprintf(&b, "  [gray::d]%s[-::-]\n", tview.Escape(text))
		} else if msg.Text != "" {
			rendered := markdown.Render(msg.Text, ml.users, ml.channelNames,
				ml.cfg.Markdown.Enabled, ml.cfg.Markdown.SyntaxTheme)
			for _, line := range strings.Split(rendered, "\n") {
				fmt.Fprintf(&b, "  %s\n", line)
			}
		}

		// Edited indicator.
		if msg.Edited != nil && msg.Edited.Timestamp != "" {
			b.WriteString("  [gray::d](edited)[-::-]\n")
		}

		// File attachments.
		for _, f := range msg.Files {
			fmt.Fprintf(&b, "  [blue]📎 %s (%s)[-]\n",
				tview.Escape(f.Name), formatFileSize(f.Size))
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

		// Thread reply count.
		if msg.ReplyCount > 0 {
			if msg.ReplyCount == 1 {
				b.WriteString("  [cyan]└─ 1 reply[-]\n")
			} else {
				fmt.Fprintf(&b, "  [cyan]└─ %d replies[-]\n", msg.ReplyCount)
			}
		}

		// Region end.
		b.WriteString(`[""]`)

		prevUser = msg.User
		prevTime = t
	}

	ml.SetText(b.String())

	// Apply selection highlight.
	if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) {
		ml.Highlight(ml.messages[ml.selectedIdx].Timestamp)
		ml.ScrollToHighlight()
	} else {
		ml.Highlight()
	}
}

// handleInput processes navigation keys.
func (ml *MessagesList) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch name {
	case ml.cfg.Keybinds.MessagesList.Down:
		ml.selectNext()
		return nil
	case ml.cfg.Keybinds.MessagesList.Up:
		ml.selectPrev()
		return nil
	case ml.cfg.Keybinds.MessagesList.Cancel:
		ml.selectedIdx = -1
		ml.Highlight()
		ml.ScrollToEnd()
		return nil

	case ml.cfg.Keybinds.MessagesList.Reply:
		if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) && ml.onReplyRequest != nil {
			msg := ml.messages[ml.selectedIdx]
			userName := resolveUserName(msg.User, msg.Username, msg.BotID, ml.users)
			// Use the message's own timestamp as the thread parent.
			threadTS := msg.Timestamp
			if msg.ThreadTimestamp != "" {
				threadTS = msg.ThreadTimestamp
			}
			ml.onReplyRequest(ml.channelID, threadTS, userName)
			return nil
		}

	case ml.cfg.Keybinds.MessagesList.Edit:
		if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) && ml.onEditRequest != nil {
			msg := ml.messages[ml.selectedIdx]
			if msg.User == ml.selfUserID {
				ml.onEditRequest(ml.channelID, msg.Timestamp, msg.Text)
				return nil
			}
		}

	case ml.cfg.Keybinds.MessagesList.Thread:
		if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) && ml.onThreadRequest != nil {
			msg := ml.messages[ml.selectedIdx]
			threadTS := msg.Timestamp
			if msg.ThreadTimestamp != "" {
				threadTS = msg.ThreadTimestamp
			}
			ml.onThreadRequest(ml.channelID, threadTS)
			return nil
		}
	}

	return event
}

// selectNext moves selection to the next message.
func (ml *MessagesList) selectNext() {
	if len(ml.messages) == 0 {
		return
	}
	if ml.selectedIdx < 0 {
		// Start selection at the last message.
		ml.selectedIdx = len(ml.messages) - 1
	} else if ml.selectedIdx < len(ml.messages)-1 {
		ml.selectedIdx++
	}
	ml.Highlight(ml.messages[ml.selectedIdx].Timestamp)
	ml.ScrollToHighlight()
}

// selectPrev moves selection to the previous message.
func (ml *MessagesList) selectPrev() {
	if len(ml.messages) == 0 {
		return
	}
	if ml.selectedIdx < 0 {
		ml.selectedIdx = len(ml.messages) - 1
	} else if ml.selectedIdx > 0 {
		ml.selectedIdx--
	}
	ml.Highlight(ml.messages[ml.selectedIdx].Timestamp)
	ml.ScrollToHighlight()
}

// parseSlackTimestamp converts a Slack timestamp (e.g. "1234567890.000100")
// to a time.Time.
func parseSlackTimestamp(ts string) time.Time {
	parts := strings.SplitN(ts, ".", 2)
	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(sec, 0)
}

// formatDateSeparator creates a centered date separator line.
func formatDateSeparator(date, char string) string {
	if char == "" {
		char = "─"
	}
	label := " " + date + " "
	// Target ~50 chars total width.
	sideLen := (50 - len(label)) / 2
	if sideLen < 3 {
		sideLen = 3
	}
	side := strings.Repeat(char, sideLen)
	return fmt.Sprintf("[gray]%s%s%s[-]", side, label, side)
}

// resolveUserName returns the best display name for a message author.
func resolveUserName(userID, username, botID string, users map[string]slack.User) string {
	if userID != "" {
		if u, ok := users[userID]; ok {
			if u.Profile.DisplayName != "" {
				return u.Profile.DisplayName
			}
			if u.RealName != "" {
				return u.RealName
			}
			if u.Name != "" {
				return u.Name
			}
			return userID
		}
	}

	// Bot messages may have username but no user ID.
	if username != "" {
		return username
	}
	if botID != "" {
		return "bot:" + botID
	}
	if userID != "" {
		return userID
	}
	return "unknown"
}

// systemMessageText returns display text for system message subtypes.
// Returns empty string for regular messages.
func systemMessageText(msg slack.Message, users map[string]slack.User) string {
	userName := resolveUserName(msg.User, msg.Username, "", users)
	switch msg.SubType {
	case "channel_join", "group_join":
		return fmt.Sprintf("%s joined the channel", userName)
	case "channel_leave", "group_leave":
		return fmt.Sprintf("%s left the channel", userName)
	case "channel_topic", "group_topic":
		return fmt.Sprintf("%s set the channel topic: %s", userName, msg.Topic)
	case "channel_purpose", "group_purpose":
		return fmt.Sprintf("%s set the channel purpose: %s", userName, msg.Purpose)
	case "channel_name", "group_name":
		return fmt.Sprintf("Channel renamed from %s to %s", msg.OldName, msg.Name)
	case "channel_archive", "group_archive":
		return fmt.Sprintf("%s archived the channel", userName)
	case "channel_unarchive":
		return fmt.Sprintf("%s unarchived the channel", userName)
	case "me_message":
		return "" // Render as regular text
	default:
		return ""
	}
}

// formatFileSize formats a byte count as a human-readable string.
func formatFileSize(size int) string {
	switch {
	case size >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(size)/float64(1<<30))
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(size)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
