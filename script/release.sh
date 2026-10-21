#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
VERSION="${1:-}"

if [[ -z "${VERSION}" ]]; then
  echo "用法: $0 <version>"
  echo "示例: $0 1.0.0"
  exit 1
fi

if ! command -v node >/dev/null 2>&1; then
  echo "比对版本需要 node" >&2
  exit 1
fi
PKG_VERSION="$(node -p "require('${ROOT_DIR}/packages/cli/package.json').version")"
if [[ "${VERSION}" != "${PKG_VERSION}" ]]; then
  echo "传入版本 ${VERSION} 与 packages/cli/package.json（${PKG_VERSION}）不一致"
  exit 1
fi

TAG="v${VERSION}"

if [[ ! -d "${DIST_DIR}" ]]; then
  echo "dist 目录不存在，请先运行 script/build.sh"
  exit 1
fi

ARTIFACTS=(
  sensecraft-cli-linux-amd64
  sensecraft-cli-linux-arm64
  sensecraft-cli-darwin-amd64
  sensecraft-cli-darwin-arm64
  sensecraft-cli-windows-amd64.exe
)

echo "准备发布 ${TAG}"
for name in "${ARTIFACTS[@]}"; do
  if [[ ! -f "${DIST_DIR}/${name}" ]]; then
    echo "缺少构建产物: ${DIST_DIR}/${name}"
    exit 1
  fi
  echo "  ✓ ${name}"
done

echo ""
echo "请在 GitHub 仓库 sensecraft-cloud-terminal 创建 Release："
echo "  Tag:  ${TAG}"
echo ""
echo "将以下文件上传为 Release 附件（文件名保持不变）："
for name in "${ARTIFACTS[@]}"; do
  echo "  ${DIST_DIR}/${name}"
done
echo ""
echo "发布 npm 包："
echo "  cd packages/cli && npm publish --access public"
