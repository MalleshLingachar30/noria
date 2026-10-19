---
title: "Fix a setup problem"
description: "Practical checks for routing, sign-in, stale usage, and service startup."
order: 8
category: "Start"
summary: "Practical checks for routing, sign-in, stale usage, and service startup."
---

## Start with Health Check

Choose **N → Health Check**, or:

```sh
noria health
noria status
```

This checks local reachability, managed routing configuration, and native executables. It sends no model prompt and does not prove live subscription acceptance.

## My client is not using Noria

Inspect `noria integrations --json` and configure the root actually read by that client. Reopen it after setup. A project, editor, managed policy, or desktop host may override the provider. See [client setup](/docs/native-clients) for exact values and surface differences.

## Stop did not change an already-running client

The process may cache its settings; reopen it. Stop restores registered file values, not a running process's memory. Manually pasted snippets and Claude Desktop's separate third-party form are not registered for restoration.

Direct use relies on the normal client's own saved login. If that account is also exhausted, direct use still has its own limit. Noria does not overwrite it with the last-used pool account.

## Sign-in did not complete

Choose **Sign In Again…** and select the intended account in the browser. Closing the browser, cancelling, or timing out leaves a pending new account disabled. `noria account relogin <email-or-id>` uses the same flow. Do not copy ordinary auth files into another pool profile as a shortcut.

## Usage is unknown or stale

Run `noria quota refresh`, or open the menu and use Refresh Usage. Cached values appear immediately while new values load. Unknown is not unlimited capacity and does not clear a confirmed rejection.

Check native client versions and `noria doctor`. Claude needs no SDK/Node bridge installation. Provider control interfaces can change with native releases.

## Startup or updates need attention

Check for a port conflict on `127.0.0.1:43119`, the private `logs/service.log`, and `noria health`. Linux user systemd may be unavailable: manual Start can use the supervisor, while login autostart needs a user service manager.

Updates wait for active requests and streaming connections to finish. Close idle client connections before retrying an update that remains deferred. Checksums and app signatures are verified before replacement; never disable OS security or strip quarantine to force an install.

## Verified locally

Installed Codex 0.162.1 and Claude Code 2.1.299 passed local forced-failover and conversation-resume tests. Both also passed config-only launches using persistent settings and the Noria credential helper, without wrapper/provider environment overrides. Background Start/Stop was exercised with isolated client settings and no accounts. These are not live subscription-account tests or certification of every desktop host.

Report Noria/native-client versions and a redacted error, not auth files, helper output, or raw conversations.
