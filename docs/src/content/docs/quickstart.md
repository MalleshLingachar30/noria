---
title: "Set up once, code normally"
description: "Install Noria, add accounts, and connect your everyday coding clients."
order: 1
category: "Start"
summary: "Install Noria, add accounts, and connect your everyday coding clients."
---

Noria runs quietly on your computer. After one-time setup, launch Codex or Claude Code as usual. Sessions stay on their account while it can serve, and Noria chooses another account when a limit requires it.

## 1. Install

**On macOS**, install the native menu bar and CLI together:

```sh
curl -fsSL https://noria.ashray.xyz/install.sh | sh
```

An **N** appears in your menu bar. The app lives beside the CLI under `~/.local/bin`; no administrator privileges are needed. The installer verifies the release checksum. Use this CLI installation route rather than dragging a browser-downloaded bundle into Applications. [Installation details](/docs/installation-and-upgrades) explain the macOS distinction.

**On Linux**, the same native installer sets up the Go CLI:

```sh
curl -fsSL https://noria.ashray.xyz/install.sh | sh
```

**On Windows**, run this from PowerShell:

```powershell
irm https://noria.ashray.xyz/install.ps1 | iex
```

All three platforms download checksum-verified binaries from [GitHub Releases](https://github.com/amxv/noria/releases). There is no Node or npm prerequisite for Noria. Install the native `codex` and/or `claude` executable for the providers you use.

## 2. Add your accounts

Click **N → Add Codex Account…** or **Add Claude Account…**. Finish the native provider sign-in in your browser, choosing the intended account. Noria saves the isolated pool login, discovers its email address, and refreshes usage.

CLI equivalents:

```sh
noria account add codex
noria account add claude
```

Repeat Add for additional accounts. **Email addresses are the default names; aliases are optional.** Automatic failover needs another usable account of the same provider and requested model.

## 3. Connect your normal clients

Choose **N → Configure Clients…**, select the client folder, and approve the change. Noria backs up the routing fields and preserves your native login, conversations, model settings, and unrelated preferences.

From the CLI:

```sh
noria start
noria integrate codex       # preview the target file and changes
noria integrate codex --yes
noria integrate claude --yes
```

Codex uses `$CODEX_HOME/config.toml`, defaulting to `~/.codex/config.toml`. Claude Code uses `$CLAUDE_CONFIG_DIR/settings.json`, defaulting to `~/.claude/settings.json`. Clients reading those roots receive persistent routing, not merely the shell that ran setup.

Use the custom folder picker or `--client-home` for separate app roots:

```sh
noria integrate codex --client-home "$HOME/.gg/codex" --yes
noria integrate claude --client-home "$HOME/.gg/claude" --yes
```

**Reopen clients that were already running** so they reload the settings. [Client setup](/docs/native-clients) has exact manual TOML/JSON examples and desktop/editor differences.

## 4. Work normally

```sh
codex
# or
claude
```

No wrapper is required. Closing the terminal or quitting the menu bar does not stop the background service.

For automatic startup, enable **Launch Menu Bar at Login** and **Start Noria at Login**. They are independent preferences:

```sh
noria preferences menubar-at-login on   # macOS menu bar
noria preferences start-at-login on    # background service
```

## Everyday controls

Open **N** to inspect five-hour and weekly quota remaining and reset times for every account. Opening the menu refreshes usage asynchronously while showing the last snapshot immediately. Select an account to enable/disable it, sign in again, or set an alias. **Health Check** tests local setup without a model prompt.

```sh
noria status
noria stop
noria start
noria upgrade
```

**Stop restores direct routing using each managed client's original saved login.** It does not copy the last-used pool account into that client. Reopen already-running clients after switching routes. If every matching account is exhausted, Noria reports the shortage rather than buying capacity or changing models.
