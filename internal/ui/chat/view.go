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
	app       *tview.Application
	cfg       *config.Config
	StatusBar *StatusBar

	ChannelsTree     *ChannelsTree
	Header           *tview.TextView
	MessagesList     *MessagesList
	MessageInput     *MessageInput
	MentionsList     *MentionsList
	ThreadView       *ThreadView
	ChannelsPicker   *ChannelsPicker
	ReactionsPicker  *ReactionsPicker
	FilePicker       *FilePicker
	SearchPicker     *SearchPicker
	PinsPicker       *PinsPicker
	StarredPicker    *StarredPicker
	UserProfilePanel *UserProfilePanel
	ChannelInfoPanel *ChannelInfoPanel
	CommandBar       *CommandBar
	WorkspacePicker  *WorkspacePicker

	outerFlex          *tview.Flex
	contentFlex        *tview.Flex
	mainFlex           *tview.Flex
	pickerModal        tview.Primitive
	reactionModal      tview.Primitive
	fileModal          tview.Primitive
	searchModal        tview.Primitive
	pinsModal          tview.Primitive
	starredModal       tview.Primitive
	userProfileModal   tview.Primitive
	channelInfoModal   tview.Primitive
	workspaceModal     tview.Primitive
	activePanel        Panel
	onMarkRead         func()
	onMarkAllRead      func()
	onPinnedMessages   func()
	onStarredItems     func()
	onChannelInfo      func()
	channelsVisible    bool
	threadVisible      bool
	pickerVisible      bool
	reactionVisible    bool
	filePickerVisible  bool
	searchVisible      bool
	pinsVisible        bool
	starredVisible     bool
	userProfileVisible bool
	channelInfoVisible bool
	commandBarVisible  bool
	workspaceVisible   bool
	onSwitchWorkspace  func(workspaceID string)
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

	// Search picker (modal overlay).
	v.SearchPicker = NewSearchPicker(cfg)
	v.SearchPicker.SetOnClose(func() {
		v.HideSearchPicker()
	})

	// Centered modal wrapper for the search picker.
	v.searchModal = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(v.SearchPicker, 80, 0, true).
			AddItem(nil, 0, 1, false),
			0, 2, true).
		AddItem(nil, 0, 1, false)

	// Pins picker (modal overlay).
	v.PinsPicker = NewPinsPicker(cfg)
	v.PinsPicker.SetOnClose(func() {
		v.HidePinsPicker()
	})

	// Centered modal wrapper for the pins picker.
	v.pinsModal = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(v.PinsPicker, 80, 0, true).
			AddItem(nil, 0, 1, false),
			0, 2, true).
		AddItem(nil, 0, 1, false)

	// Starred items picker (modal overlay).
	v.StarredPicker = NewStarredPicker(cfg)
	v.StarredPicker.SetOnClose(func() {
		v.HideStarredPicker()
	})

	// Centered modal wrapper for the starred picker.
	v.starredModal = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(v.StarredPicker, 80, 0, true).
			AddItem(nil, 0, 1, false),
			0, 2, true).
		AddItem(nil, 0, 1, false)

	// User profile panel (modal overlay).
	v.UserProfilePanel = NewUserProfilePanel(cfg)
	v.UserProfilePanel.SetOnClose(func() {
		v.HideUserProfile()
	})

	// Centered modal wrapper for the user profile panel.
	v.userProfileModal = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(v.UserProfilePanel, 50, 0, true).
			AddItem(nil, 0, 1, false),
			0, 2, true).
		AddItem(nil, 0, 1, false)

	// Channel info panel (modal overlay).
	v.ChannelInfoPanel = NewChannelInfoPanel(cfg)
	v.ChannelInfoPanel.SetOnClose(func() {
		v.HideChannelInfo()
	})

	// Centered modal wrapper for the channel info panel.
	v.channelInfoModal = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(v.ChannelInfoPanel, 60, 0, true).
			AddItem(nil, 0, 1, false),
			0, 2, true).
		AddItem(nil, 0, 1, false)

	// Workspace picker (modal overlay).
	v.WorkspacePicker = NewWorkspacePicker(cfg)
	v.WorkspacePicker.SetOnClose(func() {
		v.HideWorkspacePicker()
	})

	// Centered modal wrapper for the workspace picker.
	v.workspaceModal = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(v.WorkspacePicker, 50, 0, true).
			AddItem(nil, 0, 1, false),
			0, 2, true).
		AddItem(nil, 0, 1, false)

	// Command bar (vim-style, hidden by default).
	v.CommandBar = NewCommandBar(cfg)
	v.CommandBar.SetOnClose(func() {
		v.HideCommandBar()
	})

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

