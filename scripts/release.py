#!/usr/bin/env python3
"""Version checks and curated release notes for native-only Noria builds."""

from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parent.parent
VERSION = (ROOT / "VERSION").read_text().strip()
CHANGELOG = ROOT / "docs/src/content/docs/changelog.md"
VERSION_PATTERN = re.compile(r"^\d+\.\d+\.\d+$")


def check(tag: str) -> str:
    if not VERSION_PATTERN.fullmatch(VERSION) or tag != "v" + VERSION:
        raise ValueError(f"Release tag must exactly match VERSION: v{VERSION}")
    return VERSION


def notes(tag: str) -> str:
    version = check(tag)
    source = CHANGELOG.read_text()
    match = re.search(r"(?m)^## v?" + re.escape(version) + r"(?:\s+[^\n]*)?\n", source)
    if not match:
        raise ValueError(f"Changelog is missing manual release notes for {version}")
    body = re.split(r"(?m)^## ", source[match.end():], maxsplit=1)[0].strip()
    if not body:
        raise ValueError(f"Empty release notes for {version}")
    body = body.replace("](/docs/", "](https://noria.ashray.xyz/docs/")
    return (
        body
        + "\n\nInstall macOS/Linux: `curl -fsSL https://noria.ashray.xyz/install.sh | sh`"
        + "\n\nInstall Windows: `irm https://noria.ashray.xyz/install.ps1 | iex`"
        + "\n\nUpgrade: `noria upgrade`"
        + "\n\nDocumentation: https://noria.ashray.xyz\n"
    )


def main() -> int:
    if len(sys.argv) != 3 or sys.argv[1] not in ("check", "notes"):
        print("Usage: python3 scripts/release.py check|notes vX.Y.Z", file=sys.stderr)
        return 2
    try:
        if sys.argv[1] == "check":
            check(sys.argv[2])
        else:
            print(notes(sys.argv[2]), end="")
        return 0
    except ValueError as error:
        print(error, file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
