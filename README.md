# Slacko

A lightweight, keyboard-driven TUI (Terminal User Interface) client for [Slack](https://slack.com), built in Go. Inspired by [discordo](https://github.com/ayn2op/discordo).

> [!WARNING]
> Slacko is currently in early development. Expect breaking changes and incomplete features.

## Features

- **Workspace Navigation** - Browse workspaces, channels (public/private), direct messages, and group DMs
- **Messaging** - Send, edit, and delete messages with rich text support
- **Threads** - View and reply to threaded conversations
- **Reactions** - Add and remove emoji reactions to messages
- **Real-time Updates** - Receive messages and events in real-time via Slack Socket Mode
- **Mentions** - Autocomplete @user and #channel mentions with fuzzy search
- **File Sharing** - Upload and download file attachments
- **Search** - Search messages and channels
- **Notifications** - Desktop notifications for mentions and DMs
- **Vim-style Keybindings** - Fully customizable keyboard shortcuts
- **Theming** - Customizable colors and styles via TOML configuration
- **Markdown Rendering** - Render Slack's mrkdwn format with syntax highlighting
- **User Presence** - View online/away/DND status indicators
- **Unread Indicators** - Visual markers for unread channels and messages

## Screenshots

<!-- TODO: Add screenshots -->

## Installation

### From Source

```bash
go install github.com/m96-chan/Slacko@latest
```

### Build from Source

```bash
git clone https://github.com/m96-chan/Slacko.git
cd Slacko
go build -o slacko .
```

### Arch Linux (AUR)

<!-- TODO: AUR package -->

```bash
yay -S slacko-git
```

## Slack App Setup

Slacko requires a Slack App with specific permissions. Follow these steps:

### 1. Create a Slack App

1. Go to [api.slack.com/apps](https://api.slack.com/apps)
2. Click **"Create New App"** > **"From scratch"**
3. Name it (e.g., "Slacko TUI") and select your workspace

### 2. Enable Socket Mode

1. Navigate to **"Socket Mode"** in the left sidebar
2. Toggle **"Enable Socket Mode"** on
3. Create an **App-Level Token** with `connections:write` scope
4. Save the token (`xapp-...`) — you'll need this

### 3. Configure OAuth Scopes

Navigate to **"OAuth & Permissions"** and add the following **Bot Token Scopes**:

| Scope | Purpose |
|---|---|
| `channels:history` | View messages in public channels |
| `channels:read` | List public channels |
| `channels:write` | Manage public channels |
| `chat:write` | Send messages |
| `emoji:read` | List custom emoji |
| `files:read` | View files |
| `files:write` | Upload files |
| `groups:history` | View messages in private channels |
| `groups:read` | List private channels |
| `groups:write` | Manage private channels |
| `im:history` | View direct messages |
| `im:read` | List direct messages |
| `im:write` | Send direct messages |
| `mpim:history` | View group DMs |
| `mpim:read` | List group DMs |
| `mpim:write` | Send group DMs |
| `pins:read` | View pinned messages |
| `reactions:read` | View reactions |
| `reactions:write` | Add/remove reactions |
| `search:read` | Search messages |
| `team:read` | View workspace info |
| `users:read` | View user profiles |
| `users:read.email` | View user emails |
| `users.profile:read` | View user profiles |

### 4. Subscribe to Events

Navigate to **"Event Subscriptions"** and enable events. Add the following **Bot Events**:

- `message.channels` — Messages in public channels
- `message.groups` — Messages in private channels
- `message.im` — Direct messages
- `message.mpim` — Group DMs
- `reaction_added` — Reaction added
- `reaction_removed` — Reaction removed
- `channel_created` — New channel created
- `channel_archive` — Channel archived
- `channel_unarchive` — Channel unarchived
- `member_joined_channel` — User joined channel
- `member_left_channel` — User left channel
- `user_status_changed` — User status update

### 5. Install to Workspace

1. Navigate to **"Install App"**
2. Click **"Install to Workspace"** and authorize
3. Copy the **Bot User OAuth Token** (`xoxb-...`)

## Configuration

The configuration file is located at:

| OS | Path |
|---|---|
| Linux | `$XDG_CONFIG_HOME/slacko/config.toml` or `~/.config/slacko/config.toml` |
| macOS | `~/Library/Application Support/slacko/config.toml` |
| Windows | `%AppData%\slacko\config.toml` |

### Authentication

Set your tokens via environment variables or the config file:

```bash
# Environment variables (recommended)
export SLACKO_BOT_TOKEN="xoxb-your-bot-token"
export SLACKO_APP_TOKEN="xapp-your-app-level-token"
```

Or in `config.toml`:

```toml
# Tokens can also be stored in the OS keyring for security.
# On first launch, Slacko will prompt for tokens if not set.
```

### Example Configuration

```toml
# Focus message input automatically when a channel is selected
auto_focus = true

# Enable mouse support
mouse = true

# External editor command (uses $EDITOR if set to "default")
editor = "default"

# Maximum number of autocomplete suggestions (0 = disabled)
autocomplete_limit = 20

# Number of messages to fetch per channel (1-100)
messages_limit = 50

# Show attachment links inline
show_attachment_links = true

[markdown]
# Enable mrkdwn rendering
enabled = true
# Syntax highlighting theme for code blocks
theme = "monokai"

[timestamps]
enabled = true
format = "3:04PM"

[date_separator]
enabled = true
format = "January 2, 2006"
character = "─"

[notifications]
enabled = true
[notifications.sound]
enabled = true
only_on_mention = true

[typing_indicator]
send = true
receive = true

[threads]
# Auto-expand threads with new replies
auto_expand = false
# Show thread reply count in messages list
show_reply_count = true

[presence]
# Show user presence indicators
enabled = true
# Icons for presence states
online = "●"
away = "◐"
dnd = "⊘"
offline = "○"
```

## Keybindings

All keybindings are customizable in `config.toml` under the `[keybinds]` section.

### Global

| Key | Action |
|---|---|
| `Ctrl+W` | Focus workspace/channel tree |
| `Ctrl+T` | Focus messages list |
| `Ctrl+I` | Focus message input |
| `Ctrl+H` | Cycle focus to previous panel |
| `Ctrl+L` | Cycle focus to next panel |
| `Ctrl+B` | Toggle channel tree visibility |
| `Ctrl+K` | Open channel picker (fuzzy search) |
| `Ctrl+C` | Quit |

### Channel Tree

| Key | Action |
|---|---|
| `j` / `k` | Navigate down / up |
| `g` / `G` | Jump to first / last |
| `Enter` | Select channel or expand section |
| `-` | Collapse section |
| `p` | Move to parent node |

### Messages List

| Key | Action |
|---|---|
| `j` / `k` | Select next / previous message |
| `g` / `G` | Select first / last message |
| `J` / `K` | Scroll view without changing selection |
| `r` | Reply in thread |
| `R` | Reply in thread (also send to channel) |
| `e` | Edit own message |
| `D` then `d` | Delete own message |
| `+` | Add reaction |
| `-` | Remove reaction |
| `t` | Open thread view |
| `y` | Copy message text |
| `u` | Copy message permalink |
| `o` | Open links/attachments in browser |
| `Esc` | Cancel selection |

### Message Input

| Key | Action |
|---|---|
| `Enter` | Send message |
| `Alt+Enter` | Insert newline |
| `Tab` | Autocomplete mention |
| `Ctrl+E` | Open external editor |
| `Ctrl+\` | Open file picker for upload |
| `Ctrl+V` | Paste from clipboard |
| `Esc` | Cancel reply / clear input |

### Thread View

| Key | Action |
|---|---|
| `j` / `k` | Navigate replies |
| `r` | Reply to thread |
| `q` / `Esc` | Close thread view |

## Architecture

```
slacko/
├── cmd/                        # CLI entry point
│   └── root.go                 # Cobra root command with flags
├── internal/
│   ├── app/                    # Application lifecycle
│   │   └── app.go
│   ├── ui/                     # Terminal UI components
│   │   ├── login/              # Token input form
│   │   ├── chat/               # Main chat interface
│   │   │   ├── view.go         # Layout orchestrator
│   │   │   ├── state.go        # Slack state & event handlers
│   │   │   ├── channels_tree.go    # Workspace/channel navigation
│   │   │   ├── messages_list.go    # Message display
│   │   │   ├── message_input.go    # Message composition
│   │   │   ├── thread_view.go      # Thread panel
│   │   │   ├── channels_picker.go  # Quick channel switcher
│   │   │   ├── mentions_list.go    # @mention autocomplete
│   │   │   ├── reactions.go        # Emoji reaction picker
│   │   │   └── status_bar.go       # Typing/presence footer
│   │   └── util.go
│   ├── config/                 # Configuration system
│   │   ├── config.go           # TOML parsing
│   │   ├── config.toml         # Default configuration
│   │   ├── keybinds.go         # Keybinding definitions
│   │   └── theme.go            # Theme/color definitions
│   ├── slack/                  # Slack API abstraction
│   │   ├── client.go           # API client wrapper
│   │   └── events.go           # Socket Mode event handling
│   ├── markdown/               # mrkdwn renderer
│   │   └── renderer.go
│   ├── notifications/          # Desktop notifications
│   │   └── notifications.go
│   ├── keyring/                # Secure token storage
│   │   └── keyring.go
│   ├── clipboard/              # Clipboard operations
│   │   └── clipboard.go
│   └── logger/                 # Structured logging
│       └── logger.go
├── main.go                     # Entry point
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

### Key Dependencies

| Library | Purpose |
|---|---|
| [slack-go/slack](https://github.com/slack-go/slack) | Slack Web API + Socket Mode client |
| [rivo/tview](https://github.com/rivo/tview) | Terminal UI framework |
| [gdamore/tcell](https://github.com/gdamore/tcell) | Terminal rendering |
| [BurntSushi/toml](https://github.com/BurntSushi/toml) | Configuration parsing |
| [alecthomas/chroma](https://github.com/alecthomas/chroma) | Syntax highlighting |
| [sahilm/fuzzy](https://github.com/sahilm/fuzzy) | Fuzzy search |
| [zalando/go-keyring](https://github.com/zalando/go-keyring) | Secure token storage |
| [spf13/cobra](https://github.com/spf13/cobra) | CLI framework |

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes (`git commit -m 'Add my feature'`)
4. Push to the branch (`git push origin feature/my-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Acknowledgments

- [discordo](https://github.com/ayn2op/discordo) — The Discord TUI client that inspired this project
- [slack-go/slack](https://github.com/slack-go/slack) — Go Slack API library
- [tview](https://github.com/rivo/tview) — Terminal UI library
