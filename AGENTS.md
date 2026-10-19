# Noria agent instructions

Noria is a lightweight Go CLI and local native Codex/Claude account gateway. Public identity: `amxv/noria`, npm `@amxv/noria`, documentation `https://noria.ashray.xyz`, Apache-2.0.

## Architecture

- `cmd/noria/`: process entrypoint and error handling.
- `internal/app/`: commands, daemon lifecycle/control, native login and client wrappers.
- `internal/core/`: hard affinity, quota eligibility, placement/reservations, persisted metadata.
- `internal/provider/`: isolated native quota/auth integration and embedded Claude bridge.
- `internal/proxy/`: gateway policy, HTTP/SSE and WebSocket forwarding.
- `internal/config/`, `internal/model/`: private configuration and shared types.
- `internal/buildinfo/`: version metadata.
- `bin/noria.js`, `scripts/postinstall.js`: npm shim and release binary installation.
- `project.config.json`: repeated project identity.
- `.github/workflows/release.yml`: tag-driven release.
- `docs/`: isolated Astro/ZueDocs workspace.

## Behavioral invariants

Keep healthy sessions on their bound account until an applicable limit or unavailability requires a switch. No proactive score-based migration or snapback. New placement and failover consider independent five-hour/weekly/model constraints, reset timing, learned consumption, and in-flight reservations. Every applicable blocker must clear. Unknown telemetry stays unknown. No arbitrary quota reserve, dummy window-start prompt, paid fallback, reset-credit consumption, model swap, or provider swap.

Selection/reservation must be atomic and reconciliation must avoid double counting. Persist affinity/blocker metadata safely. Native bodies, continuation data, reasoning/tool events, and terminal events must retain their meaning and payloads. Retry only before output is committed; never claim partially delivered generations can be transparently spliced.

Isolate upstream auth profiles under absolute `NORIA_HOME` paths. Normal client wrappers preserve user histories/config roots with child-only overrides; do not persistently edit those roots. Default loopback only, random local secret, Host/Origin validation, private state, and no content/token logging.

## Tests and local work

Use `make fmt`, `make test`, `make vet`, `make lint`, `make check`, `make build`, `make build-all`, and `make install-local` as appropriate. Go changes should also receive relevant race/integration checks.

All automated tests use temporary isolated `HOME`, `NORIA_HOME`, `CODEX_HOME`, and `CLAUDE_CONFIG_DIR` values, fake credentials/native processes, and fake upstreams. Never touch actual `~/.gg/{codex,claude}` or `~/.{codex,claude}` profiles, launch real interactive account login, or spend live quota in tests. Never equate fake-upstream checks with live multi-account validation.

Keep new code files at or below 1,000 lines. Respect task ownership when other workers are active. Do not commit, push, tag, or deploy unless the current task authorizes it.

## Documentation

Update docs when command behavior changes. `docs/src/data/docs.ts` owns identity/nav/categories; content categories must match the schema. Keep the shared shell imported from `zuedocs/layouts/*`, behavior from `zuedocs/docsEnhancements`, and `zuedocs/styles.css` in the local style entrypoint. Keep project-specific content and overrides local. GitHub navigation must retain its repository URL/icon.

Run `make docs-check` then `make docs-build` **serially**. Never run Astro check/build concurrently in one workspace. Verify HTML and raw Markdown routes, responsive navigation, light/dark themes, search, and code copying. Use canonical `https://noria.ashray.xyz`, Vercel Root Directory `docs`, and output `docs/dist`.

## Release contract

Update `docs/src/content/docs/changelog.md` with real dated release notes before tagging. Follow `.agents/skills/release/SKILL.md` when the release skill is invoked. Native asset names stay `noria_<goos>_<goarch>[.exe]`; npm-installed binary stays `bin/noria-bin[.exe]`. Coordinate changes with both release workflow and postinstall.

The public GitHub release origin and npm identity must match package metadata. GitHub Actions publishes on `v*` tags and needs its npm publishing credential (`NPM_TOKEN`). Do not print secrets or include `noria env` output in diagnostics. Pin the Claude SDK bridge explicitly, currently 0.3.287; do not install dependencies on the request path.

## Noria v0.2 contracts

Go owns all account, service, integration, quota and update state. Swift is a CLI wrapper only. The public lifecycle is start/stop/restart, not serve. Stop restores only registered client-routing fields; native auth files are not copied or replaced by integration. Emails are default labels; aliases are optional and never replace immutable routing IDs.

Build the macOS menu with apps/menubar/build.sh. Distribution follows the verified native installer route: ad-hoc-signed app and CLI tarballs, SHA256SUMS, no Gatekeeper disabling or quarantine removal. No claim of Developer ID notarization.

Core docs focus on user value and one-time setup; internals belong in Reference. Keep existing native account fixtures isolated and never run live login as an automated test.
