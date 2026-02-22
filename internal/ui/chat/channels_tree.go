package chat

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/slack-go/slack"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/ui/keys"
)

// ChannelType classifies a Slack channel.
type ChannelType int

const (
	ChannelTypePublic ChannelType = iota
	ChannelTypePrivate
	ChannelTypeDM
	ChannelTypeGroupDM
	ChannelTypeShared // Slack Connect (externally shared)
)

// nodeRef stores metadata for a tree node, used as tview.TreeNode.Reference.
type nodeRef struct {
	ChannelID string
	UserID    string // populated for DMs (presence lookups)
}

// OnChannelSelectedFunc is called when the user selects a channel.
type OnChannelSelectedFunc func(channelID string)

// OnCopyChannelIDFunc is called when the user wants to copy a channel ID.
type OnCopyChannelIDFunc func(channelID string)

// ChannelsTree is a tree view that categorises channels into sections.
type ChannelsTree struct {
	*tview.TreeView
	cfg             *config.Config
	root            *tview.TreeNode
	sections        map[ChannelType]*tview.TreeNode
	nodeIndex       map[string]*tview.TreeNode // channelID → node
	channelIDs      map[*tview.TreeNode]string // node → channelID (reverse)
	unreadCounts    map[string]int             // channelID → unread count
	onSelected      OnChannelSelectedFunc
	onCopyChannelID OnCopyChannelIDFunc
}

// NewChannelsTree creates a tree with four section headers.
func NewChannelsTree(cfg *config.Config, onSelected OnChannelSelectedFunc) *ChannelsTree {
	ct := &ChannelsTree{
		TreeView:     tview.NewTreeView(),
		cfg:          cfg,
		nodeIndex:    make(map[string]*tview.TreeNode),
		channelIDs:   make(map[*tview.TreeNode]string),
		unreadCounts: make(map[string]int),
		onSelected:   onSelected,
	}

	ct.root = tview.NewTreeNode("")
	ct.SetRoot(ct.root)
	ct.SetTopLevel(1)
	ct.SetGraphics(false)
	ct.SetBorder(true).SetTitle(" Channels ")

	// Create section headers.
	ct.sections = map[ChannelType]*tview.TreeNode{
		ChannelTypePublic:  tview.NewTreeNode("Starred"),
		ChannelTypePrivate: tview.NewTreeNode("Channels"),
		ChannelTypeShared:  tview.NewTreeNode("Slack Connect"),
		ChannelTypeDM:      tview.NewTreeNode("Direct Messages"),
		ChannelTypeGroupDM: tview.NewTreeNode("Group DMs"),
	}

	// Add sections in display order.
	for _, ct2 := range []ChannelType{ChannelTypePublic, ChannelTypePrivate, ChannelTypeShared, ChannelTypeDM, ChannelTypeGroupDM} {
		node := ct.sections[ct2]
		node.SetSelectable(true)
		node.SetExpanded(true)
		ct.root.AddChild(node)
	}

	ct.SetSelectedFunc(func(node *tview.TreeNode) {
		if _, ok := ct.channelIDs[node]; ok && ct.onSelected != nil {
			ct.onSelected(ct.channelIDs[node])
		}
	})

	ct.SetInputCapture(ct.handleInput)

	return ct
}

// SetOnChannelSelected sets the callback for channel selection.
func (ct *ChannelsTree) SetOnChannelSelected(fn OnChannelSelectedFunc) {
	ct.onSelected = fn
}

// SetOnCopyChannelID sets the callback for copying a channel ID.
func (ct *ChannelsTree) SetOnCopyChannelID(fn OnCopyChannelIDFunc) {
	ct.onCopyChannelID = fn
}

// Populate clears and rebuilds the tree from the given channel/user data.
func (ct *ChannelsTree) Populate(channels []slack.Channel, users map[string]slack.User, selfUserID string) {
	// Clear existing children from each section.
	for _, section := range ct.sections {
		section.ClearChildren()
	}
	ct.nodeIndex = make(map[string]*tview.TreeNode)
	ct.channelIDs = make(map[*tview.TreeNode]string)
	ct.unreadCounts = make(map[string]int)

	// Sort channels: public/private by name, DMs/group DMs by display name.
	sorted := make([]slack.Channel, len(channels))
	copy(sorted, channels)
	sort.Slice(sorted, func(i, j int) bool {
		return channelSortKey(sorted[i], users, selfUserID) < channelSortKey(sorted[j], users, selfUserID)
	})

	for _, ch := range sorted {
		ct.addChannelNode(ch, users, selfUserID)
	}

	// Set initial selection to the first channel node if one exists.
	ct.setInitialSelection()
}

// AddChannel inserts a single channel into the correct section.
func (ct *ChannelsTree) AddChannel(ch slack.Channel, users map[string]slack.User, selfUserID string) {
	if _, exists := ct.nodeIndex[ch.ID]; exists {
		return
	}
	ct.addChannelNode(ch, users, selfUserID)
}

