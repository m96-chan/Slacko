package keyring

import (
	"os"

	gokeyring "github.com/zalando/go-keyring"

	"github.com/m96-chan/Slacko/internal/consts"
)

const (
	botTokenUser = "bot_token"
	appTokenUser = "app_token"
)

// GetBotToken returns the bot token from the SLACKO_BOT_TOKEN env var,
// falling back to the system keyring.
func GetBotToken() (string, error) {
	if v := os.Getenv("SLACKO_BOT_TOKEN"); v != "" {
		return v, nil
	}
	return gokeyring.Get(consts.Name, botTokenUser)
}

// GetAppToken returns the app-level token from the SLACKO_APP_TOKEN env var,
// falling back to the system keyring.
func GetAppToken() (string, error) {
	if v := os.Getenv("SLACKO_APP_TOKEN"); v != "" {
		return v, nil
	}
	return gokeyring.Get(consts.Name, appTokenUser)
}

// SetBotToken stores the bot token in the system keyring.
func SetBotToken(token string) error {
	return gokeyring.Set(consts.Name, botTokenUser, token)
}

// SetAppToken stores the app-level token in the system keyring.
func SetAppToken(token string) error {
	return gokeyring.Set(consts.Name, appTokenUser, token)
}

// DeleteBotToken removes the bot token from the system keyring.
func DeleteBotToken() error {
	return gokeyring.Delete(consts.Name, botTokenUser)
}

// DeleteAppToken removes the app-level token from the system keyring.
func DeleteAppToken() error {
	return gokeyring.Delete(consts.Name, appTokenUser)
}
