---
title: Native client integration
description: Launch Codex and Claude through the loopback gateway while retaining your normal client workspace.
order: 3
category: Start
summary: Child-only routing configuration, native protocols, session identity, and transport boundaries.
---

Install and authenticate the native clients you use. Noria supplies account routing; it is not an alternative coding client or a protocol translator.

## Run your normal client

Start `noria serve` in one terminal. In another terminal:

```bash
noria run codex --
# Or, after Claude account login:
noria run claude --
```

Arguments after `--` are forwarded to the native executable. For example, `noria run codex -- --help` shows native Codex help. Use the native client’s own documentation for its available options.

The wrapper retains your ordinary `HOME`, `CODEX_HOME`, and `CLAUDE_CONFIG_DIR` choices and uses child-only routing overrides. Your existing conversations, skills, settings, and project context remain available. It does not persistently rewrite the configuration in those roots. This differs from `account login`, which intentionally uses an isolated **upstream auth profile** under Noria’s home.

## Codex

The gateway uses native Responses requests. The wrapper supplies command-local `-c` overrides for a native `noria` model provider at `http://127.0.0.1:43119/codex/v1`, authenticated with `NORIA_GATEWAY_TOKEN`. It does not write `config.toml`. The upstream account is selected by Noria and its credentials are substituted only for that upstream request.

Codex account quota reads use a separate `codex app-server` process for each isolated account profile. The process running your normal conversation is not used to switch the account credentials of another client.

Noria includes HTTP/SSE forwarding and Codex WebSocket handling. WebSocket account selection and early quota handling have different lifecycle constraints from HTTP. Do not infer end-to-end compatibility with a particular native client version from a fake-upstream transport test. See [compatibility and validation](/docs/troubleshooting#compatibility-and-validation) for the release boundary.

## Claude Code

Claude uses native Messages requests against `http://127.0.0.1:43119/claude` by default. The wrapper injects the local gateway authentication for the child and enables native tool search with `ENABLE_TOOL_SEARCH=true`.

Provider headers, beta flags, model, reasoning, tool messages, and streaming events remain part of the original protocol. The gateway does not translate Claude requests into Codex requests or fall back to an API billing account.

Claude quota reads are separate from request forwarding. They launch the same installed native `claude` binary with its isolated upstream account profile and use the experimental `get_usage` control operation, with no Node.js or Agent SDK bridge. Read [accounts and authentication](/docs/accounts#claude-native-quota-reader).

## Session identity

Affinity depends on a session identity the gateway can recognize in native request metadata. Keep native session and conversation identifiers intact. Independent sessions and identifiable branches can receive independent placements. Anonymous requests are not treated as a single shared conversation and cannot be promised persistent affinity across requests.

An upstream continuation may refer to state tied to its original account. Noria preserves that continuation unchanged on a safe retry; it does not invent missing state on the next account. If the upstream rejects a cross-account continuation, the error remains visible. This is one reason live account compatibility must be checked before relying on automatic failover.

## Inspect environment output deliberately

```bash
noria env codex
noria env claude
```

`env` prints shell setup for intentional manual integration, including the **local gateway secret**. Codex output also defines a shell function carrying its command-local provider overrides; an environment variable alone does not install that provider configuration. Claude output sets `ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, and `ENABLE_TOOL_SEARCH=true`.

Do not paste this output into issues, chat, recordings, or shared logs. Prefer `noria run` for ordinary use so routing overrides stay in the child process. Shell output is intended for POSIX shells; use the wrapper when a shell cannot apply it directly. The wrapper removes inherited direct-provider credential/route variables that could bypass Noria while retaining client config roots.

Stopping the daemon makes its local route unavailable. Exit the wrapped client when finished or restart the gateway using the same Noria home. Noria does not silently redirect around itself or select a paid fallback.
