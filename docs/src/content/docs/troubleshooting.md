---
title: Troubleshooting
description: Fix client routing, sign-ins, missing quota information, and background-service issues.
order: 9
category: Reference
---

## Start with a health check

```sh
noria health
noria status
```

`health` checks the local service, configured clients, and native executables. It doesn't send a model prompt.

## Codex or Claude Code isn't using Noria

Run `noria integrations --json` to see which configuration folders are connected. Make sure your client is using one of those folders. On macOS, choose **N → Configure → Configure Codex** or **Configure Claude Code** to set them up.

If you use a custom `CODEX_HOME` or `CLAUDE_CONFIG_DIR`, configure that path explicitly with `noria integrate ... --client-home`. Then **restart the client** so it picks up the settings.

Editors and desktop apps sometimes have their own base-URL or account settings. See [Connect your clients](/docs/native-clients).

## Configure Codex failed

Upgrade to **Noria 0.3.1 or later** if you see a panic involving `reflect.Value.SetLen` or TOML arrays such as `[[skills.config]]`.

```sh
noria upgrade
noria integrate codex
noria integrate codex --yes
```

Preview the changes first. Noria keeps unrelated Codex settings in place and restores your original direct `model_provider` line when you disable routing.

## A sign-in didn't finish

Choose **Sign In Again…** from the account submenu, or run:

```sh
noria account list
noria account relogin you@example.com
```

Select the intended account in your browser. Cancelled or incomplete sign-ins aren't eligible for routing.

## Usage is missing or looks stale

```sh
noria quota refresh
noria status
```

Each account has its own independent five-hour and weekly windows. If the provider doesn't return usage information, Noria labels it unknown rather than assuming there's free allowance. A real upstream quota rejection takes priority over old readings.

For Claude Code, Noria first checks the installed `claude` binary using its passive native control protocol. Some authenticated versions return `rate_limits: null` despite an active subscription; Noria then reads the provider's usage endpoint with the account's existing isolated native OAuth credential. No Node runtime, external SDK, model prompt, or new sign-in is needed. If both sources fail, the quota remains unknown rather than displaying guessed values.

## Stop didn't switch my client back

An already-open client may have cached its settings. Close and reopen it. `noria stop` restores only client settings that Noria installed; applications manually configured to use its gateway need to be changed manually.

## The service isn't running

Run `noria start`, then `noria health`. If startup fails, another process may be using the gateway address (`127.0.0.1:43119`). Local diagnostic messages are under Noria's private `logs/service.log` directory.

## No accounts are available

Noria can't route around an exhausted pool. It waits for an applicable reset or a usable account to be added. A weekly limit can block an account even if its five-hour window still has capacity.

If you need help, share the Noria and native-client versions plus a redacted error. **Never share account credentials, token-helper output, or conversation contents.**

## Linux tray icon is missing

The Linux tray needs a graphical session with a StatusNotifierItem/AppIndicator host. Check that your desktop supports AppIndicators (GNOME Shell often needs an extension), then run `noria tray open`. Install `zenity` or `kdialog` for the tray's native confirmation and folder dialogs. The CLI and router still work normally on headless machines.
