package keyring

import (
	"os"

	gokeyring "github.com/zalando/go-keyring"

	"github.com/m96-chan/Slacko/internal/consts"
)

const (
	userTokenUser = "user_token"
	appTokenUser  = "app_token"

	// Legacy keyring key for backward compatibility.
	legacyBotTokenUser = "bot_token"
)

// GetUserToken returns the user token from the SLACKO_USER_TOKEN env var,
// falling back to SLACKO_BOT_TOKEN (legacy), then the system keyring.
func GetUserToken() (string, error) {
	if v := os.Getenv("SLACKO_USER_TOKEN"); v != "" {
		return v, nil
	}
	// Legacy env var fallback.
	if v := os.Getenv("SLACKO_BOT_TOKEN"); v != "" {
		return v, nil
	}
	// Try new keyring key first, then legacy.
	tok, err := gokeyring.Get(consts.Name, userTokenUser)
	if err == nil {
		return tok, nil
	}
	return gokeyring.Get(consts.Name, legacyBotTokenUser)
}

// GetAppToken returns the app-level token from the SLACKO_APP_TOKEN env var,
// falling back to the system keyring.
func GetAppToken() (string, error) {
	if v := os.Getenv("SLACKO_APP_TOKEN"); v != "" {
		return v, nil
	}
	return gokeyring.Get(consts.Name, appTokenUser)
}

// SetUserToken stores the user token in the system keyring.
func SetUserToken(token string) error {
	return gokeyring.Set(consts.Name, userTokenUser, token)
}

// SetAppToken stores the app-level token in the system keyring.
func SetAppToken(token string) error {
	return gokeyring.Set(consts.Name, appTokenUser, token)
}

// DeleteUserToken removes the user token from the system keyring.
func DeleteUserToken() error {
	return gokeyring.Delete(consts.Name, userTokenUser)
}

// DeleteAppToken removes the app-level token from the system keyring.
func DeleteAppToken() error {
	return gokeyring.Delete(consts.Name, appTokenUser)
}
