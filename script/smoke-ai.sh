#!/usr/bin/env bash
# Sandbox smoke for SenseCraft AI CLI (no real network).
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
      "org_id": "org-sandbox",
      "key_id": "sandbox-key",
      "access_key": "sandbox-access-key"
    }
  ]
}
EOF
export SENSECRAFT_CONFIG_FILE="$CFG"

run() {
  echo "==> $*"
  "$BIN" "$@" >/dev/null
}

# Assert an L2 command refuses to act without --yes.
needs_yes() {
  echo "==> [L2 gate] $*"
  if "$BIN" "$@" >/dev/null 2>&1; then
    echo "FAIL: expected confirmation required for: $*" >&2
    exit 1
  fi
}

run app list --mine --sandbox
run app list --public --uniform-type 40 --sandbox
run app view --id a1 --sandbox
run app fields
run app blocks
run app create --name SmokeApp --sandbox
run app bom search --name xiao --sandbox
run app bom list --id a1 --sandbox

APPJSON="$(mktemp)"
echo '{"app_name":"SmokeApp","scene":1}' >"$APPJSON"
run app update --id a1 --from-json "$APPJSON" --yes --sandbox
rm -f "$APPJSON"

BOMJSON="$(mktemp)"
echo '{"materials":[{"id":"113991254","name":"XIAO ESP32S3 Sense","quantity":"1","note":"","buy_url":"https://www.seeedstudio.com/"}]}' >"$BOMJSON"
run app bom set --id a1 --from-json "$BOMJSON" --yes --sandbox
rm -f "$BOMJSON"

run app publish --id a1 --agree-declaration --yes --sandbox
run app unpublish --id a1 --yes --sandbox
run app delete --id a1 --yes --sandbox

# Console lifecycle: published edits sketch; under-review is locked; delete private-only.
PUBJSON="$(mktemp)"
echo '{"app_name":"PublishedDraft"}' >"$PUBJSON"
echo "==> [published sketch] app update --id sandbox-published-1"
PUBOUT="$("$BIN" app update --id sandbox-published-1 --from-json "$PUBJSON" --yes --sandbox)"
if ! echo "$PUBOUT" | grep -q 'sketch_saved'; then
  echo "FAIL: published app update should mark sketch_saved" >&2
  echo "$PUBOUT" >&2
  rm -f "$PUBJSON"
  exit 1
fi
echo "==> [draft publish] app publish --id sandbox-published-1"
DRAFTOUT="$("$BIN" app publish --id sandbox-published-1 --agree-declaration --yes --sandbox)"
if ! echo "$DRAFTOUT" | grep -q 'draft_publish'; then
  echo "FAIL: publish of already-public app should mark draft_publish" >&2
  echo "$DRAFTOUT" >&2
  rm -f "$PUBJSON"
  exit 1
fi
must_fail() {
  echo "==> [must fail] $*"
  if "$BIN" "$@" >/dev/null 2>&1; then
    echo "FAIL: expected non-zero for: $*" >&2
    exit 1
  fi
}
must_fail app update --id sandbox-review-1 --from-json "$PUBJSON" --yes --sandbox
must_fail app delete --id sandbox-published-1 --yes --sandbox
must_fail app publish --id sandbox-review-1 --agree-declaration --yes --sandbox
must_fail app update --id public-1 --from-json "$PUBJSON" --yes --sandbox
# Console warrantyAccepted: --yes alone is not enough for publish
must_fail app publish --id a1 --yes --sandbox
rm -f "$PUBJSON"

run model list --public --sandbox
run model list --public --uniform-type 40 --adapteds recamera-sku --sandbox
run recamera check --model-id recamera-demo --sandbox
run recamera check --app-id recamera-app --sandbox
run recamera check --app-id recamera-multi --sandbox
run recamera open --model-id recamera-demo --print-only --sandbox
must_fail recamera check --model-id m1 --sandbox
must_fail recamera check --app-id a1 --sandbox
run model view --id m1 --sandbox
run model constants --sandbox
run model download --id m1 --url-only --sandbox

# Public library: redact file_url on stdout; refuse download (incl. --apply / --url-only).
assert_stdout_omits() {
  local needle="$1"
  shift
  echo "==> [no $needle] $*"
  local out
  out="$("$BIN" "$@")"
  if echo "$out" | grep -F -q "$needle"; then
    echo "FAIL: stdout contained $needle for: $*" >&2
    echo "$out" >&2
    exit 1
  fi
}

assert_fails_without_url() {
  echo "==> [public no-download] $*"
  local err
  set +e
  err="$("$BIN" "$@" 2>&1)"
  local code=$?
  set -e
  if [[ "$code" -eq 0 ]]; then
    echo "FAIL: expected non-zero for: $*" >&2
    echo "$err" >&2
    exit 1
  fi
  if echo "$err" | grep -Ei 'https?://'; then
    echo "FAIL: error still contains a URL for: $*" >&2
    echo "$err" >&2
    exit 1
  fi
  if ! echo "$err" | grep -q 'public models do not expose a download URL'; then
    echo "FAIL: missing public-download error for: $*" >&2
    echo "$err" >&2
    exit 1
  fi
}

assert_stdout_omits file_url model list --public --sandbox
assert_stdout_omits file_url model view --id public-1 --sandbox
echo "==> [private view keeps file_url] model view --id m1 --sandbox"
PRIVVIEW="$("$BIN" model view --id m1 --sandbox)"
if ! echo "$PRIVVIEW" | grep -F -q '"file_url"'; then
  echo "FAIL: private model view should keep file_url" >&2
  echo "$PRIVVIEW" >&2
  exit 1
