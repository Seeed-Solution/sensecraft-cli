'use strict';

const path = require('path');

// Single source of truth: packages/cli/package.json "version".
// Release builds upload binaries to ${RELEASE_BASE}/v${VERSION}/...
const VERSION = require('../package.json').version;

const RELEASE_BASE = 'https://sensecraft-statics.seeed.cc/static/bin/sensecraft-cli';

const BINARY_NAME = 'sensecraft-cli';

function releaseArtifactURL(artifact) {
  const base = RELEASE_BASE.replace(/\/+$/, '');
  return `${base}/v${VERSION}/${artifact}`;
}

function versionStampPath(vendorDir) {
  return path.join(vendorDir, '.version');
}

function checksumFilePath() {
  return path.join(__dirname, '..', 'checksums.json');
}

module.exports = {
  VERSION,
  RELEASE_BASE,
  BINARY_NAME,
  releaseArtifactURL,
  versionStampPath,
  checksumFilePath,
};
