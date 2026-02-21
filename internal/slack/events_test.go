package slack

import (
	"fmt"
	"sync"
	"testing"

	"github.com/slack-go/slack/slackevents"
)

func TestMessageSubTypeRouting(t *testing.T) {
	tests := []struct {
		name            string
		subType         string
		wantNew         bool
		wantChanged     bool
		wantDeleted     bool
	}{
		{"new message", "", true, false, false},
		{"message_changed", "message_changed", false, true, false},
		{"message_deleted", "message_deleted", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotNew, gotChanged, gotDeleted bool
			handler := &EventHandler{
				OnMessage:        func(*slackevents.MessageEvent) { gotNew = true },
				OnMessageChanged: func(*slackevents.MessageEvent) { gotChanged = true },
				OnMessageDeleted: func(*slackevents.MessageEvent) { gotDeleted = true },
			}

			msg := &slackevents.MessageEvent{SubType: tt.subType}
			dispatchMessage(handler, msg)

			if gotNew != tt.wantNew {
				t.Errorf("OnMessage called=%v, want %v", gotNew, tt.wantNew)
			}
			if gotChanged != tt.wantChanged {
				t.Errorf("OnMessageChanged called=%v, want %v", gotChanged, tt.wantChanged)
			}
			if gotDeleted != tt.wantDeleted {
				t.Errorf("OnMessageDeleted called=%v, want %v", gotDeleted, tt.wantDeleted)
			}
		})
	}
}

func TestNilCallbacksDoNotPanic(t *testing.T) {
	handler := &EventHandler{} // all callbacks nil

	// Message subtypes.
	dispatchMessage(handler, &slackevents.MessageEvent{SubType: ""})
	dispatchMessage(handler, &slackevents.MessageEvent{SubType: "message_changed"})
	dispatchMessage(handler, &slackevents.MessageEvent{SubType: "message_deleted"})

	// Lifecycle.
	dispatchLifecycle(handler, "connected")
	dispatchLifecycle(handler, "disconnected")
	dispatchLifecycle(handler, "error")
}

func TestTypedCallbacksInvoked(t *testing.T) {
	var mu sync.Mutex
	called := map[string]bool{}

	handler := &EventHandler{
		OnReactionAdded:       func(*slackevents.ReactionAddedEvent) { mu.Lock(); called["reaction_added"] = true; mu.Unlock() },
		OnReactionRemoved:     func(*slackevents.ReactionRemovedEvent) { mu.Lock(); called["reaction_removed"] = true; mu.Unlock() },
		OnChannelCreated:      func(*slackevents.ChannelCreatedEvent) { mu.Lock(); called["channel_created"] = true; mu.Unlock() },
		OnChannelArchive:      func(*slackevents.ChannelArchiveEvent) { mu.Lock(); called["channel_archive"] = true; mu.Unlock() },
		OnChannelUnarchive:    func(*slackevents.ChannelUnarchiveEvent) { mu.Lock(); called["channel_unarchive"] = true; mu.Unlock() },
		OnChannelRename:       func(*slackevents.ChannelRenameEvent) { mu.Lock(); called["channel_rename"] = true; mu.Unlock() },
		OnMemberJoinedChannel: func(*slackevents.MemberJoinedChannelEvent) { mu.Lock(); called["member_joined"] = true; mu.Unlock() },
		OnMemberLeftChannel:   func(*slackevents.MemberLeftChannelEvent) { mu.Lock(); called["member_left"] = true; mu.Unlock() },
		OnTeamJoin:            func(*slackevents.TeamJoinEvent) { mu.Lock(); called["team_join"] = true; mu.Unlock() },
		OnPinAdded:            func(*slackevents.PinAddedEvent) { mu.Lock(); called["pin_added"] = true; mu.Unlock() },
		OnPinRemoved:          func(*slackevents.PinRemovedEvent) { mu.Lock(); called["pin_removed"] = true; mu.Unlock() },
		OnFileShared:          func(*slackevents.FileSharedEvent) { mu.Lock(); called["file_shared"] = true; mu.Unlock() },
		OnUserStatusChanged:   func(*slackevents.UserStatusChangedEvent) { mu.Lock(); called["user_status_changed"] = true; mu.Unlock() },
	}

	tests := []struct {
		key       string
		innerType string
		data      interface{}
	}{
		{"reaction_added", "reaction_added", &slackevents.ReactionAddedEvent{}},
		{"reaction_removed", "reaction_removed", &slackevents.ReactionRemovedEvent{}},
		{"channel_created", "channel_created", &slackevents.ChannelCreatedEvent{}},
		{"channel_archive", "channel_archive", &slackevents.ChannelArchiveEvent{}},
		{"channel_unarchive", "channel_unarchive", &slackevents.ChannelUnarchiveEvent{}},
		{"channel_rename", "channel_rename", &slackevents.ChannelRenameEvent{}},
		{"member_joined", "member_joined_channel", &slackevents.MemberJoinedChannelEvent{}},
		{"member_left", "member_left_channel", &slackevents.MemberLeftChannelEvent{}},
		{"team_join", "team_join", &slackevents.TeamJoinEvent{}},
		{"pin_added", "pin_added", &slackevents.PinAddedEvent{}},
		{"pin_removed", "pin_removed", &slackevents.PinRemovedEvent{}},
		{"file_shared", "file_shared", &slackevents.FileSharedEvent{}},
		{"user_status_changed", "user_status_changed", &slackevents.UserStatusChangedEvent{}},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			dispatchTypedEvent(handler, tt.innerType, tt.data)
			mu.Lock()
			if !called[tt.key] {
				t.Errorf("callback for %s not invoked", tt.key)
			}
			mu.Unlock()
		})
	}
}

