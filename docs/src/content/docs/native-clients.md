---
title: Connect your clients
description: Make Codex and Claude Code use Noria by default, wherever you usually launch them.
order: 3
category: Get started
---

Noria works best when your existing coding clients use it automatically. You don't need to start each session with a special wrapper. On macOS, the **N → Configure** submenu offers **Configure Codex** and **Configure Claude Code**.

## Codex

```sh
noria integrate codex --yes
```

This configures the Codex `config.toml` read by your client. Codex CLI, the Codex desktop app, and other clients using that same configuration root can then send requests through Noria.

## Claude Code

```sh
noria integrate claude --yes
```

This configures Claude Code's `settings.json` to use Noria's local base URL and credential helper. It applies wherever that settings file is used.

*Experimental:* Claude Code smart routing may have bugs while OAuth token refresh and proxy compatibility are being improved.

For both providers, **restart clients that were already open** after the change.

## Different configuration folders

Noria normally detects `$CODEX_HOME` for Codex and `$CLAUDE_CONFIG_DIR` for Claude Code, falling back to the clients' default folders.

If your desktop app or editor uses a different folder, select **N → Configure → Choose a Custom Client Folder…**, or tell Noria explicitly:

```sh
noria integrate codex --client-home "$HOME/.gg/codex" --yes
noria integrate claude --client-home "$HOME/.gg/claude" --yes
```

`noria integrations --json` lists the folders Noria manages. An editor with its own base-URL override may need configuration inside that editor too.

## Manual settings (advanced)

The `integrate` command is the easier, reversible option. If you want to configure a client yourself, use these settings and substitute your actual absolute paths.

**Codex** — in `config.toml`:

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
```

Keep `model_provider` before any TOML tables. Preserve your other Codex settings. See the [Codex configuration reference](https://developers.openai.com/codex/config-reference).

**Claude Code** — add these values to `settings.json`:

```json
{
  "apiKeyHelper": "'/absolute/path/to/noria' auth-token --home '/absolute/path/to/.noria'",
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:43119/claude",
    "ENABLE_TOOL_SEARCH": "true",
    "ANTHROPIC_AUTH_TOKEN": "",
    "ANTHROPIC_API_KEY": "",
    "CLAUDE_CODE_OAUTH_TOKEN": ""
  }
}
```

Merge these fields into the existing file rather than replacing it. The CLI handles platform-specific quoting and backs up any fields it manages, so prefer that unless you have a particular reason to edit by hand.

## What about Claude Desktop?

Claude Desktop has its **own third-party inference configuration**, separate from Claude Code's `settings.json`. It won't automatically pick up Claude Code's integration.

Use `noria integrate claude-desktop --json` for the local helper path and gateway details, then configure **Developer → Configure Third-Party Inference…** in Claude Desktop. The [Anthropic gateway guide](https://claude.com/docs/third-party/claude-desktop/gateway) explains the setting. Local-only routing isn't available to a remote or hosted client that can't reach your machine.

## Go back to direct accounts

`noria stop` restores the routing settings Noria installed. Your previous direct account sign-in remains intact. If you've configured an app manually (including Claude Desktop), switch it back through that app's own settings.

For a permanent removal from one client root:

```sh
noria integrate codex --remove --yes
noria integrate claude --remove --yes
```
