# sensecraft-apps — CLI commands

Target binary: `sensecraft-cli` (this repo). Auth: active **SenseCAP Access Key** (`auth init --kind sensecap`) → `{ai-host}/openapi`.

> Status: **shipped**. Every command below is implemented — run it rather than describing it. If a flag is rejected, check `--help` instead of assuming the feature is missing.

## Auth prerequisite

```bash
sensecraft-cli auth info
sensecraft-cli auth init --kind sensecap   # user only, interactive
```

## Commands

### `app list`

**Status: implemented.**

List apps.

| Flag                  | Default                 | Notes                                                        |
| --------------------- | ----------------------- | ------------------------------------------------------------ |
| `--public`            | plaza mode              | `be_public=1`                                                |
| `--mine`              | default if neither flag | My apps                                                      |
| `--name`              |                         | Search keyword → API `app_name` (Console app list)           |
| `--scene`             |                         | Scene id                                                     |
| `--uniform-type`      |                         | Device type → API `uniform_type` (`40` reCamera, `32` XIAO ESP32S3, `53` reCamera Pro) |
| `--mark-type`         |                         | Optional mark filter, such as `like` / `star` when supported |
| `--page` / `--length` | 1 / 20                  | Pagination                                                   |
| `--sandbox`           |                         | Sample JSON, no network                                      |

```bash
# Application Plaza
sensecraft-cli app list --public --name "vision" --scene <scene_id> --page 1 --length 20
sensecraft-cli app list --public --uniform-type 40
sensecraft-cli app list --mine --uniform-type 40

# My Apps / marked apps
sensecraft-cli app list --mine
sensecraft-cli app list --mine --mark-type star
```

### `app view`

**Status: implemented.**

| Flag        | Required | Notes                        |
| ----------- | -------- | ---------------------------- |
| `--id`      | yes      | `app_id`                     |
| `--sketch`  | no       | Draft view (`view_sketch=1`) |
| `--sandbox` | no       | Sample JSON                  |

```bash
sensecraft-cli app view --id <app_id>
sensecraft-cli app view --id <app_id> --sketch
```

### `app fields`

**Status: implemented.** Prints JSON schema (app info + BOM). Requires AI auth profile; no network.

```bash
sensecraft-cli app fields
```

### `app blocks`

**Status: implemented.** Prints deploy layout + tool-block catalog. Requires AI auth profile; no network.

```bash
sensecraft-cli app blocks
```

### `app create` / `app update`

**Status: implemented.**

#### `app create` (not L2)

**Intent:** one-shot authoring. Full `--from-json` is allowed **without** `--yes`.

| Flag          | Notes                                                                                                                                                                       |
| ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--name`      | `app_name` only create (empty shell)                                                                                                                                        |
| `--from-json` | JSON path or `-`; must include `app_name` if `--name` omitted. Extra fields → follow-up `update_app` **only on the newly created `app_id`** (cannot target an existing app) |
| `--sandbox`   | sample JSON                                                                                                                                                                 |

Rejects `be_public=1` / `privacy=1`. Do **not** confuse with `app update` (existing app → L2).

**Success output:** CLI always prints `app_id`, `console_url` (edit `/ai/application/edit?id=…`), and `detail_url` (`/ai/application/detail/<id>?tab=custom&from=my-apps`). The `from=my-apps` query loads the sketch (cover/`images`) the same as Console 我的应用. Do **not** send users to `/application/detail/<id>` without that query — that page fetches `view_sketch=0` and will look like it has no cover after create/restore.

**Do not retry create blindly:**

- Same `app_name` can be created multiple times (no server-side uniqueness).
- If stdout looks empty/`{}` or you are unsure, run `app list --mine --name "<app_name>"` **before** another `app create`.
- Prefer updating the existing `app_id` with `app update --yes` over creating a duplicate.

```bash
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
| `--from-json`   | path or `-`; required unless `--activity-id` is set                                                       |
| `--activity-id` | optional positive integer. Joins that activity. Omit or pass empty to leave existing activity tags as-is |
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
```

`activity_id` must be a positive integer. The server rejects anything else with `11202` `Request parameters are invalid`, and rejects a missing, unpublished, deleted, or out-of-window activity with `25000` `Activity does not exist or has expired`. Either failure discards the whole edit, including other fields in the same request. Omitting `activity_id` does not change existing activity tags. `tags` is not read by the server.

### `app publish` / `app unpublish` / `app delete` (L2)

**Status: implemented.**

- `publish` → `privacy=2` (submit review). Same path as Console **发布草稿** when the app is already public (`is_sketch` + `be_public=1`). Result may include `draft_publish`. Refused if already `be_public=2` or not owner.
- **Publish Declaration (required for `publish` only):** matches Console 「发布确认」`warrantyAccepted`. Before passing `--agree-declaration`, show the user:
  - Review will lock editing until approval (or `unpublish`)
  - Declaration URL: https://sensecraft.seeed.cc/legal/en/declaration
  - Do **not** pass `--agree-declaration` unless the user explicitly agreed
- `unpublish` → `privacy=0` (private / withdraw review). Owner only. No declaration flag.
- `delete` → `POST /flow/app/v2/delete_app` — **private + owner** (`be_public=0`)
- **Never** send privacy `1`
- All require a **separate chat confirm** ([`../confirmation.md`](../confirmation.md)) then `--yes`; `publish` also requires `--agree-declaration`. 「删除应用 demo」is intent — ask first.

```bash
# after the user replies 确认:
sensecraft-cli app publish --id <app_id> --agree-declaration --yes
sensecraft-cli app unpublish --id <app_id> --yes
sensecraft-cli app delete --id <app_id> --yes
```

### `app bom`

**Status: implemented.**

| Subcommand                   | Notes                                                |
| ---------------------------- | ---------------------------------------------------- |
| `search [--name]`            | `get_production_options` → filter `material_options` |
| `list --id`                  | materials from `view_app`                            |
| `set --id --from-json --yes` | `update_app` materials[] (L2)                        |

```bash
sensecraft-cli app bom search --name "XIAO"
sensecraft-cli app bom list --id <app_id>
# Ask the user before replacing materials:
sensecraft-cli app bom set --id <app_id> --from-json ./materials.json --yes
```

## Output

Success: indented JSON on stdout (business `data` only).  
Failure: stderr + non-zero exit.
