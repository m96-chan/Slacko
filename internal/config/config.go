package config

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/m96-chan/Slacko/internal/consts"
)

//go:embed config.toml
var defaultConfig string

// Config holds the application configuration.
type Config struct {
	Mouse             bool   `toml:"mouse"`
	Editor            string `toml:"editor"`
	TimestampsEnabled bool   `toml:"timestamps_enabled"`
	TimestampsFormat  string `toml:"timestamps_format"`
	MessagesLimit     int    `toml:"messages_limit"`
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
// it writes the default config and loads that.
func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(defaultConfig), 0o600); err != nil {
			return nil, err
		}
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
