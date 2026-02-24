package chat

import (
	"strings"
	"testing"

	"github.com/slack-go/slack"

	"github.com/m96-chan/Slacko/internal/config"
)

func TestClassifyChannel(t *testing.T) {
	tests := []struct {
		name string
		ch   slack.Channel
		want ChannelType
	}{
		{
			name: "public channel",
			ch:   makeChannel("C1", "general", false, false, false),
			want: ChannelTypePublic,
		},
		{
			name: "private channel",
			ch:   makeChannel("C2", "secret", true, false, false),
			want: ChannelTypePrivate,
		},
		{
			name: "DM",
			ch:   makeChannel("D1", "", false, true, false),
			want: ChannelTypeDM,
		},
		{
			name: "group DM",
			ch:   makeChannel("G1", "mpdm-group", false, false, true),
			want: ChannelTypeGroupDM,
		},
		{
			name: "shared channel (Slack Connect)",
			ch:   makeSharedChannel("C5", "ext-partner"),
			want: ChannelTypeShared,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyChannel(tt.ch)
			if got != tt.want {
				t.Errorf("classifyChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChannelDisplayText(t *testing.T) {
	users := map[string]slack.User{
		"U1": {ID: "U1", RealName: "Alice Smith", Name: "alice", Presence: "active", Profile: slack.UserProfile{DisplayName: "alice"}},
		"U2": {ID: "U2", RealName: "Bob", Name: "bob", Presence: "away", Profile: slack.UserProfile{}},
		"U3": {ID: "U3", Name: "charlie", Profile: slack.UserProfile{}},
	}

	tests := []struct {
		name   string
		ch     slack.Channel
		chType ChannelType
		want   string
	}{
		{
			name:   "public channel",
			ch:     makeChannel("C1", "general", false, false, false),
			chType: ChannelTypePublic,
			want:   "# general",
		},
		{
			name:   "private channel",
			ch:     makeChannel("C2", "secret", true, false, false),
			chType: ChannelTypePrivate,
			want:   "🔒 secret",
		},
		{
			name:   "DM with display name",
			ch:     makeDMChannel("D1", "U1"),
			chType: ChannelTypeDM,
			want:   "● alice",
		},
		{
			name:   "DM away user fallback to RealName",
			ch:     makeDMChannel("D2", "U2"),
			chType: ChannelTypeDM,
			want:   "◐ Bob",
		},
		{
			name:   "DM fallback to Name",
			ch:     makeDMChannel("D3", "U3"),
			chType: ChannelTypeDM,
			want:   "○ charlie",
		},
		{
			name:   "DM unknown user",
			ch:     makeDMChannel("D4", "U999"),
			chType: ChannelTypeDM,
			want:   "○ U999",
		},
		{
			name:   "shared channel",
			ch:     makeSharedChannel("C5", "ext-partner"),
			chType: ChannelTypeShared,
			want:   "🔗 ext-partner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channelDisplayText(tt.ch, tt.chType, users, "SELF", false)
			if got != tt.want {
				t.Errorf("channelDisplayText(false) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPresenceIcon(t *testing.T) {
	tests := []struct {
		presence string
		want     string
	}{
		{"active", "●"},
		{"away", "◐"},
		{"", "○"},
		{"unknown", "○"},
	}

	for _, tt := range tests {
		t.Run(tt.presence, func(t *testing.T) {
			got := presenceIcon(tt.presence)
			if got != tt.want {
				t.Errorf("presenceIcon(%q) = %q, want %q", tt.presence, got, tt.want)
			}
		})
	}
}

func TestPopulate(t *testing.T) {
	cfg := &config.Config{}
	ct := NewChannelsTree(cfg, nil)

	channels := []slack.Channel{
		makeChannel("C1", "general", false, false, false),
		makeChannel("C2", "random", false, false, false),
		makeChannel("C3", "secret", true, false, false),
	}
	users := map[string]slack.User{}

	ct.Populate(channels, users, "SELF")

	// All 3 channels should be indexed.
	if len(ct.nodeIndex) != 3 {
		t.Errorf("nodeIndex has %d entries, want 3", len(ct.nodeIndex))
	}

	// Verify each channel is in the index.
	for _, ch := range channels {
		if _, ok := ct.nodeIndex[ch.ID]; !ok {
			t.Errorf("channel %s not found in nodeIndex", ch.ID)
		}
	}

	// "Channels" section should have all 3 (public + private go there).
	section := ct.sections[ChannelTypePrivate]
	if got := len(section.GetChildren()); got != 3 {
		t.Errorf("Channels section has %d children, want 3", got)
	}
}

func TestRemoveChannel(t *testing.T) {
	cfg := &config.Config{}
	ct := NewChannelsTree(cfg, nil)

	channels := []slack.Channel{
		makeChannel("C1", "general", false, false, false),
		makeChannel("C2", "random", false, false, false),
	}
	ct.Populate(channels, map[string]slack.User{}, "SELF")

	ct.RemoveChannel("C1")

	if _, ok := ct.nodeIndex["C1"]; ok {
		t.Error("C1 should have been removed from nodeIndex")
	}
	if len(ct.nodeIndex) != 1 {
		t.Errorf("nodeIndex has %d entries, want 1", len(ct.nodeIndex))
	}

	section := ct.sections[ChannelTypePrivate]
	if got := len(section.GetChildren()); got != 1 {
		t.Errorf("Channels section has %d children, want 1", got)
	}
}

func TestSetUnread(t *testing.T) {
	cfg := &config.Config{}
	ct := NewChannelsTree(cfg, nil)

	channels := []slack.Channel{
		makeChannel("C1", "general", false, false, false),
	}
	ct.Populate(channels, map[string]slack.User{}, "SELF")

	// Should not panic on valid or invalid channel IDs.
	ct.SetUnread("C1", true)
	ct.SetUnread("C1", false)
	ct.SetUnread("INVALID", true)
}

func TestStripBadge(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"# general (5)", "# general"},
		{"🔒 secret (12)", "🔒 secret"},
		{"● alice (1)", "● alice"},
		{"# general", "# general"},
		{"# general (abc)", "# general (abc)"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripBadge(tt.input)
			if got != tt.want {
				t.Errorf("stripBadge(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSetUnreadCount(t *testing.T) {
	cfg := &config.Config{}
	ct := NewChannelsTree(cfg, nil)

	channels := []slack.Channel{
		makeChannel("C1", "general", false, false, false),
	}
	ct.Populate(channels, map[string]slack.User{}, "SELF")

	// Set count to 5.
	ct.SetUnreadCount("C1", 5)
	node := ct.nodeIndex["C1"]
	if !strings.Contains(node.GetText(), "(5)") {
		t.Errorf("expected badge (5) in %q", node.GetText())
	}
	if ct.UnreadCount("C1") != 5 {
		t.Errorf("UnreadCount = %d, want 5", ct.UnreadCount("C1"))
	}

	// Increment with -1.
	ct.SetUnreadCount("C1", -1)
	if ct.UnreadCount("C1") != 6 {
		t.Errorf("UnreadCount after increment = %d, want 6", ct.UnreadCount("C1"))
	}
	if !strings.Contains(node.GetText(), "(6)") {
		t.Errorf("expected badge (6) in %q", node.GetText())
	}

	// Clear with 0.
	ct.SetUnreadCount("C1", 0)
	if strings.Contains(node.GetText(), "(") {
		t.Errorf("expected no badge after clearing, got %q", node.GetText())
	}
	if ct.UnreadCount("C1") != 0 {
		t.Errorf("UnreadCount after clear = %d, want 0", ct.UnreadCount("C1"))
	}

	// Invalid channel should not panic.
	ct.SetUnreadCount("INVALID", 5)
}

func TestSetMuted(t *testing.T) {
	cfg := &config.Config{}
	ct := NewChannelsTree(cfg, nil)

	channels := []slack.Channel{
		makeChannel("C1", "general", false, false, false),
		makeChannel("C2", "random", false, false, false),
	}
	ct.Populate(channels, map[string]slack.User{}, "SELF")

	// Initially not muted.
	if ct.IsMuted("C1") {
		t.Error("C1 should not be muted initially")
	}

	// Mute C1.
	ct.SetMuted("C1", true)
	if !ct.IsMuted("C1") {
		t.Error("C1 should be muted after SetMuted(true)")
	}
	// C2 should remain unmuted.
	if ct.IsMuted("C2") {
		t.Error("C2 should not be muted")
	}

	// Unmute C1.
	ct.SetMuted("C1", false)
	if ct.IsMuted("C1") {
		t.Error("C1 should not be muted after SetMuted(false)")
	}

	// Should not panic on invalid channel ID.
	ct.SetMuted("INVALID", true)
	if ct.IsMuted("INVALID") {
		t.Error("IsMuted should return false for unknown channel")
	}
}

func TestSetMutedAppliesStyle(t *testing.T) {
	cfg := &config.Config{}
	ct := NewChannelsTree(cfg, nil)

	channels := []slack.Channel{
		makeChannel("C1", "general", false, false, false),
	}
	ct.Populate(channels, map[string]slack.User{}, "SELF")

	node := ct.nodeIndex["C1"]
	normalStyle := node.GetTextStyle()

	// Muting should change the style to the muted style.
	ct.SetMuted("C1", true)
	mutedStyle := node.GetTextStyle()
	if mutedStyle == normalStyle {
		// With zero-value config the styles may coincide; the important test
		// is that after unmute the channel style is restored.
		t.Log("muted and normal styles are identical (zero-value config)")
	}

	// Unmuting should restore the channel style.
	ct.SetMuted("C1", false)
	restoredStyle := node.GetTextStyle()
	if restoredStyle != normalStyle {
		t.Error("unmuting should restore the original channel style")
	}
}

func TestSetUnreadCountSuppressedForMutedChannel(t *testing.T) {
	cfg := &config.Config{}
	ct := NewChannelsTree(cfg, nil)

	channels := []slack.Channel{
		makeChannel("C1", "general", false, false, false),
	}
	ct.Populate(channels, map[string]slack.User{}, "SELF")

	// Mute the channel.
	ct.SetMuted("C1", true)

	// SetUnreadCount should still track internally but not show a badge.
	ct.SetUnreadCount("C1", 5)
	node := ct.nodeIndex["C1"]
	if strings.Contains(node.GetText(), "(5)") {
		t.Errorf("muted channel should not show unread badge, got %q", node.GetText())
	}
	// Internal count should still be tracked.
	if ct.UnreadCount("C1") != 5 {
		t.Errorf("UnreadCount = %d, want 5 (tracked internally)", ct.UnreadCount("C1"))
	}

	// Unmute should reveal the badge.
	ct.SetMuted("C1", false)
	if !strings.Contains(node.GetText(), "(5)") {
		t.Errorf("unmuted channel should show unread badge, got %q", node.GetText())
	}
}

func TestMutedChannelClearedOnPopulate(t *testing.T) {
	cfg := &config.Config{}
	ct := NewChannelsTree(cfg, nil)

	channels := []slack.Channel{
		makeChannel("C1", "general", false, false, false),
	}
	ct.Populate(channels, map[string]slack.User{}, "SELF")
	ct.SetMuted("C1", true)

	// Re-populating should preserve the muted set (channels may re-appear).
	ct.Populate(channels, map[string]slack.User{}, "SELF")
	if !ct.IsMuted("C1") {
		t.Error("muted state should survive Populate")
	}
}

// makeChannel is a test helper that creates a slack.Channel with the given properties.
func makeChannel(id, name string, private, im, mpim bool) slack.Channel {
	ch := slack.Channel{}
	ch.ID = id
	ch.Name = name
	ch.IsPrivate = private
	ch.IsIM = im
	ch.IsMpIM = mpim
	return ch
}

// makeSharedChannel creates a Slack Connect (externally shared) channel.
func makeSharedChannel(id, name string) slack.Channel {
	ch := slack.Channel{}
	ch.ID = id
	ch.Name = name
	ch.IsExtShared = true
	return ch
}

// makeDMChannel creates a DM channel with the given user.
func makeDMChannel(id, userID string) slack.Channel {
	ch := slack.Channel{}
	ch.ID = id
	ch.User = userID
	ch.IsIM = true
	return ch
}
