'use strict';

const { getDefaultConfigDir } = require('./paths');

function buildEnv(baseEnv = process.env) {
  const env = { ...baseEnv };
  if (!env.SENSECRAFT_CLI_PATH) {
    env.SENSECRAFT_CLI_PATH = getDefaultConfigDir();
  }
  return env;
}

module.exports = {
  buildEnv,
};
