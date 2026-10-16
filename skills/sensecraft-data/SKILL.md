---
name: sensecraft-data
description: >-
  Operates SenseCraft CLI against SenseCAP PaaS HTTP OpenAPI to manage devices,
  groups, development kits, and sensor telemetry data. Use when the user asks to
  list/bind/unbind SenseCAP devices, manage device groups, query telemetry
  (latest/list/chart), provision dev kits, or automate SenseCAP cloud operations
  via CLI or npx. Not for SenseCraft AI apps/models/cloud.
---

# SenseCraft Data (SenseCAP PaaS)

SenseCraft CLI wrappers for the [SenseCAP HTTP OpenAPI](https://sensecap-docs.seeed.cc/). Every `device`, `group`, and `data` command sends authenticated HTTP requests to the user's configured site (Global / China). The CLI unwraps the API envelope (`code`, `msg`, `data`) and prints only the `data` payload as indented JSON on stdout.

Parent router: [`../sensecraft-ai/SKILL.md`](../sensecraft-ai/SKILL.md).

## Installation

```bash
npx @seeed-studio/sensecraft-cli install
```

This downloads the platform binary, installs `sensecraft-cli` globally, and may run `auth init` in an interactive terminal.

Run without global install:

```bash
npx @seeed-studio/sensecraft-cli device list
```

Requires Node.js >= 16.

## Agent rules

1. **Always check auth first** before `device`, `group`, or `data` commands:
   ```bash
   sensecraft-cli auth info
   ```
2. **Never run `auth init` on behalf of the user** — it requires browser OAuth in an interactive terminal.
   - SenseCAP Access Key: `sensecraft-cli auth init --kind sensecap`
   - If the user already has Key ID / Access Key, guide them to run `auth add` locally instead (do not pass secrets into the agent context).
3. **L2 confirm is a separate chat turn** ([`../confirmation.md`](../confirmation.md)) before write ops: `device bind` / `unbind`, `group create` / rename / `delete` / `move-devices`, `device devkit create`. 「解绑这台设备」is **not** confirmation — ask (操作类型 + 环境 + EUI/组), wait for `确认`, then `--yes` where the command has it (`unbind` / `group delete` / `group move-devices`). Never pass `--yes` on the same turn as the first request.
4. **Treat dev kit credentials as secrets** — `device_key` and `token` must not be logged or echoed unnecessarily.
5. **Parse stdout JSON** on success; on failure read stderr and check exit code != 0.
6. **LoRaWAN devices** must use the Global (international) site per SenseCAP platform rules.

## Initialization check

```bash
sensecraft-cli auth info
```

| State | Indicator | Action |
|-------|-----------|--------|
| Initialized (SenseCAP) | Output shows SenseCAP `[active]`, Key ID, Site, Organization ID | Proceed with `device` / `group` / `data` |
| Not initialized | Prompts to run `auth init --kind sensecap` | Ask the user to run `auth init --kind sensecap` locally; for existing Access Key use `auth add` |

## Configuration

| Item | Env var | Default |
|------|---------|---------|
| Config directory | `SENSECRAFT_CLI_PATH` | Current working directory |
| Config file | `SENSECRAFT_CONFIG_FILE` | `sensecraft.json` (in config dir or as absolute path) |

Resolved path priority: `SENSECRAFT_CONFIG_FILE` (absolute or relative to `SENSECRAFT_CLI_PATH`) → `{SENSECRAFT_CLI_PATH}/sensecraft.json` → `{cwd}/sensecraft.json`.

## API sites

| Site | OpenAPI base URL |
|------|------------------|
| Global (International) | `https://sensecap.seeed.cc/openapi` |
| China | `https://sensecap.seeed.cn/openapi` |

Authentication: HTTP Basic Auth with Key ID (username) and Access Key (password), obtained via `auth init` (OAuth Device Flow) or `auth add` (manual import).

## Output and errors

- **Success**: indented JSON (2 spaces) on stdout.
- **Failure**: human-readable error on stderr; non-zero exit code.
- **API errors**: `API error code=<code>: <msg>` when the platform returns `code != "0"`.

## Command index

| Group | Command | OpenAPI method & path | Write? |
|-------|---------|----------------------|--------|
| auth | `init` | OAuth (not OpenAPI); `--kind sensecap\|ai` | Yes |
| auth | `add` | GET `/view_organization` (verify) + local config | Yes |
| auth | `info` | Local config only (SenseCAP + AI sections) | No |
| auth | `refresh` | authapi refreshToken (AI profile only) | Yes |
| device | `list` | GET `/list_devices` | No |
| device | `view` | POST `/view_devices` | No |
| device | `channels` | POST `/list_device_channels` | No |
| device | `status` | POST `/view_device_running_status` | No |
| device | `bind` | POST `/bind_device` | Yes |
| device | `unbind` | POST `/delete_devices` | Yes |
| device | `sim` | GET `/view_simcard` | No |
| device | `devkit create` | POST `/device/create_development_kit` | Yes |
| device | `devkit key` | GET `/device/view_node_key` | No |
| group | `list` | GET `/list_groups` | No |
| group | `create` | POST `/create_group` | Yes |
| group | `update` | POST `/update_group` | Yes |
| group | `delete` | POST `/delete_group` | Yes |
| group | `move-devices` | POST `/move_devices_to_group` | Yes |
| data | `latest` | GET `/view_latest_telemetry_data` | No |
| data | `list` | GET `/list_telemetry_data` | No |
| data | `chart` | GET `/aggregate_chart_points` | No |

## Typical workflows

### Discover devices and read sensor data

```bash
sensecraft-cli auth info
sensecraft-cli group list
sensecraft-cli device list
sensecraft-cli device channels --eui <node-eui>
sensecraft-cli data latest --eui <node-eui> --measurement-id 4097 --channel-index 1
```

### Bind a new physical device

```bash
sensecraft-cli group list          # optional: get group_uuid
sensecraft-cli device bind --eui <eui> --code <code> --name "My Sensor" --group-uuid <uuid>
```

### Provision a development kit

```bash
sensecraft-cli device devkit create --sku xiao --name "lab-xiao"
sensecraft-cli device devkit key --eui <16-char-eui>
sensecraft-cli device list --device-type 3
```

## Device type reference

Used by `device list` and `device view`:

| Value | Type |
|-------|------|
| `1` | Gateway |
| `2` | Node (default) |
| `3` | Development kit |

## Dev kit SKU reference

Used by `device devkit create --sku`:

| SKU | Product |
|-----|---------|
| `blank_device` | Blank Device |
| `xiao` | XIAO |
| `grove_vision_ai` | Grove Vision AI V2 |

Limit: max 5 development kits per SKU per organization.

## Known output schema gaps

The CLI typed models and tests confirm these fields. **If you need fields beyond what is listed, ask the user to provide a live API sample** — the platform may return additional properties that pass through on untyped commands but are stripped on typed devkit create output.

| Command | Schema confidence |
|---------|-------------------|
| `device devkit create` | **Partial** — code confirms `eui`, `device_key` only |
| `device devkit key` | **High** — code confirms `eui`, `device_key`, `token` |
| `data latest/list/chart` | **High** — aligned with OpenAPI v2 paths and query params |

Dev kit OpenAPI endpoints (`/device/create_development_kit`, `/device/view_node_key`) are **not documented** in the public SenseCAP HTTP API reference at the time of writing.

## Full command reference

Every command, flag, limit, example, and documented response field: **[commands.md](commands.md)**

## External references

- SenseCAP HTTP API overview: https://sensecap-docs.seeed.cc/introduction.html
- Device Management API: https://sensecap-docs.seeed.cc/device_management.html
- Group Management API: https://sensecap-docs.seeed.cc/group_management.html
- Device Data API: https://sensecap-docs.seeed.cc/device_data.html
- Measurement ID appendix: https://sensecap-docs.seeed.cc/
