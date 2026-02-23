package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/slack-go/slack"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/markdown"
)

// testConfig returns a config with the default theme applied.
func testConfig() *config.Config {
	cfg := &config.Config{}
	cfg.DateSeparator.Enabled = true
	cfg.DateSeparator.Character = "─"
	cfg.Timestamps.Enabled = true
	cfg.Timestamps.Format = "3:04PM"
	cfg.Theme = config.BuiltinTheme("default")
	return cfg
}

func TestParseSlackTimestamp(t *testing.T) {
	tests := []struct {
		ts   string
		want int64
	}{
		{"1700000000.000100", 1700000000},
		{"1234567890.123456", 1234567890},
		{"0.000000", 0},
	}

	for _, tt := range tests {
		got := parseSlackTimestamp(tt.ts)
		if got.Unix() != tt.want {
			t.Errorf("parseSlackTimestamp(%q) = %d, want %d", tt.ts, got.Unix(), tt.want)
		}
	}
}

func TestParseSlackTimestamp_Invalid(t *testing.T) {
	got := parseSlackTimestamp("invalid")
	if !got.IsZero() {
		t.Errorf("expected zero time for invalid timestamp, got %v", got)
	}
}

func TestFormatDateSeparator(t *testing.T) {
	style := config.BuiltinTheme("default").MessagesList.DateSeparator
	result := formatDateSeparator("January 15, 2026", "─", style)
	if !strings.Contains(result, "January 15, 2026") {
		t.Errorf("date separator should contain date, got %q", result)
	}
	if !strings.Contains(result, "─") {
		t.Errorf("date separator should contain separator char, got %q", result)
	}
	if !strings.Contains(result, style.Tag()) {
		t.Errorf("date separator should have theme color tag, got %q", result)
	}
}

func TestFormatDateSeparator_EmptyChar(t *testing.T) {
	style := config.BuiltinTheme("default").MessagesList.DateSeparator
	result := formatDateSeparator("January 15, 2026", "", style)
	if !strings.Contains(result, "─") {
		t.Errorf("empty char should default to ─, got %q", result)
	}
}

