---
title: "CLI reference"
description: "Service, accounts, client integration, preferences, and updates."
order: 5
category: "Reference"
summary: "Service, accounts, client integration, preferences, and updates."
---

## Background service

```sh
noria init
noria start
noria stop
noria restart
noria status --json
noria health --json
```

Start/Stop return control to the terminal. Repeated calls are safe. Stop restores managed direct-provider settings before stopping. There is no public `serve` command.

## Accounts

```sh
noria account add codex
noria account add claude --alias "Work"
noria account list --json
noria account relogin alex@example.com
noria account rename alex@example.com "Work"
noria account rename alex@example.com ""
noria account enable alex@example.com
noria account disable alex@example.com
noria account remove alex@example.com
noria quota refresh
```

Email is the default name. Use an immutable ID when an email/alias is ambiguous. Remove retains isolated profile files for recovery.

Advanced automation can create an unsigned-in profile using `noria account add <id> --provider codex|claude`, then `noria account login <id>`.

## Persistent client setup

```sh
noria integrate codex
noria integrate codex --yes
noria integrate claude --client-home "$HOME/.gg/claude" --yes
noria integrate codex --remove --yes
noria integrations --json
```

Without `--yes`, integration previews the target and changes. Restore preserves unrelated edits; a conflicting external routing edit is not silently overwritten.

## Login preferences and menu

```sh
noria preferences --json
noria preferences start-at-login on
noria preferences menubar-at-login on
noria preferences auto-update on
noria menubar install
noria menubar open
```

The menu is macOS-only. Background lifecycle works on macOS, Linux, and Windows. Linux manual Start can fall back to a local supervisor when user systemd is unavailable; its login autostart requires a working user service manager.

## Updates and advanced helpers

```sh
noria update --check
noria update
```

Verified updates defer while requests or streaming connections are active. The menu's automatic checks call this same CLI operation.

`--home <path>` and `NORIA_HOME` select a different private pool. `noria auth-token` is a credential helper and intentionally prints a secret: never share its output. `noria env` also prints local credentials. Neither is necessary for normal daily use after integration.
