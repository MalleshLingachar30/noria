---
title: "Connect your normal clients"
description: "Persistent configuration for Codex and Claude Code, plus desktop-specific setup."
order: 3
category: "Start"
summary: "Persistent configuration for Codex and Claude Code, plus desktop-specific setup."
---

Use **Configure Clients…** or `noria integrate <provider> --yes` for reversible setup. Each client must read the configured root; a host with a separate root or explicit provider override needs its own setup. Reopen already-running clients after a change.

## Codex CLI and desktop

```sh
noria integrate codex
noria integrate codex --yes
```

The target is `$CODEX_HOME/config.toml`, defaulting to `~/.codex/config.toml`. Codex desktop and compatible local hosts reading the same root receive the provider settings too.

For manual configuration, replace the two absolute paths below. Put `model_provider` at the file's root, before any TOML tables, and preserve existing model and other settings:

```toml
model_provider = "noria"

[model_providers.noria]
name = "Noria"
base_url = "http://127.0.0.1:43119/codex/v1"
wire_api = "responses"
supports_websockets = true

[model_providers.noria.auth]
command = "/absolute/path/to/noria"
args = ["auth-token", "--home", "/absolute/path/to/.noria"]
timeout_ms = 5000
refresh_interval_ms = 300000
```

The command helper supplies Noria's local credential without a shell export or provider token in the configuration. See the [official Codex configuration reference](https://developers.openai.com/codex/config-reference).

## Claude Code

```sh
noria integrate claude
noria integrate claude --yes
```

The target is `$CLAUDE_CONFIG_DIR/settings.json`, defaulting to `~/.claude/settings.json`. The equivalent manual fields are:

```json
{
  "apiKeyHelper": "'/absolute/path/to/noria' auth-token --home '/absolute/path/to/.noria'",
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:43119/claude",
    "ANTHROPIC_AUTH_TOKEN": "",
    "ANTHROPIC_API_KEY": "",
    "CLAUDE_CODE_OAUTH_TOKEN": "",
    "ENABLE_TOOL_SEARCH": "true"
  }
}
```

Merge them into existing JSON, rather than replacing the file. Empty direct-credential overrides keep inherited tokens from bypassing the helper; the integration backs up their original values. On Windows, let the CLI generate correct executable quoting and JSON path escaping.

See [Anthropic's gateway setup guide](https://code.claude.com/docs/en/llm-gateway-connect) for settings precedence and supported client surfaces.

## Custom roots and editors

```sh
noria integrate codex --client-home "$HOME/.gg/codex" --yes
noria integrate claude --client-home "$HOME/.gg/claude" --yes
```

The menu offers detected roots and **Choose a Custom Client Folder…**. An editor that overrides the base URL, auth, or model provider must use the equivalent settings in its own integration.

Claude Code in VS Code has an additional pre-launch check using VS Code's `claudeCode.environmentVariables`; user `settings.json` reaches the child process but may not satisfy that extension check. Consult the [VS Code section of the official gateway guide](https://code.claude.com/docs/en/llm-gateway-connect#vs-code-extension) when it still asks for login. Do not store the local gateway secret in a shared workspace file.

## Claude Desktop has separate configuration

Claude Desktop's third-party inference settings are not Claude Code's `settings.json`. Use its **Developer → Configure Third-Party Inference…** window after enabling Developer Mode under **Help → Troubleshooting**.

Choose Gateway, set the base URL to `http://127.0.0.1:43119/claude`, choose a helper-script credential, and use Noria's absolute executable with arguments `auth-token`, `--home`, and the absolute Noria home. Choose Bearer authentication. If model discovery is unavailable, configure model IDs actually available to the pool; Noria does not substitute models.

Run `noria integrate claude-desktop --json` to print the exact local paths and values, without editing the app.

The corresponding configuration keys are `inferenceProvider`, `inferenceGatewayBaseUrl`, `inferenceCredentialKind`, `inferenceCredentialHelper`, `inferenceCredentialHelperArgs`, and `inferenceGatewayAuthScheme`. Use the [official configuration window guide](https://claude.com/docs/third-party/claude-desktop/in-app-configuration), [gateway guide](https://claude.com/docs/third-party/claude-desktop/gateway), and [helper reference](https://claude.com/docs/third-party/claude-desktop/credential-helper) for app-version and managed-policy requirements. This route has not been live-account validated by Noria.

**Noria does not edit that separate Desktop form, so Stop does not undo it.** Return Desktop to its standard Anthropic sign-in configuration for direct use. Cloud/remote sessions cannot use this computer's loopback endpoint, and managed policy may prevent overrides.

## Restore direct use

`noria stop` restores routing fields for profiles registered through the menu or `integrate`. The normal client's saved login remains intact throughout; an already-running process may need reopening.

Manually pasted snippets are not registered for restoration. Prefer `integrate` and inspect `noria integrations --json` to see managed files. The optional `noria run codex --` wrapper remains available for one-off tests, not ordinary setup.
