package slack

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// TypingEvent represents a user_typing event.
// Note: user_typing is an RTM-only event. This struct is provided
// so the infrastructure is ready when RTM support is added.
type TypingEvent struct {
	ChannelID string
	UserID    string
}

// EventHandler holds typed callback fields — one per event kind.
// UI components register by setting the relevant field(s).
// Nil callbacks are silently skipped.
//
// Note: user_typing and presence_change are RTM-only events and are
// not available via Socket Mode / Events API.
type EventHandler struct {
	OnMessage             func(*slackevents.MessageEvent)
	OnMessageChanged      func(*slackevents.MessageEvent) // SubType "message_changed"
	OnMessageDeleted      func(*slackevents.MessageEvent) // SubType "message_deleted"
	OnReactionAdded       func(*slackevents.ReactionAddedEvent)
	OnReactionRemoved     func(*slackevents.ReactionRemovedEvent)
	OnChannelCreated      func(*slackevents.ChannelCreatedEvent)
	OnChannelArchive      func(*slackevents.ChannelArchiveEvent)
	OnChannelUnarchive    func(*slackevents.ChannelUnarchiveEvent)
	OnChannelRename       func(*slackevents.ChannelRenameEvent)
	OnMemberJoinedChannel func(*slackevents.MemberJoinedChannelEvent)
	OnMemberLeftChannel   func(*slackevents.MemberLeftChannelEvent)
	OnTeamJoin            func(*slackevents.TeamJoinEvent)
	OnPinAdded            func(*slackevents.PinAddedEvent)
	OnPinRemoved          func(*slackevents.PinRemovedEvent)
	OnFileShared          func(*slackevents.FileSharedEvent)
	OnUserStatusChanged   func(*slackevents.UserStatusChangedEvent)
	OnTyping              func(*TypingEvent) // RTM-only; included for future support
	OnConnected           func()
	OnDisconnected        func()
	OnError               func(error)
}

// RunSocketMode creates a socketmode.Client, registers event handlers from
// the provided EventHandler, and runs the event loop. It blocks until ctx
// is cancelled or a fatal error occurs.
func (c *Client) RunSocketMode(ctx context.Context, handler *EventHandler) error {
	smClient := socketmode.New(c.api)
	smHandler := socketmode.NewSocketmodeHandler(smClient)

	registerEventHandlers(smHandler, handler)
	registerLifecycleHandlers(smHandler, handler)

	// Catch-all: log any unhandled socket mode events for debugging.
	smHandler.HandleDefault(func(evt *socketmode.Event, _ *socketmode.Client) {
		slog.Debug("unhandled socket mode event", "type", evt.Type, "data_type", fmt.Sprintf("%T", evt.Data))
	})

	return smHandler.RunEventLoopContext(ctx)
}

// registerEventHandlers wires Events API event types to the appropriate
// EventHandler callbacks.
func registerEventHandlers(smHandler *socketmode.SocketmodeHandler, handler *EventHandler) {
	// Message events — routed by SubType.
	smHandler.HandleEvents(slackevents.Message, func(evt *socketmode.Event, client *socketmode.Client) {
		client.Ack(*evt.Request)

		apiEvt, ok := evt.Data.(slackevents.EventsAPIEvent)
		if !ok {
			slog.Debug("message event: unexpected data type", "type", fmt.Sprintf("%T", evt.Data))
			return
		}
		msg, ok := apiEvt.InnerEvent.Data.(*slackevents.MessageEvent)
		if !ok {
			slog.Debug("message event: unexpected inner type", "type", fmt.Sprintf("%T", apiEvt.InnerEvent.Data))
			return
		}
		slog.Debug("message event received", "channel", msg.Channel, "user", msg.User, "subtype", msg.SubType)

		switch msg.SubType {
		case "message_changed":
			if handler.OnMessageChanged != nil {
				handler.OnMessageChanged(msg)
			}
		case "message_deleted":
			if handler.OnMessageDeleted != nil {
				handler.OnMessageDeleted(msg)
			}
		default:
			if handler.OnMessage != nil {
				handler.OnMessage(msg)
			}
		}
	})

	// Reaction events.
	registerTypedHandler(smHandler, slackevents.ReactionAdded, handler.OnReactionAdded)
	registerTypedHandler(smHandler, slackevents.ReactionRemoved, handler.OnReactionRemoved)

	// Channel events.
	registerTypedHandler(smHandler, slackevents.ChannelCreated, handler.OnChannelCreated)
	registerTypedHandler(smHandler, slackevents.ChannelArchive, handler.OnChannelArchive)
	registerTypedHandler(smHandler, slackevents.ChannelUnarchive, handler.OnChannelUnarchive)
	registerTypedHandler(smHandler, slackevents.ChannelRename, handler.OnChannelRename)

	// Membership events.
	registerTypedHandler(smHandler, slackevents.MemberJoinedChannel, handler.OnMemberJoinedChannel)
	registerTypedHandler(smHandler, slackevents.MemberLeftChannel, handler.OnMemberLeftChannel)

	// Team events.
	registerTypedHandler(smHandler, slackevents.TeamJoin, handler.OnTeamJoin)

	// Pin events.
	registerTypedHandler(smHandler, slackevents.PinAdded, handler.OnPinAdded)
	registerTypedHandler(smHandler, slackevents.PinRemoved, handler.OnPinRemoved)

	// File events.
	registerTypedHandler(smHandler, slackevents.FileShared, handler.OnFileShared)

	// User status events.
	registerTypedHandler(smHandler, slackevents.UserStatusChanged, handler.OnUserStatusChanged)
}

