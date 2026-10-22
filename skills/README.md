# Skills discovery (Cursor / Agents)

Skills for this repo live under `skills/`.

## Layout

| Path | Auth | Purpose |
|------|------|---------|
| `skills/sensecraft-ai/SKILL.md` | router | Meta: pick skill + auth kind |
| `skills/sensecraft-data/` | `sensecap` | SenseCAP PaaS: devices / groups / data |
| `skills/sensecraft-apps/` | Access Key | Application Plaza (publish / unpublish / delete) |
| `skills/sensecraft-apps/create-app/` | Access Key | Manifest-driven app lifecycle: gather a finished project into `sensecraft-app.json` (source of truth, versioned when no git), one-go create, local-first updates with drift detection (stops before publish) |
| `skills/sensecraft-files/` | Access Key | Upload file and obtain `file_url` |
| `skills/sensecraft-models/` | Access Key | Model library |
| `skills/sensecraft-recamera/` | `sensecap` Access Key | Classic reCamera 40 pick / check / open |
| `skills/sensecraft-devices-cloud/` | Access Key | Jetson / cloud kit deploy |

**Sub-skill naming:** `sensecraft-apps/create-app` is a nested sub-skill inside the `sensecraft-apps` family. The leaf name `create-app` follows the verb-noun rule (`create` + `app`). Top-level folders keep the `sensecraft-*` domain prefix. Document any new exception here before publishing it.

**Archived drafts:** superseded drafts may be kept locally in `*-legacy/` directories. They are not routable skills, are not committed, and are excluded from the npm package by `script/sync-skills-to-npm.sh`.

## Cursor / Agent setup

1. 安装 CLI：见仓库根目录 [`INSTALL.md`](../INSTALL.md) §2 / §9。
2. Point the agent at this repo (or copy/symlink these Skill folders into the agent’s skills path).
3. Prefer loading **`sensecraft-ai`** first for any SenseCraft AI question — it routes to domain skills (including SenseCAP via `sensecraft-data`).
4. **L2 writes:** follow [`confirmation.md`](confirmation.md). Never pass `--yes` on the same turn as the user's first request.
5. Binary: `sensecraft-cli` must be on `PATH` (`npx @seeed-studio/sensecraft-cli install` or build from repo root).

Finding skills after a global npm install:

```bash
ls "$(npm root -g)/@seeed-studio/sensecraft-cli/skills"
```

## Version sync

When adding a CLI command: update the domain skill `SKILL.md` + `commands.md` in the **same change**.
