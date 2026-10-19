"use strict";
const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const root = path.resolve(__dirname, "..");
const pkg = require("../package.json");
const identity = require("../project.config.json");

test("project identity is consistent across module, npm and binary", () => {
  assert.equal(pkg.name, identity.npmPackage);
  assert.equal(pkg.config.goModule, identity.goModule);
  assert.match(fs.readFileSync(path.join(root, "go.mod"), "utf8"), new RegExp(`^module ${identity.goModule.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\s`, "m"));
  assert.ok(fs.existsSync(path.join(root, `cmd/${identity.cliName}/main.go`)));
  assert.ok(pkg.files.includes("go.sum"), "npm fallback must include dependency checksums");
  assert.ok(pkg.files.includes("internal"), "embedded bridge must ship with source fallback");
});

test("published package cannot accidentally include local artifacts", () => {
  assert.ok(!pkg.files.includes("tmp"));
  assert.ok(!pkg.files.includes(".noria"));
  assert.ok(!pkg.files.includes(".vercel"));
  assert.match(fs.readFileSync(path.join(root, ".gitignore"), "utf8"), /^\/tmp\/$/m);
});

test("local build assets retain both OS and architecture", () => {
  const makefile = fs.readFileSync(path.join(root, "Makefile"), "utf8");
  assert.ok(makefile.includes('$(BIN_NAME)_$${GOOS}_$${GOARCH}$${EXT}'));
  assert.ok(makefile.includes('"windows arm64"'));
  assert.ok(!makefile.includes('$$GOOS_'), "shell would expand nonexistent GOOS_ and overwrite another platform");
});
