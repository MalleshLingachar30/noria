---
title: CLI reference
description: The commands for account management, routing, updates, and configuration.
order: 7
category: Reference
---

The `noria` CLI is the source of truth for the background service and the macOS menu bar. Add `--help` to most commands for available options.

## Start and stop

| Command | What it does |
| --- | --- |
| `noria start` | Start the background proxy and enable registered client routing. |
| `noria stop` | Stop routing and restore the direct settings Noria manages. |
| `noria restart` | Restart the background service. |
| `noria status` | Show accounts, usage remaining, reset times, and routing state. |
| `noria health` | Check client configuration and the local gateway. |

Use `--json` with `status` or `health` for structured output.

## Accounts

| Command | What it does |
| --- | --- |
| `noria account add codex` | Start the browser sign-in to add a Codex account. |
| `noria account add claude` | Start the browser sign-in to add a Claude account. |
| `noria account list` | Show accounts and their display names. |
| `noria account relogin EMAIL` | Sign in again to an existing account. |
| `noria account rename EMAIL "Alias"` | Set an optional display name; empty string restores the email. |
| `noria account enable EMAIL` | Allow new work to use the account. |
| `noria account disable EMAIL` | Exclude the account from new placement. |
| `noria account remove EMAIL` | Unregister the account. |
| `noria quota refresh` | Fetch current quota information for the pool. |

Use an account ID instead of its email when two entries are ambiguous. Find IDs with `noria account list --json`.

## Client integration

```sh
noria integrate codex                     # Preview changes
noria integrate codex --yes               # Apply changes
noria integrate claude --yes
noria integrate codex --remove --yes      # Remove managed configuration
noria integrations --json                 # List registered configurations
```

For clients using a nonstandard configuration folder, pass `--client-home` with the directory path. See [Connect your clients](/docs/native-clients).

## Preferences and upgrades

```sh
noria preferences start-at-login on
noria preferences menubar-at-login on    # macOS
noria preferences auto-update on
noria upgrade --check
noria upgrade
noria menubar open                      # macOS
```

Change `on` to `off` to disable a preference. Upgrades wait for active requests to finish.

## Advanced options

`NORIA_HOME` or `--home <path>` selects a separate Noria configuration and account pool (default `~/.noria`).

`noria auth-token` is an internal local credential helper. It prints a secret, so don't paste its output into logs or support messages. `noria env codex|claude` and `noria run codex|claude -- …` are optional, one-off routing tools; normal clients should use persistent integration instead.
