package chat

import (
	"strings"
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func TestNewUserProfilePanel(t *testing.T) {
	up := NewUserProfilePanel(&config.Config{})
	if up == nil {
		t.Fatal("NewUserProfilePanel returned nil")
	}
}

func TestUserProfilePanelSetData(t *testing.T) {
	up := NewUserProfilePanel(&config.Config{})
	up.SetData(UserProfileData{
		UserID:      "U123",
		DisplayName: "alice",
		RealName:    "Alice Smith",
		Title:       "Engineer",
		StatusEmoji: ":wave:",
		StatusText:  "Working from home",
		Email:       "alice@example.com",
		Phone:       "+1234567890",
		Presence:    "active",
	})

	text := up.content.GetText(false)
	if !strings.Contains(text, "alice") {
		t.Error("missing display name")
	}
	if !strings.Contains(text, "Alice Smith") {
		t.Error("missing real name")
	}
	if !strings.Contains(text, "Engineer") {
		t.Error("missing title")
	}
	if !strings.Contains(text, "Working from home") {
		t.Error("missing status")
	}
	if !strings.Contains(text, "alice@example.com") {
		t.Error("missing email")
	}
	if !strings.Contains(text, "+1234567890") {
		t.Error("missing phone")
	}
}

func TestUserProfilePanelBot(t *testing.T) {
	up := NewUserProfilePanel(&config.Config{})
	up.SetData(UserProfileData{
		UserID:      "U456",
		DisplayName: "bot",
		IsBot:       true,
	})

	text := up.content.GetText(false)
	if !strings.Contains(text, "BOT") {
		t.Error("missing BOT badge")
	}
}

func TestUserProfilePanelAdmin(t *testing.T) {
	up := NewUserProfilePanel(&config.Config{})
	up.SetData(UserProfileData{
		UserID:      "U789",
		DisplayName: "admin",
		IsAdmin:     true,
	})

	text := up.content.GetText(false)
	if !strings.Contains(text, "Admin") {
		t.Error("missing Admin badge")
	}
}

func TestUserProfilePanelOwner(t *testing.T) {
	up := NewUserProfilePanel(&config.Config{})
	up.SetData(UserProfileData{
		UserID:      "U000",
		DisplayName: "owner",
		IsOwner:     true,
	})

	text := up.content.GetText(false)
	if !strings.Contains(text, "Owner") {
		t.Error("missing Owner badge")
	}
}

func TestUserProfilePanelFallbackName(t *testing.T) {
	up := NewUserProfilePanel(&config.Config{})

	// Falls back to RealName when DisplayName is empty.
	up.SetData(UserProfileData{UserID: "U1", RealName: "Bob"})
	text := up.content.GetText(false)
	if !strings.Contains(text, "Bob") {
		t.Error("expected RealName fallback")
	}

	// Falls back to UserID when both are empty.
	up.SetData(UserProfileData{UserID: "U2"})
	text = up.content.GetText(false)
	if !strings.Contains(text, "U2") {
		t.Error("expected UserID fallback")
	}
}

func TestUserProfilePanelData(t *testing.T) {
	up := NewUserProfilePanel(&config.Config{})
	data := UserProfileData{UserID: "U123", DisplayName: "alice"}
	up.SetData(data)

	got := up.Data()
	if got.UserID != "U123" {
		t.Errorf("Data().UserID = %q, want %q", got.UserID, "U123")
	}
}

func TestUserProfilePanelReset(t *testing.T) {
	up := NewUserProfilePanel(&config.Config{})
	up.SetData(UserProfileData{UserID: "U123", DisplayName: "alice"})
	up.Reset()

	if up.data.UserID != "" {
		t.Errorf("after Reset, UserID = %q, want empty", up.data.UserID)
	}
	if up.content.GetText(false) != "" {
		t.Errorf("after Reset, content not empty")
	}
}

func TestUserProfilePanelSetStatus(t *testing.T) {
	up := NewUserProfilePanel(&config.Config{})
	up.SetStatus("Loading...")
	got := up.status.GetText(false)
	if got != " Loading..." {
		t.Errorf("status = %q, want %q", got, " Loading...")
	}
}

func TestUserProfilePanelCallbacks(t *testing.T) {
	up := NewUserProfilePanel(&config.Config{})

	closeCalled := false
	var dmUser, copyUser string

	up.SetOnClose(func() { closeCalled = true })
	up.SetOnOpenDM(func(id string) { dmUser = id })
	up.SetOnCopyID(func(id string) { copyUser = id })

	up.onClose()
	if !closeCalled {
		t.Error("onClose not called")
	}

	up.onOpenDM("U123")
	if dmUser != "U123" {
		t.Errorf("onOpenDM got %q, want %q", dmUser, "U123")
	}

	up.onCopyID("U456")
	if copyUser != "U456" {
		t.Errorf("onCopyID got %q, want %q", copyUser, "U456")
	}
}
