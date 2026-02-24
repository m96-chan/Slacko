package chat

import (
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func newTestGroupDMPicker() *GroupDMPicker {
	cfg := &config.Config{}
	cfg.Keybinds.GroupDMPicker.Close = "Escape"
	cfg.Keybinds.GroupDMPicker.Up = "Ctrl+P"
	cfg.Keybinds.GroupDMPicker.Down = "Ctrl+N"
	cfg.Keybinds.GroupDMPicker.Add = "Enter"
	cfg.Keybinds.GroupDMPicker.Remove = "Ctrl+D"
	cfg.Keybinds.GroupDMPicker.Confirm = "Ctrl+Enter"
	return NewGroupDMPicker(cfg)
}

func makeGroupDMUserEntry(userID, displayName string) GroupDMUserEntry {
	return GroupDMUserEntry{
		UserID:      userID,
		DisplayName: displayName,
	}
}

func TestGroupDMPicker_NewGroupDMPicker(t *testing.T) {
	gp := newTestGroupDMPicker()

	if gp == nil {
		t.Fatal("NewGroupDMPicker returned nil")
	}
	if gp.input == nil {
		t.Error("input field should not be nil")
	}
	if gp.list == nil {
		t.Error("list should not be nil")
	}
	if gp.selected == nil {
		t.Error("selected text view should not be nil")
	}
	if gp.status == nil {
		t.Error("status should not be nil")
	}
	if gp.Flex == nil {
		t.Error("Flex should not be nil")
	}
}

func TestGroupDMPicker_SetUsers(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
		makeGroupDMUserEntry("U2", "bob"),
		makeGroupDMUserEntry("U3", "charlie"),
	}

	gp.SetUsers(users)

	if len(gp.users) != 3 {
		t.Errorf("should have 3 users, got %d", len(gp.users))
	}

	// After setting users, filtered should show all.
	if gp.FilteredCount() != 3 {
		t.Errorf("filtered count should be 3, got %d", gp.FilteredCount())
	}
}

func TestGroupDMPicker_SetUsers_Empty(t *testing.T) {
	gp := newTestGroupDMPicker()

	gp.SetUsers(nil)

	if len(gp.users) != 0 {
		t.Errorf("should have 0 users, got %d", len(gp.users))
	}
	if gp.FilteredCount() != 0 {
		t.Errorf("filtered count should be 0, got %d", gp.FilteredCount())
	}
}

func TestGroupDMPicker_Reset(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
		makeGroupDMUserEntry("U2", "bob"),
	}
	gp.SetUsers(users)

	// Add a user, filter, then reset.
	gp.addCurrent()
	gp.onInputChanged("bob")
	gp.Reset()

	// After reset, input should be empty, chosen should be empty, all items visible.
	if gp.input.GetText() != "" {
		t.Errorf("input should be empty after reset, got %q", gp.input.GetText())
	}
	if gp.SelectedCount() != 0 {
		t.Errorf("chosen should be empty after reset, got %d", gp.SelectedCount())
	}
	if gp.FilteredCount() != 2 {
		t.Errorf("after reset should show all 2 users, got %d", gp.FilteredCount())
	}
}

func TestGroupDMPicker_FuzzyFilter(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
		makeGroupDMUserEntry("U2", "bob"),
		makeGroupDMUserEntry("U3", "charlie"),
	}
	gp.SetUsers(users)

	// Filter for "ali" should match "alice".
	gp.onInputChanged("ali")

	if gp.FilteredCount() < 1 {
		t.Errorf("filtering for 'ali' should match at least 1 user, got %d", gp.FilteredCount())
	}

	// Check that "alice" (U1) is in the results.
	found := false
	for _, idx := range gp.filtered {
		if gp.users[idx].UserID == "U1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("filtering for 'ali' should include alice (U1)")
	}
}

func TestGroupDMPicker_EmptyFilter(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
		makeGroupDMUserEntry("U2", "bob"),
	}
	gp.SetUsers(users)

	// Type something, then clear.
	gp.onInputChanged("ali")
	gp.onInputChanged("")

	if gp.FilteredCount() != 2 {
		t.Errorf("empty filter should show all entries, got %d", gp.FilteredCount())
	}
}

func TestGroupDMPicker_AddUser(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
		makeGroupDMUserEntry("U2", "bob"),
		makeGroupDMUserEntry("U3", "charlie"),
	}
	gp.SetUsers(users)

	// Add the first user (alice).
	gp.addCurrent()

	if gp.SelectedCount() != 1 {
		t.Errorf("should have 1 chosen user, got %d", gp.SelectedCount())
	}

	// Verify alice was added.
	chosenIDs := gp.ChosenUserIDs()
	if len(chosenIDs) != 1 || chosenIDs[0] != "U1" {
		t.Errorf("chosen user IDs should be [U1], got %v", chosenIDs)
	}

	// Alice should be excluded from the filtered list now.
	if gp.FilteredCount() != 2 {
		t.Errorf("filtered should exclude chosen users, got %d", gp.FilteredCount())
	}
}

func TestGroupDMPicker_AddMultipleUsers(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
		makeGroupDMUserEntry("U2", "bob"),
		makeGroupDMUserEntry("U3", "charlie"),
	}
	gp.SetUsers(users)

	// Add first (alice), then first of remaining (bob).
	gp.addCurrent()
	gp.addCurrent()

	if gp.SelectedCount() != 2 {
		t.Errorf("should have 2 chosen users, got %d", gp.SelectedCount())
	}

	chosenIDs := gp.ChosenUserIDs()
	if len(chosenIDs) != 2 {
		t.Errorf("should have 2 chosen user IDs, got %d", len(chosenIDs))
	}

	// Only charlie should remain in the filtered list.
	if gp.FilteredCount() != 1 {
		t.Errorf("filtered should exclude chosen users, got %d", gp.FilteredCount())
	}
}

