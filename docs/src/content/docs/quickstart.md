---
title: Get started
description: Install Noria, add your accounts, and route your usual coding tools through it.
order: 1
category: Get started
---

You only need to set up Noria once. After that, keep launching Codex and Claude Code as usual—Noria takes care of choosing accounts and switching when limits are reached.

## 1. Install Noria

**macOS and Linux**

```sh
curl -fsSL https://noria.ashray.xyz/install.sh | sh
```

**Windows (PowerShell)**

```powershell
irm https://noria.ashray.xyz/install.ps1 | iex
```

On macOS, you can manage Noria from the **N** icon in your menu bar. On every platform, you can use the `noria` command.

## 2. Add your accounts

On macOS, select **N → Configure → Add Codex Account…** or **Add Claude Account…**. Otherwise, use:

```sh
noria account add codex
noria account add claude
```

Each command opens the provider's sign-in flow in your browser. Repeat it for every account you want in the pool—including multiple accounts for the same provider.

Noria shows each account's email address and remaining allowance. You can [rename, disable, or relogin to any account](/docs/accounts) later.

## 3. Connect your clients

Start Noria, then configure the tools you already use:

```sh
noria start
noria integrate codex --yes
noria integrate claude --yes
```

These commands connect the **default Codex and Claude Code configuration folders**. They don't replace your saved conversations or normal provider login.

If your tools use another config directory, choose it in **N → Configure → Choose a Custom Client Folder…** or pass `--client-home`:

```sh
noria integrate codex --client-home "$HOME/.gg/codex" --yes
noria integrate claude --client-home "$HOME/.gg/claude" --yes
```

See [Connect your clients](/docs/native-clients) for desktop apps, editors, and other configuration roots.

## 4. Keep coding normally

Restart Codex or Claude Code if it was already open. Then use it the way you always do:

```sh
codex
# or
claude
```

There is no special launch command. Noria runs in the background, chooses an account for each new session, and keeps that conversation on its chosen account until a limit makes switching necessary.

On macOS, open the **N** menu any time to see usage and reset times, refresh the pool, or stop routing.

**Want it to start automatically?** Turn on **Start Noria at Login** in the menu, or run `noria preferences start-at-login on`.

Next: [Learn how routing works](/docs/architecture), or [manage Noria day to day](/docs/everyday-use).