// registerTypedHandler is a generic helper that registers a HandleEvents callback
// which extracts the inner event, type-asserts it, and calls the provided callback.
func registerTypedHandler[T any](smHandler *socketmode.SocketmodeHandler, eventType slackevents.EventsAPIType, callback func(*T)) {
	smHandler.HandleEvents(eventType, func(evt *socketmode.Event, client *socketmode.Client) {
		client.Ack(*evt.Request)

		apiEvt, ok := evt.Data.(slackevents.EventsAPIEvent)
		if !ok {
			return
		}
		inner, ok := apiEvt.InnerEvent.Data.(*T)
		if !ok {
			slog.Warn("unexpected inner event type",
				"event_type", eventType,
				"data_type", fmt.Sprintf("%T", apiEvt.InnerEvent.Data))
			return
		}
		if callback != nil {
			callback(inner)
		}
	})
}

// registerLifecycleHandlers wires socketmode-level connection events to the
// appropriate EventHandler callbacks.
func registerLifecycleHandlers(smHandler *socketmode.SocketmodeHandler, handler *EventHandler) {
	smHandler.Handle(socketmode.EventTypeConnecting, func(evt *socketmode.Event, _ *socketmode.Client) {
		slog.Debug("socket mode connecting")
	})

	smHandler.Handle(socketmode.EventTypeHello, func(evt *socketmode.Event, _ *socketmode.Client) {
		slog.Debug("socket mode hello received")
	})

	smHandler.Handle(socketmode.EventTypeErrorWriteFailed, func(evt *socketmode.Event, _ *socketmode.Client) {
		slog.Warn("socket mode write failed", "data", evt.Data)
		if handler.OnError != nil {
			handler.OnError(fmt.Errorf("socket mode write failed: %v", evt.Data))
		}
	})

	smHandler.Handle(socketmode.EventTypeErrorBadMessage, func(evt *socketmode.Event, _ *socketmode.Client) {
		slog.Warn("socket mode bad message", "data", evt.Data)
		if handler.OnError != nil {
			handler.OnError(fmt.Errorf("socket mode bad message: %v", evt.Data))
		}
	})

	smHandler.Handle(socketmode.EventTypeConnected, func(evt *socketmode.Event, _ *socketmode.Client) {
		slog.Info("socket mode connected")
		if handler.OnConnected != nil {
			handler.OnConnected()
		}
	})

	smHandler.Handle(socketmode.EventTypeDisconnect, func(evt *socketmode.Event, _ *socketmode.Client) {
		slog.Warn("socket mode disconnected")
		if handler.OnDisconnected != nil {
			handler.OnDisconnected()
		}
	})

	smHandler.Handle(socketmode.EventTypeIncomingError, func(evt *socketmode.Event, _ *socketmode.Client) {
		if handler.OnError == nil {
			return
		}
		if err, ok := evt.Data.(error); ok {
			handler.OnError(err)
		} else {
			handler.OnError(fmt.Errorf("socket mode incoming error: %v", evt.Data))
		}
	})

	smHandler.Handle(socketmode.EventTypeConnectionError, func(evt *socketmode.Event, _ *socketmode.Client) {
		slog.Warn("socket mode connection error", "data", evt.Data)
		if handler.OnError != nil {
			if err, ok := evt.Data.(error); ok {
				handler.OnError(err)
			} else {
				handler.OnError(fmt.Errorf("socket mode connection error: %v", evt.Data))
			}
		}
	})

	smHandler.Handle(socketmode.EventTypeInvalidAuth, func(evt *socketmode.Event, _ *socketmode.Client) {
		slog.Error("socket mode invalid auth")
		if handler.OnError != nil {
			handler.OnError(fmt.Errorf("socket mode: invalid auth"))
		}
	})
}
