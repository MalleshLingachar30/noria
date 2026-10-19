#!/usr/bin/env node
"use strict";
const pkg = require("../package.json");
const tag = process.argv[2];
if (!tag || tag !== `v${pkg.version}` || !/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(tag)) {
  console.error(`Release tag must exactly match package.json: v${pkg.version}`);
  process.exitCode = 1;
}
