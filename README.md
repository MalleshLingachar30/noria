# Noria

**Stop switching accounts. Start using more of your limits.**

Noria is a **smart local proxy for Codex and Claude Code**. Connect the accounts you already use, point your coding clients at Noria once, and keep coding normally. Noria chooses where new sessions go, keeps them on the same account for prompt caching, and switches to another usable account when a limit gets in the way.

No more logging out of one account just to log into another. No more manually tracking when five-hour and weekly allowances reset.

**[Website](https://noria.ashray.xyz)** · **[Get started](https://noria.ashray.xyz/docs/quickstart)** · **[How routing works](https://noria.ashray.xyz/docs/architecture)** · **[Releases](https://github.com/amxv/noria/releases)**

## Why use Noria?

Your accounts each have their own usage limits, their own reset times, and their own prompt caches. Picking an account is more complicated than choosing the one with the biggest number remaining.

Noria handles three things:

1. **Smarter placement.** New sessions consider the remaining five-hour and weekly allowances, when each resets, and how much work is already assigned. A little quota about to reset can be worth using before a larger allowance that lasts for days.
2. **Sticky conversations.** A conversation stays on its chosen account as long as that account can serve it. This preserves account-level prompt-cache reuse instead of bouncing between accounts on every request.
3. **Automatic failover.** If an applicable limit blocks the next request, Noria looks for another eligible account of the *same provider* and safely retries the pending request. Then it stays on that account.

```text
                        Noria (localhost)
Codex / Claude Code ─────────┬───────────────→ Your account A
                            ├───────────────→ Your account B
                            └───────────────→ Your account C

                   One local route. Multiple accounts.
```

Noria optimizes the allowances you already have; it doesn't generate extra quota or guarantee work can continue after every account is exhausted.

## Install

**macOS and Linux**

```sh
curl -fsSL https://noria.ashray.xyz/install.sh | sh
```

**Windows (PowerShell)**

```powershell
irm https://noria.ashray.xyz/install.ps1 | iex
```

On macOS you also get a native menu bar app for account management, quota usage, and Start/Stop controls.

## Set up once

```sh
# Add accounts (each opens the provider's browser sign-in)
noria account add codex
noria account add codex
noria account add claude

# Start the background proxy
noria start

# Route your existing clients through Noria
noria integrate codex --yes
noria integrate claude --yes
```

The integration changes the relevant client configuration, not your existing conversations or sign-in. **Restart any clients you already have open.** Then launch `codex` or `claude` exactly as before—no wrapper command required.

If your client uses a custom configuration folder, specify it with `--client-home`. See the [setup guide](https://noria.ashray.xyz/docs/quickstart) for desktop apps and alternative paths.

## Day to day

```sh
noria status            # Account usage and reset times
noria quota refresh     # Fetch a fresh usage snapshot
noria health            # Check local routing configuration
noria stop              # Return managed clients to direct provider routing
noria start             # Start routing again
noria upgrade           # Upgrade the CLI and macOS menu bar app
```

Accounts are displayed by **email address by default**. You can optionally set aliases, enable or disable individual accounts, and sign in again from the CLI or the macOS menu bar.

Everything Noria manages lives locally. Its gateway runs on loopback, and each upstream account has separate credentials. Model requests still go to their respective provider—Noria is a router, not a local model.

## Important boundaries

Noria doesn't switch providers or models, purchase usage, move prompt caches between accounts, or splice an already-delivered partial response. Failover retries a request only when safe. Live subscription behavior depends on each provider's supported client interfaces and terms.

The [architecture guide](https://noria.ashray.xyz/docs/architecture) explains the algorithm, and [security and limitations](https://noria.ashray.xyz/docs/security-and-limitations) covers the boundaries in more detail.

## Develop

Noria is written in Go. Its macOS menu bar is a native Swift wrapper over the CLI, which owns accounts, routing, settings, and updates. The website uses Astro and ZueDocs.

```sh
make check
go test -race ./...
make docs-install
make docs-check
make docs-build
```

See [Contributing](CONTRIBUTORS.md). Licensed under [Apache-2.0](LICENSE).
