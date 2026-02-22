package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	gokeyring "github.com/zalando/go-keyring"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/keyring"
	slackclient "github.com/m96-chan/Slacko/internal/slack"
	"github.com/m96-chan/Slacko/internal/ui/chat"
	"github.com/m96-chan/Slacko/internal/ui/keys"
	"github.com/m96-chan/Slacko/internal/ui/login"
	"github.com/rivo/tview"
)

// App is the top-level application struct.
type App struct {
	Config   *config.Config
	tview    *tview.Application
	slack    *slackclient.Client
	chatView *chat.View
	cancel   context.CancelFunc
	channels []slack.Channel
	users    map[string]slack.User
	mu       sync.Mutex
}

// New creates a new App with the given config.
func New(cfg *config.Config) *App {
	return &App{
		Config: cfg,
		tview:  tview.NewApplication(),
		users:  make(map[string]slack.User),
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

	bot, botErr := keyring.GetBotToken()
	app, appErr := keyring.GetAppToken()

	if botErr == nil && appErr == nil {
		client, err := slackclient.New(bot, app)
		if err != nil {
			slog.Warn("stored tokens invalid, showing login", "error", err)
			a.showLogin()
		} else {
			a.slack = client
			a.showMain()
		}
	} else {
		if botErr != nil && !errors.Is(botErr, gokeyring.ErrNotFound) {
			slog.Warn("error reading bot token", "error", botErr)
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

	a.chatView.StatusBar.SetConnectionStatus(
		fmt.Sprintf("%s (%s) — connecting...", a.slack.UserName, a.slack.TeamName))
	a.tview.SetRoot(a.chatView, true)
	a.chatView.FocusPanel(chat.PanelMessages)

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
			})
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
					evt.Item.Channel, evt.Item.Timestamp, evt.Reaction)
			})
		},
		OnReactionRemoved: func(evt *slackevents.ReactionRemovedEvent) {
			a.tview.QueueUpdateDraw(func() {
				a.chatView.MessagesList.RemoveReaction(
					evt.Item.Channel, evt.Item.Timestamp, evt.Reaction)
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

	a.mu.Lock()
	a.channels = channels
	a.users = userMap
	a.mu.Unlock()

	slog.Info("initial data loaded", "channels", len(channels), "users", len(users))

	a.tview.QueueUpdateDraw(func() {
		a.chatView.ChannelsTree.Populate(channels, userMap, a.slack.UserID)
		a.chatView.ChannelsPicker.SetData(channels, userMap, a.slack.UserID)
		a.chatView.MentionsList.SetUsers(userMap)
		a.chatView.MentionsList.SetChannels(channels, userMap, a.slack.UserID)
		a.chatView.MessagesList.SetSelfUserID(a.slack.UserID)
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
	for _, ch := range a.channels {
		if ch.ID == channelID {
			a.chatView.SetChannelHeader(ch.Name, ch.Topic.Value)
			break
		}
	}
	a.mu.Unlock()

	a.chatView.MessageInput.SetChannel(channelID)
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
	})
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
		}
	}()
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
		}
	}()
}

// fetchAllChannels retrieves all conversations with pagination.
func (a *App) fetchAllChannels() ([]slack.Channel, error) {
	var all []slack.Channel
	params := &slack.GetConversationsParameters{
		Types:  []string{"public_channel", "private_channel", "mpim", "im"},
		Limit:  200,
		Cursor: "",
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
