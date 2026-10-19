---
title: Changelog
description: Release notes and validation boundaries for Noria.
order: 11
category: Reference
summary: Release notes for Noria, including the Codex settings fix in v0.3.1.
---

## 0.3.1 · 2026-10-20

- Fixed **Configure Codex** crashing when an existing `config.toml` contains nested arrays of tables, such as `[[skills.config]]`.
- Keep the original `model_provider` line and its comments in place when stopping or removing Noria routing.

## 0.3.0 · 2026-10-20

- Native-only distribution: the GitHub release installers are the single path for macOS, Linux, and Windows. The Go binary and macOS menu require no Node or npm runtime or installation tooling.
- `noria upgrade --check` and `noria upgrade` are the CLI-owned upgrade workflow, also used by menu bar manual and automatic updates. Downloads remain checksum-verified and active work delays replacement.
- Removed the old named-account registration and relogin compatibility path. All new accounts now use browser sign-in, display their email by default, and have optional display-only aliases. No account migrations or duplicate commands.
- Simplified native release verification, including platform-independent extraction path checks and installer smoke tests.

## 0.2.0 · 2026-10-20

- Native macOS menu bar with email-first accounts, optional aliases, five-hour/weekly usage and reset times, refresh-on-open, account actions, and a no-prompt health check.
- Nonblocking Start/Stop/Restart on macOS, Linux, and Windows. Stop restores managed direct-client routing without changing the client's login. Removed the public `serve` command.
- Persistent Codex/Claude Code setup with credential helpers, previews, backups, comment preservation, and conflict-aware restoration.
- Browser sign-in, email discovery, and immediate quota refresh; separate service/menu login preferences.
- CLI-owned verified updates, active-work deferral, macOS app archives, and a Zodex-style CLI installer.
- Usage-first documentation, exact manual client configuration, and clear desktop/remote boundaries.

## 0.1.1 · 2026-10-20

- Claude quota reads now invoke the user's installed `claude` executable directly using its streaming JSON `initialize` and `get_usage` control requests.
- Removed the Agent SDK/Node bridge, `noria bridge setup`, its extra installation step, and Node/npm checks from native runtime prerequisites.
- Kept isolated per-account Claude authentication, no-prompt quota readings, fail-closed schema validation, and bounded process lifetimes. The control protocol remains experimental.
- Node.js remains part of the optional npm-based installer/shim, not the standalone Go binary or Claude quota reader.

## 0.1.0 · 2026-10-20

Initial Noria release.

- Local Go CLI and loopback gateway for explicitly authorized native Codex and Claude account pools.
- Account registration, isolated native login profiles, enable/disable/remove controls, quota refresh, status, doctor, and native client wrappers.
- Hard session affinity with persisted bindings; reset-aware initial and failover placement considers independent five-hour/weekly windows, scoped limits, learned window-specific consumption, in-flight reservations, and recent bound-session demand.
- Native HTTP/SSE forwarding and Codex WebSocket handling, with bounded safe retries before output is committed. Requests retain their model, settings, and continuation data.
- Token refresh is serialized per isolated account; valid tokens and failover are not held up by passive quota reads.
- Random local gateway authentication, Host/Origin checks, private state, and credential-free status diagnostics.
- Embedded Claude usage bridge with explicit installation of pinned `@anthropic-ai/claude-agent-sdk@0.3.287`; Node is required and the usage method is experimental.
- Native binary distribution through GitHub Releases and the `@amxv/noria` npm wrapper. Apache-2.0 license.

### Known boundaries

Installed Codex 0.162.1 and Claude Code 2.1.299 passed local compatibility tests for a first-request quota failure and for resuming an existing conversation after an account limit. The tests verify preserved conversation history and use fake credentials and controlled upstreams. Live subscription accounts have not yet been validated. Partially delivered generations are not transparently spliced. Caches remain account-local and warm naturally after a forced switch.

When every matching account is exhausted, Noria reports unavailability. It does not buy credits, redeem quota resets, enable paid fallback, or switch models/providers. See [troubleshooting](/docs/troubleshooting#verified-locally) for compatibility context.
