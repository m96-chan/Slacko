package chat

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// inputMode tracks the current input state.
type inputMode int

const (
	inputModeNormal inputMode = iota
	inputModeReply
	inputModeEdit
)

// OnSendFunc is called when the user sends a message.
// channelID and text are always set; threadTS is non-empty for thread replies.
type OnSendFunc func(channelID, text, threadTS string)

// OnEditFunc is called when the user submits an edited message.
type OnEditFunc func(channelID, timestamp, text string)

// MessageInput wraps tview.TextArea with send/reply/edit support.
type MessageInput struct {
	*tview.TextArea
	cfg       *config.Config
	channelID string
	mode      inputMode
	threadTS  string // set in reply mode
	editTS    string // set in edit mode
	onSend    OnSendFunc
	onEdit    OnEditFunc
	onCancel  func() // called when user cancels reply/edit

	// Autocomplete state.
	mentionsList       *MentionsList
	acKind             autocompleteKind
	acStart            int // byte offset of trigger char in text
	onShowAutocomplete func(count int)
	onHideAutocomplete func()
}

// NewMessageInput creates a new message input component.
func NewMessageInput(cfg *config.Config) *MessageInput {
	mi := &MessageInput{
		TextArea: tview.NewTextArea(),
		cfg:      cfg,
		mode:     inputModeNormal,
	}

	mi.SetBorder(true).SetTitle(" Input ")
	mi.SetPlaceholder("Type a message...")

	mi.SetInputCapture(mi.handleInput)
	mi.SetChangedFunc(mi.onTextChanged)

	return mi
}

// SetOnSend sets the callback for sending messages.
func (mi *MessageInput) SetOnSend(fn OnSendFunc) {
	mi.onSend = fn
}

// SetOnEdit sets the callback for editing messages.
func (mi *MessageInput) SetOnEdit(fn OnEditFunc) {
	mi.onEdit = fn
}

// SetOnCancel sets the callback for cancelling reply/edit mode.
func (mi *MessageInput) SetOnCancel(fn func()) {
	mi.onCancel = fn
}

// SetMentionsList sets the autocomplete dropdown reference.
func (mi *MessageInput) SetMentionsList(ml *MentionsList) {
	mi.mentionsList = ml
}

// SetOnShowAutocomplete sets the callback for showing the autocomplete dropdown.
func (mi *MessageInput) SetOnShowAutocomplete(fn func(count int)) {
	mi.onShowAutocomplete = fn
}

// SetOnHideAutocomplete sets the callback for hiding the autocomplete dropdown.
func (mi *MessageInput) SetOnHideAutocomplete(fn func()) {
	mi.onHideAutocomplete = fn
}

// SetChannel sets the active channel for outgoing messages.
func (mi *MessageInput) SetChannel(channelID string) {
	mi.channelID = channelID
	// Cancel any active reply/edit when switching channels.
	if mi.mode != inputModeNormal {
		mi.cancelMode()
	}
}

// SetReplyContext enters reply mode.
func (mi *MessageInput) SetReplyContext(threadTS, userName string) {
	mi.mode = inputModeReply
	mi.threadTS = threadTS
	mi.SetTitle(fmt.Sprintf(" Reply to %s ", userName))
}

// SetEditMode enters edit mode, populating the input with existing text.
func (mi *MessageInput) SetEditMode(timestamp, text string) {
	mi.mode = inputModeEdit
	mi.editTS = timestamp
	mi.SetTitle(" Editing ")
	mi.SetText(text, true)
}

// Mode returns the current input mode.
func (mi *MessageInput) Mode() inputMode {
	return mi.mode
}

// handleInput processes keybindings for the input area.
func (mi *MessageInput) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	// Autocomplete navigation when dropdown is active.
	if mi.acKind != acNone && mi.mentionsList != nil {
		switch {
		case name == mi.cfg.Keybinds.MessageInput.TabComplete:
			mi.completeAutocomplete()
			return nil
		case event.Key() == tcell.KeyUp:
			mi.mentionsList.SelectPrev()
			return nil
		case event.Key() == tcell.KeyDown:
			mi.mentionsList.SelectNext()
			return nil
		case name == mi.cfg.Keybinds.MessageInput.Cancel:
			mi.dismissAutocomplete()
			return nil
		}
	}

	switch name {
	case mi.cfg.Keybinds.MessageInput.Send:
		mi.dismissAutocomplete()
		mi.send()
		return nil

	case mi.cfg.Keybinds.MessageInput.Newline:
		// Transform Shift+Enter into plain Enter so TextArea adds a newline.
		return tcell.NewEventKey(tcell.KeyEnter, '\n', tcell.ModNone)

	case mi.cfg.Keybinds.MessageInput.Cancel:
		if mi.mode != inputModeNormal {
			mi.cancelMode()
			return nil
		}
		// In normal mode, let Escape propagate (focus change etc).
		return event
	}

	return event
}

// send dispatches the current input text.
func (mi *MessageInput) send() {
	text := strings.TrimSpace(mi.GetText())
	if text == "" {
		return
	}
	if mi.channelID == "" {
		return
	}

	switch mi.mode {
	case inputModeEdit:
		if mi.onEdit != nil {
			mi.onEdit(mi.channelID, mi.editTS, text)
		}
	default:
		threadTS := ""
		if mi.mode == inputModeReply {
			threadTS = mi.threadTS
		}
		if mi.onSend != nil {
			mi.onSend(mi.channelID, text, threadTS)
		}
	}

	mi.SetText("", false)
	mi.cancelMode()
}

// onTextChanged detects autocomplete triggers after each text change.
func (mi *MessageInput) onTextChanged() {
	if mi.mentionsList == nil {
		return
	}

	text := mi.GetText()
	kind, prefix, start := findAutocompleteTrigger(text)

	if kind == acNone {
		if mi.acKind != acNone {
			mi.dismissAutocomplete()
		}
		return
	}

	mi.acKind = kind
	mi.acStart = start

	count := mi.mentionsList.Filter(kind, prefix, mi.cfg.AutocompleteLimit)
	if count > 0 {
		if mi.onShowAutocomplete != nil {
			mi.onShowAutocomplete(count)
		}
	} else {
		mi.dismissAutocomplete()
	}
}

// completeAutocomplete inserts the selected suggestion into the text.
func (mi *MessageInput) completeAutocomplete() {
	if mi.mentionsList == nil {
		return
	}

	sel := mi.mentionsList.GetSelected()
	if sel.insertText == "" {
		return
	}

	text := mi.GetText()
	newText := text[:mi.acStart] + sel.insertText
	mi.SetText(newText, true)

	mi.acKind = acNone
	if mi.onHideAutocomplete != nil {
		mi.onHideAutocomplete()
	}
}

// dismissAutocomplete hides the dropdown without completing.
func (mi *MessageInput) dismissAutocomplete() {
	mi.acKind = acNone
	if mi.onHideAutocomplete != nil {
		mi.onHideAutocomplete()
	}
}

// cancelMode resets the input to normal mode.
func (mi *MessageInput) cancelMode() {
	prevMode := mi.mode
	mi.mode = inputModeNormal
	mi.threadTS = ""
	mi.editTS = ""
	mi.SetTitle(" Input ")

	// Clear text when cancelling edit mode.
	if prevMode == inputModeEdit {
		mi.SetText("", false)
	}

	if mi.onCancel != nil {
		mi.onCancel()
	}
}
