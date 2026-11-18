# SenseCraft CLI

English | [中文](README-zh.md)

SenseCraft CLI (`sensecraft-cli`) is the official command-line tool for [SenseCraft](https://sensecraft.seeed.cc), built for developers and AI agents. Read sensor data, create models, publish applications, and manage devices and reCamera from the terminal.

It covers three products:

| Product | What it manages | Commands |
|---------|-----------------|----------|
| [SenseCraft Data](https://sensecap.seeed.cc) | Devices, groups, and sensor data | `device` · `group` · `data` |
| [SenseCraft AI](https://sensecraft.seeed.cc) | Model library and reCamera | `model` · `recamera` · `cloud` |
| [CodeCraft Hub](https://codecraft.seeed.cc/hub) | Applications | `app` |

All three platforms are accessed with an open Access Key. Create and manage keys in the [SenseCraft account center](https://account.seeed.cc/account/access-keys), then save one on this machine with `auth add`. Run `auth init` to authorize the CLI with the OAuth2 Device Flow.

It also runs from scripts and CI. Agent Skills live in `skills/`.

**Install and troubleshooting → [`INSTALL.md`](INSTALL.md)** (platforms, PATH, upgrade, uninstall, offline mirror, Cursor Skills).

## Requirements

- Node.js >= 16 (npm install and distribution)
- Go to build from source (see `go.mod`)

## Install

```bash
npx @seeed-studio/sensecraft-cli install
sensecraft-cli --version
sensecraft-cli update
```

Full steps, Windows PATH, offline install, and Skills setup: [`INSTALL.md`](INSTALL.md).

```bash
sensecraft-cli --help
# or
npx @seeed-studio/sensecraft-cli device list
```

## Quick start

### 1. Sign in

```bash
# Authorize the CLI with the OAuth2 Device Flow. Choose a site: Global / China
sensecraft-cli auth init

# Key already created on the account page: write it to this machine
sensecraft-cli auth add --key-id <Key ID> --access-key <Access Key>
```

Create and manage Access Keys: <https://account.seeed.cc/account/access-keys>.

The China site supports SenseCraft Data only. To use the CLI with SenseCraft AI and CodeCraft Hub, select the international site (Global).

### 2. Show the current config

```bash
sensecraft-cli auth info
```

### 3. Examples

```bash
# SenseCraft Data
sensecraft-cli device list
sensecraft-cli data latest --eui <EUI>

# SenseCraft AI. Look up task ids with model constants
sensecraft-cli model constants
sensecraft-cli model list --public --uniform-type 40
sensecraft-cli recamera check --model-id <id>
sensecraft-cli recamera open --model-id <id> --print-only
sensecraft-cli cloud device list --uniform-type 26

# CodeCraft Hub
sensecraft-cli app list --public --name "vision"
sensecraft-cli app list --public --uniform-type 40
```

`--uniform-type`: `40` is reCamera, `32` is XIAO ESP32S3, `53` is reCamera Pro. `26` on `cloud device list` is the device type the Jetson cloud UI commonly uses.

Delete, unbind, publish, and deploy commands require `--yes`. The agent confirmation steps are in [skills/sensecraft-cli/references/confirmation.md](skills/sensecraft-cli/references/confirmation.md).

## Config

| Item | Meaning |
|------|---------|
| npm install default | macOS/Linux: `~/.config/sensecraft-cli/sensecraft.json`; Windows: `%APPDATA%\sensecraft-cli\sensecraft.json` (the wrapper sets `SENSECRAFT_CLI_PATH`) |
| `go build` binary default | `sensecraft.json` in the current working directory |
| `SENSECRAFT_CLI_PATH` | Config directory (the file name is `sensecraft.json`) |
| `SENSECRAFT_CONFIG_FILE` | Full path to the config file (wins over the two rows above) |
| `SENSECRAFT_DATA_HOST` | SenseCAP OpenAPI origin. Unset means `https://sensecap.seeed.cc` for Global and `https://sensecap.seeed.cn` for China |
| `SENSECRAFT_AI_HOST` | App and model API origin. Unset means `https://api-gateway.seeed.cc`. This does not change the pages the CLI prints |
| `SENSECRAFT_AUTH_HOST` | Account-center origin. Unset means `https://account.seeed.cc` for Global and `https://account.seeed.cn` for China |
| `SENSECRAFT_APP_SITE` | App edit and detail pages. Unset means `https://codecraft.seeed.cc/hub`. A path is allowed |
| `SENSECRAFT_CONSOLE_HOST` | Model detail page origin. Unset means `https://sensecraft.seeed.cc` |

The CLI does not switch sites because a hostname contains `test-`. Export the variable for the host you want; leave it unset to use the public site above. Values are origins with no query string. `SENSECRAFT_DATA_HOST`, `SENSECRAFT_AI_HOST`, `SENSECRAFT_AUTH_HOST`, and `SENSECRAFT_CONSOLE_HOST` must not include a path; a trailing `/openapi` or `/authapi` is removed. `SENSECRAFT_APP_SITE` may be `https://example.com/hub`.

## Output

- Successful business commands: JSON on stdout, indented by 2 spaces
- `auth info` / `auth init`: human-readable text (not JSON)
- Failure: stderr and a non-zero exit code
- Version: `sensecraft-cli --version` (same as the npm package version)

## Scripts and CI

A non-interactive environment does not open a browser. Write the key with `auth add`, and point `SENSECRAFT_CONFIG_FILE` at a config file in the project so it does not land in `~/.config/sensecraft-cli/`.

```bash
export SENSECRAFT_CONFIG_FILE="$PWD/.sensecraft/sensecraft.json"
sensecraft-cli auth add \
  --key-id "$SENSECRAFT_KEY_ID" \
  --access-key "$SENSECRAFT_ACCESS_KEY" \
  --site Global
sensecraft-cli device list
```

This file contains the Access Key. Do not commit it. `--site` defaults to `Global`. On success, parse the JSON on stdout. On failure, check the exit code; the error is on stderr. `auth info` prints text.

## AI Agent Skills

The entry is [`skills/sensecraft-cli/SKILL.md`](skills/sensecraft-cli/SKILL.md). Device, model, and app procedures are in `skills/sensecraft-cli/references/`. Turning a local project into an app uses the nested skill `skills/sensecraft-cli/create-app/`.

After install, symlink that directory into the agent skills folder. Do not copy it. A CLI upgrade updates the skill behind the link. Steps: [`INSTALL.md` §9](INSTALL.md#9-agent-skills-cursor-and-others).

## Build from source

Full steps: [`INSTALL.md` §8](INSTALL.md#8-build-from-source). Summary:

```bash
# Local debug build
go build -o sensecraft-cli .
./sensecraft-cli --version

# Release binaries for every platform → dist/ (version injected with -ldflags)
./script/build.sh
```

Checks for people contributing to this repo: `./script/smoke-ai.sh` and `./script/smoke-sensecap.sh`. The isolated install check needs `./script/build.sh` first, then `npm run test:install` inside `packages/cli`. Projects that only use the CLI do not need these scripts.

## License

[MIT](LICENSE)

---

## Command docs

This page spells out SenseCraft Data. Flags, limits, and examples for the other commands:

| Area | Doc |
|------|-----|
| SenseCraft Data | [data-commands.md](skills/sensecraft-cli/references/data-commands.md) |
| Model library | [model-commands.md](skills/sensecraft-cli/references/model-commands.md) |
| reCamera | [recamera-commands.md](skills/sensecraft-cli/references/recamera-commands.md) |
| Cloud deploy and orders | [cloud-commands.md](skills/sensecraft-cli/references/cloud-commands.md) |
| CodeCraft Hub apps | [app-commands.md](skills/sensecraft-cli/references/app-commands.md) |
| File upload | [file-commands.md](skills/sensecraft-cli/references/file-commands.md) |

## Command reference (SenseCraft Data)

### auth — credentials and config

#### `auth init`

Authorize the CLI with the OAuth2 Device Flow. Choose a site interactively: Global / China.

| Flag | Meaning |
|------|---------|
| `--sandbox` | Sandbox: do not send real network requests |

#### `auth add`

Write an existing Access Key to this machine and make it the active key. A non-interactive environment must pass `--key-id` and `--access-key`.

| Flag | Required | Default | Meaning |
|------|----------|---------|---------|
| `--key-id` | Can be omitted in an interactive terminal | — | Key ID |
| `--access-key` | Can be omitted in an interactive terminal | — | Access Key |
| `--site` | No | `Global` | `Global` or `China` |

```bash
sensecraft-cli auth add --key-id <Key ID> --access-key <Access Key> --site Global
```

#### `auth info`

`info` prints the site, organization ID, and Key ID of the current Access Key. It does not print the secret. `auth info --json` returns `{ "profiles": [...] }`. Each profile contains only `site`, `org_id`, `key_id`, `kind`, `created_at`, and `active`.

| `--site` | SenseCAP OpenAPI | SenseCraft AI OpenAPI |
|----------|------------------|------------------------|
| `Global` | `https://sensecap.seeed.cc/openapi` | `https://api-gateway.seeed.cc/openapi` |
| `China` | `https://sensecap.seeed.cn/openapi` | `https://api-gateway.seeed.cc/openapi` |

---

### device — device management (SenseCraft Data)

#### `device list`

List devices on the current account.

| Flag | Required | Default | Meaning |
|------|----------|---------|---------|
| `--device-type` | No | `2` | Device type: `1` gateway, `2` node, `3` dev kit |
| `--group-uuid` | No | — | Filter by group UUID. Omit it to return devices in every group |

```bash
# List nodes (default)
sensecraft-cli device list

# List gateways
sensecraft-cli device list --device-type 1

# List dev kits
sensecraft-cli device list --device-type 3

# List devices in one group
sensecraft-cli device list --group-uuid <group-uuid>
```

#### `device view`

Device details. Up to 50 EUIs.

| Flag | Required | Default | Meaning |
|------|----------|---------|---------|
| `--eui` | Yes | — | Device EUI. Repeat the flag for more than one |
| `--device-type` | No | `2` | Device type: `1` gateway, `2` node, `3` dev kit |

```bash
sensecraft-cli device view --eui 0123456789ABCDEF
```

#### `device channels`

List channels on a node. Up to 50 EUIs.

| Flag | Required | Meaning |
|------|----------|---------|
| `--eui` | Yes | Device EUI. Repeat the flag for more than one |

```bash
sensecraft-cli device channels --eui 0123456789ABCDEF
```

#### `device status`

Runtime status. Up to 50 EUIs.

| Flag | Required | Meaning |
|------|----------|---------|
| `--eui` | Yes | Device EUI. Repeat the flag for more than one |

```bash
sensecraft-cli device status --eui 0123456789ABCDEF
```

#### `device bind`

Bind a device to the current account.

| Flag | Required | Meaning |
|------|----------|---------|
| `--eui` | Yes | Device EUI |
| `--code` | Yes | Device bind code |
| `--name` | No | Device name |
| `--group-uuid` | No | Target group UUID |
| `--longitude` | No | Longitude |
| `--latitude` | No | Latitude |

```bash
sensecraft-cli device bind --eui 0123456789ABCDEF --code ABCD1234 --name "My sensor"
```

#### `device unbind`

Unbind devices. Up to 50 EUIs. Requires `--yes`.

| Flag | Required | Meaning |
|------|----------|---------|
| `--eui` | Yes | Device EUI. Repeat the flag for more than one |
| `--yes` | Yes | Confirm |

```bash
sensecraft-cli device unbind --eui 0123456789ABCDEF --yes
```

#### `device sim`

SIM card status.

| Flag | Required | Meaning |
|------|----------|---------|
| `--iccid` | Yes | SIM ICCID |

```bash
sensecraft-cli device sim --iccid 89860000000000000000
```

---

### device devkit — dev kits

#### `device devkit create`

Create a dev-kit device. Each SKU can be created at most 5 times in one organization.

| Flag | Required | Meaning |
|------|----------|---------|
| `--sku` | Yes | Dev-kit SKU |
| `--name` | No | Device name |

Supported SKUs:

| SKU | Meaning |
|-----|---------|
| `blank_device` | Blank Device |
| `xiao` | XIAO |
| `grove_vision_ai` | Grove Vision AI V2 |

```bash
sensecraft-cli device devkit create --sku xiao --name "My XIAO kit"
```

#### `device devkit key`

Show `device_key` and `token` for a dev-kit device. The EUI is 16 characters. Other device types are not accepted.

| Flag | Required | Meaning |
|------|----------|---------|
| `--eui` | Yes | Dev-kit EUI (16 characters) |

```bash
sensecraft-cli device devkit key --eui 0123456789ABCDEF
```

---

### group — device groups

#### `group list`

List every device group on the current account.

#### `group create`

Create a device group. Up to 50 groups.

| Flag | Required | Meaning |
|------|----------|---------|
| `--name` | Yes | Group name |

```bash
sensecraft-cli group create --name "Greenhouse sensors"
```

#### `group update`

Rename a device group.

| Flag | Required | Meaning |
|------|----------|---------|
| `--group-uuid` | Yes | Group UUID |
| `--name` | Yes | New group name |

```bash
sensecraft-cli group update --group-uuid <uuid> --name "New name"
```

#### `group delete`

Delete a device group. Devices in the group move to the default group. Requires `--yes`.

| Flag | Required | Meaning |
|------|----------|---------|
| `--group-uuid` | Yes | Group UUID |
| `--yes` | Yes | Confirm |

```bash
sensecraft-cli group delete --group-uuid <uuid> --yes
```

#### `group move-devices`

Move devices into a group. Requires `--yes`.

| Flag | Required | Meaning |
|------|----------|---------|
| `--group-uuid` | Yes | Target group UUID |
| `--eui` | Yes | Device EUI. Repeat the flag for more than one |
| `--yes` | Yes | Confirm |

```bash
sensecraft-cli group move-devices --group-uuid <uuid> --eui 0123456789ABCDEF --yes
```

---

### data — sensor data

#### `data latest`

Latest telemetry from the past year. When no channel is given, returns the latest point on each channel.

| Flag | Required | Meaning |
|------|----------|---------|
| `--eui` | Yes | Device EUI |
| `--measurement-id` | No | Measurement ID |
| `--channel-index` | No | Channel index |

```bash
sensecraft-cli data latest --eui 0123456789ABCDEF --measurement-id 4097 --channel-index 1
```

#### `data list`

Historical telemetry. One request covers about a month at most, and only the last three months can be queried.

| Flag | Required | Default | Meaning |
|------|----------|---------|---------|
| `--eui` | Yes | — | Device EUI |
| `--measurement-id` | No | — | Measurement ID |
| `--channel-index` | No | — | Channel index |
| `--limit` | No | — | Number of records |
| `--time-start` | No | — | Start timestamp (milliseconds) |
| `--time-end` | No | — | End timestamp (milliseconds) |

```bash
sensecraft-cli data list --eui 0123456789ABCDEF --limit 100 --time-start 1700000000000 --time-end 1700086400000
```

#### `data chart`

Aggregated chart points. Up to one year, and about 250 points per measurement.

| Flag | Required | Default | Meaning |
|------|----------|---------|---------|
| `--eui` | Yes | — | Device EUI |
| `--measurement-id` | No | — | Measurement ID |
| `--channel-index` | No | — | Channel index |
| `--interval` | No | `60` | Bucket length (minutes) |
| `--time-start` | No | — | Start timestamp (milliseconds) |
| `--time-end` | No | — | End timestamp (milliseconds) |

```bash
sensecraft-cli data chart --eui 0123456789ABCDEF --interval 60 --time-start 1700000000000 --time-end 1702592000000
```

---

## Apps and uploads

For an app never published, `app view --id <id>` returns only its name. Read the saved draft with `app view --id <id> --sketch`.

`app bom search --name "XIAO ESP32-S3 1.47"` matches every query token against the product name, SKU, category, note, or description, ignoring case and punctuation. Results keep catalog order. Catalog rows retain `block_support` and add boolean `block_supported` (`true` for `"是"`).

`file upload --name <file-name>` is the name sent to the server. The filename used to download the file is the object name in the returned `file_url`. A name specified for the upload is not necessarily that download name.

`app activity list` returns every activity currently being held, with `count` equal to the number of rows in `list`.

## Advanced

Successful data commands print indented JSON on stdout. An object that contains `list` and has no `count` gains `count` for the current result; `page` and a count already present are kept. `auth add` and default `auth info` print text; `auth info --json` prints profile metadata. Failures go to stderr with a non-zero exit code.

## Typical workflow

```bash
# Install and sign in
npx @seeed-studio/sensecraft-cli install

# List groups
sensecraft-cli group list

# List node devices
sensecraft-cli device list

# Device details and channels
sensecraft-cli device view --eui 0123456789ABCDEF
sensecraft-cli device channels --eui 0123456789ABCDEF

# Latest data
sensecraft-cli data latest --eui 0123456789ABCDEF
```