func TestResolveUserName(t *testing.T) {
	users := map[string]slack.User{
		"U1": {ID: "U1", RealName: "Alice Smith", Name: "alice", Profile: slack.UserProfile{DisplayName: "Alice"}},
		"U2": {ID: "U2", RealName: "Bob", Name: "bob", Profile: slack.UserProfile{}},
		"U3": {ID: "U3", Name: "charlie", Profile: slack.UserProfile{}},
	}

	tests := []struct {
		name     string
		userID   string
		username string
		botID    string
		want     string
	}{
		{"display name", "U1", "", "", "Alice"},
		{"real name fallback", "U2", "", "", "Bob"},
		{"name fallback", "U3", "", "", "charlie"},
		{"unknown user", "U999", "", "", "U999"},
		{"bot username", "", "webhook-bot", "", "webhook-bot"},
		{"bot ID", "", "", "B123", "bot:B123"},
		{"no info", "", "", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveUserName(tt.userID, tt.username, tt.botID, users, "")
			if got != tt.want {
				t.Errorf("resolveUserName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveUserMentions(t *testing.T) {
	users := map[string]slack.User{
		"U1": {ID: "U1", Name: "alice", Profile: slack.UserProfile{DisplayName: "Alice"}},
	}

	tests := []struct {
		name string
		text string
		want string
	}{
		{"simple mention", "Hello <@U1>!", "Hello @Alice!"},
		{"mention with label", "Hello <@U1|alice>!", "Hello @alice!"},
		{"unknown user", "Hello <@U999>!", "Hello @U999!"},
		{"no mentions", "Hello world!", "Hello world!"},
		{"multiple mentions", "<@U1> and <@U999>", "@Alice and @U999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := markdown.Render(tt.text, users, nil, false, "", markdown.DefaultMarkdownColors())
			if got != tt.want {
				t.Errorf("Render(%q, disabled) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestSystemMessageText(t *testing.T) {
	users := map[string]slack.User{
		"U1": {ID: "U1", Name: "alice", Profile: slack.UserProfile{DisplayName: "Alice"}},
	}

	tests := []struct {
		name    string
		subType string
		text    string
		want    string
	}{
		{"channel_join", "channel_join", "", "Alice joined the channel"},
		{"channel_leave", "channel_leave", "", "Alice left the channel"},
		{"regular message", "", "", ""},
		{"me_message", "me_message", "is thinking...", "Alice is thinking..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := slack.Message{}
			msg.User = "U1"
			msg.SubType = tt.subType
			msg.Text = tt.text
			got := systemMessageText(msg, users)
			if got != tt.want {
				t.Errorf("systemMessageText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size int
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{2621440, "2.5 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatFileSize(tt.size)
			if got != tt.want {
				t.Errorf("formatFileSize(%d) = %q, want %q", tt.size, got, tt.want)
			}
		})
	}
}

func TestSetMessages(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)

	// History returns newest-first; SetMessages should reverse.
	messages := []slack.Message{
		makeMsg("1700000002.000000", "U1", "Second"),
		makeMsg("1700000001.000000", "U1", "First"),
	}

	ml.SetMessages("C1", messages, map[string]slack.User{
		"U1": {ID: "U1", Name: "alice"},
	})

	if len(ml.messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(ml.messages))
	}
	if ml.messages[0].Text != "First" {
		t.Errorf("first message should be 'First', got %q", ml.messages[0].Text)
	}
	if ml.messages[1].Text != "Second" {
		t.Errorf("second message should be 'Second', got %q", ml.messages[1].Text)
	}
	if ml.channelID != "C1" {
		t.Errorf("channelID should be 'C1', got %q", ml.channelID)
	}
}

func TestAppendMessage(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)

	ml.SetMessages("C1", nil, map[string]slack.User{})

	msg := makeMsg("1700000001.000000", "U1", "Hello")
	ml.AppendMessage("C1", msg)

	if len(ml.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(ml.messages))
	}

	// Wrong channel should be ignored.
	ml.AppendMessage("C2", msg)
	if len(ml.messages) != 1 {
		t.Errorf("wrong channel should be ignored, got %d messages", len(ml.messages))
	}
}

func TestRemoveMessage(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)

	messages := []slack.Message{
		makeMsg("1700000002.000000", "U1", "Second"),
		makeMsg("1700000001.000000", "U1", "First"),
	}
	ml.SetMessages("C1", messages, map[string]slack.User{})

	ml.RemoveMessage("C1", "1700000001.000000")
	if len(ml.messages) != 1 {
		t.Fatalf("expected 1 message after removal, got %d", len(ml.messages))
	}
	if ml.messages[0].Text != "Second" {
		t.Errorf("remaining message should be 'Second', got %q", ml.messages[0].Text)
	}
}

func TestAddRemoveReaction(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)

	messages := []slack.Message{
		makeMsg("1700000001.000000", "U1", "Hello"),
	}
	ml.SetMessages("C1", messages, map[string]slack.User{})

	ml.AddReaction("C1", "1700000001.000000", "thumbsup", "U1")
	if len(ml.messages[0].Reactions) != 1 {
		t.Fatalf("expected 1 reaction, got %d", len(ml.messages[0].Reactions))
	}
	if ml.messages[0].Reactions[0].Count != 1 {
		t.Errorf("reaction count should be 1, got %d", ml.messages[0].Reactions[0].Count)
	}

	// Add same reaction again — should increment.
	ml.AddReaction("C1", "1700000001.000000", "thumbsup", "U2")
	if ml.messages[0].Reactions[0].Count != 2 {
		t.Errorf("reaction count should be 2, got %d", ml.messages[0].Reactions[0].Count)
	}

	// Remove one.
	ml.RemoveReaction("C1", "1700000001.000000", "thumbsup", "U2")
	if ml.messages[0].Reactions[0].Count != 1 {
		t.Errorf("reaction count should be 1 after removal, got %d", ml.messages[0].Reactions[0].Count)
	}

	// Remove last one — should remove the reaction entry.
	ml.RemoveReaction("C1", "1700000001.000000", "thumbsup", "U1")
	if len(ml.messages[0].Reactions) != 0 {
		t.Errorf("reaction should be removed entirely, got %d", len(ml.messages[0].Reactions))
	}
}

func TestRender_MessageGrouping(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)

	// Two messages from same user within grouping window.
	now := time.Now().Unix()
	messages := []slack.Message{
		makeMsg("1700000002.000000", "U1", "Second"), // reversed to oldest first
		makeMsg("1700000001.000000", "U1", "First"),
	}
	_ = now

	users := map[string]slack.User{
		"U1": {ID: "U1", Name: "alice"},
	}

	ml.SetMessages("C1", messages, users)

	text := ml.GetText(false)
	// The author name should only appear once since messages are grouped.
	if strings.Count(text, "alice") != 1 {
		t.Errorf("author should appear once due to grouping, appeared %d times in:\n%s",
			strings.Count(text, "alice"), text)
	}
}

func TestFormatNewMessagesSeparator(t *testing.T) {
	style := config.BuiltinTheme("default").MessagesList.NewMsgSeparator
	result := formatNewMessagesSeparator("─", style)
	if !strings.Contains(result, "New messages") {
		t.Errorf("separator should contain 'New messages', got %q", result)
	}
	if !strings.Contains(result, style.Tag()) {
		t.Errorf("separator should have theme color tag, got %q", result)
	}
}

func TestFormatNewMessagesSeparator_EmptyChar(t *testing.T) {
	style := config.BuiltinTheme("default").MessagesList.NewMsgSeparator
	result := formatNewMessagesSeparator("", style)
	if !strings.Contains(result, "─") {
		t.Errorf("empty char should default to ─, got %q", result)
	}
}

func TestLatestTimestamp(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)

	// Empty messages.
	if ts := ml.LatestTimestamp(); ts != "" {
		t.Errorf("LatestTimestamp() = %q, want empty", ts)
	}

	// With messages (SetMessages reverses newest-first to oldest-first).
	messages := []slack.Message{
		makeMsg("1700000003.000000", "U1", "Third"),
		makeMsg("1700000001.000000", "U1", "First"),
	}
	ml.SetMessages("C1", messages, map[string]slack.User{})

	if ts := ml.LatestTimestamp(); ts != "1700000003.000000" {
		t.Errorf("LatestTimestamp() = %q, want 1700000003.000000", ts)
	}
}

func TestSetLastRead(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)

	ml.SetLastRead("1700000001.000000")
	if ml.lastReadTS != "1700000001.000000" {
		t.Errorf("lastReadTS = %q, want 1700000001.000000", ml.lastReadTS)
	}
}

func TestRender_NewMessagesSeparator(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)

	// Set lastRead to be before the second message.
	ml.SetLastRead("1700000001.000000")

	// History returns newest-first.
	messages := []slack.Message{
		makeMsg("1700000002.000000", "U1", "Second"),
		makeMsg("1700000001.000000", "U1", "First"),
	}
	ml.SetMessages("C1", messages, map[string]slack.User{
		"U1": {ID: "U1", Name: "alice"},
	})

	text := ml.GetText(false)
	if !strings.Contains(text, "New messages") {
		t.Errorf("expected 'New messages' separator in rendered text:\n%s", text)
	}
}

