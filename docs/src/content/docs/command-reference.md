---
title: CLI reference
description: The CLI is the source of truth for routing, accounts, preferences and the menu bar.
order: 10
category: Reference
summary: The CLI is the source of truth for routing, accounts, preferences and the menu bar.
---

The macOS app calls these same commands. Each supports `--help`; `--home PATH` selects a separate Noria pool.

| Command | Purpose |
| --- | --- |
| `noria init` | Initialize private settings without replacing existing state |
| `noria start` / `stop` / `restart` | Control the background service and registered client routing |
| `noria status [--json]` | Cached account quotas, router state, integrations and preferences |
| `noria account add codex\|claude [--alias NAME]` | Browser sign-in into a new email-first pool entry |
| `noria account list [--json]` | Emails, aliases and immutable account IDs |
| `noria account login ACCOUNT` | Sign in again and refresh identity/usage |
| `noria account enable\|disable\|remove ACCOUNT` | Change eligibility or unregister an account |
| `noria account rename ACCOUNT [ALIAS]` | Set an alias, or omit it to show the email |
| `noria quota refresh [ACCOUNT]` | Read native quota without a model prompt |
| `noria integrate codex\|claude [--client-home PATH]` | Preview persistent settings |
| `noria integrate ... --yes` | Approve setup and start routing |
| `noria integrate ... --remove --yes` | Restore direct settings and unregister the integration |
| `noria integrations [--json]` | Inspect registered client profiles |
| `noria integrate claude-desktop` | Print configuration for the desktop app's own form |
| `noria health [--json]` | Local reachability and configuration checks |
| `noria doctor` | Native tool and private-file prerequisites |
| `noria preferences [KEY on\|off] [--json]` | Login/update preferences |
| `noria update [--check] [--json]` | Check or install a verified stable release |
| `noria menubar install` / `open` | Install or open the native macOS UI |
| `noria run codex\|claude -- ...` | Optional child-only routing wrapper |
| `noria env codex\|claude` | Advanced manual shell setup; contains the local token |

Preference keys are `start-at-login`, `menubar-at-login`, and `auto-update`. The menu bar setting is macOS-only.

`noria auth-token` exists for native credential helpers. It intentionally prints a secret; do not put its output into issues, recordings, or shared logs.

The old profile-only `account add <id> --provider ...` form remains compatible with existing scripts. There is no public blocking `serve` command.
