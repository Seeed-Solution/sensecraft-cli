# CodeCraft Hub

Site: https://codecraft.seeed.cc/hub

Domain skill for **CodeCraft Hub applications**. Auth: **Access Key** (HTTP Basic). The China site supports SenseCraft Data only; use Global for CodeCraft Hub.

Parent router: [`../SKILL.md`](../SKILL.md).  
SenseCAP IoT: [`data.md`](data.md).  
Child skill: [`../create-app/SKILL.md`](../create-app/SKILL.md) — turn a local project into a deployable app: discover project facts, resolve uncertain decisions, write and review human-facing copy, configure tool-block deployment, confirm BOM catalog matches through the API, and build the versioned `codecraft-app.json` before upload. Read each related reference before authoring its information. Later updates follow the manifest with drift detection and one batch confirmation per push. Publish / unpublish / delete stay in this file.

## Agent rules

1. Check auth first:
   ```bash
   sensecraft-cli auth info
   ```
   Need an active **SenseCAP Access Key**. If missing, use the split `auth init` flow in [`../SKILL.md`](../SKILL.md). Do not run bare `auth init`.
2. **L2 confirm is a separate chat turn** ([confirmation.md](confirmation.md)) before: `publish` (`--agree-declaration`), `unpublish`, `withdraw`, `delete`, `like`, `unlike`, `collect`, `uncollect`, `mark-deploy`, `clone`, **`app update` / `bom set`**. 「删除应用 demo」is **not** confirmation — ask `将从{环境}环境删除应用 demo（ID: …）。删除后可能无法恢复。是否确认执行？`, wait for `确认`, then `--yes`. Never pass `--yes` on the same turn as the first request. **Exception — batch authoring:** [`create-app`](../create-app/SKILL.md) confirms a whole push batch (fields + images + BOM + blocks) in one turn instead of per write; its one-go `app create` needs no confirmation at all.  
   **`app create` is not L2** — including `--from-json` with full fields (create + internal follow-up `update_app` on the **new** `app_id` only). Prefer one-shot create when the payload is ready. Still tell the user what will be created before running create.
