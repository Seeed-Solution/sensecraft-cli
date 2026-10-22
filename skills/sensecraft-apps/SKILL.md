---
name: sensecraft-apps
description: >-
  Manages SenseCraft Application Plaza (应用广场 / App Square): search public apps,
  view detail (docs, tool-blocks, BOM), create/edit/publish/
  unpublish/delete apps. Use when the user mentions 应用广场, Application Store,
  app BOM, deploy tool-blocks, or SenseCraft app upload.
  Requires Access Key (`auth init --kind sensecap`). Not for SenseCAP devices.
---

# SenseCraft Apps (Application Plaza)

Domain skill for **SenseCraft AI applications**. Auth: **Access Key** → `{ai-host}/openapi` (HTTP Basic).

Parent router: [`../sensecraft-ai/SKILL.md`](../sensecraft-ai/SKILL.md).  
SenseCAP IoT: [`../sensecraft-data/SKILL.md`](../sensecraft-data/SKILL.md).  
Child skill: [`create-app/SKILL.md`](create-app/SKILL.md) — manifest-driven app lifecycle: mine a finished local project into a versioned `sensecraft-app.json` manifest (source of truth), create the app in one go, then local-first updates with drift detection (one batch confirmation per push). Publish / unpublish / delete stay in this file.

## Agent rules

1. Check auth first:
   ```bash
   sensecraft-cli auth info
   ```
   Need an active **SenseCAP Access Key**. If missing, ask the user to run `sensecraft-cli auth init --kind sensecap` locally. **Never** run `auth init` for them.
2. **L2 confirm is a separate chat turn** ([`../confirmation.md`](../confirmation.md)) before: `publish` (`--agree-declaration`), `unpublish`, `delete`, **`app update` / `bom set`**. 「删除应用 demo」is **not** confirmation — ask `将从{环境}环境删除应用 demo（ID: …）。删除后可能无法恢复。是否确认执行？`, wait for `确认`, then `--yes`. Never pass `--yes` on the same turn as the first request. **Exception — batch authoring:** [`create-app`](create-app/SKILL.md) confirms a whole push batch (fields + images + BOM + blocks) in one turn instead of per write; its one-go `app create` needs no confirmation at all.  
   **`app create` is not L2** — including `--from-json` with full fields (create + internal follow-up `update_app` on the **new** `app_id` only). Prefer one-shot create when the payload is ready. Still tell the user what will be created before running create.
3. **Privacy values**: submit review → `privacy` / flow uses **`2`** (pending); unpublish/private → **`0`**. **Never send `1`** (ops-approved public; Admin only).
4. Tool-block **configuration** may be written into `deployment.blocks`. **Executing** flash / USB deploy / container install on hardware is **L3**. Classic reCamera (`40`): [`sensecraft-recamera`](../sensecraft-recamera/SKILL.md) (`check` / `open`). Other benches: Console `/ai/device/local/32|36|37|41|52`. `flash-firmware` run is out of scope this pass.
5. Parse indented JSON on stdout; on failure read stderr + exit code. After `app update`, **re-read** the app — success body may be `{}`.
6. **Rename / edit info**: sparse `{"app_name":"…"}` is OK (CLI merges other info fields from sketch). Do not invent HTTP; do **not** put `be_public=1` in user JSON. For **already published** apps the CLI injects `be_public=1` (Console info save) and returns `sketch_saved` — plaza copy is unchanged until `app publish`.
7. **Lifecycle vs Console**: edit / publish / unpublish / delete / BOM / cover only for **your** apps (`user_id` == profile). Another user's plaza app cannot be edited. `be_public=2` locks owner edit. `be_public=1` owner may save sketch then `app publish` (草稿). Delete only `be_public=0`.
8. **Never blind-retry `app create`**: names are not unique. If create output lacks `app_id` or looks empty, run `app list --mine` before creating again.

## CLI status

