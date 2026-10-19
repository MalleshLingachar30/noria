---
title: "Manage accounts and aliases"
description: "Add accounts by signing in and display their emails or optional aliases."
order: 2
category: "Start"
summary: "Add accounts by signing in and display their emails or optional aliases."
---

## Add an account

Use **Add Codex Account…** or **Add Claude Account…** in the menu, or:

```sh
noria account add codex
noria account add claude
```

The native sign-in opens in your browser. Choose the intended account. After sign-in, Noria shows its email and refreshes usage. Incomplete new sign-ins stay disabled until completed with **Sign In Again…**.

Pool authentication is separate from your ordinary client login. Adding an account does not switch the account used by a normal direct Codex or Claude installation.

## Optional aliases

Choose **Rename / Set Alias…** on an account. Leave the alias empty, or choose **Use Email as Name**, to show its email again.

```sh
noria account rename alex@example.com "Work"
noria account rename alex@example.com ""
```

The immutable internal ID stays the same, so renaming does not move sessions or discard their account affinity.

## Enable, relogin, or remove

```sh
noria account disable alex@example.com
noria account enable alex@example.com
noria account relogin alex@example.com
noria account remove alex@example.com
```

These actions are also in each account's submenu. Disabling prevents new placement on that account; it does not revoke the native login or cancel already accepted work. Removing unregisters the account but retains its profile for recovery; it is not secure erasure or provider logout.

When an email or alias is ambiguous, use the internal ID from `noria account list --json`. Adding the same subscription twice does not create more quota.

## Usage information

```sh
noria quota refresh
noria status
```

Five-hour and weekly windows have independent reset times. The menu and status table show **percentage remaining**. Missing or stale data is labeled, never presented as a full allowance. A failed telemetry read does not clear an authoritative quota rejection. Usage reads do not submit model prompts or redeem reset credits.