// RemoveChannel removes a channel from the tree.
func (ct *ChannelsTree) RemoveChannel(channelID string) {
	node, ok := ct.nodeIndex[channelID]
	if !ok {
		return
	}

	// Find and remove from parent section. tview has no RemoveChild,
	// so we clear and re-add all children except the removed one.
	for _, section := range ct.sections {
		children := section.GetChildren()
		for i, child := range children {
			if child == node {
				section.ClearChildren()
				for j, c := range children {
					if j != i {
						section.AddChild(c)
					}
				}
				break
			}
		}
	}

	delete(ct.channelIDs, node)
	delete(ct.nodeIndex, channelID)
}

// RenameChannel updates the display text for a channel.
func (ct *ChannelsTree) RenameChannel(channelID, newName string) {
	node, ok := ct.nodeIndex[channelID]
	if !ok {
		return
	}

	// Preserve icon prefix: find the first space and replace everything after it.
	text := node.GetText()
	if idx := strings.Index(text, " "); idx >= 0 {
		node.SetText(text[:idx+1] + newName)
	} else {
		node.SetText(newName)
	}
}

// SetUnread toggles the unread style on a channel node.
func (ct *ChannelsTree) SetUnread(channelID string, unread bool) {
	if unread {
		ct.SetUnreadCount(channelID, 1)
	} else {
		ct.SetUnreadCount(channelID, 0)
	}
}

// SetUnreadCount updates the unread count badge on a channel node.
// count > 0: set bold style and show "(N)" badge.
// count == 0: clear style and badge.
// count == -1: increment existing count by 1.
func (ct *ChannelsTree) SetUnreadCount(channelID string, count int) {
	node, ok := ct.nodeIndex[channelID]
	if !ok {
		return
	}

	if count == -1 {
		ct.unreadCounts[channelID]++
	} else {
		ct.unreadCounts[channelID] = count
	}

	actual := ct.unreadCounts[channelID]
	base := stripBadge(node.GetText())

	if actual > 0 {
		node.SetText(fmt.Sprintf("%s (%d)", base, actual))
		node.SetTextStyle(ct.cfg.Theme.ChannelsTree.Unread.Style)
	} else {
		node.SetText(base)
		node.SetTextStyle(ct.cfg.Theme.ChannelsTree.Channel.Style)
		delete(ct.unreadCounts, channelID)
	}
}

// badgeRe matches a trailing " (N)" badge on node text.
var badgeRe = regexp.MustCompile(` \(\d+\)$`)

// stripBadge removes a trailing " (N)" badge from node text.
func stripBadge(text string) string {
	return badgeRe.ReplaceAllString(text, "")
}

// UnreadCount returns the current unread count for a channel.
func (ct *ChannelsTree) UnreadCount(channelID string) int {
	return ct.unreadCounts[channelID]
}

// handleInput processes c (collapse) and p (parent) keys.
func (ct *ChannelsTree) handleInput(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	switch name {
	case ct.cfg.Keybinds.ChannelsTree.Collapse:
		current := ct.GetCurrentNode()
		if current == nil {
			return event
		}
		// If current is a section header, toggle its expansion.
		for _, section := range ct.sections {
			if current == section {
				section.SetExpanded(!section.IsExpanded())
				return nil
			}
		}
		// If current is a channel node, toggle its parent section.
		ref := current.GetReference()
		if ref != nil {
			for _, section := range ct.sections {
				for _, child := range section.GetChildren() {
					if child == current {
						section.SetExpanded(!section.IsExpanded())
						return nil
					}
				}
			}
		}
		return nil

	case ct.cfg.Keybinds.ChannelsTree.MoveToParent:
		current := ct.GetCurrentNode()
		if current == nil {
			return event
		}
		// If current is a channel node, move to its parent section.
		for _, section := range ct.sections {
			for _, child := range section.GetChildren() {
				if child == current {
					ct.SetCurrentNode(section)
					return nil
				}
			}
		}
		return nil

	case ct.cfg.Keybinds.ChannelsTree.CopyChannelID:
		current := ct.GetCurrentNode()
		if current == nil {
			return event
		}
		if chID, ok := ct.channelIDs[current]; ok && ct.onCopyChannelID != nil {
			ct.onCopyChannelID(chID)
			return nil
		}
	}

	return event
}

