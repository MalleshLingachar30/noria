---
title: Configuration and state
description: Where Noria keeps settings and how to select a custom pool.
order: 11
category: Reference
summary: Where Noria keeps settings and how to select a custom pool.
---

`NORIA_HOME` defaults to `~/.noria`. All preferences and routing state are owned by the Go CLI. Swift does not keep a second copy of account eligibility or quota logic.

```bash
noria init --home "$HOME/.noria-work"
noria account add codex --home "$HOME/.noria-work"
noria start --home "$HOME/.noria-work"
```

Use the same home for related commands. The managed native auth helper records the home explicitly, so GUI clients do not need to inherit a shell variable.

| File | Purpose |
| --- | --- |
| `config.json` | Registered accounts, optional aliases, listener and login/update preferences |
| `secret` | Random local gateway credential |
| `accounts/<id>/` | Isolated native auth profiles |
| `scheduler.json` | Hashed bindings, snapshots, estimates and cooldowns |
| `integrations.json` | Owned native settings and their restoration data |
| `backups/` | Private backups taken before native client integration |
| `service.json` / `daemon.json` | Desired service lifetime and current address |
| `update.json` | Last update-check result |
| `logs/service.log` | Bounded service diagnostics, not model transcripts |
| `*.lock` | Stable OS lock files; their presence does not mean a process owns them |

The listener defaults to `127.0.0.1:43119` and must use a numeric loopback IP. Stop Noria before changing the `listen` field, then start it to reinstall the registered routes at the new address. A managed desktop or another profile may need its own corresponding change.

Directories and Noria-managed state use private permissions where supported. Do not share this home or expose the listener through a public tunnel. Windows also relies on the current user's filesystem access controls.

A single macOS menu instance displays the home selected when it was opened. `noria menubar open --home PATH` records that UI selection for its next login launch. It does not merge accounts from multiple pools.
