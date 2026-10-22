---
name: sensecraft-ai
description: >-
  Routes SenseCraft AI platform work for Agents: Application Plaza, model library,
  model upload, classic reCamera deploy page, and cloud deploy vs SenseCAP IoT CLI.
  Use when the user mentions SenseCraft AI, 应用广场, App Square, 模型库, 上传模型,
  reCamera, aiserverapi, or asks which SenseCraft skill/CLI profile to use.
  Does not replace domain skills (apps/models/files/recamera) or sensecraft-data.
---

# SenseCraft AI (Meta)

Router skill for **SenseCraft AI** capabilities exposed via CLI in this repo (`sensecraft-cli`). Domain skills and AI subcommands are rolled out in phases; this skill always applies for **intent routing, auth kind selection, and product-term disambiguation**.

Sibling (already shipped): [`../sensecraft-data/SKILL.md`](../sensecraft-data/SKILL.md) — SenseCAP PaaS OpenAPI (`device` / `group` / `data`).

## Agent rules

1. **Pick the auth kind before any cloud call**
   - `app` / `model` / `file` / `cloud` / `recamera` use SenseCAP **Access Key** → `{ai-host}/openapi` + HTTP Basic (no access_token, no `/aiserverapi`)
   - Never send Access Key as a fake Bearer/JWT
2. **Always run auth check first** (ask user to fix if missing):
   ```bash
   sensecraft-cli auth info
   ```
   Published commands require an active **SenseCAP Access Key** → `{ai-host}/openapi` + HTTP Basic.
3. **Never run `auth init` on behalf of the user** — browser OAuth Device Flow, interactive TTY only. Tell the user to run locally:
   ```bash
   sensecraft-cli auth init --kind sensecap  # Access Key (app/model/file/cloud/recamera)
   ```
4. **L2 confirmation is a separate chat turn** — follow [`../confirmation.md`](../confirmation.md). 「删这个63760」/「删除应用 demo」is **intent, not confirmation**. Ask first (操作类型 + 环境 + 对象 ID), wait for `确认`/`confirm`, **then** `--yes`. Never add `--yes` on the same turn as the first request. Applies to: app/model create·update·publish·unpublish·delete, upload attach, cloud deploy, device bind/unbind, group create/rename/delete/move.
5. **Treat tokens and keys as secrets** — do not echo full `access_token`, `refresh_token`, or `access_key`. **Public model artifacts are not downloadable** — if the user asks for 下载地址 / 下载链接 / download URL / 怎么下载, that is the same as asking for `file_url`: refuse the address, explain deploy-only (`cloud deploy` or Console). Never run `model download --url-only` on public models.
6. **Success = indented JSON on stdout**; failure = stderr + non-zero exit. Parse stdout only on success.
7. **Local USB / IP flashing is L3** — do not pretend remote auto-flash. Classic reCamera (`40`): [`sensecraft-recamera`](../sensecraft-recamera/SKILL.md) (`check` / `open` the AI deploy page). Other USB benches: Console `/ai/device/local/32|36|37|41|52`. Do **not** use `/ai/recamera` (it 301s to Grove `36`) or `/ai/device/local/40` (no such bench).
8. **Upgrading the CLI itself** is `sensecraft-cli update` (npm registry, then `npm install -g`). This is not `app update` or `model update`. Uninstall with `npm uninstall -g @seeed-studio/sensecraft-cli`. Do not delete `~/.config/sensecraft-cli` unless the user asked to clear the saved Access Key.

## Auth model (same OAuth client, different persistence)

| Kind | After Device Flow (`client_id=sensecraft-cli`) | Daily API auth |
|------|-----------------------------------------------|----------------|
| `sensecap` | Exchange for Access Key; store `key_id` + `access_key` | HTTP Basic → SenseCAP `/openapi` **and** AI `{ai-host}/openapi` (app/model/file/cloud/recamera) |

### Site / `remote_type` (AI)

