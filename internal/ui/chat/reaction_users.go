package chat

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// ReactionUsersEntry holds data for a single reaction with resolved user names.
type ReactionUsersEntry struct {
	Emoji  string   // rendered emoji character
	Name   string   // emoji short name (e.g. "thumbsup")
	Users  []string // display names of users who reacted
	IsSelf bool     // whether the current user reacted with this emoji
}

// ReactionUsersPanel is a modal panel that displays who reacted to a message.
type ReactionUsersPanel struct {
	*tview.Flex
	cfg     *config.Config
	content *tview.TextView
	status  *tview.TextView
	onClose func()
}

// NewReactionUsersPanel creates a new reaction users panel component.
func NewReactionUsersPanel(cfg *config.Config) *ReactionUsersPanel {
	rp := &ReactionUsersPanel{
		cfg: cfg,
	}

	rp.content = tview.NewTextView()
	rp.content.SetDynamicColors(true)
	rp.content.SetScrollable(true)
	rp.content.SetWordWrap(true)
	rp.content.SetInputCapture(rp.handleInput)

	rp.status = tview.NewTextView()
	rp.status.SetDynamicColors(true)
	rp.status.SetTextAlign(tview.AlignLeft)
	rp.status.SetText(" [Esc]close")

	rp.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(rp.content, 0, 1, true).
		AddItem(rp.status, 1, 0, false)
	rp.SetBorder(true).SetTitle(" Reactions ")
	rp.SetInputCapture(rp.handleInput)

	return rp
}

// SetOnClose sets the callback for closing the panel.
func (rp *ReactionUsersPanel) SetOnClose(fn func()) {
	rp.onClose = fn
}

// SetReactions populates the panel with reaction entries.
func (rp *ReactionUsersPanel) SetReactions(entries []ReactionUsersEntry) {
	if len(entries) == 0 {
		rp.content.SetText("No reactions on this message.")
		return
	}

	rp.content.SetText(rp.buildReactionsText(entries))
}

// buildReactionsText formats reaction entries into tview-styled text.
func (rp *ReactionUsersPanel) buildReactionsText(entries []ReactionUsersEntry) string {
	var b strings.Builder
	theme := rp.cfg.Theme.MessagesList

	for i, e := range entries {
		if i > 0 {
			b.WriteString("\n")
		}

		// Pick style based on whether self reacted.
		var style config.StyleWrapper
		if e.IsSelf {
			style = theme.ReactionSelf
		} else {
			style = theme.ReactionOther
		}

		// Emoji + name header.
		fmt.Fprintf(&b, "%s%s :%s:%s\n", style.Tag(), e.Emoji, tview.Escape(e.Name), style.Reset())

		// User list, indented.
		fmt.Fprintf(&b, "  %s", strings.Join(e.Users, ", "))
	}

	return b.String()
}

// SetStatus updates the status text at the bottom.
func (rp *ReactionUsersPanel) SetStatus(text string) {
	rp.status.SetText(" " + text)
}

// Reset clears the panel content.
func (rp *ReactionUsersPanel) Reset() {
	rp.content.SetText("")
	rp.status.SetText(" [Esc]close")
}

// handleInput processes keybindings for the reaction users panel.
func (rp *ReactionUsersPanel) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	if name == "Escape" {
		rp.close()
		return nil
	}

	return event
}

// close signals the panel should be hidden.
func (rp *ReactionUsersPanel) close() {
	if rp.onClose != nil {
		rp.onClose()
	}
}
