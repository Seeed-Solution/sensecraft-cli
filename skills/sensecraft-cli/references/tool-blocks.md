# Deploy docs & tool-block reference

Help the Agent choose the right block and fill `config` safely. Execution on hardware is **L3** unless noted.

## Plan before writing blocks

Read this entire reference before drafting a deployment plan or tool config. Inspect `sensecraft-cli app blocks` as the static catalog, then map the real project journey to supported actions. For each action identify prerequisites, the suitable tool, actual artifact/config sources, and the observable result. Select only tools the project needs, and place each immediately after its instructions in the flat array.

Check the exact hardware variant, artifact target/architecture, deployment inputs, and device rules below. A local file awaiting upload is a planned upload; a missing artifact or unknown target is an unresolved deployment gap. Preserve a needed tool in the local draft while resolving its inputs. Do not substitute a prose-only instruction that makes an unsupported action appear executable. Report such gaps and agree on a supported path or an explicitly incomplete draft. A format-valid block is not evidence that deployment will work.

## Content format

The editor does **not** render Markdown on read. Preview wraps each block by `type` and treats `content` as HTML or plain text. Raw `**bold**`, `# heading`, or `- item` inside `content` shows up as those characters.

Write inline formatting as HTML the editor stores: `<strong>`, `<em>`, `<code>`, `<s>`, links with `target="_blank"`, file links (pdf/zip/bin/…) also with `download` and `doc-file-download-link`, images with `draggable="false"`. A paragraph that is only a video URL becomes `<video controls>` or a `video-embed-wrapper` iframe (YouTube, TikTok, Douyin, Bilibili, Vimeo, Youku, iQiyi). Compile Markdown rather than writing that HTML by hand. For interleaved prose/tools, compile locally, insert tools at their intended positions, and send the final blocks through `--from-json`. Combining `--from-md` with JSON tool blocks appends those tools after the entire compiled document; it does not place them into earlier steps. `--from-json` fills video, link, and image attributes when they are missing. Do not wrap a `paragraph` in an extra `<p>`. `code` block `content` is plain source, not HTML.

Compile a Markdown file with `sensecraft-cli app doc compile --from-md <file>` when you only want the JSON (no network). `app create --from-md` and `app update --from-md` run that compile and then write. JSON text blocks that still contain Markdown markers are rejected. How the toolbar maps onto blocks: [../../../docs/md-to-app.md](../../../docs/md-to-app.md).

Host binaries (images, videos, firmware) via `file upload` first. A relative path from the Markdown file is not a valid `src`.

## Document layout

`deployment.blocks` is a **flat** ordered array.

“Flat” describes the array of blocks, not a tool's settings. Every tool block, including `flash-firmware`, stores its settings inside one `config` object. The app site reads `block.config.*` and does not read a second copy on the block. Flattened settings belong to the separate legacy `deployment[].tools[]` format.

```text
[ intro text blocks... ] [ step ][ text|tools... ] [ step ][ text|tools... ] ...
```

| Region | Rule |
|--------|------|
| Introduction | All blocks **before** the first `type === "step"` |
| One deploy step | A `step` block + following blocks until the next `step` |

### Text blocks (`content` HTML/text)

`step`, `paragraph`, `h1`–`h6`, `bullet`, `number`, `todo`, `quote`, `code`, `divider`.

Common fields: `id`, `type`, `content`, `indent`, `title`/`expanded` for `step`.

**Step titles**: the UI already prefixes the step number (`Step x:` / `第 x 步`) — `title` is the action name only.

Prefer `paragraph` under a `step` for instructions; do not overload tool-block `config` with prose.

## Tool-block selection

| Priority | `type` | Use when | Avoid when |
|----------|--------|----------|------------|
| High | `flash-firmware` | User must burn a firmware image to a board | They only need model deploy |
| High | `deploy-ai-model` | Vision classify/detect or audio model on SenseCraft devices | Pure vibration → `ai-vibration`; pure firmware → `flash-firmware` |
| High | `ai-vibration` | Abnormal vibration detection flow | General vision/audio models |
| High | `deploy-container-app` | Edge **container** app deploy UX in the doc | Confusing it with a SenseCAP device bind |
| Lower | `node-red` | Node-RED flow + optional model binding | No flow artifact |
| Lower | `aily-blockly` | Aily Blockly public project | No project id |

