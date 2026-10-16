'use strict';

const fs = require('fs');
const os = require('os');
const path = require('path');

const { BINARY_NAME } = require('./constants');

function getPackageRoot() {
  return path.resolve(__dirname, '..');
}

function getVendorDir() {
  return path.join(getPackageRoot(), 'vendor');
}

function getBinaryPath(binaryFileName) {
  return path.join(getVendorDir(), binaryFileName);
}

function getDefaultConfigDir() {
  if (process.platform === 'win32') {
    const appData = process.env.APPDATA || path.join(os.homedir(), 'AppData', 'Roaming');
    return path.join(appData, BINARY_NAME);
  }
  return path.join(os.homedir(), '.config', BINARY_NAME);
}

/**
 * Resolve the config file the Go binary will actually read.
 * Honours the same env vars as internal/config/path.go:
 *   SENSECRAFT_CONFIG_FILE  → absolute/relative file path
 *   SENSECRAFT_CLI_PATH     → config directory (file = <dir>/sensecraft.json)
 * npm wrappers inject SENSECRAFT_CLI_PATH via buildEnv(), so the default for
 * users of the npm package is ~/.config/sensecraft-cli/sensecraft.json.
 */
function resolveConfigPath() {
  if (process.env.SENSECRAFT_CONFIG_FILE) {
    return path.resolve(process.env.SENSECRAFT_CONFIG_FILE);
  }
  const dir = process.env.SENSECRAFT_CLI_PATH || getDefaultConfigDir();
  return path.join(dir, 'sensecraft.json');
}

/**
 * True when a SenseCAP Access Key profile is present.
 * That key is also used for SenseCraft AI /openapi.
 */
function isInitialized() {
  const configPath = resolveConfigPath();
  try {
    const cfg = JSON.parse(fs.readFileSync(configPath, 'utf8'));
    return Array.isArray(cfg.app_config_list) && cfg.app_config_list.length > 0;
  } catch {
    return false;
  }
}

module.exports = {
  getPackageRoot,
  getVendorDir,
  getBinaryPath,
  getDefaultConfigDir,
  resolveConfigPath,
  isInitialized,
};
