---
name: sensecraft-cli
description: >-
  Operates the SenseCraft CLI across SenseCraft Data (devices, groups, telemetry),
  SenseCraft AI (model library, reCamera), and CodeCraft Hub (applications).
  Use when the user mentions SenseCraft, SenseCraft Data, SenseCraft AI, CodeCraft Hub,
  应用, 模型库, 上传模型, reCamera, 设备, or sensecraft-cli.
  Open only the reference file for the product being used.
---

# SenseCraft CLI

`sensecraft-cli` is the terminal for the SenseCraft cloud platform. This file is the only entry. Product procedures live in `references/`. A finished local project becomes an app through the nested `create-app` skill.

| Product | What it covers | Commands |
|---------|----------------|----------|
| SenseCraft Data | devices, groups, telemetry | `device` · `group` · `data` |
| SenseCraft AI | model library, reCamera | `model` · `recamera` · `cloud` |
| [CodeCraft Hub](https://codecraft.seeed.cc/hub) | applications | `app` |

All three platforms are accessed with an open Access Key. Create or manage a key at https://account.seeed.cc/account/access-keys.

SenseCraft Data: [`references/data.md`](references/data.md).

## Agent rules

1. **Use the Access Key before any cloud call**
   - Every cloud command uses the Access Key. SenseCraft Data calls its OpenAPI. `app` / `model` / `file` / `cloud` / `recamera` call `{ai-host}/openapi` with HTTP Basic (no access_token, no `/aiserverapi`)
   - Never send the Access Key as a fake Bearer/JWT
2. **Always run auth check first**:
   ```bash
   sensecraft-cli auth info
   ```
   Published commands require an active **SenseCAP Access Key** → `{ai-host}/openapi` + HTTP Basic.
3. **Do not run bare `auth init`** — that wizard needs a TTY. In this shell, use the split flow. Each command exits and prints JSON. Show `verification_url` and end the turn before polling:
   ```bash
   sensecraft-cli auth init --site Global --no-wait
   sensecraft-cli auth init --device-code <device_code>
   sensecraft-cli auth init --device-code <device_code> --key-id <access_id>
   ```
   Create a key with `--label <name> --access-level read-write` (or `only-read`) instead of `--key-id`. Use Global for SenseCraft AI and CodeCraft Hub. China is SenseCraft Data only. Do not print `access_token` or `access_key`. If the user already has a key, they can run `auth add` locally.
4. **L2 confirmation is a separate chat turn** — follow [references/confirmation.md](references/confirmation.md). 「删这个63760」/「删除应用 demo」is **intent, not confirmation**. Ask first (操作类型 + 环境 + 对象 ID), wait for `确认`/`confirm`, **then** `--yes`. Never add `--yes` on the same turn as the first request. Applies to: app update·publish·unpublish·withdraw·delete, model create·update·publish·unpublish·delete, upload attach, cloud deploy, device bind/unbind, group create/rename/delete/move. **Exception: `app create`, including one-shot `--from-json`, is not L2 and takes no `--yes`; resolve uncertain content decisions and review the result first.**
5. **Treat tokens and keys as secrets** — do not echo full `access_token`, `refresh_token`, or `access_key`. **Public model artifacts are not downloadable** — if the user asks for 下载地址 / 下载链接 / download URL / 怎么下载, that is the same as asking for `file_url`: refuse the address, explain deploy-only (`cloud deploy` or Console). Never run `model download --url-only` on public models.
6. **Success = indented JSON on stdout** for business commands and the split `auth init` flow. The interactive wizard prints human-readable text. Failure = stderr + non-zero exit. Parse stdout only on success.
7. **Local USB / IP flashing is L3** — do not pretend remote auto-flash. Classic reCamera (`40`): [`sensecraft-recamera`](references/recamera.md) (`check` / `open` the AI deploy page). Other USB benches: Console `/ai/device/local/32|36|37|41|52`. Do **not** use `/ai/recamera` (it 301s to Grove `36`) or `/ai/device/local/40` (no such bench).
8. **Upgrading the CLI itself** is `sensecraft-cli update` (npm registry, then `npm install -g`). This is not `app update` or `model update`. Uninstall with `npm uninstall -g @seeed-studio/sensecraft-cli`. Do not delete `~/.config/sensecraft-cli` unless the user asked to clear the saved Access Key.

## Auth

`auth init` runs Device Flow (`client_id=sensecraft-cli`), exchanges the login for an Access Key, and stores `key_id` + `access_key`. Daily calls use HTTP Basic on SenseCraft Data `/openapi` and on `{ai-host}/openapi` (SenseCraft AI and CodeCraft Hub). An existing key can be saved with `auth add`. Create or manage keys at https://account.seeed.cc/account/access-keys.

Bare `auth init` asks the user to pick Global or China. The split flow takes `--site` instead. The China site supports SenseCraft Data only. To use SenseCraft AI or CodeCraft Hub, select Global. Do not point the CLI at another host. The login access token stays in the local pending file and is not printed.

`auth info` shows the active Access Key (site, org, key id). Secrets are not printed.

## Intent → skill / command routing

| User intent | Use | Notes |
|-------------|-----|--------|
| SenseCraft Data: list/bind device, groups, telemetry | [`sensecraft-data`](references/data.md) | Shipped today |
| Browse / search CodeCraft Hub, view app | [`sensecraft-apps`](references/apps.md) + `app list` / `app view` | **CLI shipped** (list/view) |
| App fields / tool-blocks / BOM guidance | same apps skill references | Config OK; real flash = L3 |
| Turn a local project into an app (discovery + reviewed copy + tool blocks + API-confirmed BOM + manifest + upload), or update its manifest-managed app | [`create-app`](create-app/SKILL.md) | Read required references before authoring; create needs no `--yes`; existing-app updates use one batch confirmation; stops before publish |
| Create / update / publish / unpublish / withdraw / delete / bom app | same + `app …` / `app bom …` | **CLI shipped** (L2 `--yes`; publish also `--agree-declaration`; `create`, including `--from-json`, is not L2). `withdraw` is only for an app under review (`be_public=2`); `unpublish` is only for an already public app |
| Upload image/file/firmware/static/archive and obtain `file_url` | [`sensecraft-files`](references/files.md) + `file upload` | Types `1`–`2` require `--model-id`/`--app-id --yes`/`--for-app`/`--unlinked`; type `5` requires `--flow-id`; types `3`–`6` direct |
| Model library search/view/download/constants | [`sensecraft-models`](references/models.md) | **CLI shipped** |
| Upload / publish / unpublish / delete model | same + `model upload --id --yes` / `create\|update\|publish\|unpublish\|delete` | Attach bumps patch version like Console |
| Classic reCamera (`40`) pick / check / open deploy page | [`sensecraft-recamera`](references/recamera.md) + `app/model list --uniform-type 40` + `recamera check` / `open` | **CLI shipped**. Flash is Console L3 (IP/USB). Not `cloud deploy`. Not Pro `53`. |
| reCamera Pro model library | [`sensecraft-models`](references/models.md) + `model create --target recamera-pro` / `recamera-pro resolve` | Deploy is local IP/USB L3 — not `cloud deploy` |
| Jetson / cloud kit list & deploy | [`sensecraft-devices-cloud`](references/cloud.md) + `cloud …` | **CLI shipped** (list/view/deploy/order). SenseCraft AI, not SenseCraft Data `device` |
| Local USB/IP model deploy or firmware flash | [`sensecraft-recamera`](references/recamera.md) then Console Deploy, or `/ai/device/local/32\|36\|37\|41\|52` | **L3 only**; CLI does not talk to the device |

## Cross-domain example

SenseCraft Data telemetry → pick a public model → deploy to Jetson:

1. `sensecraft-cli auth info`
2. SenseCraft Data: `data latest --eui <eui>` (Access Key)
3. SenseCraft AI: `model list --public --name "…"` → `model view --id …` (metadata only; **no** `model download`)
4. Confirm with user → `cloud deploy --eui <jetson-eui> --model-id … --yes` (device pulls firmware at apply time)
5. `cloud order --request-id …`

## Product-term disambiguation

| User says | Possible meanings | Default choice |
|-----------|-------------------|----------------|
| application / App Store | CodeCraft Hub app vs in-doc deploy tool-block | **CodeCraft Hub** (`sensecraft-apps`) |
| device | SenseCraft Data node/gateway vs SenseCraft AI cloud device vs local USB board | Telemetry/bind → **sensecraft-data**; cloud device → devices-cloud; USB/serial → Console local (L3) |
| deploy a model | Cloud `deploy_model` vs local flash vs app tool-block | Ask which target; local flash = L3 |
| model download URL / download link / how to download | Public library artifact URL vs private owner file | **Public → no URL**, explain deploy; private My Models → `model download --out` (do not paste URL) |
| Data | SenseCraft Data telemetry vs training dataset | Telemetry → **sensecraft-data**; dataset → Console |
| Agent | Watcher Agent vs in-console AI Bot vs these Skills | These Skills ≠ Bot; Watcher has separate auth |

## Capability levels

| Level | Meaning | Examples |
|-------|---------|----------|
| L1 Scriptable | Agent may run directly | list / view / auth info / `auth init --no-wait` then later `--device-code` |
| L2 Confirm | Ask user first | publish (+ declaration), unpublish, withdraw, delete, deploy |
| L3 Interactive | Human-in-the-loop | bare `auth init`, Web Serial flash, payment |
| L4 Ops-only | Do not expose to end-user Agent | Admin `judge_app` |

## Shipped commands

Published commands use an Access Key. SenseCraft Data uses its OpenAPI. SenseCraft AI and CodeCraft Hub use `{ai-host}/openapi`.

1. **CodeCraft Hub** — `app` list/view/fields/blocks/doc; create/update; publish/unpublish/withdraw/delete (L2); BOM search/list/set; tool-block config via JSON (real flash = L3). `withdraw` unlocks an app under review. `unpublish` does not.
2. **SenseCraft AI** — model library (list/view/constants; **private** download only; public models deploy-only); `model upload --id --yes`; create/update; publish/unpublish/delete (L2). Cloud devices: `cloud device` / `cloud deploy` / `cloud order`. Classic reCamera 40: `recamera check` / `recamera open` (flash is L3).
3. **Files** — `file upload` is shared. Types 1–2 require `--model-id` / `--app-id --yes` / `--for-app` / `--unlinked` before the OSS POST. Type 5 also requires `--flow-id`.
4. **Local USB** — Other benches: Console `/ai/device/local/32|36|37|41|52`.

## When something is missing

SenseCraft AI and CodeCraft Hub commands are `model` / `recamera` / `cloud` and `app`. `file` is shared. If a command fails:

1. Run `sensecraft-cli auth info` — need an active Access Key.
2. Do **not** call SenseCraft Data `device`/`data` as a substitute.
3. Do **not** scrape Console cookies or invent HTTP with guessed tokens.
4. Open the matching file under `references/`. Do not invent a second skill.

## References

Read the file for the product in use. Leave the others closed.

| When | Read |
|------|------|
| Devices, groups, telemetry | [references/data.md](references/data.md), [references/data-commands.md](references/data-commands.md) |
| Applications | [references/apps.md](references/apps.md), [references/app-commands.md](references/app-commands.md), [references/app-fields.md](references/app-fields.md), [references/tool-blocks.md](references/tool-blocks.md) |
| Turn a local project into an app | [create-app/SKILL.md](create-app/SKILL.md) |
| Model library | [references/models.md](references/models.md), [references/model-commands.md](references/model-commands.md), [references/model-fields.md](references/model-fields.md) |
| Classic reCamera | [references/recamera.md](references/recamera.md), [references/recamera-commands.md](references/recamera-commands.md) |
| Cloud devices | [references/cloud.md](references/cloud.md), [references/cloud-commands.md](references/cloud-commands.md) |
| File upload | [references/files.md](references/files.md), [references/file-commands.md](references/file-commands.md) |
| Write confirmation | [references/confirmation.md](references/confirmation.md) |
