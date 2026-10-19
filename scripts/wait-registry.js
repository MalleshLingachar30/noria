#!/usr/bin/env node
"use strict";

// npm may accept a first publication before its anonymous read path is ready.
// Wait only in the release verifier; never republish or change the version.
const pkg = require("../package.json");
const { setTimeout: delay } = require("node:timers/promises");

async function main() {
  const version = process.argv[2] || pkg.version;
  const deadline = Date.now() + 5 * 60_000;
  let last = "metadata unavailable";
  while (Date.now() < deadline) {
    try {
      const response = await fetch(
        `https://registry.npmjs.org/${encodeURIComponent(pkg.name)}?noria_verify=${Date.now()}`,
        { headers: { "Cache-Control": "no-cache" }, signal: AbortSignal.timeout(15_000) }
      );
      if (response.ok) {
        const metadata = await response.json();
        if (metadata.versions?.[version]?.dist?.tarball) {
          console.log(`${pkg.name}@${version} is visible to anonymous registry clients.`);
          return;
        }
        last = "version is still processing";
      } else {
        last = `registry returned HTTP ${response.status}`;
        if (response.status === 401 || response.status === 403) {
          throw new Error(`Anonymous package access was denied: ${response.status}`);
        }
      }
    } catch (error) {
      if (String(error.message).startsWith("Anonymous package access")) throw error;
      last = error.message;
    }
    console.log(`Waiting for registry visibility: ${last}.`);
    await delay(5000);
  }
  throw new Error(`Publication did not become readable: ${last}. Rerun verification after processing; do not republish.`);
}

main().catch(error => { console.error(error.message); process.exitCode = 1; });
