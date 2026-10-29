# SenseCraft CLI — Command Reference

Complete reference for AI agents. All paths are relative to the OpenAPI base URL (`https://sensecap.seeed.cc/openapi` or `https://sensecap.seeed.cn/openapi`).

---

## auth init

OAuth Device Flow. Obtains an Access Key and writes `sensecraft.json`.

**Not an OpenAPI call.** Bare `auth init` is a TTY wizard. The split flags below are for a one-shot shell: each command prints JSON and exits. `--sandbox` simulates the wizard, or the one split step it is combined with, without network access.

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--sandbox` | No | `false` | Simulate this step without sending real requests |
| `--site` | With `--no-wait` | — | `Global` or `China`. China is SenseCraft Data only |
| `--no-wait` | No | `false` | Request a device code and return immediately |
| `--device-code` | No | — | Continue a previous `--no-wait` login |
| `--key-id` | No | — | Save this existing Access Key id after authorization |
| `--label` | No | — | Create an Access Key with this label after authorization |
| `--access-level` | No | `read-write` | With `--label`: `read-write` or `only-read` |
| `--json` | No | `false` | Accepted for agents. The split flow always prints JSON |

### Wizard

With no split flags, in a terminal:

1. Select site: Global (International) or China. China supports SenseCraft Data only. SenseCraft AI and CodeCraft Hub require Global.
2. Request a device authorization code and show the verification URL/code.
3. Wait for the user to approve in the browser; the CLI polls automatically.
4. Resolve account/organization, then select an existing Access Key or create one with a label and permission level.
5. Save to local config and set as active profile.

### Split flow

```bash
sensecraft-cli auth init --site Global --no-wait
sensecraft-cli auth init --device-code <device_code>
sensecraft-cli auth init --device-code <device_code> --key-id <access_id>
sensecraft-cli auth init --device-code <device_code> --label <name> --access-level read-write
```

1. `--no-wait` prints `verification_url`, `user_code`, `device_code`, `expires_in`, and `site`, then exits. It does not read stdin.
2. Show `verification_url` to the user and end the turn. Do not poll in that same turn.
3. After the user authorizes, `--device-code` polls, then prints the account and Access Key list (`label`, `access_id`, `access_level`).
4. A later command with `--key-id` or `--label` saves the key and sets it active.

The login access token is written to `sensecraft-auth-pending.json` next to the config file (mode `0600`) and is removed after the key is saved. It is not printed. Key secrets are not printed.

### Agent notes

- Do **not** run bare `auth init` in a non-interactive shell.
- Run the split commands above. Do not echo `access_token` or `access_key`.
- In sandbox mode, simulated credentials are written locally for testing. Use an isolated test config; sandbox credentials do not authorize real cloud calls.

---

## auth add

Manually add an existing Access Key to local config. Verifies credentials via OpenAPI `GET /view_organization`, saves to `sensecraft.json`, and sets the profile as active.

Site defaults to Global when omitted. In an interactive terminal, missing `--key-id` / `--access-key` will be prompted.

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--key-id` | Yes* | — | Access Key ID (API ID) |
| `--access-key` | Yes* | — | Access Key secret |
| `--site` | No | `Global` | Site: `Global` or `China`. China is SenseCraft Data only |

\* Required unless running interactively (then prompted).

### Steps

1. Collect Key ID, Access Key, and optional site
2. Call `GET /view_organization` with HTTP Basic Auth to verify and obtain `org_id`
3. Save profile to local config and set as active

### Output

Human-readable progress on stdout/stderr. No JSON payload.

### Agent notes

- Prefer this over `auth init` when the user already has Key ID / Access Key (e.g. from SenseCAP Portal).
- Do **not** echo `--access-key` in logs; treat it as a secret.
- Still requires network access to the selected site's OpenAPI host.

---

## auth info

Display local authentication configuration. Does not call the cloud API.

### Flags

None.

### Output (text, not JSON)

```
Config file: /path/to/sensecraft.json
Active Key ID: <key-id>

[active]
  Site: Global (International)
  Organization ID: <org-id>
  Key ID: <key-id>
  Created at: <RFC3339 timestamp>
```

When uninitialized:

```
Config file: /path/to/sensecraft.json

No Access Key configured. Run: sensecraft-cli auth init --site Global --no-wait. Show verification_url to the user unchanged and end this turn. After they authorize, run the --device-code command from that command's hint.
```

