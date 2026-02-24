package chat

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
)

// VimCommand defines a vim-style command with metadata.
type VimCommand struct {
	Name        string
	Aliases     []string
	Description string
}

// builtinVimCommands is the list of supported vim-style commands.
var builtinVimCommands = []VimCommand{
	{Name: "q", Aliases: []string{"quit"}, Description: "Quit application"},
	{Name: "theme", Description: "Switch theme"},
	{Name: "join", Description: "Join channel"},
	{Name: "leave", Description: "Leave current channel"},
	{Name: "search", Description: "Search messages"},
	{Name: "mark-read", Description: "Mark channel as read"},
	{Name: "mark-all-read", Description: "Mark all channels as read"},
	{Name: "open", Description: "Open URL in browser"},
	{Name: "reconnect", Description: "Reconnect Socket Mode"},
	{Name: "debug", Description: "Toggle debug logging"},
	{Name: "set", Description: "Change config at runtime"},
	{Name: "bookmarks", Description: "Show channel bookmarks"},
	{Name: "workspace", Aliases: []string{"ws"}, Description: "Switch workspace"},
	{Name: "members", Aliases: []string{"who"}, Description: "List channel members"},
	{Name: "create-channel", Description: "Create a new channel"},
	{Name: "invite", Description: "Invite user to channel"},
	{Name: "group-dm", Aliases: []string{"gdm"}, Description: "Create group DM"},
}

// CommandBar is a vim-style command input shown at the bottom of the screen.
type CommandBar struct {
	*tview.InputField
	cfg            *config.Config
	history        []string
	histIdx        int // -1 means not browsing history
	onExecute      func(command, args string)
	onClose        func()
	setOptionNames []string // option names for :set subcommand completion
}

// NewCommandBar creates a new command bar.
func NewCommandBar(cfg *config.Config) *CommandBar {
	cb := &CommandBar{
		InputField: tview.NewInputField(),
		cfg:        cfg,
		histIdx:    -1,
	}

	cb.SetLabel(":")
	cb.SetFieldBackgroundColor(tcell.ColorDefault)
	cb.SetLabelColor(tcell.ColorWhite)
	cb.SetBorder(false)
	cb.SetInputCapture(cb.handleInput)

	// Tab completion via autocomplete.
	cb.SetAutocompleteFunc(cb.autocomplete)
	cb.SetAutocompletedFunc(func(text string, _ int, _ int) bool {
		cb.SetText(text)
		return true
	})

	return cb
}

// SetOnExecute sets the callback for when a command is executed.
func (cb *CommandBar) SetOnExecute(fn func(command, args string)) {
	cb.onExecute = fn
}

// SetOnClose sets the callback for when the command bar is dismissed.
func (cb *CommandBar) SetOnClose(fn func()) {
	cb.onClose = fn
}

// SetSetOptionNames sets the option names available for :set subcommand completion.
func (cb *CommandBar) SetSetOptionNames(names []string) {
	cb.setOptionNames = names
}

// Reset clears the input and history index.
func (cb *CommandBar) Reset() {
	cb.SetText("")
	cb.histIdx = -1
}

func (cb *CommandBar) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEnter:
		cb.execute()
		return nil
	case tcell.KeyEscape:
		if cb.onClose != nil {
			cb.onClose()
		}
		return nil
	case tcell.KeyUp:
		cb.historyPrev()
		return nil
	case tcell.KeyDown:
		cb.historyNext()
		return nil
	case tcell.KeyTab:
		// Let tview's autocomplete handle it.
		return event
	}
	return event
}

func (cb *CommandBar) execute() {
	text := strings.TrimSpace(cb.GetText())
	if text == "" {
		if cb.onClose != nil {
			cb.onClose()
		}
		return
	}

	// Add to history (avoid duplicates of last entry).
	if len(cb.history) == 0 || cb.history[len(cb.history)-1] != text {
		cb.history = append(cb.history, text)
	}
	cb.histIdx = -1

	// Parse command and args.
	parts := strings.SplitN(text, " ", 2)
	command := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	// Resolve aliases.
	command = resolveAlias(command)

	if cb.onExecute != nil {
		cb.onExecute(command, args)
	}
}

func (cb *CommandBar) historyPrev() {
	if len(cb.history) == 0 {
		return
	}
	if cb.histIdx == -1 {
		cb.histIdx = len(cb.history) - 1
	} else if cb.histIdx > 0 {
		cb.histIdx--
	}
	cb.SetText(cb.history[cb.histIdx])
}

func (cb *CommandBar) historyNext() {
	if cb.histIdx == -1 {
		return
	}
	if cb.histIdx < len(cb.history)-1 {
		cb.histIdx++
		cb.SetText(cb.history[cb.histIdx])
	} else {
		cb.histIdx = -1
		cb.SetText("")
	}
}

func (cb *CommandBar) autocomplete(currentText string) []string {
	if currentText == "" {
		return nil
	}
	lower := strings.ToLower(currentText)

	// Check for subcommand completion (e.g., "set <option>").
	if spaceIdx := strings.Index(lower, " "); spaceIdx >= 0 {
		cmd := strings.TrimSpace(lower[:spaceIdx])
		sub := strings.TrimSpace(lower[spaceIdx+1:])

		if cmd == "set" && len(cb.setOptionNames) > 0 {
			var matches []string
			for _, name := range cb.setOptionNames {
				if sub == "" || strings.HasPrefix(name, sub) {
					matches = append(matches, "set "+name)
				}
			}
			return matches
		}
		return nil
	}

	// Complete the command name.
	var matches []string
	for _, cmd := range builtinVimCommands {
		if strings.HasPrefix(cmd.Name, lower) {
			matches = append(matches, cmd.Name)
		}
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, lower) {
				matches = append(matches, alias)
			}
		}
	}
	return matches
}

// resolveAlias maps command aliases to their canonical names.
func resolveAlias(command string) string {
	for _, cmd := range builtinVimCommands {
		if cmd.Name == command {
			return command
		}
		for _, alias := range cmd.Aliases {
			if alias == command {
				return cmd.Name
			}
		}
	}
	return command
}