// addChannelNode creates a node for a channel and adds it to the correct section.
func (ct *ChannelsTree) addChannelNode(ch slack.Channel, users map[string]slack.User, selfUserID string) {
	chType := classifyChannel(ch)
	text := channelDisplayText(ch, chType, users, selfUserID, ct.cfg.AsciiIcons)

	node := tview.NewTreeNode(text)
	node.SetReference(&nodeRef{
		ChannelID: ch.ID,
		UserID:    ch.User,
	})
	node.SetSelectable(true)

	// Apply channel theme style.
	node.SetTextStyle(ct.cfg.Theme.ChannelsTree.Channel.Style)

	// Map to the correct section.
	var section *tview.TreeNode
	switch chType {
	case ChannelTypePublic, ChannelTypePrivate:
		section = ct.sections[ChannelTypePrivate] // "Channels" section
	case ChannelTypeShared:
		section = ct.sections[ChannelTypeShared] // "Slack Connect" section
	case ChannelTypeDM:
		section = ct.sections[ChannelTypeDM]
	case ChannelTypeGroupDM:
		section = ct.sections[ChannelTypeGroupDM]
	}

	section.AddChild(node)
	ct.nodeIndex[ch.ID] = node
	ct.channelIDs[node] = ch.ID
}

// setInitialSelection sets the current node to the first channel node.
func (ct *ChannelsTree) setInitialSelection() {
	for _, ct2 := range []ChannelType{ChannelTypePrivate, ChannelTypeShared, ChannelTypeDM, ChannelTypeGroupDM} {
		section := ct.sections[ct2]
		if children := section.GetChildren(); len(children) > 0 {
			ct.SetCurrentNode(children[0])
			return
		}
	}
}

// classifyChannel determines the type of a Slack channel.
func classifyChannel(ch slack.Channel) ChannelType {
	if ch.IsIM {
		return ChannelTypeDM
	}
	if ch.IsMpIM {
		return ChannelTypeGroupDM
	}
	// Slack Connect: externally shared channels.
	if ch.IsExtShared {
		return ChannelTypeShared
	}
	if ch.IsPrivate {
		return ChannelTypePrivate
	}
	return ChannelTypePublic
}

// channelDisplayText returns the display text for a channel node.
func channelDisplayText(ch slack.Channel, chType ChannelType, users map[string]slack.User, selfUserID string, asciiIcons bool) string {
	pIcon := presenceIcon
	if asciiIcons {
		pIcon = presenceIconASCII
	}
	switch chType {
	case ChannelTypeDM:
		user, ok := users[ch.User]
		if !ok {
			return fmt.Sprintf("%s %s", pIcon(""), ch.User)
		}
		return fmt.Sprintf("%s %s", pIcon(user.Presence), dmDisplayName(user))
	case ChannelTypeGroupDM:
		groupIcon := "\U0001F465" // 👥
		if asciiIcons {
			groupIcon = "++"
		}
		if ch.Purpose.Value != "" {
			return fmt.Sprintf("%s %s", groupIcon, ch.Purpose.Value)
		}
		if ch.Name != "" {
			return fmt.Sprintf("%s %s", groupIcon, ch.Name)
		}
		return fmt.Sprintf("%s Group DM", groupIcon)
	case ChannelTypeShared:
		linkIcon := "\U0001F517" // 🔗
		if asciiIcons {
			linkIcon = "<>"
		}
		return fmt.Sprintf("%s %s", linkIcon, ch.Name)
	case ChannelTypePrivate:
		lockIcon := "\U0001F512" // 🔒
		if asciiIcons {
			lockIcon = "@"
		}
		return fmt.Sprintf("%s %s", lockIcon, ch.Name)
	default: // Public
		return fmt.Sprintf("# %s", ch.Name)
	}
}

// presenceIcon returns a presence dot based on the user's status.
func presenceIcon(presence string) string {
	switch presence {
	case "active":
		return "●"
	case "away":
		return "◐"
	default:
		return "○"
	}
}

// presenceIconASCII returns an ASCII presence indicator.
func presenceIconASCII(presence string) string {
	switch presence {
	case "active":
		return "*"
	case "away":
		return "~"
	default:
		return "o"
	}
}

// dmDisplayName returns the best display name for a user.
func dmDisplayName(user slack.User) string {
	if user.Profile.DisplayName != "" {
		return user.Profile.DisplayName
	}
	if user.RealName != "" {
		return user.RealName
	}
	if user.Name != "" {
		return user.Name
	}
	return user.ID
}

// UpdateUserPresence updates the presence icon on DM nodes for the given user.
func (ct *ChannelsTree) UpdateUserPresence(userID, presence string) {
	section := ct.sections[ChannelTypeDM]
	for _, node := range section.GetChildren() {
		ref, ok := node.GetReference().(*nodeRef)
		if !ok || ref.UserID != userID {
			continue
		}
		// Rebuild display text: replace the presence icon prefix.
		text := node.GetText()
		if idx := strings.Index(text, " "); idx >= 0 {
			node.SetText(fmt.Sprintf("%s%s", presenceIcon(presence), text[idx:]))
		}
		return
	}
}

// channelSortKey returns a string used for sorting channels.
func channelSortKey(ch slack.Channel, users map[string]slack.User, selfUserID string) string {
	if ch.IsIM {
		if u, ok := users[ch.User]; ok {
			return strings.ToLower(dmDisplayName(u))
		}
		return ch.User
	}
	return strings.ToLower(ch.Name)
}
