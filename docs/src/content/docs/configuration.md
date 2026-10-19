---
title: Configuration
description: Noria’s private home, loopback listener, model constraints, and persisted operational state.
order: 6
category: Reference
summary: Defaults and supported controls without manual credential JSON.
---

Use the CLI for ordinary account setup. You do not need to craft credential JSON or modify your existing native client configuration.

## Defaults

| Setting | Default | Control |
| --- | --- | --- |
| Noria home | Absolute `~/.noria` | `NORIA_HOME` environment variable |
| Listener | `127.0.0.1:43119` | `noria serve --listen <address>` |
| Account eligibility | Enabled; any requested model for its provider | Account enable/disable; repeatable `account add --model` |
| Claude usage SDK | `@anthropic-ai/claude-agent-sdk@0.3.287` | Explicit `noria bridge setup` |
| Gateway credential | Random local secret | Created by `noria init` |

The listen address must use a numeric loopback IP. Public interfaces such as `0.0.0.0` are rejected. This is a local single-user gateway, not a hosted inference endpoint.

## Use a different Noria home

Set the same home for every related command and terminal:

```bash
export NORIA_HOME="$HOME/.noria-work"
noria init
noria account add work --provider codex
noria account login work
```

Noria resolves the home to an absolute path. Upstream profiles are kept under that home’s `accounts/<id>`. Changing `NORIA_HOME` selects a separate pool, local secret, and scheduler state. It does not import accounts from the previous home.

## Change the listener

```bash
noria serve --listen 127.0.0.1:43120
```

Use `noria run` or freshly generated `noria env` output for that running daemon. These commands can discover its recorded listen address; old manually exported environment values may still point at the previous port. Use `noria status` to check the current daemon.

`config.json` also stores the default `listen` value. Stop the daemon before manually changing this file. Prefer the command flag for a temporary port change.

## Account controls

```bash
noria account add restricted --provider codex --model gpt-5.4
noria account disable restricted
noria account enable restricted
noria quota refresh
```

Model IDs are exact eligibility constraints, not replacements for the model in a request. The example does not guarantee access to that model. Repeat `--model` at account creation for multiple allowed names. Omit it to leave model eligibility unrestricted by local configuration; upstream access limits still apply.

Adding, removing, enabling, or disabling an account requests an immediate reload in a running daemon. Periodic quota refresh also reloads configuration. If a command reports that live reload failed, resolve that error or restart the daemon before relying on the changed pool. These controls do not cancel work already accepted upstream.

## Files in the private home

| Path | Purpose |
| --- | --- |
| `config.json` | Listener and registered account metadata |
| `secret` | Local gateway authentication secret |
| `accounts/<id>/` | Isolated native authentication profile |
| `scheduler.json` | Bindings, quota snapshots, learned estimates, and cooldown metadata |
| `daemon.json` | Current local daemon address |
| `bridge/` | Explicitly installed Claude SDK dependency |
| `*.lock` | Short control-operation or daemon ownership locks |

Treat the whole home as private. Directories use mode `0700` and Noria-managed state files use `0600` where supported. Files written by upstream native clients follow those clients’ storage behavior. Windows permissions require the operating system’s own access controls as well.

Do not edit `scheduler.json` to force a different route or clear a provider limit. Removing state discards affinity and learning; it does not restore quota. `account remove` unregisters an account but retains the native profile on disk. See [security and limitations](/docs/security-and-limitations) for credential handling and cleanup.
