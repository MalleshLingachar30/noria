---
title: Changelog
description: Highlights from Noria releases.
order: 11
category: Reference
---

## 0.3.5 · October 20, 2026

- Corrected Codex subscription-limit classification so old/resumed sessions automatically fail over to a different available account when an exhausted account returns a generic usage-limit response, without confusing ordinary per-minute throttling with subscription exhaustion.
- Fixed Claude Code's **Usage unavailable** when its installed native CLI reports `rate_limits: null` for an authenticated account. Noria still queries the native binary first, then uses that account's isolated OAuth credential to read the provider's read-only usage endpoint directly from Go (no Node, SDK, or model prompt). Quota percentages and actual reset timestamps are preserved, and unrelated breakdown/billing metadata is ignored.

## 0.3.4 · October 20, 2026

- Fixed automatic update checks when GitHub's anonymous release API is rate-limited (HTTP 403). Noria now discovers stable releases through GitHub's public latest-release redirect, with strict URL validation and its existing checksum-verified asset downloads.

## 0.3.3 · October 20, 2026

- Made macOS **Automatically Update** register an independent hourly launchd job, so verified updates continue when Noria's menu bar or router is not running. Startup repairs missing registrations, and disabling the setting removes the job.
- Hardened menu bar restarts after updates: wait for the previous process to exit before reopening, and show an actionable repair option if the updater registration fails.
- Added isolated launchd registration tests, documentation, and installation-time updater setup.

## 0.3.2 · October 20, 2026

- Reorganized the macOS menu bar: the main menu shows populated providers, their reported quota windows, Start/Stop, and Refresh. Account setup, client settings, health, startup preferences, upgrades, and documentation are under **Configure**.
- Renamed client setup to **Configure Codex** and **Configure Claude Code**. Empty provider sections and unavailable five-hour quota lines no longer clutter the menu.
- Rebuilt the Noria website, docs, and README around local smart routing, account switching, and how to use the application. Added a lightweight interactive routing example and a clearer algorithm guide.

## 0.3.1 · October 20, 2026

- Fixed **Configure Codex** crashing when an existing `config.toml` includes nested arrays of tables such as `[[skills.config]]`.
- Preserved the original `model_provider` line in its original location when restoring direct settings.

## 0.3.0 · October 20, 2026

- Switched entirely to native macOS, Linux, and Windows installers and GitHub Releases.
- Added `noria upgrade` for the CLI and native menu bar updates.
- Made account setup email-first with optional aliases; removed the previous named-account registration path.

## 0.2.0 · October 20, 2026

- Added a native macOS menu bar for usage, reset times, account management, and Start/Stop.
- Added nonblocking background service management on macOS, Linux, and Windows.
- Made Codex and Claude Code routing persistent and reversible through their regular configuration files.
- Added automatic updates, health checks, and browser-based account sign-in.

## 0.1.1 · October 20, 2026

- Switched Claude usage readings to the installed native `claude` executable.

## 0.1.0 · October 20, 2026

- Initial local proxy with Codex/Claude account pools, quota-aware selection, sticky sessions, and safe failover.

See all tagged releases and assets on [GitHub](https://github.com/amxv/noria/releases).
