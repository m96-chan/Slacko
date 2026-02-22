package chat

import (
	"testing"

	"github.com/slack-go/slack"

	"github.com/m96-chan/Slacko/internal/config"
)

func newTestReactionsPicker() *ReactionsPicker {
	cfg := &config.Config{}
	return NewReactionsPicker(cfg)
}

func TestReactionsPicker_BuildEntries(t *testing.T) {
	rp := newTestReactionsPicker()

	if len(rp.entries) == 0 {
		t.Fatal("entries should be populated from emoji map")
	}

	// Entries should be sorted alphabetically.
	for i := 1; i < len(rp.entries); i++ {
		if rp.entries[i].name < rp.entries[i-1].name {
			t.Errorf("entries not sorted: %q before %q", rp.entries[i-1].name, rp.entries[i].name)
			break
		}
	}
}

func TestReactionsPicker_ShowFrequent(t *testing.T) {
	rp := newTestReactionsPicker()
	rp.Reset()

	if len(rp.filtered) == 0 {
		t.Fatal("showFrequent should populate filtered entries")
	}

	// Should have at most len(frequentEmoji) entries.
	if len(rp.filtered) > len(frequentEmoji) {
		t.Errorf("filtered count %d > frequent count %d", len(rp.filtered), len(frequentEmoji))
	}
}

func TestReactionsPicker_Filter(t *testing.T) {
	rp := newTestReactionsPicker()

	// Filter for "thumb".
	rp.onInputChanged("thumb")

	if len(rp.filtered) == 0 {
		t.Fatal("filtering for 'thumb' should return results")
	}

	// Should include thumbsup.
	found := false
	for _, idx := range rp.filtered {
		if rp.entries[idx].name == "thumbsup" {
			found = true
			break
		}
	}
	if !found {
		t.Error("filtering for 'thumb' should include thumbsup")
	}
}

func TestReactionsPicker_FilterEmpty(t *testing.T) {
	rp := newTestReactionsPicker()

	// Empty filter shows frequent.
	rp.onInputChanged("")

	if len(rp.filtered) == 0 {
		t.Fatal("empty filter should show frequent emoji")
	}
}

func TestReactionsPicker_SelectCallback(t *testing.T) {
	rp := newTestReactionsPicker()

	var selected string
	rp.SetOnSelect(func(name string) {
		selected = name
	})
	rp.SetOnClose(func() {})

	// Show frequent and select first.
	rp.Reset()
	rp.selectCurrent()

	if selected == "" {
		t.Error("selecting an emoji should call onSelect")
	}
}

func TestReactionsPicker_CloseCallback(t *testing.T) {
	rp := newTestReactionsPicker()

	closed := false
	rp.SetOnClose(func() {
		closed = true
	})

	rp.close()

	if !closed {
		t.Error("close should call onClose")
	}
}

func TestContainsStr(t *testing.T) {
	tests := []struct {
		name string
		ss   []string
		s    string
		want bool
	}{
		{"found", []string{"a", "b", "c"}, "b", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"nil slice", nil, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsStr(tt.ss, tt.s)
			if got != tt.want {
				t.Errorf("containsStr(%v, %q) = %v, want %v", tt.ss, tt.s, got, tt.want)
			}
		})
	}
}

func TestAddReaction_TracksUser(t *testing.T) {
	cfg := &config.Config{}
	cfg.Timestamps.Enabled = false
	ml := NewMessagesList(cfg)
	ml.channelID = "C1"
	ml.selfUserID = "U1"

	ml.messages = []slack.Message{{}}
	ml.messages[0].Timestamp = "123.456"
	ml.messages[0].Channel = "C1"

	ml.AddReaction("C1", "123.456", "thumbsup", "U2")

	if len(ml.messages[0].Reactions) != 1 {
		t.Fatalf("expected 1 reaction, got %d", len(ml.messages[0].Reactions))
	}
	r := ml.messages[0].Reactions[0]
	if r.Name != "thumbsup" || r.Count != 1 {
		t.Errorf("reaction: got name=%q count=%d", r.Name, r.Count)
	}
	if len(r.Users) != 1 || r.Users[0] != "U2" {
		t.Errorf("users: got %v, want [U2]", r.Users)
	}

	// Add same reaction from different user.
	ml.AddReaction("C1", "123.456", "thumbsup", "U1")
	r = ml.messages[0].Reactions[0]
	if r.Count != 2 {
		t.Errorf("count after second add: got %d, want 2", r.Count)
	}
	if len(r.Users) != 2 {
		t.Errorf("users after second add: got %v, want 2 users", r.Users)
	}
}

func TestRemoveReaction_TracksUser(t *testing.T) {
	cfg := &config.Config{}
	cfg.Timestamps.Enabled = false
	ml := NewMessagesList(cfg)
	ml.channelID = "C1"

	ml.messages = []slack.Message{{}}
	ml.messages[0].Timestamp = "123.456"
	ml.messages[0].Channel = "C1"

	ml.AddReaction("C1", "123.456", "thumbsup", "U1")
	ml.AddReaction("C1", "123.456", "thumbsup", "U2")

	ml.RemoveReaction("C1", "123.456", "thumbsup", "U1")

	r := ml.messages[0].Reactions[0]
	if r.Count != 1 {
		t.Errorf("count after remove: got %d, want 1", r.Count)
	}
	if len(r.Users) != 1 || r.Users[0] != "U2" {
		t.Errorf("users after remove: got %v, want [U2]", r.Users)
	}

	// Remove last reaction should remove the reaction entirely.
	ml.RemoveReaction("C1", "123.456", "thumbsup", "U2")
	if len(ml.messages[0].Reactions) != 0 {
		t.Errorf("reactions should be empty after removing all, got %d", len(ml.messages[0].Reactions))
	}
}
