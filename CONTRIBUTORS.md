# Contributing to Noria

Noria is a local Go gateway for native Codex and Claude Code accounts, released as standalone binaries with checksum-verified install scripts. Start with the [development guide](https://noria.ashray.xyz/docs/development) and [routing contract](https://noria.ashray.xyz/docs/scheduling).

## Develop and validate

Install the Go version required by `go.mod`. Python 3 runs native release checks; Bun is required only to build the documentation website. End users need none of these build tools.

```sh
make check
go test -race ./...
make build
make docs-install
make docs-check
make docs-build
```

Run Astro docs checks serially. Application tests must always isolate `HOME`, `NORIA_HOME`, `CODEX_HOME`, and `CLAUDE_CONFIG_DIR`, using fake credentials and upstreams. Never touch real subscription accounts, provider configuration or conversation history in automated tests.

Preserve hard session affinity, independently timed quota windows, atomic admission and reservations, native protocol fidelity, and safe failover before output is committed. Never move healthy sessions because a score changes. Don't include tokens, auth files, transcripts, or the output of `noria auth-token` in logs.

## Native releases

Update `VERSION` and the manually curated `docs/src/content/docs/changelog.md` together. Run tests and verify the macOS arm64/x86_64 menu bar builds and ad-hoc signatures. Check both native installers and all six cross-platform binaries.

From a clean `main` branch, push the changes, then use `make release-tag VERSION=x.y.z`. GitHub Actions builds binaries and apps, writes SHA-256 checksums, publishes GitHub Releases, and tests fresh installs on macOS, Linux, and Windows. Noria does not publish a JS or npm package.

macOS installation follows Zodex's checksum-verified curl/tar pattern for its ad-hoc-signed `.app`. We never disable Gatekeeper or remove quarantine from downloaded files. The docs site auto-deploys through Vercel from the `docs/` root.

See the [installation and upgrade reference](https://noria.ashray.xyz/docs/installation-and-upgrades/) and repository release skill. Contributions are licensed under Apache-2.0.