func TestRender_NoSeparatorWhenAllRead(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)

	// Set lastRead to the latest message — no separator should appear.
	ml.SetLastRead("1700000002.000000")

	messages := []slack.Message{
		makeMsg("1700000002.000000", "U1", "Second"),
		makeMsg("1700000001.000000", "U1", "First"),
	}
	ml.SetMessages("C1", messages, map[string]slack.User{
		"U1": {ID: "U1", Name: "alice"},
	})

	text := ml.GetText(false)
	if strings.Contains(text, "New messages") {
		t.Errorf("should not show 'New messages' separator when all are read:\n%s", text)
	}
}

func TestSetSelfUserID(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)
	ml.SetSelfUserID("U123")
	if ml.selfUserID != "U123" {
		t.Errorf("selfUserID = %q, want %q", ml.selfUserID, "U123")
	}
}

func TestSetSelfTeamID(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)
	ml.SetSelfTeamID("T123")
	if ml.selfTeamID != "T123" {
		t.Errorf("selfTeamID = %q, want %q", ml.selfTeamID, "T123")
	}
}

func TestSetChannelNames(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)
	names := map[string]string{"C1": "general", "C2": "random"}
	ml.SetChannelNames(names)
	if len(ml.channelNames) != 2 {
		t.Errorf("channelNames len = %d, want 2", len(ml.channelNames))
	}
}

