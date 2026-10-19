#!/usr/bin/env node
"use strict";
const fs = require("node:fs");
const path = require("node:path");
const version = require("../package.json").version;
const changelog = fs.readFileSync(path.join(__dirname, "../docs/src/content/docs/changelog.md"), "utf8");
const sections = changelog.split(/^## /m).slice(1);
const section = sections.find(value => new RegExp(`^v?${version.replace(/\./g, "\\.")}(?:\\s|$)`).test(value));
if (!section) throw new Error(`Changelog has no manual entry for ${version}`);
const body = section.slice(section.indexOf("\n") + 1).trim()
  .replace(/\]\(\/docs\//g, "](https://noria.ashray.xyz/docs/");
if (!body) throw new Error(`Changelog entry for ${version} is empty`);
process.stdout.write(`${body}\n\nInstall: \`npm install -g @amxv/noria@${version}\`\n\nDocumentation: https://noria.ashray.xyz\n`);
