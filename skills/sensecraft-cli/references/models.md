# SenseCraft AI model library

Domain skill for **SenseCraft AI models**. Auth: **Access Key**. The China site supports SenseCraft Data only; use Global for SenseCraft AI.

Parent: [`../SKILL.md`](../SKILL.md).  
Apps: [`apps.md`](apps.md).  
Library `model download` is the published model file, not a training-job export.

## Agent rules

1. Auth first:
   ```bash
   sensecraft-cli auth info
   ```
   Prefer **SenseCAP Access Key** (`auth init`) → `{ai-host}/openapi` Basic. Missing key → the agent runs `auth init` in a PTY/TTY session, handles prompts, and rechecks auth per the [parent auth workflow](../SKILL.md#auth). The user completes browser authorization; do not ask them to run commands.
2. **L2 confirm is a separate chat turn** ([confirmation.md](confirmation.md)) before: `publish` / `unpublish` / `delete` / `create` / `model update` / attach-upload. 「删这个63760」is **not** confirmation — ask `将从{环境}环境删除模型 63760。删除后可能无法恢复。是否确认执行？`, wait for `确认`, then `--yes`. Never pass `--yes` on the same turn as the first request.
3. When the user **attaches a model file** ("上传这个模型文件" / "更新模型"):
   - **Own My Models** (including already published `be_public=1`): after chat confirm, `model update --id <model_id> --path <file> --yes`. Same as Console 编辑模型. If it is **already public**, the confirm question must say 模型信息更新后将重新进入审核流程. CLI sends `be_public=1` like Console `requestConfirmFn`. JSON includes `re_review: true`.
   - `model upload --path <file> --id <model_id> --yes` is the same attach path.
   - Do **not** unlinked-upload then a second `model update` just to bump version.
   - If they did not give an id, **ask** whether this replaces an existing private/own model or creates a new one. Do **not** call upload yet — bare `model upload --path` / `file upload --path` exits non-zero (`ErrAttachRequired`) and must not be used to "probe".
   - Brand-new model only: `model upload --path <file> --unlinked` then `model create --from-json`.
   - **Someone else's public library** id: refuse (no clone). New models only: `--unlinked` upload then `model create --from-json`.
   - Metadata-only edit: `model update --id <model_id> --from-json patch.json --yes`. Labels change also auto-increments version like Console.
   - **Delete** only when `be_public=0` (Console hides delete for published/under-review). Unpublish first.
4. Model **publish** API uses `status: 1` (to public library) / `status: 0` (privatize) — **different from apps** (`privacy` 2/0). Do not mix.
5. **Public models are deploy-only.** For public / under-review models (`be_public≠0`):
   - **Do not copy a public model into My Models.** Do not copy `model view` JSON into `model create` (including `id` / `model_id` / `from_id` / `file_url` / checksum). CLI `model create` refuses public-library sources. Tell the user to **deploy** instead.
   - **Do not download.** Phrases like 下载地址 / 下载链接 / `file_url` / download URL / 怎么下载 / 给我链接 are the **same leak** as `model download --url-only`. Do **not** run `model download` (`--apply` / `--url-only` included).
   - Do **not** print, quote, or reconstruct `file_url`, `apply_model` URLs, CDN/OSS paths, or signed query strings — even if they appear in a tool error, browser, or prior message.
   - Answer the **method**, not an address: browse metadata, then deploy so the device pulls firmware at apply time (`cloud deploy` for Jetson/`26`, or classic reCamera `40` via [`sensecraft-recamera`](recamera.md)).
   - Owner file download is only for **private** My Models (`be_public=0`). Even then, do not paste signed URLs into chat unless the user explicitly needs `--url-only` for a local script; prefer `--out` file path. Treat remaining `file_url`, checksums, and tokens as secrets.
6. Success = indented JSON stdout; failure = stderr + non-zero exit.
7. Classic reCamera (`uniform_type=40`): list with `--uniform-type 40`, then [`sensecraft-recamera`](recamera.md) `check` / `open`. **Do not** `cloud deploy`. **Do not** download public models. reCamera Pro (`53`) is a different device — use `model recamera-pro resolve`, then Console local L3 (not this 40 flow).

## CLI status

| Area | Status |
|------|--------|
| Access Key | **Implemented** (`{ai-host}/openapi` Basic) |
| `model list` / `model view` / `model download` | **Implemented** |
| `model create` / `update` / `publish` / `unpublish` / `delete` | **Implemented** (L2: `--yes`; create needs `file_url` in JSON) |
| `model upload` / `file upload` | **Implemented** (types 1–2 require `--id`/`--model-id`/`--app-id`/`--for-app`/`--unlinked`; attach auto-increments patch version like Console) |
| `model constants` | **Implemented** (`get_constant`) |
| `model recamera-pro resolve` | **Implemented** (local task/framework/algorithm/format validation) |

## Typical workflows

### Find a suitable public model

```bash
sensecraft-cli auth info
sensecraft-cli model list --public --task <task_id> --name "<keyword>"
sensecraft-cli model list --public --uniform-type 40
sensecraft-cli model list --mine --name "<keyword>"
sensecraft-cli model view --id <model_id>
```

Report: name, description, **task/category**, **supported devices** (`uniform_types` / adapteds), format, classes/labels, size/version, any invoke/deep-link fields.

If they ask to **clone** a public model into My Models, refuse. There is no `model clone`. Do not `model create` from that model's view JSON. Reply that others' public models cannot be cloned; deploy instead:

> 他人公开模型不支持克隆。Jetson/云套件用 `cloud deploy`；经典 reCamera（40）用 `recamera check` / `recamera open` 后在 Console 点部署；Pro（53）走 Console 本地工作区。

If they ask for a **download address / link / URL / 怎么下载** on a public model, do **not** fetch a URL. Reply that public models have no download link; the way to get the artifact is to **deploy** (device pulls firmware). Example:

> 公开模型不提供下载地址。Jetson/云套件用 `cloud deploy`；经典 reCamera（40）用 `recamera open` 后在 Console 点部署。设备会在 apply 时拉取固件。

### Download (private / My Models only)

```bash
sensecraft-cli model download --id <model_id> [--out ./path]
sensecraft-cli model download --id <model_id> --apply [--out ./path]
sensecraft-cli model download --id <model_id> --url-only
```

- Allowed only when `be_public=0` (owner private library). Public / under-review models **refuse** download — deploy instead.
- Default: owner library `view_model.file_url`
- `--apply`: deploy snapshot via `apply_model` → `model_snapshot.arguments.url` (still private-only; not a public-library shortcut)
- File downloads omit `url` from JSON unless `--url-only`. Never paste signed URLs into chat.

### Deploy a public model (no download)

```bash
# Jetson / cloud kit — device pulls firmware at apply time
sensecraft-cli cloud deploy --eui <EUI> --model-id <id> --yes

# reCamera Pro / USB local — Console workspace (L3); CLI does not fetch the binary
```

### Upload + publish (L2)

**Replace an existing private model's file** (user attached a binary / “上传这个模型文件” / “更新模型”):

```bash
sensecraft-cli model update --id <model_id> --path ./model.rknn --yes
# same attach: sensecraft-cli model upload --path ./model.rknn --id <model_id> --yes
# file + metadata: sensecraft-cli model update --id <model_id> --path ./model.rknn --from-json ./patch.json --yes
```

This writes `file_url` / `checksum` / `model_size` and auto-increments the patch version (Console: `1.0.0` → `1.0.1`). Do not follow with a separate `model update` just to bump version. If the model is **already published** (`be_public=1`), warn that it will **re-enter review**, then proceed with `--yes` (CLI sends `be_public=1` like Console).

**Brand-new model** only:

1. Gather fields from [model-fields.md](model-fields.md)
2. Explicit unlinked upload, then put `file_url` (+ `checksum`, `size` as `model_size`) into create JSON:
   ```bash
   sensecraft-cli model upload --path ./model.rknn --unlinked
   sensecraft-cli model create --from-json ./model.json --yes
   sensecraft-cli model create --from-json ./pro-model.json --target recamera-pro --yes
   ```

Publish / unpublish / delete still need a **chat confirm first**, then `--yes` (never same-turn as 「删这个 / 发布 / 下架」):
   ```bash
   # only after the user replies 确认 to your question
   sensecraft-cli model publish --id <model_id> --yes
   sensecraft-cli model unpublish --id <model_id> --yes
   sensecraft-cli model delete --id <model_id> --yes
   ```

Successful `model create` JSON includes `model_id` and `console_url` (Console detail: `/ai/model/detail?id=…&tab=custom`). Show that link to the user.

Create must contain the complete field set in [model-fields.md](model-fields.md). Update may be an incremental input patch: CLI loads `view_model`, merges and validates it, then submits the normalized complete Console form so removed conditional fields (such as Pro Classification `ai_framework`) do not remain stale.

Model publish uses **`status` 1/0** — not app `privacy` 2/0. Never set `be_public=1` in create/update JSON.

### reCamera Pro compatibility

When an `adapteds` entry resolves through `get_constant.device_type_array` to `uniform_type=53`, create/update applies Pro validation automatically. Use `--target recamera-pro` only as an additional assertion:

```bash
sensecraft-cli model recamera-pro resolve --task <task_id> [--ai-framework <id>]
sensecraft-cli model create --from-json ./pro.json --target recamera-pro --yes
```

- Detection (`task=1`): framework ids `1,2,8,9,10,11,12,13,14,15`.
- Classification (`task=2`): algorithm is fixed to `classify`; omit `ai_framework`.
- Pose (`task=4`): only YOLOV8 (`ai_framework=2`), mapped to `yolov8_pose`.
- Audio (`task=6`): omit framework; upload `.alpkg` (`model_format=7`) or `.zip` (`8`).
- Tracking (`task=7`): algorithm fixed to `bytetrack`; omit framework.
- Segmentation (`3`) and Generation (`5`) are hidden/unsupported for reCamera Pro upload/deploy.
- All supported non-audio tasks use `.rknn` (`model_format=6`).

Detection framework→algorithm and full upload constraints are in [model-fields.md](model-fields.md).

## Field summary (library card / detail)

| Concept | API / field hints | Notes |
|---------|-------------------|--------|
| Name | `name` | max **64** chars |
| Intro | `description` | max **300** (UI) |
| Detail body | `content` | rich text / HTML |
| Category / task | `task` | from `get_constant` → `model_task_array` (vision detect/classify, audio, …) — **do not invent ids** |
| AI framework | `ai_framework` | required for some device+task combos (e.g. reCamera Pro detection); from `ai_framework_array` |
| Format | `model_format` | from `model_format_array` (e.g. rknn, alpkg, zip, …) |
| Classes | `labels[]` with `object_name` | at least one valid class when UI requires |
| Supported devices | `adapteds` / `uniform_types` | multi-select device types |
| Tech | `version`, `precision`, `checksum`, `model_size`, `attr.conf`/`iou` | device-dependent |
| File | `file_url` | uploaded model artifact; **omitted from public list/view stdout** |
| Cover / demo | `pic_url` | inference example image (required on create UI; **≤5 MB** when uploading with `file upload --type 1`) |
| Visibility | `be_public` | `0` private · `1` public · `2` under review (display) |
| Invoke / CodeCraft | detail or env deep links | Surface any API-provided invoke URL; CodeCraft workspace base from product env — **do not implement CodeCraft compile inside this skill** |

Full create/edit compliance: [model-fields.md](model-fields.md).

## APIs (wiring reference)

Base: `{host}/openapi`, HTTP Basic (`key_id`:`access_key`).

| Action | Method | Path |
|--------|--------|------|
| List public | GET | `/model/list_model` (optional `uniform_type` / `adapteds`) |
| List mine | GET | `/model/list_custom_model` |
| View | GET | `/model/view_model` |
| Create | POST | `/model/create_model` |
| Edit | POST | `/model/edit_model` |
| Publish / privatize | POST | `/model/publish_model` (JSON number `model_id`, `status` 1\|0) |
| Delete | POST | `/model/delete_model` |
| Upload URL | GET | `/user/acquire_file_url` |
| Apply / deploy snapshot | GET | `/model/apply_model` |
| Constants (tasks/formats) | GET | `/user/get_constant` |

## Additional resources

- [model-commands.md](model-commands.md)
- [model-fields.md](model-fields.md)
