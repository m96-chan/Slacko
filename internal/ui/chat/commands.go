package chat

import "strings"

// SlashCommand defines a slash command with its metadata.
type SlashCommand struct {
	Name        string // e.g. "help"
	Description string // e.g. "Show available commands"
	Usage       string // e.g. "/help"
}

// builtinCommands is the list of supported slash commands.
var builtinCommands = []SlashCommand{
	{Name: "help", Description: "Show available commands", Usage: "/help"},
	{Name: "status", Description: "Set your status", Usage: "/status [:emoji:] [text]"},
	{Name: "clear-status", Description: "Clear your status", Usage: "/clear-status"},
	{Name: "topic", Description: "Set channel topic", Usage: "/topic [text]"},
	{Name: "leave", Description: "Leave current channel", Usage: "/leave"},
	{Name: "search", Description: "Search messages", Usage: "/search [query]"},
	{Name: "who", Description: "List channel members", Usage: "/who"},
	{Name: "mute", Description: "Mute channel notifications", Usage: "/mute"},
	{Name: "unmute", Description: "Unmute channel", Usage: "/unmute"},
	{Name: "schedule", Description: "Schedule a message", Usage: "/schedule [time] [message]"},
	{Name: "scheduled", Description: "List scheduled messages", Usage: "/scheduled"},
	{Name: "remind", Description: "Set a reminder", Usage: "/remind [what] [when]"},
	{Name: "reminders", Description: "List active reminders", Usage: "/reminders"},
	{Name: "me", Description: "Send an action message", Usage: "/me [action]"},
	{Name: "create-channel", Description: "Create a new channel", Usage: "/create-channel"},
	{Name: "logout", Description: "Log out and clear tokens", Usage: "/logout"},
}

// BuiltinCommands returns the list of builtin slash commands.
func BuiltinCommands() []SlashCommand {
	return builtinCommands
}

// BuiltinCommandEntries returns command entries for the autocomplete system.
func BuiltinCommandEntries() []commandEntry {
	entries := make([]commandEntry, len(builtinCommands))
	for i, cmd := range builtinCommands {
		entries[i] = commandEntry{
			name:        "/" + cmd.Name,
			description: cmd.Description,
			searchText:  strings.ToLower(cmd.Name),
			insertText:  "/" + cmd.Name + " ",
		}
	}
	return entries
}

// ParseSlashCommand parses a slash command string into command name and args.
// Returns ("", "") if the text is not a slash command.
func ParseSlashCommand(text string) (command, args string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", ""
	}

	text = text[1:] // strip leading /
	parts := strings.SplitN(text, " ", 2)
	command = strings.ToLower(parts[0])
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}
	return command, args
}
