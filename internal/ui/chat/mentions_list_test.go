package chat

import (
	"testing"

	"github.com/slack-go/slack"

	"github.com/m96-chan/Slacko/internal/config"
)

func newTestMentionsList() *MentionsList {
	cfg := &config.Config{}
	cfg.AutocompleteLimit = 5
	return NewMentionsList(cfg)
}

func TestMentionsList_SetUsers(t *testing.T) {
	ml := newTestMentionsList()

	users := map[string]slack.User{
		"U1": {ID: "U1", Name: "alice", RealName: "Alice Smith",
			Profile: slack.UserProfile{DisplayName: "alice"}},
		"U2": {ID: "U2", Name: "bob", RealName: "Bob Jones",
			Profile: slack.UserProfile{DisplayName: "bob"}},
	}

	ml.SetUsers(users)

	if len(ml.users) != 2 {
		t.Errorf("should have 2 users, got %d", len(ml.users))
	}
}

func TestMentionsList_SetUsers_SkipsBots(t *testing.T) {
	ml := newTestMentionsList()

	users := map[string]slack.User{
		"U1": {ID: "U1", Name: "alice"},
		"B1": {ID: "B1", Name: "slackbot", IsBot: true},
		"D1": {ID: "D1", Name: "deleted", Deleted: true},
	}

	ml.SetUsers(users)

	if len(ml.users) != 1 {
		t.Errorf("should have 1 user (skipping bots and deleted), got %d", len(ml.users))
	}
}

func TestMentionsList_FilterUsers(t *testing.T) {
	ml := newTestMentionsList()

	users := map[string]slack.User{
		"U1": {ID: "U1", Name: "alice", RealName: "Alice Smith",
			Profile: slack.UserProfile{DisplayName: "alice"}},
		"U2": {ID: "U2", Name: "bob", RealName: "Bob Jones",
			Profile: slack.UserProfile{DisplayName: "bob"}},
		"U3": {ID: "U3", Name: "charlie", RealName: "Charlie Brown",
			Profile: slack.UserProfile{DisplayName: "charlie"}},
	}
	ml.SetUsers(users)

	count := ml.Filter(acUser, "ali", 5)

	if count < 1 {
		t.Errorf("filtering for 'ali' should match at least 1 user, got %d", count)
	}

	// Check that alice is in the results.
	found := false
	for _, s := range ml.suggestions {
		if s.insertText == "<@U1> " {
			found = true
			break
		}
	}
	if !found {
		t.Error("filtering for 'ali' should include alice")
	}
}

func TestMentionsList_FilterUsers_Empty(t *testing.T) {
	ml := newTestMentionsList()

	users := map[string]slack.User{
		"U1": {ID: "U1", Name: "alice"},
		"U2": {ID: "U2", Name: "bob"},
	}
	ml.SetUsers(users)

	count := ml.Filter(acUser, "", 5)

	if count != 2 {
		t.Errorf("empty prefix should show all users, got %d", count)
	}
}

func TestMentionsList_FilterUsers_Limit(t *testing.T) {
	ml := newTestMentionsList()

	users := make(map[string]slack.User)
	for i := 0; i < 20; i++ {
		id := "U" + string(rune('A'+i))
		users[id] = slack.User{ID: id, Name: "user" + string(rune('a'+i))}
	}
	ml.SetUsers(users)

	count := ml.Filter(acUser, "", 5)

	if count != 5 {
		t.Errorf("should limit to 5 suggestions, got %d", count)
	}
}

func TestMentionsList_FilterChannels(t *testing.T) {
	ml := newTestMentionsList()

	channels := []slack.Channel{
		makePickerChannel("C1", "general", false, false, false),
		makePickerChannel("C2", "random", false, false, false),
		makePickerChannel("C3", "engineering", false, false, false),
	}
	ml.SetChannels(channels, nil, "")

	count := ml.Filter(acChannel, "gen", 5)

	if count < 1 {
		t.Errorf("filtering for 'gen' should match at least 1 channel, got %d", count)
	}

	found := false
	for _, s := range ml.suggestions {
		if s.insertText == "<#C1> " {
			found = true
			break
		}
	}
	if !found {
		t.Error("filtering for 'gen' should include #general")
	}
}

