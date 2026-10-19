---
title: Security & privacy
description: Local authentication, isolated upstream profiles, data handling, and the limits of account failover.
order: 7
category: Reference
summary: Know which credentials Noria manages and what a local gateway can promise.
---

Noria is a local gateway for accounts you explicitly authorize. Your model requests still travel to the relevant provider. “Local” describes the router and its state, not local model inference.

## Local access boundary

The listener is restricted to numeric loopback addresses, with `127.0.0.1:43119` as the default. A random local secret protects the gateway and its control operations. Host and Origin validation reject unexpected browser origins and rebinding-style requests.

These controls do not protect against every program running as your own operating-system user. A process that can read your private files or inspect your child process environment can potentially access the gateway or account credentials. Keep the Noria home private; do not publish it, expose the port through a public tunnel, or mount the home into untrusted containers.

## Upstream account isolation

`account add` creates an isolated profile; `account login` invokes the native provider’s explicit sign-in flow for that profile. Noria does not implicitly copy credentials from your usual Codex or Claude roots.

Each account has a separate auth/control context under `NORIA_HOME/accounts/<id>`. Noria explicitly selects file-backed storage for its isolated Codex logins; Claude uses its native platform credential store. Native clients manage token refresh. On macOS and Linux, Noria writes private directories and files with modes 0700 and 0600. On Windows, keep the home inside your protected user profile: Windows ACL inheritance, not POSIX mode bits, controls access. Claude’s macOS Keychain entries can be associated with a configuration directory. Removing a directory is therefore not necessarily a complete provider sign-out.

Normal `noria run` sessions retain your existing client histories, config roots, skills, and settings. Child-only routing overrides direct requests to Noria; they do not persistently edit those settings. Upstream authentication profiles and the client’s conversation workspace have different purposes.

## What is stored

Noria persists account configuration, hashed session-binding keys, quota snapshots, reset predictions, learned consumption estimates, and cooldowns. This metadata supports affinity after restart. It is still private operational information and can reveal account labels, usage patterns, and local paths.

Noria does not intentionally log request content, upstream tokens, or the gateway secret. Status and doctor output omit credential values. The native clients themselves may save their usual conversations and histories; their data handling remains under their configuration.

`noria env codex|claude` intentionally prints local gateway authentication so a user can configure a client. Treat that output as a secret. Do not attach it to support reports. Prefer `noria run` for routine launches.

## What leaves the machine

Requests go from the local gateway to the selected provider’s native service. Quota readers also contact provider services through native account tooling. There is no Noria-hosted service, dashboard, Redis, or third-party routing service in that request path.

Installing Noria or running `noria bridge setup` performs dependency or binary downloads. The Claude SDK is installed only by the explicit setup command, not on the request path. The documentation website is a separate static site.

## Limits that matter

- **No extra quota:** every matching account can be exhausted. Noria returns unavailability and reset information when known.
- **No paid escape hatch:** Noria does not enable paid fallback, buy credits, redeem quota reset credits, or switch models or providers to continue.
- **No transcript migration:** failover reuses a pending native request. Cross-account continuation state may be rejected by the provider.
- **No stream splicing:** a partially delivered generation cannot be transparently stitched to another account’s generation. Tool output is not silently replayed.
- **No cache transfer:** a forced switch starts cold on the replacement account. Warm-cache reuse depends on the provider and normal work.
- **No uptime or capacity guarantee:** reset-aware placement is a bounded heuristic based on incomplete, changing telemetry.
- **No live validation claim:** automated local tests use fake credentials and upstreams. Actual live multi-account workflows have not yet been validated.

Read [routing](/docs/scheduling) and [compatibility](/docs/troubleshooting#compatibility-and-validation) before using a pool for work where an interruption is costly.

## Removing accounts and local data

`noria account remove <id>` unregisters the account but intentionally retains its profile credentials and native history. It is not a secure erase or provider logout. Re-adding the same ID may reuse the retained profile.

For complete cleanup, stop the daemon and wrapped clients first. Review the exact profile and any platform credential-store entries through the provider’s native tools. Remove only the Noria data you intend to discard. Clearing local scheduler state does not clear an upstream limit or revoke provider credentials.

## Report a problem without secrets

Include the Noria version, operating system, native client versions, command names, and a redacted error. Account labels and local paths can also be private. Do not include `secret`, auth files, Keychain exports, `noria env` output, raw request bodies, transcripts, or full private-home archives in a public issue.
