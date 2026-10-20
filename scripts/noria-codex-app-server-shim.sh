#!/bin/sh
# Optional launcher for applications which spawn `codex app-server` directly.
# Place a copy named `codex` earlier in that application's PATH, or select it
# as the Codex executable in its settings. Never overwrite the real binary.
set -eu

# Resolve the real installed binary explicitly; falling back to `command -v
# codex` here could resolve this shim again, causing infinite recursion.
real_codex="${NORIA_NATIVE_CODEX_BIN:-$HOME/.codex/packages/standalone/current/bin/codex}"
if [ ! -x "$real_codex" ]; then
    echo 'Noria Codex shim: set NORIA_NATIVE_CODEX_BIN to the real Codex executable' >&2
    exit 1
fi
if [ "${1:-}" = app-server ]; then
    export NORIA_NATIVE_CODEX_BIN="$real_codex"
    exec "${NORIA_BINARY:-$HOME/.local/bin/noria}" run codex -- "$@"
fi
exec "$real_codex" "$@"
