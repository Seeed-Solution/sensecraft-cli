'use strict';

const { execSync, spawnSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const { installBinary } = require('./download');
const { buildEnv } = require('./env');
const { getPackageRoot, isInitialized } = require('./paths');

function getGlobalBinDir() {
  const prefix = execSync('npm prefix -g', {
    encoding: 'utf8',
    env: process.env,
  }).trim();
  if (process.platform === 'win32') {
    return prefix;
  }
  return path.join(prefix, 'bin');
}

function getGlobalCliPath() {
  const binDir = getGlobalBinDir();
  const name = process.platform === 'win32' ? 'sensecraft-cli.cmd' : 'sensecraft-cli';
  return path.join(binDir, name);
}

function installGlobally() {
  const packageRoot = getPackageRoot();
  console.log('Installing sensecraft-cli globally...');
  execSync(`npm install -g ${JSON.stringify(packageRoot)}`, {
    stdio: 'inherit',
    env: process.env,
  });

  const cliPath = getGlobalCliPath();
  if (!fs.existsSync(cliPath)) {
    throw new Error(`Global command not found: ${cliPath}. Add the npm global bin directory to PATH.`);
  }
  return cliPath;
}

function runInitIfNeeded(cliPath) {
  if (isInitialized()) {
    console.log('');
    console.log('Config file already exists; skipped initialization');
    console.log('  sensecraft-cli auth info');
    return;
  }

  if (!process.stdin.isTTY || !process.stdout.isTTY) {
    console.log('');
    console.log('No interactive terminal; skipped automatic initialization.');
    console.log('Run later: sensecraft-cli auth init');
    return;
  }

  console.log('');
  console.log('No config file found; starting initialization...');
  console.log('');

  const result = spawnSync(cliPath, ['auth', 'init'], {
    stdio: 'inherit',
    env: buildEnv(),
    shell: process.platform === 'win32',
  });

  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    process.exit(result.status == null ? 1 : result.status);
  }
}

function printDone(globalCliPath) {
  const globalBin = getGlobalBinDir();
  console.log('');
  console.log('Install complete');
  console.log('  Command: sensecraft-cli');
  console.log(`  Path: ${globalCliPath}`);
  if (!process.env.PATH || !process.env.PATH.split(path.delimiter).includes(globalBin)) {
    console.log('');
    console.log('If sensecraft-cli is not on PATH, add this directory:');
    console.log(`  ${globalBin}`);
  }
  console.log('');
}

async function runInstall() {
  await installBinary();
  const globalCliPath = installGlobally();
  runInitIfNeeded(globalCliPath);
  printDone(globalCliPath);
}

module.exports = {
  runInstall,
  installGlobally,
  runInitIfNeeded,
  getGlobalCliPath,
};
