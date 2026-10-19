---
title: Set up once, keep working
description: Install Noria, add your accounts, and use your normal clients with automatic routing.
order: 1
category: Get started
summary: Install Noria, add your accounts, and use your normal clients with automatic routing.
---

Noria keeps your coding sessions on one account until an applicable limit is reached, then continues through another eligible account. After setup, open **Codex or Claude Code normally**. You do not need a special launch command for each session.

## 1. Install

On macOS or Linux:

```bash
curl -fsSL https://noria.ashray.xyz/install.sh | bash
```

The installer verifies release checksums and installs into `~/.local/bin`. On macOS it also installs and opens the native **Noria** menu bar app. It does not change your existing Codex or Claude settings until you approve an integration.

On Windows, run this in PowerShell:

```powershell
irm https://noria.ashray.xyz/install.ps1 | iex
```

Already use npm? `npm install -g @amxv/noria` remains available. On macOS, follow it with `noria menubar install` and `noria menubar open`. Standalone Noria binaries need neither Node.js nor an Agent SDK. Install the native `codex` and/or `claude` client you intend to use.

## 2. Add your accounts

In the macOS menu, choose **Add Codex Account…** or **Add Claude Account…**. Complete sign-in in your browser. Your account appears under its email address, followed by its five-hour and weekly usage.

Without the menu bar:

```bash
noria account add codex
noria account add claude
```

Repeat for each distinct subscription. No invented account name is required. Use a different account at each sign-in; adding the same subscription twice does not create extra allowance. At least two accounts for a provider are needed for automatic failover.

## 3. Configure your usual clients once

Choose **Configure Clients…** in the menu and select the Codex or Claude Code profile you actually use. Review the path, then approve. Custom configuration folders can be selected too.

The CLI equivalent previews the change first:

```bash
noria integrate codex
noria integrate codex --yes

noria integrate claude
noria integrate claude --yes
```

Noria backs up the original settings, installs a local base URL and token helper, and starts its background service. **Reopen existing client windows once** so they pick up the new settings. Your normal login, history, skills, model settings, and project preferences remain intact.

For custom roots such as GoldenGoose, desktop-specific setup, or manual TOML/JSON snippets, see [Connect your clients](/docs/native-clients).

## 4. Use your tools normally

Run `codex`, run `claude`, or open a local client that reads the configured profile. Noria chooses accounts automatically and preserves session affinity. Open the menu to see usage and reset countdowns.

Turn on **Launch Menu Bar at Login** and **Start Noria at Login** to make this your everyday setup. They are separate choices: the service can run without the menu bar.

**Stop Noria** restores the direct settings of the profiles Noria manages; their normal native login is unchanged. **Start Noria** turns pool routing back on. Reopen clients that cache routing after switching. See [Everyday use](/docs/everyday-use).
