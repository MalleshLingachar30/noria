# Contributing to Noria

Noria is a local native Codex/Claude account gateway, distributed as `@amxv/noria`. Start with the [development guide](https://noria.ashray.xyz/docs/development) and [routing contract](https://noria.ashray.xyz/docs/scheduling).

## Prerequisites and checks

Use the Go version in `go.mod`, Node.js for the npm wrapper and script checks, and Bun for the isolated docs workspace. Native clients are required for interactive use; automated tests substitute fake processes and upstreams. Claude quota reads invoke the installed native binary directly; no Node.js or Agent SDK bridge is required.

```bash
make check
go test -race ./...
make build
./dist/noria --help
make docs-install
make docs-check
make docs-build
```

Run `make docs-check` then `make docs-build` serially. For site work use `make docs-dev`; keep all web tooling in `docs/` and shared layouts imported from ZueDocs.

## Isolation and behavior

Never use actual developer auth, histories, or native config roots in automated tests. Set temporary `HOME`, `NORIA_HOME`, `CODEX_HOME`, and `CLAUDE_CONFIG_DIR`; use fake credentials, fake native executables, and HTTP/WebSocket test upstreams. Do not run real login flows or spend live quota during automated validation.

Preserve hard session affinity, independent reset windows, atomic reservations, and request/model/provider fidelity. Do not move healthy sessions because scores change or a reset happens elsewhere. Retry only when safe before output is committed. Do not replay delivered tools or claim arbitrary midstream splicing.

Account authentication profiles are isolated under Noria’s home. Normal `noria run` clients retain their existing conversation/config roots with child-only overrides. Do not persistently edit normal native settings to make a test pass.

No request-content or token logs. Do not include secrets, auth files, transcripts, or `noria env` output in reports. Label fake-upstream results accurately; live multi-account validation remains a separate explicit activity.

## Releases

Update `docs/src/content/docs/changelog.md` with the real behavior and release date. Reconcile command help, SDK pin, wrappers, and transport caveats. Run the checks above and verify release-platform builds and npm contents.

After maintainer review, stage, commit, and push the release changes, then use `make release-tag VERSION=x.y.z`. This pushes a `v*` tag and triggers GitHub Actions to publish native assets and npm. The workflow needs its npm publishing credential (`NPM_TOKEN`). Never commit or print credentials.

Keep asset names `noria_<goos>_<goarch>[.exe]` and installed paths `bin/noria-bin[.exe]` consistent with the workflow and postinstall script. The docs site uses canonical URL `https://noria.ashray.xyz`, Vercel Root Directory `docs`, and output `docs/dist`.

See the [release guide](https://noria.ashray.xyz/docs/release-and-npm) and repository release skill. Contributions are under Apache-2.0.
