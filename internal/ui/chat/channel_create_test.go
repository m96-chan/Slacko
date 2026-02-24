package chat

import (
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func TestNewChannelCreateForm(t *testing.T) {
	f := NewChannelCreateForm(&config.Config{})
	if f == nil {
		t.Fatal("NewChannelCreateForm returned nil")
	}
	if f.form == nil {
		t.Fatal("form field is nil")
	}
	if f.nameInput == nil {
		t.Fatal("nameInput field is nil")
	}
}

func TestChannelCreateFormReset(t *testing.T) {
	f := NewChannelCreateForm(&config.Config{})

	// Set some values.
	f.nameInput.SetText("test-channel")
	f.privateCheck = true

	f.Reset()

	// After reset, fields should be cleared.
	if f.nameInput.GetText() != "" {
		t.Errorf("after Reset, name = %q, want empty", f.nameInput.GetText())
	}
	if f.privateCheck {
		t.Error("after Reset, privateCheck should be false")
	}
}

func TestChannelCreateFormCallbacks(t *testing.T) {
	f := NewChannelCreateForm(&config.Config{})

	var createName string
	var createPrivate bool
	closeCalled := false

	f.SetOnCreate(func(name string, isPrivate bool) {
		createName = name
		createPrivate = isPrivate
	})
	f.SetOnClose(func() { closeCalled = true })

	if f.onCreate == nil {
		t.Error("onCreate not set")
	}
	if f.onClose == nil {
		t.Error("onClose not set")
	}

	// Trigger callbacks directly.
	f.onClose()
	if !closeCalled {
		t.Error("onClose not called")
	}

	f.onCreate("my-channel", true)
	if createName != "my-channel" {
		t.Errorf("onCreate name = %q, want %q", createName, "my-channel")
	}
	if !createPrivate {
		t.Error("onCreate isPrivate = false, want true")
	}
}

func TestChannelCreateFormGetName(t *testing.T) {
	f := NewChannelCreateForm(&config.Config{})

	// Initially empty.
	if f.GetName() != "" {
		t.Errorf("initial name = %q, want empty", f.GetName())
	}

	// Set a name.
	f.nameInput.SetText("new-channel")
	if f.GetName() != "new-channel" {
		t.Errorf("name = %q, want %q", f.GetName(), "new-channel")
	}
}

func TestChannelCreateFormIsPrivate(t *testing.T) {
	f := NewChannelCreateForm(&config.Config{})

	// Initially false.
	if f.IsPrivate() {
		t.Error("initial IsPrivate = true, want false")
	}

	// Set private.
	f.privateCheck = true
	if !f.IsPrivate() {
		t.Error("IsPrivate = false after setting, want true")
	}
}

func TestChannelCreateFormSetStatus(t *testing.T) {
	f := NewChannelCreateForm(&config.Config{})

	f.SetStatus("Creating...")
	got := f.status.GetText(false)
	if got != " Creating..." {
		t.Errorf("status = %q, want %q", got, " Creating...")
	}
}

func TestChannelCreateFormValidation(t *testing.T) {
	f := NewChannelCreateForm(&config.Config{})

	// Empty name should not be valid.
	if f.validate() {
		t.Error("validate() = true for empty name, want false")
	}

	// Set a valid name.
	f.nameInput.SetText("valid-channel")
	if !f.validate() {
		t.Error("validate() = false for valid name, want true")
	}

	// Name with spaces should not be valid (Slack doesn't allow spaces).
	f.nameInput.SetText("has spaces")
	if f.validate() {
		t.Error("validate() = true for name with spaces, want false")
	}

	// Name with uppercase should not be valid (Slack requires lowercase).
	f.nameInput.SetText("HasUpperCase")
	if f.validate() {
		t.Error("validate() = true for name with uppercase, want false")
	}

	// Name with hyphens and numbers is valid.
	f.nameInput.SetText("my-channel-123")
	if !f.validate() {
		t.Error("validate() = false for valid hyphenated name, want true")
	}

	// Name with underscores is valid.
	f.nameInput.SetText("my_channel")
	if !f.validate() {
		t.Error("validate() = false for valid underscored name, want true")
	}
}
