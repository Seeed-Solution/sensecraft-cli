# Manifest reference — `codecraft-app.json`

The manifest is the local source of truth for one CodeCraft Hub app. It is **not** the CLI payload verbatim: it carries local-only bookkeeping (paths, hashes, sync state) that must be stripped when building `--from-json` bodies.

Read this contract and the template before constructing a manifest or payload. First read [app-fields.md](../../references/app-fields.md) for info/BOM, [tool-blocks.md](../../references/tool-blocks.md) for deployment, and [content-review.md](content-review.md) for human-facing copy. Serialization follows project discovery and resolution of material decisions.

Top-level sections: `manifest_version`, `identity`, `info`, `images`, `deployment`, `bom`, `sync`.

Canonical filename: `codecraft-app.json`, or `codecraft-app.<slug>.json` for multiple apps. Legacy `sensecraft-app.json` / `sensecraft-app.<slug>.json` files remain discoverable and migrate by filename only; preserve app identity and sync state, carry forward review notes and any retained baseline under the new stem without overwriting existing files, and leave historical snapshots intact. This naming change does not change `manifest_version` or the API payload.

The template is an unfilled scaffold: null permission choices must be resolved from the user/session; empty images/blocks do not propose a cover or deployment plan. It is not ready to submit. Keep evidence, decisions, BOM query/selected catalog results, artifact paths/hashes and review findings in `.sensecraft/<manifest-stem>.review.md`. This sidecar is local-only and never projected into API fields. Final app values belong only in the canonical manifest.

## `manifest_version`

Integer, currently `1`. Optional backward-compatible fields such as `info.activity_id` do not require migration; older version-1 manifests may omit them. Bump the version for an incompatible required shape or interpretation change.

## `identity` — which remote app, which account

| Key | Filled when | Notes |
|-----|-------------|-------|
| `app_id` | after create | `null` until created. Once set, NEVER re-run create for this manifest |
| `remote_type` | preflight | `Global` / `China` / `Private` from `auth info`. Update flow hard-stops on mismatch. CodeCraft Hub requires Global; China supports SenseCraft Data only |
| `org_id` | preflight | from `auth info` |
| `account` | preflight | account reported by the completed auth wizard or otherwise verified for this profile; `auth info` does not show it. Keep `null` if unknown |
| `created_at` | after create | **Local UTC timestamp recorded when create succeeds** — the CLI returns no created time |

## `info` — app info gate fields

| Key | Rule |
|-----|------|
| `app_name` | non-empty string, ≤ 60 UTF-16 code units |
| `excerpt` | non-empty string, ≤ 300 UTF-16 code units |
| `scene` | string id from `app_scenarios_array` (`sensecraft-cli model constants`) |
| `license` | string, may be `""` |
| `author` | string; use the authenticated account when established, or an explicitly chosen author; otherwise `""`. Do not invent an account from `auth info` |
| `allow_clone` | integer `0` or `1` only in the canonical manifest; no implicit default |
| `allow_commerce` | integer `0` or `1` only in the canonical manifest; no implicit default |
| `activity_id` | optional positive integer from the current `app activity list`, within the CLI's int64 range. `null`/absent means no join selected for a new app; never use `0` or a string |

Full field compliance: [../../references/app-fields.md](../../references/app-fields.md).

### Activity selection and participation

