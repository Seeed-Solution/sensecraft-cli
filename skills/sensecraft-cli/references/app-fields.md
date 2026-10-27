# Application info & BOM field reference

Agent fill-in guide for SenseCraft custom apps (v3). Aligns with Console `infoForm.vue` / `billForm.vue` / `update_app`.

## App information

| Field                      | UI type                    | Required                   | Compliance                                                                                                                                                                                                                                            |
| -------------------------- | -------------------------- | -------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `app_name`                 | text                       | yes                        | max **60** chars, trim; CLI rejects empty / overlength before API                                                                                                                                                                                     |
| `excerpt`                  | text (textarea)            | yes                        | max **300** chars, trim; CLI rejects empty / overlength (align publish validateForm)                                                                                                                                                                  |
| `scene`                    | single select              | yes                        | string id from `app_scenarios_array`. **Resolve ids with `sensecraft-cli model constants`**; CLI rejects empty / unknown ids                                                                                                                          |
| `allow_clone`              | boolean → API `0`/`1`      | yes                        | whether others may clone; CLI rejects null / values other than 0\|1                                                                                                                                                                                   |
| `allow_commerce`           | boolean → API `0`/`1`      | yes                        | commercial use flag; CLI rejects null / values other than 0\|1                                                                                                                                                                                        |
| `images`                   | multi image upload → URL[] | yes                        | **JPG / GIF / PNG** (CLI also accepts WebP like Console); **≤ 5 MB** each; **max 16** (Console `maxImageCount`); **min 1**; first image = **cover**; prefer **4:3**. Also send `pic_url` (= `images[0]`). CLI rejects null / `[]` / non-http(s) URLs. |
| `license`                  | text / derived             | with full save             | string; often set with license confirmations                                                                                                                                                                                                          |
| `be_public`                | lifecycle                  | n/a for draft edit         | `0` private · `1` public (ops) · `2` pending review. User JSON must not set `1`. Published owner saves: CLI injects `1` (sketch). `2` locks edit.                                                                                                     |
| `app_id`                   | id                         | required on update         | from create/list/view                                                                                                                                                                                                                                 |
| `introduction`             | legacy long text           | no                         | if non-empty, editor may merge into intro blocks — prefer `deployment.blocks` for new content                                                                                                                                                         |
| `author`                   | string                     | recommended on save        | current user id/account                                                                                                                                                                                                                               |
| `activity_id`              | positive integer           | no                         | Join this activity. Omit or send blank to leave existing activity tags unchanged. CLI sends a JSON number. `0`, negatives, and non-integers are rejected before POST. Server: not a positive integer → `11202` `Request parameters are invalid`; missing / unpublished / deleted / outside start–end → `25000` `Activity does not exist or has expired` (the whole edit is discarded). |
| `cancel_activity_id`       | positive integer           | no                         | Leave this activity. Omit or send blank to leave participation unchanged. CLI sends a JSON number and does not change `allow_clone`. Must not equal `activity_id`. |
| `flow_data` / `model_data` | Node-RED sync              | only with `node-red` block | `flow_data` must be **JSON string** at top level                                                                                                                                                                                                      |

### Create vs update

- **Create (CLI one-shot)**: `POST /flow/app/v3/create_app` then `update_app` with the rest of `--from-json` on the **new** `app_id` only — **no `--yes`**. Editing an **existing** app is `app update --yes` (L2).
  - The follow-up `update_app` also merges missing info keys when a sketch already exists (same merge as `app update`), so an echoed create result may show sketch values for omitted keys.
  - CLI always surfaces `app_id` / `console_url` / `detail_url` after create, even when `update_app` returns `{}`.
  - **Never** retry `app create` solely because the JSON looked empty — run `app list --mine` first; duplicate names are allowed and will spawn extra apps.
