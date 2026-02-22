package chat

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// Panel identifies which panel is focused.
type Panel int

const (
	PanelChannels Panel = iota
	PanelMessages
	PanelInput
	PanelThread
)

// View is the main chat layout containing all panels.
type View struct {
	*tview.Pages
	app        *tview.Application
	cfg        *config.Config
	StatusBar  *StatusBar

	ChannelsTree    *ChannelsTree
	Header          *tview.TextView
	MessagesList    *MessagesList
	MessageInput    *MessageInput
	MentionsList    *MentionsList
	ThreadView      *ThreadView
	ChannelsPicker  *ChannelsPicker
	ReactionsPicker *ReactionsPicker
	FilePicker      *FilePicker

	outerFlex        *tview.Flex
	contentFlex      *tview.Flex
	mainFlex         *tview.Flex
	pickerModal      tview.Primitive
	reactionModal    tview.Primitive
	fileModal        tview.Primitive
	activePanel      Panel
	channelsVisible  bool
	threadVisible    bool
	pickerVisible    bool
	reactionVisible  bool
	filePickerVisible bool
}

// New creates the main chat view with the full flex layout.
//
// Layout:
//
//	Outer Flex (FlexRow)
//	├── mainFlex (FlexColumn)
//	│   ├── ChannelsTree (fixed 30 cols)
//	│   └── contentFlex (FlexRow)
//	│       ├── Header (fixed 1 row)
//	│       ├── Messages (proportional)
//	│       └── Input (fixed 3 rows)
//	└── StatusBar (fixed 1 row)
func New(app *tview.Application, cfg *config.Config) *View {
	v := &View{
		app:             app,
		cfg:             cfg,
		channelsVisible: true,
	}

	// Channel tree (left sidebar).
	v.ChannelsTree = NewChannelsTree(cfg, nil)

	// Header (channel name + topic).
	v.Header = tview.NewTextView().
		SetDynamicColors(true)
	v.Header.SetBorder(false)

	// Messages area.
	v.MessagesList = NewMessagesList(cfg)

	// Input area.
	v.MessageInput = NewMessageInput(cfg)

	// Mentions autocomplete dropdown (hidden by default, 0 height).
	v.MentionsList = NewMentionsList(cfg)

	// Wire autocomplete show/hide between MessageInput and View.
	v.MessageInput.SetMentionsList(v.MentionsList)
	v.MessageInput.SetOnShowAutocomplete(func(count int) {
		v.showMentions(count)
	})
	v.MessageInput.SetOnHideAutocomplete(func() {
		v.hideMentions()
	})

	// Thread view (hidden by default).
	v.ThreadView = NewThreadView(app, cfg)

	// Status bar.
	v.StatusBar = NewStatusBar(cfg)

	// Content flex (right side): header, messages, mentions dropdown, input.
	v.contentFlex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.Header, 1, 0, false).
		AddItem(v.MessagesList, 0, 1, false).
		AddItem(v.MentionsList, 0, 0, false).
		AddItem(v.MessageInput, 3, 0, false)

	// Main flex (horizontal): channel tree + content.
	v.mainFlex = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(v.ChannelsTree, 30, 0, false).
		AddItem(v.contentFlex, 0, 1, false)

	// Outer flex (vertical): main + status bar.
	v.outerFlex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.mainFlex, 0, 1, false).
		AddItem(v.StatusBar, 1, 0, false)

	// Channel picker (modal overlay).
	v.ChannelsPicker = NewChannelsPicker(cfg)
	v.ChannelsPicker.SetOnClose(func() {
		v.HidePicker()
	})

	// Centered modal wrapper for the picker.
	v.pickerModal = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(v.ChannelsPicker, 60, 0, true).
			AddItem(nil, 0, 1, false),
			0, 2, true).
		AddItem(nil, 0, 1, false)

	// Reaction picker (modal overlay).
	v.ReactionsPicker = NewReactionsPicker(cfg)
	v.ReactionsPicker.SetOnClose(func() {
		v.HideReactionPicker()
	})

	// Centered modal wrapper for the reaction picker.
	v.reactionModal = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(v.ReactionsPicker, 40, 0, true).
			AddItem(nil, 0, 1, false),
			0, 2, true).
		AddItem(nil, 0, 1, false)

	// File picker (modal overlay).
	v.FilePicker = NewFilePicker(cfg)
	v.FilePicker.SetOnClose(func() {
		v.HideFilePicker()
	})

	// Centered modal wrapper for the file picker.
	v.fileModal = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(v.FilePicker, 60, 0, true).
			AddItem(nil, 0, 1, false),
			0, 2, true).
		AddItem(nil, 0, 1, false)

	// Pages: main layout + modal overlays.
	v.Pages = tview.NewPages().
		AddPage("main", v.outerFlex, true, true)

	// Default focus on messages.
	v.activePanel = PanelMessages
	v.applyBorderStyles()

	return v
}

