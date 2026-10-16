'use strict';

function resolvePlatformTarget() {
  const platform = process.platform;
  const arch = process.arch;

  if (platform === 'darwin' && arch === 'x64') {
    return { artifact: 'sensecraft-cli-darwin-amd64', binary: 'sensecraft-cli' };
  }
  if (platform === 'darwin' && arch === 'arm64') {
    return { artifact: 'sensecraft-cli-darwin-arm64', binary: 'sensecraft-cli' };
  }
  if (platform === 'linux' && arch === 'x64') {
    return { artifact: 'sensecraft-cli-linux-amd64', binary: 'sensecraft-cli' };
  }
  if (platform === 'linux' && arch === 'arm64') {
    return { artifact: 'sensecraft-cli-linux-arm64', binary: 'sensecraft-cli' };
  }
  if (platform === 'win32' && arch === 'x64') {
    return {
      artifact: 'sensecraft-cli-windows-amd64.exe',
      binary: 'sensecraft-cli.exe',
    };
  }
  if (platform === 'win32' && arch === 'arm64') {
    return {
      artifact: 'sensecraft-cli-windows-arm64.exe',
      binary: 'sensecraft-cli.exe',
    };
  }

  throw new Error(
    `Unsupported platform: ${platform}/${arch}. ` +
      'Supported: darwin/linux (x64, arm64) and windows (x64, arm64).'
  );
}

module.exports = {
  resolvePlatformTarget,
};
