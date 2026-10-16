# Deploy docs & tool-block reference

Help the Agent choose the right block and fill `config` safely. Execution on hardware is **L3** unless noted.

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

Prefer `paragraph` under a `step` for instructions; do not overload tool-block `config` with prose.

## Tool-block selection

| Priority | `type` | Use when | Avoid when |
|----------|--------|----------|------------|
| High | `flash-firmware` | User must burn a firmware image to a board | They only need model deploy |
| High | `deploy-ai-model` | Vision classify/detect or audio model on SenseCraft devices | Pure vibration → `ai-vibration`; pure firmware → `flash-firmware` |
| High | `ai-vibration` | Abnormal vibration detection flow | General vision/audio models |
| High | `deploy-container-app` | Edge **container** app deploy UX in the doc | Confusing with App Plaza **clone** or SenseCAP bind |
| Lower | `node-red` | Node-RED flow + optional model binding | No flow artifact |
| Lower | `aily-blockly` | Aily Blockly public project | No project id |

Multiple tool-blocks may appear in one step (e.g. paragraph → flash → paragraph → deploy-ai-model).

## High-priority configs

### `flash-firmware`

| Path | Meaning |
|------|---------|
| `config.device_id` | Device SKU |
| `config.device_type` | Compat field (may be `""`) |
| `config.firmware_url` | Firmware URL |
| `config.firmware_size` | Size string |

**Agent:** write URLs/SKU into the app doc. **Do not** claim the firmware was flashed until the user completes Console/Web Serial (L3).

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

**Agent:** configure the block for documentation. Installing the container is **not** `app clone`. Prefer linking users to Console preview deploy when the CLI has no container control.

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
    "title": "Step 1: Flash firmware",
    "content": "Step 1: Flash firmware",
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
    "config": {
      "device_id": "",
      "device_type": "",
      "firmware_url": "https://example.com/fw.bin",
      "firmware_size": ""
    }
  }
]
```
