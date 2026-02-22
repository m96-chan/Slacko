package chat

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// UserProfileData holds the data to display in the user profile panel.
type UserProfileData struct {
	UserID      string
	DisplayName string
	RealName    string
	Title       string
	StatusEmoji string
	StatusText  string
	Timezone    string
	TzOffset    int // seconds offset from UTC
	Presence    string
	Email       string
	Phone       string
	IsBot       bool
	IsAdmin     bool
	IsOwner     bool
}

// UserProfilePanel is a modal panel that displays user profile details.
type UserProfilePanel struct {
	*tview.Flex
	cfg      *config.Config
	content  *tview.TextView
	status   *tview.TextView
	data     UserProfileData
	onClose  func()
	onOpenDM func(userID string)
	onCopyID func(userID string)
}

// NewUserProfilePanel creates a new user profile panel component.
func NewUserProfilePanel(cfg *config.Config) *UserProfilePanel {
	up := &UserProfilePanel{
		cfg: cfg,
	}

	up.content = tview.NewTextView()
	up.content.SetDynamicColors(true)
	up.content.SetScrollable(true)
	up.content.SetWordWrap(true)
	up.content.SetInputCapture(up.handleInput)

	up.status = tview.NewTextView()
	up.status.SetDynamicColors(true)
	up.status.SetTextAlign(tview.AlignLeft)
	up.status.SetText(" [d]m  [i]d  [Esc]close")

	up.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(up.content, 0, 1, true).
		AddItem(up.status, 1, 0, false)
	up.SetBorder(true).SetTitle(" User Profile ")

	return up
}

// SetOnClose sets the callback for closing the panel.
func (up *UserProfilePanel) SetOnClose(fn func()) {
	up.onClose = fn
}

// SetOnOpenDM sets the callback for opening a DM with the user.
func (up *UserProfilePanel) SetOnOpenDM(fn func(userID string)) {
	up.onOpenDM = fn
}

// SetOnCopyID sets the callback for copying the user's ID.
func (up *UserProfilePanel) SetOnCopyID(fn func(userID string)) {
	up.onCopyID = fn
}

// SetData populates the panel with user profile info.
func (up *UserProfilePanel) SetData(data UserProfileData) {
	up.data = data
	up.render()
}

// SetStatus updates the status text at the bottom.
func (up *UserProfilePanel) SetStatus(text string) {
	up.status.SetText(" " + text)
}

// Data returns the current user profile data.
func (up *UserProfilePanel) Data() UserProfileData {
	return up.data
}

// Reset clears the panel content.
func (up *UserProfilePanel) Reset() {
	up.content.SetText("")
	up.data = UserProfileData{}
	up.status.SetText(" [d]m  [i]d  [Esc]close")
}

// render builds the display text from the current data.
func (up *UserProfilePanel) render() {
	d := up.data

	// Name header.
	name := d.DisplayName
	if name == "" {
		name = d.RealName
	}
	if name == "" {
		name = d.UserID
	}
	text := fmt.Sprintf("[::b]%s[::-]\n", tview.Escape(name))

	// Show real name below display name if different.
	if d.RealName != "" && d.RealName != d.DisplayName {
		text += fmt.Sprintf("[gray]%s[-]\n", tview.Escape(d.RealName))
	}

	// Role badges.
	if d.IsBot {
		text += "[gray]BOT[-]\n"
	}
	if d.IsOwner {
		text += "[yellow::b]Workspace Owner[::-]\n"
	} else if d.IsAdmin {
		text += "[yellow]Workspace Admin[-]\n"
	}
	text += "\n"

	// Title.
	if d.Title != "" {
		text += fmt.Sprintf("[::b]Title[::-]      %s\n", tview.Escape(d.Title))
	}

	// Status.
	if d.StatusText != "" || d.StatusEmoji != "" {
		statusStr := ""
		if d.StatusEmoji != "" {
			statusStr = d.StatusEmoji + " "
		}
		statusStr += d.StatusText
		text += fmt.Sprintf("[::b]Status[::-]     %s\n", tview.Escape(statusStr))
	}

	// Presence.
	if d.Presence != "" {
		icon := presenceIcon(d.Presence)
		text += fmt.Sprintf("[::b]Presence[::-]   %s %s\n", icon, d.Presence)
	}

	// Timezone.
	if d.Timezone != "" {
		localTime := time.Now().UTC().Add(time.Duration(d.TzOffset) * time.Second)
		text += fmt.Sprintf("[::b]Timezone[::-]   %s (%s)\n", tview.Escape(d.Timezone), localTime.Format("3:04 PM"))
	}

	// Email.
	if d.Email != "" {
		text += fmt.Sprintf("[::b]Email[::-]      %s\n", tview.Escape(d.Email))
	}

	// Phone.
	if d.Phone != "" {
		text += fmt.Sprintf("[::b]Phone[::-]      %s\n", tview.Escape(d.Phone))
	}

	up.content.SetText(text)
}

// handleInput processes keybindings for the user profile panel.
func (up *UserProfilePanel) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch {
	case name == up.cfg.Keybinds.UserProfilePanel.Close:
		up.close()
		return nil

	case name == up.cfg.Keybinds.UserProfilePanel.OpenDM:
		if up.onOpenDM != nil && up.data.UserID != "" {
			up.onOpenDM(up.data.UserID)
			up.close()
		}
		return nil

	case name == up.cfg.Keybinds.UserProfilePanel.CopyID:
		if up.onCopyID != nil && up.data.UserID != "" {
			up.onCopyID(up.data.UserID)
		}
		return nil
	}

	return event
}

// close signals the panel should be hidden.
func (up *UserProfilePanel) close() {
	if up.onClose != nil {
		up.onClose()
	}
}
