package chat

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"
	"github.com/slack-go/slack"

	"github.com/m96-chan/Slacko/internal/config"
)

// autocompleteKind identifies the type of autocomplete trigger.
type autocompleteKind int

const (
	acNone autocompleteKind = iota
	acUser
	acChannel
	acCommand
)

// suggestion holds a single autocomplete result.
type suggestion struct {
	display    string // shown in the dropdown list
	insertText string // inserted into the input on completion
}

// userEntry holds precomputed data for a user suggestion.
type userEntry struct {
	userID      string
	displayText string // e.g. "● alice (Alice Smith)"
	searchText  string // lowercased for fuzzy matching
	insertText  string // e.g. "<@U123> "
}

// channelEntry holds precomputed data for a channel suggestion.
type channelEntry struct {
	channelID   string
	displayText string // e.g. "# general"
	searchText  string // lowercased for fuzzy matching
	insertText  string // e.g. "<#C123> "
}

// commandEntry holds precomputed data for a command suggestion.
type commandEntry struct {
	name        string // e.g. "/help"
	description string // e.g. "Show available commands"
	searchText  string // lowercased for fuzzy matching
	insertText  string // e.g. "/help "
}

// MentionsList displays autocomplete suggestions in a dropdown.
type MentionsList struct {
	*tview.List
	cfg         *config.Config
	users       []userEntry
	channels    []channelEntry
	commands    []commandEntry
	suggestions []suggestion
}

// NewMentionsList creates a new mentions autocomplete dropdown.
func NewMentionsList(cfg *config.Config) *MentionsList {
	ml := &MentionsList{
		List: tview.NewList(),
		cfg:  cfg,
	}

	ml.ShowSecondaryText(false)
	ml.SetHighlightFullLine(true)
	ml.SetWrapAround(false)
	ml.SetBorder(true)

	return ml
}

// SetUsers populates the user autocomplete data.
func (ml *MentionsList) SetUsers(users map[string]slack.User) {
	ml.users = make([]userEntry, 0, len(users))

	for _, u := range users {
		if u.Deleted || u.IsBot {
			continue
		}

		displayName := u.Profile.DisplayName
		if displayName == "" {
			displayName = u.RealName
		}
		if displayName == "" {
			displayName = u.Name
		}
		if displayName == "" {
			displayName = u.ID
		}

		display := fmt.Sprintf("%s %s", presenceIcon(u.Presence), displayName)
		if u.RealName != "" && u.RealName != displayName {
			display += fmt.Sprintf(" (%s)", u.RealName)
		}

		search := strings.Join([]string{
			strings.ToLower(u.Profile.DisplayName),
			strings.ToLower(u.RealName),
			strings.ToLower(u.Name),
		}, " ")

		ml.users = append(ml.users, userEntry{
			userID:      u.ID,
			displayText: display,
			searchText:  search,
			insertText:  fmt.Sprintf("<@%s> ", u.ID),
		})
	}
}

// SetChannels populates the channel autocomplete data.
func (ml *MentionsList) SetChannels(channels []slack.Channel, users map[string]slack.User, selfUserID string) {
	ml.channels = make([]channelEntry, 0, len(channels))

	for _, ch := range channels {
		chType := classifyChannel(ch)
		display := channelDisplayText(ch, chType, users, selfUserID, ml.cfg.AsciiIcons)
		search := pickerSearchText(ch, chType, users)

		ml.channels = append(ml.channels, channelEntry{
			channelID:   ch.ID,
			displayText: display,
			searchText:  search,
			insertText:  fmt.Sprintf("<#%s> ", ch.ID),
		})
	}
}

// SetCommands sets the available slash commands for autocomplete.
func (ml *MentionsList) SetCommands(cmds []commandEntry) {
	ml.commands = cmds
}

// Filter runs fuzzy matching for the given trigger kind and prefix.
// Returns the number of matching suggestions.
func (ml *MentionsList) Filter(kind autocompleteKind, prefix string, limit int) int {
	ml.Clear()
	ml.suggestions = nil

	switch kind {
	case acUser:
		return ml.filterUsers(prefix, limit)
	case acChannel:
		return ml.filterChannels(prefix, limit)
	case acCommand:
		return ml.filterCommands(prefix, limit)
	}

	return 0
}

// GetSelected returns the currently selected suggestion.
func (ml *MentionsList) GetSelected() suggestion {
	idx := ml.GetCurrentItem()
	if idx < 0 || idx >= len(ml.suggestions) {
		return suggestion{}
	}
	return ml.suggestions[idx]
}

// SelectNext moves selection to the next suggestion.
func (ml *MentionsList) SelectNext() {
	cur := ml.GetCurrentItem()
	if cur < ml.GetItemCount()-1 {
		ml.SetCurrentItem(cur + 1)
	}
}

