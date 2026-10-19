---
title: Changelog
description: Release notes and validation boundaries for Noria.
order: 16
category: Reference
summary: Noria v0.2.0 adds one-time client setup, background service controls, and the native macOS menu bar.
---

## 0.2.0 · 2026-10-20

- One-time, previewable Codex/Claude Code integration with native credential helpers, private backups, and conflict-aware restoration. Ordinary client launches can use the pool without a wrapper.
- Nonblocking start, stop, and restart on macOS, Linux, and Windows. Stop restores registered direct-client settings while retaining the original native login. Removed the public foreground serve command.
- Native Swift macOS menu bar with account quotas and reset countdowns, asynchronous refresh on open, browser sign-in, optional aliases, per-account controls, and Health Check.
- Email-first account labels and immutable routing IDs. Relogin disables the entry until identity is confirmed; duplicate identities do not become duplicate capacity.
- Separate menu-bar and service-at-login preferences. Verified CLI/app updates defer around active requests and are controlled by the Go CLI.
- Native installers, ad-hoc-signed macOS app bundles, and focused everyday-use documentation.

The live provider/account boundary remains: automated native-client tests use fake upstreams and isolated accounts. App-specific Claude Desktop configuration is separate from the registered CLI settings.

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

When every matching account is exhausted, Noria reports unavailability. It does not buy credits, redeem quota resets, enable paid fallback, or switch models/providers. See [troubleshooting](/docs/troubleshooting#compatibility-and-validation) for compatibility context.
