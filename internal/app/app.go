package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	gokeyring "github.com/zalando/go-keyring"

	"github.com/m96-chan/Slacko/internal/clipboard"
	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/keyring"
	"github.com/m96-chan/Slacko/internal/markdown"
	"github.com/m96-chan/Slacko/internal/notifications"
	slackclient "github.com/m96-chan/Slacko/internal/slack"
	"github.com/m96-chan/Slacko/internal/typing"
	"github.com/m96-chan/Slacko/internal/ui/chat"
	"github.com/m96-chan/Slacko/internal/ui/keys"
	"github.com/m96-chan/Slacko/internal/ui/login"
	"github.com/rivo/tview"
)

// App is the top-level application struct.
type App struct {
	Config         *config.Config
	tview          *tview.Application
	slack          *slackclient.Client
	chatView       *chat.View
	notifier       *notifications.Notifier
	cancel         context.CancelFunc
	channels       []slack.Channel
	users          map[string]slack.User
	dmSet          map[string]bool            // set of DM channel IDs
	lastRead       map[string]string          // channelID → last-read timestamp
	pinnedMsgs     map[string]map[string]bool // channelID → set of pinned timestamps
	currentChannel string
	typingTracker  *typing.Tracker
	mu             sync.Mutex
}

// New creates a new App with the given config.
func New(cfg *config.Config) *App {
	return &App{
		Config:     cfg,
		tview:      tview.NewApplication(),
		users:      make(map[string]slack.User),
		dmSet:      make(map[string]bool),
		lastRead:   make(map[string]string),
		pinnedMsgs: make(map[string]map[string]bool),
		notifier:   notifications.New(),
	}
}

// Run starts the TUI event loop. It attempts to authenticate using stored
// tokens and shows the login form when tokens are missing or invalid.
func (a *App) Run() error {
	a.tview.EnableMouse(a.Config.Mouse)

	// Set up OS signal handling for graceful shutdown.
	sigCtx, sigStop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCtx.Done()
		sigStop()
		a.shutdown()
	}()

	// Register global keybindings.
	a.tview.SetInputCapture(a.handleGlobalKey)

	user, userErr := keyring.GetUserToken()
	app, appErr := keyring.GetAppToken()

	if userErr == nil && appErr == nil {
		client, err := slackclient.New(user, app)
		if err != nil {
			slog.Warn("stored tokens invalid, showing login", "error", err)
			a.showLogin()
		} else {
			a.slack = client
			a.showMain()
		}
	} else {
		if userErr != nil && !errors.Is(userErr, gokeyring.ErrNotFound) {
			slog.Warn("error reading user token", "error", userErr)
		}
		if appErr != nil && !errors.Is(appErr, gokeyring.ErrNotFound) {
			slog.Warn("error reading app token", "error", appErr)
		}
		a.showLogin()
	}

	return a.tview.Run()
}

// shutdown tears down Socket Mode and stops the TUI.
func (a *App) shutdown() {
	if a.cancel != nil {
		a.cancel()
	}
	a.tview.Stop()
}

// logout deletes stored tokens and returns to the login screen.
// Must be called from the tview event loop (slash command or vim command handler).
func (a *App) logout() {
	// Stop Socket Mode.
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}

	// Delete tokens from keyring (best-effort).
	_ = keyring.DeleteUserToken()
	_ = keyring.DeleteAppToken()

	a.slack = nil
	a.chatView = nil
	a.showLogin()
}

// handleGlobalKey processes global keybindings. It returns nil to consume the
// event or the original event to let it propagate.
func (a *App) handleGlobalKey(event *tcell.EventKey) *tcell.EventKey {
	name := keys.Normalize(event.Name())

	if name == a.Config.Keybinds.Quit {
		a.shutdown()
		return nil
	}

	// Delegate to chat view if active.
	if a.chatView != nil {
		return a.chatView.HandleKey(event)
	}

	return event
}

// showLogin sets the root to the login form.
func (a *App) showLogin() {
	form := login.New(a.tview, a.Config, func(client *slackclient.Client) {
		a.slack = client
		a.showMain()
	})
	a.tview.SetRoot(form, true)
}