// SetOnChannelSelected sets the callback invoked when a channel is selected.
func (v *View) SetOnChannelSelected(fn OnChannelSelectedFunc) {
	v.ChannelsTree.SetOnChannelSelected(fn)
}

// FocusPanel sets focus to the given panel and updates border colors.
func (v *View) FocusPanel(panel Panel) {
	v.activePanel = panel
	v.applyBorderStyles()

	switch panel {
	case PanelChannels:
		v.app.SetFocus(v.ChannelsTree)
	case PanelMessages:
		v.app.SetFocus(v.MessagesList)
	case PanelInput:
		v.app.SetFocus(v.MessageInput)
	case PanelThread:
		v.ThreadView.FocusReplies()
	}
}

// HandleKey processes chat-level keybindings. Returns nil to consume the event.
func (v *View) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	// Toggle channels sidebar (Ctrl+B — non-rune, always works).
	if name == v.cfg.Keybinds.ToggleChannels {
		v.ToggleChannels()
		return nil
	}

	// Toggle channel picker (Ctrl+K — non-rune, always works).
	if name == v.cfg.Keybinds.ChannelPicker {
		if v.pickerVisible {
			v.HidePicker()
		} else {
			v.ShowPicker()
		}
		return nil
	}

	// When a modal is visible, all other keys go to its input.
	if v.pickerVisible || v.reactionVisible || v.filePickerVisible {
		return event
	}

	// Skip Rune-based focus keybinds when text input is active so the user can type.
	skipRune := (v.activePanel == PanelInput) ||
		(v.activePanel == PanelThread && v.ThreadView.IsInputFocused())
	if skipRune && event.Key() == tcell.KeyRune {
		return event
	}

	// Close thread panel if visible.
	if name == v.cfg.Keybinds.ToggleThread && v.threadVisible {
		v.CloseThread()
		return nil
	}

	switch name {
	case v.cfg.Keybinds.FocusChannels:
		v.FocusPanel(PanelChannels)
		return nil
	case v.cfg.Keybinds.FocusMessages:
		v.FocusPanel(PanelMessages)
		return nil
	case v.cfg.Keybinds.FocusInput:
		v.FocusPanel(PanelInput)
		return nil
	}

	return event
}

// ToggleChannels shows or hides the channel tree sidebar.
func (v *View) ToggleChannels() {
	v.channelsVisible = !v.channelsVisible
	v.rebuildMainFlex()

	// If channels were hidden and were focused, move focus to messages.
	if !v.channelsVisible && v.activePanel == PanelChannels {
		v.FocusPanel(PanelMessages)
	}
}

// ShowPicker shows the channel picker modal overlay.
func (v *View) ShowPicker() {
	v.pickerVisible = true
	v.ChannelsPicker.Reset()
	v.Pages.AddPage("picker", v.pickerModal, true, true)
	v.app.SetFocus(v.ChannelsPicker.input)
}

// HidePicker hides the channel picker and restores focus.
func (v *View) HidePicker() {
	v.pickerVisible = false
	v.Pages.RemovePage("picker")
	v.FocusPanel(v.activePanel)
}

// ShowReactionPicker shows the reaction picker modal overlay.
func (v *View) ShowReactionPicker() {
	v.reactionVisible = true
	v.ReactionsPicker.Reset()
	v.Pages.AddPage("reaction", v.reactionModal, true, true)
	v.app.SetFocus(v.ReactionsPicker.input)
}

