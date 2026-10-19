---
title: Development
description: Build and test Noria, understand its main packages, and edit the ZueDocs site.
order: 10
category: Reference
---

Noria is a Go smart proxy with a native Swift macOS menu bar app. The documentation site uses Astro and ZueDocs.

## Build from source

```sh
git clone https://github.com/amxv/noria.git
cd noria
make check
go test -race ./...
make build
```

`make build-all` creates platform-specific Go binaries. The macOS menu bar source lives in `apps/menubar/`.

## Code map

| Directory | Purpose |
| --- | --- |
| `internal/core/` | Session affinity, quota windows, placement, and reservations |
| `internal/provider/` | Separate native Codex and Claude account handling |
| `internal/proxy/` | Local HTTP/SSE and WebSocket routing |
| `internal/integration/` | Persistent client configuration and restoration |
| `internal/service/` | Background service and login management |
| `internal/app/` | CLI commands and status |
| `apps/menubar/` | Swift user interface for the CLI |
| `docs/` | Astro/ZueDocs website |

## Test safely

All tests must use temporary `HOME`, `NORIA_HOME`, `CODEX_HOME`, and `CLAUDE_CONFIG_DIR` locations, plus fake credentials and test upstreams. Never run automated tests against your personal subscriptions or modify real client settings.

Preserve three key invariants: healthy sessions remain on their account, different quota windows reset independently, and failed requests are retried only before output has been committed. Keep provider protocols and model IDs unchanged.

## Work on the website

```sh
make docs-install
make docs-dev
make docs-check
make docs-build
```

Keep ZueDocs for the shared documentation layouts and use project-local styles and components for Noria. Run the check and build serially, not simultaneously.

The Vercel project is connected to the repository's `main` branch with `docs/` as its root. Its ignored-build script deploys only when `docs/` or the root native install scripts change; code-only changes skip the site build.

For contributions and release details, see [CONTRIBUTORS.md](https://github.com/amxv/noria/blob/main/CONTRIBUTORS.md).
