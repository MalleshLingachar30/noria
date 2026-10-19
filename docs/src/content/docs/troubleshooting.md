---
title: Troubleshooting
description: Fix common setup issues without exposing credentials or changing account limits.
order: 6
category: Use Noria
summary: Fix common setup issues without exposing credentials or changing account limits.
---

## Start here

Choose **Health Check** in the menu, or run:

```bash
noria health
noria status
noria doctor
```

Health checks the local endpoint and registered settings. It does not send a model prompt or spend usage to “test” an account.

## My ordinary client is not using Noria

Run `noria integrations --json` and confirm the configuration home matches the one your client actually reads. Use **Configure Clients…** to choose another home, then reopen that client. For manual settings and GUI-specific exceptions, see [Connect your clients](/docs/native-clients).

Clear conflicting credential/base-URL/cloud-provider overrides. A host with its own command-line overrides, a managed Claude Desktop profile, or a remote environment can ignore your local settings.

## I stopped Noria and an open client still tries localhost

Stop restores the files Noria manages. Some applications cache configuration until they restart: close and reopen that client. Its normal native login remains available. A manual Claude Desktop third-party configuration must be switched in the desktop app itself.

## An account shows unknown usage

```bash
noria quota refresh
noria account login person@example.com
```

First refresh. If sign-in is required, use the account's **Sign In Again…** action. A disabled account can still show its last snapshot; failed reads never mean unlimited capacity. Your native Codex/Claude version must support the quota control interface.

## I accidentally signed in to the same account twice

The duplicate is not enabled as extra capacity. Remove the disabled pending entry or sign into the intended different account. Use `noria account list --json` for stable IDs when labels are ambiguous.

## Updates are deferred

An active generation has priority over a restart. Let it finish, then run `noria update` again, or leave automatic updates enabled. Downloads and checksums are verified before installation. An update error does not authorize a paid fallback or reset credit redemption.

## A settings conflict is reported

Noria refuses to overwrite a routing field changed by another tool after integration. Inspect the named file and its backup. Restore the managed field to its installed value, then use `noria integrate ... --remove --yes`, or remove the integration manually after reviewing the saved values. Do not restore an entire old backup over unrelated new settings.

## macOS warns about a manually downloaded app

Use the documented native installer, which follows the Zodex-style verified curl/tar path. The app is ad-hoc signed, not Developer ID notarized. A browser-downloaded quarantined copy may still be assessed differently. Noria does not disable Gatekeeper or strip quarantine flags. See [Security and distribution](/docs/security-and-limitations).

## Compatibility and validation

Tests cover temporary profiles, fake provider backends, native Codex/Claude clients using saved configuration and token helpers, quota failover, resume, and service/configuration round trips. macOS launchd behavior is exercised locally. CI tests and builds cover macOS, Linux and Windows.

The installed native-client compatibility checks used Codex 0.162.1 and Claude Code 2.1.299. They did not use real subscription credentials or deliberately exhaust live account quotas. End-to-end live browser login and provider-specific quota accounting still need validation with the accounts you add.

## Share a useful report

Include Noria/native client versions, OS, the command, a redacted error, and whether output had already started. Do not include auth files, `noria env` output, token-helper output, raw requests, or a private-home archive.
