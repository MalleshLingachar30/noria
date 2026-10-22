#!/bin/sh
set -eu
# Verified native CLI and matching desktop app installation.
repo='https://github.com/amxv/noria'
install_dir=${NORIA_INSTALL_DIR:-"$HOME/.local/bin"}
case "$(uname -s)" in Darwin) os=darwin;; Linux) os=linux;; *) echo 'On Windows use https://noria.ashray.xyz/install.ps1' >&2; exit 1;; esac
case "$(uname -m)" in arm64|aarch64) arch=arm64;; x86_64|amd64) arch=amd64;; *) echo 'Unsupported CPU architecture' >&2; exit 1;; esac
case "$install_dir" in /*) ;; *) echo 'NORIA_INSTALL_DIR must be absolute' >&2; exit 1;; esac
for tool in curl tar; do command -v "$tool" >/dev/null 2>&1 || { echo "$tool is required" >&2; exit 1; }; done
if [ -e "$install_dir/noria" ] || [ -L "$install_dir/noria" ] || [ -e "$install_dir/Noria.app" ] || [ -L "$install_dir/Noria.app" ] || [ -e "$install_dir/noria-tray" ] || [ -L "$install_dir/noria-tray" ]; then
  echo 'Noria is already installed. Run noria upgrade to update it safely.' >&2
  exit 1
fi
version=${NORIA_VERSION:-}
if [ -n "$version" ]; then
  version=${version#v}
  printf '%s\n' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$' || { echo 'Invalid NORIA_VERSION' >&2; exit 1; }
  base="$repo/releases/download/v$version"
else
  base="$repo/releases/latest/download"
fi
umask 077
stage=$(mktemp -d "${TMPDIR:-/tmp}/noria-install.XXXXXX")
trap 'rm -rf "$stage"' EXIT HUP INT TERM
asset="noria_${os}_${arch}.tar.gz"
curl --proto '=https' --tlsv1.2 -fsSL --retry 3 "$base/SHA256SUMS" -o "$stage/SHA256SUMS"
curl --proto '=https' --tlsv1.2 -fsSL --retry 3 "$base/$asset" -o "$stage/$asset"
expected=$(awk -v file="$asset" '$2==file {print $1}' "$stage/SHA256SUMS")
[ "${#expected}" -eq 64 ] || { echo 'Release checksum is missing or ambiguous' >&2; exit 1; }
if command -v shasum >/dev/null 2>&1; then actual=$(shasum -a 256 "$stage/$asset" | awk '{print $1}'); else actual=$(sha256sum "$stage/$asset" | awk '{print $1}'); fi
[ "$actual" = "$expected" ] || { echo 'Release checksum mismatch; nothing was installed.' >&2; exit 1; }
mkdir -p "$install_dir"
mkdir "$stage/unpacked"
tar -tzf "$stage/$asset" | while IFS= read -r entry; do
  case "$entry" in /*|../*|*/../*|*\\*|*:*) echo 'Unsafe archive member' >&2; exit 1;; esac
done
tar -xzf "$stage/$asset" -C "$stage/unpacked"
if [ "$os" = darwin ]; then
  codesign --verify --deep --strict "$stage/unpacked/Noria.app"
  cp "$stage/unpacked/noria" "$install_dir/.noria-next"
  chmod 755 "$install_dir/.noria-next"
  # Copy the complete signed bundle before changing the CLI entrypoint.
  cp -R "$stage/unpacked/Noria.app" "$install_dir/Noria.app"
else
  test -f "$stage/unpacked/noria" && test -f "$stage/unpacked/noria-tray" || { echo 'Release is missing the native tray and CLI' >&2; exit 1; }
  cp "$stage/unpacked/noria-tray" "$install_dir/.noria-tray-next"
  chmod 755 "$install_dir/.noria-tray-next"
  mv -f "$install_dir/.noria-tray-next" "$install_dir/noria-tray"
  cp "$stage/unpacked/noria" "$install_dir/.noria-next"
  chmod 755 "$install_dir/.noria-next"
fi
mv -f "$install_dir/.noria-next" "$install_dir/noria"
"$install_dir/noria" --version
"$install_dir/noria" init
if [ "${NORIA_NO_LAUNCH:-0}" != 1 ]; then
  # Register the independent launchd updater immediately. It keeps checking
  # even if the menu bar is quit or the routing service is stopped.
  if ! "$install_dir/noria" preferences auto-update on >/dev/null; then
    echo 'Automatic updates could not be scheduled. Re-enable them in the menu bar after login.' >&2
  fi
fi
case ":${PATH:-}:" in *":$install_dir:"*) ;; *) printf '\nAdd this to your shell profile, then open a new terminal:\n  export PATH="%s:$PATH"\n' "$install_dir";; esac
if [ "${NORIA_NO_LAUNCH:-0}" != 1 ]; then
  if [ "$os" = darwin ]; then
    "$install_dir/noria" menubar open
  elif [ -n "${DBUS_SESSION_BUS_ADDRESS:-}" ] && { [ -n "${DISPLAY:-}" ] || [ -n "${WAYLAND_DISPLAY:-}" ]; }; then
    "$install_dir/noria" tray open || echo 'You can open the desktop tray later with: noria tray open' >&2
  fi
fi
printf '\nInstalled. Add your accounts, configure your clients once, and use Codex or Claude normally.\nhttps://noria.ashray.xyz/docs/quickstart/\n'
