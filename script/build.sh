#!/usr/bin/env bash
# Cross-compile sensecraft-cli for every platform the npm installer knows about.
# Run from the repository root: ./script/build.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-}"
if [[ -z "$VERSION" ]]; then
  if git describe --tags --exact-match HEAD >/dev/null 2>&1; then
    VERSION="$(git describe --tags --exact-match HEAD)"
    VERSION="${VERSION#v}"
  else
    VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
  fi
fi
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo none)"
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
LDFLAGS="-s -w \
  -X sensecraft-cloud-terminal/internal/version.Version=${VERSION} \
  -X sensecraft-cloud-terminal/internal/version.Commit=${COMMIT} \
  -X sensecraft-cloud-terminal/internal/version.Date=${DATE}"

OUT_DIR="${OUT_DIR:-$ROOT/dist}"
mkdir -p "$OUT_DIR"

targets=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
  "windows/arm64"
)

for t in "${targets[@]}"; do
  GOOS="${t%/*}"
  GOARCH="${t#*/}"
  name="sensecraft-cli-${GOOS}-${GOARCH}"
  [[ "$GOOS" == "windows" ]] && name="${name}.exe"
  echo "==> building ${name} (version=${VERSION})"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -trimpath -ldflags "$LDFLAGS" -o "$OUT_DIR/$name" .
done

# Convenience symlink for the host platform (no GOOS/GOARCH prefix).
host="sensecraft-cli-$(go env GOOS)-$(go env GOARCH)"
[[ "$(go env GOOS)" == "windows" ]] && host="${host}.exe"
ln -sfn "$host" "$OUT_DIR/sensecraft-cli$( [[ "$(go env GOOS)" == windows ]] && echo .exe )"

echo "OK: binaries in $OUT_DIR (version=${VERSION} commit=${COMMIT})"
