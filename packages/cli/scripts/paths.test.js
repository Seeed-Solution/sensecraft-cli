#!/usr/bin/env node
'use strict';

const assert = require('assert');
const fs = require('fs');
const os = require('os');
const path = require('path');

const { VERSION, releaseArtifactURL } = require('./constants');
const { isInitialized, resolveConfigPath } = require('./paths');

function withEnv(patch, fn) {
  const saved = {};
  for (const k of Object.keys(patch)) {
    saved[k] = process.env[k];
    if (patch[k] === undefined) delete process.env[k];
    else process.env[k] = patch[k];
  }
  try {
    return fn();
  } finally {
    for (const k of Object.keys(patch)) {
      if (saved[k] === undefined) delete process.env[k];
      else process.env[k] = saved[k];
    }
  }
}

function testReleaseURL() {
  const url = releaseArtifactURL('sensecraft-cli-darwin-arm64');
  assert.ok(url.startsWith('https://sensecraft-statics.seeed.cc/static/bin/sensecraft-cli/'), url);
  assert.ok(url.includes(`/v${VERSION}/sensecraft-cli-darwin-arm64`), url);
  assert.equal(VERSION, require('../package.json').version);
}

function testResolveConfigPath() {
  withEnv(
    {
      SENSECRAFT_CONFIG_FILE: '/tmp/custom.json',
      SENSECRAFT_CLI_PATH: undefined,
    },
    () => {
      assert.equal(resolveConfigPath(), path.resolve('/tmp/custom.json'));
    }
  );
  withEnv(
    {
      SENSECRAFT_CONFIG_FILE: undefined,
      SENSECRAFT_CLI_PATH: '/tmp/cfgdir',
    },
    () => {
      assert.equal(resolveConfigPath(), path.join('/tmp/cfgdir', 'sensecraft.json'));
    }
  );
}

function testIsInitialized() {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'sc-init-'));
  const cfg = path.join(tmp, 'sensecraft.json');
  fs.writeFileSync(
    cfg,
    JSON.stringify({
      use_ai: 'p1',
      ai_profiles: [{ profile_id: 'p1', access_token: 't' }],
    })
  );
  withEnv({ SENSECRAFT_CONFIG_FILE: cfg, SENSECRAFT_CLI_PATH: undefined }, () => {
    assert.equal(isInitialized(), false, 'token-only config is not initialized');
  });

  fs.writeFileSync(cfg, JSON.stringify({ app_config_list: [] }));
  withEnv({ SENSECRAFT_CONFIG_FILE: cfg }, () => {
    assert.equal(isInitialized(), false, 'empty app_config_list alone is not initialized');
  });

  fs.writeFileSync(
    cfg,
    JSON.stringify({
      app_config_list: [{ key_id: 'k', access_key: 'a' }],
    })
  );
  withEnv({ SENSECRAFT_CONFIG_FILE: cfg }, () => {
    assert.equal(isInitialized(), true);
  });

  fs.rmSync(tmp, { recursive: true, force: true });
}

testReleaseURL();
testResolveConfigPath();
testIsInitialized();
console.log('OK: packages/cli/scripts/paths.test.js');