Multiple tool-blocks may appear in one step (e.g. paragraph → flash → paragraph → deploy-ai-model).

## High-priority configs

### `flash-firmware`

Write every field below inside **`block.config`**. Keep `id` and `type` on the block. Do not also put these fields on the block, and do not add a `config.config` wrapper. The app site reads `config` only. `app create` / `app update` copy a flat-only block into `config`, then remove the flat fields. When both copies exist and differ, `config` is kept.

| Field | How to write |
|-------|----------------|
| `device_id` | Material SKU string, e.g. `"100017057"`, matching the BOM row's SKU in `id`. Not a production type (`"29"`) or an internal catalog product id |
| `device_type` | `uniform_type` string. Use `""` when the board has none (Wio / Grove may) |
| `flash_mode` | `"bin"` when a firmware URL is set. `"zip"` for a project zip |
| `firmware_url` / `firmware_name` / `firmware_size` | Bin mode. `firmware_size` is a string, not a number. Empty `firmware_name` is filled from the URL by the editor |
| `flash_address` | Only `"0x0"`, `"0x4000"`, `"0x10000"`, `"0x90000"`. Use the device default below. The editor resets a non-default address the first time the doc is opened. Do not write `0x20000` here |
| `project_zip_url` / `project_zip_name` / `project_zip_size` / `precheck_fields` | Zip mode only. Sizes are strings. `precheck_fields` is `[{ key, name, info, type, default }]` from `sensecraft_config.h`, or omit / `[]` |
| `muse_gadget` | JSON boolean, default `false`; write `true`/`false`, never strings or numbers. Some read paths coerce non-booleans by truthiness, so `"false"` can enable the option. `true` shows the Muse SDK token input under the flash button in preview |
| `full_flash` | JSON boolean, default `false`. Kept only on the Muse SKUs below; any other device is reset to `false`. `true` erases the whole flash before the write. When `muse_gadget` is `true`, set `full_flash` to `false`. The editor hides the checkbox while Muse is on |

`muse_gadget: true` is kept only for these SKUs. The CLI and `validate_manifest.py` reject `muse_gadget: true` or `full_flash: true` for any other `device_id`. The editor resets those flags to `false` when the document is opened. `full_flash` uses the same SKU list:

| Device | `device_id` (SKU) |
|--------|-------------------|
| SenseCAP Indicator D1 / D1S / D1L / D1Pro | `114993068` (aliases on the same row are kept) |
| SenseCAP Watcher | `113991315` or `113991275` |
| reTerminal E1001 | `100073581` |
| reTerminal E1002 | `100017057` |
| XIAO 1.47" Touch Display | `100069905` |
| reTerminal Sticky | `100056818` |

When `muse_gadget` is `true`, set `full_flash` to `false`. The editor hides that checkbox until Muse is turned off. `firmware_url` must be `muse-gadget.bin` linked at `0x20000`. A binary linked at `0x90000` or `0x10000` will not boot. Checking the box does not change `flash_address`: E100x stays `0x90000`, Sticky / Indicator / 1.47" Touch stay `0x0`, Watcher stays on the XIAO address rule.

`full_flash: true` with `muse_gadget: false` is the switch back from a Muse image to another firmware. E1001 / E1002 then also rewrite the stock bootloader at `0x0` and the partition table at `0x8000`; the app stays at `0x90000`. Indicator, 1.47" Touch, Sticky, and Watcher non-Muse images are already merged at `0x0`, so the flag only turns on erase-all. Do not put the reTerminal stock bootloader into those images, and do not put it into a Muse image.

Do **not** write an SDK token (`mgst_…`), `flash_target`, `flash_address_adapted`, or the bootloader, partition table, OTA, and PHY addresses and files. The app site supplies those. A token stays in the browser session. Watcher fills `flash_target` with `"xiao"` on open; do not write `"himax"`.

Checked Muse flash differences are documentation only. Do not implement them in the CLI:

