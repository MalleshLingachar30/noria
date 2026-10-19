---
title: "Configuration and private state"
description: "Paths, backups, native controls, and service boundaries."
order: 6
category: "Reference"
summary: "Paths, backups, native controls, and service boundaries."
---

`NORIA_HOME` defaults to `~/.noria`. Pool authentication profiles live under that home, separately from your normal client history and settings.

| Private path | Purpose |
| --- | --- |
| `config.json` | Account/email/alias metadata and preferences |
| `secret` | Random local gateway credential |
| `accounts/<id>/` | Isolated native sign-in profiles |
| `scheduler.json` | Bindings, usage snapshots, blockers, and learned estimates |
| `integrations.json`, `backups/` | Managed routing fields and original settings backups |
| `service.json`, `daemon.json` | Desired runtime state and live address |
| `logs/service.log` | Private bounded service diagnostics |
| `update.json` | Last update check/result |

Private files have restrictive permissions where supported; Windows also relies on user access controls. Never publish the private home.

## Native controls

Codex usage comes from the installed `codex app-server` with the account's isolated `CODEX_HOME`. Claude usage comes from the installed native `claude` executable's experimental `get_usage` control request. There is no Node/Agent SDK runtime bridge, user prompt, or reset-credit redemption in these reads.

Noria forwards native Responses HTTP/SSE/WebSocket and Claude Messages HTTP/SSE. Authentication is substituted upstream; models and request bodies are not translated between providers.

## Service ownership

An OS service or local supervisor owns the daemon. Swift only renders CLI state and invokes CLI actions. OS locks release on process exit, so a crash does not leave a fake live PID-file lock. Account and binding metadata persist across restarts.

The listener is restricted to numeric loopback addresses. Public interfaces are rejected. Applications running remotely or with explicit provider overrides need their own supported configuration.