// showMain sets the root to the chat layout and starts Socket Mode in the
// background.
func (a *App) showMain() {
	// Cancel any previous Socket Mode connection.
	if a.cancel != nil {
		a.cancel()
	}

	a.chatView = chat.New(a.tview, a.Config)
	a.chatView.SetOnChannelSelected(a.onChannelSelected)

	// Wire message input callbacks.
	a.chatView.MessageInput.SetOnSend(a.onMessageSend)
	a.chatView.MessageInput.SetOnEdit(a.onMessageEdit)
	a.chatView.MessageInput.SetOnSlashCommand(a.executeSlashCommand)

	// Wire reply/edit triggers from messages list.
	a.chatView.MessagesList.SetOnReplyRequest(func(channelID, threadTS, userName string) {
		a.chatView.MessageInput.SetReplyContext(threadTS, userName)
		a.chatView.FocusPanel(chat.PanelInput)
	})
	a.chatView.MessagesList.SetOnEditRequest(func(channelID, timestamp, text string) {
		a.chatView.MessageInput.SetEditMode(timestamp, text)
		a.chatView.FocusPanel(chat.PanelInput)
	})

	// Wire thread open from messages list.
	a.chatView.MessagesList.SetOnThreadRequest(func(channelID, threadTS string) {
		a.chatView.OpenThread()
		go a.loadThread(channelID, threadTS)
	})

	// Wire thread view callbacks.
	a.chatView.ThreadView.SetOnSend(a.onThreadReplySend)
	a.chatView.ThreadView.SetOnClose(func() {
		a.chatView.CloseThread()
	})

	// Wire channel picker selection.
	a.chatView.ChannelsPicker.SetOnSelect(a.onChannelSelected)

	// Wire reaction picker.
	var reactionChannelID, reactionTimestamp string
	a.chatView.MessagesList.SetOnReactionAddRequest(func(channelID, timestamp string) {
		reactionChannelID = channelID
		reactionTimestamp = timestamp
		a.chatView.ShowReactionPicker()
	})
	a.chatView.ReactionsPicker.SetOnSelect(func(emojiName string) {
		chID := reactionChannelID
		ts := reactionTimestamp
		go func() {
			err := a.slack.AddReaction(emojiName, slack.NewRefToMessage(chID, ts))
			if err != nil {
				slog.Error("failed to add reaction", "channel", chID, "error", err)
			}
		}()
	})
	a.chatView.MessagesList.SetOnReactionRemoveRequest(func(channelID, timestamp, reaction string) {
		go func() {
			err := a.slack.RemoveReaction(reaction, slack.NewRefToMessage(channelID, timestamp))
			if err != nil {
				slog.Error("failed to remove reaction", "channel", channelID, "error", err)
			}
		}()
	})

	// Wire search picker.
	a.chatView.SearchPicker.SetOnSearch(func(query string) {
		go a.searchMessages(query)
	})
	a.chatView.SearchPicker.SetOnSelect(func(channelID, timestamp string) {
		a.chatView.HideSearchPicker()
		a.onChannelSelected(channelID)
	})

	// Wire file picker: Ctrl+F from input opens picker, selection triggers upload.
	a.chatView.MessageInput.SetOnOpenFilePicker(func() {
		a.chatView.ShowFilePicker()
	})
	a.chatView.FilePicker.SetOnSelect(func(filePath string) {
		channelID := a.chatView.MessageInput.ChannelID()
		if channelID == "" {
			return
		}
		threadTS := ""
		if a.chatView.MessageInput.Mode() == chat.InputModeReply {
			threadTS = a.chatView.MessageInput.ThreadTS()
		}
		go a.uploadFile(channelID, threadTS, filePath)
	})

	// Wire file open from messages list.
	a.chatView.MessagesList.SetOnFileOpenRequest(func(channelID string, file slack.File) {
		go a.openFile(file)
	})

	// Wire pins picker selection: jump to the channel/message.
	a.chatView.PinsPicker.SetOnSelect(func(channelID, timestamp string) {
		a.chatView.HidePinsPicker()
	})

	// Wire pin/unpin toggle from messages list.
	a.chatView.MessagesList.SetOnPinRequest(func(channelID, timestamp string, pin bool) {
		go func() {
			ref := slack.NewRefToMessage(channelID, timestamp)
			if pin {
				if err := a.slack.AddPin(channelID, ref); err != nil {
					slog.Error("failed to pin message", "channel", channelID, "error", err)
					return
				}
			} else {
				if err := a.slack.RemovePin(channelID, ref); err != nil {
					slog.Error("failed to unpin message", "channel", channelID, "error", err)
					return
				}
			}
			a.mu.Lock()
			if a.pinnedMsgs[channelID] == nil {
				a.pinnedMsgs[channelID] = make(map[string]bool)
			}
			if pin {
				a.pinnedMsgs[channelID][timestamp] = true
			} else {
				delete(a.pinnedMsgs[channelID], timestamp)
			}
			a.mu.Unlock()
			a.tview.QueueUpdateDraw(func() {
				a.chatView.MessagesList.SetPinned(timestamp, pin)
			})
		}()
	})

	// Wire star/unstar toggle from messages list.
	a.chatView.MessagesList.SetOnStarRequest(func(channelID, timestamp string, star bool) {
		go func() {
			ref := slack.NewRefToMessage(channelID, timestamp)
			if star {
				if err := a.slack.AddStar(channelID, ref); err != nil {
					slog.Error("failed to star message", "channel", channelID, "error", err)
					return
				}
			} else {
				if err := a.slack.RemoveStar(channelID, ref); err != nil {
					slog.Error("failed to unstar message", "channel", channelID, "error", err)
					return
				}
			}
			a.tview.QueueUpdateDraw(func() {
				a.chatView.MessagesList.SetStarred(timestamp, star)
				if star {
					a.chatView.StatusBar.SetTypingIndicator("Message starred")
				} else {
					a.chatView.StatusBar.SetTypingIndicator("Message unstarred")
				}
			})
			go func() {
				<-time.After(2 * time.Second)
				a.tview.QueueUpdateDraw(func() {
					a.chatView.StatusBar.SetTypingIndicator("")
				})
			}()
		}()
	})

	// Wire clipboard: yank (copy message text).
	a.chatView.MessagesList.SetOnYank(func(text string) {
		if err := clipboard.WriteText(text); err != nil {
			slog.Error("failed to copy to clipboard", "error", err)
			return
		}
		a.tview.QueueUpdateDraw(func() {
			a.chatView.StatusBar.SetTypingIndicator("Copied message text")
		})
		go func() {
			<-time.After(2 * time.Second)
			a.tview.QueueUpdateDraw(func() {
				a.chatView.StatusBar.SetTypingIndicator("")
			})
		}()
	})

	// Wire clipboard: copy permalink.
	a.chatView.MessagesList.SetOnCopyPermalink(func(channelID, timestamp string) {
		go func() {
			permalink, err := a.slack.GetPermalink(channelID, timestamp)
			if err != nil {
				slog.Error("failed to get permalink", "channel", channelID, "error", err)
				return
			}
			if err := clipboard.WriteText(permalink); err != nil {
				slog.Error("failed to copy permalink to clipboard", "error", err)
				return
			}
			a.tview.QueueUpdateDraw(func() {
				a.chatView.StatusBar.SetTypingIndicator("Copied permalink")
			})
			<-time.After(2 * time.Second)
			a.tview.QueueUpdateDraw(func() {
				a.chatView.StatusBar.SetTypingIndicator("")
			})
		}()
	})

	// Wire clipboard: copy channel ID.
	a.chatView.ChannelsTree.SetOnCopyChannelID(func(channelID string) {
		if err := clipboard.WriteText(channelID); err != nil {
			slog.Error("failed to copy channel ID to clipboard", "error", err)
			return
		}
		a.tview.QueueUpdateDraw(func() {
			a.chatView.StatusBar.SetTypingIndicator("Copied channel ID")
		})
		go func() {
			<-time.After(2 * time.Second)
			a.tview.QueueUpdateDraw(func() {
				a.chatView.StatusBar.SetTypingIndicator("")
			})
		}()
	})

	// Wire typing indicators.
	a.typingTracker = typing.NewTracker(func(channelID string) {
		a.mu.Lock()
		isCurrent := channelID == a.currentChannel
		a.mu.Unlock()
		if isCurrent && a.Config.TypingIndicator.Receive {
			status := a.typingTracker.FormatStatus(channelID)
			a.tview.QueueUpdateDraw(func() {
				a.chatView.StatusBar.SetTypingIndicator(status)
			})
		}
	})
	if a.Config.TypingIndicator.Send {
		a.chatView.MessageInput.SetOnTyping(func(channelID string) {
			// Typing send is a no-op until RTM support is added.
			slog.Debug("typing indicator send", "channel", channelID)
		})
	}

	// Wire pinned messages popup: fetch pins when user opens the popup.
	a.chatView.SetOnPinnedMessages(func() {
		a.mu.Lock()
		ch := a.currentChannel
		a.mu.Unlock()
		if ch == "" {
			return
		}
		a.chatView.PinsPicker.SetStatus("Loading...")
		go a.loadPinnedMessages(ch)
	})

	// Wire bookmarks popup: fetch bookmarks when user opens the popup.
	a.chatView.SetOnBookmarks(func() {
		a.mu.Lock()
		ch := a.currentChannel
		a.mu.Unlock()
		if ch == "" {
			return
		}
		a.chatView.BookmarksPicker.SetStatus("Loading...")
		go a.loadChannelBookmarks(ch)
	})
	a.chatView.BookmarksPicker.SetOnSelect(func(link string) {
		a.chatView.HideBookmarksPicker()
		go a.openURL(link)
	})

	// Wire starred items popup: fetch all starred items when user opens the popup.
	a.chatView.SetOnStarredItems(func() {
		a.chatView.StarredPicker.SetStatus("Loading...")
		go a.loadStarredItems()
	})
	a.chatView.StarredPicker.SetOnUnstar(func(channelID, timestamp string) {
		go func() {
			ref := slack.NewRefToMessage(channelID, timestamp)
			if err := a.slack.RemoveStar(channelID, ref); err != nil {
				slog.Error("failed to unstar message", "channel", channelID, "error", err)
			}
			// Also update the messages list if viewing the same channel.
			a.mu.Lock()
			isCurrent := channelID == a.currentChannel
			a.mu.Unlock()
			if isCurrent {
				a.tview.QueueUpdateDraw(func() {
					a.chatView.MessagesList.SetStarred(timestamp, false)
				})
			}
		}()
	})

	// Wire members picker: selecting a member opens their profile.
	a.chatView.MembersPicker.SetOnSelect(func(userID string) {
		a.chatView.HideMembersPicker()
		a.chatView.ShowUserProfile()
		a.chatView.UserProfilePanel.SetStatus("Loading...")
		go a.loadUserProfile(userID)
	})

	// Wire user profile panel.
	a.chatView.MessagesList.SetOnUserProfileRequest(func(userID string) {
		a.chatView.ShowUserProfile()
		a.chatView.UserProfilePanel.SetStatus("Loading...")
		go a.loadUserProfile(userID)
	})
	a.chatView.UserProfilePanel.SetOnOpenDM(func(userID string) {
		// Find or switch to the DM channel for this user.
		a.mu.Lock()
		var dmChannelID string
		for _, ch := range a.channels {
			if ch.IsIM && ch.User == userID {
				dmChannelID = ch.ID
				break
			}
		}
		a.mu.Unlock()
		if dmChannelID != "" {
			a.tview.QueueUpdateDraw(func() {
				a.chatView.HideUserProfile()
			})
			a.onChannelSelected(dmChannelID)
		}
	})
	a.chatView.UserProfilePanel.SetOnCopyID(func(userID string) {
		if err := clipboard.WriteText(userID); err != nil {
			slog.Error("failed to copy user ID to clipboard", "error", err)
			return
		}
		a.tview.QueueUpdateDraw(func() {
			a.chatView.UserProfilePanel.SetStatus("Copied user ID: " + userID)
		})
	})

	// Wire reaction users panel: resolve user IDs to names and display.
	a.chatView.MessagesList.SetOnViewReactionsRequest(func(channelID, timestamp string, reactions []slack.ItemReaction) {
		a.chatView.ShowReactionUsers()

		a.mu.Lock()
		users := a.users
		selfUserID := a.slack.UserID
		a.mu.Unlock()

		entries := make([]chat.ReactionUsersEntry, 0, len(reactions))
		for _, r := range reactions {
			names := make([]string, 0, len(r.Users))
			isSelf := false
			for _, uid := range r.Users {
				if uid == selfUserID {
					isSelf = true
				}
				if u, ok := users[uid]; ok {
					name := u.Profile.DisplayName
					if name == "" {
						name = u.RealName
					}
					if name == "" {
						name = u.Name
					}
					if name == "" {
						name = uid
					}
					names = append(names, name)
				} else {
					names = append(names, uid)
				}
			}
			entries = append(entries, chat.ReactionUsersEntry{
				Emoji:  markdown.LookupEmoji(r.Name),
				Name:   r.Name,
				Users:  names,
				IsSelf: isSelf,
			})
		}

		a.chatView.ReactionUsersPanel.SetReactions(entries)
		a.chatView.ReactionUsersPanel.SetStatus("[Esc]close")
	})

	// Wire workspace picker.
	a.chatView.SetOnSwitchWorkspace(func(workspaceID string) {
		go a.switchWorkspace(workspaceID)
	})

	// Wire invite picker: selecting a user invites them to the current channel.
	a.chatView.InvitePicker.SetOnSelect(func(userID string) {
		a.chatView.HideInvitePicker()
		a.mu.Lock()
		ch := a.currentChannel
		a.mu.Unlock()
		if ch == "" {
			return
		}
		go a.inviteUserToChannel(ch, userID)
	})

	// Wire group DM picker: confirming creates the group DM.
	a.chatView.GroupDMPicker.SetOnCreate(func(userIDs []string) {
		a.chatView.HideGroupDMPicker()
		go a.createGroupDM(userIDs)
	})

	// Wire vim command bar.
	a.chatView.CommandBar.SetOnExecute(func(command, args string) {
		a.chatView.HideCommandBar()
		a.executeVimCommand(command, args)
	})
	a.chatView.CommandBar.SetSetOptionNames(RuntimeOptionNames())

	// Wire channel create form.
	a.chatView.ChannelCreateForm.SetOnCreate(func(name string, isPrivate bool) {
		a.chatView.ChannelCreateForm.SetStatus("Creating channel...")
		go a.cmdCreateChannel(name, isPrivate)
	})

	// Wire channel info panel.
	a.chatView.SetOnChannelInfo(func() {
		a.mu.Lock()
		ch := a.currentChannel
		a.mu.Unlock()
		if ch == "" {
			return
		}
		a.chatView.ChannelInfoPanel.SetStatus("Loading...")
		go a.loadChannelInfo(ch)
	})
	a.chatView.ChannelInfoPanel.SetOnSetTopic(func(channelID string) {
		// Open editor for topic input using the external editor.
		go a.editChannelField(channelID, "topic")
	})
	a.chatView.ChannelInfoPanel.SetOnSetPurpose(func(channelID string) {
		go a.editChannelField(channelID, "purpose")
	})
	a.chatView.ChannelInfoPanel.SetOnLeave(func(channelID string) {
		go a.leaveChannel(channelID)
	})

	// Wire mark-as-read keybinds.
	a.chatView.SetOnMarkRead(func() {
		a.mu.Lock()
		ch := a.currentChannel
		a.mu.Unlock()
		if ch == "" {
			return
		}
		ts := a.chatView.MessagesList.LatestTimestamp()
		if ts != "" {
			a.markChannelRead(ch, ts)
		}
	})
	a.chatView.SetOnMarkAllRead(func() {
		go a.markAllChannelsRead()
	})

	a.chatView.StatusBar.SetConnectionStatus(
		fmt.Sprintf("%s (%s) — connecting...", a.slack.UserName, a.slack.TeamName))
	a.tview.SetRoot(a.chatView, true)
	a.chatView.FocusPanel(chat.PanelChannels)

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	handler := &slackclient.EventHandler{
		OnConnected: func() {
			slog.Info("socket mode connected")
			a.tview.QueueUpdateDraw(func() {
				a.chatView.StatusBar.SetConnectionStatus(
					fmt.Sprintf("%s (%s) — connected", a.slack.UserName, a.slack.TeamName))
			})

			// Fetch initial channel and user data.
			go a.fetchInitialData()
		},
		OnDisconnected: func() {
			slog.Warn("socket mode disconnected")
			a.tview.QueueUpdateDraw(func() {
				a.chatView.StatusBar.SetConnectionStatus(
					fmt.Sprintf("%s (%s) — disconnected", a.slack.UserName, a.slack.TeamName))
			})
		},
		OnError: func(err error) {
			slog.Error("socket mode error", "error", err)
			a.tview.QueueUpdateDraw(func() {
				a.chatView.StatusBar.SetConnectionStatus(
					fmt.Sprintf("%s (%s) — error: %s", a.slack.UserName, a.slack.TeamName, err.Error()))
			})
		},
		OnChannelCreated: func(evt *slackevents.ChannelCreatedEvent) {
			a.mu.Lock()
			users := a.users
			a.mu.Unlock()
			// Construct a minimal slack.Channel from the event.
			ch := slack.Channel{}
			ch.ID = evt.Channel.ID
			ch.Name = evt.Channel.Name
			ch.IsChannel = evt.Channel.IsChannel
			a.tview.QueueUpdateDraw(func() {
				a.chatView.ChannelsTree.AddChannel(ch, users, a.slack.UserID)
			})
		},
		OnChannelRename: func(evt *slackevents.ChannelRenameEvent) {
			a.tview.QueueUpdateDraw(func() {
				a.chatView.ChannelsTree.RenameChannel(evt.Channel.ID, evt.Channel.Name)
			})
		},
		OnChannelArchive: func(evt *slackevents.ChannelArchiveEvent) {
			a.tview.QueueUpdateDraw(func() {
				a.chatView.ChannelsTree.RemoveChannel(evt.Channel)
			})
		},
		OnMemberLeftChannel: func(evt *slackevents.MemberLeftChannelEvent) {
			if evt.User == a.slack.UserID {
				a.tview.QueueUpdateDraw(func() {
					a.chatView.ChannelsTree.RemoveChannel(evt.Channel)
				})
			}
		},
		OnMessage: func(evt *slackevents.MessageEvent) {
			msg := slack.Message{}
			msg.User = evt.User
			msg.Text = evt.Text
			msg.Timestamp = evt.TimeStamp
			msg.ThreadTimestamp = evt.ThreadTimeStamp
			msg.Channel = evt.Channel

			a.mu.Lock()
			isCurrent := evt.Channel == a.currentChannel
			a.mu.Unlock()

			a.tview.QueueUpdateDraw(func() {
				a.chatView.MessagesList.AppendMessage(evt.Channel, msg)
				// Increment reply count on parent message for thread replies.
				if evt.ThreadTimeStamp != "" && evt.TimeStamp != evt.ThreadTimeStamp {
					a.chatView.MessagesList.IncrementReplyCount(evt.Channel, evt.ThreadTimeStamp)
				}
				// Update thread view if this is a reply in the open thread.
				if a.chatView.ThreadView.IsOpen() &&
					a.chatView.ThreadView.ChannelID() == evt.Channel &&
					evt.ThreadTimeStamp != "" &&
					a.chatView.ThreadView.ThreadTS() == evt.ThreadTimeStamp {
					a.chatView.ThreadView.AppendReply(msg)
				}
				// Update unread badge for background channels.
				if !isCurrent {
					a.chatView.ChannelsTree.SetUnreadCount(evt.Channel, -1)
				}
			})

			// Auto-mark current channel as read.
			if isCurrent {
				go func() {
					if err := a.slack.MarkConversation(evt.Channel, evt.TimeStamp); err != nil {
						slog.Error("failed to mark conversation", "channel", evt.Channel, "error", err)
					}
				}()
				a.mu.Lock()
				a.lastRead[evt.Channel] = evt.TimeStamp
				a.mu.Unlock()
			}

			// Desktop notifications.
			a.maybeNotify(evt)
		},
		OnMessageChanged: func(evt *slackevents.MessageEvent) {
			if evt.Message == nil {
				return
			}
			a.tview.QueueUpdateDraw(func() {
				a.chatView.MessagesList.UpdateMessage(
					evt.Channel, evt.Message.Timestamp, evt.Message.Text)
				if a.chatView.ThreadView.IsOpen() &&
					a.chatView.ThreadView.ChannelID() == evt.Channel {
					a.chatView.ThreadView.UpdateReply(evt.Message.Timestamp, evt.Message.Text)
				}
			})
		},
		OnMessageDeleted: func(evt *slackevents.MessageEvent) {
			a.tview.QueueUpdateDraw(func() {
				a.chatView.MessagesList.RemoveMessage(
					evt.Channel, evt.PreviousMessage.Timestamp)
				if a.chatView.ThreadView.IsOpen() &&
					a.chatView.ThreadView.ChannelID() == evt.Channel {
					a.chatView.ThreadView.RemoveReply(evt.PreviousMessage.Timestamp)
				}
			})
		},
		OnReactionAdded: func(evt *slackevents.ReactionAddedEvent) {
			a.tview.QueueUpdateDraw(func() {
				a.chatView.MessagesList.AddReaction(
					evt.Item.Channel, evt.Item.Timestamp, evt.Reaction, evt.User)
			})
		},
		OnReactionRemoved: func(evt *slackevents.ReactionRemovedEvent) {
			a.tview.QueueUpdateDraw(func() {
				a.chatView.MessagesList.RemoveReaction(
					evt.Item.Channel, evt.Item.Timestamp, evt.Reaction, evt.User)
			})
		},
		OnPinAdded: func(evt *slackevents.PinAddedEvent) {
			channel := evt.Channel
			ts := ""
			if evt.Item.Message != nil {
				ts = evt.Item.Message.Timestamp
			}
			if ts == "" {
				return
			}
			a.mu.Lock()
			if a.pinnedMsgs[channel] == nil {
				a.pinnedMsgs[channel] = make(map[string]bool)
			}
			a.pinnedMsgs[channel][ts] = true
			isCurrent := channel == a.currentChannel
			a.mu.Unlock()
			if isCurrent {
				a.tview.QueueUpdateDraw(func() {
					a.chatView.MessagesList.SetPinned(ts, true)
				})
			}
		},
		OnPinRemoved: func(evt *slackevents.PinRemovedEvent) {
			channel := evt.Channel
			ts := ""
			if evt.Item.Message != nil {
				ts = evt.Item.Message.Timestamp
			}
			if ts == "" {
				return
			}
			a.mu.Lock()
			if a.pinnedMsgs[channel] != nil {
				delete(a.pinnedMsgs[channel], ts)
			}
			isCurrent := channel == a.currentChannel
			a.mu.Unlock()
			if isCurrent {
				a.tview.QueueUpdateDraw(func() {
					a.chatView.MessagesList.SetPinned(ts, false)
				})
			}
		},
		OnTyping: func(evt *slackclient.TypingEvent) {
			if a.typingTracker == nil || !a.Config.TypingIndicator.Receive {
				return
			}
			a.mu.Lock()
			userName := evt.UserID
			if u, ok := a.users[evt.UserID]; ok {
				if u.Profile.DisplayName != "" {
					userName = u.Profile.DisplayName
				} else if u.RealName != "" {
					userName = u.RealName
				} else if u.Name != "" {
					userName = u.Name
				}
			}
			a.mu.Unlock()
			a.typingTracker.Add(evt.ChannelID, evt.UserID, userName)
		},
		OnUserStatusChanged: func(evt *slackevents.UserStatusChangedEvent) {
			a.mu.Lock()
			if u, ok := a.users[evt.User.ID]; ok {
				u.Profile.StatusText = evt.User.Profile.StatusText
				u.Profile.StatusEmoji = evt.User.Profile.StatusEmoji
				u.RealName = evt.User.RealName
				if evt.User.Profile.DisplayName != "" {
					u.Profile.DisplayName = evt.User.Profile.DisplayName
				}
				a.users[evt.User.ID] = u
			}
			users := a.users
			a.mu.Unlock()

			a.tview.QueueUpdateDraw(func() {
				a.chatView.ChannelsTree.UpdateUserPresence(evt.User.ID, "")
				a.chatView.MessagesList.UpdateUsers(users)
				if a.chatView.ThreadView.IsOpen() {
					a.chatView.ThreadView.UpdateUsers(users)
				}
			})
		},
	}

	go func() {
		if err := a.slack.RunSocketMode(ctx, handler); err != nil {
			slog.Error("socket mode exited", "error", err)
		}
	}()
}

