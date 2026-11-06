# sensecraft-cli

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25-blue.svg)](https://go.dev/)
[![npm version](https://img.shields.io/npm/v/@seeed-studio/sensecraft-cli.svg)](https://www.npmjs.com/package/@seeed-studio/sensecraft-cli)

[中文](./README.zh.md) | [English](./README.md)

Command-line tool for [SenseCAP](https://sensecap.seeed.cc) devices and [SenseCraft AI](https://sensecraft.seeed.cc). One OpenAPI Access Key covers devices, telemetry, apps, models, files, and cloud devices. Built for people and for AI agents. Skills ship in [`skills/`](./skills/).

[Install](#install) · [AI Agent](#quick-start-for-ai-agents) · [Skills](#agent-skills) · [Auth](#authentication) · [Commands](#commands) · [Advanced](#advanced) · [Security](#security)

## What it does

| Area | What you can do |
|------|-----------------|
| Devices | List, view, bind, and unbind gateways, nodes, and development kits |
| Groups | Create groups and move devices |
| Telemetry | Latest values, history, and chart points |
| Apps | CodeCraft Hub: browse apps, manage your apps, materials, and activities |
| Models | Search the library, upload, and publish models |
| Files | Upload a local file and receive a `file_url` |
| Cloud devices | List Jetson and other cloud kits, deploy a model |
| reCamera | Check a classic reCamera app or model and open the Console deploy page |

The Console website stays at [https://sensecraft.seeed.cc](https://sensecraft.seeed.cc).

## Install

Node.js 16 or newer, plus network access to the SenseCAP and SenseCraft AI APIs. Building from source also needs Go 1.25 or newer (`go.mod`).

```bash
npx @seeed-studio/sensecraft-cli install
sensecraft-cli --version
```

On Windows the command is `sensecraft-cli.cmd`. PATH, upgrades, and uninstall are in [`INSTALL.md`](INSTALL.md).

### Quick start

Create an OpenAPI Access Key in the SenseCAP portal, then:

```bash
# 1. Save the key. --site is Global or China (default Global).
sensecraft-cli auth add --key-id <Key ID> --access-key <Access Key> --site Global

# 2. Confirm. The Access Key is not printed.
sensecraft-cli auth info

# 3. Call the API
sensecraft-cli device list
```

In an interactive terminal you can run `auth add` with no flags and fill in the prompts.

## Quick start for AI agents

Do this with the user. Do not ask them to paste the Access Key into the chat, and do not pass the secret on a command line that will be logged.

1. Install, if the command is missing:

```bash
npx @seeed-studio/sensecraft-cli install
```

2. Ask the user to run `auth add` on their own machine.
3. Check that a key is active:

```bash
sensecraft-cli auth info
```

4. Run a read command, for example `sensecraft-cli device list` or `sensecraft-cli app list --public`.
5. Load [`skills/sensecraft-cli/SKILL.md`](skills/sensecraft-cli/SKILL.md) first. It routes to the product reference.

Write commands (`unbind`, `delete`, `publish`, `deploy`, and similar) need `--yes` only after the user has confirmed that exact action.

## Agent Skills

| Skill | Description |
|-------|-------------|
| [`sensecraft-cli`](skills/sensecraft-cli/SKILL.md) | SenseCraft Data, SenseCraft AI, and CodeCraft Hub |
| [`create-app`](skills/sensecraft-cli/create-app/SKILL.md) | Turn a local project into a CodeCraft Hub app |

After a global npm install:

```bash
ls "$(npm root -g)/@seeed-studio/sensecraft-cli/skills"
```

## Authentication

| Command | Description |
|---------|-------------|
| `auth add` | Verify an Access Key, save it, and make it active |
| `auth info` | Show local profile metadata; add `--json` for structured output |

```bash
sensecraft-cli auth add --key-id <Key ID> --access-key <Access Key> --site Global
sensecraft-cli auth info
sensecraft-cli auth info --json
```

`auth info --json` returns `{ "profiles": [...] }`. Each profile contains only `site`, `org_id`, `key_id`, `kind`, `created_at`, and `active`.

| `--site` | SenseCAP OpenAPI | SenseCraft AI OpenAPI |
|----------|------------------|------------------------|
| `Global` | `https://sensecap.seeed.cc/openapi` | `https://api-gateway.seeed.cc/openapi` |
| `China` | `https://sensecap.seeed.cn/openapi` | `https://api-gateway.seeed.cc/openapi` |

The China site supports SenseCraft Data only. Use Global for SenseCraft AI and CodeCraft Hub. The same Access Key calls the APIs over HTTP Basic (Key ID as the username, Access Key as the password).

## Commands

Run `sensecraft-cli <command> --help` for flags. One example from each area:

```bash
sensecraft-cli device list --device-type 1
sensecraft-cli data latest --eui <EUI>

sensecraft-cli app list --public --sort popular
sensecraft-cli model list --public
sensecraft-cli file upload --type 1 --path ./cover.png --for-app
sensecraft-cli cloud device list
sensecraft-cli recamera check --model-id <id>
```

| Command | Description |
|---------|-------------|
| `device` | Gateways (`--device-type 1`), nodes (`2`, default), development kits (`3`) |
| `group` | Device groups |
| `data` | Latest telemetry, history, chart points |
| `app` | CodeCraft Hub and your apps. Omit `--public` to list your own apps |
| `model` | Model library. Omit `--mine` to list public models |
| `file` | Upload. `--type` `1` image, `2` model, `3` app, `4` firmware, `5` static resource (`--flow-id` required), `6` archive |
| `cloud` | Cloud devices and model deploy |
| `recamera` | Check or open the Console page. Flashing stays in the browser |
| `update` | Install the latest npm release |

`--sort` on `app list` is `popular`, `newest`, or `likes`. Common `--uniform-type` values: `40` reCamera, `32` XIAO ESP32S3, `53` reCamera Pro.

For an app never published, `app view --id <id>` returns only its name. Read the saved draft with `app view --id <id> --sketch`.

`app bom search --name "XIAO ESP32-S3 1.47"` matches every query token against the product name, SKU, category, note, or description, ignoring case and punctuation. Results keep catalog order. Catalog rows retain `block_support` and add boolean `block_supported` (`true` for `"是"`).

`file upload --name <file-name>` is the name sent to the server. The server may rewrite part of that name. Use the returned `file_url` as the access link.

`app activity list` returns every activity currently being held, with `count` equal to the number of rows in `list`.

## Advanced

Successful data commands print indented JSON on stdout. An object that contains `list` and has no `count` gains `count` for the current result; `page` and a count already present are kept. `auth add` and default `auth info` print text; `auth info --json` prints profile metadata. Failures go to stderr with a non-zero exit code.

`sensecraft-cli --version` prints the npm package version. A binary from plain `go build` prints `dev`.

Config file: `sensecraft.json`.

| How you run it | Where the file is |
|----------------|-------------------|
| npm install | macOS / Linux: `~/.config/sensecraft-cli/sensecraft.json`. Windows: `%APPDATA%\sensecraft-cli\sensecraft.json` |
| `go build` binary | `./sensecraft.json` |

After a successful command, stderr may show one English `Warning:` (a newer CLI is available; run `sensecraft-cli update`) and one `Notice:` (an ongoing activity, with dates and a link). Each line appears at most once every 24 hours. An interactive terminal then asks `Ignore this activity? [y/N]`.

## Build from source

```bash
go build -o sensecraft-cli .
./sensecraft-cli --version

./script/build.sh
```

`./script/build.sh` cross-compiles into `dist/` using the version in `packages/cli/package.json`.

## Security

The Access Key is a secret. Do not commit it, and do not paste it into an agent chat. `auth info` prints the Key ID only.

Commands that unbind, delete, publish, or deploy do nothing until you pass `--yes`. An agent must get a separate confirmation from the user before adding that flag.

This CLI does not flash firmware over USB or serial. `recamera open` only opens the Console deploy page.

## License

[MIT](LICENSE)
