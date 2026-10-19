---
title: Connect your clients
description: Persistent settings for Codex and Claude Code, plus desktop and custom-profile guidance.
order: 4
category: Get started
summary: Persistent settings for Codex and Claude Code, plus desktop and custom-profile guidance.
---

Noria connects **client profiles**, not individual terminal launches. Set up the profile each app reads once, then use that app normally. Profiles with explicit higher-priority overrides, separate desktop configurations, or remote runtimes need their own setup.

## Automatic setup

```bash
noria integrate codex
noria integrate codex --yes
noria integrate claude
noria integrate claude --yes
```

The first command previews the settings path and proposed changes. `--yes` authorizes the change, keeps a backup, and starts Noria. The macOS menu offers the same action under **Configure Clients…**.

To use a custom home:

```bash
noria integrate codex --client-home "$HOME/.gg/codex" --yes
noria integrate claude --client-home "$HOME/.gg/claude" --yes
```

Defaults are `$CODEX_HOME/config.toml` or `~/.codex/config.toml`, and `$CLAUDE_CONFIG_DIR/settings.json` or `~/.claude/settings.json`. Noria does not copy conversations or move these homes.

Reopen existing clients after setup. Then run `codex` or `claude` without a Noria wrapper.

## Codex CLI and local desktop clients

For manual setup, add this to the **user-level** `config.toml`, merging it with existing settings. Replace the absolute executable/home paths with yours:

```toml
model_provider = "noria"

[model_providers.noria]
name = "Noria"
base_url = "http://127.0.0.1:43119/codex/v1"
wire_api = "responses"
supports_websockets = true

[model_providers.noria.auth]
command = "/Users/you/.local/bin/noria"
args = ["auth-token", "--home", "/Users/you/.noria"]
```

The command-backed auth helper reads Noria's local token at runtime. You do not need to export an environment variable into a GUI app or paste a token into TOML. Keep your existing model, reasoning, tools, and project settings.

Codex ignores provider-routing keys in project-local `.codex/config.toml`; use the user-level file. The Codex desktop app and other local hosts benefit when they use that same configuration home. Configure each distinct home used by a host, and check for host-supplied overrides.

[Official Codex configuration reference](https://developers.openai.com/codex/config-reference)

## Claude Code CLI and hosts that read its settings

Merge the following into the user-level `settings.json`, retaining other keys and environment entries:

```json
{
  "apiKeyHelper": "'/Users/you/.local/bin/noria' auth-token --home '/Users/you/.noria'",
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:43119/claude",
    "ENABLE_TOOL_SEARCH": "true"
  }
}
```

Use your actual paths. Noria's automatic integration generates platform-correct quoting, including Windows. It keeps the local gateway token out of the JSON file.

An inherited `ANTHROPIC_AUTH_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`, or cloud-provider override can take precedence. Clear conflicting overrides for this profile; **Health Check** reports common ones without printing their values.

For the VS Code extension, the extension's own authentication UI may also require routing/auth configuration through `claudeCode.environmentVariables` in VS Code settings. Its spawned Claude Code process and its pre-login check are separate surfaces. Follow the [official gateway setup](https://code.claude.com/docs/en/llm-gateway-connect#vs-code-extension) for the installed extension version.

## Claude Desktop

Claude Desktop's third-party inference configuration is separate from Claude Code's `settings.json`. It does **not** inherit `ANTHROPIC_BASE_URL` from that file.

In Claude Desktop, open **Help → Troubleshooting → Enable Developer Mode**, then **Developer → Configure Third-Party Inference…**. Select **Gateway** and set the base URL to:

```text
http://127.0.0.1:43119/claude
```

Use **Helper script** as the credential source, with the absolute Noria executable and arguments `auth-token`, `--home`, and your absolute Noria home. `noria integrate claude-desktop` prints the corresponding configuration keys without a secret:

```json
{
  "inferenceProvider": "gateway",
  "inferenceGatewayBaseUrl": "http://127.0.0.1:43119/claude",
  "inferenceGatewayAuthScheme": "bearer",
  "inferenceCredentialKind": "helper-script",
  "inferenceCredentialHelper": "/Users/you/.local/bin/noria",
  "inferenceCredentialHelperArgs": ["auth-token", "--home", "/Users/you/.noria"]
}
```

Choose models actually available to your subscription in the desktop form; Noria does not invent a model catalog. Apply the form and reopen the app. Managed organizational settings can make this form read-only and take precedence over local choices.

This app-specific setup is manual. Noria's Stop/Start restoration manages the Codex/Claude Code profiles registered through `noria integrate`, not Claude Desktop's configuration library. To use Claude Desktop directly after stopping Noria, switch its inference configuration back in the app. Cloud-hosted and remote sessions do not automatically reach your machine's localhost.

[Official desktop gateway guide](https://claude.com/docs/third-party/claude-desktop/gateway) · [Configuration reference](https://claude.com/docs/third-party/claude-desktop/configuration)

## Restore or inspect managed settings

```bash
noria integrations --json
noria integrate codex --remove --yes
noria integrate claude --remove --yes
```

Removing an integration restores only the fields Noria installed. Unrelated edits remain intact. If a Noria-owned field was edited externally, restoration stops rather than overwriting it; the health report identifies the affected file.

Prefer automatic integration when you want Stop/Start to manage routing. Manual snippets are not automatically registered for restoration.

## Optional one-off wrapper

`noria run codex --` and `noria run claude --` remain useful for a one-off test without persistent settings. They are not required for everyday use after integration.