// fetchInitialData loads channels and users from Slack after connecting.
func (a *App) fetchInitialData() {
	channels, err := a.fetchAllChannels()
	if err != nil {
		slog.Error("failed to fetch channels", "error", err)
		return
	}

	users, err := a.slack.GetUsers()
	if err != nil {
		slog.Error("failed to fetch users", "error", err)
		return
	}

	userMap := make(map[string]slack.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	dmSet := make(map[string]bool)
	for _, ch := range channels {
		if ch.IsIM {
			dmSet[ch.ID] = true
		}
	}

	lastReadMap := make(map[string]string, len(channels))
	for _, ch := range channels {
		if ch.LastRead != "" {
			lastReadMap[ch.ID] = ch.LastRead
		}
	}

	a.mu.Lock()
	a.channels = channels
	a.users = userMap
	a.dmSet = dmSet
	a.lastRead = lastReadMap
	a.mu.Unlock()

	slog.Info("initial data loaded", "channels", len(channels), "users", len(users))

	// Migrate legacy tokens and populate workspace picker.
	if err := keyring.MigrateDefaultWorkspace(a.slack.TeamID, a.slack.TeamName); err != nil {
		slog.Warn("failed to migrate workspace to registry", "error", err)
	}
	a.populateWorkspacePicker()

	channelNames := make(map[string]string, len(channels))
	for _, ch := range channels {
		channelNames[ch.ID] = ch.Name
	}

	a.tview.QueueUpdateDraw(func() {
		a.chatView.ChannelsTree.Populate(channels, userMap, a.slack.UserID)
		a.chatView.ChannelsPicker.SetData(channels, userMap, a.slack.UserID)
		a.chatView.MentionsList.SetUsers(userMap)
		a.chatView.MentionsList.SetChannels(channels, userMap, a.slack.UserID)
		a.chatView.MentionsList.SetCommands(chat.BuiltinCommandEntries())
		a.chatView.MessagesList.SetSelfUserID(a.slack.UserID)
		a.chatView.MessagesList.SetSelfTeamID(a.slack.TeamID)
		a.chatView.ThreadView.SetSelfTeamID(a.slack.TeamID)
		a.chatView.MessagesList.SetChannelNames(channelNames)
		a.chatView.ThreadView.SetChannelNames(channelNames)
		// Set unread count badges for channels with unreads.
		for _, ch := range channels {
			if ch.UnreadCountDisplay > 0 {
				a.chatView.ChannelsTree.SetUnreadCount(ch.ID, ch.UnreadCountDisplay)
			}
		}
		a.chatView.StatusBar.SetConnectionStatus(
			fmt.Sprintf("%s (%s) — connected (%d channels, %d users)",
				a.slack.UserName, a.slack.TeamName, len(channels), len(users)))
	})
}

// onChannelSelected is called when the user selects a channel in the tree.
func (a *App) onChannelSelected(channelID string) {
	// Close thread if open when switching channels.
	if a.chatView.ThreadView.IsOpen() {
		a.chatView.CloseThread()
	}

	a.mu.Lock()
	a.currentChannel = channelID
	lr := a.lastRead[channelID]
	for _, ch := range a.channels {
		if ch.ID == channelID {
			a.chatView.SetChannelHeader(ch.Name, ch.Topic.Value)
			break
		}
	}
	a.mu.Unlock()

	a.chatView.MessagesList.SetLastRead(lr)
	a.chatView.MessageInput.SetChannel(channelID)
	// Clear typing indicator for the new channel.
	if a.typingTracker != nil {
		a.chatView.StatusBar.SetTypingIndicator("")
	}
	go a.loadMessages(channelID)
}

// loadMessages fetches conversation history and updates the messages list.
func (a *App) loadMessages(channelID string) {
	resp, err := a.slack.GetConversationHistory(&slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Limit:     a.Config.MessagesLimit,
	})
	if err != nil {
		slog.Error("failed to fetch messages", "channel", channelID, "error", err)
		return
	}

	a.mu.Lock()
	users := a.users
	a.mu.Unlock()

	a.tview.QueueUpdateDraw(func() {
		a.chatView.MessagesList.SetMessages(channelID, resp.Messages, users)
		a.updateChannelPresence(channelID, resp.Messages, users)
	})

	// Auto-mark channel as read with the newest message timestamp.
	if len(resp.Messages) > 0 {
		latestTS := resp.Messages[0].Timestamp // newest-first from API
		a.markChannelRead(channelID, latestTS)
	}

	// Fetch pinned messages for pin indicators.
	go a.fetchChannelPins(channelID)
}

