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
	"sync"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	gokeyring "github.com/zalando/go-keyring"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/keyring"
	"github.com/m96-chan/Slacko/internal/notifications"
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
	notifier *notifications.Notifier
	cancel   context.CancelFunc
	channels       []slack.Channel
	users          map[string]slack.User
	dmSet          map[string]bool   // set of DM channel IDs
	lastRead       map[string]string // channelID → last-read timestamp
	currentChannel string
	mu             sync.Mutex
}

// New creates a new App with the given config.
func New(cfg *config.Config) *App {
	return &App{
		Config:   cfg,
		tview:    tview.NewApplication(),
		users:    make(map[string]slack.User),
		dmSet:    make(map[string]bool),
		lastRead: make(map[string]string),
		notifier: notifications.New(),
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

	channelNames := make(map[string]string, len(channels))
	for _, ch := range channels {
		channelNames[ch.ID] = ch.Name
	}

	a.tview.QueueUpdateDraw(func() {
		a.chatView.ChannelsTree.Populate(channels, userMap, a.slack.UserID)
		a.chatView.ChannelsPicker.SetData(channels, userMap, a.slack.UserID)
		a.chatView.MentionsList.SetUsers(userMap)
		a.chatView.MentionsList.SetChannels(channels, userMap, a.slack.UserID)
		a.chatView.MessagesList.SetSelfUserID(a.slack.UserID)
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
