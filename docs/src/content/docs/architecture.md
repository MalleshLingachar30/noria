---
title: Architecture and limits
description: Reference for native protocols, quota telemetry, service lifetime, and recovery boundaries.
order: 12
category: Reference
summary: Reference for native protocols, quota telemetry, service lifetime, and recovery boundaries.
---

Noria is a Go daemon and CLI. Its Swift menu bar is a native presentation layer over CLI JSON and commands. No Noria cloud service or external database is in the request path.

## Provider controls

Codex quota reads use the installed `codex app-server` in the account's isolated `CODEX_HOME`. Claude quota reads invoke the installed `claude` binary with control-only streaming JSON: `initialize`, then `get_usage`. No Node SDK is needed and no user turn is submitted for telemetry.

The Claude control operation is experimental. Native versions can change authentication and control schemas. Unknown or stale data remains unknown; it is never converted into a full allowance. Backend quota rejections are authoritative.

## Scheduler

Accounts share the scheduling policy but not credentials or protocols. Admission checks every applicable limit. Initial placement and failover use bounded expiry/capacity estimates, recent bound-session demand, and atomic in-flight reservations. Per-window observations train consumption estimates; reset epochs are not mixed into one consumption sample.

Bindings survive restarts. Account aliases do not change binding keys. The scheduler does not move a healthy binding because a score changes.

## Native streaming

Codex Responses uses HTTP/SSE or WebSockets; Claude uses native Messages. Noria preserves model settings and event bodies. A rejected, unfulfilled request can be retried before output is delivered. Already-delivered text or tool output is not spliced into a new generation or silently replayed.

A provider may reject continuation state from a different account. Native-client mock tests verify ordinary resume/failover behavior, but do not prove every live provider feature or subscription combination.

## Service and updates

macOS uses user-scoped launchd. Linux uses systemd --user when available, otherwise a detached local supervisor. Windows uses a detached user supervisor and Task Scheduler for login startup. No administrator service or stored user password is required for the standard setup.

The updater stages checksum-verified release assets. A short admission gate is acquired only when no model request is active. In-flight work defers the update; a stalled updater's admission gate expires. Idle WebSocket connections may reconnect after a service update. The router's bindings remain in its private state.

Stop restores registered direct-client routing fields, leaving original native credentials untouched. It does not redirect a manually configured app or force an already-running app to reload its own settings.
