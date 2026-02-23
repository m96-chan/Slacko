---
layout: default
title: Privacy Policy — Slacko
---

# Privacy Policy

**Last updated: February 23, 2026**

## Overview

Slacko is an open-source terminal client for Slack. Your privacy is important to us. This policy explains what data Slacko accesses and how it is handled.

## Data Collection

**Slacko does not collect, store, or transmit any user data to the developer or any third party.**

All communication happens directly between your device and Slack's API servers. Slacko is a client application that runs entirely on your local machine.

## Slack API Access

When you authorize Slacko, it requests the following categories of access to your Slack workspace:

- **Read access** — View messages, channels, users, files, reactions, pins, and stars in workspaces you belong to
- **Write access** — Send messages, upload files, add reactions, pin messages, and set your status
- **Search** — Search messages and files within your workspaces

These permissions are used solely to provide the TUI client functionality. Slacko does not access data beyond what is displayed to you in the application.

## Token Storage

Your Slack authentication tokens are stored locally on your device using your operating system's secure keyring (e.g., GNOME Keyring, macOS Keychain, Windows Credential Manager). Tokens are never transmitted to any server other than Slack's API.

## OAuth Proxy

When using the default OAuth login flow, the authorization process passes through a Cloudflare Worker (`slacko-oauth.m96-chan.dev`). This proxy:

- Exchanges the OAuth authorization code for an access token with Slack's API
- Passes the token directly to your local machine via a form POST to `localhost`
- **Does not log, store, or retain any tokens or user data**
- Source code is available at [github.com/m96-chan/Slacko](https://github.com/m96-chan/Slacko/tree/main/workers/oauth-proxy)

You can bypass the proxy entirely by configuring your own Slack App with `client_secret` in self-hosted mode.

## Third-Party Services

Slacko interacts only with:

- **Slack API** (`api.slack.com`) — For all workspace functionality
- **OAuth Proxy** (`slacko-oauth.m96-chan.dev`) — Only during the initial OAuth login (optional)

No analytics, telemetry, or tracking services are used.

## Open Source

Slacko is fully open source under the MIT License. You can audit the complete source code at [github.com/m96-chan/Slacko](https://github.com/m96-chan/Slacko).

## Changes

If this privacy policy changes, the updated version will be posted here with a revised date.

## Contact

For questions about this privacy policy, please open an issue on [GitHub](https://github.com/m96-chan/Slacko/issues).