| Device | Layout | Serial / reset |
|--------|--------|----------------|
| E1001 / E1002 / Indicator | Five segments, no erase-all: `0x0` bootloader, `0x10000` partition table, `0x17000` OTA, `0x19000` PHY, app at `0x20000` | Close the port after the write. The user presses reset. Do not reopen the port immediately |
| Watcher | Four segments, no erase-all and no PHY: `0x0`, `0x10000`, OTA `0x1D000`, app `0x20000`. Do not download `nvsfactory` in the browser. Close the port after the write. Do not reset: an RTS pulse or an on-chip watchdog reset, followed by closing the port, exits macOS Chrome. The page asks the user to replug USB, then send the token | `1a86:55d2` |
| Sticky | Same four segments as Watcher, no erase-all, its own bootloader, no `nvsfactory` backup. Checked baud is 115200 | Checked: `1a86:55d3`. After the write, hard-reset, wait until the port is idle, then send the token at 115200 and close. Do not toggle the port in the same moment as the reset. Unchecked: no USB restriction, one merged image from `0x0`, watchdog reset after the write |
| 1.47" Touch | Same four segments as Watcher, no erase-all, its own bootloader, no `nvsfactory` backup | `303a:1001`. Close the port after the write. The user presses reset |

Unchecked `muse_gadget` keeps the single-file device table below and does not require a token. `full_flash` may still be `true` on a Muse SKU.

| Device | `device_id` (SKU) | `flash_mode` | `flash_address` | File |
|--------|-------------------|--------------|-----------------|------|
| XIAO ESP32C3 | `113991054` | `bin` or `zip` | URL contains `application` → `0x10000`, otherwise `0x0` | `.bin` or zip |
| XIAO ESP32C6 / S3 / S3 Plus | matching SKU (`material.device_id` empty; row comes from `app_block_options`) | same as C3 | same as C3 | same as C3 |
| XIAO ESP32S3 Sense | `113991115` | same as C3 | same as C3 | same as C3 |
| SenseCAP Watcher | `113991315` or `113991275` (one group) | `bin` | same as XIAO, optional | `.bin` on the Xiao / ESP32-S3 port |
| Wio Terminal | `102991299` | `bin` | `0x4000` | `.bin` |
| Grove Beginner Kit | `110061162` | `bin` | `0x0` (address bar hidden; `.hex` uses addresses inside the file) | `.bin` or `.hex`. `.hex` locks the board to this kit |
| XIAO 0.96" / 1.14" IPS, 1.47" Touch (ESP32-S3) | representative SKU of the group; 1.47" Touch Muse SKU is `100069905` | `bin` | `0x0` | merged `.bin`. nRF52840 variants cannot be flashed |
| reTerminal E1001 | `100073581` | `bin` | `0x90000` | application-region `.bin`, not a full image with a bootloader |
| reTerminal E1002 | `100017057` | `bin` | `0x90000` | same as E1001 |
| XIAO e-paper / EE02–EE05 DIY (name contains Display, and is not IPS / Round / Touch / Sense) | matching SKU | `bin` | `0x90000` | same as E1001 |
| reTerminal Sticky | `100056818` | `bin` | `0x0` | merged `.bin` |
| SenseCAP Indicator D1 / D1S / D1L / D1Pro | one group; example `114993068` (D1) | `bin` | `0x0` | merged ESP32-S3 `.bin`, not the RP2040 port |

`.hex` is only for Grove Beginner (`110061162`). E100x, e-paper, Sticky, IPS, Touch, and Indicator accept `.bin` only. Do not use zip for Grove, Wio, those screens, or Indicator. XIAO C3 / C6 / S3 / S3 Sense / S3 Plus / Watcher keep 115200 and `flashSize` / `flashMode` / `flashFreq` = `keep`; do not apply the screen `0x90000` or merged-`0x0` rules to them.

**Agent:** write the SKU, file, and address into the app doc. **Do not** claim the firmware was flashed. Serial flashing stays L3 (Console / Web Serial).

### `deploy-ai-model` (vision / audio / related)