// SelectPrev moves selection to the previous suggestion.
func (ml *MentionsList) SelectPrev() {
	cur := ml.GetCurrentItem()
	if cur > 0 {
		ml.SetCurrentItem(cur - 1)
	}
}

// filterUsers runs fuzzy matching against user entries.
func (ml *MentionsList) filterUsers(prefix string, limit int) int {
	if prefix == "" {
		// Show all users up to limit.
		count := len(ml.users)
		if count > limit {
			count = limit
		}
		for i := 0; i < count; i++ {
			u := ml.users[i]
			ml.suggestions = append(ml.suggestions, suggestion{
				display:    u.displayText,
				insertText: u.insertText,
			})
			ml.AddItem(u.displayText, "", 0, nil)
		}
		if count > 0 {
			ml.SetCurrentItem(0)
		}
		return count
	}

	targets := make([]string, len(ml.users))
	for i, u := range ml.users {
		targets[i] = u.searchText
	}

	matches := fuzzy.Find(prefix, targets)

	count := len(matches)
	if count > limit {
		count = limit
	}

	for i := 0; i < count; i++ {
		u := ml.users[matches[i].Index]
		ml.suggestions = append(ml.suggestions, suggestion{
			display:    u.displayText,
			insertText: u.insertText,
		})
		ml.AddItem(u.displayText, "", 0, nil)
	}

	if count > 0 {
		ml.SetCurrentItem(0)
	}

	return count
}

// filterChannels runs fuzzy matching against channel entries.
func (ml *MentionsList) filterChannels(prefix string, limit int) int {
	if prefix == "" {
		count := len(ml.channels)
		if count > limit {
			count = limit
		}
		for i := 0; i < count; i++ {
			ch := ml.channels[i]
			ml.suggestions = append(ml.suggestions, suggestion{
				display:    ch.displayText,
				insertText: ch.insertText,
			})
			ml.AddItem(ch.displayText, "", 0, nil)
		}
		if count > 0 {
			ml.SetCurrentItem(0)
		}
		return count
	}

	targets := make([]string, len(ml.channels))
	for i, ch := range ml.channels {
		targets[i] = ch.searchText
	}

	matches := fuzzy.Find(prefix, targets)

	count := len(matches)
	if count > limit {
		count = limit
	}

	for i := 0; i < count; i++ {
		ch := ml.channels[matches[i].Index]
		ml.suggestions = append(ml.suggestions, suggestion{
			display:    ch.displayText,
			insertText: ch.insertText,
		})
		ml.AddItem(ch.displayText, "", 0, nil)
	}

	if count > 0 {
		ml.SetCurrentItem(0)
	}

	return count
}

// filterCommands runs fuzzy matching against command entries.
func (ml *MentionsList) filterCommands(prefix string, limit int) int {
	if prefix == "" {
		count := len(ml.commands)
		if count > limit {
			count = limit
		}
		for i := 0; i < count; i++ {
			cmd := ml.commands[i]
			display := fmt.Sprintf("%s — %s", cmd.name, cmd.description)
			ml.suggestions = append(ml.suggestions, suggestion{
				display:    display,
				insertText: cmd.insertText,
			})
			ml.AddItem(display, "", 0, nil)
		}
		if count > 0 {
			ml.SetCurrentItem(0)
		}
		return count
	}

	targets := make([]string, len(ml.commands))
	for i, cmd := range ml.commands {
		targets[i] = cmd.searchText
	}

	matches := fuzzy.Find(prefix, targets)

	count := len(matches)
	if count > limit {
		count = limit
	}

	for i := 0; i < count; i++ {
		cmd := ml.commands[matches[i].Index]
		display := fmt.Sprintf("%s — %s", cmd.name, cmd.description)
		ml.suggestions = append(ml.suggestions, suggestion{
			display:    display,
			insertText: cmd.insertText,
		})
		ml.AddItem(display, "", 0, nil)
	}

	if count > 0 {
		ml.SetCurrentItem(0)
	}

	return count
}

// findAutocompleteTrigger scans text backwards from the end to find
// an autocomplete trigger (@, #, /). Returns the trigger kind, the prefix
// after the trigger, and the byte offset of the trigger character.
func findAutocompleteTrigger(text string) (autocompleteKind, string, int) {
	if text == "" {
		return acNone, "", -1
	}

	// Check for slash command: "/" must be at position 0.
	if text[0] == '/' {
		// Only trigger if no space yet (user is still typing the command name).
		prefix := text[1:]
		if !strings.Contains(prefix, " ") {
			return acCommand, prefix, 0
		}
		return acNone, "", -1
	}

	for i := len(text) - 1; i >= 0; i-- {
		b := text[i]
		// Stop at word boundaries.
		if b == ' ' || b == '\n' || b == '\r' || b == '\t' {
			return acNone, "", -1
		}
		switch b {
		case '@':
			return acUser, text[i+1:], i
		case '#':
			return acChannel, text[i+1:], i
		}
	}

	return acNone, "", -1
}
