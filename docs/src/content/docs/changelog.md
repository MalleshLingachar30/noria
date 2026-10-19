---
title: Changelog
description: Release notes and validation boundaries for Noria.
order: 11
category: Development
summary: The initial v0.1.0 release, dated October 20, 2026.
---

## 0.1.0 · 2026-10-20

Initial Noria release.

- Local Go CLI and loopback gateway for explicitly authorized native Codex and Claude account pools.
- Account registration, isolated native login profiles, enable/disable/remove controls, quota refresh, status, doctor, and native client wrappers.
- Hard session affinity with persisted bindings; reset-aware initial and failover placement considers independent five-hour/weekly windows, scoped limits, learned consumption, and in-flight reservations.
- Native HTTP/SSE forwarding and Codex WebSocket handling, with bounded safe retries before output is committed. Requests retain their model, settings, and continuation data.
- Random local gateway authentication, Host/Origin checks, private state, and credential-free status diagnostics.
- Embedded Claude usage bridge with explicit installation of pinned `@anthropic-ai/claude-agent-sdk@0.3.287`; Node is required and the usage method is experimental.
- Native binary distribution through GitHub Releases and the `@amxv/noria` npm wrapper. Apache-2.0 license.

### Known boundaries

Installed Codex 0.162.1 and Claude Code 2.1.299 passed local compatibility tests for a first-request quota failure and for resuming an existing conversation after an account limit. The tests verify preserved conversation history and use fake credentials and controlled upstreams. Live subscription accounts have not yet been validated. Partially delivered generations are not transparently spliced. Caches remain account-local and warm naturally after a forced switch.

When every matching account is exhausted, Noria reports unavailability. It does not buy credits, redeem quota resets, enable paid fallback, or switch models/providers. See [troubleshooting](/docs/troubleshooting#compatibility-and-validation) for compatibility context.