func TestSetPinnedMessages(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)
	ml.SetMessages("C1", nil, map[string]slack.User{})
	ml.SetPinnedMessages([]string{"ts1", "ts2"})
	if !ml.pinnedSet["ts1"] || !ml.pinnedSet["ts2"] {
		t.Errorf("pinnedSet should contain ts1 and ts2")
	}
}

func TestSetPinned(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)
	ml.SetMessages("C1", nil, map[string]slack.User{})

	ml.SetPinned("ts1", true)
	if !ml.pinnedSet["ts1"] {
		t.Error("ts1 should be pinned")
	}

	ml.SetPinned("ts1", false)
	if ml.pinnedSet["ts1"] {
		t.Error("ts1 should be unpinned")
	}
}

func TestSetStarredMessages(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)
	ml.SetMessages("C1", nil, map[string]slack.User{})
	ml.SetStarredMessages([]string{"ts1", "ts3"})
	if !ml.starredSet["ts1"] || !ml.starredSet["ts3"] {
		t.Errorf("starredSet should contain ts1 and ts3")
	}
}

func TestSetStarred(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)
	ml.SetMessages("C1", nil, map[string]slack.User{})

	ml.SetStarred("ts1", true)
	if !ml.starredSet["ts1"] {
		t.Error("ts1 should be starred")
	}

	ml.SetStarred("ts1", false)
	if ml.starredSet["ts1"] {
		t.Error("ts1 should be unstarred")
	}
}

func TestUpdateMessage(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)

	messages := []slack.Message{
		makeMsg("1700000001.000000", "U1", "Original"),
	}
	ml.SetMessages("C1", messages, map[string]slack.User{})

	ml.UpdateMessage("C1", "1700000001.000000", "Updated")
	if ml.messages[0].Text != "Updated" {
		t.Errorf("text = %q, want %q", ml.messages[0].Text, "Updated")
	}
	if ml.messages[0].Edited == nil {
		t.Error("Edited should be set")
	}

	// Wrong channel should be ignored.
	ml.UpdateMessage("C2", "1700000001.000000", "Ignored")
	if ml.messages[0].Text != "Updated" {
		t.Errorf("wrong channel should be ignored, text = %q", ml.messages[0].Text)
	}
}

func TestIncrementReplyCount(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)

	messages := []slack.Message{
		makeMsg("1700000001.000000", "U1", "Parent"),
	}
	ml.SetMessages("C1", messages, map[string]slack.User{})

	ml.IncrementReplyCount("C1", "1700000001.000000")
	if ml.messages[0].ReplyCount != 1 {
		t.Errorf("reply count = %d, want 1", ml.messages[0].ReplyCount)
	}

	ml.IncrementReplyCount("C1", "1700000001.000000")
	if ml.messages[0].ReplyCount != 2 {
		t.Errorf("reply count = %d, want 2", ml.messages[0].ReplyCount)
	}

	// Wrong channel should be ignored.
	ml.IncrementReplyCount("C2", "1700000001.000000")
	if ml.messages[0].ReplyCount != 2 {
		t.Errorf("wrong channel should be ignored, reply count = %d", ml.messages[0].ReplyCount)
	}
}

