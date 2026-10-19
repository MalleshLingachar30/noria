#!/usr/bin/env python3
"""Deploy the exact pushed docs commit to the linked Vercel project.

This is a maintainer-only GitHub Actions utility, never part of the Noria CLI.
It uses Vercel's Git source API directly, without an npm/Node deployment tool.
"""
import json
import os
import re
import sys
import time
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

TEAM = "team_2PPM4u67yxTW3kxgW77QuGHj"
PROJECT = "noria"
REPO = 1425977093
API = "https://api.vercel.com/v13/deployments"


def payload(commit: str) -> bytes:
    if not re.fullmatch(r"[0-9a-fA-F]{40}", commit):
        raise ValueError("NORIA_DEPLOY_COMMIT must be an exact 40-character Git commit SHA")
    return json.dumps({"name": PROJECT, "target": "production", "gitSource": {
        "type": "github", "repoId": REPO, "ref": commit.lower()
    }}).encode()


def request(url: str, token: str, data: bytes | None = None) -> dict:
    headers = {"Authorization": "Bearer " + token, "Accept": "application/json"}
    if data is not None:
        headers["Content-Type"] = "application/json"
    req = Request(url, data=data, headers=headers, method="POST" if data is not None else "GET")
    try:
        with urlopen(req, timeout=25) as response:
            return json.load(response)
    except HTTPError as error:
        # Don't print response bodies: they can contain unrelated account data.
        raise RuntimeError(f"Vercel deployment API returned HTTP {error.code}") from None
    except URLError:
        raise RuntimeError("Could not reach the Vercel deployment API") from None


def main() -> int:
    token = os.environ.get("VERCEL_TOKEN", "")
    if not token:
        raise RuntimeError("VERCEL_TOKEN secret is not configured")
    commit = os.environ.get("NORIA_DEPLOY_COMMIT", "")
    body = payload(commit)
    target = API + "?teamId=" + TEAM
    deployment = request(target, token, body)
    ident = deployment.get("id", "")
    if not re.fullmatch(r"dpl_[a-zA-Z0-9]+", ident):
        raise RuntimeError("Vercel returned an invalid deployment identifier")
    print("Created Noria documentation deployment for", commit[:12], "as", ident, flush=True)
    deadline = time.monotonic() + 240
    while time.monotonic() < deadline:
        result = request(API + "/" + ident + "?teamId=" + TEAM, token)
        state = result.get("readyState", "")
        if state == "READY":
            print("Documentation production deployment ready: https://noria.ashray.xyz", flush=True)
            return 0
        if state in ("ERROR", "CANCELED"):
            raise RuntimeError(f"Documentation deployment ended with state {state}")
        time.sleep(4)
    raise RuntimeError("Documentation deployment did not become ready within the verification window")


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (RuntimeError, ValueError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
