# Model library & upload field reference

Aligned with Console `views/model/add.vue`, `detail.vue`, and `constants/sensecraftModel.js`.

## Identity & copy

| Field | Type | Required (create UI) | Compliance |
|-------|------|----------------------|------------|
| `name` | text | yes | max **64** (`SENSECRAFT_MODEL_NAME_MAX_LENGTH`); trim before API |
| `description` | text | yes | max **300** |
| `content` | rich text | yes | model detail / documentation body |
| `pic_url` | image URL | yes | inference example image; upload via `file upload --type 1` (**≤5 MB**); JPG/PNG/GIF/WebP; must be http(s) URL |
| `scenario` | option | yes | id from `get_constant` **`scenarios_array`** (not app `app_scenarios_array`); CLI rejects unknown ids |
| `precision` | option | yes | id from **`model_precision_array`** (1=INT8, 2=FLOAT16, 3=FLOAT32 typical); CLI rejects unknown ids |
| `file_url` | URL | yes | model binary URL; http(s); extension must match `model_format` |
| `preparation` | string[] | often | prep steps (UI list) |

## Classification & devices

| Field | Type | Required | Compliance |
|-------|------|----------|------------|
| `task` | option (id) | yes | From **`get_constant` → `model_task_array`**. Product categories include vision detection, vision classification, audio, vibration-related tasks, etc. **Resolve names→ids from constants**; do not hardcode stale enums in Agent replies when constants are available. |
| `ai_framework` | option (id) | conditional | From `ai_framework_array`, filtered by device/task (e.g. reCamera Pro detection). Classification/audio may hide framework and map algorithm by task. |
| `adapteds` | multi device | yes | Supported device entries; drives `uniform_types` and accept filters |
| `model_format` | option | yes | From `model_format_array` (examples in code comments: alpkg=`7`, zip=`8`; rknn common for vision). Accept filter follows format + task (e.g. audio → `.rknn` / `.alpkg` / `.zip` rules in UI). |

## Classes / labels

| Field | Type | Required | Compliance |
|-------|------|----------|------------|
| `labels[]` | list | usually yes | Each item: `object_id` (**number or numeric string, not JSON `null`**), `object_name` (non-empty, pass `validateClassName`) |
| classes | derived | — | Often mirrored from label names for deploy |

Empty or invalid class names block save/deploy in Console.

CLI-enforced class-name rule matches Console `validateClassName`: ASCII letters, digits, and spaces only; at least one letter; the first character cannot be a digit.

## Device family and format matrix

`adapteds` contains device option ids, not uniform types. Create/update resolves every id through the current `get_constant.device_type_array` and enforces the same selection families as Console:

| Resolved `uniform_type` | Selection | Allowed `model_format` / file |
|-------------------------|-----------|-------------------------------|
| `32`, `52` | group may multi-select; cannot mix with other families | `2` / `.tflite` |
| `36`, `37`, `41` | group may multi-select; cannot mix with other families | `2` / `.tflite` |
| `26` | Jetson/reComputer entries may multi-select | `3` / `.engine` or `4` / `.pt` |
| `40` | single selection | `5` / `.cvimodel` |
| `53` | single selection | Pro matrix below |
| other | single selection | current format option except `2`; extension follows format |

Non-Pro payloads do not submit `ai_framework`, matching Console.

## reCamera Pro (`uniform_type=53`)

Create/update automatically recognizes Pro from `adapteds`. The resolver remains useful for planning:

```bash
sensecraft-cli model recamera-pro resolve --task <task> [--ai-framework <id>]
```

### Supported task/format matrix

| Task id | Meaning | `ai_framework` | Device algorithm | Format / file |
|---------|---------|----------------|------------------|---------------|
| `1` | Detection | required; see map below | mapped framework algorithm | `6` / `.rknn` |
| `2` | Classification | omit | `classify` | `6` / `.rknn` |
| `4` | Pose | only `2` (YOLOV8) | `yolov8_pose` | `6` / `.rknn` |
| `6` | Audio | omit | package-defined | `7` / `.alpkg` or `8` / `.zip` |
| `7` | Tracking | omit | `bytetrack` | `6` / `.rknn` |
| `3`, `5` | Segmentation, Generation | — | — | hidden/unsupported |

