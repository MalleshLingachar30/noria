"use strict";
const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const crypto = require("node:crypto");
const installer = require("./postinstall");

test("all six supported release assets map correctly", () => {
  for (const [platform, goos] of [["darwin", "darwin"], ["linux", "linux"], ["win32", "windows"]]) {
    for (const [arch, goarch] of [["x64", "amd64"], ["arm64", "arm64"]]) {
      assert.equal(installer.assetForPlatform(platform, arch), `noria_${goos}_${goarch}${platform === "win32" ? ".exe" : ""}`);
    }
  }
  assert.equal(installer.assetForPlatform("freebsd", "x64"), null);
  assert.equal(installer.assetForPlatform("linux", "riscv64"), null);
});

test("repository parser rejects malformed metadata and strips .git", () => {
  assert.deepEqual(installer.repositoryIdentity("git+https://github.com/amxv/noria.git"), { owner: "amxv", repo: "noria" });
  assert.deepEqual(installer.repositoryIdentity("git@github.com:amxv/noria.git"), { owner: "amxv", repo: "noria" });
  assert.throws(() => installer.repositoryIdentity("https://example.com/amxv/noria"));
});

test("checksum matching is exact, supports binary marker, and rejects missing entries", () => {
  const sum = "ab".repeat(32);
  assert.equal(installer.expectedDigest(`${sum}  noria_darwin_arm64\n`, "noria_darwin_arm64"), sum);
  assert.equal(installer.expectedDigest(`${sum} *noria_windows_arm64.exe\r\n`, "noria_windows_arm64.exe"), sum);
  assert.throws(() => installer.expectedDigest(`${sum}  noria_darwin_arm64_extra\n`, "noria_darwin_arm64"));
});

test("a corrupt download cannot pass checksum verification", t => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "noria-installer-test-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  const file = path.join(dir, "binary");
  fs.writeFileSync(file, "complete binary");
  const sum = crypto.createHash("sha256").update("complete binary").digest("hex");
  installer.verifyFile(file, sum);
  fs.writeFileSync(file, "truncated");
  assert.throws(() => installer.verifyFile(file, sum), /checksum mismatch/);
});

test("downloads reject insecure URLs and redirect loops without network access", async () => {
  await assert.rejects(installer.downloadToFile("http://example.com/asset", "/unused"), /HTTPS/);
  await assert.rejects(installer.downloadToFile("https://example.com/asset", "/unused", 6), /redirect/);
});

test("npm source fallback embeds the actual release version", () => {
  assert.match(installer.buildLdflags({ config: { goModule: "github.com/amxv/noria" }, version: "0.1.0" }), /noria\/internal\/buildinfo.Version=0.1.0$/);
});
