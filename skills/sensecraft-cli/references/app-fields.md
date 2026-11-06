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
| `flow_data` / `model_data` | Node-RED sync              | only with `node-red` block | Block `config.flow_data` is an object or array; `config.model_data` is an object or null. Neither is a JSON string inside the block. On save the app writes `JSON.stringify(config.flow_data)` to this top-level `flow_data` string and copies `model_data` here. |

### Activity selection during app authoring

For a new app, assess participation using the project discovery results; do not wait for the user to name a competition. Respect any established preference to join or skip one. Run:

```bash
sensecraft-cli app activity list
```

Read [the activity command reference](app-commands.md#app-activity-list) before using the result. `app activity list` returns every activity currently being held: the server already limits the list to the current time, and the CLI does not drop rows by time again. Each row has `activity_id`, `name`, UTC `start_time`/`end_time`, and `guide_url`. It does **not** return a structured requirements field. Open and read the API-provided guide for each plausible candidate, including its official rules where linked, before claiming eligibility. Missing/inaccessible guidance is an information gap, not proof that there are no requirements. Do not invent an activity-detail CLI or guessed API endpoint.

Compare the documented theme, entry window/deadline, eligibility, supported/required hardware or features, required artifacts/demo/docs, and licensing/clone/commercial conditions with the actual project and settled user choices. Distinguish what the project already satisfies from deliverables still needed. A successful empty list means no currently available activity; a failed request is not an empty list. Sandbox output is not a live activity or eligibility check.

- **Clear suitable activity:** select it with a concise evidence-based reason. If several qualify and one is clearly preferable for this project, explain that choice. Do not use result order or a popular-list `priority_activity_id` as the entry decision.
- **No suitable activity:** select no participation and explain the mismatch or lack of open activities. Omit `activity_id`; do not use `0`, invent an ID, or ask a redundant opt-in question when the evidence supports the decision.
- **Uncertain choice or eligibility:** ask with concrete options: candidate names/IDs, dates with a timezone, important requirements and project fit/gaps, plus **no activity**. Recommend the best-supported option. Reuse known answers. Do not change the project's agreed permissions or invent missing eligibility facts to make an entry fit. If new permission choices or required consent are unresolved, ask about those with the activity options.

Record the chosen activity or no-activity reason, lookup time, API/guide sources, applicable rules, and project evidence in local review notes. For `create-app`, store the optional selection in `info.activity_id` and include its positive integer at payload top level on the initial one-shot create; the CLI applies it in the new app's follow-up save. Always send the resolved `allow_clone` explicitly: the CLI may otherwise turn it on when joining an activity. An explicit `allow_clone: 0` is technically preserved by the CLI, but does not establish eligibility for a contest requiring cloning.

Recheck the current list and time window just before submitting. An invalid/expired activity can reject the entire follow-up save after an app ID already exists: preserve that ID, resolve the activity choice, and resume by update rather than creating a duplicate. Verify any exposed saved association and distinguish a join request accepted by the API from verified participation. Activity association does not by itself complete entry requirements or publish the app.

For an existing app, preserve participation unless changing it is part of the requested batch. Omission/null/blank leaves participation unchanged, not removed; leaving uses `cancel_activity_id` through the parent workflow. Do not repeatedly send a historical `activity_id` during unrelated edits.

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

### Catalog verification before authoring a BOM

Read this reference before writing BOM rows. Derive required devices, app-specific accessories, variants, and quantities from project evidence, then resolve catalog identity through the live API:

```bash
sensecraft-cli app bom search --name "<product name or SKU>"
```

The CLI filters `material_options` from `get_production_options` and returns `list[]` and `count`. Match the **exact variant**, not merely the first similar name. A result's **`sku` becomes the saved material row's `id`**; its internal product `id`, if present, is not the SKU. Copy the returned `name` and `buy_url`; derive `quantity` (string) and the app-specific `note` from the project. Do not fabricate SKU or purchase URLs from memory or examples.

Record the query and selected SKU/name/URL in local review notes, with the project evidence for quantity and purpose. Several plausible matches need a user decision. A successful no-match search after reasonable name/SKU variants permits a custom row; an auth/network/API failure leaves the candidate **unverified**, not absent from the catalog. Sandbox results are simulated and cannot confirm a real product. Report custom rows and unverified candidates separately from API-confirmed catalog rows. A project with no required hardware may use an empty BOM.

Catalog verification proves product identity; it does not prove the app saved that row. After create or BOM update, use `app bom list --id <app_id> --sketch` and compare the saved rows with the manifest.

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
      "id": "113991115",
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
