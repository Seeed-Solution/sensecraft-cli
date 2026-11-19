'use strict';

const fs = require('fs');
const https = require('https');
const path = require('path');
const crypto = require('crypto');

const {
  VERSION,
  releaseArtifactURL,
  versionStampPath,
  checksumFilePath,
} = require('./constants');
const { getVendorDir, getBinaryPath } = require('./paths');
const { resolvePlatformTarget } = require('./platform');

class ChecksumError extends Error {
  constructor(message) {
    super(message);
    this.name = 'ChecksumError';
    this.code = 'CHECKSUM';
  }
}

function assertDownloadURL(url, allowedHost) {
  let parsed;
  try {
    parsed = new URL(url);
  } catch {
    throw new Error(`Unsupported download URL: ${url}`);
  }
  if (parsed.protocol !== 'https:') {
    throw new Error(`Refusing non-HTTPS download URL: ${url}`);
  }
  if (allowedHost && parsed.host !== allowedHost) {
    throw new Error(`Refusing download host ${parsed.host}`);
  }
  return parsed;
}

function request(url, redirects = 0, allowedHost) {
  return new Promise((resolve, reject) => {
    let parsed;
    try {
      parsed = assertDownloadURL(url, allowedHost);
    } catch (err) {
      reject(err);
      return;
    }
    const permit = allowedHost || parsed.host;
    https
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
          request(next, redirects + 1, permit).then(resolve, reject);
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

function readChecksums() {
  const file = checksumFilePath();
  let parsed;
  try {
    parsed = JSON.parse(fs.readFileSync(file, 'utf8'));
  } catch (err) {
    throw new ChecksumError(`Cannot read release checksums (${file}): ${err.message}`);
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new ChecksumError(`Release checksums file is invalid: ${file}`);
  }
  return parsed;
}

function verifyDownload(artifact, data, checksums) {
  const expected = checksums[artifact];
  if (typeof expected !== 'string' || !/^[0-9a-f]{64}$/.test(expected)) {
    throw new ChecksumError(`Missing checksum for ${artifact}`);
  }
  const got = sha256(data);
  if (got !== expected) {
    throw new ChecksumError(
      `Checksum mismatch for ${artifact}: expected ${expected}, got ${got}`
    );
  }
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
  verifyDownload(artifact, data, readChecksums());

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
  ChecksumError,
  assertDownloadURL,
  verifyDownload,
  sha256,
};