// loadPinnedMessages fetches pinned messages for a channel and updates the UI.
func (a *App) loadPinnedMessages(channelID string) {
	items, err := a.slack.ListPins(channelID)
	if err != nil {
		slog.Error("failed to fetch pins", "channel", channelID, "error", err)
		a.tview.QueueUpdateDraw(func() {
			a.chatView.PinsPicker.SetStatus("Failed to load pins")
		})
		return
	}

	a.mu.Lock()
	users := a.users
	a.mu.Unlock()

	pinnedSet := make(map[string]bool)
	entries := make([]chat.PinnedEntry, 0, len(items))
	for _, item := range items {
		if item.Message == nil {
			continue
		}
		ts := item.Message.Timestamp
		pinnedSet[ts] = true
		userName := ""
		if u, ok := users[item.Message.User]; ok {
			if u.Profile.DisplayName != "" {
				userName = u.Profile.DisplayName
			} else if u.RealName != "" {
				userName = u.RealName
			} else {
				userName = u.Name
			}
		}
		if userName == "" {
			userName = item.Message.User
		}
		entries = append(entries, chat.PinnedEntry{
			ChannelID: channelID,
			Timestamp: ts,
			UserName:  userName,
			Text:      item.Message.Text,
		})
	}

	a.mu.Lock()
	a.pinnedMsgs[channelID] = pinnedSet
	a.mu.Unlock()

	a.tview.QueueUpdateDraw(func() {
		a.chatView.PinsPicker.SetPins(entries)
		if len(entries) == 1 {
			a.chatView.PinsPicker.SetStatus("1 pinned message")
		} else {
			a.chatView.PinsPicker.SetStatus(fmt.Sprintf("%d pinned messages", len(entries)))
		}
	})
}

// loadChannelBookmarks fetches bookmarks for a channel and updates the UI.
func (a *App) loadChannelBookmarks(channelID string) {
	bookmarks, err := a.slack.ListBookmarks(channelID)
	if err != nil {
		slog.Error("failed to fetch bookmarks", "channel", channelID, "error", err)
		a.tview.QueueUpdateDraw(func() {
			a.chatView.BookmarksPicker.SetStatus("Failed to load bookmarks (ensure bookmarks:read scope is granted)")
		})
		return
	}

	entries := make([]chat.BookmarkEntry, 0, len(bookmarks))
	for _, b := range bookmarks {
		entries = append(entries, chat.BookmarkEntry{
			ID:    b.ID,
			Title: b.Title,
			Link:  b.Link,
			Type:  b.Type,
		})
	}

	a.tview.QueueUpdateDraw(func() {
		a.chatView.BookmarksPicker.SetBookmarks(entries)
		if len(entries) == 1 {
			a.chatView.BookmarksPicker.SetStatus("1 bookmark  [Enter]open  [Esc]close")
		} else {
			a.chatView.BookmarksPicker.SetStatus(fmt.Sprintf("%d bookmarks  [Enter]open  [Esc]close", len(entries)))
		}
	})
}

// fetchChannelPins fetches pinned message timestamps for a channel and updates the messages list.
func (a *App) fetchChannelPins(channelID string) {
	items, err := a.slack.ListPins(channelID)
	if err != nil {
		slog.Error("failed to fetch pins", "channel", channelID, "error", err)
		return
	}

	pinnedSet := make(map[string]bool, len(items))
	timestamps := make([]string, 0, len(items))
	for _, item := range items {
		if item.Message != nil {
			pinnedSet[item.Message.Timestamp] = true
			timestamps = append(timestamps, item.Message.Timestamp)
		}
	}

	a.mu.Lock()
	a.pinnedMsgs[channelID] = pinnedSet
	a.mu.Unlock()

	a.tview.QueueUpdateDraw(func() {
		a.chatView.MessagesList.SetPinnedMessages(timestamps)
	})
}

// loadUserProfile fetches a user's profile and populates the user profile panel.
func (a *App) loadUserProfile(userID string) {
	user, err := a.slack.GetUserInfo(userID)
	if err != nil {
		slog.Error("failed to fetch user profile", "user", userID, "error", err)
		a.tview.QueueUpdateDraw(func() {
			a.chatView.UserProfilePanel.SetStatus("Failed to load profile")
		})
		return
	}

	data := chat.UserProfileData{
		UserID:      user.ID,
		DisplayName: user.Profile.DisplayName,
		RealName:    user.RealName,
		Title:       user.Profile.Title,
		StatusEmoji: user.Profile.StatusEmoji,
		StatusText:  user.Profile.StatusText,
		Timezone:    user.TZ,
		TzOffset:    user.TZOffset,
		Presence:    user.Presence,
		Email:       user.Profile.Email,
		Phone:       user.Profile.Phone,
		IsBot:       user.IsBot,
		IsAdmin:     user.IsAdmin,
		IsOwner:     user.IsOwner,
	}

	a.tview.QueueUpdateDraw(func() {
		a.chatView.UserProfilePanel.SetData(data)
		a.chatView.UserProfilePanel.SetStatus("[d]m  [i]d  [Esc]close")
	})
}

