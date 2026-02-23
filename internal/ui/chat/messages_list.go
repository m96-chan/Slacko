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

// OnReactionAddRequestFunc is called when the user wants to add a reaction.
type OnReactionAddRequestFunc func(channelID, timestamp string)

// OnReactionRemoveRequestFunc is called when the user wants to remove a reaction.
type OnReactionRemoveRequestFunc func(channelID, timestamp, reaction string)

// OnFileOpenRequestFunc is called when the user wants to open/download a file.
type OnFileOpenRequestFunc func(channelID string, file slack.File)

// OnPinRequestFunc is called when the user wants to pin or unpin a message.
type OnPinRequestFunc func(channelID, timestamp string, pinned bool)

// OnStarRequestFunc is called when the user wants to star or unstar a message.
type OnStarRequestFunc func(channelID, timestamp string, starred bool)

// OnYankFunc is called when the user wants to copy message text.
type OnYankFunc func(text string)

// OnCopyPermalinkFunc is called when the user wants to copy a message permalink.
type OnCopyPermalinkFunc func(channelID, timestamp string)

// OnUserProfileRequestFunc is called when the user wants to view a user's profile.
type OnUserProfileRequestFunc func(userID string)

// MessagesList displays conversation messages with selection and scrolling.
type MessagesList struct {
	*tview.TextView
	cfg                     *config.Config
	mdColors                markdown.MarkdownColors
	messages                []slack.Message // oldest first
	users                   map[string]slack.User
	channelNames            map[string]string // channelID → name
	pinnedSet               map[string]bool   // set of pinned message timestamps
	starredSet              map[string]bool   // set of starred message timestamps
	selectedIdx             int               // -1 = no selection
	channelID               string
	selfUserID              string
	selfTeamID              string
	onReplyRequest          OnReplyRequestFunc
	onEditRequest           OnEditRequestFunc
	onThreadRequest         OnThreadRequestFunc
	onReactionAddRequest    OnReactionAddRequestFunc
	onReactionRemoveRequest OnReactionRemoveRequestFunc
	onFileOpenRequest       OnFileOpenRequestFunc
	onPinRequest            OnPinRequestFunc
	onStarRequest           OnStarRequestFunc
	onYank                  OnYankFunc
	onCopyPermalink         OnCopyPermalinkFunc
	onUserProfileRequest    OnUserProfileRequestFunc
	lastReadTS              string // last-read timestamp for "New messages" separator
}

// NewMessagesList creates a new messages list component.
func NewMessagesList(cfg *config.Config) *MessagesList {
	ml := &MessagesList{
		TextView:     tview.NewTextView(),
		cfg:          cfg,
		selectedIdx:  -1,
		users:        make(map[string]slack.User),
		channelNames: make(map[string]string),
		pinnedSet:    make(map[string]bool),
		starredSet:   make(map[string]bool),
		mdColors:     mdColorsFromTheme(cfg.Theme.Markdown),
	}

	ml.SetDynamicColors(true)
	ml.SetRegions(true)
	ml.SetScrollable(true)
	ml.SetWordWrap(true)
	ml.SetBorder(true).SetTitle(" Messages ")

	ml.SetInputCapture(ml.handleInput)

	return ml
}

// mdColorsFromTheme converts theme markdown styles to MarkdownColors tags.
func mdColorsFromTheme(m config.MarkdownTheme) markdown.MarkdownColors {
	mc := markdown.MarkdownColors{
		UserMention:    m.UserMention.Tag(),
		ChannelMention: m.ChannelMention.Tag(),
		SpecialMention: m.SpecialMention.Tag(),
		Link:           m.Link.Tag(),
		InlineCode:     m.InlineCode.Tag(),
		CodeFence:      m.CodeFence.Tag(),
		BlockquoteMark: m.BlockquoteMark.Tag(),
		BlockquoteText: m.BlockquoteText.Tag(),
	}
	// Fall back to defaults if all tags are empty (zero-value theme).
	if mc.UserMention == "[-]" || mc.UserMention == "[-:-:-]" {
		return markdown.DefaultMarkdownColors()
	}
	return mc
}

// SetSelfUserID sets the current user's ID for edit permission checks.
func (ml *MessagesList) SetSelfUserID(id string) {
	ml.selfUserID = id
}

