# sensecraft-models — CLI commands

Binary: `sensecraft-cli`. Auth: **Access Key** (`auth init`) → `{ai-host}/openapi`.

> Status: **shipped**. Every command below is implemented — run it rather than describing it.

## Prerequisite

```bash
sensecraft-cli auth info
sensecraft-cli auth init --site Global --no-wait   # split flow; see SKILL.md
```

## Commands

### `model list`

**Status: implemented.**

| Flag | Notes |
|------|-------|
| `--public` | `list_model` (default if neither flag) |
| `--mine` | `list_custom_model` |
| `--name` | keyword → API `search` (Console model library search box) |
| `--task` | task id from constants |
| `--uniform-type` | device type → API `uniform_type` (`40` reCamera, `32` XIAO ESP32S3, `53` reCamera Pro) |
| `--adapteds` | device id → API `adapteds` (Console cardList SKU filter) |
| `--page` / `--length` | pagination |
| `--sandbox` | sample JSON |

```bash
# Public library (default when neither --public nor --mine is set)
sensecraft-cli model list --public --task 1 --name "vision" --page 1 --length 20
sensecraft-cli model list --public --uniform-type 40
sensecraft-cli model list --mine --uniform-type 40

# My Models
sensecraft-cli model list --mine --name "custom"
```

Classic reCamera (`40`) pick-and-open: [recamera-commands.md](recamera-commands.md). Do not `cloud deploy` those models.

### `model view --id <id>`

**Status: implemented.** Return name, description, task, format, labels/classes, uniform_types, version, size, be_public, etc.

Public / under-review models **omit `file_url`**. Do not treat the missing field as a bug.

```bash
sensecraft-cli model view --id <model_id>
```

### `model download --id <id> [--out path]`

**Status: implemented.** Private owner models only (`be_public=0`).

| Flag | Notes |
|------|-------|
| `--id` | required |
| `--out` | file or directory (default: URL basename in cwd) |
| `--apply` | use `apply_model` snapshot URL instead of `file_url` (still private-only) |
| `--url-only` | print JSON with URL; do not write a file |
| `--sandbox` | sample bytes / sample URL |

Stdout JSON: `model_id`, `source` (`file_url`\|`apply_model`), optional `path`/`bytes`/`filename`. `url` is included **only** with `--url-only`.

Public library models exit non-zero: deploy via `cloud deploy` or Console so the device pulls firmware at apply time.

If the user asks for a download **address** (URL / 链接 / `file_url`) or **how to download** a public model, do **not** retry `--url-only` or dump view JSON. Tell them there is no download URL; the method is deploy-time firmware pull.

### `model constants`

**Status: implemented.** Dumps `get_constant` (`model_task_array`, `model_format_array`, `ai_framework_array`, …) for Agent id↔name mapping.

```bash
sensecraft-cli model constants
```

### `model recamera-pro resolve`

**Status: implemented.** Local compatibility check; no auth or network required. It validates a reCamera Pro task/framework combination and returns the device-side `algorithm`, required model format, file extension, and deploy mode.

```bash
# Detection: task 1 + YOLOV8 framework id 2 → algorithm yolov8
sensecraft-cli model recamera-pro resolve --task 1 --ai-framework 2

# Classification → algorithm classify
sensecraft-cli model recamera-pro resolve --task 2

# Pose only accepts YOLOV8 id 2 → algorithm yolov8_pose
sensecraft-cli model recamera-pro resolve --task 4 --ai-framework 2

# Audio accepts .alpkg (format 7) or .zip (format 8)
sensecraft-cli model recamera-pro resolve --task 6
```

Unsupported combinations print structured JSON and exit non-zero.

### `model upload --path <file>`

**Status: implemented.** Uploads `file_type=2` with checksum.

| Flag | Required | Notes |
|------|----------|-------|
| `--path` | yes | Local model binary |
| `--id` | one of `--id` / `--unlinked` | Private My Models id to attach; public ids refused |
| `--unlinked` | one of `--id` / `--unlinked` | Explicit orphan for brand-new `model create` only |
| `--yes` | with `--id` | L2; without it, attach does **not** upload |
| `--sandbox` | no | Sample JSON, no network |

