# Noria

**Keep your agents in flow.**

A lightweight Go CLI and local gateway for native Codex and Claude account pools. A session stays on its account until an applicable limit requires a switch. Initial and failover placement considers independent five-hour/weekly resets, learned consumption, and work already in flight.

[Documentation](https://noria.ashray.xyz) · [Quickstart](https://noria.ashray.xyz/docs/quickstart) · [Releases](https://github.com/amxv/noria/releases)

## Install and start

Install the native Codex or Claude client for the provider you use, then install Noria:

```bash
npm install -g @amxv/noria
noria init
noria account add work --provider codex
noria account login work
noria quota refresh
noria serve
```

Complete the native interactive sign-in yourself. In another terminal:

```bash
noria run codex --
```

Add another authorized account with a different local ID to create a failover pool. Only accounts matching the original provider and requested model are eligible.

For Claude, Node.js and an explicit pinned SDK bridge installation are required:

```bash
noria bridge setup
noria account add claude-work --provider claude
noria account login claude-work
noria quota refresh
# With noria serve running in another terminal:
noria run claude --
```

The bridge uses `@anthropic-ai/claude-agent-sdk@0.3.287` and an experimental usage method. Dependencies are installed only during explicit setup. The npm shim needs Node; direct native binaries are also available from Releases.

## The routing contract

- Hard session affinity. A better score or another account’s reset does not move a healthy session. No snapback after quota failover.
- Independent account clocks. Five-hour, weekly, and applicable model limits must all permit use. Unknown telemetry stays unknown.
- Reset-aware placement. Bounded expiry/capacity estimates and atomic reservations guide new work, rather than rotating accounts per request.
- Native requests. Preserve provider, model, settings, tool events, and continuation data. No protocol translation or model swaps.
- Local state. Default loopback endpoint `127.0.0.1:43119`, random local authentication, private profiles under `~/.noria` (`NORIA_HOME` overrides).
- Preserve normal client history/config roots. Isolate upstream account authentication; inject routing overrides only into the child client.

## Boundaries

Noria does not create quota, buy credits, redeem resets, or enable paid fallback. If every matching account is unavailable, it reports that and reset information when known. A forced switch starts with a cold cache on the replacement account. A partially delivered generation cannot be transparently spliced, and cross-account continuation can depend on state held by the provider.

**Live multi-account workflows have not yet been validated.** Installed Codex and Claude clients pass local tests that resume saved conversations, force a quota rejection on one account, and continue on another while retaining prior turns. These tests use fake credentials and test upstreams, not live subscription capacity. See [compatibility and validation](https://noria.ashray.xyz/docs/troubleshooting#compatibility-and-validation) before relying on a pool.

`noria status`, `noria doctor`, and `noria quota refresh` help inspect setup. `noria env` intentionally prints the local gateway secret; do not share its output. `account remove` unregisters an account but retains its native profile credentials/history.

## Development

Use the Go version in `go.mod`, Node for packaging checks, and Bun for docs:

```bash
make check
go test -race ./...
make build
make docs-install
make docs-check
make docs-build
```

Run docs check and build serially. Tests must use temporary isolated homes, fake native processes, and fake upstreams, never the developer’s actual authentication roots. See [CONTRIBUTORS.md](CONTRIBUTORS.md) and the [development guide](https://noria.ashray.xyz/docs/development).

Apache-2.0. See [LICENSE](LICENSE).
