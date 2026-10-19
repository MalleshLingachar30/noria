---
title: Changelog
description: Highlights from Noria releases.
order: 11
category: Reference
---

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
