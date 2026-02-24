package chat

import (
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func newTestInvitePicker() *InvitePicker {
	cfg := &config.Config{}
	cfg.Keybinds.InvitePicker.Close = "Escape"
	cfg.Keybinds.InvitePicker.Up = "Ctrl+P"
	cfg.Keybinds.InvitePicker.Down = "Ctrl+N"
	cfg.Keybinds.InvitePicker.Select = "Enter"
	return NewInvitePicker(cfg)
}

func makeInviteUserEntry(userID, displayName, realName string) InviteUserEntry {
	return InviteUserEntry{
		UserID:      userID,
		DisplayName: displayName,
		RealName:    realName,
	}
}

func TestInvitePicker_NewInvitePicker(t *testing.T) {
	ip := newTestInvitePicker()

	if ip == nil {
		t.Fatal("NewInvitePicker returned nil")
	}
	if ip.input == nil {
		t.Error("input field should not be nil")
	}
	if ip.list == nil {
		t.Error("list should not be nil")
	}
	if ip.status == nil {
		t.Error("status should not be nil")
	}
	if ip.Flex == nil {
		t.Error("Flex should not be nil")
	}
}

func TestInvitePicker_SetUsers(t *testing.T) {
	ip := newTestInvitePicker()

	users := []InviteUserEntry{
		makeInviteUserEntry("U1", "alice", "Alice Smith"),
		makeInviteUserEntry("U2", "bob", "Bob Jones"),
		makeInviteUserEntry("U3", "charlie", "Charlie Brown"),
	}

	ip.SetUsers(users)

	if len(ip.users) != 3 {
		t.Errorf("should have 3 users, got %d", len(ip.users))
	}

	// After setting users, filtered should show all.
	if ip.FilteredCount() != 3 {
		t.Errorf("filtered count should be 3, got %d", ip.FilteredCount())
	}
}

func TestInvitePicker_SetUsers_Empty(t *testing.T) {
	ip := newTestInvitePicker()

	ip.SetUsers(nil)

	if len(ip.users) != 0 {
		t.Errorf("should have 0 users, got %d", len(ip.users))
	}
	if ip.FilteredCount() != 0 {
		t.Errorf("filtered count should be 0, got %d", ip.FilteredCount())
	}
}

func TestInvitePicker_Reset(t *testing.T) {
	ip := newTestInvitePicker()

	users := []InviteUserEntry{
		makeInviteUserEntry("U1", "alice", "Alice Smith"),
		makeInviteUserEntry("U2", "bob", "Bob Jones"),
	}
	ip.SetUsers(users)

	// Filter down, then reset.
	ip.onInputChanged("alice")
	ip.Reset()

	// After reset, input should be empty and all items visible.
	if ip.input.GetText() != "" {
		t.Errorf("input should be empty after reset, got %q", ip.input.GetText())
	}
	if ip.FilteredCount() != 2 {
		t.Errorf("after reset should show all 2 users, got %d", ip.FilteredCount())
	}
}

func TestInvitePicker_FuzzyFilter(t *testing.T) {
	ip := newTestInvitePicker()

	users := []InviteUserEntry{
		makeInviteUserEntry("U1", "alice", "Alice Smith"),
		makeInviteUserEntry("U2", "bob", "Bob Jones"),
		makeInviteUserEntry("U3", "charlie", "Charlie Brown"),
	}
	ip.SetUsers(users)

	// Filter for "ali" should match "alice".
	ip.onInputChanged("ali")

	if ip.FilteredCount() < 1 {
		t.Errorf("filtering for 'ali' should match at least 1 user, got %d", ip.FilteredCount())
	}

	// Check that "alice" (U1) is in the results.
	found := false
	for _, idx := range ip.filtered {
		if ip.users[idx].UserID == "U1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("filtering for 'ali' should include alice (U1)")
	}
}

func TestInvitePicker_FuzzyFilter_RealName(t *testing.T) {
	ip := newTestInvitePicker()

	users := []InviteUserEntry{
		makeInviteUserEntry("U1", "alice", "Alice Smith"),
		makeInviteUserEntry("U2", "bob", "Bob Jones"),
	}
	ip.SetUsers(users)

	// Filter for "jones" should match "Bob Jones" by real name.
	ip.onInputChanged("jones")

	found := false
	for _, idx := range ip.filtered {
		if ip.users[idx].UserID == "U2" {
			found = true
			break
		}
	}
	if !found {
		t.Error("filtering for 'jones' should match Bob Jones (U2)")
	}
}

func TestInvitePicker_EmptyFilter(t *testing.T) {
	ip := newTestInvitePicker()

	users := []InviteUserEntry{
		makeInviteUserEntry("U1", "alice", "Alice Smith"),
		makeInviteUserEntry("U2", "bob", "Bob Jones"),
	}
	ip.SetUsers(users)

	// Type something, then clear.
	ip.onInputChanged("ali")
	ip.onInputChanged("")

	if ip.FilteredCount() != 2 {
		t.Errorf("empty filter should show all entries, got %d", ip.FilteredCount())
	}
}

func TestInvitePicker_SelectCallsCallback(t *testing.T) {
	ip := newTestInvitePicker()

	users := []InviteUserEntry{
		makeInviteUserEntry("U1", "alice", "Alice Smith"),
		makeInviteUserEntry("U2", "bob", "Bob Jones"),
	}
	ip.SetUsers(users)

	var gotID string
	ip.SetOnSelect(func(userID string) {
		gotID = userID
	})

	closeCalled := false
	ip.SetOnClose(func() {
		closeCalled = true
	})

	ip.selectCurrent()

	if gotID != "U1" {
		t.Errorf("selected user should be U1, got %q", gotID)
	}
	if !closeCalled {
		t.Error("close callback should have been called after selection")
	}
}

func TestInvitePicker_SelectNoCallback(t *testing.T) {
	ip := newTestInvitePicker()

	users := []InviteUserEntry{
		makeInviteUserEntry("U1", "alice", "Alice Smith"),
	}
	ip.SetUsers(users)

	// Should not panic without callbacks set.
	ip.selectCurrent()
}

func TestInvitePicker_SelectEmpty(t *testing.T) {
	ip := newTestInvitePicker()

	// No users, select should do nothing.
	var gotID string
	ip.SetOnSelect(func(userID string) {
		gotID = userID
	})

	ip.selectCurrent()

	if gotID != "" {
		t.Errorf("select on empty list should not call callback, got %q", gotID)
	}
}

func TestInvitePicker_DisplayTextFormat(t *testing.T) {
	ip := newTestInvitePicker()

	users := []InviteUserEntry{
		makeInviteUserEntry("U1", "alice", "Alice Smith"),
		makeInviteUserEntry("U2", "", "Bob Jones"),
		makeInviteUserEntry("U3", "", ""),
	}
	ip.SetUsers(users)

	// Verify list has the right number of items.
	if ip.list.GetItemCount() != 3 {
		t.Errorf("list should have 3 items, got %d", ip.list.GetItemCount())
	}
}

func TestInvitePicker_StatusText(t *testing.T) {
	ip := newTestInvitePicker()

	ip.SetStatus("Loading...")
	got := ip.status.GetText(false)
	if got != "Loading..." {
		t.Errorf("status text should be 'Loading...', got %q", got)
	}
}

func TestInvitePicker_FilterUpdatesStatus(t *testing.T) {
	ip := newTestInvitePicker()

	users := []InviteUserEntry{
		makeInviteUserEntry("U1", "alice", "Alice Smith"),
		makeInviteUserEntry("U2", "bob", "Bob Jones"),
		makeInviteUserEntry("U3", "charlie", "Charlie Brown"),
	}
	ip.SetUsers(users)

	// Filter to subset.
	ip.onInputChanged("ali")

	got := ip.status.GetText(false)
	if got == "" {
		t.Error("status should not be empty after filtering")
	}
}

func TestInviteUserSearchText(t *testing.T) {
	entry := InviteUserEntry{
		UserID:      "U1",
		DisplayName: "Alice",
		RealName:    "Alice Smith",
	}

	text := inviteUserSearchText(entry)
	if text == "" {
		t.Error("search text should not be empty")
	}
}
