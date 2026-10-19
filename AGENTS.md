# Noria development contract

Work on the user-selected machine and worktree. Do not delegate when the user has prohibited delegation. Never run test prompts or authentication against the developer's actual subscription profiles.

## Architecture

Go owns account identity, hard session affinity, quota snapshots, persistent integration backups, lifecycle, preferences, and updates. Native Swift is a presentation wrapper around CLI commands and JSON, not a second scheduler or settings store.

Public lifecycle is `start`, `stop`, `restart`. There is no public `serve` command; internal daemon commands exist for OS supervision. Stop restores managed direct-client configuration without copying pool credentials into ordinary native profiles. Reopen already-running clients when settings are cached.

Integration is an explicit, previewable, reversible user-authorized operation. Tests must use temporary CODEX_HOME, CLAUDE_CONFIG_DIR, NORIA_HOME and HOME. Preserve unrelated edits and comments. Refuse to overwrite external routing changes.

Accounts display email by default; aliases are optional and never replace immutable routing IDs. Native logins retain provider-owned OAuth storage/callbacks. Do not log auth URLs, secrets, tokens, or model request content.

## Checks

Run `make check`, `go test -race ./...`, native persistent-config fixtures, and cross-platform service tests. Build Swift for arm64 and x86_64, verify signatures, and exercise CLI/menu JSON. Run `make docs-check` then `make docs-build` serially, never concurrently.

Code files should stay below 1000 lines. Add tests for lifecycle idempotence, config roundtrips/conflicts, stable aliases, atomic reservations, safe early failover, deferred updates, and release checksum/archive validation.

## Distribution

Use the Zodex-style checksum-verified CLI download/extract path with an ad-hoc signed macOS app. Never disable Gatekeeper, clear quarantine, or claim Developer ID notarization. Browser downloads and managed policies may differ. macOS CLI and Noria.app are sibling artifacts.

Update package.json and manual changelog, verify CI, then push the release tag. Confirm six binaries, two macOS archives, checksums, npm install smoke, and docs deployment. Never publish from a concurrently changing checkout or discard someone else's worktree changes.
