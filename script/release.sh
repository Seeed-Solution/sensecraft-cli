#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
VERSION="${1:-}"

if [[ -z "${VERSION}" ]]; then
  echo "usage: $0 <version>"
  echo "example: $0 1.0.0"
  exit 1
fi

if ! command -v node >/dev/null 2>&1; then
  echo "node is required to compare versions" >&2
  exit 1
fi
PKG_VERSION="$(node -p "require('${ROOT_DIR}/packages/cli/package.json').version")"
if [[ "${VERSION}" != "${PKG_VERSION}" ]]; then
  echo "version ${VERSION} does not match packages/cli/package.json (${PKG_VERSION})"
  exit 1
fi

TAG="v${VERSION}"

if [[ ! -d "${DIST_DIR}" ]]; then
  echo "dist does not exist; run script/build.sh first"
  exit 1
fi

ARTIFACTS=(
  sensecraft-cli-linux-amd64
  sensecraft-cli-linux-arm64
  sensecraft-cli-darwin-amd64
  sensecraft-cli-darwin-arm64
  sensecraft-cli-windows-amd64.exe
)

echo "Preparing release ${TAG}"
for name in "${ARTIFACTS[@]}"; do
  if [[ ! -f "${DIST_DIR}/${name}" ]]; then
    echo "missing build artifact: ${DIST_DIR}/${name}"
    exit 1
  fi
  echo "  ✓ ${name}"
done

echo ""
echo "Create a GitHub Release in the sensecraft-cli repository:"
echo "  Tag:  ${TAG}"
echo ""
echo "Upload these files as Release assets (keep the file names):"
for name in "${ARTIFACTS[@]}"; do
  echo "  ${DIST_DIR}/${name}"
done
echo ""
echo "Publish the npm package:"
echo "  cd packages/cli && npm publish --access public"
