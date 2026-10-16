#!/usr/bin/env bash
# Copy Agent Skills into the npm package tree so they ship with
# `@seeed-studio/sensecraft-cli`. npm `files` cannot reach outside packages/cli/,
# so this must run before `npm pack` / `npm publish` (wired as `prepack`).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PKG="$ROOT/packages/cli"

rm -rf "$PKG/skills" "$PKG/sensecraft-cli"

# Copy the skills tree shipped with the CLI.
cp -R "$ROOT/skills" "$PKG/skills"

# Ship the repo LICENSE next to the package so npm consumers get it.
if [[ -f "$ROOT/LICENSE" ]]; then
  cp "$ROOT/LICENSE" "$PKG/LICENSE"
fi
if [[ -f "$ROOT/INSTALL.md" ]]; then
  cp "$ROOT/INSTALL.md" "$PKG/INSTALL.md"
fi

# Short install-oriented README for the npm page.
cat >"$PKG/README.md" <<'EOF'
# @seeed-studio/sensecraft-cli

Cloud CLI for SenseCAP devices and SenseCraft AI (apps, models, files, cloud deploy).

## Install

```bash
npx @seeed-studio/sensecraft-cli install
sensecraft-cli --version
```

Requires Node.js ≥ 16. Downloads a platform binary pinned to this package version.

**Full guide** (PATH, Windows, upgrade/uninstall, offline mirror, Cursor Skills): see [`INSTALL.md`](INSTALL.md) in this package (same file as the repo root).

## Agent Skills

After install, Skill markdown lives under this package's `skills/` directory:

| Skill | Path |
|-------|------|
| Meta router | `skills/sensecraft-ai/SKILL.md` |
| SenseCAP PaaS | `skills/sensecraft-data/` |
| Application Plaza | `skills/sensecraft-apps/` |
| File upload | `skills/sensecraft-files/` |
| Model library | `skills/sensecraft-models/` |
| Classic reCamera 40 | `skills/sensecraft-recamera/` |
| Cloud devices | `skills/sensecraft-devices-cloud/` |

```bash
ls "$(npm root -g)/@seeed-studio/sensecraft-cli/skills"
```

Point your Agent (Cursor, etc.) at these folders, or copy/symlink them into the Agent's skills path. See `skills/README.md` and `INSTALL.md` §9.

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
