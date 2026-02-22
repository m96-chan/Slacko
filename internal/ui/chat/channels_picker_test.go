package chat

import (
	"testing"

	"github.com/slack-go/slack"

	"github.com/m96-chan/Slacko/internal/config"
)

func newTestPicker() *ChannelsPicker {
	cfg := &config.Config{}
	cfg.Keybinds.ChannelsPicker.Close = "Escape"
	cfg.Keybinds.ChannelsPicker.Up = "Ctrl+P"
	cfg.Keybinds.ChannelsPicker.Down = "Ctrl+N"
	cfg.Keybinds.ChannelsPicker.Select = "Enter"
	cfg.Keybinds.ChannelPicker = "Ctrl+K"
	return NewChannelsPicker(cfg)
}

func makePickerChannel(id, name string, isIM, isMpIM, isPrivate bool) slack.Channel {
	ch := slack.Channel{}
	ch.ID = id
	ch.Name = name
	ch.IsIM = isIM
	ch.IsMpIM = isMpIM
	ch.IsPrivate = isPrivate
	return ch
}

func TestChannelsPicker_SetData(t *testing.T) {
	cp := newTestPicker()

	channels := []slack.Channel{
		makePickerChannel("C1", "general", false, false, false),
		makePickerChannel("C2", "random", false, false, false),
		makePickerChannel("C3", "secret", false, false, true),
	}

	cp.SetData(channels, nil, "")

	if len(cp.entries) != 3 {
		t.Errorf("should have 3 entries, got %d", len(cp.entries))
	}
}

func TestChannelsPicker_ShowAll(t *testing.T) {
	cp := newTestPicker()

	channels := []slack.Channel{
		makePickerChannel("C1", "general", false, false, false),
		makePickerChannel("C2", "random", false, false, false),
	}
	cp.SetData(channels, nil, "")
	cp.Reset()

	if cp.FilteredCount() != 2 {
		t.Errorf("showAll should show all 2 entries, got %d", cp.FilteredCount())
	}
}

func TestChannelsPicker_FuzzyFilter(t *testing.T) {
	cp := newTestPicker()

	channels := []slack.Channel{
		makePickerChannel("C1", "general", false, false, false),
		makePickerChannel("C2", "random", false, false, false),
		makePickerChannel("C3", "engineering", false, false, false),
	}
	cp.SetData(channels, nil, "")

	// Filter for "gen" should match "general" and "engineering".
	cp.onInputChanged("gen")

	if cp.FilteredCount() < 1 {
		t.Errorf("filtering for 'gen' should match at least 1 channel, got %d", cp.FilteredCount())
	}

	// Check that "general" is in the results.
	found := false
	for _, idx := range cp.filtered {
		if cp.entries[idx].channelID == "C1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("filtering for 'gen' should include 'general'")
	}
}

func TestChannelsPicker_EmptyFilter(t *testing.T) {
	cp := newTestPicker()

	channels := []slack.Channel{
		makePickerChannel("C1", "general", false, false, false),
		makePickerChannel("C2", "random", false, false, false),
	}
	cp.SetData(channels, nil, "")

	// Type something, then clear.
	cp.onInputChanged("gen")
	cp.onInputChanged("")

	if cp.FilteredCount() != 2 {
		t.Errorf("empty filter should show all entries, got %d", cp.FilteredCount())
	}
}

func TestChannelsPicker_SelectCallsCallback(t *testing.T) {
	cp := newTestPicker()

	channels := []slack.Channel{
		makePickerChannel("C1", "general", false, false, false),
		makePickerChannel("C2", "random", false, false, false),
	}
	cp.SetData(channels, nil, "")
	cp.Reset()

	var gotID string
	cp.SetOnSelect(func(channelID string) {
		gotID = channelID
	})

	closeCalled := false
	cp.SetOnClose(func() {
		closeCalled = true
	})

	cp.selectCurrent()

	if gotID != "C1" {
		t.Errorf("selected channel should be C1, got %q", gotID)
	}
	if !closeCalled {
		t.Error("close callback should have been called after selection")
	}
}

func TestChannelsPicker_DMSearch(t *testing.T) {
	cp := newTestPicker()

	channels := []slack.Channel{
		makePickerChannel("C1", "general", false, false, false),
	}
	dmCh := slack.Channel{}
	dmCh.ID = "D1"
	dmCh.IsIM = true
	dmCh.User = "U1"
	channels = append(channels, dmCh)

	users := map[string]slack.User{
		"U1": {
			ID:       "U1",
			Name:     "alice",
			RealName: "Alice Smith",
			Profile: slack.UserProfile{
				DisplayName: "alice",
			},
		},
	}

	cp.SetData(channels, users, "U0")

	// Search for "alice" should match the DM.
	cp.onInputChanged("alice")

	found := false
	for _, idx := range cp.filtered {
		if cp.entries[idx].channelID == "D1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("searching 'alice' should match the DM channel")
	}
}

func TestPickerSearchText(t *testing.T) {
	tests := []struct {
		name     string
		ch       slack.Channel
		chType   ChannelType
		users    map[string]slack.User
		contains string
	}{
		{
			name:     "public channel",
			ch:       makePickerChannel("C1", "general", false, false, false),
			chType:   ChannelTypePublic,
			contains: "general",
		},
		{
			name:     "private channel",
			ch:       makePickerChannel("C2", "secret", false, false, true),
			chType:   ChannelTypePrivate,
			contains: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pickerSearchText(tt.ch, tt.chType, tt.users)
			if !containsSubstring(result, tt.contains) {
				t.Errorf("search text %q should contain %q", result, tt.contains)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchContains(s, substr)
}

func searchContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
