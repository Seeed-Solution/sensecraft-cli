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
  console.log('正在全局安装 sensecraft-cli...');
  execSync(`npm install -g ${JSON.stringify(packageRoot)}`, {
    stdio: 'inherit',
    env: process.env,
  });

  const cliPath = getGlobalCliPath();
  if (!fs.existsSync(cliPath)) {
    throw new Error(`全局命令未找到: ${cliPath}，请确认 npm 全局 bin 目录在 PATH 中`);
  }
  return cliPath;
}

function runInitIfNeeded(cliPath) {
  if (isInitialized()) {
    console.log('');
    console.log('✓ 配置文件已存在，跳过初始化');
    console.log('  sensecraft-cli auth info');
    return;
  }

  if (!process.stdin.isTTY || !process.stdout.isTTY) {
    console.log('');
    console.log('未检测到交互式终端，已跳过自动初始化。');
    console.log('请稍后运行: sensecraft-cli auth init');
    return;
  }

  console.log('');
  console.log('未检测到配置文件，开始自动初始化...');
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
  console.log('✓ 安装完成');
  console.log(`  命令: sensecraft-cli`);
  console.log(`  路径: ${globalCliPath}`);
  if (!process.env.PATH || !process.env.PATH.split(path.delimiter).includes(globalBin)) {
    console.log('');
    console.log('提示: 若无法直接运行 sensecraft-cli，请将以下目录加入 PATH:');
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