3. **Privacy values**: submit review → `privacy` / flow uses **`2`** (pending); unpublish/private → **`0`**. **Never send `1`** (ops-approved public; Admin only).
4. Tool-block **configuration** may be written into `deployment.blocks`. **Executing** flash / USB deploy / container install on hardware is **L3**. Classic reCamera (`40`): [`sensecraft-recamera`](recamera.md) (`check` / `open`). Other benches: Console `/ai/device/local/32|36|37|41|52`. `flash-firmware` run is out of scope this pass.
5. Parse indented JSON on stdout; on failure read stderr + exit code. After `app update`, **re-read with `--sketch`** — success body may be `{}`, and default `app view` is the published snapshot (often an empty create shell / name-only on private drafts).
6. **Rename / edit info**: sparse `{"app_name":"…"}` is OK (CLI merges other info fields from sketch). Do not invent HTTP; do **not** put `be_public=1` in user JSON. For **already published** apps the CLI injects `be_public=1` (Console info save) and returns `sketch_saved` — plaza copy is unchanged until `app publish`.
7. **Lifecycle vs Console**: edit / publish / unpublish / withdraw / delete / BOM / cover only for **your** apps (`user_id` == profile). Another user's plaza app cannot be edited. `be_public=2` locks owner edit and has no editor link; unlock with `app withdraw`, not `app unpublish`. `be_public=1` owner may save sketch then `app publish` (草稿). `app unpublish` is only for an already public app. Delete only `be_public=0`.
8. **Never blind-retry `app create`**: names are not unique. If create output lacks `app_id` or looks empty, run `app list --mine` before creating again.
9. **Sketch vs published** ([below](#sketch-vs-published-snapshots)): `app create` / `app update` write the **sketch**. Verify with `app view --sketch`. Do not treat default `app view` or a bare Hub preview URL as proof that CLI content is live.

## Sketch vs published snapshots

Each app has two `view_app` snapshots:

| Snapshot | CLI | `view_sketch` | Written by |
| -------- | --- | ------------- | ---------- |
| **Sketch** (draft) | `app view --id … --sketch` | `1` | `app create` follow-up save, `app update`, Hub editor save, BOM set |
| **Published** | `app view --id …` (default) | `0` | `create_app` shell (often name-only); promoted on `app publish` |

`app view` stdout always includes CLI markers (not server fields):

- `view_kind`: `"sketch"` or `"published"`
- `view_sketch`: `1` or `0`

Do **not** confuse these with `is_sketch` from the API (`1` means a draft differs from the published snapshot).

**Private drafts (`be_public=0`):** default `app view` (no `--sketch`) is frequently only the create shell — no excerpt, deployment, or flash blocks — while `--sketch` shows what CLI just wrote. Always re-read with `--sketch` after create/update.

**Hub pages:**

- **Editor** (`console_url` / `/hub/application/edit?id=…`): loads sketch (`view_sketch=1`). Flash-firmware blocks here see CLI content. Keep the editor closed while CLI writes; an open editor can auto-save older content over a fresh `app update`.
- **Preview** (`detail_url` = `/hub/view-application/<id>-<slug>`, no query): visitors see the published snapshot. Author login *tries* to show the draft, but a private draft's published snapshot may still look empty. Prefer the **editor** (or a my-apps / user-center entry) to test flash on a private draft — not the bare preview URL alone.
- **Plaza /「查看已发布版本」** (`from=application-store`): always the published snapshot.

**When the published snapshot updates:** create shell at `app create`; then usually only when `app publish` promotes the sketch (submit review / publish draft). Daily editor or CLI saves stay on the sketch. For an already-public app, plaza stays on the old published copy until `app publish` again (`sketch_saved` hint).

## CLI status

| Area                                                                   | Status                                                                                                                                                                                 |
| ---------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Access Key                                                         | **Implemented** (`auth init` or `auth add`; `app` uses `{ai-host}/openapi` Basic)                                                                                      |
| `app list` / `app view` / `app fields` / `app blocks` / `app activity list` | **Implemented**. CodeCraft Hub hot list is `--public --sort popular` |
| `app doc compile` | **Implemented**. Markdown → editor `deployment.blocks`. No auth, no network. `app create --from-md` / `app update --from-md` compile then write. `--from-json` fills video, file-link, and image HTML, and rejects raw Markdown in text-block `content` |
| `app create` / `update` / `publish` / `unpublish` / `withdraw` / `delete` | **Implemented** (L2 = `--yes`; publish also `--agree-declaration`; **`create` never needs `--yes`**, even with full `--from-json`). `withdraw` is review-only (`be_public=2`); `unpublish` is for an already public app |
| `app like` / `unlike` / `collect` / `uncollect` / `mark-deploy` / `clone` | **Implemented** (L2 = `--yes`) |
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

Have the user close this app's CodeCraft Hub editor page before CLI updates and reopen it after verification. The open editor saves the copy it loaded and can replace CLI changes made meanwhile. Apply this sequence to info, deployment, BOM, cover, and activity updates.

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

`activity_id` is an optional positive integer. List ids with `app activity list`. Omit it to leave existing activity tags unchanged. `tags` is ignored. A bad id fails the whole edit (`11202`); an unknown or inactive activity also fails the whole edit (`25000` `Activity does not exist or has expired`). If this update sets `--activity-id` and the JSON does not set `allow_clone`, CLI writes `allow_clone=1` unless the sketch is already `1`.

### Leave an activity (L2)

```bash
sensecraft-cli app update --id <app_id> --cancel-activity-id 1 --yes
```

`cancel_activity_id` is an optional positive integer sent as a JSON number. It does not change `allow_clone` and is not checked against `app activity list`. Omit it to leave participation unchanged. Do not send the same id as `activity_id` in one update.

### Publish / unpublish / delete (L2)

```bash
# After user agrees to Publish Declaration + confirms submit:
sensecraft-cli app publish --id <app_id> --agree-declaration --yes
sensecraft-cli app unpublish --id <app_id> --yes    # already public (be_public=1) → private
sensecraft-cli app withdraw --id <app_id> --yes     # under review (be_public=2) only
sensecraft-cli app delete --id <app_id> --yes       # private only (be_public=0)
```

`app publish` matches Console 「发布确认」: show https://sensecraft.seeed.cc/legal/en/declaration and warn that review locks editing until approval or `app withdraw`; only then pass `--agree-declaration --yes`. Missing either flag → non-zero. API body is still only `{app_id, privacy:2}` (agreement is client-side, like Console `warrantyAccepted`). stdout includes `detail_url`: `https://codecraft.seeed.cc/hub/view-application/<id>-<slug>`. Give the user that URL as printed. Do not include `console_url` or `/application/edit`. Do not append `from`, `section`, `state`, or `tab`. Do **not** invent `/application/detail/<id>`.

**Never** send `privacy=1` (ops-approved public). After editing a **published** app, tell the user plaza is unchanged until they confirm `app publish --agree-declaration --yes`.

### BOM

```bash
sensecraft-cli app bom search --name "XIAO"
sensecraft-cli app bom list --id <app_id>
# confirm with user, then:
sensecraft-cli app bom set --id <app_id> --from-json ./materials.json --yes
```

Seeed rows need `id`(=SKU). Custom rows omit `id`. No manual custom image upload.

`app bom search` matches every normalized query token against the product name, SKU, category, note or description, ignoring case and punctuation. `XIAO ESP32-S3 1.47`, `1.47 Touch`, and `Touch Display ESP32S3` match `XIAO 1.47" Touch Display (ESP32-S3)`; exact SKU searches also work. Results keep catalog order and return `{count, list}`. Catalog rows retain `block_support` and add boolean `block_supported` (`true` for `"是"`). Object outputs that contain `list` and have no `count` gain `count` for the current result; existing `page` and a server-provided `count` stay as they are.

### Author path

For a local project/repository or an existing `codecraft-app.json`, use [create-app](../create-app/SKILL.md) for the full discovery, writing review, BOM verification, and manifest workflow. The abbreviated commands below apply to content that is already authored and reviewed; do not bypass that workflow by uploading a developer README directly.

0. Resolve `scene`, inspect fields, and inspect block layouts:
   ```bash
   sensecraft-cli model constants
   sensecraft-cli app fields
   sensecraft-cli app blocks
   ```
   Use `app_scenarios_array` from constants and the two static catalogs; never invent option ids or block shapes.
1. Fill required app info (see [app-fields.md](app-fields.md)).
2. Build intro + stepped deploy + tool-blocks ([tool-blocks.md](tool-blocks.md)). When the source is Markdown, run `sensecraft-cli app create --from-md <file>` (add `--from-json` for scene, cover, BOM, and tool-blocks). JSON `deployment.blocks` are appended after the compiled document. `app doc compile --from-md` only previews. Do not leave `**bold**` or list markers in JSON `content`.
3. BOM: Seeed SKU rows and/or custom rows ([app-fields.md](app-fields.md#bom-materials))
4. **Prefer one-shot create** when JSON is complete (**no `--yes`**):
   ```bash
   sensecraft-cli app create --from-json ./app.json
   # CLI returns app_id + console_url + detail_url (even if update_app body was {}).
   # detail_url is https://codecraft.seeed.cc/hub/view-application/<id>-<slug>
   # (test AI host or remote_type Develop → https://test-codecraft.seeed.cc/hub). No query.
   # Content landed in the sketch — verify with app view --sketch (view_kind=sketch).
   # For private drafts, test flash on the editor (console_url), not the bare preview alone.
   # A non-empty web_link on a list item is the link to give the user.
   # Restore JSON may have pic_url (list) or images[] (view); CLI writes both.
   # Show links to the user. NEVER re-run create because stdout looked empty —
   # check first: app list --mine --name "<name>"
   ```
   Incremental path (empty shell, then L2 edit of existing app):
   ```bash
   sensecraft-cli app create --name "My App"
   sensecraft-cli app update --id <app_id> --from-json ./app.json --yes
   sensecraft-cli app view --id <app_id> --sketch   # confirm blocks / flash config
   ```
5. User agrees to Publish Declaration + confirms → `app publish --agree-declaration --yes` (submit review). After that, give `detail_url` only. `be_public=2` has no editor link.

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

Full table + `update_app` keys: [app-fields.md](app-fields.md).

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

Details: [tool-blocks.md](tool-blocks.md).

## BOM (summary)

| Kind          | Keys                                              | Notes                                                                              |
| ------------- | ------------------------------------------------- | ---------------------------------------------------------------------------------- |
| Seeed catalog | `id` (SKU), `quantity`, `name`, `note`, `buy_url` | `price` / `pic_url` come from the catalog by SKU — report them, never persist them |
| Custom        | `quantity`, `name`, `note`, `buy_url`             | **No `id`**, and **no price / image** — Console offers no input for either         |

## APIs (implementation reference)

Base: `{host}/openapi` with HTTP Basic (`key_id`:`access_key`).

| Action  | Method | Path                                                             |
| ------- | ------ | ---------------------------------------------------------------- |
| List    | GET    | `/flow/app/v2/list_app` (`be_public`, `order_type` `1`/`2`/`3`, `priority_activity_id`, `mark_type` `1`/`0`) |
| Activities | GET | `/activity/list` (no `time`; server default is now, every activity currently being held) |
| View    | GET    | `/flow/app/v3/view_app`                                          |
| Create  | POST   | `/flow/app/v3/create_app`                                        |
| Update  | POST   | `/flow/app/v3/update_app`                                        |
| Clone   | POST   | `/flow/app/v3/clone_app`                                         |
| Privacy | POST   | `/flow/app/v3/edit_privacy_app` (`2` submit / `0` private). Not for withdrawing review |
| Withdraw | POST  | `/flow/app/v3/callback_sketch` (`be_public=2` only)              |
| Mark    | POST   | `/flow/app/v2/mark_app` (`op_type` 1/0/3/2/4)                    |
| Delete  | POST   | `/flow/app/v2/delete_app`                                        |

Authoritative field dump (Console repo): `website-sensecraft-console/docs/app-edit-v3-custom-app-field-reference.md`.

## Additional resources

- [app-commands.md](app-commands.md) — CLI flags for every shipped `app` subcommand
- [app-fields.md](app-fields.md) — app info + BOM compliance
- [tool-blocks.md](tool-blocks.md) — tool-block configs and selection