---

## device list

List devices under the current account.

**OpenAPI:** `GET /list_devices`

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--device-type` | No | `2` | Device type: `1`=gateway, `2`=node, `3`=development kit |
| `--group-uuid` | No | — | Filter by group UUID; omit to return devices from all groups |

### Query parameters sent to API

| Parameter | Source |
|-----------|--------|
| `device_type` | `--device-type` |
| `group_uuid` | `--group-uuid` (omitted if empty) |

### Examples

```bash
sensecraft-cli device list
sensecraft-cli device list --device-type 1
sensecraft-cli device list --device-type 3
sensecraft-cli device list --group-uuid 80523B280630E611
```

### Response schema

JSON **array** of objects:

| Field | Type | Description |
|-------|------|-------------|
| `device_eui` | string | Device unique identifier (EUI) |
| `device_name` | string | Device display name |

### Example output

```json
[
  {
    "device_eui": "2CF7F12010700088",
    "device_name": "device2CF7F12010700088"
  }
]
```

---

## device view

Get detailed information for one or more devices.

**OpenAPI:** `POST /view_devices`

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--eui` | **Yes** | — | Device EUI; repeat flag for multiple devices (max **50** per request) |
| `--device-type` | No | `2` | Device type: `1`=gateway, `2`=node, `3`=development kit |

### Request body sent to API

```json
{
  "device_euis": ["<eui1>", "<eui2>"],
  "device_type": 2
}
```

### Examples

```bash
sensecraft-cli device view --eui 2CF7F15000100122
sensecraft-cli device view --eui EUI1 --eui EUI2 --device-type 1
```

### Response schema