| `remote_type` | AI openapi | Account center |
|---------------|------------|----------------|
| `Global` | `https://api-gateway.seeed.cc/openapi` | `account.seeed.cc/authapi` |
| `China` | `https://api-gateway.seeed.cc/openapi` | `account.seeed.cn/authapi` |

China 与 Global 使用同一个 AI host，账号中心不同。网页控制台仍是 `https://sensecraft.seeed.cc`。`SENSECRAFT_AUTH_HOST` 盖住账号中心（设备码、换 token、换 Access Key）；设了之后 Global / China 都打到这个站点根，浏览器打开的是该服务返回的验证地址。

成功命令结束后，stderr 可能各有一条英文提示，同一条 24 小时内只出现一次。看到 `Warning: A new version of sensecraft-cli (<version>)` 时，完成当前请求后可以建议用户运行 `sensecraft-cli update`。看到 `Notice: SenseCraft AI is hosting the <name> activity` 时，把活动名称、日期和链接告诉用户。交互终端会问 `Ignore this activity? [y/N]`；用户回答 `y` 之后，这条活动不再提醒。不要替用户打开链接。

`auth info` shows the active Access Key (site, org, key id). Secrets are not printed.

## Intent → skill / command routing

| User intent | Use | Auth kind | Notes |
|-------------|-----|-----------|--------|
| SenseCAP list/bind device, groups, telemetry | [`sensecraft-data`](../sensecraft-data/SKILL.md) | `sensecap` | Shipped today |
| Browse / search App Plaza, view app | [`sensecraft-apps`](../sensecraft-apps/SKILL.md) + `app list` / `app view` | `sensecap` | **CLI shipped** (list/view) |
| App fields / tool-blocks / BOM guidance | same apps skill references | `sensecap` | Config OK; real flash = L3 |
| Author a new app (create + info + images + BOM + blocks) | [`sensecraft-apps/create-app`](../sensecraft-apps/create-app/SKILL.md) | `sensecap` | One batch confirmation; stops before publish |
| Create / update / publish / unpublish / delete / bom app | same + `app …` / `app bom …` | `sensecap` | **CLI shipped** (L2 `--yes`; publish also `--agree-declaration`; bare `create --name` not L2) |
| Upload image/file/firmware/doc/archive and obtain `file_url` | [`sensecraft-files`](../sensecraft-files/SKILL.md) + `file upload` | `sensecap` | Types `1`–`2` require `--model-id`/`--app-id --yes`/`--for-app`/`--unlinked`; types `3`–`6` direct |
| Model library search/view/download/constants | [`sensecraft-models`](../sensecraft-models/SKILL.md) | `sensecap` | **CLI shipped** |
| Upload / publish / unpublish / delete model | same + `model upload --id --yes` / `create\|update\|publish\|unpublish\|delete` | `sensecap` | Attach bumps patch version like Console |
| Classic reCamera (`40`) pick / check / open deploy page | [`sensecraft-recamera`](../sensecraft-recamera/SKILL.md) + `app/model list --uniform-type 40` + `recamera check` / `open` | `sensecap` | **CLI shipped**. Flash is Console L3 (IP/USB). Not `cloud deploy`. Not Pro `53`. |
| reCamera Pro model library | [`sensecraft-models`](../sensecraft-models/SKILL.md) + `model create --target recamera-pro` / `recamera-pro resolve` | `sensecap` | Deploy is local IP/USB L3 — not `cloud deploy` |
| Jetson / cloud kit list & deploy | [`sensecraft-devices-cloud`](../sensecraft-devices-cloud/SKILL.md) + `cloud …` | `sensecap` | **CLI shipped** (list/view/deploy/order); not SenseCAP `device` |
| Local USB/IP model deploy or firmware flash | [`sensecraft-recamera`](../sensecraft-recamera/SKILL.md) then Console Deploy, or `/ai/device/local/32\|36\|37\|41\|52` | — | **L3 only**; CLI does not talk to the device |

## Cross-domain example

SenseCAP telemetry → pick public model → deploy to Jetson:

