---
title: Install & quickstart
description: Install Noria, sign in to isolated accounts, and route your first native coding session.
order: 1
category: Start
summary: Install, add accounts, start the local gateway, and run a native client.
---

Noria runs on your machine between your native coding clients and the accounts you sign in. A session stays with its assigned account until an applicable limit makes that account unavailable.

## 1. Install

```bash
npm install -g @amxv/noria
noria --version
noria --help
```

The npm installer downloads the native binary for your platform from [GitHub Releases](https://github.com/amxv/noria/releases). macOS, Linux, and Windows each have amd64 and arm64 builds. If the download fails, installation can fall back to a local Go build when Go is available. You can also download the matching binary directly from a release.

Install the native `codex` or `claude` executable for the provider you want to use. Noria does not replace either client. Claude quota reads also need Node.js, npm, and the explicit SDK setup below.

Node is also needed to invoke the npm shim. A directly downloaded Go binary can run Codex routing without Node; the Claude quota bridge still needs it. The development environment uses Node 24.22, not a promised minimum-version matrix. Existing provider subscriptions and explicit authorization to use each account are prerequisites; Noria does not create them.

## 2. Create your local configuration

```bash
noria init
```

The default home is `~/.noria`; set `NORIA_HOME` before running Noria to use another location. Initialization creates private local configuration and a random gateway secret. The default gateway address is `http://127.0.0.1:43119`.

## 3. Add and sign in to your accounts

For Codex:

```bash
noria account add codex-work --provider codex
noria account login codex-work
noria account add codex-personal --provider codex
noria account login codex-personal
```

For Claude Code:

```bash
noria bridge setup
noria account add claude-work --provider claude
noria account login claude-work
```

Each login opens the provider’s native interactive sign-in flow for an isolated Noria account profile. **You complete the sign-in yourself.** Noria does not import your normal client credentials. One account is enough to start; automatic failover needs another usable account for the same provider and requested model.

`noria bridge setup` explicitly installs `@anthropic-ai/claude-agent-sdk@0.3.287` under Noria’s home for Claude quota reads. The bridge uses an experimental SDK method. It does not install dependencies while handling requests. See [accounts and authentication](/docs/accounts).

## 4. Check availability

```bash
noria doctor
noria account list
noria quota refresh
```

Quota refresh reads provider telemetry. It does not send a dummy prompt, redeem reset credits, or turn unknown quota into a full allowance. A failed or incomplete read needs attention; see [troubleshooting](/docs/troubleshooting).

## 5. Start the gateway

Keep this terminal open:

```bash
noria serve
```

In another terminal, use the same `NORIA_HOME` and start a client:

```bash
noria run codex --
```

Or:

```bash
noria run claude --
```

The wrapper supplies routing configuration to the child process. Your existing client home, conversation history, skills, and settings remain in place. It does not persistently rewrite your normal client configuration. See [native client integration](/docs/native-clients) for transport details and manual setup.

## 6. Inspect the route

```bash
noria status
noria status --json
```

Status is local operational metadata, not a transcript viewer. A healthy session keeps its account even when another account resets or has more headroom. After a necessary switch, the new account’s cache warms naturally.

Read [how scheduling works](/docs/scheduling) and [security and privacy](/docs/security-and-limitations) before relying on a multi-account pool. Noria preserves the requested model and does not enable a paid fallback. When every matching account is unavailable, it reports that honestly. **Actual live multi-account validation has not yet been performed**; see [compatibility](/docs/troubleshooting#compatibility-and-validation).
