package chat

import (
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func newTestMembersPicker() *MembersPicker {
	cfg := &config.Config{}
	cfg.Keybinds.MembersPicker.Close = "Escape"
	cfg.Keybinds.MembersPicker.Up = "Ctrl+P"
	cfg.Keybinds.MembersPicker.Down = "Ctrl+N"
	cfg.Keybinds.MembersPicker.Select = "Enter"
	return NewMembersPicker(cfg)
}

func makeMemberEntry(userID, displayName, realName, presence string, isBot bool) MemberEntry {
	return MemberEntry{
		UserID:      userID,
		DisplayName: displayName,
		RealName:    realName,
		Presence:    presence,
		IsBot:       isBot,
	}
}

func TestMembersPicker_NewMembersPicker(t *testing.T) {
	mp := newTestMembersPicker()

	if mp == nil {
		t.Fatal("NewMembersPicker returned nil")
	}
	if mp.input == nil {
		t.Error("input field should not be nil")
	}
	if mp.list == nil {
		t.Error("list should not be nil")
	}
	if mp.status == nil {
		t.Error("status should not be nil")
	}
	if mp.Flex == nil {
		t.Error("Flex should not be nil")
	}
}

func TestMembersPicker_SetMembers(t *testing.T) {
	mp := newTestMembersPicker()

	members := []MemberEntry{
		makeMemberEntry("U1", "alice", "Alice Smith", "active", false),
		makeMemberEntry("U2", "bob", "Bob Jones", "away", false),
		makeMemberEntry("U3", "charlie", "Charlie Brown", "", false),
	}

	mp.SetMembers(members)

	if len(mp.members) != 3 {
		t.Errorf("should have 3 members, got %d", len(mp.members))
	}

	// After setting members, filtered should show all.
	if mp.FilteredCount() != 3 {
		t.Errorf("filtered count should be 3, got %d", mp.FilteredCount())
	}
}

func TestMembersPicker_SetMembers_Empty(t *testing.T) {
	mp := newTestMembersPicker()

	mp.SetMembers(nil)

	if len(mp.members) != 0 {
		t.Errorf("should have 0 members, got %d", len(mp.members))
	}
	if mp.FilteredCount() != 0 {
		t.Errorf("filtered count should be 0, got %d", mp.FilteredCount())
	}
}

func TestMembersPicker_Reset(t *testing.T) {
	mp := newTestMembersPicker()

	members := []MemberEntry{
		makeMemberEntry("U1", "alice", "Alice Smith", "active", false),
		makeMemberEntry("U2", "bob", "Bob Jones", "away", false),
	}
	mp.SetMembers(members)

	// Filter down, then reset.
	mp.onInputChanged("alice")
	mp.Reset()

	// After reset, input should be empty and all items visible.
	if mp.input.GetText() != "" {
		t.Errorf("input should be empty after reset, got %q", mp.input.GetText())
	}
	if mp.FilteredCount() != 2 {
		t.Errorf("after reset should show all 2 members, got %d", mp.FilteredCount())
	}
}

func TestMembersPicker_FuzzyFilter(t *testing.T) {
	mp := newTestMembersPicker()

	members := []MemberEntry{
		makeMemberEntry("U1", "alice", "Alice Smith", "active", false),
		makeMemberEntry("U2", "bob", "Bob Jones", "away", false),
		makeMemberEntry("U3", "charlie", "Charlie Brown", "", false),
	}
	mp.SetMembers(members)

	// Filter for "ali" should match "alice".
	mp.onInputChanged("ali")

	if mp.FilteredCount() < 1 {
		t.Errorf("filtering for 'ali' should match at least 1 member, got %d", mp.FilteredCount())
	}

	// Check that "alice" (U1) is in the results.
	found := false
	for _, idx := range mp.filtered {
		if mp.members[idx].UserID == "U1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("filtering for 'ali' should include alice (U1)")
	}
}

func TestMembersPicker_FuzzyFilter_RealName(t *testing.T) {
	mp := newTestMembersPicker()

	members := []MemberEntry{
		makeMemberEntry("U1", "alice", "Alice Smith", "active", false),
		makeMemberEntry("U2", "bob", "Bob Jones", "away", false),
	}
	mp.SetMembers(members)

	// Filter for "jones" should match "Bob Jones" by real name.
	mp.onInputChanged("jones")

	found := false
	for _, idx := range mp.filtered {
		if mp.members[idx].UserID == "U2" {
			found = true
			break
		}
	}
	if !found {
		t.Error("filtering for 'jones' should match Bob Jones (U2)")
	}
}

func TestMembersPicker_EmptyFilter(t *testing.T) {
	mp := newTestMembersPicker()

	members := []MemberEntry{
		makeMemberEntry("U1", "alice", "Alice Smith", "active", false),
		makeMemberEntry("U2", "bob", "Bob Jones", "away", false),
	}
	mp.SetMembers(members)

	// Type something, then clear.
	mp.onInputChanged("ali")
	mp.onInputChanged("")

	if mp.FilteredCount() != 2 {
		t.Errorf("empty filter should show all entries, got %d", mp.FilteredCount())
	}
}

func TestMembersPicker_SelectCallsCallback(t *testing.T) {
	mp := newTestMembersPicker()

	members := []MemberEntry{
		makeMemberEntry("U1", "alice", "Alice Smith", "active", false),
		makeMemberEntry("U2", "bob", "Bob Jones", "away", false),
	}
	mp.SetMembers(members)

	var gotID string
	mp.SetOnSelect(func(userID string) {
		gotID = userID
	})

	closeCalled := false
	mp.SetOnClose(func() {
		closeCalled = true
	})

	mp.selectCurrent()

	if gotID != "U1" {
		t.Errorf("selected member should be U1, got %q", gotID)
	}
	if !closeCalled {
		t.Error("close callback should have been called after selection")
	}
}

func TestMembersPicker_SelectNoCallback(t *testing.T) {
	mp := newTestMembersPicker()

	members := []MemberEntry{
		makeMemberEntry("U1", "alice", "Alice Smith", "active", false),
	}
	mp.SetMembers(members)

	// Should not panic without callbacks set.
	mp.selectCurrent()
}

func TestMembersPicker_SelectEmpty(t *testing.T) {
	mp := newTestMembersPicker()

	// No members, select should do nothing.
	var gotID string
	mp.SetOnSelect(func(userID string) {
		gotID = userID
	})

	mp.selectCurrent()

	if gotID != "" {
		t.Errorf("select on empty list should not call callback, got %q", gotID)
	}
}

func TestMembersPicker_DisplayTextFormat(t *testing.T) {
	mp := newTestMembersPicker()

	members := []MemberEntry{
		makeMemberEntry("U1", "alice", "Alice Smith", "active", false),
		makeMemberEntry("U2", "", "Bob Jones", "away", false),
		makeMemberEntry("U3", "botuser", "Bot User", "", true),
	}
	mp.SetMembers(members)

	// Verify list has the right number of items.
	if mp.list.GetItemCount() != 3 {
		t.Errorf("list should have 3 items, got %d", mp.list.GetItemCount())
	}
}

func TestMembersPicker_StatusText(t *testing.T) {
	mp := newTestMembersPicker()

	mp.SetStatus("Loading...")
	// The status text should be set (we can check the text view contains it).
	got := mp.status.GetText(false)
	if got != "Loading..." {
		t.Errorf("status text should be 'Loading...', got %q", got)
	}
}

func TestMembersPicker_BotMembers(t *testing.T) {
	mp := newTestMembersPicker()

	members := []MemberEntry{
		makeMemberEntry("U1", "alice", "Alice Smith", "active", false),
		makeMemberEntry("B1", "slackbot", "Slackbot", "", true),
	}
	mp.SetMembers(members)

	// Both should appear.
	if mp.FilteredCount() != 2 {
		t.Errorf("should show 2 members including bot, got %d", mp.FilteredCount())
	}
}

func TestMembersPicker_PresenceIndicator(t *testing.T) {
	mp := newTestMembersPicker()

	members := []MemberEntry{
		makeMemberEntry("U1", "alice", "Alice Smith", "active", false),
		makeMemberEntry("U2", "bob", "Bob Jones", "away", false),
		makeMemberEntry("U3", "charlie", "Charlie Brown", "", false),
	}
	mp.SetMembers(members)

	// Check that display text includes presence indicators.
	// The first item should have the active indicator.
	mainText, _ := mp.list.GetItemText(0)
	if mainText == "" {
		t.Error("first item display text should not be empty")
	}
}
