---
title: Everyday use
description: Manage your pool from the menu bar or a few nonblocking commands.
order: 4
category: Start
summary: Manage your pool from the menu bar or a few nonblocking commands.
---

Once your accounts and clients are connected, Noria runs independently in the background. Keep using your normal coding tools.

## The menu bar

The **N** icon shows the router state. Open it to see every account's email or chosen alias, five-hour and weekly percentages **remaining**, and reset countdowns. Hover over a quota line for the exact local reset date and time.

Cached values appear immediately. A fresh read runs when the menu opens, without blocking its controls. A failed read is shown as unavailable or stale, never as free quota.

Use an account's submenu to disable/enable it, sign in again, rename it, or remove it from the pool. **Health Check** verifies the local endpoint and registered settings without sending a model prompt. There are no quota-exhaustion notifications.

**Quit Menu Bar** closes only the UI. It does not stop your router or change client settings.

## Start and stop

```bash
noria start
noria stop
noria restart
noria status
```

These commands return to your terminal after the operation; no terminal needs to remain open. Start is safe to repeat. The public foreground `serve` command is no longer part of the workflow.

Start enables Noria routing in its registered profiles. Stop restores their previous direct-provider settings, retaining the native client's last login. It does not sign that client into a different Noria pool account. Clients that keep configuration in memory need to be reopened after a route change.

Manually configured applications that Noria does not manage—such as a Claude Desktop third-party profile entered through its own form—must be switched back in that application's settings. Noria cannot rewrite a cloud-hosted client's routing.

## Login preferences

The menu exposes independent checkboxes:

**Launch Menu Bar at Login** starts the native macOS UI. **Start Noria at Login** starts routing automatically, even if you do not open the UI. **Automatically Update** lets the CLI install verified stable releases when it can safely restart.

```bash
noria preferences start-at-login on
noria preferences menubar-at-login on
noria preferences auto-update on
```

Use `off` to disable a preference. macOS may ask you to approve a login item in System Settings. Linux login startup requires a systemd user session; manual background start/stop works without one. Windows uses the current user's Task Scheduler, without storing a password.

## Updates

```bash
noria upgrade --check
noria upgrade
```

The updater verifies checksums and stages the new binary before replacement. The macOS CLI and menu app update together. Active requests take priority: a busy router defers applying the update rather than cancelling a generation. Automatic checks run from the service as well as through the menu's CLI actions.

## Check an account now

```bash
noria quota refresh
noria status --json
noria health --json
```

These commands read operational metadata, not your conversations. If every matching account is exhausted, Noria reports that and any known next usable reset. It never buys extra quota or changes your requested model.