### Detection framework map

| `ai_framework` id | Device `algorithm` |
|-------------------|--------------------|
| `1` | `yolov5` |
| `2` | `yolov8` |
| `8` | `yolov5u` |
| `9` | `yolov10` |
| `10` | `yolo11` |
| `11` | `yolo12` |
| `12` | `yolo26` |
| `13` | `nanodet` |
| `14` | `yolox` |
| `15` | `detr` |

The IDs must still exist in the current `get_constant` response. Do not substitute display names for IDs in library payloads.

reCamera Pro deployment is local device IP/USB (Console workspace). It does not use cloud `update_stream`; the CLI resolver validates metadata but does not pretend to complete physical deployment.

## File & tech metadata

| Field | Type | Required | Compliance |
|-------|------|----------|------------|
| `file_url` | URL | yes | Via `acquire_file_url` upload (`upload-type` model); http(s); extension must match `model_format` |
| `checksum` | string | yes in UI | MD5/checksum from upload; keep in sync with `file_url` |
| `model_size` | number/string | from upload | bytes / display size |
| `version` | text | yes | exact `major.minor.patch`; major `1–99`, minor/patch `0–99`, no leading zeros; update cannot decrease; unchanged version auto-increments when labels/file changes |
| `precision` | option | yes | id from `model_precision_array`; CLI rejects unknown ids |
| `attr.conf` / `attr.iou` | number | Watcher-related | sliders when adapted devices include Watcher |

CLI create/update validates the **complete** merged payload (Console save rules) before calling the API: empty/overlong `name`/`description`, missing required fields, invalid option ids (`task`/`scenario`/`precision`/`model_format`), bad labels, device/format matrix, and http(s) `pic_url`/`file_url`. Invalid input → non-zero exit + field-level error (not opaque API text). Sparse update patches are merged with `view_model` first, then validated.

## Visibility lifecycle

| `be_public` | Meaning |
|-------------|---------|
| `0` | Private (My Models). Delete allowed. |
| `1` | Public library. **Owner** may edit (re-enters review; CLI `edit_model` sends `be_public=1`). Others: deploy-only, no clone/download/edit. |
| `2` | Under review. Owner may edit; delete hidden until unpublished. |

### Update API

`POST /model/edit_model` requires **`model_id` and `id`** (same value). CLI `model update` sets both.

### Publish API (not the same as apps)

`POST /model/publish_model`

| Intent | Body |
|--------|------|
| Publish to public library | `{ "model_id": <id>, "status": 1 }` |
| Privatize / unpublish | `{ "model_id": <id>, "status": 0 }` |

`model_id` must be a **JSON number** (e.g. `63758`, not `"63758"`). CLI converts a numeric `--id` before POST. String `model_id` returns aiserver **11202**.

**L2 user confirmation required.** Do not confuse with app `edit_privacy_app` (`privacy` 2/0).

## Download & invoke

| Path | Who | Behavior |
|------|-----|----------|
| Detail `file_url` | **Private owner** download in UI / CLI | Console: owner-only file box. CLI: `model download` only when `be_public=0`. Public list/view **omit** `file_url`. |
| `GET /model/apply_model` | Deploy pipelines (device pull) | Snapshot URL for device apply — **not** a user-facing download for public models. Jetson: `cloud deploy`. reCamera Pro: Console local L3. |
| CodeCraft | External | If detail or product config exposes workspace/compile invoke links, **return them to the user**. This skill does **not** run CodeCraft compilation. Bases (prod/test) live in Console `envApi` (`codecraftHomeUrl` / `codecraftWorkspaceUrl`). |

## Delete

`POST /model/delete_model` with `{ model_id }` — **L2 confirm**; owner **private** only (`be_public=0`). Published / under-review: `model unpublish` first.
