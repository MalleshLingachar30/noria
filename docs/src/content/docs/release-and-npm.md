---
title: "Install and update reference"
description: "Verified binaries, the macOS app installation path, and update behavior."
order: 10
category: "Reference"
summary: "Verified binaries, the macOS app installation path, and update behavior."
---

## macOS installation

```sh
curl -fsSL https://noria.ashray.xyz/install.sh | sh
```

The installer downloads the platform archive and SHA-256 manifest, verifies the archive and ad-hoc app signature, and installs `noria` with its sibling `Noria.app` under `~/.local/bin`. The menu bar has no Dock icon. No administrator privileges are needed.

This is Zodex-style CLI distribution: curl and extraction do not introduce the browser-download quarantine path. The installer does not disable Gatekeeper, remove quarantine flags, or claim notarization. Browser-downloaded bundles and organizational policies may produce additional checks.

Use `noria update` for an existing installation so replacement is coordinated with active work.

## Other platforms and npm

```sh
npm install -g @amxv/noria
```

The optional Node shim installs a checksum-verified native binary. Node is not required by the standalone CLI or Claude quota reader. Windows and Linux have the same account/router/background commands; the native menu bar requires macOS 13 or later.

Releases contain six OS/architecture binaries, two macOS app archives, licenses, and `SHA256SUMS`. Select your actual OS and architecture from [GitHub Releases](https://github.com/amxv/noria/releases).

## Automatic updates

The background service and menu periodically invoke the CLI updater when Automatic Updates is enabled. The Go CLI owns the preference, download, verification, deferral, replacement, and service restart. Manual `noria update` remains available when the menu is closed.

Open requests defer replacement; idle streaming connections reconnect after a safe service update. Failed verification leaves the installed release unchanged. Replacement retains rollback information so a failed new service can return to the prior release.

## Maintainers

Update version and changelog, run Go/race/native-client tests, verify both macOS app architectures, and check/build docs serially. Tags trigger CI, six native binaries, two macOS archives, checksums, npm publishing, and published-install tests. Verify results rather than equating a tag push with release completion.

Docs auto-deploy from `main` using the Vercel project's `docs` root. Do not include private runtime data in a package or deployment.
