package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/m96-chan/Slacko/internal/config"
)

func TestNewChannelInfoPanel(t *testing.T) {
	ci := NewChannelInfoPanel(&config.Config{})
	if ci == nil {
		t.Fatal("NewChannelInfoPanel returned nil")
	}
}

func TestChannelInfoPanelSetData(t *testing.T) {
	ci := NewChannelInfoPanel(&config.Config{})
	ci.SetData(ChannelInfoData{
		ChannelID:   "C123",
		Name:        "general",
		Description: "Main channel",
		Topic:       "Today's topic",
		Purpose:     "General discussion",
		Creator:     "alice",
		Created:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		NumMembers:  42,
		NumPins:     3,
	})

	text := ci.content.GetText(false)
	if !strings.Contains(text, "general") {
		t.Error("missing channel name")
	}
	if !strings.Contains(text, "Main channel") {
		t.Error("missing description")
	}
	if !strings.Contains(text, "Today's topic") {
		t.Error("missing topic")
	}
	if !strings.Contains(text, "General discussion") {
		t.Error("missing purpose")
	}
	if !strings.Contains(text, "alice") {
		t.Error("missing creator")
	}
	if !strings.Contains(text, "42") {
		t.Error("missing member count")
	}
	if !strings.Contains(text, "3") {
		t.Error("missing pin count")
	}
	if !strings.Contains(text, "Public Channel") {
		t.Error("missing channel type")
	}
}

func TestChannelInfoPanelPrivateChannel(t *testing.T) {
	ci := NewChannelInfoPanel(&config.Config{})
	ci.SetData(ChannelInfoData{
		Name:      "secret",
		IsPrivate: true,
	})

	text := ci.content.GetText(false)
	if !strings.Contains(text, "Private Channel") {
		t.Error("expected Private Channel type")
	}
}

func TestChannelInfoPanelDM(t *testing.T) {
	ci := NewChannelInfoPanel(&config.Config{})
	ci.SetData(ChannelInfoData{
		Name: "alice",
		IsDM: true,
	})

	text := ci.content.GetText(false)
	if !strings.Contains(text, "Direct Message") {
		t.Error("expected Direct Message type")
	}
}

func TestChannelInfoPanelArchived(t *testing.T) {
	ci := NewChannelInfoPanel(&config.Config{})
	ci.SetData(ChannelInfoData{
		Name:       "old-channel",
		IsArchived: true,
	})

	text := ci.content.GetText(false)
	if !strings.Contains(text, "archived") {
		t.Error("missing archived indicator")
	}
}

func TestChannelInfoPanelData(t *testing.T) {
	ci := NewChannelInfoPanel(&config.Config{})
	data := ChannelInfoData{ChannelID: "C123", Name: "general"}
	ci.SetData(data)

	got := ci.Data()
	if got.ChannelID != "C123" {
		t.Errorf("Data().ChannelID = %q, want %q", got.ChannelID, "C123")
	}
}

func TestChannelInfoPanelReset(t *testing.T) {
	ci := NewChannelInfoPanel(&config.Config{})
	ci.SetData(ChannelInfoData{ChannelID: "C123", Name: "general"})
	ci.Reset()

	if ci.data.ChannelID != "" {
		t.Errorf("after Reset, ChannelID = %q, want empty", ci.data.ChannelID)
	}
	if ci.content.GetText(false) != "" {
		t.Errorf("after Reset, content not empty")
	}
}

func TestChannelInfoPanelSetStatus(t *testing.T) {
	ci := NewChannelInfoPanel(&config.Config{})
	ci.SetStatus("Loading...")
	got := ci.status.GetText(false)
	if got != " Loading..." {
		t.Errorf("status = %q, want %q", got, " Loading...")
	}
}

func TestChannelInfoPanelCallbacks(t *testing.T) {
	ci := NewChannelInfoPanel(&config.Config{})

	closeCalled := false
	var topicCh, purposeCh, leaveCh string

	ci.SetOnClose(func() { closeCalled = true })
	ci.SetOnSetTopic(func(id string) { topicCh = id })
	ci.SetOnSetPurpose(func(id string) { purposeCh = id })
	ci.SetOnLeave(func(id string) { leaveCh = id })

	ci.onClose()
	if !closeCalled {
		t.Error("onClose not called")
	}

	ci.onSetTopic("C1")
	if topicCh != "C1" {
		t.Errorf("onSetTopic got %q, want %q", topicCh, "C1")
	}

	ci.onSetPurpose("C2")
	if purposeCh != "C2" {
		t.Errorf("onSetPurpose got %q, want %q", purposeCh, "C2")
	}

	ci.onLeave("C3")
	if leaveCh != "C3" {
		t.Errorf("onLeave got %q, want %q", leaveCh, "C3")
	}
}