Read [activity selection guidance](../../references/app-fields.md#activity-selection-during-app-authoring) before filling `info.activity_id`. Use project facts and the API/guide requirements to choose a suitable activity or none; ask with candidate options plus no activity when the decision is uncertain. The template's `null` is an unselected optional field, not a reason to skip the assessment. Record the activity name/ID, time window, rules/guide links, fit or gaps, and rationale in review notes.

Project `info.activity_id` to top-level `activity_id` on the initial create, or when an update/resume intentionally joins an activity. Always preserve the explicit clone/commercial flags; resolve any conflict with activity requirements rather than relying on the CLI's implicit clone opt-in. Recheck the current API list before submitting. After success, retain the selection as local metadata and record the join result separately in review notes; routine edits must not resend a historical ID. Omission/null on an update preserves existing participation and does not leave an activity. Leaving is an explicit `cancel_activity_id` operation in the parent workflow.

`activity_id` is a join instruction, not a scalar that `app view` is guaranteed to echo. Check any exposed saved activity association, but do not claim participation verified if the response lacks it. Keep selection/submission evidence separately from the content drift baseline. Association also does not establish that a competition entry meets all submission or publication requirements.

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

Flat ordered array, copied through to the payload unchanged. Blocks before the first `step` = introduction; each `step` starts a deploy step. Read [../../references/tool-blocks.md](../../references/tool-blocks.md) before selecting or configuring tools. After `file upload --type 4`, put the URL on the `flash-firmware` block as flat `firmware_url` (plus `firmware_name` and string `firmware_size`). Do not nest those fields under `config`. The editor wraps them into `config` when the doc is opened. A needed block with missing inputs may stay in a local draft, but the app is not deployment-ready until those inputs have a usable path; mark gaps in review notes rather than inventing URLs or IDs.

## `bom.materials`

Seeed catalog rows carry `id` (SKU), `quantity`, `name`, `note`, `buy_url` — never `price`/`pic_url` (resolved by SKU server-side). Set `id` from the live `app bom search` result's `sku`, not an internal product ID. Preserve the lookup evidence in review notes. Custom rows carry `name` + `quantity` (required) and optionally `note` / `buy_url` — no `id`, no image, no price. Read [catalog verification and BOM fields](../../references/app-fields.md#catalog-verification-before-authoring-a-bom) before writing rows; custom entries are not API-confirmed catalog products.

Names are non-empty strings and quantities are positive integer strings such as `"1"`. A custom row omits `id` entirely; `"id": null` is not a custom row. Notes and purchase URLs, when supplied, are strings; an optional empty purchase URL is allowed, but a non-empty one must be HTTP(S). Local shape validation cannot prove a catalog match or the correct quantity.

## `sync` — local vs remote bookkeeping

| Key | Notes |
|-----|-------|
| `local_revision` | positive integer; increment once per logical batch of app-content/asset changes. Updating identity, upload bookkeeping within that batch, or sync metadata does not itself create another revision |
| `last_pushed_revision` | `local_revision` at the last verified complete push. `0` + set `app_id` = half-finished create → resume via update |
| `last_pushed_at` | RFC3339 timestamp with timezone of the last verified complete push, or `null` before one |
| `last_pushed_hash` | full `sha256:<64 lowercase hex digits>` digest of the retained verified baseline, or `null` before one |

### Retain the comparison baseline

A hash alone cannot support field-level diffs or a three-way merge after local edits. After verifying the saved sketch and BOM, retain the exact normalized comparison object in `.sensecraft/<manifest-stem>.baseline.json`. This file is the last verified snapshot, not the current app source. Preserve it between sessions; keep it out of the API payload. Updating the review notes must not destroy the baseline.

Build that object as `{"info": {...}, "images": ["<URL>", ...], "deployment": {...}, "materials": [...]}` using the stable info fields (`app_name`, `excerpt`, `scene`, `license`, `author`, `allow_clone`, `allow_commerce`), ordered image URLs, complete deployment, and BOM. Exclude the optional join instruction `activity_id` from this `info` projection; review notes retain that action's evidence/result. Accept only understood server normalization; reconcile it into the canonical manifest before recording success. Compare normalized saved values with the intended values first. A partial save or unresolved difference must not advance `last_pushed_*` or replace the baseline.

**Hash algorithm:** serialize the comparison object as UTF-8 canonical JSON with sorted keys, no insignificant whitespace, and Unicode preserved (`json.dumps(obj, sort_keys=True, separators=(",", ":"), ensure_ascii=False)`). Hash those bytes with SHA-256 and prefix the full hexadecimal digest with `sha256:`. Array order is significant.

### Compare on update or resume

1. Load the retained baseline and check its digest against `sync.last_pushed_hash`. Compare it with the current local projection and the latest remote sketch/BOM to identify local changes, remote changes, and real conflicts.
2. Normalize equivalent representations consistently: numeric scene/SKU/device-type/size/quantity values to their canonical strings; clone/commercial flags to integers; image objects to URL strings. For `flash-firmware` only, flatten editor-wrapped `config` back onto the block, retaining its fields. Keep other tool configs nested. Treat missing optional empty note/URL strings consistently. Preserve block, image, and BOM order and every substantive tool field.
3. Exclude the `activity_id` join instruction, derived duplicates (`pic_url`, top-level Node-RED mirrors), and unrelated identity/lifecycle/timestamp fields from all content-drift projections. Assess any intentional participation change separately, using review evidence and the current activity rules. Do not drop a non-empty stable managed field from both sides merely because a remote response omitted it; that is a verification gap to resolve. Do not normalize away a new/deleted/reordered block, a changed URL, or an unknown substantive field.
4. If an old manifest has only a hash and no retained baseline, recover it from a trusted last-pushed snapshot when possible. Otherwise inspect the remote and present the concrete differences for reconciliation before writing. Do not use the already-edited local manifest as an assumed old baseline.

Only a verified complete create/update replaces the retained baseline and advances `last_pushed_*`. Keep successful upload URLs and known app IDs after partial failures so the next run resumes the same app.

## Manifest → CLI payload

Build `--from-json` bodies by projection — keep server fields, drop local-only ones:

| Manifest | Payload key |
|----------|-------------|
| `info.*` except `activity_id` | same names at top level |
| `info.activity_id` | top-level positive integer `activity_id` on initial create or an intentional join/resume. Null/absent → omit the key entirely. Routine update → omit even if a historical selection is retained |
| `images[].file_url` (in order) | `images` (array of URLs); `pic_url` = first element. **Empty → omit both keys entirely** — the info gate rejects `images: []`, but accepts the key being absent |
| `deployment` | `deployment` |
| `bom.materials` | `materials` |
| `node-red` block `config.flow_data` | additionally derive top-level `flow_data` as a JSON **string**, when present |
| `node-red` block `config.model_data` | additionally copy top-level `model_data` as an object, when present |
| `identity`, `sync`, `images[].path/sha256/role`, `manifest_version` | **never sent** |

Node-RED mirrors are derived at projection time, not stored as competing top-level manifest values. If several Node-RED blocks have conflicting flow/model snapshots, resolve which one should supply the app-level mirrors before uploading. Read the Node-RED section of the tool-block reference first.

Never include `be_public` / `privacy` in any payload, or local review/baseline data. Populate pending upload URLs in the canonical manifest before generating a payload; do not send null image URLs or local filesystem paths as remote URLs.

## Local validation and its limits

Run `python3 <skill-dir>/assets/validate_manifest.py <manifest> --scene-ids <actual-comma-separated-ids>` before uploads and again after inserting upload URLs. It rejects malformed sections, unsupported versions, incorrect canonical field types, invalid image/BOM/block shapes, raw Markdown in prose blocks, and recognizable credential values without echoing them. Missing images or pending tool inputs can be warnings; a warning-bearing pass does not establish deployment readiness.

The helper intentionally uses a conservative UTF-16 budget for names/excerpts; the current Go CLI counts Unicode code points. It checks file extensions/size/hash for images, not their visual content. It checks the optional activity ID's numeric shape, not whether the activity is open, suitable, or joined. It does not contact an API, confirm catalog products or scene freshness, scan firmware/archive bytes, prove device compatibility, validate every tool-specific config, or perform the editorial review. Those checks remain part of the discovery/review/upload workflow.

## Worked example

The template is a blank scaffold; the snapshot below illustrates a filled, coverless manifest. It describes a fictional timer: IDs, scene, permissions, device/product matches, and URLs must be resolved for the actual project. The reserved `.invalid` URLs deliberately cannot serve artifacts and are **not** an upload-ready result or evidence of API confirmation. The full digest is computed from this example's comparison projection, which would also be retained in the baseline file after a verified real push.

```json
{
  "manifest_version": 1,
  "identity": {
    "app_id": "12345",
    "remote_type": "Global",
    "org_id": "example-org",
    "account": "maker@example.com",
    "created_at": "2026-10-28T08:00:00Z"
  },
  "info": {
    "app_name": "Lap Timer",
    "excerpt": "Time laps with physical buttons and read elapsed time on the Wio Terminal display.",
    "scene": "5",
    "license": "MIT",
    "author": "maker@example.com",
    "allow_clone": 1,
    "allow_commerce": 0,
    "activity_id": null
  },
  "images": [],
  "deployment": {
    "blocks": [
      {
        "id": "intro_1",
        "type": "paragraph",
        "content": "The timer runs on the board and shows elapsed time on its display.",
        "indent": 0
      },
      {
        "id": "step_1",
        "type": "step",
        "title": "Flash the timer",
        "content": "Flash the timer",
        "expanded": true,
        "indent": 0
      },
      {
        "id": "para_1",
        "type": "paragraph",
        "content": "Connect the Wio Terminal over USB, then use the firmware tool below.",
        "indent": 0
      },
      {
        "id": "ff_1",
        "type": "flash-firmware",
        "device_id": "102991299",
        "device_type": "",
        "flash_mode": "bin",
        "firmware_url": "https://files.example.invalid/lap-timer.bin",
        "firmware_name": "lap-timer.bin",
        "firmware_size": "412316",
        "flash_address": "0x4000",
        "muse_gadget": false,
        "full_flash": false
      },
      {
        "id": "step_2",
        "type": "step",
        "title": "Time a lap",
        "content": "Time a lap",
        "expanded": true,
        "indent": 0
      },
      {
        "id": "para_2",
        "type": "paragraph",
        "content": "Restart the board. Press the start button to run the timer and the lap button to record an elapsed time on the display.",
        "indent": 0
      }
    ]
  },
  "bom": {
    "materials": [
      {
        "id": "102991299",
        "name": "Wio Terminal",
        "quantity": "1",
        "note": "Runs the timer and displays elapsed time",
        "buy_url": "https://shop.example.invalid/wio-terminal"
      }
    ]
  },
  "sync": {
    "local_revision": 1,
    "last_pushed_revision": 1,
    "last_pushed_at": "2026-10-28T08:01:00Z",
    "last_pushed_hash": "sha256:880e59411f09e0e462e0bc10e160fe344b91c576bf8fe886d44eb6cce5f31261"
  }
}
```
