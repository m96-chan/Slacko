package keyring

import (
	"testing"

	gokeyring "github.com/zalando/go-keyring"
)

func TestSetAndGetBotToken(t *testing.T) {
	gokeyring.MockInit()

	const token = "xoxb-test-bot-token"
	if err := SetBotToken(token); err != nil {
		t.Fatalf("SetBotToken: %v", err)
	}

	got, err := GetBotToken()
	if err != nil {
		t.Fatalf("GetBotToken: %v", err)
	}
	if got != token {
		t.Errorf("got %q, want %q", got, token)
	}
}

func TestSetAndGetAppToken(t *testing.T) {
	gokeyring.MockInit()

	const token = "xapp-test-app-token"
	if err := SetAppToken(token); err != nil {
		t.Fatalf("SetAppToken: %v", err)
	}

	got, err := GetAppToken()
	if err != nil {
		t.Fatalf("GetAppToken: %v", err)
	}
	if got != token {
		t.Errorf("got %q, want %q", got, token)
	}
}

func TestEnvVarFallback(t *testing.T) {
	gokeyring.MockInit()

	// Store a value in keyring
	if err := SetBotToken("keyring-value"); err != nil {
		t.Fatalf("SetBotToken: %v", err)
	}

	// Env var should take priority
	t.Setenv("SLACKO_BOT_TOKEN", "env-value")
	got, err := GetBotToken()
	if err != nil {
		t.Fatalf("GetBotToken: %v", err)
	}
	if got != "env-value" {
		t.Errorf("got %q, want %q", got, "env-value")
	}

	// Same for app token
	if err := SetAppToken("keyring-app-value"); err != nil {
		t.Fatalf("SetAppToken: %v", err)
	}
	t.Setenv("SLACKO_APP_TOKEN", "env-app-value")
	got, err = GetAppToken()
	if err != nil {
		t.Fatalf("GetAppToken: %v", err)
	}
	if got != "env-app-value" {
		t.Errorf("got %q, want %q", got, "env-app-value")
	}
}

func TestMissingTokenReturnsErrNotFound(t *testing.T) {
	gokeyring.MockInit()

	_, err := GetBotToken()
	if err != gokeyring.ErrNotFound {
		t.Errorf("got error %v, want ErrNotFound", err)
	}

	_, err = GetAppToken()
	if err != gokeyring.ErrNotFound {
		t.Errorf("got error %v, want ErrNotFound", err)
	}
}

func TestDeleteRemovesToken(t *testing.T) {
	gokeyring.MockInit()

	if err := SetBotToken("to-delete"); err != nil {
		t.Fatalf("SetBotToken: %v", err)
	}
	if err := DeleteBotToken(); err != nil {
		t.Fatalf("DeleteBotToken: %v", err)
	}
	_, err := GetBotToken()
	if err != gokeyring.ErrNotFound {
		t.Errorf("got error %v, want ErrNotFound after delete", err)
	}

	if err := SetAppToken("to-delete"); err != nil {
		t.Fatalf("SetAppToken: %v", err)
	}
	if err := DeleteAppToken(); err != nil {
		t.Fatalf("DeleteAppToken: %v", err)
	}
	_, err = GetAppToken()
	if err != gokeyring.ErrNotFound {
		t.Errorf("got error %v, want ErrNotFound after delete", err)
	}
}
