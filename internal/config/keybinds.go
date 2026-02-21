package config

// Keybinds holds all keybinding configuration. Values are plain strings
// matching the tcell.EventKey.Name() format (e.g. "Rune[j]", "Ctrl+W", "Enter").
type Keybinds struct {
	FocusChannels string `toml:"focus_channels"`
	FocusMessages string `toml:"focus_messages"`
	FocusInput    string `toml:"focus_input"`
	ToggleThread  string `toml:"toggle_thread"`
	Quit          string `toml:"quit"`
	Help          string `toml:"help"`
	SwitchTeam    string `toml:"switch_team"`
	CommandMode   string `toml:"command_mode"`

	ChannelsTree ChannelsTreeKeybinds `toml:"channels_tree"`
	MessagesList MessagesListKeybinds `toml:"messages_list"`
	MessageInput MessageInputKeybinds `toml:"message_input"`
	ThreadView   ThreadViewKeybinds   `toml:"thread_view"`
}

// ChannelsTreeKeybinds holds keybindings for the channels tree panel.
type ChannelsTreeKeybinds struct {
	Up            string `toml:"up"`
	Down          string `toml:"down"`
	Top           string `toml:"top"`
	Bottom        string `toml:"bottom"`
	SelectCurrent string `toml:"select_current"`
	Collapse      string `toml:"collapse"`
	MoveToParent  string `toml:"move_to_parent"`
}

// MessagesListKeybinds holds keybindings for the messages list panel.
type MessagesListKeybinds struct {
	SelectCurrent string `toml:"select_current"`
	ScrollUp      string `toml:"scroll_up"`
	ScrollDown    string `toml:"scroll_down"`
	Reply         string `toml:"reply"`
	Edit          string `toml:"edit"`
	Delete        string `toml:"delete"`
	Reactions     string `toml:"reactions"`
	Thread        string `toml:"thread"`
	Yank          string `toml:"yank"`
	Cancel        string `toml:"cancel"`
}

// MessageInputKeybinds holds keybindings for the message input area.
type MessageInputKeybinds struct {
	Send           string `toml:"send"`
	Newline        string `toml:"newline"`
	TabComplete    string `toml:"tab_complete"`
	OpenEditor     string `toml:"open_editor"`
	OpenFilePicker string `toml:"open_file_picker"`
	Paste          string `toml:"paste"`
	Cancel         string `toml:"cancel"`
}

// ThreadViewKeybinds holds keybindings for the thread view panel.
type ThreadViewKeybinds struct {
	Up    string `toml:"up"`
	Down  string `toml:"down"`
	Reply string `toml:"reply"`
	Close string `toml:"close"`
}
