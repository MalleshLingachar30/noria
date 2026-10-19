# Noria

**Set up once. Use your coding tools normally.**

Noria pools your authorized Codex and Claude accounts, keeps each session on one account until a real limit, and chooses the next account using independent quota/reset clocks. A native macOS menu bar puts accounts, remaining usage and everyday controls one click away.

[Quickstart](https://noria.ashray.xyz/docs/quickstart/) · [Connect clients](https://noria.ashray.xyz/docs/native-clients/) · [Releases](https://github.com/amxv/noria/releases)

## Install

macOS / Linux:

```bash
curl -fsSL https://noria.ashray.xyz/install.sh | bash
```

Windows PowerShell:

```powershell
irm https://noria.ashray.xyz/install.ps1 | iex
```

The native installers verify release checksums. On macOS, the installer also opens Noria's Swift menu bar. The standalone router needs no Node.js or Agent SDK. The optional npm distribution remains `npm install -g @amxv/noria`.

## One-time setup

Use **Add Codex Account… / Add Claude Account…** and **Configure Clients…** in the menu. Or:

```bash
noria account add codex
noria account add claude
noria integrate codex --yes
noria integrate claude --yes
```

Complete browser sign-in for each intended subscription. Accounts display their email unless you add an optional alias. Integration backs up and changes only native routing fields. Reopen existing clients once, then launch `codex`, `claude`, or a local host reading the configured profile normally.

## Everyday use

```bash
noria start
noria status
noria stop
```

The service runs in the background. Stop restores the direct settings of registered client profiles and keeps their normal login intact. The menu and service have separate login-start preferences; quitting the menu does not stop the router.

Use `noria account rename EMAIL "Alias"`, per-account enable/disable/relogin, `noria health`, and `noria update` as needed. All account, quota, service, integration and update state lives in the Go CLI, not in Swift.

## Boundaries

Native client-specific configuration matters: Claude Desktop's third-party setup is separate from Claude Code's settings file, and remote/cloud clients do not automatically reach localhost. See the client guide.

Hard affinity takes priority over proactive rebalancing. Noria does not create quota, buy credits, redeem resets, change the requested model, or splice partially delivered streams. Automated native-client tests use isolated profiles and fake upstreams; live subscription accounting still requires testing with your signed-in accounts.

The supported macOS installer follows a verified curl/tar route with an ad-hoc-signed app. It neither disables Gatekeeper nor strips quarantine; browser-downloaded app copies can still trigger warnings because the app is not Developer ID notarized.

## Development

```bash
make check
go test -race ./...
make build
make docs-check
make docs-build
```

See [development](https://noria.ashray.xyz/docs/development/) for isolated native fixtures and Swift builds. Apache-2.0; see LICENSE and THIRD_PARTY_NOTICES.md.
