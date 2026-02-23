---
layout: default
title: Slacko — TUI Client for Slack
---

# Slacko

A lightweight, keyboard-driven TUI (Terminal User Interface) client for [Slack](https://slack.com), built in Go.

## Features

- **Real-time messaging** — Send, edit, and delete messages with real-time updates via Socket Mode
- **Full workspace navigation** — Public/private channels, DMs, group DMs, and threads
- **Vim-style keybindings** — Fully customizable keyboard shortcuts
- **Reactions & emoji** — Add and remove emoji reactions
- **File sharing** — Upload and download file attachments
- **Search** — Search messages and channels with fuzzy matching
- **Desktop notifications** — Get notified for mentions and DMs
- **Theming** — Customizable colors and styles via TOML configuration
- **Markdown rendering** — Render Slack's mrkdwn format with syntax highlighting
- **Secure token storage** — Tokens stored in your OS keyring

## Installation

### Homebrew (macOS / Linux)

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

### Go

```bash
go install github.com/m96-chan/Slacko@latest
```

### Binary Releases

Download pre-built binaries from [GitHub Releases](https://github.com/m96-chan/Slacko/releases).

## Getting Started

Just run `slacko` — it will open your browser for Slack authorization. After granting access, you're connected.

## Links

- [GitHub Repository](https://github.com/m96-chan/Slacko)
- [Privacy Policy](privacy)
- [Terms of Service](terms)
- [Support](support)
- [License (MIT)](https://github.com/m96-chan/Slacko/blob/main/LICENSE)
