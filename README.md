<p align="center">
  <img src="docs/assets/icon.png" alt="Slacko" width="128">
</p>

<h1 align="center">Slacko</h1>

<p align="center">
  A lightweight, keyboard-driven TUI client for <a href="https://slack.com">Slack</a>, built in Go.
</p>

<p align="center">
  <a href="https://github.com/m96-chan/Slacko/actions/workflows/ci.yml"><img src="https://github.com/m96-chan/Slacko/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://goreportcard.com/report/github.com/m96-chan/Slacko"><img src="https://goreportcard.com/badge/github.com/m96-chan/Slacko" alt="Go Report Card"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
</p>

<p align="center"><img src="docs/assets/hero.png" alt="Slacko — TUI Client for Slack" width="600"></p>

## Features

- **Workspace Navigation** — Browse channels (public/private), DMs, group DMs, and Slack Connect channels
- **Messaging** — Send, edit, and delete messages with rich text support
- **Threads** — View and reply to threaded conversations
- **Reactions** — Add and remove emoji reactions
- **Real-time Updates** — Receive messages and events via Slack Socket Mode
- **Mentions** — Autocomplete @user and #channel mentions with fuzzy search
- **File Sharing** — Upload and download file attachments
- **Search** — Search messages and channels
- **Notifications** — Desktop notifications for mentions and DMs
- **Vim-style Keybindings** — Fully customizable keyboard shortcuts with command mode
- **Theming** — Customizable colors and styles via TOML configuration
- **Markdown Rendering** — Render Slack's mrkdwn format with syntax highlighting
- **User Presence** — Online/away/DND status indicators
- **Unread Indicators** — Visual markers for unread channels and messages
- **Multi-workspace** — Switch between multiple Slack workspaces
- **OAuth Login** — Browser-based authorization with zero configuration

## Installation

### Homebrew

```bash
brew tap m96-chan/tap
brew install slacko
```

### Arch Linux (AUR)

```bash
yay -S slacko-bin
```

### Nix

```bash
nix profile install github:m96-chan/Slacko
```

Or try it without installing:

```bash
nix run github:m96-chan/Slacko
```

### Go

```bash
go install github.com/m96-chan/Slacko@latest
```

### Binary Releases

Download pre-built binaries from [GitHub Releases](https://github.com/m96-chan/Slacko/releases).

### Build from Source

```bash
git clone https://github.com/m96-chan/Slacko.git
cd Slacko
go build -o slacko .
```

## Getting Started

Just run `slacko` — it will open your browser for Slack authorization. After granting access, you're connected.

```bash
slacko
```

### Alternative: Manual Token Setup

If you prefer to use your own Slack App, see the [Slack App Setup Guide](docs/SLACK_APP_SETUP.md) for detailed instructions.

```bash
export SLACKO_USER_TOKEN="xoxp-..."
export SLACKO_APP_TOKEN="xapp-..."
slacko
```

Tokens are stored securely in your OS keyring after first login.

## Configuration

The configuration file is located at:

| OS | Path |
|---|---|
| Linux | `~/.config/slacko/config.toml` |
| macOS | `~/Library/Application Support/slacko/config.toml` |
| Windows | `%AppData%\slacko\config.toml` |

See [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for the full reference.

### Example

```toml
mouse = true
editor = "default"
auto_focus = true
messages_limit = 50

[markdown]
enabled = true
syntax_theme = "monokai"

[timestamps]
enabled = true
format = "3:04PM"

[notifications]
enabled = true

[typing_indicator]
send = true
receive = true

[presence]
enabled = true
```

## Keybindings

All keybindings are customizable in `config.toml`. See [docs/KEYBINDINGS.md](docs/KEYBINDINGS.md) for the full reference.

### Global

| Key | Action |
|---|---|
| `1` / `2` / `3` | Focus channels / messages / input |
| `Ctrl+W` | Focus previous panel |
| `Ctrl+L` | Focus next panel |
| `Ctrl+B` | Toggle channel tree |
| `Ctrl+K` | Channel picker (fuzzy search) |
| `Ctrl+S` | Search messages |
| `Ctrl+T` | Switch workspace |
| `:` | Command mode |
| `?` | Help |
| `Ctrl+C` | Quit |

### Messages

| Key | Action |
|---|---|
| `j` / `k` | Navigate messages |
| `r` | Reply in thread |
| `e` | Edit own message |
| `d` | Delete own message |
| `+` / `-` | Add / remove reaction |
| `t` | Open thread |
| `y` | Copy message text |
| `o` | Open links in browser |

### Input

| Key | Action |
|---|---|
| `Enter` | Send message |
| `Shift+Enter` | Newline |
| `Tab` | Autocomplete mention |
| `Ctrl+E` | External editor |
| `Ctrl+F` | File upload |
| `Ctrl+V` | Paste from clipboard |

## Architecture

```
slacko/
├── internal/
│   ├── app/                    # Application lifecycle & event wiring
│   ├── ui/
│   │   ├── login/              # OAuth & manual token login
│   │   └── chat/               # Main chat interface
│   │       ├── view.go         # Layout orchestrator
│   │       ├── channels_tree.go
│   │       ├── messages_list.go
│   │       ├── message_input.go
│   │       ├── thread_view.go
│   │       └── ...
│   ├── config/                 # TOML configuration system
│   ├── slack/                  # Slack API client & Socket Mode events
│   ├── oauth/                  # Local OAuth flow (browser-based)
│   ├── keyring/                # Secure token storage (OS keyring)
│   ├── markdown/               # Slack mrkdwn renderer
│   ├── notifications/          # Desktop notifications
│   ├── clipboard/              # Clipboard operations
│   └── logger/                 # Structured logging
├── workers/
│   └── oauth-proxy/            # Cloudflare Worker for OAuth proxy
├── docs/                       # Documentation & GitHub Pages site
├── main.go
└── go.mod
```

### Key Dependencies

| Library | Purpose |
|---|---|
| [slack-go/slack](https://github.com/slack-go/slack) | Slack Web API + Socket Mode |
| [rivo/tview](https://github.com/rivo/tview) | Terminal UI framework |
| [gdamore/tcell](https://github.com/gdamore/tcell) | Terminal rendering |
| [BurntSushi/toml](https://github.com/BurntSushi/toml) | Configuration parsing |
| [alecthomas/chroma](https://github.com/alecthomas/chroma) | Syntax highlighting |
| [sahilm/fuzzy](https://github.com/sahilm/fuzzy) | Fuzzy search |
| [zalando/go-keyring](https://github.com/zalando/go-keyring) | Secure token storage |

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Acknowledgments

- [discordo](https://github.com/ayn2op/discordo) — The Discord TUI client that inspired this project
- [slack-go/slack](https://github.com/slack-go/slack) — Go Slack API library
- [tview](https://github.com/rivo/tview) — Terminal UI library
