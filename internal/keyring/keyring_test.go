package keyring

import (
	"testing"

	gokeyring "github.com/zalando/go-keyring"
)

func TestSetAndGetUserToken(t *testing.T) {
	gokeyring.MockInit()

	const token = "xoxp-test-user-token"
	if err := SetUserToken(token); err != nil {
		t.Fatalf("SetUserToken: %v", err)
	}

	got, err := GetUserToken()
	if err != nil {
		t.Fatalf("GetUserToken: %v", err)
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
	if err := SetUserToken("keyring-value"); err != nil {
		t.Fatalf("SetUserToken: %v", err)
	}

	// Env var should take priority
	t.Setenv("SLACKO_USER_TOKEN", "env-value")
	got, err := GetUserToken()
	if err != nil {
		t.Fatalf("GetUserToken: %v", err)
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

func TestLegacyBotTokenEnvFallback(t *testing.T) {
	gokeyring.MockInit()

	// Legacy SLACKO_BOT_TOKEN should be picked up by GetUserToken.
	t.Setenv("SLACKO_BOT_TOKEN", "legacy-bot-value")
	got, err := GetUserToken()
	if err != nil {
		t.Fatalf("GetUserToken: %v", err)
	}
	if got != "legacy-bot-value" {
		t.Errorf("got %q, want %q", got, "legacy-bot-value")
	}

	// SLACKO_USER_TOKEN takes priority over SLACKO_BOT_TOKEN.
	t.Setenv("SLACKO_USER_TOKEN", "new-user-value")
	got, err = GetUserToken()
	if err != nil {
		t.Fatalf("GetUserToken: %v", err)
	}
	if got != "new-user-value" {
		t.Errorf("got %q, want %q", got, "new-user-value")
	}
}

func TestMissingTokenReturnsErrNotFound(t *testing.T) {
	gokeyring.MockInit()

	_, err := GetUserToken()
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

	if err := SetUserToken("to-delete"); err != nil {
		t.Fatalf("SetUserToken: %v", err)
	}
	if err := DeleteUserToken(); err != nil {
		t.Fatalf("DeleteUserToken: %v", err)
	}
	_, err := GetUserToken()
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
