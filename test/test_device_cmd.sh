#!/bin/bash

set -euo pipefail
cd "$(dirname "$0")"

# config.json in this folder is a placeholder template: copy it and fill in real
# credentials outside the repo, then point SENSECRAFT_CONFIG_FILE at that copy.
export SENSECRAFT_CONFIG_FILE="${SENSECRAFT_CONFIG_FILE:-$PWD/config.json}"

"${SENSECRAFT_CLI:-../dist/sensecraft-cli}" device list