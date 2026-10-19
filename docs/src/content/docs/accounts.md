---
title: Manage accounts
description: Add accounts with browser sign-in, use email-first labels, and set optional aliases.
order: 3
category: Use Noria
summary: Add accounts with browser sign-in, use email-first labels, and set optional aliases.
---

## Add an account

Choose **Add Codex Account…** or **Add Claude Account…** in the menu, or run:

```bash
noria account add codex
noria account add claude
```

The installed native client handles browser sign-in and saves credentials into an isolated Noria profile. Your ordinary client login is not replaced. Once authentication completes, Noria reads the account email and refreshes its quota.

Account emails are the default display names. A private, stable ID identifies each entry internally; renaming never changes session bindings. A failed sign-in remains disabled until it is completed or removed.

## Optional names

In an account's submenu, choose **Rename / Set Alias…**. Leave it blank, or select **Use Email as Name**, to return to the email display.

```bash
noria account rename person@example.com "Work"
noria account rename Work
```

Commands accept a stable ID, a unique email, or a unique alias. If the same email exists under both providers, use the ID from `noria account list --json` to avoid ambiguity.

## Disable, sign in again, or remove

```bash
noria account disable person@example.com
noria account enable person@example.com
noria account login person@example.com
noria account remove person@example.com
```

Disabling removes an account from eligible routing without signing it out. New requests use another eligible account; already accepted work is not deliberately replayed.

Relogin temporarily disables the entry while native credentials change, then confirms its identity and refreshes usage. A duplicate sign-in does not become a second spendable copy of the same account.

Removing unregisters the account but retains its private native profile for recovery. It is not a provider logout or secure erase. See [Security and data](/docs/security-and-limitations) before deleting auth storage or platform keychain entries.

## Account access

Use only subscriptions you are authorized to access. A Codex request is served only by the Codex pool; a Claude request stays in the Claude pool. A local model restriction cannot grant access that the provider does not allow.

The legacy `noria account add <id> --provider codex|claude` registration command remains supported for existing scripts. New users should use the browser-sign-in commands above.
