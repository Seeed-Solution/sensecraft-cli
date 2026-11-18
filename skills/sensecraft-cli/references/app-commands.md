# sensecraft-apps — CLI commands

Target binary: `sensecraft-cli` (this repo). Auth: active **SenseCAP Access Key** (`auth init`) → `{ai-host}/openapi`.

> Status: **shipped**. Every command below is implemented — run it rather than describing it. If a flag is rejected, check `--help` instead of assuming the feature is missing.

## Auth prerequisite

```bash
sensecraft-cli auth info
sensecraft-cli auth init --site Global --no-wait   # split flow; see SKILL.md
```

## Commands

### `app list`

**Status: implemented.**

List apps.

| Flag                  | Default                 | Notes                                                        |
| --------------------- | ----------------------- | ------------------------------------------------------------ |
| `--public`            |                         | CodeCraft Hub (`be_public=1`). Required for the store; not the default |
| `--mine`              | default if neither flag | My apps. Omit both visibility flags for the same result     |
| `--name`              |                         | Search keyword → API `app_name` (Console app list)           |
| `--scene`             |                         | Scene id                                                     |
| `--uniform-type`      |                         | Device type → API `uniform_type` (`40` reCamera, `32` XIAO ESP32S3, `53` reCamera Pro) |
| `--sort`              |                         | `popular`, `newest`, or `likes` → `sort_by`. Omit to leave the API unordered |
| `--priority-activity-id` |                    | Positive integer. Only with `--sort popular`. If omitted on popular, CLI fills the first ongoing activity |
| `--mark-type`         |                         | `1` = my likes, `0` = my collections                         |
| `--likes`             |                         | `mark_type=1` and `be_public=1`. Not with `--mine`           |
| `--collections`       |                         | `mark_type=0` and `be_public=1`. Not with `--mine`           |
| `--page` / `--length` | 1 / 20                  | Pagination                                                   |
| `--sandbox`           |                         | Sample JSON, no network. Includes `sandbox_query`           |

```bash
# CodeCraft Hub. Omit visibility flags for my apps. Hot sort is explicit.
sensecraft-cli app list --public --name "vision" --scene <scene_id> --page 1 --length 20
sensecraft-cli app list --public --sort popular --length 12
sensecraft-cli app list --public --sort newest
sensecraft-cli app list --public --likes
sensecraft-cli app list --public --collections
sensecraft-cli app list --public --uniform-type 40
sensecraft-cli app list --mine --uniform-type 40

# My apps
sensecraft-cli app list --mine
sensecraft-cli app list --public --mark-type 1
```

### `app view`

**Status: implemented.**

| Flag        | Required | Notes                                                              |
| ----------- | -------- | ------------------------------------------------------------------ |
| `--id`      | yes      | `app_id`                                                           |
| `--sketch`  | no       | Draft / sketch (`view_sketch=1`). Default = published snapshot (`0`) |
| `--sandbox` | no       | Sample JSON                                                        |

```bash
sensecraft-cli app view --id <app_id>            # published snapshot (view_kind=published)
sensecraft-cli app view --id <app_id> --sketch   # draft / sketch (view_kind=sketch)
```

Stdout always includes CLI markers:

- `view_kind`: `"sketch"` \| `"published"`
- `view_sketch`: `1` \| `0`

These mark **which snapshot this command requested**. They are not the same as API `is_sketch` (whether a draft differs from the published copy).

