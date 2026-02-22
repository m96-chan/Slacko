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
)

// View is the main chat layout containing all panels.
type View struct {
	*tview.Flex
	app        *tview.Application
	cfg        *config.Config
	StatusBar  *StatusBar

	ChannelTree *tview.TreeView
	Header      *tview.TextView
	Messages    *tview.TextView
	Input       *tview.TextArea

	contentFlex     *tview.Flex
	mainFlex        *tview.Flex
	activePanel     Panel
	channelsVisible bool
}

// New creates the main chat view with the full flex layout.
//
// Layout:
//
//	Outer Flex (FlexRow)
//	├── mainFlex (FlexColumn)
//	│   ├── ChannelTree (fixed 30 cols)
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
	v.ChannelTree = tview.NewTreeView()
	root := tview.NewTreeNode("Channels")
	v.ChannelTree.SetRoot(root).SetCurrentNode(root)
	v.ChannelTree.SetBorder(true).SetTitle(" Channels ")

	// Header (channel name + topic).
	v.Header = tview.NewTextView().
		SetDynamicColors(true)
	v.Header.SetBorder(false)

	// Messages area.
	v.Messages = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	v.Messages.SetBorder(true).SetTitle(" Messages ")

	// Input area.
	v.Input = tview.NewTextArea()
	v.Input.SetBorder(true).SetTitle(" Input ")
	v.Input.SetPlaceholder("Type a message...")

	// Status bar.
	v.StatusBar = NewStatusBar(cfg)

	// Content flex (right side): header, messages, input stacked vertically.
	v.contentFlex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.Header, 1, 0, false).
		AddItem(v.Messages, 0, 1, false).
		AddItem(v.Input, 3, 0, false)

	// Main flex (horizontal): channel tree + content.
	v.mainFlex = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(v.ChannelTree, 30, 0, false).
		AddItem(v.contentFlex, 0, 1, false)

	// Outer flex (vertical): main + status bar.
	v.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.mainFlex, 0, 1, false).
		AddItem(v.StatusBar, 1, 0, false)

	// Default focus on messages.
	v.activePanel = PanelMessages
	v.applyBorderStyles()

	return v
}

// FocusPanel sets focus to the given panel and updates border colors.
func (v *View) FocusPanel(panel Panel) {
	v.activePanel = panel
	v.applyBorderStyles()

	switch panel {
	case PanelChannels:
		v.app.SetFocus(v.ChannelTree)
	case PanelMessages:
		v.app.SetFocus(v.Messages)
	case PanelInput:
		v.app.SetFocus(v.Input)
	}
}

// HandleKey processes chat-level keybindings. Returns nil to consume the event.
func (v *View) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	// Toggle channels sidebar.
	if name == v.cfg.Keybinds.ToggleChannels {
		v.ToggleChannels()
		return nil
	}

	// Skip Rune-based focus keybinds when input is active so the user can type.
	if v.activePanel == PanelInput && event.Key() == tcell.KeyRune {
		return event
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

// SetChannelHeader updates the header with channel name and topic.
func (v *View) SetChannelHeader(name, topic string) {
	text := fmt.Sprintf(" [::b]#%s[::-]", name)
	if topic != "" {
		text += fmt.Sprintf("  —  %s", topic)
	}
	v.Header.SetText(text)
}

// rebuildMainFlex reconstructs the main flex after toggling the channel tree.
// tview has no InsertItem, so we Clear() and re-add items.
func (v *View) rebuildMainFlex() {
	v.mainFlex.Clear()
	if v.channelsVisible {
		v.mainFlex.AddItem(v.ChannelTree, 30, 0, false)
	}
	v.mainFlex.AddItem(v.contentFlex, 0, 1, false)
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
		{v.ChannelTree, PanelChannels},
		{v.Messages, PanelMessages},
		{v.Input, PanelInput},
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
