# Slack App Setup Guide

This guide walks you through creating a Slack App for use with Slacko.

## 1. Create a Slack App

1. Go to [api.slack.com/apps](https://api.slack.com/apps)
2. Click **"Create New App"** > **"From scratch"**
3. Name it (e.g., "Slacko TUI") and select your workspace
4. Click **"Create App"**

## 2. Enable Socket Mode

Socket Mode allows Slacko to receive events without a public HTTP endpoint.

1. In the left sidebar, click **"Socket Mode"**
2. Toggle **"Enable Socket Mode"** on
3. When prompted, create an **App-Level Token**:
   - Name: `slacko-socket`
   - Scope: `connections:write`
4. Click **"Generate"**
5. **Save this token** (`xapp-...`) - this is your App Token

## 3. Configure User Token Scopes

Navigate to **"OAuth & Permissions"** in the left sidebar. Under **"User Token Scopes"**, add:

### Required Scopes

| Scope | Purpose |
|---|---|
| `channels:history` | View messages in public channels |
| `channels:read` | List public channels |
| `channels:write` | Join/leave channels, set topic |
| `chat:write` | Send messages |
| `emoji:read` | List custom emoji for reaction picker |
| `files:read` | View file attachments |
| `files:write` | Upload files |
| `groups:history` | View messages in private channels |
| `groups:read` | List private channels |
| `groups:write` | Manage private channels |
| `im:history` | View direct messages |
| `im:read` | List direct message conversations |
| `im:write` | Send direct messages |
| `mpim:history` | View group DM messages |
| `mpim:read` | List group DM conversations |
| `mpim:write` | Send group DMs |
| `pins:read` | View pinned messages |
| `pins:write` | Pin/unpin messages |
| `reactions:read` | View emoji reactions |
| `reactions:write` | Add/remove reactions |
| `search:read` | Search messages and files |
| `stars:read` | View starred items |
| `stars:write` | Star/unstar items |
| `team:read` | View workspace info |
| `users:read` | View user profiles and presence |
| `users:read.email` | View user email addresses |
| `users.profile:read` | View detailed user profiles |
| `users.profile:write` | Set own status |
| `reminders:read` | View reminders |
| `reminders:write` | Create reminders |

## 4. Subscribe to Events

Navigate to **"Event Subscriptions"** in the left sidebar:

1. Toggle **"Enable Events"** on
2. Expand **"Subscribe to events on behalf of users"** (NOT "Subscribe to bot events") and add:

| Event | Description |
|---|---|
| `message.channels` | Messages in public channels |
| `message.groups` | Messages in private channels |
| `message.im` | Direct messages |
| `message.mpim` | Group DM messages |
| `reaction_added` | Reaction added to a message |
| `reaction_removed` | Reaction removed from a message |
| `channel_created` | New channel created |
| `channel_archive` | Channel archived |
| `channel_unarchive` | Channel unarchived |
| `channel_rename` | Channel renamed |
| `member_joined_channel` | User joined a channel |
| `member_left_channel` | User left a channel |
| `user_status_changed` | User status/presence changed |

3. Click **"Save Changes"**

## 5. Install to Workspace

1. Navigate to **"Install App"** in the left sidebar
2. Click **"Install to Workspace"**
3. Review the permissions and click **"Allow"**
4. **Copy the User OAuth Token** (`xoxp-...`) - this is your User Token

## 6. Configure Slacko

You have several options for providing tokens:

### Option A: OAuth Login (Recommended for teams)

If a workspace admin distributes Slacko with OAuth credentials, each user can log in via browser without manually copying tokens.

#### Admin Setup

1. In your Slack App settings, navigate to **"OAuth & Permissions"**
2. Under **"Redirect URLs"**, add: `http://localhost`
3. Note the **Client ID** and **Client Secret** from **"Basic Information"**

Then configure Slacko via `config.toml` or environment variables:

**config.toml:**
```toml
[oauth]
client_id = "your-client-id"
client_secret = "your-client-secret"
app_token = "xapp-your-app-token"
```

**Environment variables:**
```bash
export SLACKO_CLIENT_ID="your-client-id"
export SLACKO_CLIENT_SECRET="your-client-secret"
export SLACKO_APP_TOKEN="xapp-your-app-token"
```

#### User Experience

When OAuth is configured, Slacko shows a single "Authorize with Slack" button.
Pressing it opens the browser for Slack authorization. After granting access,
the token is saved automatically and Slacko connects.

### Option B: Environment Variables

```bash
export SLACKO_USER_TOKEN="xoxp-your-user-token"
export SLACKO_APP_TOKEN="xapp-your-app-token"
slacko
```

### Option C: Interactive Login

Just run `slacko` - it will prompt for tokens on first launch and store them securely in your OS keyring.

### Option D: Shell RC File

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
export SLACKO_USER_TOKEN="xoxp-..."
export SLACKO_APP_TOKEN="xapp-..."
```

## Troubleshooting

### "Authentication failed"
- Verify both tokens are correct (user token starts with `xoxp-`, app token with `xapp-`)
- Make sure you installed the app to the correct workspace
- Re-install the app if you changed scopes after initial install

### "Not in channel" errors
- The bot must be invited to private channels: `/invite @YourBotName`
- Public channels are accessible automatically

### Missing messages or events
- Verify all event subscriptions from step 4 are enabled
- Check that Socket Mode is enabled (step 2)
- Re-install the app after adding new event subscriptions

### "rate_limited" errors
- Slacko has built-in retry logic for rate limits
- If persistent, reduce `messages_limit` in config.toml
