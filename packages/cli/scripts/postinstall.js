'use strict';

const { VERSION, RELEASE_BASE } = require('./constants');
const { installBinary } = require('./download');

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
    console.error('');
    console.error('Try:');
    console.error(
      `  1. Confirm Release / OSS has the v${VERSION} artifact under:`
    );
    console.error(`     ${RELEASE_BASE.replace(/\/+$/, '')}/v${VERSION}/`);
    console.error(
      '  2. Set SENSECRAFT_CLI_LOCAL_DIST to a local dist/ directory and reinstall'
    );
    console.error(
      '  3. Set SENSECRAFT_CLI_RELEASE_BASE to override the download base URL'
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
