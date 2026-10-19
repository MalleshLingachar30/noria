---
title: Accounts & authentication
description: Create isolated native authentication profiles and manage only the accounts you explicitly authorize.
order: 2
category: Start
summary: Explicit sign-in, private account profiles, model eligibility, and quota readers.
---

Noria separates **upstream account authentication** from **your normal client workspace**. Account profiles belong under `NORIA_HOME`; the client you launch keeps its usual history and settings.

## Create a profile

```bash
noria account add work --provider codex
noria account login work
noria account list
```

Use `--provider claude` for a Claude Code account. Account IDs are local labels: 1–64 letters, digits, underscores, or hyphens, beginning with a letter or digit. Choose labels that will be safe to include in local status output.

`account add` creates a profile without copying credentials. `account login` invokes `codex login` with that account’s `CODEX_HOME`, or `claude auth login` with that account’s `CLAUDE_CONFIG_DIR`. Complete the native interactive flow as the intended account. Adding a second profile does not create a new provider subscription.

Do not copy your normal auth files into Noria as a setup shortcut. Native login manages its own credential storage. Claude Code can use a macOS Keychain entry associated with the configuration directory or credential files, depending on authentication method and platform. See [Claude’s authentication documentation](https://code.claude.com/docs/en/authentication).

## Restrict model eligibility

```bash
noria account add specialist --provider codex --model gpt-5.4
```

Repeat `--model` to allow several exact model names. This is a routing constraint, not a model substitution: requests retain their original model. Use model IDs available to your account; the example is not a promise of model access.

## Enable, disable, or remove

```bash
noria account disable work
noria account enable work
noria account remove work
```

Disable an account before maintenance to remove it from eligible placement. Account changes request an immediate reload in a running daemon. If the command reports that configuration was saved but live reload failed, stop/restart the daemon or resolve the control error before relying on the change. Disabling does not cancel work already accepted upstream.

Removing an account unregisters it but **retains its native profile credentials and history**. It does not securely erase credentials or sign you out. Re-adding the same ID may reuse the retained profile. It does not sign you out of unrelated native client profiles. See [security and privacy](/docs/security-and-limitations#removing-accounts-and-local-data) before deleting local data.

## Codex quota reader

Noria launches the installed `codex app-server` with each account’s isolated `CODEX_HOME`. It initializes the experimental API and reads `account/rateLimits/read`; the canonical Codex bucket is preferred over unrelated buckets.

Five-hour and weekly windows are recognized by duration, not assumed array order. Usage percentages mean **used**, while scheduling explanations may display **remaining**. Missing windows and resets remain unknown. Model-scoped limits are separate constraints. Noria does not consume quota reset credits to read usage.

The provider interface is described in the [Codex app-server documentation](https://developers.openai.com/codex/app-server).

## Claude quota bridge

```bash
noria bridge setup
noria quota refresh claude-work
```

The Go binary embeds a small JavaScript bridge. Node.js and the pinned `@anthropic-ai/claude-agent-sdk@0.3.287` dependency are also required. `bridge setup` performs the explicit installation under `NORIA_HOME/bridge`.

The bridge calls the SDK handle’s `usage_EXPERIMENTAL_MAY_CHANGE_DO_NOT_RELY_ON_THIS_API_YET()` method with a parked prompt iterator. No user turn is submitted for the usage read. This method is experimental and can change or be unavailable; Noria validates the returned snapshot and treats missing telemetry as unknown. It is not a stable, dependency-free Go API.

Both five-hour and seven-day utilization are read independently. Partial snapshots do not invent the missing window. If quota reads fail, run `noria doctor` and check [troubleshooting](/docs/troubleshooting).

## Use accounts you are authorized to use

Provider account access and usage remain subject to the relevant provider’s terms. Refer to [OpenAI’s terms](https://openai.com/policies/terms-of-use/) and [Anthropic’s consumer terms](https://www.anthropic.com/legal/consumer-terms) for the applicable account rules.
