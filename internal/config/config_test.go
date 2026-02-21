package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestDefaultPath(t *testing.T) {
	p := DefaultPath()
	if p == "" {
		t.Fatal("DefaultPath returned empty string")
	}
	if filepath.Base(p) != "config.toml" {
		t.Errorf("DefaultPath should end with config.toml, got %s", p)
	}
}

func TestLoadMissingFileWritesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.toml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// File should have been created.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file was not created: %v", err)
	}

	// Should have default values.
	if !cfg.Mouse {
		t.Error("expected mouse=true from defaults")
	}
	if cfg.MessagesLimit != 50 {
		t.Errorf("expected messages_limit=50, got %d", cfg.MessagesLimit)
	}
	if !cfg.Timestamps.Enabled {
		t.Error("expected timestamps.enabled=true from defaults")
	}
	if cfg.Keybinds.Quit != "Ctrl+C" {
		t.Errorf("expected keybinds.quit=Ctrl+C, got %s", cfg.Keybinds.Quit)
	}
}

func TestLoadPartialOverridePreservesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Write a partial config that only overrides messages_limit.
	partial := []byte("messages_limit = 25\n")
	if err := os.WriteFile(path, partial, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Overridden value should apply.
	if cfg.MessagesLimit != 25 {
		t.Errorf("expected messages_limit=25, got %d", cfg.MessagesLimit)
	}

	// Defaults should be preserved.
	if !cfg.Mouse {
		t.Error("expected mouse=true from defaults (not overridden)")
	}
	if cfg.Timestamps.Format != "3:04PM" {
		t.Errorf("expected timestamps.format=3:04PM, got %s", cfg.Timestamps.Format)
	}
	if cfg.Keybinds.ChannelsTree.Up != "Rune[k]" {
		t.Errorf("expected keybinds.channels_tree.up=Rune[k], got %s", cfg.Keybinds.ChannelsTree.Up)
	}
}

func TestValidationRejectsOutOfRange(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{"messages_limit too low", "messages_limit = 0\n"},
		{"messages_limit too high", "messages_limit = 200\n"},
		{"autocomplete_limit negative", "autocomplete_limit = -1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.toml")
			if err := os.WriteFile(path, []byte(tt.config), 0o600); err != nil {
				t.Fatal(err)
			}

			_, err := Load(path)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestInvalidTOMLErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("not valid [[ toml"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid TOML, got nil")
	}
}

func TestEditorResolution(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Set editor to "default" and EDITOR env var.
	content := []byte("editor = \"default\"\nmessages_limit = 50\nautocomplete_limit = 10\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("EDITOR", "nano")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Editor != "nano" {
		t.Errorf("expected editor=nano from $EDITOR, got %s", cfg.Editor)
	}
}

func TestEditorFallbackToVi(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	content := []byte("editor = \"default\"\nmessages_limit = 50\nautocomplete_limit = 10\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("EDITOR", "")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Editor != "vi" {
		t.Errorf("expected editor=vi fallback, got %s", cfg.Editor)
	}
}

func TestEmbeddedConfigIsValidTOML(t *testing.T) {
	var cfg Config
	if err := toml.Unmarshal(defaultConfig, &cfg); err != nil {
		t.Fatalf("embedded config.toml is not valid TOML: %v", err)
	}
}
