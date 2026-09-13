#!/usr/bin/env node
'use strict';
// bin/log-labor.js — thin launcher for the log-labor Go CLI (npm
// wrapper, esbuild/@openai/codex pattern). Node is only the delivery
// truck: after spawn, the binary runs natively and Node exits with its
// status code.
const { spawnSync } = require('child_process');

const PLATFORMS = [
  ['win32', 'x64', 'log-labor.exe'],
  ['darwin', 'arm64', 'log-labor'],
  ['darwin', 'x64', 'log-labor'],
  ['linux', 'arm64', 'log-labor'],
  ['linux', 'x64', 'log-labor'],
];

let bin = null;
for (const [os, cpu, exe] of PLATFORMS) {
  if (os !== process.platform || cpu !== process.arch) continue;
  try { bin = require.resolve(`log-labor-${os}-${cpu}/${exe}`); break; } catch (_) { /* not installed */ }
}

if (!bin) {
  console.error(
    'log-labor: no binary for ' + process.platform + '/' + process.arch +
    ' — reinstall the package (npm i -g log-labor)');
  process.exit(2);
}

const r = spawnSync(bin, process.argv.slice(2), { stdio: 'inherit' });
if (r.error) {
  console.error('log-labor: failed to run ' + bin + ': ' + r.error.message);
  process.exit(2);
}
process.exit(r.status == null ? 2 : r.status);
