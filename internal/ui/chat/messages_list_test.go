package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/slack-go/slack"

	"github.com/m96-chan/Slacko/internal/config"
)

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
	result := formatDateSeparator("January 15, 2026", "─")
	if !strings.Contains(result, "January 15, 2026") {
		t.Errorf("date separator should contain date, got %q", result)
	}
	if !strings.Contains(result, "─") {
		t.Errorf("date separator should contain separator char, got %q", result)
	}
	if !strings.HasPrefix(result, "[gray]") {
		t.Errorf("date separator should have gray color tag, got %q", result)
	}
}

func TestFormatDateSeparator_EmptyChar(t *testing.T) {
	result := formatDateSeparator("January 15, 2026", "")
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
			got := resolveUserName(tt.userID, tt.username, tt.botID, users)
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
			got := resolveUserMentions(tt.text, users)
			if got != tt.want {
				t.Errorf("resolveUserMentions(%q) = %q, want %q", tt.text, got, tt.want)
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
		want    string
	}{
		{"channel_join", "channel_join", "Alice joined the channel"},
		{"channel_leave", "channel_leave", "Alice left the channel"},
		{"regular message", "", ""},
		{"me_message", "me_message", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := slack.Message{}
			msg.User = "U1"
			msg.SubType = tt.subType
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
	cfg := &config.Config{}
	cfg.DateSeparator.Enabled = true
	cfg.DateSeparator.Character = "─"
	cfg.Timestamps.Enabled = true
	cfg.Timestamps.Format = "3:04PM"
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
	cfg := &config.Config{}
	cfg.DateSeparator.Enabled = true
	cfg.DateSeparator.Character = "─"
	cfg.Timestamps.Enabled = true
	cfg.Timestamps.Format = "3:04PM"
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
	cfg := &config.Config{}
	cfg.DateSeparator.Enabled = true
	cfg.DateSeparator.Character = "─"
	cfg.Timestamps.Enabled = true
	cfg.Timestamps.Format = "3:04PM"
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
	cfg := &config.Config{}
	cfg.DateSeparator.Enabled = true
	cfg.DateSeparator.Character = "─"
	cfg.Timestamps.Enabled = true
	cfg.Timestamps.Format = "3:04PM"
	ml := NewMessagesList(cfg)

	messages := []slack.Message{
		makeMsg("1700000001.000000", "U1", "Hello"),
	}
	ml.SetMessages("C1", messages, map[string]slack.User{})

	ml.AddReaction("C1", "1700000001.000000", "thumbsup")
	if len(ml.messages[0].Reactions) != 1 {
		t.Fatalf("expected 1 reaction, got %d", len(ml.messages[0].Reactions))
	}
	if ml.messages[0].Reactions[0].Count != 1 {
		t.Errorf("reaction count should be 1, got %d", ml.messages[0].Reactions[0].Count)
	}

	// Add same reaction again — should increment.
	ml.AddReaction("C1", "1700000001.000000", "thumbsup")
	if ml.messages[0].Reactions[0].Count != 2 {
		t.Errorf("reaction count should be 2, got %d", ml.messages[0].Reactions[0].Count)
	}

	// Remove one.
	ml.RemoveReaction("C1", "1700000001.000000", "thumbsup")
	if ml.messages[0].Reactions[0].Count != 1 {
		t.Errorf("reaction count should be 1 after removal, got %d", ml.messages[0].Reactions[0].Count)
	}

	// Remove last one — should remove the reaction entry.
	ml.RemoveReaction("C1", "1700000001.000000", "thumbsup")
	if len(ml.messages[0].Reactions) != 0 {
		t.Errorf("reaction should be removed entirely, got %d", len(ml.messages[0].Reactions))
	}
}

func TestRender_MessageGrouping(t *testing.T) {
	cfg := &config.Config{}
	cfg.DateSeparator.Enabled = true
	cfg.DateSeparator.Character = "─"
	cfg.Timestamps.Enabled = true
	cfg.Timestamps.Format = "3:04PM"
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

// makeMsg creates a test slack.Message.
func makeMsg(ts, user, text string) slack.Message {
	msg := slack.Message{}
	msg.Timestamp = ts
	msg.User = user
	msg.Text = text
	return msg
}