When replacing an existing private model's artifact, pass `--id` so the file is attached and the **patch version auto-increments** (Console `incrementVersion`: `1.0.0` → `1.0.1` when `file_url` changes). This is L2 (`--yes`). Bare `model upload --path` (no `--id`/`--unlinked`) exits non-zero **before** the OSS upload. Interactive TTY guides association; agents must ask the user first. JSON includes `attached`, `model_id`, `previous_version`, `version`.

```bash
# Existing model: attach + bump version
sensecraft-cli model upload --path ./model.rknn --id <model_id> --yes

# New model only: capture file_url for model create --from-json
sensecraft-cli model upload --path ./model.rknn --unlinked
```

### `model create` / `model update` (L2)

**Status: implemented.**

| Command | Flags |
|---------|--------|
| `create` | `--from-json` (required, complete payload), `--yes`, optional assertion `--target recamera-pro` |
| `update` | `--id`, `--from-json` and/or `--path`, `--yes`, optional assertion `--target recamera-pro` |

Both commands load current `get_constant`. Create validates the complete payload; update first merges the patch with `view_model`, then applies the same checks and submits a normalized complete form. `--path` uploads first (type=2 + checksum) and overwrites `file_url`/`checksum`/`model_size` on the patch — same as Console edit. If `labels` or `file_url` change and the version string was left unchanged, patch version auto-increments (`1.0.0` → `1.0.1`). **Own published** models (`be_public=1`) can be updated; CLI sends `be_public=1` so they **re-enter review** (Console add.vue). Others' public-library ids are refused. Enforced checks include required fields, name/description lengths, labels, version, option ids, device-family exclusivity, device→format/file extension, checksum, and Pro task/framework rules. Non-Pro payloads strip `ai_framework`.

reCamera Pro is detected automatically when an `adapteds` entry resolves to `uniform_type=53`; `--target recamera-pro` is only an extra assertion. See [model-fields.md](model-fields.md).

`create` refuses payloads that still carry a **public-library** `id` / `model_id` / `from_id` (others' public models cannot be cloned). Upload your own file via `model upload`, then create. Public models are deploy-only.

`create` response always includes `model_id` + `console_url` (`https://sensecraft.seeed.cc/ai/model/detail?id=…&tab=custom`, or the origin in `SENSECRAFT_CONSOLE_HOST`). Show `console_url` to the user after success.

```bash
sensecraft-cli model create --from-json ./model.json --yes
sensecraft-cli model create --from-json ./pro-model.json --target recamera-pro --yes
sensecraft-cli model update --id <model_id> --from-json ./model.json --yes
sensecraft-cli model update --id <model_id> --path ./model.tflite --yes
sensecraft-cli model update --id <model_id> --path ./model.tflite --from-json ./patch.json --yes
```

### `model publish` / `model unpublish` (L2)

**Status: implemented.**

| Command | API `status` |
|---------|----------------|
| `publish` | `1` |
| `unpublish` | `0` |

Require a **separate chat confirmation** (see [confirmation.md](confirmation.md)), then `--yes`. 「发布这个 / 下架 / 删这个」is intent — ask first.

```bash
# After the user replies 确认:
# Submit to public library
sensecraft-cli model publish --id <model_id> --yes

# Return to private
sensecraft-cli model unpublish --id <model_id> --yes
```

### `model delete --id <id>` (L2)

**Status: implemented.** Maps to `delete_model`; requires `--yes`. Refuses `be_public≠0` (unpublish first).

User 「删这个63760」→ ask `将从{环境}环境删除模型 63760。删除后可能无法恢复。是否确认执行？` → wait for `确认` → then:

```bash
sensecraft-cli model delete --id <model_id> --yes
```

## Output

Success: indented JSON on stdout.  
Failure: stderr + non-zero exit.
