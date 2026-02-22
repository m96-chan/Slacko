package chat

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// ChannelInfoData holds the data to display in the channel info panel.
type ChannelInfoData struct {
	ChannelID   string
	Name        string
	Description string
	Topic       string
	Purpose     string
	Creator     string
	Created     time.Time
	NumMembers  int
	NumPins     int
	IsArchived  bool
	IsPrivate   bool
	IsDM        bool
}

// ChannelInfoPanel is a modal panel that displays channel details.
type ChannelInfoPanel struct {
	*tview.Flex
	cfg        *config.Config
	content    *tview.TextView
	status     *tview.TextView
	data       ChannelInfoData
	onClose    func()
	onSetTopic func(channelID string)
	onSetPurpose func(channelID string)
	onLeave    func(channelID string)
}

// NewChannelInfoPanel creates a new channel info panel component.
func NewChannelInfoPanel(cfg *config.Config) *ChannelInfoPanel {
	ci := &ChannelInfoPanel{
		cfg: cfg,
	}

	ci.content = tview.NewTextView()
	ci.content.SetDynamicColors(true)
	ci.content.SetScrollable(true)
	ci.content.SetWordWrap(true)
	ci.content.SetInputCapture(ci.handleInput)

	ci.status = tview.NewTextView()
	ci.status.SetDynamicColors(true)
	ci.status.SetTextAlign(tview.AlignLeft)
	ci.status.SetText(" [t]opic  [p]urpose  [l]eave  [Esc]close")

	ci.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ci.content, 0, 1, true).
		AddItem(ci.status, 1, 0, false)
	ci.SetBorder(true).SetTitle(" Channel Info ")

	return ci
}

// SetOnClose sets the callback for closing the panel.
func (ci *ChannelInfoPanel) SetOnClose(fn func()) {
	ci.onClose = fn
}

// SetOnSetTopic sets the callback for setting the channel topic.
func (ci *ChannelInfoPanel) SetOnSetTopic(fn func(channelID string)) {
	ci.onSetTopic = fn
}

// SetOnSetPurpose sets the callback for setting the channel purpose.
func (ci *ChannelInfoPanel) SetOnSetPurpose(fn func(channelID string)) {
	ci.onSetPurpose = fn
}

// SetOnLeave sets the callback for leaving the channel.
func (ci *ChannelInfoPanel) SetOnLeave(fn func(channelID string)) {
	ci.onLeave = fn
}

// SetData populates the panel with channel info.
func (ci *ChannelInfoPanel) SetData(data ChannelInfoData) {
	ci.data = data
	ci.render()
}

// SetStatus updates the status text at the bottom.
func (ci *ChannelInfoPanel) SetStatus(text string) {
	ci.status.SetText(" " + text)
}

// Data returns the current channel info data.
func (ci *ChannelInfoPanel) Data() ChannelInfoData {
	return ci.data
}

// Reset clears the panel content.
func (ci *ChannelInfoPanel) Reset() {
	ci.content.SetText("")
	ci.data = ChannelInfoData{}
	ci.status.SetText(" [t]opic  [p]urpose  [l]eave  [Esc]close")
}

// render builds the display text from the current data.
func (ci *ChannelInfoPanel) render() {
	d := ci.data

	var channelType string
	switch {
	case d.IsDM:
		channelType = "Direct Message"
	case d.IsPrivate:
		channelType = "Private Channel"
	default:
		channelType = "Public Channel"
	}

	text := fmt.Sprintf("[::b]#%s[::-]\n", tview.Escape(d.Name))
	text += fmt.Sprintf("[gray]%s[-]\n\n", channelType)

	if d.Description != "" {
		text += fmt.Sprintf("[::b]Description[::-]\n%s\n\n", tview.Escape(d.Description))
	}

	if d.Topic != "" {
		text += fmt.Sprintf("[::b]Topic[::-]\n%s\n\n", tview.Escape(d.Topic))
	}

	if d.Purpose != "" {
		text += fmt.Sprintf("[::b]Purpose[::-]\n%s\n\n", tview.Escape(d.Purpose))
	}

	if d.Creator != "" {
		text += fmt.Sprintf("[::b]Created by[::-]  %s\n", tview.Escape(d.Creator))
	}

	if !d.Created.IsZero() {
		text += fmt.Sprintf("[::b]Created[::-]     %s\n", d.Created.Format("January 2, 2006"))
	}

	if d.NumMembers > 0 {
		text += fmt.Sprintf("[::b]Members[::-]     %d\n", d.NumMembers)
	}

	if d.NumPins > 0 {
		text += fmt.Sprintf("[::b]Pinned[::-]      %d\n", d.NumPins)
	}

	if d.IsArchived {
		text += "\n[red::b]This channel is archived[-::-]"
	}

	ci.content.SetText(text)
}

// handleInput processes keybindings for the channel info panel.
func (ci *ChannelInfoPanel) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch {
	case name == ci.cfg.Keybinds.ChannelInfoPanel.Close:
		ci.close()
		return nil

	case name == ci.cfg.Keybinds.ChannelInfoPanel.SetTopic:
		if ci.onSetTopic != nil && ci.data.ChannelID != "" {
			ci.onSetTopic(ci.data.ChannelID)
		}
		return nil

	case name == ci.cfg.Keybinds.ChannelInfoPanel.SetPurpose:
		if ci.onSetPurpose != nil && ci.data.ChannelID != "" {
			ci.onSetPurpose(ci.data.ChannelID)
		}
		return nil

	case name == ci.cfg.Keybinds.ChannelInfoPanel.Leave:
		if ci.onLeave != nil && ci.data.ChannelID != "" {
			ci.onLeave(ci.data.ChannelID)
		}
		return nil

	case name == ci.cfg.Keybinds.ChannelInfo:
		// Toggle: pressing Ctrl+O again closes the panel.
		ci.close()
		return nil
	}

	return event
}

// close signals the panel should be hidden.
func (ci *ChannelInfoPanel) close() {
	if ci.onClose != nil {
		ci.onClose()
	}
}
