---
title: Your account pool
description: Add multiple Codex and Claude Code accounts, view limits, and manage sign-ins.
order: 2
category: Get started
---

Every account you add has its own credentials, usage allowance, and reset times. Noria shows **email addresses by default**. Aliases are optional.

## Add an account

From the macOS menu bar, choose **Configure → Add Codex Account…** or **Configure → Add Claude Account…**.

Or use the CLI:

```sh
noria account add codex
noria account add claude
```

Sign in through your browser and repeat for additional accounts. For example, run `noria account add codex` three times to add three different Codex accounts.

Noria checks each new account's identity and usage. If sign-in is interrupted, select **Sign In Again…** or run `noria account relogin <id>` using the ID shown by `noria account list --json`.

## Check remaining usage

```sh
noria status
noria quota refresh
```

The menu and status command show **percentage remaining** for the five-hour and weekly allowances, plus their reset times. The windows reset independently on each account. If usage is temporarily unavailable, Noria shows it as unknown—not as free quota.

## Rename or temporarily disable

Choose an account's submenu on macOS, or use:

```sh
noria account rename you@example.com "Work"
noria account rename you@example.com ""       # Show email again
noria account disable you@example.com
noria account enable you@example.com
noria account relogin you@example.com
```

An alias changes only what you see, not where an existing session is routed. A disabled account is excluded from new placement; this does not cancel already accepted work or sign you out.

To remove an account entirely from the pool, run `noria account remove you@example.com`. The isolated login files are retained for recovery.

If two accounts have the same email or alias, use their unique IDs from `noria account list --json`.

**Good to know:** Adding the same subscription more than once won't increase its allowance. Noria only helps distribute work across genuinely separate authorized accounts.
