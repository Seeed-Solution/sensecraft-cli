#!/usr/bin/env node
'use strict';

const fs = require('fs');
const { spawnSync } = require('child_process');

const { installBinary } = require('../scripts/download');
const { buildEnv } = require('../scripts/env');
const { runInstall } = require('../scripts/install');
const { getBinaryPath } = require('../scripts/paths');
const { resolvePlatformTarget } = require('../scripts/platform');

async function ensureBinary() {
  const { binary } = resolvePlatformTarget();
  const binaryPath = getBinaryPath(binary);
  if (fs.existsSync(binaryPath)) {
    return binaryPath;
  }
  const result = await installBinary();
  return result.targetPath;
}

async function runCli(args) {
  const binaryPath = await ensureBinary();
  const result = spawnSync(binaryPath, args, {
    stdio: 'inherit',
    env: buildEnv(),
  });

  if (result.error) {
    console.error(result.error.message);
    process.exit(1);
  }

  process.exit(result.status == null ? 1 : result.status);
}

async function main() {
  const args = process.argv.slice(2);

  if (args[0] === 'install') {
    try {
      await runInstall();
    } catch (err) {
      console.error('');
      console.error('Install failed:', err.message);
      console.error('');
      process.exit(1);
    }
    return;
  }

  await runCli(args);
}

main();
