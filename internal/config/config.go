package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/m96-chan/Slacko/internal/consts"
)

//go:embed config.toml
var defaultConfig []byte

// Config holds the application configuration.
type Config struct {
	Mouse               bool   `toml:"mouse"`
	Editor              string `toml:"editor"`
	AutoFocus           bool   `toml:"auto_focus"`
	ShowAttachmentLinks bool   `toml:"show_attachment_links"`
	AutocompleteLimit   int    `toml:"autocomplete_limit"`
	MessagesLimit       int    `toml:"messages_limit"`

	Markdown       MarkdownConfig  `toml:"markdown"`
	Timestamps     Timestamps      `toml:"timestamps"`
	DateSeparator  DateSeparator   `toml:"date_separator"`
	Notifications  Notifications   `toml:"notifications"`
	TypingIndicator TypingIndicator `toml:"typing_indicator"`
	Threads        Threads         `toml:"threads"`
	Presence       Presence        `toml:"presence"`

	Keybinds Keybinds `toml:"keybinds"`
	Theme    Theme    `toml:"theme"`
}

// MarkdownConfig controls markdown rendering in messages.
type MarkdownConfig struct {
	Enabled bool `toml:"enabled"`
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

// TypingIndicator controls the typing indicator display.
type TypingIndicator struct {
	Enabled bool `toml:"enabled"`
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