func TestFormatAttachments(t *testing.T) {
	theme := config.BuiltinTheme("default")
	styles := attachmentStyles{
		Title:  theme.MessagesList.FileAttachment,
		Text:   theme.MessagesList.FileAttachment,
		Footer: theme.MessagesList.FileAttachment,
	}

	t.Run("empty", func(t *testing.T) {
		got := formatAttachments(nil, styles, false)
		if got != "" {
			t.Errorf("expected empty for nil attachments, got %q", got)
		}
	})

	t.Run("with title and text", func(t *testing.T) {
		atts := []slack.Attachment{
			{Title: "Test Article", Text: "Some body text"},
		}
		got := formatAttachments(atts, styles, false)
		if !strings.Contains(got, "Test Article") {
			t.Error("missing title")
		}
		if !strings.Contains(got, "Some body text") {
			t.Error("missing body text")
		}
	})

	t.Run("with author", func(t *testing.T) {
		atts := []slack.Attachment{
			{AuthorName: "John", Title: "Post"},
		}
		got := formatAttachments(atts, styles, false)
		if !strings.Contains(got, "John") {
			t.Error("missing author")
		}
	})

	t.Run("with pretext", func(t *testing.T) {
		atts := []slack.Attachment{
			{Pretext: "Before block", Title: "Card"},
		}
		got := formatAttachments(atts, styles, false)
		if !strings.Contains(got, "Before block") {
			t.Error("missing pretext")
		}
	})

	t.Run("with image", func(t *testing.T) {
		atts := []slack.Attachment{
			{Title: "Photo", ImageURL: "http://example.com/img.png", ImageWidth: 640, ImageHeight: 480},
		}
		got := formatAttachments(atts, styles, false)
		if !strings.Contains(got, "640x480") {
			t.Error("missing image dimensions")
		}
	})

	t.Run("with footer", func(t *testing.T) {
		atts := []slack.Attachment{
			{Title: "Link", Footer: "example.com"},
		}
		got := formatAttachments(atts, styles, false)
		if !strings.Contains(got, "example.com") {
			t.Error("missing footer")
		}
	})

	t.Run("skip empty attachment", func(t *testing.T) {
		atts := []slack.Attachment{
			{},
			{Title: "Real one"},
		}
		got := formatAttachments(atts, styles, false)
		if !strings.Contains(got, "Real one") {
			t.Error("missing non-empty attachment")
		}
	})

	t.Run("truncate long text", func(t *testing.T) {
		longText := strings.Repeat("a", 500)
		atts := []slack.Attachment{
			{Title: "Long", Text: longText},
		}
		got := formatAttachments(atts, styles, false)
		if !strings.Contains(got, "…") {
			t.Error("expected truncation ellipsis")
		}
	})

	t.Run("show links fallback", func(t *testing.T) {
		atts := []slack.Attachment{
			{Title: "Article", FromURL: "http://example.com/article"},
		}
		got := formatAttachments(atts, styles, true)
		if !strings.Contains(got, "example.com/article") {
			t.Error("missing FromURL in footer when showLinks=true")
		}
	})
}

func TestMessageListSetterCallbacks(t *testing.T) {
	cfg := testConfig()
	ml := NewMessagesList(cfg)

	// Test all setter callbacks don't panic.
	ml.SetOnReplyRequest(func(string, string, string) {})
	ml.SetOnEditRequest(func(string, string, string) {})
	ml.SetOnThreadRequest(func(string, string) {})
	ml.SetOnReactionAddRequest(func(string, string) {})
	ml.SetOnReactionRemoveRequest(func(string, string, string) {})
	ml.SetOnFileOpenRequest(func(string, slack.File) {})
	ml.SetOnPinRequest(func(string, string, bool) {})
	ml.SetOnStarRequest(func(string, string, bool) {})
	ml.SetOnYank(func(string) {})
	ml.SetOnCopyPermalink(func(string, string) {})
	ml.SetOnUserProfileRequest(func(string) {})
	ml.SetOnViewReactionsRequest(func(string, string, []slack.ItemReaction) {})

	// Verify they're set.
	if ml.onReplyRequest == nil {
		t.Error("onReplyRequest not set")
	}
	if ml.onEditRequest == nil {
		t.Error("onEditRequest not set")
	}
	if ml.onThreadRequest == nil {
		t.Error("onThreadRequest not set")
	}
	if ml.onViewReactionsRequest == nil {
		t.Error("onViewReactionsRequest not set")
	}
}

// makeMsg creates a test slack.Message.
func makeMsg(ts, user, text string) slack.Message {
	msg := slack.Message{}
	msg.Timestamp = ts
	msg.User = user
	msg.Text = text
	return msg
}
