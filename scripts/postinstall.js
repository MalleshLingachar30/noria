#!/usr/bin/env node
"use strict";

const fs = require("node:fs");
const path = require("node:path");
const https = require("node:https");
const crypto = require("node:crypto");
const { pipeline } = require("node:stream/promises");
const { spawnSync } = require("node:child_process");

const MAX_REDIRECTS = 5;
const TIMEOUT_MS = 60_000;
const MAX_DOWNLOAD_BYTES = 128 * 1024 * 1024;

function assetForPlatform(platform, arch, binary = "noria") {
  const goos = { darwin: "darwin", linux: "linux", win32: "windows" }[platform];
  const goarch = { x64: "amd64", arm64: "arm64" }[arch];
  return goos && goarch ? `${binary}_${goos}_${goarch}${goos === "windows" ? ".exe" : ""}` : null;
}

function repositoryIdentity(repositoryURL) {
  const match = String(repositoryURL || "").match(
    /github\.com[:/]([A-Za-z0-9_.-]+)\/([A-Za-z0-9_.-]+?)(?:\.git)?$/
  );
  if (!match) throw new Error("package.json must identify a GitHub repository");
  return { owner: match[1], repo: match[2] };
}

function buildLdflags(pkg) {
  return `-s -w -X ${pkg.config.goModule}/internal/buildinfo.Version=${pkg.version}`;
}

function expectedDigest(checksums, assetName) {
  for (const line of checksums.split(/\r?\n/)) {
    const match = line.trim().match(/^([a-fA-F0-9]{64})\s+\*?(.+)$/);
    if (match && match[2] === assetName) return match[1].toLowerCase();
  }
  throw new Error(`Release checksum is missing for ${assetName}`);
}

function verifyFile(filePath, checksum) {
  const actual = crypto.createHash("sha256").update(fs.readFileSync(filePath)).digest("hex");
  if (actual !== checksum) throw new Error("Release binary checksum mismatch");
}

function downloadToFile(url, destination, redirects = 0) {
  const target = new URL(url);
  if (target.protocol !== "https:") return Promise.reject(new Error("Release downloads must use HTTPS"));
  if (redirects > MAX_REDIRECTS) return Promise.reject(new Error("Too many download redirects"));
  return new Promise((resolve, reject) => {
    const request = https.get(target, { headers: { "User-Agent": "noria-installer" } }, response => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        response.resume();
        downloadToFile(new URL(response.headers.location, target).href, destination, redirects + 1)
          .then(resolve, reject);
        return;
      }
      if (response.statusCode !== 200) {
        response.resume();
        reject(new Error(`Release download returned HTTP ${response.statusCode}`));
        return;
      }
      let size = 0;
      response.on("data", chunk => {
        size += chunk.length;
        if (size > MAX_DOWNLOAD_BYTES) response.destroy(new Error("Release download is unexpectedly large"));
      });
      pipeline(response, fs.createWriteStream(destination, { flags: "wx", mode: 0o600 }))
        .then(resolve, reject);
    });
    request.setTimeout(TIMEOUT_MS, () => request.destroy(new Error("Release download timed out")));
    request.on("error", reject);
  });
}

function fallbackBuild(pkg, root, destination) {
  if (process.env.NORIA_REQUIRE_PREBUILT === "1") {
    throw new Error("A verified prebuilt release is required; source fallback is disabled");
  }
  const cli = pkg.config.cliName;
  if (spawnSync("go", ["version"], { stdio: "ignore" }).status !== 0 ||
      !fs.existsSync(path.join(root, "cmd", cli, "main.go"))) {
    throw new Error("No usable release binary or Go source-build fallback. Install Go or retry npm installation.");
  }
  const result = spawnSync("go", ["build", "-trimpath", `-ldflags=${buildLdflags(pkg)}`,
    "-o", destination, `./cmd/${cli}`], { cwd: root, stdio: "inherit" });
  if (result.status !== 0) throw new Error("Building Noria from source failed");
}

async function main() {
  const root = path.resolve(__dirname, "..");
  const pkg = require(path.join(root, "package.json"));
  const { owner, repo } = repositoryIdentity(pkg.repository?.url);
  const binary = pkg.config.cliBinaryName;
  const binDir = path.join(root, "bin");
  const destination = path.join(binDir, process.platform === "win32" ? `${binary}.exe` : `${binary}-bin`);
  fs.mkdirSync(binDir, { recursive: true });
  const temporary = fs.mkdtempSync(path.join(binDir, ".noria-install-"));
  const staged = path.join(temporary, "binary");
  const checksumFile = path.join(temporary, "SHA256SUMS");
  const asset = assetForPlatform(process.platform, process.arch, binary);
  try {
    let downloaded = false;
    if (asset) {
      const base = `https://github.com/${owner}/${repo}/releases/download/v${pkg.version}`;
      try {
        await downloadToFile(`${base}/SHA256SUMS`, checksumFile);
        const checksum = expectedDigest(fs.readFileSync(checksumFile, "utf8"), asset);
        await downloadToFile(`${base}/${asset}`, staged);
        verifyFile(staged, checksum);
        downloaded = true;
      } catch (error) {
        console.warn(`Prebuilt Noria unavailable: ${error.message}. Trying a local Go build.`);
        fs.rmSync(staged, { force: true });
      }
    }
    if (!downloaded) fallbackBuild(pkg, root, staged);
    if (process.platform !== "win32") fs.chmodSync(staged, 0o755);
    // The staged file is complete and verified before replacing an existing install.
    fs.renameSync(staged, destination);
    console.log(`Installed ${pkg.config.cliName} ${pkg.version} (${downloaded ? "verified release binary" : "local Go build"}).`);
  } finally {
    fs.rmSync(temporary, { recursive: true, force: true });
  }
}

module.exports = { assetForPlatform, repositoryIdentity, buildLdflags, expectedDigest, verifyFile, downloadToFile };
if (require.main === module) main().catch(error => {
  console.error(`Noria installation failed: ${error.message}`);
  process.exitCode = 1;
});