func TestMentionsList_GetSelected(t *testing.T) {
	ml := newTestMentionsList()

	users := map[string]slack.User{
		"U1": {ID: "U1", Name: "alice"},
		"U2": {ID: "U2", Name: "bob"},
	}
	ml.SetUsers(users)
	ml.Filter(acUser, "", 5)

	sel := ml.GetSelected()
	if sel.insertText == "" {
		t.Error("GetSelected should return a suggestion when items exist")
	}
}

func TestMentionsList_SelectNavigation(t *testing.T) {
	ml := newTestMentionsList()

	users := map[string]slack.User{
		"U1": {ID: "U1", Name: "alice"},
		"U2": {ID: "U2", Name: "bob"},
		"U3": {ID: "U3", Name: "charlie"},
	}
	ml.SetUsers(users)
	ml.Filter(acUser, "", 5)

	// Initially at 0.
	if ml.GetCurrentItem() != 0 {
		t.Errorf("should start at item 0, got %d", ml.GetCurrentItem())
	}

	ml.SelectNext()
	if ml.GetCurrentItem() != 1 {
		t.Errorf("after SelectNext, should be at item 1, got %d", ml.GetCurrentItem())
	}

	ml.SelectPrev()
	if ml.GetCurrentItem() != 0 {
		t.Errorf("after SelectPrev, should be at item 0, got %d", ml.GetCurrentItem())
	}

	// Should not go below 0.
	ml.SelectPrev()
	if ml.GetCurrentItem() != 0 {
		t.Errorf("SelectPrev at 0 should stay at 0, got %d", ml.GetCurrentItem())
	}
}

func TestFindAutocompleteTrigger(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantKind   autocompleteKind
		wantPrefix string
		wantStart  int
	}{
		{"empty", "", acNone, "", -1},
		{"no trigger", "hello world", acNone, "", -1},
		{"at sign only", "@", acUser, "", 0},
		{"at with prefix", "@ali", acUser, "ali", 0},
		{"at after text", "hello @bob", acUser, "bob", 6},
		{"hash sign only", "#", acChannel, "", 0},
		{"hash with prefix", "#gen", acChannel, "gen", 0},
		{"hash after text", "hello #gen", acChannel, "gen", 6},
		{"space after trigger stops", "@ali ", acNone, "", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, prefix, start := findAutocompleteTrigger(tt.text)
			if kind != tt.wantKind {
				t.Errorf("kind: got %d, want %d", kind, tt.wantKind)
			}
			if prefix != tt.wantPrefix {
				t.Errorf("prefix: got %q, want %q", prefix, tt.wantPrefix)
			}
			if start != tt.wantStart {
				t.Errorf("start: got %d, want %d", start, tt.wantStart)
			}
		})
	}
}

func TestMessageInput_CompleteAutocomplete(t *testing.T) {
	cfg := &config.Config{}
	cfg.Keybinds.MessageInput.Send = "Enter"
	cfg.Keybinds.MessageInput.Newline = "Shift+Enter"
	cfg.Keybinds.MessageInput.Cancel = "Escape"
	cfg.Keybinds.MessageInput.TabComplete = "Tab"
	cfg.AutocompleteLimit = 5

	mi := NewMessageInput(cfg)
	ml := NewMentionsList(cfg)

	users := map[string]slack.User{
		"U1": {ID: "U1", Name: "alice",
			Profile: slack.UserProfile{DisplayName: "alice"}},
	}
	ml.SetUsers(users)

	mi.SetMentionsList(ml)

	// Simulate typing "hello @ali"
	mi.SetText("hello @ali", true)
	mi.acKind = acUser
	mi.acStart = 6
	ml.Filter(acUser, "ali", 5)

	// Complete.
	mi.completeAutocomplete()

	got := mi.GetText()
	expected := "hello <@U1> "
	if got != expected {
		t.Errorf("after completion, text should be %q, got %q", expected, got)
	}
	if mi.acKind != acNone {
		t.Error("acKind should be acNone after completion")
	}
}
