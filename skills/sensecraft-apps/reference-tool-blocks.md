# Deploy docs & tool-block reference

Help the Agent choose the right block and fill `config` safely. Execution on hardware is **L3** unless noted.

## Content format

Text-block `content` (introduction and step text alike) renders **Markdown** — inline HTML like `<p>` also renders and appears in existing docs. Images and externally hosted media are available through ordinary Markdown:

```markdown
![wiring diagram](https://files.example.com/wiring.png)
[Watch the demo video](https://example.com/demo.mp4)
```

Host binaries (images, videos) via `file upload` first; Markdown only links out, it does not embed uploads by itself.

## Document layout

`deployment.blocks` is a **flat** ordered array.

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

Write these as **flat fields** on the block. The editor wraps them into `config` when it reloads the doc. Do not nest them under `config` in `app create` / `app update`.

| Field | How to write |
|-------|----------------|
| `device_id` | Material SKU string, e.g. `"100017057"`. Not a production type id (`"29"`) and not the material row `id` |
| `device_type` | `uniform_type` string. Use `""` when the board has none (Wio / Grove may) |
| `flash_mode` | `"bin"` when a firmware URL is set. `"zip"` for a project zip |
| `firmware_url` / `firmware_name` / `firmware_size` | Bin mode. `firmware_size` is a string, not a number. Empty `firmware_name` is filled from the URL by the editor |
| `flash_address` | Only `"0x0"`, `"0x4000"`, `"0x10000"`, `"0x90000"`. Use the device default below. The editor resets a non-default address the first time the doc is opened. Do not write `0x20000` here |
| `project_zip_url` / `project_zip_name` / `project_zip_size` / `precheck_fields` | Zip mode only. Sizes are strings. `precheck_fields` is `[{ key, name, info, type, default }]` from `sensecraft_config.h`, or omit / `[]` |
| `muse_gadget` | Boolean, default `false`. A missing field is `false`. `"true"` and `1` are read as `true`. `true` shows the Muse SDK token input under the flash button in preview |

`muse_gadget: true` is kept only for these SKUs. Any other device written `true` is reset to `false` by the editor:

| Device | `device_id` (SKU) |
|--------|-------------------|
| SenseCAP Indicator D1 / D1S / D1L / D1Pro | `114993068` (aliases on the same row are kept) |
| SenseCAP Watcher | `113991315` or `113991275` |
| reTerminal E1001 | `100073581` |
| reTerminal E1002 | `100017057` |
| XIAO 1.47" Touch Display | `100069905` |
| reTerminal Sticky | `100056818` |

When `muse_gadget` is `true`, `firmware_url` must be `muse-gadget.bin` linked at `0x20000`. A binary linked at `0x90000` or `0x10000` will not boot. Checking the box does not change `flash_address`: E100x stays `0x90000`, Sticky / Indicator / 1.47" Touch stay `0x0`, Watcher stays on the XIAO address rule.

Do **not** write an SDK token (`mgst_…`), `flash_target`, `flash_address_adapted`, or the bootloader, partition table, OTA, and PHY addresses and files. The app site supplies those. A token stays in the browser session. Watcher fills `flash_target` with `"xiao"` on open; do not write `"himax"`.

Checked Muse flash differences are documentation only. Do not implement them in the CLI:

| Device | Layout | Serial / reset |
|--------|--------|----------------|
| E1001 / E1002 / Indicator | Five segments, no full-chip erase: `0x0` bootloader, `0x10000` partition table, `0x17000` OTA, `0x19000` PHY, app at `0x20000` | E1001 / E1002 do not auto-reset after the write |
| Watcher | Four segments, no PHY, no full-chip erase: `0x0`, `0x10000`, OTA `0x1D000`, app `0x20000`. Back up `nvsfactory` from `0x9000` for length `0x32000` first | `1a86:55d2` |
| Sticky | Same four segments as Watcher, its own bootloader, no `nvsfactory` backup | Checked: `1a86:55d3`. Unchecked: no USB restriction, one merged image from `0x0` |
| 1.47" Touch | Same four segments as Watcher, its own bootloader, no `nvsfactory` backup | `303a:1001`. Does not auto-reset after the write |

Unchecked `muse_gadget` keeps the single-file device table below and does not require a token.

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

Used for **container / edge app** deployment UI inside the application doc.

**Agent:** configure the block for documentation. Installing the container is not a CLI command. Prefer linking users to Console preview deploy when the CLI has no container control.

## Other tool-blocks

### `node-red`

| Path | Notes |
|------|-------|
| `config.flow_data_url` | Flow file URL |
| `config.flow_data` | Parsed flow; also sync top-level `flow_data` as **JSON string** |
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
    "content": "<p>Connect the board over USB, then flash.</p>",
    "indent": 0
  },
  {
    "id": "ff_1",
    "type": "flash-firmware",
    "device_id": "100017057",
    "device_type": "",
    "flash_mode": "bin",
    "firmware_url": "https://example.com/e1002-app.bin",
    "firmware_name": "e1002-app.bin",
    "firmware_size": "1234567",
    "flash_address": "0x90000",
    "muse_gadget": false
  }
]
```
