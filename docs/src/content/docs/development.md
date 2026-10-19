---
title: Development
description: Build Noria locally and verify routing changes with isolated profiles and fake upstreams.
order: 9
category: Development
summary: Repository map, local checks, documentation workflow, and behavioral invariants.
---

Noria is a Go CLI and local gateway. The npm package distributes the native executable; the documentation lives in a separate Astro/ZueDocs workspace.

## Build and check

Use the Go version declared in `go.mod`, Node.js for the npm wrapper checks, and Bun for the docs workspace. Native Codex and Claude executables are needed for interactive use, not for tests that substitute fake processes. The Claude quota bridge also requires Node and its pinned SDK.

```bash
git clone https://github.com/amxv/noria.git
cd noria
make check
go test -race ./...
make build
./dist/noria --help
```

Use `make install-local` to install a local build. `make build-all` builds the configured release targets. Neither is a provider login.

## Code map

| Path | Responsibility |
| --- | --- |
| `cmd/noria/` | Process entrypoint and errors |
| `internal/app/` | CLI, daemon lifecycle, control commands, native wrappers |
| `internal/core/` | Affinity, quota eligibility, placement, reservations, persisted state |
| `internal/provider/` | Native quota readers, credentials, embedded Claude bridge |
| `internal/proxy/` | Local gateway policy, HTTP/SSE and WebSocket transports |
| `internal/config/` | Private home, configuration, atomic writes, locks |
| `internal/model/` | Shared account and quota vocabulary |
| `internal/buildinfo/` | Version metadata |
| `bin/`, `scripts/` | npm shim and native-binary installation |
| `docs/` | Isolated static docs and marketing site |

## Preserve the routing contract

Changes should retain hard session affinity, no snapback after quota failover, independent applicable windows, and model/provider fidelity. Keep selection and reservations atomic. Unknown telemetry remains unknown, and older snapshots must not erase newer rejections. Release reservations once and reconcile observed consumption without double counting.

Test retry behavior before the output-commit boundary and visible failure after it. Preserve native event payloads and cancellation. Distinguish quota, auth, throughput, network, provider server, and nonretryable request errors. Do not turn an arbitrary network error into a permanent account move.

## Test without real accounts

Always use temporary `HOME`, `NORIA_HOME`, `CODEX_HOME`, and `CLAUDE_CONFIG_DIR` roots in tests. Use fake credentials, fake native executables, and HTTP/WebSocket test upstreams. Never import the developer’s native auth or access their real conversation roots. Do not run an interactive login, spend live model quota, or use dummy prompts to start windows as part of automated checks.

Tests should exercise concurrency, reset order/nullability, scoped limits, persisted bindings, early quota errors, request preservation, local auth, Host/Origin policy, and cancellation. A passing fake-upstream test is not live subscription validation. Keep that distinction in release notes.

## Work on the website

```bash
make docs-install
make docs-dev
```

The site uses the shared `zuedocs` layouts and enhancements. Keep project content and purposeful style overrides local rather than forking the shared shell. Categories in Markdown must match `docs/src/data/docs.ts` and its content schema.

Validate serially:

```bash
make docs-check
make docs-build
```

Never run Astro check and build concurrently in the same workspace. Test responsive navigation, theme switching, code copying, search, article links, and `/docs/<slug>.md` raw Markdown routes after a production build.

The canonical site is `https://noria.ashray.xyz`. Vercel should use `docs` as its Root Directory; the site builds into `docs/dist`. See [release and distribution](/docs/release-and-npm) for publishing responsibilities.