fi
assert_fails_without_url model download --id public-1 --sandbox
assert_fails_without_url model download --id public-1 --apply --sandbox
assert_fails_without_url model download --id public-1 --url-only --sandbox

echo "==> [private download omits url] model download --id m1 --out"
PRIVOUT="$(mktemp -d)"
PRIVJSON="$("$BIN" model download --id m1 --out "$PRIVOUT" --sandbox)"
if echo "$PRIVJSON" | grep -Eq '"url"[[:space:]]*:'; then
  echo "FAIL: private file download JSON must omit url" >&2
  echo "$PRIVJSON" >&2
  rm -rf "$PRIVOUT"
  exit 1
fi
if ! echo "$PRIVJSON" | grep -q '"path"'; then
  echo "FAIL: private file download JSON missing path" >&2
  echo "$PRIVJSON" >&2
  rm -rf "$PRIVOUT"
  exit 1
fi
rm -rf "$PRIVOUT"
run model recamera-pro resolve --task 1 --ai-framework 2
run model recamera-pro resolve --task 2
run model recamera-pro resolve --task 4 --ai-framework 2
run model recamera-pro resolve --task 6
PROJSON="$(mktemp)"
cat >"$PROJSON" <<'JSON'
{
  "name": "ProSmoke",
  "description": "Sandbox Pro model",
  "content": "<p>Sandbox Pro model details</p>",
  "pic_url": "https://sandbox.example.com/pro.png",
  "adapteds": ["sandbox-pro"],
  "task": 2,
  "model_format": 6,
  "precision": "1",
  "scenario": "1",
  "version": "1.0.0",
  "checksum": "sandbox-pro-md5",
  "file_url": "https://sandbox.example.com/model.rknn",
  "labels": [{ "object_id": 0, "object_name": "cat" }]
}
JSON
run model create --from-json "$PROJSON" --target recamera-pro --yes --sandbox
rm -f "$PROJSON"

MODELJSON="$(mktemp)"
cat >"$MODELJSON" <<'JSON'
{
  "name": "SmokeModel",
  "description": "Sandbox model",
  "content": "<p>Sandbox model details</p>",
  "pic_url": "https://sandbox.example.com/model.png",
  "adapteds": ["sandbox-vision"],
  "model_format": 2,
  "task": 1,
  "scenario": "1",
  "version": "1.0.0",
  "precision": "1",
  "checksum": "sandbox-model-md5",
  "file_url": "https://sandbox.example.com/model.tflite",
  "labels": [{ "object_id": 0, "object_name": "smoke" }]
}
JSON
run model create --from-json "$MODELJSON" --yes --sandbox
run model update --id m1 --from-json "$MODELJSON" --yes --sandbox
rm -f "$MODELJSON"

run model publish --id m1 --yes --sandbox
run model unpublish --id m1 --yes --sandbox
run model delete --id m1 --yes --sandbox

TMPF="$(mktemp -t sensecraft-upload)"
mv "$TMPF" "${TMPF}.tflite"
TMPF="${TMPF}.tflite"
echo hi >"$TMPF"
# Types 1–2 require attach or explicit --unlinked (orphan gate).
must_fail file upload --path "$TMPF" --type 2 --checksum --sandbox
must_fail model upload --path "$TMPF" --sandbox
run file upload --path "$TMPF" --type 2 --checksum --unlinked --sandbox
run model upload --path "$TMPF" --unlinked --sandbox
run model upload --path "$TMPF" --id mine-1 --yes --sandbox
run file upload --path "$TMPF" --type 4 --checksum --sandbox
rm -f "$TMPF"

run cloud device list --uniform-type 26 --sandbox
run cloud device view --eui SANDBOXEUI000001 --sandbox
run cloud deploy --eui SANDBOXEUI000001 --model-id m1 --yes --sandbox
run cloud order --request-id sandbox-deploy-req --device-uuid sandbox-uuid --sandbox

# Every L2 command must refuse to act when --yes is absent.
GATEJSON="$(mktemp)"
echo '{"app_name":"SmokeApp"}' >"$GATEJSON"
needs_yes app delete --id a1 --sandbox
needs_yes app update --id a1 --from-json "$GATEJSON" --sandbox
needs_yes app publish --id a1 --agree-declaration --sandbox
needs_yes app unpublish --id a1 --sandbox
needs_yes app bom set --id a1 --from-json "$GATEJSON" --sandbox
needs_yes model create --from-json "$GATEJSON" --sandbox
needs_yes model update --id m1 --from-json "$GATEJSON" --sandbox
needs_yes model publish --id m1 --sandbox
needs_yes model unpublish --id m1 --sandbox
needs_yes model delete --id m1 --sandbox
needs_yes cloud deploy --eui SANDBOXEUI000001 --model-id m1 --sandbox
rm -f "$GATEJSON"

# app create is deliberately NOT an L2 op: full --from-json must work without --yes.
CREATEJSON="$(mktemp)"
echo '{"app_name":"SmokeFullCreate","scene":1,"introduction":"smoke"}' >"$CREATEJSON"
run app create --from-json "$CREATEJSON" --sandbox
rm -f "$CREATEJSON"

echo "OK: AI sandbox smoke passed"
