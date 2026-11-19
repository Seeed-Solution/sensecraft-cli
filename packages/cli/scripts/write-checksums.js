'use strict';

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

function sha256File(file) {
  const data = fs.readFileSync(file);
  return crypto.createHash('sha256').update(data).digest('hex');
}

function buildChecksums(distDir, names) {
  const out = {};
  for (const name of names) {
    const file = path.join(distDir, name);
    if (!fs.existsSync(file) || !fs.statSync(file).isFile()) {
      throw new Error(`Missing artifact: ${file}`);
    }
    out[name] = sha256File(file);
  }
  return out;
}

function main() {
  const [distDir, outFile, ...names] = process.argv.slice(2);
  if (!distDir || !outFile || names.length === 0) {
    console.error('usage: write-checksums.js <dist-dir> <out-file> <artifact>...');
    process.exit(1);
  }
  const checksums = buildChecksums(distDir, names);
  fs.writeFileSync(outFile, JSON.stringify(checksums, null, 2) + '\n');
  console.log(`Wrote ${outFile}`);
  for (const name of names) {
    console.log(`  ${checksums[name]}  ${name}`);
  }
}

if (require.main === module) {
  try {
    main();
  } catch (err) {
    console.error(err.message);
    process.exit(1);
  }
}

module.exports = {
  buildChecksums,
};
