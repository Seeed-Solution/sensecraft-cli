#!/usr/bin/env bash
# Sandbox smoke for SenseCAP OpenAPI CLI (no real network).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="${SENSECRAFT_CLI_BIN:-}"
OWN_BIN=0
if [[ -z "$BIN" ]]; then
  BIN="$(mktemp -t sensecraft-cli.XXXXXX)"
  (cd "$ROOT" && go build -o "$BIN" .)
  OWN_BIN=1
fi

CFG="$(mktemp)"
cleanup() {
  rm -f "$CFG"
  if [[ "$OWN_BIN" -eq 1 ]]; then
    rm -f "$BIN"
  fi
}
trap cleanup EXIT

cat >"$CFG" <<'EOF'
{
  "use_api": "sandbox-key",
  "app_config_list": [
    {
      "remote_type": "Global",
      "label": "sandbox",
      "org_id": "sandbox-org",
      "key_id": "sandbox-key",
      "access_key": "sandbox-access-key",
      "access_level": "read-write",
      "api_status": "active"
    }
  ]
}
EOF
export SENSECRAFT_CONFIG_FILE="$CFG"

run() {
  echo "==> $*"
  "$BIN" "$@" >/dev/null
}

run device list --sandbox
run group list --sandbox
run data latest --eui SANDBOXEUI000001 --sandbox

# L2 SenseCAP writes must refuse without --yes (no network; gate fails before API).
needs_yes() {
  echo "==> (expect fail) $*"
  if "$BIN" "$@" >/dev/null 2>&1; then
    echo "ERROR: expected confirmation failure for: $*" >&2
    exit 1
  fi
}
needs_yes device unbind --eui SANDBOXEUI000001
needs_yes group delete --group-uuid sandbox-group-1
needs_yes group move-devices --group-uuid sandbox-group-1 --eui SANDBOXEUI000001

echo "OK: SenseCAP sandbox smoke passed"
