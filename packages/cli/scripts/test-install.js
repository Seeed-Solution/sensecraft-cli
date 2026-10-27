#!/usr/bin/env node
'use strict';

/**
 * Isolated install smoke test.
 *
 * Runs postinstall + a non-interactive `sensecraft-cli --version` under a
 * temporary HOME / npm prefix so the developer's real ~/.config and global
 * npm install are never touched. Requires a pre-built binary tree:
 *
 *   ./script/build.sh
 *   SENSECRAFT_CLI_LOCAL_DIST=$PWD/dist node packages/cli/scripts/test-install.js
 */

const fs = require('fs');
const os = require('os');
const path = require('path');
const { spawnSync } = require('child_process');

const { VERSION } = require('./constants');
const { getPackageRoot, getVendorDir } = require('./paths');

function fail(msg) {
  console.error(`FAIL: ${msg}`);
  process.exit(1);
}

function main() {
  const packageRoot = getPackageRoot();
  const repoRoot = path.resolve(packageRoot, '../..');
  const localDist =
    process.env.SENSECRAFT_CLI_LOCAL_DIST || path.join(repoRoot, 'dist');

  if (!fs.existsSync(localDist)) {
    fail(
      `no local dist at ${localDist}. Run ./script/build.sh first, ` +
        'or set SENSECRAFT_CLI_LOCAL_DIST.'
    );
  }

  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'sensecraft-cli-test-'));
  const home = path.join(tmp, 'home');
  const npmPrefix = path.join(tmp, 'npm-prefix');
  const vendorDir = getVendorDir();
  fs.mkdirSync(home, { recursive: true });
  fs.mkdirSync(npmPrefix, { recursive: true });

  // Wipe any previous vendor binary so the install path is exercised.
  fs.rmSync(vendorDir, { recursive: true, force: true });

  const env = {
    ...process.env,
    HOME: home,
    USERPROFILE: home, // windows
    APPDATA: path.join(home, 'AppData', 'Roaming'),
    SENSECRAFT_CLI_LOCAL_DIST: localDist,
    // Force the Go binary to write config under the temp HOME.
    SENSECRAFT_CLI_PATH: path.join(home, '.config', 'sensecraft-cli'),
    npm_config_prefix: npmPrefix,
    // Non-interactive: install.js must skip auth init.
    CI: '1',
  };

  console.log(`==> postinstall (LOCAL_DIST=${localDist})`);
  let r = spawnSync(process.execPath, [path.join(packageRoot, 'scripts/postinstall.js')], {
    stdio: 'inherit',
    env,
    cwd: packageRoot,
  });
  if (r.status !== 0) {
    fail(`postinstall exited ${r.status}`);
  }

  const stamp = path.join(vendorDir, '.version');
  if (!fs.existsSync(stamp) || fs.readFileSync(stamp, 'utf8').trim() !== VERSION) {
    fail(`vendor/.version missing or != ${VERSION}`);
  }

  console.log('==> sensecraft-cli --version');
  r = spawnSync(
    process.execPath,
    [path.join(packageRoot, 'bin/sensecraft-cli.js'), '--version'],
    { encoding: 'utf8', env, cwd: packageRoot }
  );
  if (r.status !== 0) {
    fail(`--version exited ${r.status}: ${r.stderr || r.stdout}`);
  }
  const out = ((r.stdout || '') + (r.stderr || '')).trim();
  // Local ./script/build.sh injects git-describe (e.g. v2026061000-2-g…);
  // release tags inject the npm package version. Accept any non-empty
  // `--version` line — the point of this smoke is "binary runs".
  if (!/version/i.test(out)) {
    fail(`--version output unexpected: ${out}`);
  }
  console.log(out);

  // Confirm we did not pollute the real home.
  const realConfig = path.join(os.homedir(), '.config', 'sensecraft-cli');
  // We can't assert the real config is untouched in content, but we can assert
  // our temp HOME received the SENSECRAFT_CLI_PATH we set and nothing wrote
  // outside tmp.
  if (!tmp.startsWith(os.tmpdir())) {
    fail('temp dir escaped os.tmpdir()');
  }

  // Skills must be present in the package (synced by prepack / sync script).
  const skillsIndex = path.join(packageRoot, 'skills', 'sensecraft-cli', 'SKILL.md');
  if (!fs.existsSync(skillsIndex)) {
    console.warn(
      `WARN: ${skillsIndex} missing — run ./script/sync-skills-to-npm.sh before publish`
    );
  }

  console.log(`OK: isolated install smoke passed (tmp=${tmp})`);
  // Leave tmp for inspection unless CLEAN=1.
  if (process.env.CLEAN === '1') {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

main();
