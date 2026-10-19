---
title: Security and distribution
description: Understand local access, private account storage, and the macOS installation path.
order: 13
category: Reference
summary: Understand local access, private account storage, and the macOS installation path.
---

Noria runs locally, but model requests still go to the selected provider. Use only accounts you are authorized to access and follow each provider's applicable account terms.

## Credentials and local access

The router listens on loopback and requires a random local token. Host/Origin checks reject unexpected browser traffic. These safeguards do not protect against every program running as your own OS user; keep the Noria home private.

Each pool entry uses a separate native auth profile. The installed client manages OAuth and platform storage. Noria does not implicitly import your normal auth roots, does not log request bodies, and does not print upstream credentials in status.

Managed client integrations use token-helper commands rather than storing the local gateway token in TOML/JSON. The helper intentionally prints a secret to its caller. Do not paste that output into logs or support reports.

Backups can contain existing sensitive native settings, so they stay inside the private Noria home. Account email addresses and aliases are also private operational information.

## Reversible setup

Integration changes only the routing fields it owns. Stop restores those fields and preserves the client's previous native login. Unrelated settings edits remain intact; conflicting edits to managed fields are reported instead of overwritten.

Removing a pool account unregisters it but retains its auth profile. It is not a secure erase or provider logout. Stop the service before reviewing and removing a specific profile or its native keychain entry.

## macOS app installation

The native installer downloads a release archive over HTTPS, verifies its SHA-256 manifest, extracts the CLI and ad-hoc-signed app, and installs them in a user-owned directory. This is the same distribution approach described in the Zodex packaging guide.

The supported curl/tar installation path does not depend on disabling Gatekeeper or stripping quarantine. **The app is not Developer ID notarized.** A copy downloaded through a browser and opened from Finder can still carry quarantine and trigger a warning; Noria does not claim otherwise.

Automatic updates use verified release assets and validate the macOS bundle signature before swapping files.

## Limits

There is no added quota, automatic spending, model substitution, or silent provider swap. All accounts can be exhausted. A partially delivered generation cannot be safely stitched across accounts. Some applications need a restart to reload configuration, and app-specific/hosted routing is not a machine-wide network interception feature.

See [Architecture and limits](/docs/architecture) for retry and telemetry boundaries.
