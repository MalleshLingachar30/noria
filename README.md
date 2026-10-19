# Noria

**Set up once. Use your coding agents normally.**

Noria is a lightweight local account router for Codex and Claude Code. Healthy sessions stay on their account to preserve prompt-cache reuse; new sessions and required failovers are placed using each account's own remaining allowance and reset times.

[Get started](https://noria.ashray.xyz/docs/quickstart/) · [Documentation](https://noria.ashray.xyz/docs/) · [Releases](https://github.com/amxv/noria/releases)

## macOS: CLI and native menu bar

```sh
curl -fsSL https://noria.ashray.xyz/install.sh | sh
```

Click **N**, add your accounts, and approve **Configure Clients…** once. Account titles default to their email addresses. Aliases, enable/disable, relogin, usage refresh, Start/Stop, login preferences, health checks, and updates are available from the menu.

## CLI setup

```sh
npm install -g @amxv/noria
noria init
noria account add codex
noria account add claude
noria start
noria integrate codex --yes
noria integrate claude --yes
```

Each Add opens native browser sign-in. Repeat it for additional accounts. Reopen existing clients after configuration, then run `codex` or `claude` normally. No wrapper is required. Standalone binaries need no Node/Agent SDK runtime.

```sh
noria status
noria stop      # restore managed direct settings and existing native login
noria start
noria update
```

Closing a terminal or quitting the menu does not stop the background service. Set login preferences for automatic startup. Stop does not copy a pool account into the normal client; clients that cache settings may need reopening.

## Boundaries

Noria never buys credits, redeems quota resets, or changes providers/models. No quota exists when all accounts are exhausted. A forced switch starts a cold account cache. Already-delivered model output is not spliced or replayed. Tests use isolated profiles and controlled upstreams, not live subscriptions.

The macOS app uses Zodex-style, checksum-verified CLI installation and ad-hoc signing, not Developer ID notarization. No security settings or quarantine flags are weakened. Browser downloads can be treated differently. See [installation details](https://noria.ashray.xyz/docs/release-and-npm/) and [client-specific setup](https://noria.ashray.xyz/docs/native-clients/), including Claude Desktop's separate form.

## Development

```sh
make check
go test -race ./...
make docs-install
make docs-check
make docs-build
```

All tests must use temporary auth/config homes. The Go CLI owns application state; Swift is presentation only. Apache-2.0. See LICENSE and THIRD_PARTY_NOTICES.md.
