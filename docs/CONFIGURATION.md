# Configuration Reference

Configuration file location:

| OS | Path |
|---|---|
| Linux | `$XDG_CONFIG_HOME/slacko/config.toml` or `~/.config/slacko/config.toml` |
| macOS | `~/Library/Application Support/slacko/config.toml` |
| Windows | `%AppData%\slacko\config.toml` |

A default config is created on first launch if none exists.

## General Settings

| Key | Type | Default | Description |
|---|---|---|---|
| `mouse` | bool | `true` | Enable mouse support |
| `editor` | string | `"default"` | External editor command (`"default"` uses `$EDITOR`) |
| `auto_focus` | bool | `true` | Focus message input when a channel is selected |
| `show_attachment_links` | bool | `true` | Show attachment source URLs |
| `autocomplete_limit` | int | `10` | Max autocomplete suggestions (0 = disabled) |
| `messages_limit` | int | `50` | Messages to fetch per channel (1-100) |
| `download_dir` | string | `""` | File download directory (empty = system default) |
| `ascii_icons` | bool | `false` | Use ASCII-only icons instead of Unicode |

## Sections

### `[markdown]`

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `true` | Enable mrkdwn rendering |
| `syntax_theme` | string | `"monokai"` | Chroma syntax highlighting theme for code blocks |

### `[timestamps]`

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `true` | Show message timestamps |
| `format` | string | `"3:04PM"` | Go time format string |

### `[date_separator]`

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `true` | Show date separators between messages |
| `character` | string | `"─"` | Separator line character |

### `[notifications]`

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `true` | Enable desktop notifications |

#### `[notifications.sound]`

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `false` | Enable notification sound |

### `[typing_indicator]`

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `true` | Enable typing indicators |
| `send` | bool | `true` | Send typing events to Slack |
| `receive` | bool | `true` | Show when others are typing |

### `[threads]`

| Key | Type | Default | Description |
|---|---|---|---|
| `show_inline` | bool | `false` | Show thread replies inline |

### `[presence]`

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `true` | Show user presence indicators |

## Theme System

### Presets

Set `theme.preset` to use a builtin theme:

```toml
[theme]
preset = "default"  # default, dark, light, monokai, solarized_dark, solarized_light
```

### Custom Overrides

Override individual styles on top of a preset:

```toml
[theme]
preset = "monokai"

# Override just the author color
[theme.messages_list.author]
foreground = "cyan"
attributes = "bold"
```

### Style Format

Each style supports:
- `foreground` - Color name (`"red"`, `"green"`, `"#ff0000"`)
- `background` - Color name
- `attributes` - Pipe-separated: `"bold"`, `"dim"`, `"underline"`, `"bold|underline"`

### Theme Sections

- `[theme.border]` - `.focused`, `.normal`
- `[theme.title]` - `.focused`, `.normal`
- `[theme.channels_tree]` - `.channel`, `.selected`, `.unread`
- `[theme.messages_list]` - `.message`, `.author`, `.timestamp`, `.selected`, `.reply`, `.system_message`, `.edited_indicator`, `.pin_indicator`, `.file_attachment`, `.reaction_self`, `.reaction_other`, `.date_separator`, `.new_msg_separator`
- `[theme.message_input]` - `.text`, `.placeholder`
- `[theme.thread_view]` - `.author`, `.timestamp`, `.parent_label`, `.separator`, `.edited_indicator`, `.file_attachment`, `.reaction`
- `[theme.markdown_style]` - `.user_mention`, `.channel_mention`, `.special_mention`, `.link`, `.inline_code`, `.code_fence`, `.blockquote_mark`, `.blockquote_text`
- `[theme.modal]` - `.input_background`, `.secondary_text`
- `[theme.status_bar]` - `.text`, `.background`

## Environment Variables

| Variable | Description |
|---|---|
| `SLACKO_USER_TOKEN` | User OAuth Token (`xoxp-...`) |
| `SLACKO_APP_TOKEN` | App-Level Token (`xapp-...`) |
| `EDITOR` | External editor for message composition |
