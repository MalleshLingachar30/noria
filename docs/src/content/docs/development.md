---
title: Development
description: Build and validate Noria using isolated fixtures, not real account state.
order: 14
category: Reference
summary: Build and validate Noria using isolated fixtures, not real account state.
---

The CLI is the source of truth. Add user-facing behavior to Go first, with structured results and tests, then expose it through Swift. Do not duplicate routing, account state or preference storage in the UI.

```bash
make check
go test -race ./...
make build
make docs-check
make docs-build
```

Run docs check and build serially. Node is used for npm packaging tests and Bun for the Astro site; neither is a runtime dependency of the standalone router. macOS app builds use `apps/menubar/build.sh` and Swift/AppKit.

Tests must use temporary `HOME`, `NORIA_HOME`, `CODEX_HOME`, and `CLAUDE_CONFIG_DIR` where native state is involved. Never touch real authentication roots, login items, or client configuration in an automated fixture.

Opt-in installed-client fixtures use `NORIA_NATIVE_COMPAT=1`. Set `NORIA_TEST_CODEX_BIN` to the actual native Codex executable rather than an environment-dependent package-manager shim. Set `NORIA_TEST_NORIA_BIN` to the built Go CLI for the persistent-settings/helper tests. They use fake local upstreams, not paid model traffic.

Key areas: `internal/core` scheduling, `internal/provider` native auth/usage, `internal/integration` reversible settings, `internal/service` background lifetime, `internal/distribution` verified assets, `internal/proxy` request transport, and `apps/menubar` presentation.