// SetSelfTeamID sets the authenticated user's team ID for external user detection.
func (ml *MessagesList) SetSelfTeamID(id string) {
	ml.selfTeamID = id
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

// SetOnReactionAddRequest sets the callback for adding reactions.
func (ml *MessagesList) SetOnReactionAddRequest(fn OnReactionAddRequestFunc) {
	ml.onReactionAddRequest = fn
}

// SetOnReactionRemoveRequest sets the callback for removing reactions.
func (ml *MessagesList) SetOnReactionRemoveRequest(fn OnReactionRemoveRequestFunc) {
	ml.onReactionRemoveRequest = fn
}

// SetOnFileOpenRequest sets the callback for opening/downloading files.
func (ml *MessagesList) SetOnFileOpenRequest(fn OnFileOpenRequestFunc) {
	ml.onFileOpenRequest = fn
}

// SetOnPinRequest sets the callback for pin/unpin requests.
func (ml *MessagesList) SetOnPinRequest(fn OnPinRequestFunc) {
	ml.onPinRequest = fn
}

// SetOnStarRequest sets the callback for star/unstar requests.
func (ml *MessagesList) SetOnStarRequest(fn OnStarRequestFunc) {
	ml.onStarRequest = fn
}

// SetOnYank sets the callback for yanking (copying) message text.
func (ml *MessagesList) SetOnYank(fn OnYankFunc) {
	ml.onYank = fn
}

// SetOnCopyPermalink sets the callback for copying a message permalink.
func (ml *MessagesList) SetOnCopyPermalink(fn OnCopyPermalinkFunc) {
	ml.onCopyPermalink = fn
}

// SetOnUserProfileRequest sets the callback for viewing a user's profile.
func (ml *MessagesList) SetOnUserProfileRequest(fn OnUserProfileRequestFunc) {
	ml.onUserProfileRequest = fn
}

// SetPinnedMessages sets the full set of pinned message timestamps for the current channel.
func (ml *MessagesList) SetPinnedMessages(timestamps []string) {
	ml.pinnedSet = make(map[string]bool, len(timestamps))
	for _, ts := range timestamps {
		ml.pinnedSet[ts] = true
	}
	ml.render()
}

// SetPinned updates the pinned state of a single message.
func (ml *MessagesList) SetPinned(timestamp string, pinned bool) {
	if pinned {
		ml.pinnedSet[timestamp] = true
	} else {
		delete(ml.pinnedSet, timestamp)
	}
	ml.render()
}

// SetStarredMessages sets the full set of starred message timestamps for the current channel.
func (ml *MessagesList) SetStarredMessages(timestamps []string) {
	ml.starredSet = make(map[string]bool, len(timestamps))
	for _, ts := range timestamps {
		ml.starredSet[ts] = true
	}
	ml.render()
}

// SetStarred updates the starred state of a single message.
func (ml *MessagesList) SetStarred(timestamp string, starred bool) {
	if starred {
		ml.starredSet[timestamp] = true
	} else {
		delete(ml.starredSet, timestamp)
	}
	ml.render()
}

// SetLastRead sets the last-read timestamp for the "New messages" separator.
func (ml *MessagesList) SetLastRead(ts string) {
	ml.lastReadTS = ts
}

// LatestTimestamp returns the timestamp of the newest message, or empty string.
func (ml *MessagesList) LatestTimestamp() string {
	if len(ml.messages) == 0 {
		return ""
	}
	return ml.messages[len(ml.messages)-1].Timestamp
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
	ml.pinnedSet = make(map[string]bool)
	ml.starredSet = make(map[string]bool)

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
func (ml *MessagesList) AddReaction(channelID, timestamp, reaction, userID string) {
	if channelID != ml.channelID {
		return
	}

	for i := range ml.messages {
		if ml.messages[i].Timestamp == timestamp {
			// Check if reaction already exists.
			for j := range ml.messages[i].Reactions {
				if ml.messages[i].Reactions[j].Name == reaction {
					ml.messages[i].Reactions[j].Count++
					if userID != "" {
						ml.messages[i].Reactions[j].Users = append(
							ml.messages[i].Reactions[j].Users, userID)
					}
					ml.render()
					return
				}
			}
			// New reaction.
			r := slack.ItemReaction{Name: reaction, Count: 1}
			if userID != "" {
				r.Users = []string{userID}
			}
			ml.messages[i].Reactions = append(ml.messages[i].Reactions, r)
			ml.render()
			return
		}
	}
}

// RemoveReaction decrements or removes a reaction on a message.
func (ml *MessagesList) RemoveReaction(channelID, timestamp, reaction, userID string) {
	if channelID != ml.channelID {
		return
	}

	for i := range ml.messages {
		if ml.messages[i].Timestamp == timestamp {
			for j := range ml.messages[i].Reactions {
				if ml.messages[i].Reactions[j].Name == reaction {
					ml.messages[i].Reactions[j].Count--
					if userID != "" {
						users := ml.messages[i].Reactions[j].Users
						for k, u := range users {
							if u == userID {
								ml.messages[i].Reactions[j].Users = append(users[:k], users[k+1:]...)
								break
							}
						}
					}
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

// UpdateUsers updates the users map and re-renders to reflect status changes.
func (ml *MessagesList) UpdateUsers(users map[string]slack.User) {
	ml.users = users
	ml.render()
}

// render rebuilds the full text content from messages.
func (ml *MessagesList) render() {
	var b strings.Builder

	theme := ml.cfg.Theme.MessagesList

	var prevDate string
	var prevUser string
	var prevTime time.Time
	newMsgSeparatorShown := false

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
			b.WriteString(formatDateSeparator(dateStr, ml.cfg.DateSeparator.Character, theme.DateSeparator))
			b.WriteString("\n")
			prevDate = dateStr
			prevUser = ""
		}

		// "New messages" separator — shown once at the first message after lastReadTS.
		if !newMsgSeparatorShown && ml.lastReadTS != "" && msg.Timestamp > ml.lastReadTS {
			b.WriteString(formatNewMessagesSeparator(ml.cfg.DateSeparator.Character, theme.NewMsgSeparator))
			b.WriteString("\n")
			newMsgSeparatorShown = true
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
				fmt.Fprintf(&b, "%s%s%s ", theme.Timestamp.Tag(), timeStr, theme.Timestamp.Reset())
			}
			userName := resolveUserName(msg.User, msg.Username, msg.BotID, ml.users, ml.selfTeamID)
			// Presence icon before author name.
			if ml.cfg.Presence.Enabled {
				if u, ok := ml.users[msg.User]; ok {
					fmt.Fprintf(&b, "%s ", presenceIcon(u.Presence))
				}
			}
			fmt.Fprintf(&b, "%s%s%s", theme.Author.Tag(), tview.Escape(userName), theme.Author.Reset())
			// Status emoji after author name.
			if u, ok := ml.users[msg.User]; ok && u.Profile.StatusEmoji != "" {
				emoji := u.Profile.StatusEmoji
				// Strip colons if present (e.g. ":calendar:" → "calendar").
				emoji = strings.TrimPrefix(emoji, ":")
				emoji = strings.TrimSuffix(emoji, ":")
				if emoji != "" {
					fmt.Fprintf(&b, " %s", markdown.LookupEmoji(emoji))
				}
			}
			b.WriteString("\n")
		}

		// System message subtypes.
		if text := systemMessageText(msg, ml.users); text != "" {
			fmt.Fprintf(&b, "  %s%s%s\n", theme.SystemMessage.Tag(), tview.Escape(text), theme.SystemMessage.Reset())
		} else if msg.Text != "" {
			rendered := markdown.Render(msg.Text, ml.users, ml.channelNames,
				ml.cfg.Markdown.Enabled, ml.cfg.Markdown.SyntaxTheme, ml.mdColors)
			for _, line := range strings.Split(rendered, "\n") {
				fmt.Fprintf(&b, "  %s\n", line)
			}
		}

		// Edited indicator.
		if msg.Edited != nil && msg.Edited.Timestamp != "" {
			fmt.Fprintf(&b, "  %s(edited)%s\n", theme.EditedIndicator.Tag(), theme.EditedIndicator.Reset())
		}

		// Pin indicator.
		if ml.pinnedSet[msg.Timestamp] {
			pinIcon := "\U0001F4CC"
			if ml.cfg.AsciiIcons {
				pinIcon = "[PIN]"
			}
			fmt.Fprintf(&b, "  %s%s pinned%s\n", theme.PinIndicator.Tag(), pinIcon, theme.PinIndicator.Reset())
		}

		// Star indicator.
		if ml.starredSet[msg.Timestamp] {
			starIcon := "\u2B50"
			if ml.cfg.AsciiIcons {
				starIcon = "[STAR]"
			}
			fmt.Fprintf(&b, "  %s%s starred%s\n", theme.PinIndicator.Tag(), starIcon, theme.PinIndicator.Reset())
		}

		// File attachments.
		for _, f := range msg.Files {
			icon := fileIcon(f.Name, ml.cfg.AsciiIcons)
			fmt.Fprintf(&b, "  %s%s %s (%s)%s\n",
				theme.FileAttachment.Tag(), icon, tview.Escape(f.Name), formatFileSize(f.Size), theme.FileAttachment.Reset())
		}

		// Link previews / rich attachments.
		if len(msg.Attachments) > 0 {
			b.WriteString(formatAttachments(msg.Attachments, attachmentStyles{
				Title:  theme.AttachmentTitle,
				Text:   theme.AttachmentText,
				Footer: theme.AttachmentFooter,
			}, ml.cfg.ShowAttachmentLinks))
		}

		// Reactions.
		if len(msg.Reactions) > 0 {
			b.WriteString("  ")
			for j, r := range msg.Reactions {
				if j > 0 {
					b.WriteString("  ")
				}
				emoji := markdown.LookupEmoji(r.Name)
				isSelf := containsStr(r.Users, ml.selfUserID)
				if isSelf {
					fmt.Fprintf(&b, "%s%s %d%s", theme.ReactionSelf.Tag(), emoji, r.Count, theme.ReactionSelf.Reset())
				} else {
					fmt.Fprintf(&b, "%s%s %d%s", theme.ReactionOther.Tag(), emoji, r.Count, theme.ReactionOther.Reset())
				}
			}
			b.WriteString("\n")
		}

		// Thread reply count.
		if msg.ReplyCount > 0 {
			if msg.ReplyCount == 1 {
				fmt.Fprintf(&b, "  %s└─ 1 reply%s\n", theme.Reply.Tag(), theme.Reply.Reset())
			} else {
				fmt.Fprintf(&b, "  %s└─ %d replies%s\n", theme.Reply.Tag(), msg.ReplyCount, theme.Reply.Reset())
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
			userName := resolveUserName(msg.User, msg.Username, msg.BotID, ml.users, ml.selfTeamID)
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

	case ml.cfg.Keybinds.MessagesList.Reactions:
		if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) && ml.onReactionAddRequest != nil {
			msg := ml.messages[ml.selectedIdx]
			ml.onReactionAddRequest(ml.channelID, msg.Timestamp)
			return nil
		}

	case ml.cfg.Keybinds.MessagesList.RemoveReaction:
		if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) && ml.onReactionRemoveRequest != nil {
			msg := ml.messages[ml.selectedIdx]
			// Find the first reaction from the current user and remove it.
			for _, r := range msg.Reactions {
				if containsStr(r.Users, ml.selfUserID) {
					ml.onReactionRemoveRequest(ml.channelID, msg.Timestamp, r.Name)
					return nil
				}
			}
		}

	case ml.cfg.Keybinds.MessagesList.OpenFile:
		if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) && ml.onFileOpenRequest != nil {
			msg := ml.messages[ml.selectedIdx]
			if len(msg.Files) > 0 {
				ml.onFileOpenRequest(ml.channelID, msg.Files[0])
				return nil
			}
		}

	case ml.cfg.Keybinds.MessagesList.Pin:
		if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) && ml.onPinRequest != nil {
			msg := ml.messages[ml.selectedIdx]
			ml.onPinRequest(ml.channelID, msg.Timestamp, !ml.pinnedSet[msg.Timestamp])
			return nil
		}

	case ml.cfg.Keybinds.MessagesList.Star:
		if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) && ml.onStarRequest != nil {
			msg := ml.messages[ml.selectedIdx]
			ml.onStarRequest(ml.channelID, msg.Timestamp, !ml.starredSet[msg.Timestamp])
			return nil
		}

	case ml.cfg.Keybinds.MessagesList.Yank:
		if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) && ml.onYank != nil {
			ml.onYank(ml.messages[ml.selectedIdx].Text)
			return nil
		}

	case ml.cfg.Keybinds.MessagesList.CopyPermalink:
		if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) && ml.onCopyPermalink != nil {
			msg := ml.messages[ml.selectedIdx]
			ml.onCopyPermalink(ml.channelID, msg.Timestamp)
			return nil
		}

	case ml.cfg.Keybinds.MessagesList.UserProfile:
		if ml.selectedIdx >= 0 && ml.selectedIdx < len(ml.messages) && ml.onUserProfileRequest != nil {
			msg := ml.messages[ml.selectedIdx]
			if msg.User != "" {
				ml.onUserProfileRequest(msg.User)
				return nil
			}
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
func formatDateSeparator(date, char string, style config.StyleWrapper) string {
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
	return fmt.Sprintf("%s%s%s%s%s", style.Tag(), side, label, side, style.Reset())
}

// formatNewMessagesSeparator creates a centered "New messages" separator line.
func formatNewMessagesSeparator(char string, style config.StyleWrapper) string {
	if char == "" {
		char = "─"
	}
	label := " New messages "
	sideLen := (50 - len(label)) / 2
	if sideLen < 3 {
		sideLen = 3
	}
	side := strings.Repeat(char, sideLen)
	return fmt.Sprintf("%s%s%s%s%s", style.Tag(), side, label, side, style.Reset())
}

// resolveUserName returns the best display name for a message author.
// If selfTeamID is non-empty and the user belongs to a different team, appends " [ext]".
func resolveUserName(userID, username, botID string, users map[string]slack.User, selfTeamID string) string {
	if userID != "" {
		if u, ok := users[userID]; ok {
			name := u.Profile.DisplayName
			if name == "" {
				name = u.RealName
			}
			if name == "" {
				name = u.Name
			}
			if name == "" {
				name = userID
			}
			if selfTeamID != "" && u.TeamID != "" && u.TeamID != selfTeamID {
				name += " [ext]"
			}
			return name
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
	userName := resolveUserName(msg.User, msg.Username, "", users, "")
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
		return fmt.Sprintf("%s %s", userName, msg.Text)
	default:
		return ""
	}
}

// containsStr checks if a string slice contains a value.
func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// attachmentStyles groups the three theme styles used for rendering attachments.
type attachmentStyles struct {
	Title  config.StyleWrapper
	Text   config.StyleWrapper
	Footer config.StyleWrapper
}

// maxAttachmentTextLen is the maximum length for attachment body text before truncation.
const maxAttachmentTextLen = 300

// formatAttachments renders Slack message attachments (link previews, bot cards, etc.)
// into tview-formatted lines. Each line is indented with two spaces and a vertical bar.
func formatAttachments(attachments []slack.Attachment, styles attachmentStyles, showLinks bool) string {
	if len(attachments) == 0 {
		return ""
	}

	var b strings.Builder
	for _, att := range attachments {
		// Skip empty attachments.
		if att.Title == "" && att.Text == "" && att.Fallback == "" && att.Pretext == "" {
			continue
		}

		// Pretext appears above the attachment block.
		if att.Pretext != "" {
			fmt.Fprintf(&b, "  %s\n", tview.Escape(att.Pretext))
		}

		// Author line.
		if att.AuthorName != "" {
			fmt.Fprintf(&b, "  │ %s%s%s\n", styles.Footer.Tag(), tview.Escape(att.AuthorName), styles.Footer.Reset())
		}

		// Title line.
		if att.Title != "" {
			fmt.Fprintf(&b, "  │ %s%s%s\n", styles.Title.Tag(), tview.Escape(att.Title), styles.Title.Reset())
		}

		// Body text (truncated).
		if att.Text != "" {
			text := att.Text
			if len(text) > maxAttachmentTextLen {
				text = text[:maxAttachmentTextLen] + "…"
			}
			for _, line := range strings.Split(text, "\n") {
				fmt.Fprintf(&b, "  │ %s%s%s\n", styles.Text.Tag(), tview.Escape(line), styles.Text.Reset())
			}
		}

		// Fields (key/value pairs often used by bot attachments).
		for _, field := range att.Fields {
			fmt.Fprintf(&b, "  │ %s%s:%s %s\n",
				styles.Title.Tag(), tview.Escape(field.Title), styles.Title.Reset(),
				tview.Escape(fmt.Sprintf("%v", field.Value)))
		}

		// Image info.
		if att.ImageURL != "" {
			dims := ""
			if att.ImageWidth > 0 && att.ImageHeight > 0 {
				dims = fmt.Sprintf(" (%dx%d)", att.ImageWidth, att.ImageHeight)
			}
			fmt.Fprintf(&b, "  │ %s[image%s]%s\n", styles.Footer.Tag(), dims, styles.Footer.Reset())
		}

		// Footer / service name.
		footer := att.Footer
		if footer == "" {
			footer = att.ServiceName
		}
		if footer == "" && showLinks && att.FromURL != "" {
			footer = att.FromURL
		}
		if footer != "" {
			fmt.Fprintf(&b, "  │ %s%s%s\n", styles.Footer.Tag(), tview.Escape(footer), styles.Footer.Reset())
		}
	}
	return b.String()
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