// loadStarredItems fetches all starred items and populates the starred picker.
func (a *App) loadStarredItems() {
	items, err := a.slack.ListAllStars()
	if err != nil {
		slog.Error("failed to fetch starred items", "error", err)
		a.tview.QueueUpdateDraw(func() {
			a.chatView.StarredPicker.SetStatus("Failed to load starred items")
		})
		return
	}

	a.mu.Lock()
	users := a.users
	// Build channel name map.
	channelNameMap := make(map[string]string, len(a.channels))
	for _, ch := range a.channels {
		channelNameMap[ch.ID] = ch.Name
	}
	a.mu.Unlock()

	entries := make([]chat.StarredEntry, 0, len(items))
	for _, item := range items {
		if item.Message == nil {
			continue
		}
		userName := ""
		if u, ok := users[item.Message.User]; ok {
			if u.Profile.DisplayName != "" {
				userName = u.Profile.DisplayName
			} else if u.RealName != "" {
				userName = u.RealName
			} else {
				userName = u.Name
			}
		}
		if userName == "" {
			userName = item.Message.User
		}
		entries = append(entries, chat.StarredEntry{
			ChannelID:   item.Channel,
			ChannelName: channelNameMap[item.Channel],
			Timestamp:   item.Message.Timestamp,
			UserName:    userName,
			Text:        item.Message.Text,
		})
	}

	a.tview.QueueUpdateDraw(func() {
		a.chatView.StarredPicker.SetStarred(entries)
		if len(entries) == 1 {
			a.chatView.StarredPicker.SetStatus("1 starred message")
		} else {
			a.chatView.StarredPicker.SetStatus(fmt.Sprintf("%d starred messages", len(entries)))
		}
	})
}

// searchMessages searches Slack messages and updates the search picker with results.
func (a *App) searchMessages(query string) {
	results, err := a.slack.SearchMessages(query, slack.SearchParameters{
		Count:         20,
		Sort:          "timestamp",
		SortDirection: "desc",
	})
	if err != nil {
		slog.Error("failed to search messages", "query", query, "error", err)
		a.tview.QueueUpdateDraw(func() {
			a.chatView.SearchPicker.SetStatus("Search failed")
		})
		return
	}

	a.mu.Lock()
	users := a.users
	a.mu.Unlock()

	entries := make([]chat.SearchResultEntry, 0, len(results.Matches))
	for _, m := range results.Matches {
		userName := m.Username
		if userName == "" {
			if u, ok := users[m.User]; ok {
				if u.Profile.DisplayName != "" {
					userName = u.Profile.DisplayName
				} else if u.RealName != "" {
					userName = u.RealName
				} else {
					userName = u.Name
				}
			}
		}

		entries = append(entries, chat.SearchResultEntry{
			ChannelID:   m.Channel.ID,
			ChannelName: m.Channel.Name,
			UserName:    userName,
			Timestamp:   m.Timestamp,
			Text:        m.Text,
		})
	}

	a.tview.QueueUpdateDraw(func() {
		a.chatView.SearchPicker.SetResults(entries)
		a.chatView.SearchPicker.SetStatus(fmt.Sprintf("%d results", results.Total))
	})
}

// updateChannelPresence counts online users from the given messages and updates the status bar.
func (a *App) updateChannelPresence(channelID string, messages []slack.Message, users map[string]slack.User) {
	if !a.Config.Presence.Enabled {
		a.chatView.StatusBar.SetChannelPresence(0, 0)
		return
	}

	seen := make(map[string]bool)
	var online, total int
	for _, msg := range messages {
		if msg.User == "" || seen[msg.User] {
			continue
		}
		seen[msg.User] = true
		total++
		if u, ok := users[msg.User]; ok && u.Presence == "active" {
			online++
		}
	}
	a.chatView.StatusBar.SetChannelPresence(online, total)
}

// onMessageSend handles sending a new message or thread reply.
func (a *App) onMessageSend(channelID, text, threadTS string) {
	go func() {
		opts := []slack.MsgOption{slack.MsgOptionText(text, false)}
		if threadTS != "" {
			opts = append(opts, slack.MsgOptionTS(threadTS))
		}
		_, _, err := a.slack.PostMessage(channelID, opts...)
		if err != nil {
			slog.Error("failed to send message", "channel", channelID, "error", err)
			a.showCommandFeedback("Send failed: " + err.Error())
		}
	}()
}

// sendMeMessage sends a /me action message to the channel.
func (a *App) sendMeMessage(channelID, text string) {
	opts := []slack.MsgOption{
		slack.MsgOptionText(text, false),
		slack.MsgOptionMeMessage(),
	}
	_, _, err := a.slack.PostMessage(channelID, opts...)
	if err != nil {
		slog.Error("failed to send /me message", "channel", channelID, "error", err)
		a.showCommandFeedback("Send failed: " + err.Error())
	}
}

// onThreadReplySend handles sending a reply in the thread view.
func (a *App) onThreadReplySend(channelID, text, threadTS string) {
	go func() {
		opts := []slack.MsgOption{
			slack.MsgOptionText(text, false),
			slack.MsgOptionTS(threadTS),
		}
		_, _, err := a.slack.PostMessage(channelID, opts...)
		if err != nil {
			slog.Error("failed to send thread reply", "channel", channelID, "thread", threadTS, "error", err)
			a.showCommandFeedback("Reply failed: " + err.Error())
		}
	}()
}

// loadThread fetches thread replies and updates the thread view.
func (a *App) loadThread(channelID, threadTS string) {
	msgs, _, _, err := a.slack.GetConversationReplies(&slack.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: threadTS,
	})
	if err != nil {
		slog.Error("failed to fetch thread replies", "channel", channelID, "thread", threadTS, "error", err)
		return
	}

	a.mu.Lock()
	users := a.users
	a.mu.Unlock()

	a.tview.QueueUpdateDraw(func() {
		a.chatView.ThreadView.SetMessages(channelID, threadTS, msgs, users)
	})
}

// onMessageEdit handles editing an existing message.
func (a *App) onMessageEdit(channelID, timestamp, text string) {
	go func() {
		_, _, _, err := a.slack.UpdateMessage(channelID, timestamp,
			slack.MsgOptionText(text, false))
		if err != nil {
			slog.Error("failed to edit message", "channel", channelID, "error", err)
			a.showCommandFeedback("Edit failed: " + err.Error())
		}
	}()
}

// maybeNotify sends a desktop notification if the message warrants one.
func (a *App) maybeNotify(evt *slackevents.MessageEvent) {
	if !a.Config.Notifications.Enabled {
		return
	}
	// Never notify for own messages.
	if evt.User == a.slack.UserID {
		return
	}

	a.mu.Lock()
	isDM := a.dmSet[evt.Channel]
	users := a.users
	a.mu.Unlock()

	mention := notifications.DetectMention(evt.Text, a.slack.UserID, isDM)
	if mention == notifications.MentionNone {
		return
	}

	// Resolve sender name.
	sender := evt.User
	if u, ok := users[evt.User]; ok {
		if u.Profile.DisplayName != "" {
			sender = u.Profile.DisplayName
		} else if u.RealName != "" {
			sender = u.RealName
		} else if u.Name != "" {
			sender = u.Name
		}
	}

	body := notifications.StripMrkdwn(evt.Text)

	var title string
	switch mention {
	case notifications.MentionDM:
		title = fmt.Sprintf("DM from %s", sender)
	case notifications.MentionDirect:
		title = fmt.Sprintf("%s mentioned you", sender)
	default:
		title = fmt.Sprintf("%s in channel", sender)
	}

	a.notifier.Send(title, body)
}

// uploadFile uploads a local file to the given channel, optionally in a thread.
func (a *App) uploadFile(channelID, threadTS, filePath string) {
	name := filepath.Base(filePath)

	a.tview.QueueUpdateDraw(func() {
		a.chatView.StatusBar.SetConnectionStatus(
			fmt.Sprintf("%s (%s) — uploading %s...", a.slack.UserName, a.slack.TeamName, name))
	})

	params := slack.UploadFileParameters{
		File:            filePath,
		Filename:        name,
		Channel:         channelID,
		ThreadTimestamp: threadTS,
	}

	_, err := a.slack.UploadFile(params)

	a.tview.QueueUpdateDraw(func() {
		if err != nil {
			slog.Error("failed to upload file", "file", filePath, "error", err)
			a.chatView.StatusBar.SetConnectionStatus(
				fmt.Sprintf("%s (%s) — upload failed: %s", a.slack.UserName, a.slack.TeamName, err.Error()))
		} else {
			a.chatView.StatusBar.SetConnectionStatus(
				fmt.Sprintf("%s (%s) — uploaded %s", a.slack.UserName, a.slack.TeamName, name))
		}
	})
}

// openFile downloads a Slack file and opens it with the system default application.
func (a *App) openFile(file slack.File) {
	url := file.URLPrivateDownload
	if url == "" {
		url = file.URLPrivate
	}
	if url == "" {
		slog.Error("no download URL for file", "file_id", file.ID)
		return
	}

	a.tview.QueueUpdateDraw(func() {
		a.chatView.StatusBar.SetConnectionStatus(
			fmt.Sprintf("%s (%s) — downloading %s...", a.slack.UserName, a.slack.TeamName, file.Name))
	})

	// Ensure download directory exists.
	downloadDir := a.Config.DownloadDir
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		slog.Error("failed to create download dir", "dir", downloadDir, "error", err)
		return
	}

	destPath := filepath.Join(downloadDir, file.Name)

	// Download with auth token.
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("failed to create download request", "error", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+a.slack.Token())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("failed to download file", "error", err)
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(destPath)
	if err != nil {
		slog.Error("failed to create file", "path", destPath, "error", err)
		return
	}
	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		slog.Error("failed to write file", "path", destPath, "error", err)
		return
	}

	a.tview.QueueUpdateDraw(func() {
		a.chatView.StatusBar.SetConnectionStatus(
			fmt.Sprintf("%s (%s) — opening %s", a.slack.UserName, a.slack.TeamName, file.Name))
	})

	// Open with system default.
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", destPath)
	default:
		cmd = exec.Command("xdg-open", destPath)
	}
	if err := cmd.Start(); err != nil {
		slog.Error("failed to open file", "path", destPath, "error", err)
	}
}

