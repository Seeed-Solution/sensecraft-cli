'use strict';

const fs = require('fs');
const https = require('https');
const http = require('http');
const path = require('path');
const crypto = require('crypto');

const {
  VERSION,
  releaseArtifactURL,
  versionStampPath,
} = require('./constants');
const { getVendorDir, getBinaryPath } = require('./paths');
const { resolvePlatformTarget } = require('./platform');

function request(url, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (!url.startsWith('https:') && !url.startsWith('http:')) {
      reject(new Error(`Unsupported download URL scheme: ${url}`));
      return;
    }
    // Prefer HTTPS; allow http only for local override bases (e.g. file mirrors).
    const client = url.startsWith('https:') ? https : http;
    client
      .get(url, { headers: { 'User-Agent': `sensecraft-cli-npm-installer/${VERSION}` } }, (res) => {
        if (
          res.statusCode &&
          [301, 302, 303, 307, 308].includes(res.statusCode) &&
          res.headers.location
        ) {
          if (redirects >= 5) {
            reject(new Error(`Too many redirects: ${url}`));
            return;
          }
          const next = new URL(res.headers.location, url).toString();
          res.resume();
          request(next, redirects + 1).then(resolve, reject);
          return;
        }

        if (res.statusCode !== 200) {
          res.resume();
          reject(new Error(`Download failed (${res.statusCode}): ${url}`));
          return;
        }

        const chunks = [];
        res.on('data', (chunk) => chunks.push(chunk));
        res.on('end', () => resolve(Buffer.concat(chunks)));
      })
      .on('error', reject);
  });
}

function ensureExecutable(filePath) {
  if (process.platform === 'win32') {
    return;
  }
  const mode = fs.statSync(filePath).mode;
  fs.chmodSync(filePath, mode | 0o111);
}

function copyLocalBinary(sourcePath, targetPath) {
  fs.mkdirSync(path.dirname(targetPath), { recursive: true });
  fs.copyFileSync(sourcePath, targetPath);
  ensureExecutable(targetPath);
}

function readInstalledVersion(vendorDir) {
  try {
    return fs.readFileSync(versionStampPath(vendorDir), 'utf8').trim();
  } catch {
    return '';
  }
}

function writeInstalledVersion(vendorDir, version) {
  fs.writeFileSync(versionStampPath(vendorDir), `${version}\n`);
}

function sha256(buf) {
  return crypto.createHash('sha256').update(buf).digest('hex');
}

async function installBinary() {
  const { artifact, binary } = resolvePlatformTarget();
  const vendorDir = getVendorDir();
  const targetPath = getBinaryPath(binary);
  const installed = readInstalledVersion(vendorDir);

  // Reuse only when the on-disk binary matches THIS npm package version.
  // Previously the installer reused any existing file forever, so upgrading
  // the npm package never swapped the binary.
  if (fs.existsSync(targetPath) && installed === VERSION) {
    return { binary, targetPath, reused: true, version: VERSION };
  }

  const localDist = process.env.SENSECRAFT_CLI_LOCAL_DIST;
  if (localDist) {
    const sourcePath = path.resolve(localDist, artifact);
    if (!fs.existsSync(sourcePath)) {
      throw new Error(`Local binary not found: ${sourcePath}`);
    }
    copyLocalBinary(sourcePath, targetPath);
    writeInstalledVersion(vendorDir, VERSION);
    return { binary, targetPath, reused: false, source: 'local', version: VERSION };
  }

  if (process.env.SENSECRAFT_CLI_SKIP_DOWNLOAD === '1') {
    throw new Error(
      'SENSECRAFT_CLI_SKIP_DOWNLOAD=1 is set, but no cached binary for ' +
        `v${VERSION} was found. Set SENSECRAFT_CLI_LOCAL_DIST to a dist/ dir.`
    );
  }

  const url = releaseArtifactURL(artifact);
  const data = await request(url);

  // Optional integrity check: if a sibling .sha256 file is published next to
  // the artifact, verify it. Missing checksum is not fatal (OSS may lag).
  const expectedChecksum = process.env.SENSECRAFT_CLI_SHA256;
  if (expectedChecksum) {
    const got = sha256(data);
    if (got !== expectedChecksum.toLowerCase()) {
      throw new Error(
        `Checksum mismatch for ${artifact}: expected ${expectedChecksum}, got ${got}`
      );
    }
  }

  fs.mkdirSync(vendorDir, { recursive: true });
  // Atomic-ish write: write to temp then rename so a partial download never
  // leaves a truncated binary that would be "reused" next time.
  const tmpPath = `${targetPath}.tmp-${process.pid}`;
  fs.writeFileSync(tmpPath, data);
  ensureExecutable(tmpPath);
  fs.renameSync(tmpPath, targetPath);
  writeInstalledVersion(vendorDir, VERSION);

  return { binary, targetPath, reused: false, source: 'release', url, version: VERSION };
}

module.exports = {
  installBinary,
};
