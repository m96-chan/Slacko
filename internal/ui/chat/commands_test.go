package chat

import (
	"testing"
)

func TestParseSlashCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantCmd string
		wantArg string
	}{
		{"simple command", "/help", "help", ""},
		{"command with args", "/status :wave: hello", "status", ":wave: hello"},
		{"command uppercase", "/HELP", "help", ""},
		{"not a command", "hello world", "", ""},
		{"empty string", "", "", ""},
		{"just slash", "/", "", ""},
		{"command with whitespace", "  /topic new topic  ", "topic", "new topic"},
		{"command with multiple spaces", "/search  query text  ", "search", "query text"},
		{"leave no args", "/leave", "leave", ""},
		{"schedule with time", "/schedule in 30m meeting", "schedule", "in 30m meeting"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args := ParseSlashCommand(tt.input)
			if cmd != tt.wantCmd {
				t.Errorf("ParseSlashCommand(%q) cmd = %q, want %q", tt.input, cmd, tt.wantCmd)
			}
			if args != tt.wantArg {
				t.Errorf("ParseSlashCommand(%q) args = %q, want %q", tt.input, args, tt.wantArg)
			}
		})
	}
}

func TestBuiltinCommands(t *testing.T) {
	cmds := BuiltinCommands()
	if len(cmds) == 0 {
		t.Fatal("BuiltinCommands() returned empty slice")
	}

	// Check that all commands have required fields.
	for _, cmd := range cmds {
		if cmd.Name == "" {
			t.Error("command has empty Name")
		}
		if cmd.Description == "" {
			t.Errorf("command %q has empty Description", cmd.Name)
		}
		if cmd.Usage == "" {
			t.Errorf("command %q has empty Usage", cmd.Name)
		}
	}
}

func TestBuiltinCommandEntries(t *testing.T) {
	entries := BuiltinCommandEntries()
	cmds := BuiltinCommands()
	if len(entries) != len(cmds) {
		t.Fatalf("BuiltinCommandEntries() has %d entries, want %d", len(entries), len(cmds))
	}

	for i, e := range entries {
		if e.name != "/"+cmds[i].Name {
			t.Errorf("entry[%d].name = %q, want %q", i, e.name, "/"+cmds[i].Name)
		}
		if e.description != cmds[i].Description {
			t.Errorf("entry[%d].description = %q, want %q", i, e.description, cmds[i].Description)
		}
		if e.insertText != "/"+cmds[i].Name+" " {
			t.Errorf("entry[%d].insertText = %q, want %q", i, e.insertText, "/"+cmds[i].Name+" ")
		}
		if e.searchText == "" {
			t.Errorf("entry[%d].searchText is empty", i)
		}
	}
}
