# Manifest reference — `sensecraft-app.json`

The manifest is the local source of truth for one SenseCraft app. It is **not** the CLI payload verbatim: it carries local-only bookkeeping (paths, hashes, sync state) that must be stripped when building `--from-json` bodies.

Top-level sections: `manifest_version`, `identity`, `info`, `images`, `deployment`, `bom`, `sync`.

## `manifest_version`

Integer, currently `1`. Bump only when the schema changes shape.

## `identity` — which remote app, which account

| Key | Filled when | Notes |
|-----|-------------|-------|
| `app_id` | after create | `null` until created. Once set, NEVER re-run create for this manifest |
| `remote_type` | preflight | `Global` / `China` / `Private` from `auth info`. Update flow hard-stops on mismatch. CodeCraft Hub requires Global; China supports SenseCraft Data only |
| `org_id` | preflight | from `auth info` |
| `account` | preflight | from `auth info` |
| `created_at` | after create | **Local UTC timestamp recorded when create succeeds** — the CLI returns no created time |

## `info` — app info gate fields

| Key | Rule |
|-----|------|
| `app_name` | non-empty, ≤ 60 UTF-16 code units |
| `excerpt` | non-empty, ≤ 300 UTF-16 code units |
| `scene` | string id from `app_scenarios_array` (`sensecraft-cli model constants`) |
| `license` | string, may be `""` |
| `author` | string; default the `auth info` account |
| `allow_clone` | `0` or `1` only |
| `allow_commerce` | `0` or `1` only |

Full field compliance: [../../references/app-fields.md](../../references/app-fields.md).

## `images` — local paths with upload state

Array, order matters: **index 0 is the cover** (becomes `pic_url`). Up to 16 entries. May be empty at create time — a private draft works without images; a cover is required before publishing.

| Key | Notes |
|-----|-------|
| `path` | local file, relative to the manifest |
| `sha256` | hex digest of file bytes; `null` until computed. Drives re-upload: upload only when changed or `file_url` is null |
| `file_url` | returned by `file upload --type 1 --for-app`; `null` until uploaded |
| `role` | `cover` for index 0, `gallery` otherwise |

Formats JPG/JPEG/PNG/GIF/WebP, ≤ 5 MB each, prefer 4:3. SVG is rejected by the CLI.

## `deployment.blocks`

Flat ordered array, copied through to the payload unchanged. Blocks before the first `step` = introduction; each `step` starts a deploy step. Tool-block configs (`flash-firmware`, `deploy-ai-model`, …): [../../references/tool-blocks.md](../../references/tool-blocks.md). Firmware URLs land in `config.firmware_url` after a `file upload --type 4`.

## `bom.materials`

Seeed catalog rows carry `id` (SKU), `quantity`, `name`, `note`, `buy_url` — never `price`/`pic_url` (resolved by SKU server-side). Custom rows carry `name` + `quantity` (required) and optionally `note` / `buy_url` — no `id`, no image, no price. Details: [../../references/app-fields.md](../../references/app-fields.md#bom-materials).

## `sync` — local vs remote bookkeeping

| Key | Notes |
|-----|-------|
| `local_revision` | integer, `+1` on every local edit after creation of the manifest |
| `last_pushed_revision` | `local_revision` value as of the last successful create/push. `0` + set `app_id` = half-finished create → resume via update |
| `last_pushed_at` | RFC3339 of last successful push |
| `last_pushed_hash` | drift baseline — see the pinned algorithm below |

**`last_pushed_hash` algorithm** (compute exactly this way so every session reproduces the same baseline):

1. Build the projection object `{"info": {...}, "images": ["<file_url>", ...], "deployment": {...}, "materials": [...]}` from the pushed payload — derived keys like `pic_url` are **excluded** (the hash covers the source fields, not derived duplicates).
2. Serialize as canonical JSON: UTF-8, keys sorted, no insignificant whitespace (Python: `json.dumps(obj, sort_keys=True, separators=(",", ":"), ensure_ascii=False)`).
3. `last_pushed_hash = "sha256:" + hex(SHA-256 of those bytes)`.

On the drift check, apply the same projection to the `app view --sketch` response before hashing, with these equivalence rules:

- **Same key set**: a key absent from the remote view is dropped from **both** sides before hashing (server-normalized views may omit keys).
- **Type normalization**: coerce values to the manifest's forms — e.g. a remote `"scene": 1` (number) compares as `"1"` (string).
- **Derived keys excluded** on both sides (`pic_url`, `app_id`, timestamps, review/lifecycle fields like `be_public`).

## Manifest → CLI payload

Build `--from-json` bodies by projection — keep server fields, drop local-only ones:

| Manifest | Payload key |
|----------|-------------|
| `info.*` | same names at top level |
| `images[].file_url` (in order) | `images` (array of URLs); `pic_url` = first element. **Empty → omit both keys entirely** — the info gate rejects `images: []`, but accepts the key being absent |
| `deployment` | `deployment` |
| `bom.materials` | `materials` |
| `identity`, `sync`, `images[].path/sha256/role`, `manifest_version` | **never sent** |

Never include `be_public` / `privacy` in any payload.

## Worked example

See [../assets/manifest-template.json](../assets/manifest-template.json) for a fresh manifest (pre-create), and below for the same manifest after a successful create:

```json
{
  "manifest_version": 1,
  "identity": {
    "app_id": "12345",
    "remote_type": "Global",
    "org_id": "org-9",
    "account": "maker@example.com",
    "created_at": "2026-10-20T08:30:00Z"
  },
  "info": {
    "app_name": "Jar Lid Counter",
    "excerpt": "Counts screw-cap jars on a conveyor with a XIAO ESP32S3 Sense and a FOMO model.",
    "scene": "7",
    "license": "MIT",
    "author": "maker@example.com",
    "allow_clone": 1,
    "allow_commerce": 0
  },
  "images": [
    {
      "path": "docs/images/cover.png",
      "sha256": "b5bb9d…",
      "file_url": "https://files.seeedstudio.com/…/cover.png",
      "role": "cover"
    }
  ],
  "deployment": {
    "blocks": [
      { "id": "intro_1", "type": "paragraph", "content": "<p>What this app does.</p>", "indent": 0 },
      { "id": "step_1", "type": "step", "title": "Step 1: Flash firmware", "content": "Step 1: Flash firmware", "expanded": true, "indent": 0 },
      {
        "id": "ff_1",
        "type": "flash-firmware",
        "config": {
          "device_id": "",
          "device_type": "",
          "firmware_url": "https://files.seeedstudio.com/…/firmware.bin",
          "firmware_size": "412316"
        }
      }
    ]
  },
  "bom": {
    "materials": [
      {
        "id": "113991254",
        "quantity": "1",
        "name": "Seeed Studio XIAO ESP32S3 Sense",
        "note": "Main MCU",
        "buy_url": "https://www.seeedstudio.com/"
      }
    ]
  },
  "sync": {
    "local_revision": 1,
    "last_pushed_revision": 1,
    "last_pushed_at": "2026-10-20T08:31:02Z",
    "last_pushed_hash": "sha256:9f2c…"
  }
}
```
