#!/usr/bin/env bash
# Copy Agent Skills into the npm package tree so they ship with
# `@seeed-studio/sensecraft-cli`. npm `files` cannot reach outside packages/cli/,
# so this must run before `npm pack` / `npm publish` (wired as `prepack`).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PKG="$ROOT/packages/cli"

rm -rf "$PKG/skills" "$PKG/sensecraft-cli"

# Copy the skills tree shipped with the CLI.
# `*-legacy/` dirs are archived drafts kept for reference — never ship them.
cp -R "$ROOT/skills" "$PKG/skills"
find "$PKG/skills" -type d -name '*-legacy' -prune -exec rm -rf {} +

# Ship the repo LICENSE next to the package so npm consumers get it.
if [[ -f "$ROOT/LICENSE" ]]; then
  cp "$ROOT/LICENSE" "$PKG/LICENSE"
fi
# packages/cli/INSTALL.md stays in the package.
# Its login section is `auth add`. The repo root guide uses `auth init`.
if [[ -f "$ROOT/README-zh.md" ]]; then
  cp "$ROOT/README-zh.md" "$PKG/README-zh.md"
fi

# Short install-oriented README for the npm page.
cat >"$PKG/README.md" <<'EOF'
# @seeed-studio/sensecraft-cli

The official command-line tool for [SenseCraft](https://sensecraft.seeed.cc). Read sensor data, create models, and publish applications — for developers and AI agents.

Chinese: [README-zh.md](README-zh.md).

## Install

```bash
npx @seeed-studio/sensecraft-cli install
sensecraft-cli --version
```

Requires Node.js ≥ 16. Downloads a platform binary pinned to this package version.

**Full guide** (PATH, Windows, upgrade, uninstall, offline mirror, Cursor Skills): [`INSTALL.md`](INSTALL.md).

## Agent Skills

After install, Skill markdown lives under this package's `skills/` directory:

One skill: `skills/sensecraft-cli/SKILL.md`. Product procedures are in `references/`. `create-app/` is the nested skill for turning a local project into an app.

```bash
SRC="$(npm root -g)/@seeed-studio/sensecraft-cli/skills/sensecraft-cli"
mkdir -p ~/.cursor/skills
ln -sfn "$SRC" ~/.cursor/skills/sensecraft-cli
```

Symlink the directory. Do not copy it. `sensecraft-cli update` refreshes the files behind the link. See `INSTALL.md` §9.

## Config

The npm wrapper sets `SENSECRAFT_CLI_PATH` to:

- macOS / Linux: `~/.config/sensecraft-cli/`
- Windows: `%APPDATA%\sensecraft-cli\`

Override with `SENSECRAFT_CONFIG_FILE` or `SENSECRAFT_CLI_PATH`.

## Version

```bash
sensecraft-cli --version
```

The binary version matches this npm package version.
EOF

echo "OK: synced skills → $PKG/skills"