| Area                                                                   | Status                                                                                                                                                                                 |
| ---------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Auth `--kind sensecap`                                                 | **Implemented** (`auth init --kind sensecap`; `app` uses `{ai-host}/openapi` Basic)                                                                                                    |
| `app list` / `app view` / `app fields` / `app blocks`                  | **Implemented**                                                                                                                                                                        |
| `app create` / `update` / `publish` / `unpublish` / `delete`           | **Implemented** (L2 = `--yes`; publish also `--agree-declaration`; **`create` never needs `--yes`**, even with full `--from-json`)                                                      |
| Cover / attachment upload                                              | `file upload --path <img> --type 1 --app-id <id> --yes` (prepends cover, **5 MB**). Unlinked: `file upload --type 1 --for-app` when building a new app payload.                        |
| `app bom search\|list\|set`                                            | **Implemented** (catalog = local filter on `get_production_options`; set via `update_app`)                                                                                             |
| `scene` / option ids                                                   | Use `sensecraft-cli model constants` (`app_scenarios_array`)                                                                                                                           |
| BOM **price / image**                                                  | Seeed rows: **readable** from `app bom search` (`price`, `pic_url`), resolved by SKU — not stored in `materials[]`. Custom rows: price/image **unsupported** (no Console input) — custom rows themselves are fine |

When a command exists, prefer it. When it does not, explain the gap and use field/block references below to prepare payloads — do not invent HTTP with guessed tokens; do not misuse SenseCAP CLI.

## Typical workflows

### Discover public apps

```bash
sensecraft-cli auth info
sensecraft-cli app list --public --name "<keyword>" --scene <id>
sensecraft-cli app list --public --uniform-type 40
sensecraft-cli app list --mine --uniform-type 40
sensecraft-cli app view --id <app_id>
```

Summarize: name, excerpt, scene, tool-block types used, BOM highlights, deploy steps.

### Rename an existing app (L2)

```bash
# Confirm with the user first. Sparse name-only JSON is OK — CLI merges other info fields.
echo '{"app_name":"Dimmer Tutorial · Grove Beginner Kit"}' \
  | sensecraft-cli app update --id <app_id> --from-json - --yes
sensecraft-cli app view --id <app_id> --sketch   # must show new app_name
```

### Join an activity (L2)

```bash
sensecraft-cli app update --id <app_id> --activity-id 1 --yes
```

`activity_id` is an optional positive integer. Omit it to leave existing activity tags unchanged. `tags` is ignored. A bad id fails the whole edit (`11202`); an unknown or inactive activity also fails the whole edit (`25000` `Activity does not exist or has expired`).

### Publish / unpublish / delete (L2)

```bash
# After user agrees to Publish Declaration + confirms submit:
sensecraft-cli app publish --id <app_id> --agree-declaration --yes
sensecraft-cli app unpublish --id <app_id> --yes    # privacy=0 private / withdraw review
sensecraft-cli app delete --id <app_id> --yes       # private only (be_public=0)
```

`app publish` matches Console 「发布确认」: show https://sensecraft.seeed.cc/legal/en/declaration and warn that review locks editing; only then pass `--agree-declaration --yes`. Missing either flag → non-zero. API body is still only `{app_id, privacy:2}` (agreement is client-side, like Console `warrantyAccepted`).

**Never** send `privacy=1` (ops-approved public). After editing a **published** app, tell the user plaza is unchanged until they confirm `app publish --agree-declaration --yes`.

### BOM

```bash
sensecraft-cli app bom search --name "XIAO"
sensecraft-cli app bom list --id <app_id>
# confirm with user, then:
sensecraft-cli app bom set --id <app_id> --from-json ./materials.json --yes
```

Seeed rows need `id`(=SKU). Custom rows omit `id`. No manual custom image upload.

### Author path

0. Resolve `scene`, inspect fields, and inspect block layouts:
   ```bash
   sensecraft-cli model constants
   sensecraft-cli app fields
   sensecraft-cli app blocks
   ```
   Use `app_scenarios_array` from constants and the two static catalogs; never invent option ids or block shapes.
