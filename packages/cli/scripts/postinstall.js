'use strict';

const { VERSION, RELEASE_BASE } = require('./constants');
const { installBinary, ChecksumError } = require('./download');

async function main() {
  try {
    const result = await installBinary();
    if (result.reused) {
      console.log(`sensecraft-cli v${VERSION} already installed (reused cache)`);
    } else {
      console.log(
        `sensecraft-cli v${VERSION} installed from ${result.source || 'release'}`
      );
    }
  } catch (err) {
    console.error('');
    console.error(`sensecraft-cli binary download failed: ${err.message}`);
    if (err instanceof ChecksumError) {
      console.error('The download does not match the checksum shipped in this package.');
      console.error('');
      process.exit(1);
    }
    console.error('');
    console.error('Try:');
    console.error(
      `  1. Confirm the v${VERSION} artifact is published under:`
    );
    console.error(`     ${RELEASE_BASE}/v${VERSION}/`);
    console.error(
      '  2. Set SENSECRAFT_CLI_LOCAL_DIST to a local dist/ directory and reinstall'
    );
    console.error('');
    // Don't fail the whole npm install on a transient download error when the
    // binary can still be fetched on first invocation (bin/sensecraft-cli.js
    // calls ensureBinary). Exit 0 so `npm install` succeeds; the wrapper will
    // retry. Set SENSECRAFT_CLI_STRICT_POSTINSTALL=1 to make this fatal.
    if (process.env.SENSECRAFT_CLI_STRICT_POSTINSTALL === '1') {
      process.exit(1);
    }
  }
}

main();
