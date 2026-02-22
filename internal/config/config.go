package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/m96-chan/Slacko/internal/consts"
)

//go:embed config.toml
var defaultConfig []byte

// OAuthConfig holds OAuth credentials for browser-based login.
// ProxyURL is used for public distribution (Cloudflare Worker holds the secret).
// ClientSecret is used for self-hosted setups (direct exchange with Slack).
type OAuthConfig struct {
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
	AppToken     string `toml:"app_token"`
	ProxyURL     string `toml:"proxy_url"`
}

// Config holds the application configuration.
type Config struct {
	Mouse               bool   `toml:"mouse"`
	Editor              string `toml:"editor"`
	AutoFocus           bool   `toml:"auto_focus"`
	ShowAttachmentLinks bool   `toml:"show_attachment_links"`
	AutocompleteLimit   int    `toml:"autocomplete_limit"`
	MessagesLimit       int    `toml:"messages_limit"`
	DownloadDir         string `toml:"download_dir"`

	AsciiIcons bool `toml:"ascii_icons"`

	Markdown        MarkdownConfig  `toml:"markdown"`
	Timestamps      Timestamps      `toml:"timestamps"`
	DateSeparator   DateSeparator   `toml:"date_separator"`
	Notifications   Notifications   `toml:"notifications"`
	TypingIndicator TypingIndicator `toml:"typing_indicator"`
	Threads         Threads         `toml:"threads"`
	Presence        Presence        `toml:"presence"`
	OAuth           OAuthConfig     `toml:"oauth"`

	Keybinds Keybinds `toml:"keybinds"`
	Theme    Theme    `toml:"theme"`
}

// MarkdownConfig controls markdown rendering in messages.
type MarkdownConfig struct {
	Enabled     bool   `toml:"enabled"`
	SyntaxTheme string `toml:"syntax_theme"`
}

// Timestamps controls message timestamp display.
type Timestamps struct {
	Enabled bool   `toml:"enabled"`
	Format  string `toml:"format"`
}

// DateSeparator controls date separator lines between messages.
type DateSeparator struct {
	Enabled   bool   `toml:"enabled"`
	Character string `toml:"character"`
}

// Notifications controls desktop notification behavior.
type Notifications struct {
	Enabled bool              `toml:"enabled"`
	Sound   NotificationSound `toml:"sound"`
}

// NotificationSound controls notification sound behavior.
type NotificationSound struct {
	Enabled bool `toml:"enabled"`
}

// TypingIndicator controls typing indicator behavior.
type TypingIndicator struct {
	Enabled bool `toml:"enabled"`
	Send    bool `toml:"send"`
	Receive bool `toml:"receive"`
}

// Threads controls thread display behavior.
type Threads struct {
	ShowInline bool `toml:"show_inline"`
}

// Presence controls user presence display.
type Presence struct {
	Enabled bool `toml:"enabled"`
}

// DefaultPath returns the default config file path.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, consts.Name, "config.toml")
}

// Load reads the config from the given path. If the file does not exist,
// it writes the default config and loads that. Config loading is two-phase:
// embedded defaults are applied first, then the user file overlays on top.
func Load(path string) (*Config, error) {
	// Phase 1: unmarshal embedded defaults.
	var cfg Config
	if err := toml.Unmarshal(defaultConfig, &cfg); err != nil {
		return nil, fmt.Errorf("parsing embedded config: %w", err)
	}

	// Write default config if file does not exist.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, defaultConfig, 0o600); err != nil {
			return nil, err
		}
	}

	// Phase 2: overlay user file on top of defaults.
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Phase 3: resolve theme preset — load the built-in preset as the base,
	// then re-decode the user's [theme] section on top.
	cfg.Theme = resolveTheme(path, cfg.Theme)

	applyDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return &cfg, nil
}

// applyDefaults resolves computed defaults that can't be expressed in TOML.
func applyDefaults(cfg *Config) {
	// Resolve "default" editor to $EDITOR or "vi".
	if cfg.Editor == "default" {
		if env := os.Getenv("EDITOR"); env != "" {
			cfg.Editor = env
		} else {
			cfg.Editor = "vi"
		}
	}

	// Ensure date separator character is set.
	if cfg.DateSeparator.Character == "" {
		cfg.DateSeparator.Character = "─"
	}

	// Resolve download directory.
	if cfg.DownloadDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.TempDir()
		}
		cfg.DownloadDir = filepath.Join(home, "Downloads")
	}

	// OAuth env var overrides.
	if v := os.Getenv("SLACKO_CLIENT_ID"); v != "" {
		cfg.OAuth.ClientID = v
	}
	if v := os.Getenv("SLACKO_CLIENT_SECRET"); v != "" {
		cfg.OAuth.ClientSecret = v
	}
	if v := os.Getenv("SLACKO_APP_TOKEN"); v != "" {
		cfg.OAuth.AppToken = v
	}
	if v := os.Getenv("SLACKO_PROXY_URL"); v != "" {
		cfg.OAuth.ProxyURL = v
	}
}

// validate checks that config values are within acceptable ranges.
func validate(cfg *Config) error {
	if cfg.MessagesLimit < 1 || cfg.MessagesLimit > 100 {
		return fmt.Errorf("messages_limit must be between 1 and 100, got %d", cfg.MessagesLimit)
	}
	if cfg.AutocompleteLimit < 0 {
		return fmt.Errorf("autocomplete_limit must be >= 0, got %d", cfg.AutocompleteLimit)
	}
	return nil
}

// resolveTheme loads the built-in preset as a base theme, then re-decodes the
// user's config file on top so that only user-specified fields override the preset.
func resolveTheme(configPath string, userTheme Theme) Theme {
	base := BuiltinTheme(userTheme.Preset)
	base.Preset = userTheme.Preset

	data, err := os.ReadFile(configPath)
	if err != nil {
		return base
	}

	type themeOnly struct {
		Theme Theme `toml:"theme"`
	}
	wrapper := themeOnly{Theme: base}
	if _, err := toml.NewDecoder(bytes.NewReader(data)).Decode(&wrapper); err == nil {
		wrapper.Theme.Preset = userTheme.Preset
		return wrapper.Theme
	}
	return base
}
