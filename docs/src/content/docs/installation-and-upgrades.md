---
title: "Native installation and upgrades"
description: "Install Noria without Node, verify binaries, and upgrade from the CLI or menu."
order: 10
category: "Reference"
summary: "Native installers, ad-hoc macOS signing, automatic updates, and release verification."
---

## Install without a package manager

**macOS (CLI and menu bar) and Linux (CLI):**

```sh
curl -fsSL https://noria.ashray.xyz/install.sh | sh
```

The installer downloads the platform archive and SHA-256 manifest, verifies the archive and ad-hoc app signature, and installs `noria` with its sibling `Noria.app` under `~/.local/bin`. The menu bar has no Dock icon. No administrator privileges are needed.

**Windows (PowerShell, CLI):**

```powershell
irm https://noria.ashray.xyz/install.ps1 | iex
```

The Windows installer verifies the exact GitHub release checksum and installs into your user profile without asking for administrator privileges. It adds the binary directory to the user PATH. All three installers run `noria init`, preserving existing configuration.

This is Zodex-style CLI distribution: curl and extraction do not introduce the browser-download quarantine path. The installer does not disable Gatekeeper, remove quarantine flags, or claim notarization. Browser-downloaded bundles and organizational policies may produce additional checks.

Use **`noria upgrade`** for an existing installation so replacement is coordinated with active work. Install scripts do not overwrite an existing executable; they direct you to the upgrader.

## Native GitHub release assets

Releases contain six OS/architecture binaries, two macOS app archives, licenses, and `SHA256SUMS`. Both the runtime and Claude quota reader are native binaries: **no Node, npm, or JavaScript install shim is needed**. Windows and Linux have the same CLI account/router/background commands; the native menu bar requires macOS 13 or later.

Choose the matching archive/installer from [GitHub Releases](https://github.com/amxv/noria/releases). The shell and PowerShell scripts use GitHub's published checksum manifest, not third-party mirrors.

## Upgrade manually

```sh
noria upgrade --check
noria upgrade
```

From the macOS menu bar choose **Check for Updates…** and confirm an available upgrade. Both entry points call the same Go CLI updater; Swift does not download or install binaries itself.

## Automatic upgrades

The background service and menu periodically invoke `noria upgrade` when **Automatically Update** is enabled. The Go CLI owns the preference, download, verification, deferral, replacement, and service restart. The CLI works even when the menu bar is closed.

Open requests defer replacement; idle streaming connections reconnect after a safe service update. Failed verification leaves the installed release unchanged. Replacement retains rollback information so a failed new service can return to the prior release.

## Maintainers

Update `VERSION` and the changelog, run Go/race/native-client tests, verify both macOS app architectures, and check/build docs serially. Tags trigger CI, six native binaries, two macOS archives, checksums, and direct native-install tests. Verify results rather than equating a tag push with release completion.

Docs auto-deploy from `main` using the Vercel project's `docs` root. Do not include private runtime data in a package or deployment.