**After `app create` / `app update`:** always re-read with `--sketch`. On a private draft, default `app view` is often only the create shell (name, little or no deployment). Full info, BOM, and flash-firmware blocks live in the sketch until `app publish` promotes them. See [apps.md — Sketch vs published](apps.md#sketch-vs-published-snapshots).

**Hub flash testing (private draft):** use the editor (`console_url`), or open the app from my-apps / user-center. Do not assume the bare `detail_url` preview shows sketch content.

### `app fields`

**Status: implemented.** Prints JSON schema (app info + BOM). Requires AI auth profile; static output, zero network — **no `--sandbox` flag; run bare**.

```bash
sensecraft-cli app fields
```

### `app blocks`

**Status: implemented.** Prints deploy layout + tool-block catalog. Requires AI auth profile; static output, zero network — **no `--sandbox` flag; run bare**.

```bash
sensecraft-cli app blocks
```

### `app doc compile`

**Status: implemented.** Compiles a Markdown file into editor `deployment.blocks`. No Access Key, no network, no `--sandbox`. Does not upload files and does not insert tool-blocks.

```bash
sensecraft-cli app doc compile --from-md ./README.md
# stdin: sensecraft-cli app doc compile --from-md -
```

Use stdout `deployment` in `app create` / `app update`, or skip this step and pass the file as `app create --from-md` / `app update --from-md`. `suggested_app_name` is the opening `# title`. `suggested_excerpt` is the paragraph under it. `pending_uploads` lists local image/video/file paths that were left out of `content`; upload those, then put the https URL in `src` or `href`.

`--from-json` rejects text blocks whose `content` is still Markdown (`**bold**`, `` `code` ``, a leading `#` or list marker). `code` block source is exempt. The same write fills a paragraph that is only a video URL, a remote file link, and a remote image up to the editor's HTML. A video URL inside a sentence is left as text.

### `app create` / `app update`

**Status: implemented.**

#### `app create` (not L2)

**Intent:** one-shot authoring. Full `--from-json` is allowed **without** `--yes`.

| Flag          | Notes                                                                                                                                                                       |
| ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--name`      | `app_name` only create (empty shell). Optional when JSON or the Markdown title provides one                                                                                |
| `--from-md`   | Markdown path or `-`. Compiled into editor blocks. Fills `app_name` / `excerpt` when omitted. Appends JSON `deployment.blocks` after the document. stdout adds `pending_uploads` and `warnings` |
| `--from-json` | JSON path or `-`; must include `app_name` if `--name` and `--from-md` omit it. Extra fields → follow-up `update_app` **only on the newly created `app_id`** (cannot target an existing app). Text blocks get editor video, link, and image HTML |
| `--sandbox`   | sample JSON                                                                                                                                                                 |

Rejects `be_public=1` / `privacy=1`. Do **not** confuse with `app update` (existing app → L2).

**Success output:** CLI always prints `app_id`, `console_url`, and `detail_url` on CodeCraft Hub (`https://codecraft.seeed.cc/hub`, or the base in `SENSECRAFT_APP_SITE` when that variable is set):

- edit: `/application/edit?id=<app_id>`
- app detail: `/view-application/<app_id>-<slug>`

Owner and plaza links are the same path. Do not append `from`, `section`, `state`, or `tab`. The site shows the author's draft when they are logged in, and the published page to everyone else. If a list item has a non-empty `web_link`, give the user that URL instead of `view-application`. Model pages use `https://sensecraft.seeed.cc` unless `SENSECRAFT_CONSOLE_HOST` is set. Legal pages stay on `sensecraft.seeed.cc`.

**Do not retry create blindly:**

- Same `app_name` can be created multiple times (no server-side uniqueness).
- If stdout looks empty/`{}` or you are unsure, run `app list --mine --name "<app_name>"` **before** another `app create`.
- Prefer updating the existing `app_id` with `app update --yes` over creating a duplicate.

```bash
# Markdown document. Scene, cover, BOM, and tool-blocks stay in the JSON.
sensecraft-cli app create --from-md ./README.md --from-json ./app.json

# One-shot full create: deliberately not L2
sensecraft-cli app create --from-json ./app.json
# Expect app_id + console_url + detail_url; if missing, list before retrying:
sensecraft-cli app list --mine --name "My App"

# Empty shell
sensecraft-cli app create --name "My App"
```

#### `app update` (L2)

| Flag            | Notes                                                                                                      |
| --------------- | ---------------------------------------------------------------------------------------------------------- |
| `--id`          | required                                                                                                   |
| `--from-json`   | path or `-`; optional when `--from-md` or `--activity-id` is set                                          |
| `--from-md`     | Markdown path or `-`. Same compile as create. Does not replace an `app_name` / `excerpt` already in the JSON |
| `--activity-id` | optional positive integer. Joins that activity. Omit or pass empty to leave existing activity tags as-is |
| `--cancel-activity-id` | optional positive integer. Leaves that activity. Does not change `allow_clone` |
| `--yes`         | after user confirmation                                                                                    |
| `--sandbox`     | sample JSON                                                                                                |

Rejects `be_public=1` / `privacy=1` in **user** payload. If the app is already public (`be_public=1`), CLI injects `be_public=1` on info saves (Console) and JSON includes `sketch_saved`. Under review (`be_public=2`): **refused**. Others' apps (`user_id` ≠ profile): **refused**.

**Info patch merge:** if `--from-json` touches any of `app_name` / `excerpt` / `scene` / `allow_clone` / `allow_commerce` / `images` / `license` / `author`, CLI loads `view_app` (sketch) and fills missing info keys before `update_app`. Sparse `{"app_name":"…"}` rename is OK. Materials/deployment-only JSON is **not** force-merged with info. Private info saves still do **not** copy `be_public` from the existing view.

**Always verify** after update (`app view` / `app list`) — API may return `{}` on success.

```bash
# Ask the user before changing an existing app:
# Rename (sparse OK — CLI merges other info fields):
echo '{"app_name":"New Name"}' | sensecraft-cli app update --id <app_id> --from-json - --yes
sensecraft-cli app view --id <app_id> --sketch   # confirm app_name

sensecraft-cli app update --id <app_id> --from-json ./app.json --yes

# Join activity 1. activity_id is sent as a JSON number. tags in JSON is dropped.
sensecraft-cli app update --id <app_id> --activity-id 1 --yes

# Leave activity 1. cancel_activity_id is sent as a JSON number.
sensecraft-cli app update --id <app_id> --cancel-activity-id 1 --yes
```

`activity_id` must be a positive integer. List every activity currently being held with `app activity list` first. Before POST, CLI loads that same server list. If that request succeeds and does not contain the id, CLI does not POST. A failed list request is ignored. The server still rejects a bad id with `11202` and a missing or expired activity with `25000` `Activity does not exist or has expired` (the whole edit is discarded). Omitting `activity_id` does not change existing activity tags. `tags` is not read by the server.

`cancel_activity_id` must be a positive integer. CLI sends it as a JSON number and does not check the ongoing list or change `allow_clone`. Omitting it leaves participation unchanged. The same id cannot be both `activity_id` and `cancel_activity_id`.

When `--activity-id` is set and this JSON does not include `allow_clone`, CLI sets `allow_clone` to `1` unless the sketch is already `1`. That matches the web contest save. An explicit `allow_clone` in the JSON is kept.

### `app publish` / `app unpublish` / `app delete` (L2)

**Status: implemented.**

- `publish` → `privacy=2` (submit review). Same path as Console **发布草稿** when the app is already public (`is_sketch` + `be_public=1`). Result may include `draft_publish`. Refused if already `be_public=2` or not owner. stdout `detail_url` is `/view-application/<id>-<slug>` with no query. Give the user that URL only. Do not include `console_url` or `/application/edit`: review locks editing, and opening the editor returns `The application must not allow editing`. Do not append navigation params or replace the detail URL with `/application/detail/<id>`.
- **Publish Declaration (required for `publish` only):** matches Console 「发布确认」`warrantyAccepted`. Before passing `--agree-declaration`, show the user:
  - Review will lock editing until approval (or `app withdraw`). Do not use `unpublish` to leave review
  - Declaration URL: https://sensecraft.seeed.cc/legal/en/declaration
  - Do **not** pass `--agree-declaration` unless the user explicitly agreed
- `unpublish` → `privacy=0` for an **already public** app (make it private). Owner only. No declaration flag. Do **not** use it to withdraw a review.
- `withdraw` → `POST /flow/app/v3/callback_sketch` with `{app_id}` only, and only when `be_public=2`. Public apps: `unpublish`. Private apps: nothing to withdraw.
- `delete` → `POST /flow/app/v2/delete_app` — **private + owner** (`be_public=0`)
- **Never** send privacy `1`
- All require a **separate chat confirm** ([confirmation.md](confirmation.md)) then `--yes`; `publish` also requires `--agree-declaration`. 「删除应用 demo」is intent — ask first.

```bash
# after the user replies 确认:
sensecraft-cli app publish --id <app_id> --agree-declaration --yes
sensecraft-cli app unpublish --id <app_id> --yes
sensecraft-cli app withdraw --id <app_id> --yes
sensecraft-cli app delete --id <app_id> --yes
```

### `app activity list`

**Status: implemented.** Read-only. No Access Key. `GET /activity/list` with no `time` query. The server defaults to the current time and returns every activity currently being held, not a page and not activities that have ended or not started. The CLI does not filter that set again. stdout is `{ "count": N, "list": [...] }` with numeric `activity_id`, `name`, `guide_url`, and `start_time` / `end_time` converted to UTC (`2026-10-18T16:00:00Z`). `count` is the number of activities in this result. An empty result is `{ "count": 0, "list": [] }`. Rows missing an id, a name, or either time are omitted. Tag ids are not shown.

```bash
sensecraft-cli app activity list
```

The returned `guide_url` is the source for requirements and official rules; requirements are not included as a structured CLI field. Read candidate guides and compare them with the project before selecting an activity. Follow [activity selection during authoring](app-fields.md#activity-selection-during-app-authoring) for presenting the candidates, no-activity option, and recommendation that the user chooses from. Do not confuse `priority_activity_id` (list ordering) with `activity_id` (joining).

### `app like` / `app unlike` / `app collect` / `app uncollect` / `app mark-deploy` (L2)

**Status: implemented.** `POST /flow/app/v2/mark_app`.

| Command | `op_type` |
| --- | --- |
| `like` | `1` |
| `unlike` | `0` |
| `collect` | `3` |
| `uncollect` | `2` |
| `mark-deploy` | `4` (records a download; does not flash) |

stdout always includes `app_id` and `op_type`. Whether the user already liked or collected is `like_time` / `follow_time` on `app view` (non-zero means set). Counts are `like_num` / `follow_num`.

```bash
sensecraft-cli app like --id <app_id> --yes
sensecraft-cli app mark-deploy --id <app_id> --yes
```

### `app clone` (L2)

**Status: implemented.** `POST /flow/app/v3/clone_app`. Refuses when source `allow_clone` is `0` and does not POST. Without `--name`, the name is `Clone_` plus the first 54 characters of the source name. stdout includes the new `app_id`, `console_url`, and `detail_url`. Does not copy `be_public` and does not publish.

```bash
sensecraft-cli app clone --id <source_app_id> --yes
sensecraft-cli app clone --id <source_app_id> --name "New name" --yes
```

### `app bom`

**Status: implemented.**

| Subcommand                   | Notes                                                |
| ---------------------------- | ---------------------------------------------------- |
| `search [--name]`            | `get_production_options` → filter `material_options` |
| `list --id [--sketch]`       | materials from `view_app`; use `--sketch` to verify saved draft rows |
| `set --id --from-json --yes` | `update_app` materials[] (L2)                        |

Search returns `{count, list}` in catalog order. Every query token must occur in the product name, SKU, category, note or description after case and punctuation normalization; `ESP32S3` and `ESP32-S3` are equivalent. `XIAO ESP32-S3 1.47`, `1.47 Touch`, and `Touch Display ESP32-S3` all find `XIAO 1.47" Touch Display (ESP32-S3)`. An exact SKU query also matches. Each returned catalog `block_support` is preserved, with `block_supported: true` for `"是"` and `false` otherwise.

```bash
sensecraft-cli app bom search --name "XIAO"
sensecraft-cli app bom list --id <app_id>
# Ask the user before replacing materials:
sensecraft-cli app bom set --id <app_id> --from-json ./materials.json --yes
```

## Output

Success: indented JSON on stdout (business `data` only).  
Failure: stderr + non-zero exit.
