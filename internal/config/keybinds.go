package config

// Keybinds holds all keybinding configuration. Values are plain strings
// matching the tcell.EventKey.Name() format (e.g. "Rune[j]", "Ctrl+W", "Enter").
type Keybinds struct {
	FocusChannels  string `toml:"focus_channels"`
	FocusMessages  string `toml:"focus_messages"`
	FocusInput     string `toml:"focus_input"`
	ToggleThread   string `toml:"toggle_thread"`
	ToggleChannels string `toml:"toggle_channels"`
	ChannelPicker  string `toml:"channel_picker"`
	Search         string `toml:"search"`
	Quit           string `toml:"quit"`
	Help           string `toml:"help"`
	SwitchTeam     string `toml:"switch_team"`
	CommandMode    string `toml:"command_mode"`
	MarkRead        string `toml:"mark_read"`
	MarkAllRead     string `toml:"mark_all_read"`
	PinnedMessages  string `toml:"pinned_messages"`

	ChannelsTree   ChannelsTreeKeybinds   `toml:"channels_tree"`
	MessagesList   MessagesListKeybinds   `toml:"messages_list"`
	MessageInput   MessageInputKeybinds   `toml:"message_input"`
	ThreadView     ThreadViewKeybinds     `toml:"thread_view"`
	ChannelsPicker ChannelsPickerKeybinds `toml:"channels_picker"`
	FilePicker     FilePickerKeybinds     `toml:"file_picker"`
	SearchPicker   SearchPickerKeybinds   `toml:"search_picker"`
	PinsPicker     PinsPickerKeybinds     `toml:"pins_picker"`
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
	Up              string `toml:"up"`
	Down            string `toml:"down"`
	SelectCurrent   string `toml:"select_current"`
	ScrollUp        string `toml:"scroll_up"`
	ScrollDown      string `toml:"scroll_down"`
	Reply           string `toml:"reply"`
	Edit            string `toml:"edit"`
	Delete          string `toml:"delete"`
	Reactions       string `toml:"reactions"`
	RemoveReaction  string `toml:"remove_reaction"`
	Thread          string `toml:"thread"`
	Yank            string `toml:"yank"`
	OpenFile        string `toml:"open_file"`
	Pin             string `toml:"pin"`
	Cancel          string `toml:"cancel"`
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

// ChannelsPickerKeybinds holds keybindings for the channel picker popup.
type ChannelsPickerKeybinds struct {
	Close  string `toml:"close"`
	Up     string `toml:"up"`
	Down   string `toml:"down"`
	Select string `toml:"select"`
}

// FilePickerKeybinds holds keybindings for the file picker popup.
type FilePickerKeybinds struct {
	Close  string `toml:"close"`
	Up     string `toml:"up"`
	Down   string `toml:"down"`
	Select string `toml:"select"`
}

// SearchPickerKeybinds holds keybindings for the search picker popup.
type SearchPickerKeybinds struct {
	Close  string `toml:"close"`
	Up     string `toml:"up"`
	Down   string `toml:"down"`
	Select string `toml:"select"`
}

// PinsPickerKeybinds holds keybindings for the pinned messages picker popup.
type PinsPickerKeybinds struct {
	Close  string `toml:"close"`
	Up     string `toml:"up"`
	Down   string `toml:"down"`
	Select string `toml:"select"`
}
