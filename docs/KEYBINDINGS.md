# Keybindings Reference

All keybindings are customizable in `config.toml` under `[keybinds]`.

## Global

| Key | Config Key | Action |
|---|---|---|
| `1` | `focus_channels` | Focus channel tree |
| `2` | `focus_messages` | Focus messages list |
| `3` | `focus_input` | Focus message input |
| `t` | `toggle_thread` | Toggle thread view |
| `Ctrl+B` | `toggle_channels` | Toggle channel tree visibility |
| `Ctrl+K` | `channel_picker` | Open channel picker |
| `Ctrl+S` | `search` | Open search |
| `Ctrl+C` | `quit` | Quit |
| `?` | `help` | Show help |
| `Ctrl+T` | `switch_team` | Switch workspace |
| `:` | `command_mode` | Open vim command bar |
| `m` | `mark_read` | Mark channel as read |
| `M` | `mark_all_read` | Mark all channels as read |
| `P` | `pinned_messages` | Show pinned messages |
| `S` | `starred_items` | Show starred items |
| `Ctrl+O` | `channel_info` | Show channel info |

## Channel Tree

| Key | Config Key | Action |
|---|---|---|
| `k` / `j` | `up` / `down` | Navigate up / down |
| `g` / `G` | `top` / `bottom` | Jump to first / last |
| `Enter` | `select_current` | Select channel or expand |
| `c` | `collapse` | Collapse section |
| `p` | `move_to_parent` | Move to parent node |
| `i` | `copy_channel_id` | Copy channel ID |

## Messages List

| Key | Config Key | Action |
|---|---|---|
| `k` / `j` | `scroll_up` / `scroll_down` | Select prev / next message |
| `Enter` | `select_current` | Select message |
| `r` | `reply` | Reply in thread |
| `e` | `edit` | Edit own message |
| `d` | `delete` | Delete own message |
| `+` | `reactions` | Add reaction |
| `-` | `remove_reaction` | Remove reaction |
| `t` | `thread` | Open thread view |
| `y` | `yank` | Copy message text |
| `u` | `copy_permalink` | Copy permalink |
| `o` | `open_file` | Open file/link |
| `p` | `pin` | Pin/unpin message |
| `s` | `star` | Star/unstar message |
| `U` | `user_profile` | View user profile |
| `Esc` | `cancel` | Cancel selection |

## Message Input

| Key | Config Key | Action |
|---|---|---|
| `Enter` | `send` | Send message |
| `Shift+Enter` | `newline` | Insert newline |
| `Tab` | `tab_complete` | Autocomplete |
| `Ctrl+E` | `open_editor` | Open external editor |
| `Ctrl+F` | `open_file_picker` | Open file picker |
| `Ctrl+V` | `paste` | Paste from clipboard |
| `Esc` | `cancel` | Cancel reply/edit |

## Thread View

| Key | Config Key | Action |
|---|---|---|
| `k` / `j` | `up` / `down` | Navigate replies |
| `r` | `reply` | Reply to thread |
| `Esc` | `close` | Close thread |

## Pickers (Channels, Search, Pins, Starred, Files)

| Key | Config Key | Action |
|---|---|---|
| `Esc` | `close` | Close picker |
| `Ctrl+P` | `up` | Move up |
| `Ctrl+N` | `down` | Move down |
| `Enter` | `select` | Select item |

The starred picker also has:

| Key | Config Key | Action |
|---|---|---|
| `x` | `unstar` | Remove star from item |

## Slash Commands

Type these in the message input:

| Command | Description |
|---|---|
| `/help` | Show available commands |
| `/status :emoji: text` | Set your status |
| `/clear-status` | Clear your status |
| `/topic new topic` | Set channel topic |
| `/leave` | Leave current channel |
| `/join #channel` | Join a channel |
| `/who` | List channel members |
| `/schedule in 30m message` | Schedule a message |
| `/scheduled` | List scheduled messages |
| `/remind in 1h reminder` | Set a reminder |
| `/reminders` | List your reminders |
| `/search query` | Search messages |
| `/open url` | Open URL in browser |

## Vim Command Mode

Press `:` to open the command bar. Available commands:

| Command | Aliases | Description |
|---|---|---|
| `:q` | `:quit` | Quit Slacko |
| `:theme name` | | Switch theme preset |
| `:join #channel` | | Join a channel |
| `:leave` | | Leave current channel |
| `:search query` | | Search messages |
| `:mark-read` | | Mark channel as read |
| `:mark-all-read` | | Mark all channels as read |
| `:open url` | | Open URL in browser |
| `:reconnect` | | Reconnect to Slack |
| `:debug` | | Toggle debug logging |
| `:set key=value` | | Set a config value |
| `:workspace` | `:ws` | Switch workspace |
