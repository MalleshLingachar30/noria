---
title: Troubleshooting
description: Diagnose local setup, quota reads, account availability, and interrupted native streams.
order: 8
category: Reference
summary: Practical checks and the current live-compatibility boundary.
---

## Start with local checks

```bash
noria --version
noria doctor
noria account list
noria status
```

`doctor` checks local prerequisites needed by your enabled account providers and private profile permissions. It does not validate a subscription, spend live model quota, or prove a real multi-account workflow. A Codex-only pool does not require the Claude SDK bridge.

Use the same `NORIA_HOME` in the daemon terminal and the client terminal. A different home means a different pool and local secret.

## The daemon is unavailable

Run `noria serve` and keep that terminal open. Check for a port conflict, then inspect the address printed at startup. You can select another numeric loopback port:

```bash
noria serve --listen 127.0.0.1:43120
```

Launch the client again with `noria run`. Manually exported environment values from an older daemon may still point to the old address. `status` can show saved state when the daemon is unavailable; saved metadata is not evidence that the gateway is running.

## An account is missing or disabled

```bash
noria account list
noria account enable work
noria quota refresh
```

Check provider and model eligibility. Claude accounts cannot serve Codex requests, and a model restriction cannot grant upstream access to a model. Account commands request immediate live reload. If that reload reports failure, resolve the control error or restart the daemon. `quota refresh` also reloads configured accounts and updates their telemetry.

## Login or authentication fails

Use `noria account login <id>` to open the native sign-in flow for the intended profile. Do not fix it by copying your normal auth files into another account’s directory. Check that the native executable is on `PATH` and that sign-in finishes as the intended account.

A login may need an interactive terminal and a browser. Complete this setup yourself. If it succeeds but the gateway still reports an auth blocker, run `noria quota refresh <id>` for a fresh account read. Do not attach credentials or raw native login output to an issue.

## Claude quota reads fail

```bash
node --version
npm --version
noria bridge setup
noria quota refresh claude-work
```

The embedded bridge needs Node and `@anthropic-ai/claude-agent-sdk@0.3.287` installed under the selected Noria home. The usage method is experimental; an upstream change, missing method, incompatible schema, unavailable rate-limit data, or failed account auth can prevent a snapshot.

Do not “repair” the bridge by installing an arbitrary latest SDK into the private directory. Use the explicit pinned setup and report a redacted compatibility error. Noria treats missing and partial telemetry as unknown, not as unlimited capacity.

## Every account is unavailable

Read each account’s applicable five-hour, weekly, and model-scoped limits. A five-hour reset does not clear an exhausted weekly window. Authentication and cooldown blockers must also clear.

If no matching account can serve, Noria reports unavailability. An earliest retry estimate is returned only when enough reset information is known. It may be later than the next individual clock because all blockers on an account have to clear. With no known reset, refresh quota or address authentication rather than retrying in a tight loop.

Noria does not purchase capacity or use paid fallback. Adding another profile for the same subscription does not create more allowance.

## A session does not move to the account with more quota

That is hard affinity working as intended. Scores matter at placement and failover. A healthy bound session is not moved merely because another account resets. After a necessary switch, it does not automatically return to the old account.

If a client omits recognizable session metadata, the gateway cannot promise continuity across its requests. Use the supported native wrapper and retain native conversation identifiers. See [native client integration](/docs/native-clients).

## A stream stops, or continuation is rejected

Noria cannot transparently splice an already delivered response across accounts. Review the native client’s error and conversation state before retrying, particularly when tools have run. Do not assume a second attempt can safely replay a tool action.

Continuations may depend on state held by the original upstream account. Noria preserves the original continuation data on failover; another account may reject it. That error is surfaced instead of rewriting your conversation into a new request.

## A lock remains after a crash

Noria uses local locks to protect shared configuration, native account operations, and daemon state. It does not silently steal a lock. Follow the error’s exact path and verify no Noria or account-control process still owns that operation before manually removing a stale lock. Never clear a live lock to launch a competing daemon on the same home.

## Compatibility and validation

v0.1.0 is an early release. Local automated tests use fake credentials, fake native processes, and HTTP/WebSocket test upstreams. **No actual live multi-account validation has been performed.** Transport tests do not prove that a current subscription backend accepts a cross-account continuation or every native client feature.

Native Responses HTTP/SSE, native Messages HTTP/SSE, and Codex WebSocket forwarding are implementation surfaces. Early quota retry is bounded by the point where output is committed. Native client versions, provider authentication behavior, beta headers, experimental Claude usage schemas, and continuation rules can change.

The installed Codex 0.162.1 and Claude Code 2.1.299 clients both passed a forced quota-failover check through Noria against local mock upstreams: Codex over WebSockets and Claude over HTTP/SSE. A second test completes a turn on account A, resumes the same saved conversation, rejects A's next request, and verifies account B receives the previous user and assistant turns. These checks used temporary profiles and fake credentials, not live subscription capacity. Node 24.22 was used for local validation. This is not a minimum-version guarantee. Complete your own explicit account login and controlled native session checks before relying on a pool. Noria does not run a live validation automatically.

## Report a reproducible issue

Include the operating system, Noria and native client versions, redacted command and error, whether the daemon was running, and whether the failure was before or after output began. See [security and privacy](/docs/security-and-limitations#report-a-problem-without-secrets) for what to omit.