// markChannelRead updates the last-read timestamp, clears the unread badge,
// and calls MarkConversation on the Slack API.
func (a *App) markChannelRead(channelID, ts string) {
	a.mu.Lock()
	a.lastRead[channelID] = ts
	a.mu.Unlock()

	a.tview.QueueUpdateDraw(func() {
		a.chatView.ChannelsTree.SetUnreadCount(channelID, 0)
	})

	go func() {
		if err := a.slack.MarkConversation(channelID, ts); err != nil {
			slog.Error("failed to mark conversation", "channel", channelID, "error", err)
		}
	}()
}

// markAllChannelsRead marks all channels as read using their latest known message.
func (a *App) markAllChannelsRead() {
	a.mu.Lock()
	channels := make([]slack.Channel, len(a.channels))
	copy(channels, a.channels)
	a.mu.Unlock()

	for _, ch := range channels {
		a.mu.Lock()
		lr := a.lastRead[ch.ID]
		a.mu.Unlock()
		if lr != "" {
			// Only mark if there are unreads.
			if a.chatView.ChannelsTree.UnreadCount(ch.ID) > 0 {
				a.markChannelRead(ch.ID, lr)
			}
		}
	}
}

// loadChannelInfo fetches channel details and populates the info panel.
func (a *App) loadChannelInfo(channelID string) {
	ch, err := a.slack.GetConversationInfo(channelID)
	if err != nil {
		slog.Error("failed to fetch channel info", "channel", channelID, "error", err)
		a.tview.QueueUpdateDraw(func() {
			a.chatView.ChannelInfoPanel.SetStatus("Failed to load channel info")
		})
		return
	}

	// Resolve creator name.
	a.mu.Lock()
	creatorName := ch.Creator
	if u, ok := a.users[ch.Creator]; ok {
		if u.Profile.DisplayName != "" {
			creatorName = u.Profile.DisplayName
		} else if u.RealName != "" {
			creatorName = u.RealName
		} else if u.Name != "" {
			creatorName = u.Name
		}
	}
	a.mu.Unlock()

	// Count pins from cached data.
	a.mu.Lock()
	pinCount := len(a.pinnedMsgs[channelID])
	a.mu.Unlock()

	data := chat.ChannelInfoData{
		ChannelID:   channelID,
		Name:        ch.Name,
		Description: ch.Purpose.Value,
		Topic:       ch.Topic.Value,
		Purpose:     ch.Purpose.Value,
		Creator:     creatorName,
		Created:     ch.Created.Time(),
		NumMembers:  ch.NumMembers,
		NumPins:     pinCount,
		IsArchived:  ch.IsArchived,
		IsPrivate:   ch.IsPrivate,
		IsDM:        ch.IsIM,
	}

	a.tview.QueueUpdateDraw(func() {
		a.chatView.ChannelInfoPanel.SetData(data)
		a.chatView.ChannelInfoPanel.SetStatus(" [t]opic  [p]urpose  [l]eave  [Esc]close")
	})
}

// editChannelField opens the external editor to set a channel's topic or purpose.
func (a *App) editChannelField(channelID, field string) {
	// Create a temp file with the current value.
	a.mu.Lock()
	var current string
	for _, ch := range a.channels {
		if ch.ID == channelID {
			switch field {
			case "topic":
				current = ch.Topic.Value
			case "purpose":
				current = ch.Purpose.Value
			}
			break
		}
	}
	a.mu.Unlock()

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("slacko-%s-*.txt", field))
	if err != nil {
		slog.Error("failed to create temp file", "error", err)
		return
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(current); err != nil {
		tmpFile.Close()
		slog.Error("failed to write temp file", "error", err)
		return
	}
	tmpFile.Close()

	// Suspend TUI and open editor.
	a.tview.Suspend(func() {
		cmd := exec.Command(a.Config.Editor, tmpPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			slog.Error("editor failed", "error", err)
			return
		}
	})

	// Read the edited value.
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		slog.Error("failed to read temp file", "error", err)
		return
	}
	newValue := strings.TrimSpace(string(data))

	if newValue == current {
		return
	}

	// Update via API.
	switch field {
	case "topic":
		if _, err := a.slack.SetTopic(channelID, newValue); err != nil {
			slog.Error("failed to set topic", "channel", channelID, "error", err)
			a.tview.QueueUpdateDraw(func() {
				a.chatView.ChannelInfoPanel.SetStatus("Failed to set topic")
			})
			return
		}
	case "purpose":
		if _, err := a.slack.SetPurpose(channelID, newValue); err != nil {
			slog.Error("failed to set purpose", "channel", channelID, "error", err)
			a.tview.QueueUpdateDraw(func() {
				a.chatView.ChannelInfoPanel.SetStatus("Failed to set purpose")
			})
			return
		}
	}

	// Refresh the info panel.
	a.tview.QueueUpdateDraw(func() {
		a.chatView.SetChannelHeader(a.chatView.ChannelInfoPanel.Data().Name, newValue)
	})
	go a.loadChannelInfo(channelID)
}

// leaveChannel leaves a channel and updates the UI.
func (a *App) leaveChannel(channelID string) {
	_, err := a.slack.LeaveConversation(channelID)
	if err != nil {
		slog.Error("failed to leave channel", "channel", channelID, "error", err)
		a.tview.QueueUpdateDraw(func() {
			a.chatView.ChannelInfoPanel.SetStatus("Failed to leave channel")
		})
		return
	}

	a.tview.QueueUpdateDraw(func() {
		a.chatView.HideChannelInfo()
		a.chatView.ChannelsTree.RemoveChannel(channelID)
	})

	// Remove from cached channels.
	a.mu.Lock()
	for i, ch := range a.channels {
		if ch.ID == channelID {
			a.channels = append(a.channels[:i], a.channels[i+1:]...)
			break
		}
	}
	a.mu.Unlock()
}

// executeSlashCommand handles slash commands from the message input.
func (a *App) executeSlashCommand(channelID, command, args string) {
	switch command {
	case "help":
		a.showCommandFeedback(a.formatHelpText())
	case "status":
		go a.cmdSetStatus(args)
	case "clear-status":
		go a.cmdClearStatus()
	case "topic":
		go a.cmdSetTopic(channelID, args)
	case "leave":
		go a.leaveChannel(channelID)
	case "search":
		if args != "" {
			a.chatView.ShowSearchPicker()
			go a.searchMessages(args)
		} else {
			a.chatView.ShowSearchPicker()
		}
	case "who":
		a.chatView.ShowMembersPicker()
		a.chatView.MembersPicker.SetStatus("Loading...")
		go a.loadChannelMembers(channelID)
	case "me":
		if args == "" {
			a.showCommandFeedback("Usage: /me [action]")
			return
		}
		go a.sendMeMessage(channelID, args)
	case "mute":
		a.chatView.ChannelsTree.SetMuted(channelID, true)
		a.showCommandFeedback("Channel muted")
	case "unmute":
		a.chatView.ChannelsTree.SetMuted(channelID, false)
		a.showCommandFeedback("Channel unmuted")
	case "schedule":
		go a.cmdScheduleMessage(channelID, args)
	case "scheduled":
		go a.cmdListScheduledMessages(channelID)
	case "remind":
		go a.cmdRemind(args)
	case "reminders":
		go a.cmdListReminders()
	case "create-channel":
		a.chatView.ShowChannelCreateForm()
	case "invite":
		a.openInvitePicker(channelID)
	case "logout":
		a.logout()
	default:
		a.showCommandFeedback("Unknown command: /" + command)
	}
}

// formatHelpText builds the /help output listing available slash commands.
func (a *App) formatHelpText() string {
	cmds := chat.BuiltinCommands()
	var b strings.Builder
	b.WriteString("Available commands:")
	for _, cmd := range cmds {
		b.WriteString(fmt.Sprintf("  %s — %s", cmd.Usage, cmd.Description))
	}
	return b.String()
}

// showCommandFeedback shows a temporary status bar message.
// Safe to call from any goroutine, including the tview event loop.
func (a *App) showCommandFeedback(msg string) {
	go func() {
		a.tview.QueueUpdateDraw(func() {
			a.chatView.StatusBar.SetTypingIndicator(msg)
		})
		<-time.After(4 * time.Second)
		a.tview.QueueUpdateDraw(func() {
			a.chatView.StatusBar.SetTypingIndicator("")
		})
	}()
}

// cmdSetStatus sets the user's Slack status.
func (a *App) cmdSetStatus(args string) {
	var emoji, text string
	args = strings.TrimSpace(args)
	if args == "" {
		a.showCommandFeedback("Usage: /status [:emoji:] [text]")
		return
	}
	// Parse optional :emoji: prefix.
	if strings.HasPrefix(args, ":") {
		end := strings.Index(args[1:], ":")
		if end >= 0 {
			emoji = args[:end+2]
			text = strings.TrimSpace(args[end+2:])
		} else {
			text = args
		}
	} else {
		text = args
	}
	if err := a.slack.SetUserCustomStatus(text, emoji); err != nil {
		slog.Error("failed to set status", "error", err)
		a.showCommandFeedback("Failed to set status")
		return
	}
	a.showCommandFeedback("Status updated")
}

// cmdClearStatus clears the user's Slack status.
func (a *App) cmdClearStatus() {
	if err := a.slack.SetUserCustomStatus("", ""); err != nil {
		slog.Error("failed to clear status", "error", err)
		a.showCommandFeedback("Failed to clear status")
		return
	}
	a.showCommandFeedback("Status cleared")
}