- **Update (`app update`)**: always include `app_id` (CLI `--id`).
  - **Info fields** (`app_name`, `excerpt`, `scene`, `allow_clone`, `allow_commerce`, `images`, `license`, `author`): aiserver often **ignores** a body that only contains `app_name`. CLI **auto-merges** missing info keys from `app view --sketch` before calling `update_app` (does **not** merge `be_public` from user/existing for private apps). If current `be_public=1`, CLI **injects** `be_public=1` like Console info save (plaza unchanged until `app publish`). `be_public=2` refuses update. Agents may pass a sparse rename patch; still **re-read** with `app view` / `app list` after update because success may be `{}`.
  - **Other sections** (`deployment`, `materials`, …): may be sent alone without forcing an info merge. Full leave-save usually sends info + deployment + materials together.
  - **Activity**: `--activity-id <positive integer>` or JSON `activity_id`. Do not send `tags`; the server ignores it and the CLI strips it. A blank `activity_id` is omitted. If `activity_id` is set and this JSON omits `allow_clone`, CLI sets `allow_clone` to `1` when the current sketch is not already `1`. `--cancel-activity-id <positive integer>` or JSON `cancel_activity_id` leaves that activity and does not change `allow_clone`. Do not send the same id in both fields.
  - **Rename workflow**: confirm with user (L2) → `app update --id … --from-json '{"app_name":"…"}' --yes` (CLI fills the rest) → verify name via `app view --id … --sketch` or `app list --mine --name …`.

## Visibility / review

| User action                     | Value to send                     | Level      |
| ------------------------------- | --------------------------------- | ---------- |
| Keep / make private (unpublish) | `0`                               | L2 confirm |
| Submit for review               | `2`                               | L2 confirm |
| Become public `1`               | **Do not send as end-user Agent** | L4 Admin   |

API: `POST /flow/app/v3/edit_privacy_app` (CLI name: `publish` / `unpublish`).

## BOM (`materials[]`)

### Seeed Studio catalog row

| Field      | Required | Notes                                     |
| ---------- | -------- | ----------------------------------------- |
| `id`       | yes      | product **SKU**                           |
| `name`     | yes      | product name                              |
| `quantity` | yes      | string                                    |
| `note`     | yes      | remark (may be empty string if UI allows) |
| `buy_url`  | yes      | purchase link                             |

Pick products with `app bom search` (filters `material_options` from `get_production_options`).

**Price / image for Seeed rows:** `price` and `pic_url` **exist in the catalog** and are returned by `app bom search`, but they are **not part of `materials[]`**. Console re-looks them up by SKU (`billForm.vue` picker, `preview.vue` BOM total). So:

- Reporting a Seeed product's price/image to the user: **supported** — read them from `app bom search`.
- Writing them into the app: **not a field** — persist the SKU in `id` and the platform resolves price/image.

### Custom / other product row

| Field      | Required | Notes                                                                                       |
| ---------- | -------- | ------------------------------------------------------------------------------------------- |
| `name`     | yes      | product name                                                                                |
| `quantity` | yes      | string (Console UI integer stepper; API often string)                                       |
| `note`     | no       | optional remark (Console placeholder: optional)                                             |
| `buy_url`  | no       | optional purchase link (invalid URL may become `""`)                                        |
| `id`       | **omit** | custom rows must **not** include `id`                                                       |
| image      | —        | **Not supported**: the Console "其他" row has no image input and no upload path             |
| price      | —        | **Not supported**: no price input; `preview.vue` counts custom rows as `0` in the BOM total |

Custom rows are exactly `name` / `quantity` / `note` / `buy_url` (Console `buildMaterialsArray`). Do not invent `price` / `pic_url` / `image` keys — they are silently dropped, and the app will look "saved but lost".

### Example

```json
{
  "materials": [
    {
      "id": "113991254",
      "quantity": "1",
      "name": "Seeed Studio XIAO ESP32S3 Sense",
      "note": "Main MCU",
      "buy_url": "https://www.seeedstudio.com/"
    },
    {
      "quantity": "1",
      "name": "Custom enclosure",
      "note": "3D printed",
      "buy_url": "https://example.com/part"
    }
  ]
}
```
