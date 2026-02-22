# Architecture

## Overview

Slacko is a terminal-based Slack client built with Go. It uses [tview](https://github.com/rivo/tview) for the UI and [slack-go/slack](https://github.com/slack-go/slack) for the Slack API, communicating in real-time via Socket Mode.

```
┌─────────────────────────────────────────────────┐
│                   main.go / cmd/                │
│                  CLI entry point                │
└──────────────────────┬──────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────┐
│                 internal/app/                   │
│         Application lifecycle & wiring          │
│   - Connects UI callbacks to Slack API calls    │
│   - Handles Socket Mode events                  │
│   - Manages state transitions                   │
└───────┬─────────────────────────────┬───────────┘
        │                             │
┌───────▼───────────┐     ┌───────────▼───────────┐
│   internal/ui/    │     │   internal/slack/      │
│   TUI components  │     │   API client wrapper   │
│                   │     │   + event handling      │
│  ├── chat/        │     └───────────────────────┘
│  │  ├── view.go   │
│  │  ├── channels  │
│  │  ├── messages  │
│  │  ├── input     │
│  │  ├── thread    │
│  │  └── pickers   │
│  └── login/       │
└───────────────────┘
```

## Key Components

### `internal/app/app.go` - Application Core

The central orchestrator. It:
- Creates the Slack client and tview application
- Builds all UI components and wires their callbacks
- Handles all Socket Mode events (messages, reactions, presence, etc.)
- Routes slash commands and vim commands to appropriate handlers

### `internal/ui/chat/view.go` - Layout Manager

Manages the main chat layout using tview Flex containers:
- Left panel: `ChannelsTree` (collapsible)
- Center panel: `MessagesList` + `MessageInput` (+ optional `ThreadView`)
- Bottom: `StatusBar` (or `CommandBar` in command mode)
- Modal overlays: pickers, profiles, channel info

### `internal/ui/chat/messages_list.go` - Message Display

Renders messages with:
- Author grouping (consecutive messages from same user)
- Date separators and "New messages" markers
- Themed colors via `StyleWrapper.Tag()`
- Reactions, pins, stars, file attachments, link previews
- Markdown rendering via `internal/markdown/`

### `internal/ui/chat/channels_tree.go` - Channel Navigation

Tree-based channel browser with sections:
- Channels (public), Private Channels, Direct Messages, Group DMs, Slack Connect
- Unread badges, presence indicators, typing indicators

### `internal/slack/client.go` - API Abstraction

Wraps `slack-go/slack` with:
- Rate limit retry logic
- Simplified method signatures
- All API calls used by the app (messages, reactions, files, search, etc.)

### `internal/slack/events.go` - Event Handling

Processes Socket Mode events and dispatches them to registered callbacks.

## Data Flow

### Message Sending
```
User types → MessageInput.send()
  → OnSend callback → app.go handleSend()
    → client.SendMessage() → Slack API
      → Socket Mode event → app.go handleMessageEvent()
        → MessagesList.AppendMessage() → render()
```

### Slash Command
```
User types "/status :wave: Working"
  → MessageInput.send() detects "/" prefix
    → ParseSlashCommand() → OnSlashCommand callback
      → app.go executeSlashCommand()
        → client.SetUserCustomStatus()
```

## Configuration System

Three-phase loading in `config.Load()`:

1. **Embedded defaults**: `config.toml` embedded via `go:embed`
2. **User overrides**: `~/.config/slacko/config.toml` decoded on top
3. **Environment variables**: `SLACKO_USER_TOKEN`, `SLACKO_APP_TOKEN`

Theme resolution adds a 4th step:
- Read `theme.preset` from user config
- Load builtin preset as base
- Re-decode user config on top (only user-specified fields override)

## Design Patterns

### Callback Wiring
UI components expose `SetOnXxx()` methods. `app.go` wires them to API calls:
```go
view.MessagesList.SetOnReplyRequest(func(channelID, threadTS, userName string) {
    // Set up reply context in the input
})
```

### Modal Overlay
Each modal picker follows the pattern:
1. Component struct embedding `*tview.Flex`
2. `Show`/`Hide` methods on `View` that swap the root layout
3. `SetOnSelect`/`SetOnClose` callbacks
4. `Reset()` to clear state on open

### Theme System
`StyleWrapper` wraps `tcell.Style` with stored TOML strings. `Tag()` emits tview color tags like `[green::b]`. All UI rendering uses theme-driven tags instead of hardcoded colors. Six builtin presets are available.