1. Fill required app info (see [reference-app-fields.md](reference-app-fields.md)).
2. Build intro + stepped deploy + tool-blocks ([reference-tool-blocks.md](reference-tool-blocks.md)).
3. BOM: Seeed SKU rows and/or custom rows ([reference-app-fields.md](reference-app-fields.md#bom-materials))
4. **Prefer one-shot create** when JSON is complete (**no `--yes`**):
   ```bash
   sensecraft-cli app create --from-json ./app.json
   # CLI returns app_id + console_url + detail_url (even if update_app body was {}).
   # detail_url includes from=my-apps so cover/images load from the sketch.
   # Restore JSON may have pic_url (list) or images[] (view); CLI writes both.
   # Show links to the user. NEVER re-run create because stdout looked empty —
   # check first: app list --mine --name "<name>"
   ```
   Incremental path (empty shell, then L2 edit of existing app):
   ```bash
   sensecraft-cli app create --name "My App"
   sensecraft-cli app update --id <app_id> --from-json ./app.json --yes
   ```
5. User agrees to Publish Declaration + confirms → `app publish --agree-declaration --yes` (submit review)

Never put `be_public=1` / `privacy=1` in JSON; use `app publish`.

## App info fields (summary)

| Field            | Type               | Required        | Constraints                                                                    |
| ---------------- | ------------------ | --------------- | ------------------------------------------------------------------------------ |
| `app_name`       | text               | yes             | trim, max **60**                                                               |
| `excerpt`        | text               | yes             | trim, max **300**                                                              |
| `scene`          | option (string id) | yes             | id from `app_scenarios_array` — list via `sensecraft-cli model constants`      |
| `allow_clone`    | option 0/1         | yes             | boolean in UI                                                                  |
| `allow_commerce` | option 0/1         | yes             | boolean in UI                                                                  |
| `images`         | image URL[]        | yes             | JPG/GIF/PNG(/WebP), **≤5 MB**/image, **max 16**, first = cover, prefer **4:3** |
| `license`        | text               | often with save | string                                                                         |
| `be_public`      | 0/1/2              | lifecycle       | 0 private, 1 public (ops), 2 pending review                                    |

Full table + `update_app` keys: [reference-app-fields.md](reference-app-fields.md).

## Deploy docs + tool-blocks (summary)

Structure: `deployment.blocks` flat array.

- Blocks **before first `step`** = Introduction
- From each `step` until next `step` = one deploy step (text + tool-blocks mixed)

**High-priority tool-blocks** (pick by scenario):

| `type`                      | When to use                                               | Execute?                                            |
| --------------------------- | --------------------------------------------------------- | --------------------------------------------------- |
| `flash-firmware`            | Burn firmware to supported board                          | Config = L1/L2 write; **run flash = L3**            |
| `deploy-ai-model`           | Vision / audio AI model to device (task/device in config) | Config OK; **run deploy = L3**                      |
| `ai-vibration`              | Vibration detection on XIAO-class device                  | Config OK; run = L3                                 |
| `deploy-container-app`      | Edge container app in the app doc                         | Config OK; installing the container is L3            |
| `node-red` / `aily-blockly` | Flow / Blockly projects                                   | Lower priority unless user asks                     |

Details: [reference-tool-blocks.md](reference-tool-blocks.md).

## BOM (summary)

| Kind          | Keys                                              | Notes                                                                              |
| ------------- | ------------------------------------------------- | ---------------------------------------------------------------------------------- |
| Seeed catalog | `id` (SKU), `quantity`, `name`, `note`, `buy_url` | `price` / `pic_url` come from the catalog by SKU — report them, never persist them |
| Custom        | `quantity`, `name`, `note`, `buy_url`             | **No `id`**, and **no price / image** — Console offers no input for either         |

## APIs (implementation reference)

Base: `{host}/openapi` with HTTP Basic (`key_id`:`access_key`).

| Action  | Method | Path                                                             |
| ------- | ------ | ---------------------------------------------------------------- |
| List    | GET    | `/flow/app/v2/list_app` (`be_public=1` for plaza; optional `uniform_type`) |
| View    | GET    | `/flow/app/v3/view_app`                                          |
| Create  | POST   | `/flow/app/v3/create_app`                                        |
| Update  | POST   | `/flow/app/v3/update_app`                                        |
| Privacy | POST   | `/flow/app/v3/edit_privacy_app` (`2` submit / `0` private)       |
| Delete  | POST   | `/flow/app/v2/delete_app` (confirm path in code when wiring CLI) |

Authoritative field dump (Console repo): `website-sensecraft-console/docs/app-edit-v3-custom-app-field-reference.md`.

## Additional resources

- [commands.md](commands.md) — CLI flags for every shipped `app` subcommand
- [reference-app-fields.md](reference-app-fields.md) — app info + BOM compliance
- [reference-tool-blocks.md](reference-tool-blocks.md) — tool-block configs and selection
