#!/usr/bin/env bash
# Noria native installer: verify release bytes before installing executable code.
set -euo pipefail
version="${NORIA_VERSION:-}"
prefix="${NORIA_INSTALL_DIR:-$HOME/.local/bin}"
base='https://github.com/amxv/noria/releases'
for tool in curl tar; do command -v "$tool" >/dev/null || { echo "Missing required tool: $tool" >&2; exit 1; }; done
case "$(uname -s)" in Darwin) os=darwin;; Linux) os=linux;; *) echo 'Use the PowerShell installer on Windows.' >&2; exit 1;; esac
case "$(uname -m)" in arm64|aarch64) arch=arm64;; x86_64|amd64) arch=amd64;; *) echo 'Unsupported architecture' >&2; exit 1;; esac
if [ -z "$version" ]; then
 version=$(curl --proto '=https' --tlsv1.2 -fsSL https://api.github.com/repos/amxv/noria/releases/latest | sed -n 's/.*"tag_name": *"v\([^"]*\)".*/\1/p' | head -n 1)
fi
version="${version#v}"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Could not resolve a stable release version' >&2; exit 1; }
case "$prefix" in /*) ;; *) echo 'NORIA_INSTALL_DIR must be absolute' >&2; exit 1;; esac
asset="noria_${os}_${arch}"
if [ "$os" = darwin ]; then asset="$asset.tar.gz"; fi
stage=$(mktemp -d "${TMPDIR:-/tmp}/noria-install.XXXXXX")
trap 'rm -rf "$stage"' EXIT
curl --proto '=https' --tlsv1.2 -fsSL "$base/download/v$version/SHA256SUMS" -o "$stage/SHA256SUMS"
curl --proto '=https' --tlsv1.2 -fsSL "$base/download/v$version/$asset" -o "$stage/$asset"
expected=$(awk -v name="$asset" '$2==name {print $1}' "$stage/SHA256SUMS")
[[ "$expected" =~ ^[a-fA-F0-9]{64}$ ]] || { echo 'Missing or ambiguous release checksum' >&2; exit 1; }
if command -v shasum >/dev/null; then actual=$(shasum -a 256 "$stage/$asset" | awk '{print $1}'); else actual=$(sha256sum "$stage/$asset" | awk '{print $1}'); fi
[ "$actual" = "$expected" ] || { echo 'Checksum mismatch; nothing was installed' >&2; exit 1; }
mkdir -p "$prefix"
if [ "$os" = darwin ]; then
 mkdir "$stage/unpacked"
 # Permit only the published bundle, binary and license files. No symlinks.
 tar -tzf "$stage/$asset" | while IFS= read -r name; do
  case "$name" in noria|Noria.app|Noria.app/*|LICENSE|THIRD_PARTY_NOTICES.md) ;; *) echo "Unexpected archive member" >&2; exit 1;; esac
  case "/$name/" in */../*) echo 'Unsafe archive member' >&2; exit 1;; esac
 done
 if tar -tvzf "$stage/$asset" | grep -E '^[lhbcps]' >/dev/null; then echo 'Links or special files in release archive' >&2; exit 1; fi
 tar -xzf "$stage/$asset" -C "$stage/unpacked"
 codesign --verify --deep --strict "$stage/unpacked/Noria.app"
 cp "$stage/unpacked/noria" "$stage/noria"; chmod 755 "$stage/noria"
 "$stage/noria" --version
 # Existing services must use the CLI's drain-aware updater, not a blind copy.
 if [ -x "$prefix/noria" ] && "$prefix/noria" status --json 2>/dev/null | grep -Eq '"running": *true'; then
  echo 'Noria is running. Use noria update, or stop Noria before rerunning this installer.' >&2; exit 1
 fi
 cp "$stage/noria" "$prefix/.noria-installing"; chmod 755 "$prefix/.noria-installing"
 "$stage/noria" init >/dev/null
 # Install the app through the validated bundle path; app stays beside the CLI.
 rm -rf "$prefix/.Noria.app-installing"
 cp -R "$stage/unpacked/Noria.app" "$prefix/.Noria.app-installing"
 codesign --verify --deep --strict "$prefix/.Noria.app-installing"
 if [ -e "$prefix/Noria.app" ]; then mv "$prefix/Noria.app" "$stage/old-Noria.app"; fi
 if ! mv "$prefix/.Noria.app-installing" "$prefix/Noria.app"; then
  [ ! -e "$stage/old-Noria.app" ] || mv "$stage/old-Noria.app" "$prefix/Noria.app"; exit 1
 fi
 mv -f "$prefix/.noria-installing" "$prefix/noria"
else
 if [ -x "$prefix/noria" ] && "$prefix/noria" status --json 2>/dev/null | grep -Eq '"running": *true'; then echo 'Run noria update or stop the service before reinstalling.' >&2; exit 1; fi
 cp "$stage/$asset" "$prefix/.noria-installing"; chmod 755 "$prefix/.noria-installing"; mv -f "$prefix/.noria-installing" "$prefix/noria"
fi
"$prefix/noria" init >/dev/null
printf '\nInstalled Noria %s in %s\n' "$version" "$prefix"
case ":$PATH:" in *":$prefix:"*) ;; *) printf 'Add this directory to PATH, for example:\n  export PATH="%s:$PATH"\n' "$prefix";; esac
if [ "$os" = darwin ]; then
 echo 'Opening Noria. Add your accounts, then choose Configure Clients once.'
 "$prefix/noria" menubar open
else
 echo 'Next: noria account add codex   (or claude), then noria integrate codex --yes'
fi
# No quarantine stripping or Gatekeeper changes. curl/tar-created files follow
# the same user-scoped distribution route as Zodex's installer.
