package chat

import (
	"strings"
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func TestNewReactionUsersPanel(t *testing.T) {
	rp := NewReactionUsersPanel(&config.Config{})
	if rp == nil {
		t.Fatal("NewReactionUsersPanel returned nil")
	}
}

func TestReactionUsersPanelSetReactions(t *testing.T) {
	rp := NewReactionUsersPanel(&config.Config{})
	rp.SetReactions([]ReactionUsersEntry{
		{Emoji: "\U0001F44D", Name: "thumbsup", Users: []string{"alice", "bob", "charlie"}, IsSelf: false},
		{Emoji: "\u2764\uFE0F", Name: "heart", Users: []string{"alice", "dave"}, IsSelf: true},
	})

	text := rp.content.GetText(false)
	if !strings.Contains(text, "alice") {
		t.Error("missing user 'alice'")
	}
	if !strings.Contains(text, "bob") {
		t.Error("missing user 'bob'")
	}
	if !strings.Contains(text, "charlie") {
		t.Error("missing user 'charlie'")
	}
	if !strings.Contains(text, "dave") {
		t.Error("missing user 'dave'")
	}
	if !strings.Contains(text, "thumbsup") {
		t.Error("missing emoji name 'thumbsup'")
	}
	if !strings.Contains(text, "heart") {
		t.Error("missing emoji name 'heart'")
	}
}

func TestReactionUsersPanelSetReactionsEmpty(t *testing.T) {
	rp := NewReactionUsersPanel(&config.Config{})
	rp.SetReactions([]ReactionUsersEntry{})

	text := rp.content.GetText(false)
	if !strings.Contains(text, "No reactions") {
		t.Errorf("expected 'No reactions' for empty list, got %q", text)
	}
}

func TestReactionUsersPanelSelfHighlight(t *testing.T) {
	cfg := &config.Config{}
	cfg.Theme = config.BuiltinTheme("default")
	rp := NewReactionUsersPanel(cfg)

	// Test that both self and non-self reactions render correctly.
	rp.SetReactions([]ReactionUsersEntry{
		{Emoji: "\U0001F44D", Name: "thumbsup", Users: []string{"alice"}, IsSelf: true},
		{Emoji: "\u2764\uFE0F", Name: "heart", Users: []string{"bob"}, IsSelf: false},
	})

	text := rp.content.GetText(false)
	// Both reactions should be present.
	if !strings.Contains(text, "alice") {
		t.Error("missing self-reacting user 'alice'")
	}
	if !strings.Contains(text, "bob") {
		t.Error("missing other-reacting user 'bob'")
	}

	// Verify rendered format includes the formatted text with both entries.
	if !strings.Contains(text, "thumbsup") {
		t.Error("missing emoji name for self reaction")
	}
	if !strings.Contains(text, "heart") {
		t.Error("missing emoji name for other reaction")
	}
}

func TestReactionUsersPanelBuildText(t *testing.T) {
	cfg := &config.Config{}
	cfg.Theme = config.BuiltinTheme("default")
	rp := NewReactionUsersPanel(cfg)

	// Verify that buildReactionsText returns text with styling tags.
	entries := []ReactionUsersEntry{
		{Emoji: "\U0001F44D", Name: "thumbsup", Users: []string{"alice"}, IsSelf: true},
		{Emoji: "\u2764\uFE0F", Name: "heart", Users: []string{"bob"}, IsSelf: false},
	}

	raw := rp.buildReactionsText(entries)
	selfTag := cfg.Theme.MessagesList.ReactionSelf.Tag()
	otherTag := cfg.Theme.MessagesList.ReactionOther.Tag()

	if selfTag != "[-]" && selfTag != "[-:-:-]" {
		if !strings.Contains(raw, selfTag) {
			t.Errorf("expected self reaction tag %q in raw text:\n%s", selfTag, raw)
		}
	}
	if otherTag != "[-]" && otherTag != "[-:-:-]" {
		if !strings.Contains(raw, otherTag) {
			t.Errorf("expected other reaction tag %q in raw text:\n%s", otherTag, raw)
		}
	}
}

func TestReactionUsersPanelReset(t *testing.T) {
	rp := NewReactionUsersPanel(&config.Config{})
	rp.SetReactions([]ReactionUsersEntry{
		{Emoji: "\U0001F44D", Name: "thumbsup", Users: []string{"alice"}},
	})
	rp.Reset()

	text := rp.content.GetText(false)
	if text != "" {
		t.Errorf("after Reset, content should be empty, got %q", text)
	}
}

func TestReactionUsersPanelCallbacks(t *testing.T) {
	rp := NewReactionUsersPanel(&config.Config{})

	closeCalled := false
	rp.SetOnClose(func() { closeCalled = true })
	rp.onClose()

	if !closeCalled {
		t.Error("onClose not called")
	}
}

func TestReactionUsersPanelSetStatus(t *testing.T) {
	rp := NewReactionUsersPanel(&config.Config{})
	rp.SetStatus("Loading...")
	got := rp.status.GetText(false)
	if got != " Loading..." {
		t.Errorf("status = %q, want %q", got, " Loading...")
	}
}

func TestReactionUsersPanelSingleUser(t *testing.T) {
	rp := NewReactionUsersPanel(&config.Config{})
	rp.SetReactions([]ReactionUsersEntry{
		{Emoji: "\U0001F44D", Name: "thumbsup", Users: []string{"alice"}, IsSelf: false},
	})

	text := rp.content.GetText(false)
	if !strings.Contains(text, "alice") {
		t.Error("missing single user 'alice'")
	}
	// Should not contain commas for single user.
	if strings.Contains(text, ",") {
		t.Error("single user should not have commas")
	}
}

func TestReactionUsersPanelMultipleReactions(t *testing.T) {
	rp := NewReactionUsersPanel(&config.Config{})
	rp.SetReactions([]ReactionUsersEntry{
		{Emoji: "\U0001F44D", Name: "thumbsup", Users: []string{"alice", "bob"}, IsSelf: false},
		{Emoji: "\u2764\uFE0F", Name: "heart", Users: []string{"charlie"}, IsSelf: false},
		{Emoji: "\U0001F389", Name: "tada", Users: []string{"dave", "eve", "frank"}, IsSelf: true},
	})

	text := rp.content.GetText(false)
	// All users should be present.
	for _, name := range []string{"alice", "bob", "charlie", "dave", "eve", "frank"} {
		if !strings.Contains(text, name) {
			t.Errorf("missing user %q", name)
		}
	}
	// All emoji names should be present.
	for _, name := range []string{"thumbsup", "heart", "tada"} {
		if !strings.Contains(text, name) {
			t.Errorf("missing emoji name %q", name)
		}
	}
}