func TestGroupDMPicker_AddDuplicate(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
	}
	gp.SetUsers(users)

	gp.addCurrent()

	// After adding the only user, filtered is empty; addCurrent should be a no-op.
	gp.addCurrent()

	if gp.SelectedCount() != 1 {
		t.Errorf("should still have 1 chosen user, got %d", gp.SelectedCount())
	}
}

func TestGroupDMPicker_RemoveLastChosen(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
		makeGroupDMUserEntry("U2", "bob"),
	}
	gp.SetUsers(users)

	// Add alice, then remove her.
	gp.addCurrent()
	if gp.SelectedCount() != 1 {
		t.Fatalf("should have 1 chosen user, got %d", gp.SelectedCount())
	}

	gp.removeLastChosen()

	if gp.SelectedCount() != 0 {
		t.Errorf("should have 0 chosen users after removal, got %d", gp.SelectedCount())
	}

	// Alice should reappear in the filtered list.
	if gp.FilteredCount() != 2 {
		t.Errorf("filtered should show all 2 users after removal, got %d", gp.FilteredCount())
	}
}

func TestGroupDMPicker_RemoveFromEmpty(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
	}
	gp.SetUsers(users)

	// Should not panic.
	gp.removeLastChosen()

	if gp.SelectedCount() != 0 {
		t.Errorf("should have 0 chosen users, got %d", gp.SelectedCount())
	}
}

func TestGroupDMPicker_SelectedCount(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
		makeGroupDMUserEntry("U2", "bob"),
		makeGroupDMUserEntry("U3", "charlie"),
	}
	gp.SetUsers(users)

	if gp.SelectedCount() != 0 {
		t.Errorf("initial selected count should be 0, got %d", gp.SelectedCount())
	}

	gp.addCurrent()
	if gp.SelectedCount() != 1 {
		t.Errorf("after adding 1, count should be 1, got %d", gp.SelectedCount())
	}

	gp.addCurrent()
	if gp.SelectedCount() != 2 {
		t.Errorf("after adding 2, count should be 2, got %d", gp.SelectedCount())
	}

	gp.removeLastChosen()
	if gp.SelectedCount() != 1 {
		t.Errorf("after removing 1, count should be 1, got %d", gp.SelectedCount())
	}
}

func TestGroupDMPicker_OnCreateCallback(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
		makeGroupDMUserEntry("U2", "bob"),
	}
	gp.SetUsers(users)

	var gotIDs []string
	gp.SetOnCreate(func(userIDs []string) {
		gotIDs = userIDs
	})

	// Add both users, then confirm.
	gp.addCurrent()
	gp.addCurrent()
	gp.confirm()

	if len(gotIDs) != 2 {
		t.Errorf("onCreate should receive 2 user IDs, got %d", len(gotIDs))
	}
}

func TestGroupDMPicker_ConfirmWithNoUsers(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
	}
	gp.SetUsers(users)

	var createCalled bool
	gp.SetOnCreate(func(userIDs []string) {
		createCalled = true
	})

	// Confirm without selecting anyone should not call onCreate.
	gp.confirm()

	if createCalled {
		t.Error("onCreate should not be called when no users are selected")
	}
}

func TestGroupDMPicker_OnCloseCallback(t *testing.T) {
	gp := newTestGroupDMPicker()

	closeCalled := false
	gp.SetOnClose(func() {
		closeCalled = true
	})

	gp.close()

	if !closeCalled {
		t.Error("onClose should be called")
	}
}

func TestGroupDMPicker_FilterExcludesChosen(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
		makeGroupDMUserEntry("U2", "alison"),
		makeGroupDMUserEntry("U3", "bob"),
	}
	gp.SetUsers(users)

	// Add alice.
	gp.addCurrent()

	// Filter for "ali" should only match alison, not alice who is already chosen.
	gp.onInputChanged("ali")

	for _, idx := range gp.filtered {
		if gp.users[idx].UserID == "U1" {
			t.Error("chosen user alice (U1) should not appear in filtered results")
		}
	}

	found := false
	for _, idx := range gp.filtered {
		if gp.users[idx].UserID == "U2" {
			found = true
			break
		}
	}
	if !found {
		t.Error("alison (U2) should appear in filtered results")
	}
}

func TestGroupDMPicker_StatusText(t *testing.T) {
	gp := newTestGroupDMPicker()

	gp.SetStatus("Loading...")
	got := gp.status.GetText(false)
	if got != "Loading..." {
		t.Errorf("status text should be 'Loading...', got %q", got)
	}
}

func TestGroupDMPicker_ChosenUserIDs(t *testing.T) {
	gp := newTestGroupDMPicker()

	users := []GroupDMUserEntry{
		makeGroupDMUserEntry("U1", "alice"),
		makeGroupDMUserEntry("U2", "bob"),
	}
	gp.SetUsers(users)

	ids := gp.ChosenUserIDs()
	if len(ids) != 0 {
		t.Errorf("initial chosen IDs should be empty, got %v", ids)
	}

	gp.addCurrent()
	ids = gp.ChosenUserIDs()
	if len(ids) != 1 || ids[0] != "U1" {
		t.Errorf("after adding first, chosen IDs should be [U1], got %v", ids)
	}
}
