# Contributing to Slacko

Thank you for your interest in contributing to Slacko! This document covers everything you need to get started.

## Development Setup

### Prerequisites

- **Go 1.25+** ([download](https://go.dev/dl/))
- **golangci-lint** ([install](https://golangci-lint.run/welcome/install/)) for linting
- A working Slack workspace with a configured Slack App (see [Slack App Setup](docs/SLACK_APP_SETUP.md))

### Clone and Build

```bash
git clone https://github.com/m96-chan/Slacko.git
cd Slacko
go build -o slacko .
```

Or use the Makefile:

```bash
make build    # Build the binary
make install  # Install to $GOPATH/bin
```

### Running

```bash
export SLACKO_BOT_TOKEN="xoxb-..."
export SLACKO_APP_TOKEN="xapp-..."
./slacko
```

## Project Structure

```
slacko/
├── main.go                          # Entry point
├── cmd/root.go                      # CLI flags and startup
├── internal/
│   ├── app/app.go                   # Application lifecycle, callback wiring, event dispatch
│   ├── ui/
│   │   ├── login/form.go            # Token input form (shown when tokens are missing)
│   │   ├── chat/
│   │   │   ├── view.go              # Main layout orchestrator (flex panels, modal overlays)
│   │   │   ├── channels_tree.go     # Workspace/channel sidebar navigation
│   │   │   ├── messages_list.go     # Message display and selection
│   │   │   ├── message_input.go     # Message composition with autocomplete
│   │   │   ├── thread_view.go       # Thread panel (replies)
│   │   │   ├── channels_picker.go   # Ctrl+K fuzzy channel switcher
│   │   │   ├── mentions_list.go     # @user / #channel autocomplete dropdown
│   │   │   ├── reactions_picker.go  # Emoji reaction picker
│   │   │   ├── search_picker.go     # Message search modal
│   │   │   ├── file_picker.go       # File upload picker
│   │   │   ├── pins_picker.go       # Pinned messages viewer
│   │   │   ├── starred_picker.go    # Starred items viewer
│   │   │   ├── user_profile.go      # User profile panel
│   │   │   ├── channel_info.go      # Channel info panel
│   │   │   ├── workspace_picker.go  # Multi-workspace switcher
│   │   │   ├── command_bar.go       # Vim-style : command input
│   │   │   ├── commands.go          # Slash command definitions
│   │   │   ├── status_bar.go        # Bottom status/typing bar
│   │   │   └── timeparse.go         # Time parsing for /schedule
│   │   └── keys/keys.go             # Key name normalization
│   ├── config/
│   │   ├── config.go                # TOML config loading (3-phase)
│   │   ├── config.toml              # Embedded default configuration
│   │   ├── keybinds.go              # Keybinding struct definitions
│   │   ├── theme.go                 # Theme/style types and TOML unmarshalling
│   │   └── themes.go                # Built-in theme presets
│   ├── slack/
│   │   ├── client.go                # Slack API wrapper with rate-limit retry
│   │   └── events.go                # Socket Mode event loop and dispatch
│   ├── markdown/renderer.go         # Slack mrkdwn to tview rendering
│   ├── notifications/notifier.go    # Desktop notification support
│   ├── keyring/
│   │   ├── keyring.go               # OS keyring token storage
│   │   └── workspaces.go            # Multi-workspace credential management
│   ├── clipboard/clipboard.go       # System clipboard operations
│   ├── typing/tracker.go            # Typing indicator state tracking
│   └── logger/logger.go             # Structured logging setup
├── Makefile
├── go.mod / go.sum
└── LICENSE
```

## Development Workflow

### Running Tests

```bash
go test ./...
# or
make test
```

### Linting

The project uses [golangci-lint](https://golangci-lint.run/) with the configuration in `.golangci.yml`:

```bash
golangci-lint run ./...
# or
make lint
```

### Formatting and Vetting

```bash
gofmt -s -w .     # or: make fmt
go vet ./...       # or: make vet
```

### Building

```bash
go build -o slacko .   # or: make build
```

## Code Style

- Follow standard Go conventions ([Effective Go](https://go.dev/doc/effective_go), [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments))
- Run `gofmt` and `go vet` before committing -- CI will enforce this
- Exported types and functions must have doc comments
- Keep functions focused; if a handler grows large, extract helpers
- Use `slog` for logging (not `fmt.Println` or `log`)
- Error messages should be lowercase and not end with punctuation
- Test new functionality -- aim for coverage of non-trivial logic

## Pull Request Process

1. **Fork** the repository and create a feature branch:
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make your changes** with clear, atomic commits. Write meaningful commit messages:
   ```
   Add fuzzy search to workspace picker

   Integrates sahilm/fuzzy for workspace name matching in the
   workspace picker modal (Ctrl+T). Closes #42.
   ```

3. **Run checks locally** before pushing:
   ```bash
   make fmt
   make vet
   make lint
   make test
   ```

4. **Push** and open a Pull Request against `main`:
   ```bash
   git push origin feature/my-feature
   ```

5. **In the PR description**, reference any related issues (e.g., `Closes #42`) and describe what changed and why.

6. **CI must pass** -- the PR will be checked with `go test`, `golangci-lint`, and `go vet`.

7. Respond to review feedback. Once approved, the PR will be squash-merged.

## Issue Guidelines

- **Bug reports**: Include your OS, terminal emulator, Go version, and steps to reproduce. Paste any error output from the log.
- **Feature requests**: Describe the use case and expected behavior. If you have a design in mind, sketch it out.
- **Questions**: Open a Discussion instead of an issue.

Use the provided issue templates (bug report / feature request) when available.

## Architecture Overview

For a detailed look at how the codebase is organized and how data flows through the application, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