// HideReactionPicker hides the reaction picker and restores focus.
func (v *View) HideReactionPicker() {
	v.reactionVisible = false
	v.Pages.RemovePage("reaction")
	v.FocusPanel(v.activePanel)
}

// ShowFilePicker shows the file picker modal overlay.
func (v *View) ShowFilePicker() {
	v.filePickerVisible = true
	v.FilePicker.Reset()
	v.Pages.AddPage("filepicker", v.fileModal, true, true)
	v.app.SetFocus(v.FilePicker.input)
}

// HideFilePicker hides the file picker and restores focus.
func (v *View) HideFilePicker() {
	v.filePickerVisible = false
	v.Pages.RemovePage("filepicker")
	v.FocusPanel(v.activePanel)
}

// showMentions resizes the mentions dropdown to show the given number of items.
func (v *View) showMentions(count int) {
	// Height = items + 2 for border.
	height := count + 2
	v.contentFlex.ResizeItem(v.MentionsList, height, 0)
}

// hideMentions collapses the mentions dropdown to zero height.
func (v *View) hideMentions() {
	v.contentFlex.ResizeItem(v.MentionsList, 0, 0)
}

// SetChannelHeader updates the header with channel name and topic.
func (v *View) SetChannelHeader(name, topic string) {
	text := fmt.Sprintf(" [::b]#%s[::-]", name)
	if topic != "" {
		text += fmt.Sprintf("  —  %s", topic)
	}
	v.Header.SetText(text)
}

// OpenThread shows the thread panel and focuses it.
func (v *View) OpenThread() {
	v.threadVisible = true
	v.rebuildMainFlex()
	v.FocusPanel(PanelThread)
}

// CloseThread hides the thread panel and clears its state.
func (v *View) CloseThread() {
	v.threadVisible = false
	v.ThreadView.Clear()
	v.rebuildMainFlex()
	v.applyBorderStyles()
	if v.activePanel == PanelThread {
		v.FocusPanel(PanelMessages)
	}
}

// rebuildMainFlex reconstructs the main flex after toggling panels.
// tview has no InsertItem, so we Clear() and re-add items.
func (v *View) rebuildMainFlex() {
	v.mainFlex.Clear()
	if v.channelsVisible {
		v.mainFlex.AddItem(v.ChannelsTree, 30, 0, false)
	}
	v.mainFlex.AddItem(v.contentFlex, 0, 1, false)
	if v.threadVisible {
		v.mainFlex.AddItem(v.ThreadView, 0, 1, false)
	}
}

// applyBorderStyles updates border colors based on which panel is active.
func (v *View) applyBorderStyles() {
	focusedFg, _, _ := v.cfg.Theme.Border.Focused.Style.Decompose()
	normalFg, _, _ := v.cfg.Theme.Border.Normal.Style.Decompose()
	focusedTitleFg, _, focusedTitleAttrs := v.cfg.Theme.Title.Focused.Style.Decompose()
	normalTitleFg, _, normalTitleAttrs := v.cfg.Theme.Title.Normal.Style.Decompose()

	type bordered struct {
		prim  tview.Primitive
		panel Panel
	}

	panels := []bordered{
		{v.ChannelsTree, PanelChannels},
		{v.MessagesList, PanelMessages},
		{v.MessageInput, PanelInput},
	}
	if v.threadVisible {
		panels = append(panels,
			bordered{v.ThreadView.repliesView, PanelThread},
			bordered{v.ThreadView.replyInput, PanelThread},
		)
	}

	for _, p := range panels {
		box := p.prim.(interface {
			SetBorderColor(tcell.Color) *tview.Box
			SetTitleColor(tcell.Color) *tview.Box
			SetTitleAttributes(tcell.AttrMask) *tview.Box
		})
		if p.panel == v.activePanel {
			box.SetBorderColor(focusedFg)
			box.SetTitleColor(focusedTitleFg)
			box.SetTitleAttributes(focusedTitleAttrs)
		} else {
			box.SetBorderColor(normalFg)
			box.SetTitleColor(normalTitleFg)
			box.SetTitleAttributes(normalTitleAttrs)
		}
	}
}
