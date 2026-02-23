package config

import (
	"testing"
)

func TestMakeStyle_Tag(t *testing.T) {
	tests := []struct {
		name string
		fg   string
		bg   string
		attr string
		want string
	}{
		{"fg only", "green", "", "", "[green]"},
		{"fg+attr", "green", "", "b", "[green:-:b]"},
		{"fg+bg+attr", "green", "black", "b", "[green:black:b]"},
		{"fg+bg", "white", "blue", "", "[white:blue:-]"},
		{"empty", "", "", "", "[-]"},
		{"attr only", "", "", "d", "[-:-:d]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := makeStyle(tt.fg, tt.bg, tt.attr)
			got := s.Tag()
			if got != tt.want {
				t.Errorf("makeStyle(%q,%q,%q).Tag() = %q, want %q", tt.fg, tt.bg, tt.attr, got, tt.want)
			}
		})
	}
}

func TestStyleWrapper_Reset(t *testing.T) {
	tests := []struct {
		name string
		fg   string
		bg   string
		attr string
		want string
	}{
		{"fg only", "green", "", "", "[-]"},
		{"fg+attr", "green", "", "b", "[-::-]"},
		{"fg+bg+attr", "green", "black", "b", "[-::-]"},
		{"empty", "", "", "", "[-]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := makeStyle(tt.fg, tt.bg, tt.attr)
			got := s.Reset()
			if got != tt.want {
				t.Errorf("Reset() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAttrsToTviewString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"bold", "b"},
		{"bold|underline", "bu"},
		{"dim|italic", "di"},
		{"bold|italic|underline|dim|reverse|blink|strikethrough", "biudrls"},
		{"", ""},
		{"none", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := attrsToTviewString(tt.input)
			if got != tt.want {
				t.Errorf("attrsToTviewString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuiltinTheme_Default(t *testing.T) {
	theme := BuiltinTheme("default")
	if theme.Preset != "default" {
		t.Errorf("expected preset=default, got %q", theme.Preset)
	}
	// Spot-check: author should be green+bold.
	if theme.MessagesList.Author.Tag() != "[green:-:b]" {
		t.Errorf("author tag = %q, want [green:-:b]", theme.MessagesList.Author.Tag())
	}
	// Markdown link should be green+underline.
	if theme.Markdown.Link.Tag() != "[green:-:u]" {
		t.Errorf("link tag = %q, want [green:-:u]", theme.Markdown.Link.Tag())
	}
}

func TestBuiltinTheme_UnknownFallsBackToDefault(t *testing.T) {
	theme := BuiltinTheme("nonexistent")
	def := BuiltinTheme("default")
	if theme.MessagesList.Author.Tag() != def.MessagesList.Author.Tag() {
		t.Error("unknown preset should fall back to default")
	}
}

func TestBuiltinTheme_AllPresetsPopulated(t *testing.T) {
	presets := []string{"default", "dark", "light", "monokai", "solarized_dark", "solarized_light", "high_contrast", "monochrome"}
	for _, name := range presets {
		t.Run(name, func(t *testing.T) {
			theme := BuiltinTheme(name)
			if theme.Preset != name {
				t.Errorf("preset = %q, want %q", theme.Preset, name)
			}
			// Ensure critical fields are non-empty.
			if theme.MessagesList.Author.Tag() == "[-]" {
				t.Error("author tag should not be empty default")
			}
			if theme.Markdown.UserMention.Tag() == "[-]" {
				t.Error("user mention tag should not be empty default")
			}
		})
	}
}

func TestMakeStyle_ForegroundBackground(t *testing.T) {
	s := makeStyle("green", "blue", "b")
	fg := s.Foreground()
	bg := s.Background()
	if fg == 0 {
		t.Error("foreground should be set")
	}
	if bg == 0 {
		t.Error("background should be set")
	}
}