| Path | Meaning |
|------|---------|
| `config.model` | `null` or `{ type, model }` |
| `config.model.type` | `"model"` \| `"upload"` \| `"defer"` |
| `config.model.model` | Metadata: `id`, `name`, `task`, `labels`/`classes`, `uniform_types`, `isCustom`, … |
| `config.deploy_tool_entry` | `"vision"` or `"sound"`. Both toolbar entries use this block type. `sound` excludes reCamera Pro (`uniform_type` `53`) |
| `config.device_type` | Device `uniform_type` string (e.g. `"32"`) |
| `config.powerMode` | 0–5, default `2` (XIAO power-on infer) |
| `config.conf` / `config.iou` | Detection UI thresholds |
| `config.threshold` | Audio classify 0–1 when task/device needs it |
| `config.triggerAction` | Optional GPIO/LED/image trigger object |

**Scenes:** vision classify/detect → set `task` accordingly; sound → audio task + threshold; vibration-only product → prefer `ai-vibration`.

### `ai-vibration`

| Path | Default / notes |
|------|-----------------|
| `config.detectionType` | e.g. Abnormal Vibration Detection |
| `config.deviceType` | e.g. XIAO ESP32S3 Sense |
| `config.windowSize` | 192–768, default 384 |
| `config.isPower` | default false |
| `config.gpio` | `"-1"` = off |
| `config.dStatus` | default `"1"` |

### `deploy-container-app`

Used for **container / edge app** deployment UI inside the application doc. Fields live in `config`, same names as the editor's defaults:

| Path | Notes |
|------|-------|
| `config.app_id` / `config.app_key` | Fleet app id and key |
| `config.app_display_name` / `config.app_icon` / `config.app_description` | Card text and icon |
| `config.architecture` | Selected architecture string |
| `config.architectures` | Architecture list |
| `config.version` | Selected version string. Not `app_version` |
| `config.version_options` | Version list |
| `config.install_form` | Install form object |
| `config.merged_service` | Merged service snapshot |
| `config.user_override_flags` | Which form fields the user overrode |
| `config.form_field_values` | Form values |
| `config.raw_by_version` | Raw detail by version |
| `config.detail_form_fields` | Form field schema |

**Agent:** configure the block for documentation. Installing the container is not a CLI command. Prefer linking users to Console preview deploy when the CLI has no container control.

## Other tool-blocks

### `node-red`

| Path | Notes |
|------|-------|
| `config.flow_data_url` | Flow file URL |
| `config.flow_data` | Parsed flow object **or array**, not a JSON string; also sync top-level `flow_data` as **JSON string** |
| `config.model_id` | Cascade pair; `['3', id]` = my model |
| `config.model_data` | Snapshot; sync top-level `model_data` |

### `aily-blockly`

| Path | Notes |
|------|-------|
| `config.project_id` | Public project id |
| `config.device_id` | Deploy device SKU |
| `config.config` | Extension placeholder; avoid nested `config.config` objects |

## Minimal step + tool example

```json
[
  {
    "id": "step_1",
    "type": "step",
    "title": "Flash firmware",
    "content": "Flash firmware",
    "expanded": true,
    "indent": 0
  },
  {
    "id": "para_1",
    "type": "paragraph",
    "content": "Connect the board over USB, then flash.",
    "indent": 0
  },
  {
    "id": "ff_1",
    "type": "flash-firmware",
    "config": {
      "device_id": "100017057",
      "device_type": "",
      "flash_mode": "bin",
      "firmware_url": "https://example.com/e1002-app.bin",
      "firmware_name": "e1002-app.bin",
      "firmware_size": "1234567",
      "flash_address": "0x90000",
      "muse_gadget": false,
      "full_flash": false
    }
  }
]
```

## Repair an existing malformed block

Read the latest owner sketch, clone its **complete** `deployment`, and move the affected block's settings into `config`, then remove those fields from the block. Preserve its ID, all values, order, and every other block. If flat and nested values disagree, keep `config`. An update containing only a one-element `deployment.blocks` array replaces the document; it is not a patch of that one block.

Validate the corrected document and follow the parent's existing-app write protocol. Re-read the sketch afterward and check the saved `config` and unchanged surrounding blocks. Reconcile the project's local manifest and sync baseline too, so a later push does not restore the malformed shape. A successful API response only proves storage; verify the consumer-visible field paths as well.
