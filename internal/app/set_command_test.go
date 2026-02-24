package app

import (
	"strings"
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

// ---------------------------------------------------------------------------
// ParseSetCommand tests
// ---------------------------------------------------------------------------

func TestParseSetCommand_AssignOn(t *testing.T) {
	opt, val, query, err := ParseSetCommand("timestamps=on")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt != "timestamps" {
		t.Errorf("option = %q, want %q", opt, "timestamps")
	}
	if val != "on" {
		t.Errorf("value = %q, want %q", val, "on")
	}
	if query {
		t.Error("query should be false")
	}
}

func TestParseSetCommand_AssignOff(t *testing.T) {
	opt, val, query, err := ParseSetCommand("mouse=off")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt != "mouse" {
		t.Errorf("option = %q, want %q", opt, "mouse")
	}
	if val != "off" {
		t.Errorf("value = %q, want %q", val, "off")
	}
	if query {
		t.Error("query should be false")
	}
}

func TestParseSetCommand_AssignTrue(t *testing.T) {
	opt, val, query, err := ParseSetCommand("markdown=true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt != "markdown" {
		t.Errorf("option = %q, want %q", opt, "markdown")
	}
	if val != "true" {
		t.Errorf("value = %q, want %q", val, "true")
	}
	if query {
		t.Error("query should be false")
	}
}

func TestParseSetCommand_AssignFalse(t *testing.T) {
	opt, val, query, err := ParseSetCommand("presence=false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt != "presence" {
		t.Errorf("option = %q, want %q", opt, "presence")
	}
	if val != "false" {
		t.Errorf("value = %q, want %q", val, "false")
	}
	if query {
		t.Error("query should be false")
	}
}

func TestParseSetCommand_AssignYes(t *testing.T) {
	opt, val, query, err := ParseSetCommand("typing=yes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt != "typing" {
		t.Errorf("option = %q, want %q", opt, "typing")
	}
	if val != "yes" {
		t.Errorf("value = %q, want %q", val, "yes")
	}
	if query {
		t.Error("query should be false")
	}
}

func TestParseSetCommand_AssignNo(t *testing.T) {
	opt, val, query, err := ParseSetCommand("date_separator=no")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt != "date_separator" {
		t.Errorf("option = %q, want %q", opt, "date_separator")
	}
	if val != "no" {
		t.Errorf("value = %q, want %q", val, "no")
	}
	if query {
		t.Error("query should be false")
	}
}

func TestParseSetCommand_Query(t *testing.T) {
	opt, val, query, err := ParseSetCommand("mouse?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt != "mouse" {
		t.Errorf("option = %q, want %q", opt, "mouse")
	}
	if val != "" {
		t.Errorf("value = %q, want empty", val)
	}
	if !query {
		t.Error("query should be true")
	}
}

func TestParseSetCommand_Toggle(t *testing.T) {
	opt, val, query, err := ParseSetCommand("timestamps")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt != "timestamps" {
		t.Errorf("option = %q, want %q", opt, "timestamps")
	}
	if val != "" {
		t.Errorf("value = %q, want empty", val)
	}
	if query {
		t.Error("query should be false for toggle")
	}
}

func TestParseSetCommand_SpaceAssign(t *testing.T) {
	opt, val, query, err := ParseSetCommand("timestamps on")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt != "timestamps" {
		t.Errorf("option = %q, want %q", opt, "timestamps")
	}
	if val != "on" {
		t.Errorf("value = %q, want %q", val, "on")
	}
	if query {
		t.Error("query should be false")
	}
}

func TestParseSetCommand_TrimsWhitespace(t *testing.T) {
	opt, val, query, err := ParseSetCommand("  mouse = on  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt != "mouse" {
		t.Errorf("option = %q, want %q", opt, "mouse")
	}
	if val != "on" {
		t.Errorf("value = %q, want %q", val, "on")
	}
	if query {
		t.Error("query should be false")
	}
}

func TestParseSetCommand_Empty(t *testing.T) {
	_, _, _, err := ParseSetCommand("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseSetCommand_InvalidValue(t *testing.T) {
	_, _, _, err := ParseSetCommand("timestamps=maybe")
	if err == nil {
		t.Error("expected error for invalid value")
	}
}

// ---------------------------------------------------------------------------
// ApplySetCommand tests -- toggle
// ---------------------------------------------------------------------------

func TestApplySetCommand_ToggleTimestamps(t *testing.T) {
	cfg := &config.Config{}
	cfg.Timestamps.Enabled = false

	msg, err := ApplySetCommand(cfg, "timestamps", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Timestamps.Enabled {
		t.Error("timestamps should be toggled to true")
	}
	if !strings.Contains(msg, "on") {
		t.Errorf("message should contain 'on', got %q", msg)
	}

	// Toggle again.
	msg, err = ApplySetCommand(cfg, "timestamps", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Timestamps.Enabled {
		t.Error("timestamps should be toggled back to false")
	}
	if !strings.Contains(msg, "off") {
		t.Errorf("message should contain 'off', got %q", msg)
	}
}

func TestApplySetCommand_ToggleMouse(t *testing.T) {
	cfg := &config.Config{Mouse: false}

	msg, err := ApplySetCommand(cfg, "mouse", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Mouse {
		t.Error("mouse should be toggled to true")
	}
	if !strings.Contains(msg, "on") {
		t.Errorf("message should contain 'on', got %q", msg)
	}
}

func TestApplySetCommand_ToggleMarkdown(t *testing.T) {
	cfg := &config.Config{}
	cfg.Markdown.Enabled = true

	_, err := ApplySetCommand(cfg, "markdown", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Markdown.Enabled {
		t.Error("markdown should be toggled to false")
	}
}

func TestApplySetCommand_ToggleTyping(t *testing.T) {
	cfg := &config.Config{}
	cfg.TypingIndicator.Enabled = false

	_, err := ApplySetCommand(cfg, "typing", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.TypingIndicator.Enabled {
		t.Error("typing should be toggled to true")
	}
}

func TestApplySetCommand_TogglePresence(t *testing.T) {
	cfg := &config.Config{}
	cfg.Presence.Enabled = true

	_, err := ApplySetCommand(cfg, "presence", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Presence.Enabled {
		t.Error("presence should be toggled to false")
	}
}

func TestApplySetCommand_ToggleDateSeparator(t *testing.T) {
	cfg := &config.Config{}
	cfg.DateSeparator.Enabled = false

	_, err := ApplySetCommand(cfg, "date_separator", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.DateSeparator.Enabled {
		t.Error("date_separator should be toggled to true")
	}
}

// ---------------------------------------------------------------------------
// ApplySetCommand tests -- explicit set
// ---------------------------------------------------------------------------

func TestApplySetCommand_SetTimestampsOn(t *testing.T) {
	cfg := &config.Config{}
	cfg.Timestamps.Enabled = false

	msg, err := ApplySetCommand(cfg, "timestamps", "on")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Timestamps.Enabled {
		t.Error("timestamps should be set to true")
	}
	if !strings.Contains(msg, "on") {
		t.Errorf("message should contain 'on', got %q", msg)
	}
}

func TestApplySetCommand_SetTimestampsOff(t *testing.T) {
	cfg := &config.Config{}
	cfg.Timestamps.Enabled = true

	msg, err := ApplySetCommand(cfg, "timestamps", "off")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Timestamps.Enabled {
		t.Error("timestamps should be set to false")
	}
	if !strings.Contains(msg, "off") {
		t.Errorf("message should contain 'off', got %q", msg)
	}
}

func TestApplySetCommand_SetMouseTrue(t *testing.T) {
	cfg := &config.Config{Mouse: false}

	_, err := ApplySetCommand(cfg, "mouse", "true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Mouse {
		t.Error("mouse should be set to true")
	}
}

func TestApplySetCommand_SetMouseFalse(t *testing.T) {
	cfg := &config.Config{Mouse: true}

	_, err := ApplySetCommand(cfg, "mouse", "false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mouse {
		t.Error("mouse should be set to false")
	}
}

func TestApplySetCommand_SetMouseYes(t *testing.T) {
	cfg := &config.Config{Mouse: false}

	_, err := ApplySetCommand(cfg, "mouse", "yes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Mouse {
		t.Error("mouse should be set to true")
	}
}

func TestApplySetCommand_SetMouseNo(t *testing.T) {
	cfg := &config.Config{Mouse: true}

	_, err := ApplySetCommand(cfg, "mouse", "no")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mouse {
		t.Error("mouse should be set to false")
	}
}

func TestApplySetCommand_UnknownOption(t *testing.T) {
	cfg := &config.Config{}

	_, err := ApplySetCommand(cfg, "nonexistent", "on")
	if err == nil {
		t.Error("expected error for unknown option")
	}
	if !strings.Contains(err.Error(), "unknown option") {
		t.Errorf("error should mention 'unknown option', got %q", err.Error())
	}
}

func TestApplySetCommand_InvalidValue(t *testing.T) {
	cfg := &config.Config{}

	_, err := ApplySetCommand(cfg, "mouse", "maybe")
	if err == nil {
		t.Error("expected error for invalid value")
	}
	if !strings.Contains(err.Error(), "invalid value") {
		t.Errorf("error should mention 'invalid value', got %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// QueryOption tests
// ---------------------------------------------------------------------------

func TestQueryOption_Mouse(t *testing.T) {
	cfg := &config.Config{Mouse: true}

	msg, err := QueryOption(cfg, "mouse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, "mouse") {
		t.Errorf("message should contain 'mouse', got %q", msg)
	}
	if !strings.Contains(msg, "on") {
		t.Errorf("message should contain 'on', got %q", msg)
	}
}

func TestQueryOption_MouseOff(t *testing.T) {
	cfg := &config.Config{Mouse: false}

	msg, err := QueryOption(cfg, "mouse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, "off") {
		t.Errorf("message should contain 'off', got %q", msg)
	}
}

func TestQueryOption_Timestamps(t *testing.T) {
	cfg := &config.Config{}
	cfg.Timestamps.Enabled = true

	msg, err := QueryOption(cfg, "timestamps")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, "timestamps") {
		t.Errorf("message should contain 'timestamps', got %q", msg)
	}
	if !strings.Contains(msg, "on") {
		t.Errorf("message should contain 'on', got %q", msg)
	}
}

func TestQueryOption_UnknownOption(t *testing.T) {
	cfg := &config.Config{}

	_, err := QueryOption(cfg, "nonexistent")
	if err == nil {
		t.Error("expected error for unknown option")
	}
}

// ---------------------------------------------------------------------------
// ListRuntimeOptions tests
// ---------------------------------------------------------------------------

func TestListRuntimeOptions_ContainsAllOptions(t *testing.T) {
	cfg := &config.Config{Mouse: true}
	cfg.Timestamps.Enabled = true
	cfg.Markdown.Enabled = false
	cfg.TypingIndicator.Enabled = true
	cfg.Presence.Enabled = false
	cfg.DateSeparator.Enabled = true

	result := ListRuntimeOptions(cfg)

	expectedOptions := []string{
		"mouse", "timestamps", "markdown", "typing", "presence", "date_separator",
	}
	for _, opt := range expectedOptions {
		if !strings.Contains(result, opt) {
			t.Errorf("result should contain %q, got %q", opt, result)
		}
	}
}

func TestListRuntimeOptions_ShowsValues(t *testing.T) {
	cfg := &config.Config{Mouse: true}
	cfg.Timestamps.Enabled = false
	cfg.Markdown.Enabled = true

	result := ListRuntimeOptions(cfg)

	// mouse should show on.
	if !strings.Contains(result, "on") {
		t.Errorf("result should contain 'on', got %q", result)
	}
	// Some options should show off.
	if !strings.Contains(result, "off") {
		t.Errorf("result should contain 'off', got %q", result)
	}
}

// ---------------------------------------------------------------------------
// RuntimeOptionNames tests
// ---------------------------------------------------------------------------

func TestRuntimeOptionNames_ReturnsAllOptions(t *testing.T) {
	names := RuntimeOptionNames()
	if len(names) == 0 {
		t.Fatal("RuntimeOptionNames() returned empty slice")
	}

	expected := map[string]bool{
		"mouse": true, "timestamps": true, "markdown": true,
		"typing": true, "presence": true, "date_separator": true,
	}
	for _, name := range names {
		if !expected[name] {
			t.Errorf("unexpected option name: %q", name)
		}
	}
	if len(names) != len(expected) {
		t.Errorf("got %d names, want %d", len(names), len(expected))
	}
}

// ---------------------------------------------------------------------------
// ParseBoolValue tests
// ---------------------------------------------------------------------------

func TestParseBoolValue_ValidValues(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"on", true},
		{"off", false},
		{"true", true},
		{"false", false},
		{"yes", true},
		{"no", false},
		{"ON", true},
		{"OFF", false},
		{"True", true},
		{"False", false},
		{"Yes", true},
		{"No", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseBoolValue(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseBoolValue(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseBoolValue_InvalidValues(t *testing.T) {
	invalids := []string{"maybe", "1", "0", "enabled", ""}
	for _, input := range invalids {
		t.Run(input, func(t *testing.T) {
			_, err := ParseBoolValue(input)
			if err == nil {
				t.Errorf("expected error for %q", input)
			}
		})
	}
}