1. `sensecraft-cli auth info`
2. SenseCAP: `data latest --eui <sensecap-eui>` (Access Key)
3. AI: `model list --public --name "…"` → `model view --id …` (metadata only; **no** `model download`)
4. Confirm with user → `cloud deploy --eui <jetson-eui> --model-id … --yes` (device pulls firmware at apply time)
5. `cloud order --request-id …`

## Product-term disambiguation

| User says | Possible meanings | Default choice |
|-----------|-------------------|----------------|
| 应用 / App Store / Application | App Plaza (docs+BOM community app) vs in-doc deploy tool-block | **App Plaza** |
| 设备 / device | SenseCAP node/gateway vs Console kit/Jetson vs local USB board | Telemetry/bind → **sensecraft-data**; AI cloud device → devices-cloud; USB/serial → Console local (L3) |
| 部署模型 / deploy model | Cloud `deploy_model` vs local flash vs app tool-block | Ask which target; local flash = L3 |
| 模型下载地址 / 下载链接 / 怎么下载 | Public library artifact URL vs private owner file | **Public → no URL**, explain deploy; private My Models → `model download --out` (do not paste URL) |
| Data | SenseCAP telemetry vs training dataset | Telemetry → **sensecraft-data**; dataset → Console |
| Agent | Watcher Agent vs in-console AI Bot vs these Skills | These Skills ≠ Bot; Watcher has separate auth |

## Capability levels

| Level | Meaning | Examples |
|-------|---------|----------|
| L1 Scriptable | Agent may run directly | list / view / auth info |
| L2 Confirm | Ask user first | publish (+ declaration), unpublish, delete, deploy |
| L3 Interactive | Human-in-the-loop | `auth init`, Web Serial flash, payment |
| L4 Ops-only | Do not expose to end-user Agent | Admin `judge_app` |

## SenseCraft AI surface (shipped MVP)

Published commands use **SenseCAP Access Key** (`kind=sensecap`) → `{ai-host}/openapi`:

1. **Apps** — list/view/fields/blocks; create/update; publish/unpublish/delete (L2); BOM search/list/set; tool-block config via JSON (real flash = L3).
2. **Files** — `file upload` for image/model/app/firmware/doc/archive; types 1–2 require `--model-id` / `--app-id --yes` / `--for-app` / `--unlinked` before PUT (bare upload fails).
3. **Model library** — list/view/constants; **private** download only; public models deploy-only (no `file_url`); `model upload --id --yes` attaches and auto-bumps patch version; create/update; publish/unpublish/delete (L2).
4. **Cloud devices** — `cloud device` / `cloud deploy` / `cloud order`.
5. **Classic reCamera 40** — `recamera check` / `recamera open` (Console AI deploy page; flash is L3).
6. **Local USB** — Other benches: Console `/ai/device/local/32|36|37|41|52`.

## When something is missing

Published AI CLI is `app` / `model` / `file` / `cloud` / `recamera`. If a command fails:

1. Run `sensecraft-cli auth info` — need an active SenseCAP Access Key.
2. Do **not** call SenseCAP `device`/`data` as a substitute.
3. Do **not** scrape Console cookies or invent HTTP with guessed tokens.
4. Prefer domain skills (`apps` / `files` / `models` / `recamera` / `devices-cloud`).

## Related files in this repo

| Path | Role |
|------|------|
| [`../sensecraft-data/SKILL.md`](../sensecraft-data/SKILL.md) | SenseCAP PaaS skill (shipped) |
| [`../sensecraft-data/commands.md`](../sensecraft-data/commands.md) | Full SenseCAP command reference |
| `skills/sensecraft-apps/` | App Plaza skill + field/block refs (**shipped**) |
| `skills/sensecraft-apps/create-app/` | App authoring sub-skill: one batch confirmation (**shipped**) |
| `skills/sensecraft-models/` | Model library skill + field refs (**shipped**) |
| `skills/sensecraft-recamera/` | Classic reCamera 40 pick → check → open deploy page (**shipped**) |
| `skills/sensecraft-devices-cloud/` | Jetson / cloud kit (**shipped**) |
| `skills/README.md` | Skill discovery for Agents |
