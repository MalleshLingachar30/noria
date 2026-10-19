---
title: Release & distribution
description: How Noria’s native binaries, npm shim, changelog, and tag-driven release fit together.
order: 10
category: Development
summary: The maintainer workflow for @amxv/noria and public GitHub release assets.
---

Noria is distributed as `@amxv/noria` through npm and native binaries through [GitHub Releases](https://github.com/amxv/noria/releases). The project is licensed under Apache-2.0.

## Prepare a release

Review the application and documentation together. Verify that help output, native quota control behavior, client wrapper behavior, and transport limitations match the implementation. Do not claim live account testing from fake-upstream results.

Update the [changelog](/docs/changelog) with the release date and actual behavior, then run checks:

```bash
make check
go test -race ./...
make docs-check
make docs-build
```

Docs validation is serial. Review supported platform builds and the package’s contents before publishing. The Go binary implements Claude's `get_usage` control request directly, with no additional runtime dependency.

After maintainer review, commit and push the release changes, then create and push the version tag. For v0.1.1:

```bash
make release-tag VERSION=0.1.1
```

This target creates and pushes a tag; it is a publishing action, not a dry run. The `v*` tag triggers GitHub Actions to run the release pipeline and publish native assets and npm. Maintainers should follow the repository’s release skill/checklist and verify the workflow’s result rather than assume tag creation means publication succeeded.

## Publishing prerequisites

The npm package name must be publishable under the maintainer’s account or organization. The release workflow requires the repository’s configured npm publishing credential (`NPM_TOKEN`). Manage it in the credential store or GitHub secret settings; do not commit or print it.

The public `amxv/noria` repository is the normal anonymous download path. Package version, release tag, binary download origin, and project metadata must agree. Deployment and release credentials are maintainer concerns; they are not required for ordinary local use.

## Native assets

Release names follow this contract:

```text
noria_darwin_amd64
noria_darwin_arm64
noria_linux_amd64
noria_linux_arm64
noria_windows_amd64.exe
noria_windows_arm64.exe
```

The npm postinstall script resolves the asset for the platform. The installed native path is `bin/noria-bin`, with `.exe` on Windows. Changes to asset names require coordinated updates to the release workflow and installer.

## npm installation

```bash
npm install -g @amxv/noria
```

The Node shim invokes the packaged native binary. Postinstall downloads it from GitHub Releases and can fall back to a local Go build if the download fails and Go is available. Download and installation failures should remain visible rather than masquerading as a successful setup.

Native Codex/Claude tools are separate prerequisites. npm installation does not sign in to provider accounts. Claude quota reads use the native `claude` binary directly. Node is needed for the npm distribution shim only; direct native binaries have no Node or Agent SDK dependency.

## Documentation deployment

The static website is independent of the gateway. Its canonical URL is `https://noria.ashray.xyz`; use `docs` as the deployment Root Directory. Build output is `docs/dist`. A docs deployment never hosts your local accounts or forwards your model requests.