// SetOnMarkRead sets the callback invoked when the user presses the mark-read key.
func (v *View) SetOnMarkRead(fn func()) {
	v.onMarkRead = fn
}

// SetOnMarkAllRead sets the callback invoked when the user presses the mark-all-read key.
func (v *View) SetOnMarkAllRead(fn func()) {
	v.onMarkAllRead = fn
}

// SetOnPinnedMessages sets the callback invoked when the user opens the pinned messages popup.
func (v *View) SetOnPinnedMessages(fn func()) {
	v.onPinnedMessages = fn
}

// SetOnStarredItems sets the callback invoked when the user opens the starred items popup.
func (v *View) SetOnStarredItems(fn func()) {
	v.onStarredItems = fn
}

// SetOnChannelInfo sets the callback invoked when the user opens the channel info panel.
func (v *View) SetOnChannelInfo(fn func()) {
	v.onChannelInfo = fn
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

	// Toggle search picker (Ctrl+S — non-rune, always works).
	if name == v.cfg.Keybinds.Search {
		if v.searchVisible {
			v.HideSearchPicker()
		} else {
			v.ShowSearchPicker()
		}
		return nil
	}

	// Toggle workspace picker (Ctrl+T — non-rune, always works).
	if name == v.cfg.Keybinds.SwitchTeam {
		if v.workspaceVisible {
			v.HideWorkspacePicker()
		} else {
			v.ShowWorkspacePicker()
		}
		return nil
	}

	// Toggle channel info (Ctrl+O — non-rune, always works).
	if name == v.cfg.Keybinds.ChannelInfo {
		if v.channelInfoVisible {
			v.HideChannelInfo()
		} else {
			v.ShowChannelInfo()
			if v.onChannelInfo != nil {
				v.onChannelInfo()
			}
		}
		return nil
	}

	// Focus cycling (Ctrl-based, works even in text input).
	if name == v.cfg.Keybinds.FocusPrevious {
		v.cycleFocus(-1)
		return nil
	}
	if name == v.cfg.Keybinds.FocusNext {
		v.cycleFocus(1)
		return nil
	}

	// When a modal or command bar is visible, all other keys go to its input.
	if v.pickerVisible || v.reactionVisible || v.filePickerVisible || v.searchVisible || v.pinsVisible || v.starredVisible || v.userProfileVisible || v.channelInfoVisible || v.commandBarVisible || v.workspaceVisible {
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
	case v.cfg.Keybinds.MarkRead:
		if v.onMarkRead != nil {
			v.onMarkRead()
		}
		return nil
	case v.cfg.Keybinds.MarkAllRead:
		if v.onMarkAllRead != nil {
			v.onMarkAllRead()
		}
		return nil
	case v.cfg.Keybinds.PinnedMessages:
		if v.pinsVisible {
			v.HidePinsPicker()
		} else {
			v.ShowPinsPicker()
			if v.onPinnedMessages != nil {
				v.onPinnedMessages()
			}
		}
		return nil
	case v.cfg.Keybinds.StarredItems:
		if v.starredVisible {
			v.HideStarredPicker()
		} else {
			v.ShowStarredPicker()
			if v.onStarredItems != nil {
				v.onStarredItems()
			}
		}
		return nil
	case v.cfg.Keybinds.CommandMode:
		v.ShowCommandBar()
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

// ShowSearchPicker shows the search picker modal overlay.
func (v *View) ShowSearchPicker() {
	v.searchVisible = true
	v.SearchPicker.Reset()
	v.Pages.AddPage("search", v.searchModal, true, true)
	v.app.SetFocus(v.SearchPicker.input)
}

// HideSearchPicker hides the search picker and restores focus.
func (v *View) HideSearchPicker() {
	v.searchVisible = false
	v.Pages.RemovePage("search")
	v.FocusPanel(v.activePanel)
}

// ShowPinsPicker shows the pinned messages picker modal overlay.
func (v *View) ShowPinsPicker() {
	v.pinsVisible = true
	v.PinsPicker.Reset()
	v.Pages.AddPage("pins", v.pinsModal, true, true)
	v.app.SetFocus(v.PinsPicker.list)
}

// HidePinsPicker hides the pinned messages picker and restores focus.
func (v *View) HidePinsPicker() {
	v.pinsVisible = false
	v.Pages.RemovePage("pins")
	v.FocusPanel(v.activePanel)
}

// ShowStarredPicker shows the starred items picker modal overlay.
func (v *View) ShowStarredPicker() {
	v.starredVisible = true
	v.StarredPicker.Reset()
	v.Pages.AddPage("starred", v.starredModal, true, true)
	v.app.SetFocus(v.StarredPicker.list)
}

// HideStarredPicker hides the starred items picker and restores focus.
func (v *View) HideStarredPicker() {
	v.starredVisible = false
	v.Pages.RemovePage("starred")
	v.FocusPanel(v.activePanel)
}

// ShowUserProfile shows the user profile panel modal overlay.
func (v *View) ShowUserProfile() {
	v.userProfileVisible = true
	v.UserProfilePanel.Reset()
	v.Pages.AddPage("userprofile", v.userProfileModal, true, true)
	v.app.SetFocus(v.UserProfilePanel.content)
}

// HideUserProfile hides the user profile panel and restores focus.
func (v *View) HideUserProfile() {
	v.userProfileVisible = false
	v.Pages.RemovePage("userprofile")
	v.FocusPanel(v.activePanel)
}

// ShowChannelInfo shows the channel info panel modal overlay.
func (v *View) ShowChannelInfo() {
	v.channelInfoVisible = true
	v.ChannelInfoPanel.Reset()
	v.Pages.AddPage("channelinfo", v.channelInfoModal, true, true)
	v.app.SetFocus(v.ChannelInfoPanel.content)
}

// HideChannelInfo hides the channel info panel and restores focus.
func (v *View) HideChannelInfo() {
	v.channelInfoVisible = false
	v.Pages.RemovePage("channelinfo")
	v.FocusPanel(v.activePanel)
}

// SetOnSwitchWorkspace sets the callback for workspace switching.
func (v *View) SetOnSwitchWorkspace(fn func(workspaceID string)) {
	v.onSwitchWorkspace = fn
	v.WorkspacePicker.SetOnSelect(func(id string) {
		v.HideWorkspacePicker()
		if fn != nil {
			fn(id)
		}
	})
}

// ShowWorkspacePicker shows the workspace picker modal overlay.
func (v *View) ShowWorkspacePicker() {
	v.workspaceVisible = true
	v.WorkspacePicker.Reset()
	v.Pages.AddPage("workspace", v.workspaceModal, true, true)
	v.app.SetFocus(v.WorkspacePicker.list)
}

// HideWorkspacePicker hides the workspace picker and restores focus.
func (v *View) HideWorkspacePicker() {
	v.workspaceVisible = false
	v.Pages.RemovePage("workspace")
	v.FocusPanel(v.activePanel)
}

// ShowCommandBar shows the vim-style command bar at the bottom.
func (v *View) ShowCommandBar() {
	v.commandBarVisible = true
	v.CommandBar.Reset()
	// Replace status bar with command bar.
	v.outerFlex.RemoveItem(v.StatusBar)
	v.outerFlex.AddItem(v.CommandBar, 1, 0, true)
	v.app.SetFocus(v.CommandBar)
}

// HideCommandBar hides the command bar and restores the status bar.
func (v *View) HideCommandBar() {
	v.commandBarVisible = false
	v.outerFlex.RemoveItem(v.CommandBar)
	v.outerFlex.AddItem(v.StatusBar, 1, 0, false)
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

// cycleFocus moves focus to the next or previous panel.
// direction: +1 for next, -1 for previous.
func (v *View) cycleFocus(direction int) {
	panels := []Panel{PanelChannels, PanelMessages, PanelInput}
	if v.threadVisible {
		panels = append(panels, PanelThread)
	}
	if !v.channelsVisible {
		panels = panels[1:] // skip PanelChannels
	}

	cur := 0
	for i, p := range panels {
		if p == v.activePanel {
			cur = i
			break
		}
	}
	next := (cur + direction + len(panels)) % len(panels)
	v.FocusPanel(panels[next])
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
	focusedTitleFg, _, _ := v.cfg.Theme.Title.Focused.Style.Decompose()
	normalTitleFg, _, _ := v.cfg.Theme.Title.Normal.Style.Decompose()

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
		})
		if p.panel == v.activePanel {
			box.SetBorderColor(focusedFg)
			box.SetTitleColor(focusedTitleFg)
		} else {
			box.SetBorderColor(normalFg)
			box.SetTitleColor(normalTitleFg)
		}
	}
}
