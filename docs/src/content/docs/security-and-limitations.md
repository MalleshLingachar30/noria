---
title: Security and limitations
description: What stays local, what goes upstream, and what routing cannot guarantee.
order: 6
category: How it works
---

## What stays on your machine?

Noria's account pool, routing decisions, usage snapshots, and session bindings live locally. Each account uses its own sign-in profile. The gateway listens only on your computer's loopback interface and requires a local credential.

Your model requests **still go to the selected provider** (OpenAI or Anthropic). Noria is a proxy, not an offline model. Your usage remains subject to the provider's terms and account limits.

Noria doesn't change your normal client login to add accounts to the pool. If you stop Noria, it restores client settings that it installed so you can use the original direct-provider account again. Apps configured manually must be switched back manually.

## What Noria can—and cannot—do

- **Can:** Choose an account for a new conversation based on usable capacity, reset times, and active work.
- **Can:** Keep a conversation pinned to its account so prompt caching is preserved as long as possible.
- **Can:** Retry an unfulfilled request on another eligible account when a provider limit blocks it.
- **Cannot:** Create additional usage allowance, purchase extra credits, or make an exhausted pool available.
- **Cannot:** Switch between Codex and Claude providers or substitute a different model to hide exhaustion.
- **Cannot:** Move prompt caches between accounts, replay completed tool actions, or transparently stitch together partially streamed output.

## Supported clients

Codex and Claude Code use native protocols, which can evolve over time. Locally tested versions have passed ordinary session and failover fixtures, but live subscription behavior and individual desktop hosts may vary.

Claude Code smart routing is currently **experimental**. OAuth refresh and proxy compatibility are still being worked on, and some sessions may fail.

Some applications have their own provider configuration or run remotely. These need a reachable local Noria endpoint and may not be configurable with the same settings as a local terminal client.

## Credentials and access

Account credentials are separate from your existing client profiles. Keep your Noria home private and don't publish credential-helper output, expose its local gateway publicly, or share login files. Software already running as your OS user may be able to access local processes and files.

An account removed from Noria is unregistered from routing, but its local sign-in files may remain for recovery. Removing it is not the same as logging out of the provider everywhere.