JSON **array** of device detail objects (per [SenseCAP Device Management API](https://sensecap-docs.seeed.cc/device_management.html)):

| Field | Type | Description |
|-------|------|-------------|
| `frequency` | string | Communication frequency |
| `device_eui` | string | Device EUI |
| `device_name` | string | Device name |
| `device_network` | number | Network type: `1`=LoRaWAN, `2`=NB-IoT, `3`=2G, `4`=LoRaPP |
| `position` | object | GPS location (`latitude`, `longitude`) |
| `position_source` | number | `0`=manually set, `1`=reported by device |
| `hardware_version` | string | Hardware version |
| `software_version` | string | Software/firmware version |
| `sim` | object | SIM info (e.g. `iccid`) |

Additional fields may be present depending on device type.

---

## device channels

List measurement channels for node devices.

**OpenAPI:** `POST /list_device_channels`

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--eui` | **Yes** | — | Node device EUI; repeat for multiple (max **50**) |

### Request body

```json
{ "device_euis": ["<eui1>", "<eui2>"] }
```

### Examples

```bash
sensecraft-cli device channels --eui 2CF7F16221200060
```

### Response schema

JSON **array**; each element:

| Field | Type | Description |
|-------|------|-------------|
| `device_eui` | string | Device EUI |
| `channels` | array | Channel list for this device |

Each channel object:

| Field | Type | Description |
|-------|------|-------------|
| `channel_index` | number | Channel number (use as `--channel-index` in data commands) |
| `sensor_id` | string | Sensor identifier |
| `sensor_status` | number | `0`=idle, `1`=normal, `2`=abnormal |
| `channel_type` | number | `1`=485 Sensor, `2`=Seeed Sensor, `3`=485 Output, `4`=Seeed Output |
| `sensor_type` | string | Sensor type ID |
| `channel_name` | string | Channel name |
| `measurement_ids` | array of string | Measurement IDs (use as `--measurement-id` in data commands) |

### Agent tip

Run this before `data latest`, `data list`, or `data chart` to discover valid `measurement_ids` and channel numbers.

---

## device status

Get running status for devices.

**OpenAPI:** `POST /view_device_running_status`

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--eui` | **Yes** | — | Device EUI; repeat for multiple (max **50**) |

### Request body

```json
{ "device_euis": ["<eui1>"] }
```

### Examples

```bash
sensecraft-cli device status --eui 2CF7F16221200060
sensecraft-cli device status --eui EUI1 --eui EUI2
```

### Response schema

JSON **array** of status objects:

| Field | Type | Description |
|-------|------|-------------|
| `device_eui` | string | Device EUI |
| `latest_message_time` | string | Last message time (ISO 8601); empty if none |
| `online_status` | number | `0`=offline, `1`=online |
| `battery_status` | number | `0`=low battery, `1`=good battery |
| `battery_digit` | number | Battery level percentage |
| `report_frequency` | number | Report interval in minutes; `-1` if unavailable |

---

## device bind

Bind a device to the current account.

**OpenAPI:** `POST /bind_device`

**Write operation** — confirm EUI and binding code with the user.

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--eui` | **Yes** | — | Device EUI (printed on device label) |
| `--code` | **Yes** | — | Device binding code (printed on device label) |
| `--name` | No | — | Custom device display name |
| `--group-uuid` | No | — | Target group UUID (from `group list`) |
| `--longitude` | No | — | Longitude string |
| `--latitude` | No | — | Latitude string |

### Request body sent to API

| JSON field | CLI flag |
|------------|----------|
| `eui` | `--eui` |
| `code` | `--code` |
| `device_name` | `--name` (omitted if empty) |
| `group_uuid` | `--group-uuid` (omitted if empty) |
| `longitude` | `--longitude` (omitted if empty) |
| `latitude` | `--latitude` (omitted if empty) |

### Examples

```bash
sensecraft-cli device bind --eui 2CF7F15000100122 --code ABCD1234
sensecraft-cli device bind --eui 2CF7F15000100122 --code ABCD1234 \
  --name "Greenhouse Node 1" --group-uuid 80523B280630E611 \
  --longitude 113.926979 --latitude 22.573724
```

### Response schema

Empty object on success:

```json
{}
```

---

## device unbind

Remove devices from the current account (devices can be re-bound via the SenseCAP App).

**OpenAPI:** `POST /delete_devices`

**Write operation** — L2: separate chat confirm, then `--yes`.

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--eui` | **Yes** | — | Device EUI; repeat for multiple (max **50**) |
| `--yes` | **Yes** | `false` | Confirm L2 unbind |

### Request body

```json
{ "device_euis": ["<eui1>", "<eui2>"] }
```

### Examples

```bash
sensecraft-cli device unbind --eui 2CF7F15000100122 --yes
sensecraft-cli device unbind --eui EUI1 --eui EUI2 --yes
```

### Response schema

```json
{}
```

---

## device sim

Query SIM card status by ICCID.

**OpenAPI:** `GET /view_simcard?iccid=<iccid>`

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--iccid` | **Yes** | — | SIM card ICCID |

### Examples

```bash
sensecraft-cli device sim --iccid 89860446091891237424
```

### Response schema

| Field | Type | Description |
|-------|------|-------------|
| `iccid` | string | ICCID |
| `msisdn` | string | MSISDN |
| `activateTime` | string | Activation date (timestamp ms) |
| `expiryDate` | string | Billing end date (timestamp ms) |
| `status` | string | `0`=unknown, `1`=normal, `2`=single stop, `3`=stop, `4`=pre-sale, `5`=sale, `6`=transfer, `7`=sleep, `8`=pending activation |
| `flow` | string | Current month data usage |
| `residueFlow` | string | Remaining data allowance |

---

## device devkit create

Create a development kit virtual device.

**OpenAPI:** `POST /device/create_development_kit`

**Write operation.** Max **5** kits per SKU per organization.

> **Schema note:** This endpoint is not in the public SenseCAP API docs. The CLI only types `eui` and `device_key`. If the live API returns additional fields, they may be omitted from CLI output. Ask the user for a sample response if more fields are needed.

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--sku` | **Yes** | — | Dev kit SKU (see SKU table in SKILL.md) |
| `--name` | No | — | Device display name |

### Request body sent to API

```json
{
  "sku": "xiao",
  "dev_name": "my-xiao"
}
```

(`dev_name` omitted when `--name` is not set.)

### Examples

```bash
sensecraft-cli device devkit create --sku blank_device
sensecraft-cli device devkit create --sku xiao --name "lab-xiao"
sensecraft-cli device devkit create --sku grove_vision_ai --name "vision-lab"
```

### Response schema (confirmed by CLI code)

| Field | Type | Description |
|-------|------|-------------|
| `eui` | string | 16-character device EUI assigned to the kit |
| `device_key` | string | Device access key for onboarding |

### Example output

```json
{
  "eui": "0123456789ABCDEF",
  "device_key": "your-device-key"
}
```

### Follow-up

Use `device devkit key --eui <eui>` to retrieve `token` for MQTT/network configuration.

---

## device devkit key

Retrieve development kit credentials by EUI.

**OpenAPI:** `GET /device/view_node_key?node_eui=<eui>`

> **Schema note:** Not in public SenseCAP API docs. CLI validates EUI length (16 chars) and types three response fields.

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--eui` | **Yes** | — | 16-character dev kit EUI (auto uppercased and trimmed) |

### Examples

```bash
sensecraft-cli device devkit key --eui 0123456789ABCDEF
```

### Response schema (confirmed by CLI code)

| Field | Type | Description |
|-------|------|-------------|
| `eui` | string | Device EUI |
| `device_key` | string | Device access key |
| `token` | string | Authentication token (e.g. for MQTT connection) |

### Example output

```json
{
  "eui": "0123456789ABCDEF",
  "device_key": "your-device-key",
  "token": "your-token"
}
```

### Agent notes

- EUI must be exactly **16 characters**; otherwise CLI returns a validation error before calling the API.
- Only supported for dev kit devices (Blank Device / XIAO / Grove Vision AI V2).
- **Treat all credential fields as secrets.**

---

## group list

List all device groups.

**OpenAPI:** `GET /list_groups`

### Flags

None.

### Example

```bash
sensecraft-cli group list
```

### Response schema

JSON **array** of group objects:

| Field | Type | Description |
|-------|------|-------------|
| `group_name` | string | Group display name |
| `group_uuid` | string | Group UUID; empty string for the default group |

### Example output

```json
[
  { "group_name": "Default", "group_uuid": "" },
  { "group_name": "test group", "group_uuid": "80523B280630E611" }
]
```

---

## group create

Create a new device group.

**OpenAPI:** `POST /create_group`

**Write operation.** Max **50** groups per account.

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--name` | **Yes** | — | Group name |

### Request body

```json
{ "name": "Greenhouse A" }
```

### Example

```bash
sensecraft-cli group create --name "Greenhouse A"
```

### Response schema

| Field | Type | Description |
|-------|------|-------------|
| `group_name` | string | Created group name |
| `group_uuid` | string | New group UUID |

---

## group update

Rename a device group.

**OpenAPI:** `POST /update_group`

**Write operation.**

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--group-uuid` | **Yes** | — | Group UUID to update |
| `--name` | **Yes** | — | New group name |

### Request body

```json
{ "group_uuid": "<uuid>", "name": "New Name" }
```

### Example

```bash
sensecraft-cli group update --group-uuid 80523B280630E611 --name "Renamed Group"
```

### Response schema

```json
{}
```

---

## group delete

Delete a device group. Devices in the group move to the default group.

**OpenAPI:** `POST /delete_group?group_uuid=<uuid>`

**Write operation** — L2: separate chat confirm, then `--yes`. The default group cannot be deleted.

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--group-uuid` | **Yes** | — | Group UUID to delete |
| `--yes` | **Yes** | `false` | Confirm L2 delete |

### Example

```bash
sensecraft-cli group delete --group-uuid 80523B280630E611 --yes
```

### Response schema

```json
{}
```

---

## group move-devices

Move devices to a target group.

**OpenAPI:** `POST /move_devices_to_group`

**Write operation** — L2: separate chat confirm, then `--yes`.

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--group-uuid` | **Yes** | — | Target group UUID |
| `--eui` | **Yes** | — | Device EUI; repeat for multiple |
| `--yes` | **Yes** | `false` | Confirm L2 move |

### Request body

```json
{
  "group_uuid": "<target-uuid>",
  "devices": ["<eui1>", "<eui2>"]
}
```

### Example

```bash
sensecraft-cli group move-devices \
  --group-uuid 80523B280630E611 \
  --eui 2CF7F12004100001 --eui 2CF7F12004100002 --yes
```

### Response schema

```json
{}
```

---

## data latest

Get the latest telemetry data for a device within the past year.
If `channel_index` is omitted, returns the latest point under each channel.

**OpenAPI:** `GET /view_latest_telemetry_data`

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--eui` | **Yes** | — | Device EUI |
| `--measurement-id` | No | — | Measurement ID (from `device channels`) |
| `--channel-index` | No | — | Channel index (from `device channels`) |

### Query parameters sent to API

| Parameter | Source |
|-----------|--------|
| `device_eui` | `--eui` |
| `measurement_id` | `--measurement-id` (omitted if empty) |
| `channel_index` | `--channel-index` (omitted if empty) |

### Examples

```bash
sensecraft-cli data latest --eui 2CF7F12010700001
sensecraft-cli data latest --eui 2CF7F12010700001 --measurement-id 4104 --channel-index 32
```

### Response schema

| Field | Description |
|-------|-------------|
| `channel_index` | Channel number |
| `points` | Array of `{ measurement_value, measurement_id, time }` |

---

## data list

Get historical telemetry data for a device.

**OpenAPI:** `GET /list_telemetry_data`

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--eui` | **Yes** | — | Device EUI |
| `--measurement-id` | No | — | Measurement ID |
| `--channel-index` | No | — | Channel index |
| `--limit` | No | `0` | Number of records to return (`0` means API default) |
| `--time-start` | No | `0` | Start timestamp in **milliseconds** |
| `--time-end` | No | `0` | End timestamp in **milliseconds** |

### Limits

- About **1 month** of data per request
- Only data within the last **three months** can be queried

### Query parameters sent to API

| Parameter | Source |
|-----------|--------|
| `device_eui` | `--eui` |
| `measurement_id` | `--measurement-id` |
| `channel_index` | `--channel-index` |
| `limit` | `--limit` (only if > 0) |
| `time_start` | `--time-start` (only if > 0) |
| `time_end` | `--time-end` (only if > 0) |

### Examples

```bash
sensecraft-cli data list --eui 2CF7F12210400082 --limit 100
sensecraft-cli data list --eui 2CF7F12210400082 \
  --measurement-id 4097 --channel-index 1 \
  --time-start 1700000000000 --time-end 1700086400000 \
  --limit 400
```

### Response schema

Columnar `list` structure: `list[0]` is `[channel_index, measurement_id]` pairs; `list[1]` is corresponding measurement series.

---

## data chart

Get aggregated telemetry chart points (interval averages).

**OpenAPI:** `GET /aggregate_chart_points`

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--eui` | **Yes** | — | Device EUI |
| `--measurement-id` | No | — | Measurement ID |
| `--channel-index` | No | — | Channel index |
| `--interval` | No | `60` | Interval length in **minutes** |
| `--time-start` | No | `0` | Start timestamp in **milliseconds** |
| `--time-end` | No | `0` | End timestamp in **milliseconds** |

### Limits

- Time range: up to **1 year**
- Max **250** points per measurement (platform auto-rebins if exceeded)

### Query parameters sent to API

| Parameter | Source |
|-----------|--------|
| `device_eui` | `--eui` |
| `measurement_id` | `--measurement-id` |
| `channel_index` | `--channel-index` |
| `interval` | `--interval` |
| `time_start` | `--time-start` (only if > 0) |
| `time_end` | `--time-end` (only if > 0) |

### Examples

```bash
sensecraft-cli data chart --eui 2CF7F12210400082
sensecraft-cli data chart --eui 2CF7F12210400082 \
  --interval 60 --measurement-id 4106 --channel-index 1 \
  --time-start 1700000000000 --time-end 1735689600000
```

### Response schema

| Field | Description |
|-------|-------------|
| `channel` | Channel number |
| `lists` | Array of `{ average_value, measurement_id, time }` |

---

## Common measurement IDs

Use with `--measurement-id`. Full list: SenseCAP appendix or `https://sensecap-statics.seeed.cn/refer/def/sensor.json`.

| ID | Measurement | Unit |
|----|-------------|------|
| `4097` | Air Temperature | ℃ |
| `4098` | Air Humidity | %RH |
| `4099` | Light Intensity | lux |
| `4100` | Barometric Pressure | hPa |
| `4101` | CO2 | ppm |
| `4104` | Soil Moisture | % |

Discover device-specific IDs via `device channels`.

---

## Error handling

| Scenario | Behavior |
|----------|----------|
| Missing config / Access Key | Error before API call; message mentions `sensecraft.json` |
| Missing required flag | Cobra validation error on stderr |
| API business error | `API error code=<code>: <msg>` |
| Invalid devkit SKU | Local validation error listing allowed SKUs |
| Invalid devkit EUI length | Local validation: must be 16 characters |
| Network failure | HTTP error on stderr |

SenseCAP error code list: https://sensecap-docs.seeed.cc/
