#!/usr/bin/env node
'use strict';

/**
 * Unit tests for binary install version pinning.
 * Exercises the upgrade path: a stale vendor/.version must force reinstall.
 */

const assert = require('assert');
const fs = require('fs');
const os = require('os');
const path = require('path');

const { VERSION } = require('./constants');
const { installBinary } = require('./download');
const { getVendorDir, getBinaryPath } = require('./paths');
const { resolvePlatformTarget } = require('./platform');

async function withTempLocalDist(fn) {
  const { artifact, binary } = resolvePlatformTarget();
  const dist = fs.mkdtempSync(path.join(os.tmpdir(), 'sc-dist-'));
  const src = path.join(dist, artifact);
  fs.writeFileSync(src, 'FAKE-BINARY-CONTENT-v' + VERSION);

  const vendorDir = getVendorDir();
  const targetPath = getBinaryPath(binary);
  const stampPath = path.join(vendorDir, '.version');

  // Snapshot / wipe vendor so tests don't fight a real install.
  const backup = fs.mkdtempSync(path.join(os.tmpdir(), 'sc-vendor-bak-'));
  if (fs.existsSync(vendorDir)) {
    fs.cpSync(vendorDir, path.join(backup, 'vendor'), { recursive: true });
  }
  fs.rmSync(vendorDir, { recursive: true, force: true });

  const prev = process.env.SENSECRAFT_CLI_LOCAL_DIST;
  process.env.SENSECRAFT_CLI_LOCAL_DIST = dist;
  try {
    await fn({ artifact, binary, dist, src, vendorDir, targetPath, stampPath });
  } finally {
    if (prev === undefined) delete process.env.SENSECRAFT_CLI_LOCAL_DIST;
    else process.env.SENSECRAFT_CLI_LOCAL_DIST = prev;
    fs.rmSync(vendorDir, { recursive: true, force: true });
    const bak = path.join(backup, 'vendor');
    if (fs.existsSync(bak)) {
      fs.cpSync(bak, vendorDir, { recursive: true });
    }
    fs.rmSync(backup, { recursive: true, force: true });
    fs.rmSync(dist, { recursive: true, force: true });
  }
}

async function testFreshInstallWritesVersionStamp() {
  await withTempLocalDist(async ({ targetPath, stampPath }) => {
    const r = await installBinary();
    assert.equal(r.reused, false);
    assert.equal(r.source, 'local');
    assert.equal(r.version, VERSION);
    assert.ok(fs.existsSync(targetPath));
    assert.equal(fs.readFileSync(stampPath, 'utf8').trim(), VERSION);
  });
}

async function testReuseWhenVersionMatches() {
  await withTempLocalDist(async ({ targetPath, stampPath }) => {
    await installBinary();
    const first = fs.readFileSync(targetPath);
    const r = await installBinary();
    assert.equal(r.reused, true);
    assert.deepEqual(fs.readFileSync(targetPath), first);
    assert.equal(fs.readFileSync(stampPath, 'utf8').trim(), VERSION);
  });
}

async function testStaleVersionForcesReinstall() {
  await withTempLocalDist(async ({ targetPath, stampPath, src }) => {
    await installBinary();
    // Simulate an older npm package left a binary + stamp behind.
    fs.writeFileSync(stampPath, '0.0.1\n');
    fs.writeFileSync(targetPath, 'OLD-BINARY');
    fs.writeFileSync(src, 'NEW-BINARY-AFTER-UPGRADE');

    const r = await installBinary();
    assert.equal(r.reused, false, 'stale .version must not reuse');
    assert.equal(fs.readFileSync(targetPath, 'utf8'), 'NEW-BINARY-AFTER-UPGRADE');
    assert.equal(fs.readFileSync(stampPath, 'utf8').trim(), VERSION);
  });
}

async function testMissingStampForcesReinstall() {
  await withTempLocalDist(async ({ targetPath, stampPath }) => {
    await installBinary();
    fs.rmSync(stampPath);
    fs.writeFileSync(targetPath, 'ORPHAN-BINARY');
    const r = await installBinary();
    assert.equal(r.reused, false, 'missing .version must not reuse');
    assert.equal(fs.readFileSync(stampPath, 'utf8').trim(), VERSION);
  });
}

(async () => {
  await testFreshInstallWritesVersionStamp();
  await testReuseWhenVersionMatches();
  await testStaleVersionForcesReinstall();
  await testMissingStampForcesReinstall();
  console.log('OK: packages/cli/scripts/download.test.js');
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
