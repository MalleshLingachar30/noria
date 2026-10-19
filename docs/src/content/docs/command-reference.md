---
title: Command reference
description: Set up accounts, operate the local gateway, read quota, and launch native clients.
order: 5
category: Reference
summary: The v0.1.0 command surface and the side effects of each command.
---

Run `noria --help` for the overview and `<command> --help` for command-local usage. `noria --version` prints the installed version. Examples use a POSIX shell unless noted.

## Initialize

```bash
noria init
```

Creates the private Noria home, initial configuration, and random local gateway secret. Existing account registration is preserved. The default home is `~/.noria`; set `NORIA_HOME` to choose another.

## Manage accounts

```bash
noria account add <id> --provider codex|claude [--model <model> ...]
noria account login <id>
noria account list
noria account disable <id>
noria account enable <id>
noria account remove <id>
```

| Operation | Effect |
| --- | --- |
| `add` | Registers an enabled account and creates an isolated profile. No credential import. |
| `login` | Runs native `codex login` or `claude auth login` for that account. Interactive sign-in is your action. |
| `list` | Lists account labels, provider, enabled state, and model restrictions. No credential values. |
| `disable` / `enable` | Changes whether the account is eligible for routing. |
| `remove` | Unregisters the account. Retains native profile credentials/history; does not securely erase or revoke them. |

The ID must be 1–64 letters, digits, underscores, or hyphens and start with a letter or digit. It is a local label, not an upstream account identifier. `--model` can be repeated and restricts exact requested model names; it never changes a request’s model.

Registration and eligibility changes request immediate reload in a running daemon. A reported live-reload failure needs attention even if the configuration was saved. See [accounts](/docs/accounts) for auth isolation.

## Serve

```bash
noria serve
noria serve --listen 127.0.0.1:43120
```

Runs the gateway in the foreground. The default is `127.0.0.1:43119`. Numeric loopback listeners only. The daemon refreshes quota, maintains persisted affinity, and handles native provider requests. Stop it with Ctrl+C for graceful shutdown. Only one daemon should own a Noria home.

## Inspect status

```bash
noria status
noria status --json
```

Queries the local daemon when available, otherwise shows saved state and reports that the daemon is unavailable. The default table shows five-hour and weekly headroom, reset countdowns, in-flight requests, and account state. JSON adds scoped quota windows and precise timestamps. Both forms omit credentials, profile paths, and session identifiers. A saved snapshot can be stale; JSON output is not proof that an account can serve.

## Refresh quota

```bash
noria quota refresh
noria quota refresh <id>
```

Reads native quota telemetry for all enabled accounts or one account. With a running daemon, refresh updates its live scheduler. Otherwise it updates saved scheduler state while holding the appropriate local lock. Inspect the per-account results; an unavailable read does not mean free capacity.

Codex uses `account/rateLimits/read` through the native app-server. Claude uses the explicit SDK bridge. Neither reader submits a user prompt or redeems reset credits.

## Check prerequisites

```bash
noria doctor
```

Checks local configuration, permissions, native executables, and bridge installation. It does not authenticate a real workflow or run a live model request. Read diagnostics individually when only one provider is needed. See [troubleshooting](/docs/troubleshooting).

## Launch a native client

```bash
noria run codex -- [native arguments...]
noria run claude -- [native arguments...]
```

Runs the installed native executable with child-only routing overrides. It retains the user’s existing client history/config roots and does not persistently rewrite them. Arguments after `--` go to the native client. The daemon must be running separately.

## Print manual setup

```bash
noria env codex
noria env claude
```

Prints POSIX shell configuration/instructions for intentional integration. **This output contains local gateway authentication.** Do not publish it or capture it in shared logs. Prefer the `run` wrapper for routine launches. See [native client integration](/docs/native-clients).

## Install the Claude bridge

```bash
noria bridge setup
```

Explicitly installs `@anthropic-ai/claude-agent-sdk@0.3.287` under `NORIA_HOME/bridge` using npm. Requires Node.js, npm, and network access for installation. The bridge script is embedded in the Go binary; its SDK dependency is separate. There is no automatic network installation during request handling.

The pinned usage method is experimental and validated at runtime. This setup does not log into an account or send a model prompt.
