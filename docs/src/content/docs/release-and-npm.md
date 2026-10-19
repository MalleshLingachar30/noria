---
title: Release reference
description: Publish the native CLI, macOS bundles, npm shim, and the auto-deployed documentation.
order: 15
category: Reference
summary: Publish the native CLI, macOS bundles, npm shim, and the auto-deployed documentation.
---

Update `package.json` and the manual changelog together, run local checks, push main, and wait for CI before creating the matching `vX.Y.Z` tag. A tag starts release builds; it is not proof that all publication and install checks succeeded.

The release publishes six native binaries (macOS/Linux/Windows, amd64/arm64), two macOS tarballs containing the CLI plus `Noria.app`, licenses, and `SHA256SUMS`. Mac bundles are built with Swift and ad-hoc signed. The npm wrapper downloads the matching verified bare binary; `noria menubar install` fetches the matching bundle when needed.

The installer scripts are served as `/install.sh` and `/install.ps1`. They verify downloads before installation and do not disable OS security features. Test a fresh published installation after every release, not only a source build.

The npm workflow uses the repository's configured `NPM_TOKEN`; never print or commit it. Registry propagation is handled by a read-only visibility check, not by republishing the same version.

The docs site auto-deploys from GitHub main through the Vercel project `noria`, with `docs` as Root Directory. Validate the deployed HTTPS pages and current version after push. The website does not host the local account pool or model traffic.