// cmdSetTopic sets the channel topic.
func (a *App) cmdSetTopic(channelID, topic string) {
	if topic == "" {
		a.showCommandFeedback("Usage: /topic [text]")
		return
	}
	if _, err := a.slack.SetTopic(channelID, topic); err != nil {
		slog.Error("failed to set topic", "channel", channelID, "error", err)
		a.showCommandFeedback("Failed to set topic")
		return
	}
	a.tview.QueueUpdateDraw(func() {
		// Find channel name.
		a.mu.Lock()
		var name string
		for _, ch := range a.channels {
			if ch.ID == channelID {
				name = ch.Name
				break
			}
		}
		a.mu.Unlock()
		a.chatView.SetChannelHeader(name, topic)
	})
	a.showCommandFeedback("Topic updated")
}

// loadChannelMembers fetches all members of a channel and populates the members picker.
func (a *App) loadChannelMembers(channelID string) {
	var allUserIDs []string
	cursor := ""
	for {
		userIDs, nextCursor, err := a.slack.GetUsersInConversation(channelID, cursor, 200)
		if err != nil {
			slog.Error("failed to get channel members", "channel", channelID, "error", err)
			a.tview.QueueUpdateDraw(func() {
				a.chatView.MembersPicker.SetStatus("Failed to load members")
			})
			return
		}
		allUserIDs = append(allUserIDs, userIDs...)
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	a.mu.Lock()
	users := a.users
	a.mu.Unlock()

	entries := make([]chat.MemberEntry, 0, len(allUserIDs))
	for _, uid := range allUserIDs {
		entry := chat.MemberEntry{UserID: uid}
		if u, ok := users[uid]; ok {
			entry.DisplayName = u.Profile.DisplayName
			entry.RealName = u.RealName
			entry.IsBot = u.IsBot
			if entry.DisplayName == "" && u.Name != "" {
				entry.DisplayName = u.Name
			}
		}
		entries = append(entries, entry)
	}

	a.tview.QueueUpdateDraw(func() {
		a.chatView.MembersPicker.SetMembers(entries)
	})
}

// openInvitePicker opens the invite picker populated with workspace users
// who are not already in the given channel.
func (a *App) openInvitePicker(channelID string) {
	a.chatView.ShowInvitePicker()
	a.chatView.InvitePicker.SetStatus("Loading users...")

	go func() {
		// Fetch current channel members.
		memberSet := make(map[string]bool)
		cursor := ""
		for {
			userIDs, nextCursor, err := a.slack.GetUsersInConversation(channelID, cursor, 200)
			if err != nil {
				slog.Error("failed to get channel members for invite", "channel", channelID, "error", err)
				a.tview.QueueUpdateDraw(func() {
					a.chatView.InvitePicker.SetStatus("Failed to load members")
				})
				return
			}
			for _, uid := range userIDs {
				memberSet[uid] = true
			}
			if nextCursor == "" {
				break
			}
			cursor = nextCursor
		}

		a.mu.Lock()
		users := a.users
		a.mu.Unlock()

		// Build entries from workspace users, excluding current members and bots.
		entries := make([]chat.InviteUserEntry, 0)
		for _, u := range users {
			if memberSet[u.ID] || u.IsBot || u.Deleted {
				continue
			}
			entry := chat.InviteUserEntry{
				UserID:   u.ID,
				RealName: u.RealName,
			}
			if u.Profile.DisplayName != "" {
				entry.DisplayName = u.Profile.DisplayName
			} else if u.Name != "" {
				entry.DisplayName = u.Name
			}
			entries = append(entries, entry)
		}

		a.tview.QueueUpdateDraw(func() {
			a.chatView.InvitePicker.SetUsers(entries)
		})
	}()
}

// inviteUserToChannel invites a user to a channel via the Slack API.
func (a *App) inviteUserToChannel(channelID, userID string) {
	_, err := a.slack.InviteUsersToConversation(channelID, userID)
	if err != nil {
		slog.Error("failed to invite user", "channel", channelID, "user", userID, "error", err)
		a.showCommandFeedback("Invite failed: " + err.Error())
		return
	}

	// Resolve user name for feedback.
	a.mu.Lock()
	userName := userID
	if u, ok := a.users[userID]; ok {
		if u.Profile.DisplayName != "" {
			userName = u.Profile.DisplayName
		} else if u.RealName != "" {
			userName = u.RealName
		} else if u.Name != "" {
			userName = u.Name
		}
	}
	a.mu.Unlock()

	a.showCommandFeedback("Invited " + userName + " to channel")
}

// cmdScheduleMessage schedules a message for future delivery.
func (a *App) cmdScheduleMessage(channelID, args string) {
	if args == "" {
		a.showCommandFeedback("Usage: /schedule [time] [message]")
		return
	}

	postAt, message, err := chat.ParseFutureTime(args)
	if err != nil {
		a.showCommandFeedback("Invalid time: " + err.Error())
		return
	}
	if message == "" {
		a.showCommandFeedback("Please provide a message after the time")
		return
	}
	if postAt.Before(time.Now()) {
		a.showCommandFeedback("Scheduled time must be in the future")
		return
	}

	postAtStr := fmt.Sprintf("%d", postAt.Unix())
	_, _, err = a.slack.ScheduleMessage(channelID, postAtStr, message)
	if err != nil {
		slog.Error("failed to schedule message", "error", err)
		a.showCommandFeedback("Failed to schedule message: " + err.Error())
		return
	}

	a.showCommandFeedback(fmt.Sprintf("Message scheduled for %s", postAt.Format("Jan 2 15:04")))
}

// cmdListScheduledMessages lists scheduled messages for the current channel.
func (a *App) cmdListScheduledMessages(channelID string) {
	msgs, err := a.slack.GetScheduledMessages(channelID)
	if err != nil {
		slog.Error("failed to list scheduled messages", "error", err)
		a.showCommandFeedback("Failed to list scheduled messages")
		return
	}

	if len(msgs) == 0 {
		a.showCommandFeedback("No scheduled messages")
		return
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%d scheduled: ", len(msgs)))
	for i, m := range msgs {
		t := time.Unix(int64(m.PostAt), 0)
		if i > 0 {
			b.WriteString(" | ")
		}
		text := m.Text
		if len(text) > 30 {
			text = text[:27] + "..."
		}
		b.WriteString(fmt.Sprintf("%s: %s", t.Format("Jan 2 15:04"), text))
		if i >= 4 {
			b.WriteString(fmt.Sprintf(" (+%d more)", len(msgs)-5))
			break
		}
	}
	a.showCommandFeedback(b.String())
}

// cmdRemind creates a reminder for the current user.
func (a *App) cmdRemind(args string) {
	if args == "" {
		a.showCommandFeedback("Usage: /remind [what] [when]  (e.g. /remind standup in 30m)")
		return
	}

	// Slack's reminder API accepts natural language for the time.
	// We pass the whole args string and let Slack parse it.
	rem, err := a.slack.AddReminder(args, "")
	if err != nil {
		slog.Error("failed to create reminder", "error", err)
		a.showCommandFeedback("Failed to create reminder: " + err.Error())
		return
	}

	t := time.Unix(int64(rem.Time), 0)
	a.showCommandFeedback(fmt.Sprintf("Reminder set for %s: %s", t.Format("Jan 2 15:04"), rem.Text))
}

// cmdListReminders lists active reminders.
func (a *App) cmdListReminders() {
	rems, err := a.slack.ListReminders()
	if err != nil {
		slog.Error("failed to list reminders", "error", err)
		a.showCommandFeedback("Failed to list reminders")
		return
	}

	if len(rems) == 0 {
		a.showCommandFeedback("No active reminders")
		return
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%d reminders: ", len(rems)))
	for i, r := range rems {
		if i > 0 {
			b.WriteString(" | ")
		}
		t := time.Unix(int64(r.Time), 0)
		text := r.Text
		if len(text) > 30 {
			text = text[:27] + "..."
		}
		b.WriteString(fmt.Sprintf("%s: %s", t.Format("Jan 2 15:04"), text))
		if i >= 4 {
			b.WriteString(fmt.Sprintf(" (+%d more)", len(rems)-5))
			break
		}
	}
	a.showCommandFeedback(b.String())
}

// executeVimCommand handles vim-style :commands from the command bar.
func (a *App) executeVimCommand(command, args string) {
	switch command {
	case "q":
		a.shutdown()
	case "theme":
		if args == "" {
			a.showCommandFeedback("Usage: :theme [name] (default, dark, light, monokai, solarized_dark, solarized_light, high_contrast, monochrome)")
		} else {
			a.showCommandFeedback("Theme switching requires restart. Set theme.preset = \"" + args + "\" in config.")
		}
	case "join":
		if args == "" {
			a.showCommandFeedback("Usage: :join #channel")
			return
		}
		channelName := strings.TrimPrefix(args, "#")
		go a.cmdJoinChannel(channelName)
	case "leave":
		a.mu.Lock()
		ch := a.currentChannel
		a.mu.Unlock()
		if ch != "" {
			go a.leaveChannel(ch)
		}
	case "search":
		if args != "" {
			a.chatView.ShowSearchPicker()
			go a.searchMessages(args)
		} else {
			a.chatView.ShowSearchPicker()
		}
	case "mark-read":
		a.mu.Lock()
		ch := a.currentChannel
		a.mu.Unlock()
		if ch != "" {
			ts := a.chatView.MessagesList.LatestTimestamp()
			if ts != "" {
				a.markChannelRead(ch, ts)
				a.showCommandFeedback("Channel marked as read")
			}
		}
	case "mark-all-read":
		go func() {
			a.markAllChannelsRead()
			a.showCommandFeedback("All channels marked as read")
		}()
	case "open":
		if args == "" {
			a.showCommandFeedback("Usage: :open [url]")
			return
		}
		go a.openURL(args)
	case "reconnect":
		a.showCommandFeedback("Reconnecting...")
		if a.cancel != nil {
			a.cancel()
		}
		go a.showMain()
	case "debug":
		go a.toggleDebugLogging()
	case "set":
		if args == "" {
			a.showCommandFeedback(ListRuntimeOptions(a.Config))
			return
		}

		option, value, query, err := ParseSetCommand(args)
		if err != nil {
			a.showCommandFeedback("Error: " + err.Error())
			return
		}

		if query {
			msg, err := QueryOption(a.Config, option)
			if err != nil {
				a.showCommandFeedback("Error: " + err.Error())
			} else {
				a.showCommandFeedback(msg)
			}
			return
		}

		msg, err := ApplySetCommand(a.Config, option, value)
		if err != nil {
			a.showCommandFeedback("Error: " + err.Error())
			return
		}
		a.showCommandFeedback(msg)

		// Apply side effects for options that need immediate action.
		switch option {
		case "mouse":
			a.tview.EnableMouse(a.Config.Mouse)
		}
	case "bookmarks":
		a.mu.Lock()
		ch := a.currentChannel
		a.mu.Unlock()
		if ch == "" {
			a.showCommandFeedback("No channel selected")
			return
		}
		a.chatView.ShowBookmarksPicker()
		a.chatView.BookmarksPicker.SetStatus("Loading...")
		go a.loadChannelBookmarks(ch)
	case "workspace":
		a.populateWorkspacePicker()
		a.tview.QueueUpdateDraw(func() {
			a.chatView.ShowWorkspacePicker()
		})
	case "members":
		a.mu.Lock()
		ch := a.currentChannel
		a.mu.Unlock()
		if ch != "" {
			a.tview.QueueUpdateDraw(func() {
				a.chatView.ShowMembersPicker()
				a.chatView.MembersPicker.SetStatus("Loading...")
			})
			go a.loadChannelMembers(ch)
		}
	case "create-channel":
		a.chatView.ShowChannelCreateForm()
	case "invite":
		a.mu.Lock()
		ch := a.currentChannel
		a.mu.Unlock()
		if ch != "" {
			a.openInvitePicker(ch)
		}
	case "group-dm":
		a.openGroupDMPicker()
	case "logout":
		a.logout()
	default:
		a.showCommandFeedback("Unknown command: :" + command)
	}
}

// cmdJoinChannel joins a channel by name.
func (a *App) cmdJoinChannel(channelName string) {
	a.mu.Lock()
	var channelID string
	for _, ch := range a.channels {
		if ch.Name == channelName {
			channelID = ch.ID
			break
		}
	}
	a.mu.Unlock()

	if channelID != "" {
		// Already in the channel list, just switch to it.
		a.tview.QueueUpdateDraw(func() {
			a.onChannelSelected(channelID)
		})
		return
	}

	// Try to join via API.
	ch, _, _, err := a.slack.API().JoinConversation(channelName)
	if err != nil {
		slog.Error("failed to join channel", "channel", channelName, "error", err)
		a.showCommandFeedback("Failed to join #" + channelName + ": " + err.Error())
		return
	}

	a.mu.Lock()
	a.channels = append(a.channels, *ch)
	users := a.users
	a.mu.Unlock()

	a.tview.QueueUpdateDraw(func() {
		a.chatView.ChannelsTree.AddChannel(*ch, users, a.slack.UserID)
		a.onChannelSelected(ch.ID)
	})
	a.showCommandFeedback("Joined #" + channelName)
}

// cmdCreateChannel creates a new Slack channel and switches to it.
func (a *App) cmdCreateChannel(name string, isPrivate bool) {
	ch, err := a.slack.CreateConversation(name, isPrivate)
	if err != nil {
		slog.Error("failed to create channel", "name", name, "error", err)
		a.tview.QueueUpdateDraw(func() {
			a.chatView.ChannelCreateForm.SetStatus("Failed: " + err.Error())
		})
		return
	}

	a.mu.Lock()
	a.channels = append(a.channels, *ch)
	users := a.users
	a.mu.Unlock()

	a.tview.QueueUpdateDraw(func() {
		a.chatView.HideChannelCreateForm()
		a.chatView.ChannelsTree.AddChannel(*ch, users, a.slack.UserID)
		a.onChannelSelected(ch.ID)
	})

	channelType := "public"
	if isPrivate {
		channelType = "private"
	}
	a.showCommandFeedback(fmt.Sprintf("Created %s channel #%s", channelType, name))
}

// openURL opens a URL in the system's default browser.
func (a *App) openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		slog.Error("failed to open URL", "url", url, "error", err)
		a.showCommandFeedback("Failed to open URL")
		return
	}
	a.showCommandFeedback("Opened: " + url)
}

// toggleDebugLogging toggles the log level between debug and info.
func (a *App) toggleDebugLogging() {
	// Toggle is a best-effort feature; just show feedback.
	a.showCommandFeedback("Debug logging toggled (check log output)")
	slog.Debug("debug logging enabled via :debug command")
}

// openGroupDMPicker opens the group DM picker populated with workspace users.
func (a *App) openGroupDMPicker() {
	a.mu.Lock()
	users := a.users
	selfID := a.slack.UserID
	a.mu.Unlock()

	// Build entries from workspace users, excluding self, bots, and deleted users.
	entries := make([]chat.GroupDMUserEntry, 0)
	for _, u := range users {
		if u.ID == selfID || u.IsBot || u.Deleted {
			continue
		}
		entry := chat.GroupDMUserEntry{
			UserID: u.ID,
		}
		if u.Profile.DisplayName != "" {
			entry.DisplayName = u.Profile.DisplayName
		} else if u.RealName != "" {
			entry.DisplayName = u.RealName
		} else if u.Name != "" {
			entry.DisplayName = u.Name
		} else {
			entry.DisplayName = u.ID
		}
		entries = append(entries, entry)
	}

	a.tview.QueueUpdateDraw(func() {
		a.chatView.GroupDMPicker.SetUsers(entries)
		a.chatView.ShowGroupDMPicker()
	})
}

// createGroupDM creates a group DM conversation with the given user IDs.
func (a *App) createGroupDM(userIDs []string) {
	ch, err := a.slack.OpenConversation(userIDs)
	if err != nil {
		slog.Error("failed to create group DM", "users", userIDs, "error", err)
		a.showCommandFeedback("Failed to create group DM: " + err.Error())
		return
	}

	// Add the new channel to the cached channels list.
	a.mu.Lock()
	a.channels = append(a.channels, *ch)
	users := a.users
	a.mu.Unlock()

	a.tview.QueueUpdateDraw(func() {
		a.chatView.ChannelsTree.AddChannel(*ch, users, a.slack.UserID)
		a.onChannelSelected(ch.ID)
	})

	// Build a user names list for feedback.
	var names []string
	a.mu.Lock()
	for _, uid := range userIDs {
		name := uid
		if u, ok := a.users[uid]; ok {
			if u.Profile.DisplayName != "" {
				name = u.Profile.DisplayName
			} else if u.RealName != "" {
				name = u.RealName
			} else if u.Name != "" {
				name = u.Name
			}
		}
		names = append(names, name)
	}
	a.mu.Unlock()

	a.showCommandFeedback("Group DM created with " + strings.Join(names, ", "))
}

// populateWorkspacePicker refreshes the workspace picker with stored workspaces.
func (a *App) populateWorkspacePicker() {
	ws, err := keyring.ListWorkspaces()
	if err != nil {
		slog.Warn("failed to list workspaces", "error", err)
		return
	}

	entries := make([]chat.WorkspaceEntry, len(ws))
	for i, w := range ws {
		entries[i] = chat.WorkspaceEntry{
			ID:   w.ID,
			Name: w.Name,
		}
	}

	a.tview.QueueUpdateDraw(func() {
		a.chatView.WorkspacePicker.SetCurrentWorkspace(a.slack.TeamID)
		a.chatView.WorkspacePicker.SetWorkspaces(entries)
	})
}

// switchWorkspace disconnects from the current workspace and connects to a new one.
func (a *App) switchWorkspace(workspaceID string) {
	ws, err := keyring.ListWorkspaces()
	if err != nil {
		slog.Error("failed to list workspaces", "error", err)
		a.showCommandFeedback("Failed to list workspaces")
		return
	}

	var target keyring.Workspace
	found := false
	for _, w := range ws {
		if w.ID == workspaceID {
			target = w
			found = true
			break
		}
	}
	if !found {
		a.showCommandFeedback("Workspace not found: " + workspaceID)
		return
	}

	tokens, err := keyring.GetWorkspaceTokens(target)
	if err != nil {
		slog.Error("failed to get workspace tokens", "workspace", target.Name, "error", err)
		a.showCommandFeedback("Failed to get tokens for " + target.Name)
		return
	}

	// Disconnect current.
	if a.cancel != nil {
		a.cancel()
	}

	// Create new client.
	client, err := slackclient.New(tokens.UserToken, tokens.AppToken)
	if err != nil {
		slog.Error("failed to create client for workspace", "workspace", target.Name, "error", err)
		a.showCommandFeedback("Failed to connect to " + target.Name)
		return
	}

	a.slack = client
	a.mu.Lock()
	a.currentChannel = ""
	a.channels = nil
	a.users = make(map[string]slack.User)
	a.dmSet = make(map[string]bool)
	a.lastRead = make(map[string]string)
	a.pinnedMsgs = make(map[string]map[string]bool)
	a.mu.Unlock()

	a.tview.QueueUpdateDraw(func() {
		a.showMain()
	})
}

// fetchAllChannels retrieves all conversations with pagination.
func (a *App) fetchAllChannels() ([]slack.Channel, error) {
	var all []slack.Channel
	params := &slack.GetConversationsParameters{
		Types:           []string{"public_channel", "private_channel", "mpim", "im"},
		Limit:           200,
		ExcludeArchived: true,
	}

	for {
		channels, cursor, err := a.slack.GetConversations(params)
		if err != nil {
			return nil, err
		}
		all = append(all, channels...)
		if cursor == "" {
			break
		}
		params.Cursor = cursor
	}

	return all, nil
}
