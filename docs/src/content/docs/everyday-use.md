---
title: Day-to-day use
description: Start, stop, check usage, and upgrade Noria while you keep coding.
order: 4
category: Get started
---

After setup, Noria stays out of your way. Just open Codex or Claude Code normally.

## On macOS: the menu bar

Click the **N** icon to see your accounts, remaining allowance, and reset times. Opening the menu refreshes usage in the background. Only providers you've added are shown, and a quota window appears only if the provider reports it—for example, Codex plans without a five-hour limit won't show an empty five-hour row.

The main menu stays focused on account usage, **Start/Stop**, and **Refresh Usage**. Under **Configure**, you'll find **Configure Codex**, **Configure Claude Code**, account additions, Health Check, login preferences, updates, and documentation. To relogin, rename, or disable an existing account, use its submenu.

Closing the menu bar app **does not** stop the background service.

## In your terminal

```sh
noria status           # Accounts and remaining limits
noria quota refresh    # Read usage now
noria health           # Check your local setup
noria stop             # Temporarily use providers directly
noria start            # Route through Noria again
noria restart
noria upgrade          # Check and install a new release
```

Start and Stop return immediately; you don't need to leave a terminal open.

**What happens when you stop?** Noria restores routing settings that it installed, so your clients use their original direct provider account again. Restart already-running clients if they have cached the old settings. Noria does not change the account you're signed into directly.

## Start at login

On macOS, **Launch Menu Bar at Login** and **Start Noria at Login** are separate settings. You can have one without the other.

The CLI equivalents are:

```sh
noria preferences start-at-login on
noria preferences menubar-at-login on    # macOS
noria preferences auto-update on
```

Use `off` to disable a preference.

## Upgrades

Choose **Check for Updates…** in the macOS menu, or run:

```sh
noria upgrade --check
noria upgrade
```

Noria will wait rather than interrupt an active model request during an upgrade. See [Installation & upgrades](/docs/installation-and-upgrades) for details.

If something looks wrong, start with `noria health` and the [troubleshooting guide](/docs/troubleshooting).