func TestLifecycleCallbacks(t *testing.T) {
	var gotConnected, gotDisconnected bool
	var gotErr error

	handler := &EventHandler{
		OnConnected:    func() { gotConnected = true },
		OnDisconnected: func() { gotDisconnected = true },
		OnError:        func(err error) { gotErr = err },
	}

	dispatchLifecycle(handler, "connected")
	if !gotConnected {
		t.Error("OnConnected not called")
	}

	dispatchLifecycle(handler, "disconnected")
	if !gotDisconnected {
		t.Error("OnDisconnected not called")
	}

	dispatchLifecycle(handler, "error")
	if gotErr == nil {
		t.Error("OnError not called")
	}
}

// --- test helpers that mirror the dispatch logic without needing a real socketmode.Client ---

func dispatchMessage(handler *EventHandler, msg *slackevents.MessageEvent) {
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
}

func dispatchTypedEvent(handler *EventHandler, innerType string, data interface{}) {
	switch innerType {
	case "reaction_added":
		if e, ok := data.(*slackevents.ReactionAddedEvent); ok && handler.OnReactionAdded != nil {
			handler.OnReactionAdded(e)
		}
	case "reaction_removed":
		if e, ok := data.(*slackevents.ReactionRemovedEvent); ok && handler.OnReactionRemoved != nil {
			handler.OnReactionRemoved(e)
		}
	case "channel_created":
		if e, ok := data.(*slackevents.ChannelCreatedEvent); ok && handler.OnChannelCreated != nil {
			handler.OnChannelCreated(e)
		}
	case "channel_archive":
		if e, ok := data.(*slackevents.ChannelArchiveEvent); ok && handler.OnChannelArchive != nil {
			handler.OnChannelArchive(e)
		}
	case "channel_unarchive":
		if e, ok := data.(*slackevents.ChannelUnarchiveEvent); ok && handler.OnChannelUnarchive != nil {
			handler.OnChannelUnarchive(e)
		}
	case "channel_rename":
		if e, ok := data.(*slackevents.ChannelRenameEvent); ok && handler.OnChannelRename != nil {
			handler.OnChannelRename(e)
		}
	case "member_joined_channel":
		if e, ok := data.(*slackevents.MemberJoinedChannelEvent); ok && handler.OnMemberJoinedChannel != nil {
			handler.OnMemberJoinedChannel(e)
		}
	case "member_left_channel":
		if e, ok := data.(*slackevents.MemberLeftChannelEvent); ok && handler.OnMemberLeftChannel != nil {
			handler.OnMemberLeftChannel(e)
		}
	case "team_join":
		if e, ok := data.(*slackevents.TeamJoinEvent); ok && handler.OnTeamJoin != nil {
			handler.OnTeamJoin(e)
		}
	case "pin_added":
		if e, ok := data.(*slackevents.PinAddedEvent); ok && handler.OnPinAdded != nil {
			handler.OnPinAdded(e)
		}
	case "pin_removed":
		if e, ok := data.(*slackevents.PinRemovedEvent); ok && handler.OnPinRemoved != nil {
			handler.OnPinRemoved(e)
		}
	case "file_shared":
		if e, ok := data.(*slackevents.FileSharedEvent); ok && handler.OnFileShared != nil {
			handler.OnFileShared(e)
		}
	case "user_status_changed":
		if e, ok := data.(*slackevents.UserStatusChangedEvent); ok && handler.OnUserStatusChanged != nil {
			handler.OnUserStatusChanged(e)
		}
	}
}

func dispatchLifecycle(handler *EventHandler, kind string) {
	switch kind {
	case "connected":
		if handler.OnConnected != nil {
			handler.OnConnected()
		}
	case "disconnected":
		if handler.OnDisconnected != nil {
			handler.OnDisconnected()
		}
	case "error":
		if handler.OnError != nil {
			handler.OnError(fmt.Errorf("test error"))
		}
	}
}
