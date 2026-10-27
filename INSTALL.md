# SenseCraft CLI install guide

How to install, upgrade, and uninstall `sensecraft-cli`, and how to connect Agent Skills to Cursor and other tools. Commands and usage: [`README.md`](README.md) (Chinese: [`README-zh.md`](README-zh.md)).

## 1. Requirements

| Item | Requirement |
|------|-------------|
| Node.js | **≥ 16** (18 or 20 LTS recommended) |
| Network | Access to the npm registry and the binary download host (Aliyun OSS by default) |
| Optional | Go, only when building from source. See `go.mod` for the version |

### Supported platforms

| OS | Architecture | Binary name |
|----|--------------|-------------|
| macOS | Intel (`x64`) | `sensecraft-cli-darwin-amd64` |
| macOS | Apple Silicon (`arm64`) | `sensecraft-cli-darwin-arm64` |
| Linux | `x64` / `arm64` | `sensecraft-cli-linux-amd64` / `…-arm64` |
| Windows | `x64` / `arm64` | `sensecraft-cli-windows-amd64.exe` / `…-arm64.exe` |

Any other combination fails install with `Unsupported platform`.

---

## 2. Recommended install (npm / npx)

This path fits most people. The installer:

1. Downloads the binary for this platform, pinned to this npm package version
2. Registers the `sensecraft-cli` command globally
3. In an interactive terminal, offers `auth init` when no local config exists yet

### macOS / Linux

```bash
# Check Node
node -v    # >= v16

# Install (the first run downloads the binary)
npx @seeed-studio/sensecraft-cli install

# Check
sensecraft-cli --version
sensecraft-cli --help
```

If the shell cannot find the command, add the npm global bin directory to PATH:

```bash
npm prefix -g
# Common paths:
#   macOS (Homebrew Node): /opt/homebrew/bin or /usr/local/bin
#   Linux: ~/.npm-global/bin or /usr/bin
echo 'export PATH="$(npm prefix -g)/bin:$PATH"' >> ~/.zshrc   # or ~/.bashrc
source ~/.zshrc
```

### Windows (PowerShell or cmd)

```powershell
node -v
npx @seeed-studio/sensecraft-cli install
sensecraft-cli --version
```

- The global command is usually `sensecraft-cli.cmd`, in the `npm prefix -g` directory.
- If a new terminal still cannot find it, add that directory to the system **PATH**, or sign out and open the terminal again.
- PowerShell 5+ or Windows Terminal is the comfortable choice. Git Bash usually works too.

### Call it without a global install

```bash
npx @seeed-studio/sensecraft-cli --version
npx @seeed-studio/sensecraft-cli auth info
npx @seeed-studio/sensecraft-cli device list
```

Each `npx` run may still trigger postinstall, which downloads the binary or reuses the local cache.

### Install a specific version

```bash
npx @seeed-studio/sensecraft-cli@<version> install
# or
npm install -g @seeed-studio/sensecraft-cli@<version>
```

Binary URLs look like:

`{RELEASE_BASE}/v{version}/sensecraft-cli-{os}-{arch}[.exe]`

Default `RELEASE_BASE`:

`https://sensecraft-statics.seeed.cc/static/bin/sensecraft-cli`

---

## 3. First sign-in

After install:

```bash
# Open Access Key for SenseCraft Data, SenseCraft AI, and CodeCraft Hub
sensecraft-cli auth init
```

`auth init` authorizes the CLI with the OAuth2 Device Flow. If you already have a key, use `auth add`. Create and manage keys at <https://account.seeed.cc/account/access-keys>.

Choose a site during init:

| Choice | Use |
|--------|-----|
| Global | International site. SenseCraft Data, SenseCraft AI, and CodeCraft Hub |
| China | China site. SenseCraft Data only |

The China site supports SenseCraft Data only. To use the CLI with SenseCraft AI and CodeCraft Hub, select the international site (Global).

Show the config:

```bash
sensecraft-cli auth info
```

### Scripts and CI

A non-interactive environment does not run `auth init`. Write the key with `auth add`, and point `SENSECRAFT_CONFIG_FILE` at a config file in the project:

```bash
export SENSECRAFT_CONFIG_FILE="$PWD/.sensecraft/sensecraft.json"
sensecraft-cli auth add \
  --key-id "$SENSECRAFT_KEY_ID" \
  --access-key "$SENSECRAFT_ACCESS_KEY" \
  --site Global
sensecraft-cli device list
```

This file contains the Access Key. Do not commit it. `--site` defaults to `Global`. On success, a business command prints indented JSON on stdout. `auth info` and `auth init` print plain text. On failure, the message is on stderr and the exit code is non-zero.

---

## 4. Config file location

| Situation | Path |
|-----------|------|
| **npm / npx install (default)** | macOS/Linux: `~/.config/sensecraft-cli/sensecraft.json`; Windows: `%APPDATA%\sensecraft-cli\sensecraft.json` |
| **Running a `go build` binary** | `sensecraft.json` in the current working directory, unless an environment variable is set |

Environment variables, highest priority first:

| Variable | Meaning |
|----------|---------|
| `SENSECRAFT_CONFIG_FILE` | Full path to the config file |
| `SENSECRAFT_CLI_PATH` | Config **directory** (the file name is always `sensecraft.json`) |

The npm wrapper sets `SENSECRAFT_CLI_PATH` to the default directory above, so an npm-installed CLI and a plain `go build` binary do not read the same file by default.

### Move a config file

```bash
# Example: move a config from the working directory to the npm default
mkdir -p ~/.config/sensecraft-cli
cp ./sensecraft.json ~/.config/sensecraft-cli/sensecraft.json
```

Windows:

```powershell
New-Item -ItemType Directory -Force "$env:APPDATA\sensecraft-cli"
Copy-Item .\sensecraft.json "$env:APPDATA\sensecraft-cli\sensecraft.json"
```

---

## 5. Upgrade

When the CLI was installed with npm:

```bash
sensecraft-cli update
```

The command looks up the latest version on npm and runs `npm install -g @seeed-studio/sensecraft-cli@<version>`. It reinstalls even when the local version number already matches. The installer then compares `vendor/.version` with the new package version and downloads the binary again when they differ. The Access Key in the config directory is kept. `--version` prints the local version only and does not contact npm.

Other forms:

```bash
sensecraft-cli update --check   # query only, do not install
npm install -g @seeed-studio/sensecraft-cli@latest
```

While you run other commands, a newer version is reported once on stderr, with a hint to run `sensecraft-cli update`. The same version is mentioned at most once every 24 hours.

If the program was not installed with npm (for example a binary you compiled yourself), `update` only prints the manual install command. It does not replace the file that is running.

Check:

```bash
sensecraft-cli --version
# should match the npm package version
```

---

## 6. Uninstall

```bash
npm uninstall -g @seeed-studio/sensecraft-cli
```

Optional cleanup:

```bash
# Config (this deletes the local Access Key; back it up first)
rm -rf ~/.config/sensecraft-cli          # macOS / Linux
# Windows: Remove-Item -Recurse "$env:APPDATA\sensecraft-cli"

# npx cache (optional)
npm cache clean --force
```

The `vendor/` binary inside the package directory goes away with the global package.

---

## 7. Offline, intranet, or a custom binary host

### Use a local `dist/` you already built

```bash
./script/build.sh
export SENSECRAFT_CLI_LOCAL_DIST="$PWD/dist"
npx @seeed-studio/sensecraft-cli install
# or, from a cloned packages/cli:
# SENSECRAFT_CLI_LOCAL_DIST=$PWD/../../dist npm run postinstall
```

### Override the download base URL

By default the installer fetches `{base}/v{version}/{artifact}` from OSS. For an internal mirror:

```bash
export SENSECRAFT_CLI_RELEASE_BASE="https://your-mirror.example.com/sensecraft-cli"
npx @seeed-studio/sensecraft-cli install
```

The mirror layout must match a release, for example:

```text
…/sensecraft-cli/v<version>/sensecraft-cli-darwin-arm64
…/sensecraft-cli/v<version>/sensecraft-cli-linux-amd64
…
```

### Skip the download and reuse `vendor/`

```bash
export SENSECRAFT_CLI_SKIP_DOWNLOAD=1
# Install fails when vendor has no binary for the current version
```

### Optional checksum

If the publisher provides a SHA256:

```bash
export SENSECRAFT_CLI_SHA256="<hex>"
```

Install fails when the download does not match.

### Proxy

The installer follows Node and system proxy variables, for example:

```bash
export HTTPS_PROXY=http://proxy.example.com:8080
export HTTP_PROXY=http://proxy.example.com:8080
```

---

## 8. Build from source

```bash
git clone <this repository URL>
cd sensecraft-cloud-terminal

# Local debug build
go build -o sensecraft-cli .
./sensecraft-cli --version

# Release artifacts for every platform → dist/
./script/build.sh
```

A binary built from source stores config in `sensecraft.json` in the **current directory**. To share the npm install's config:

```bash
export SENSECRAFT_CLI_PATH="$HOME/.config/sensecraft-cli"
./sensecraft-cli auth info
```

Checks for people contributing to this repo: `./script/smoke-ai.sh` and `./script/smoke-sensecap.sh`. The isolated install check needs `./script/build.sh` first, then:

```bash
cd packages/cli && SENSECRAFT_CLI_LOCAL_DIST=$PWD/../../dist npm run test:install
```

Projects that only use the CLI do not need these scripts.

---

## 9. Agent Skills (Cursor and others)

Skills are Markdown capability notes for an agent. They are not CLI subcommands. There is one entry: `skills/sensecraft-cli/SKILL.md`. Product procedures are in `references/` next to it. Turning a local project into an app uses the nested skill `skills/sensecraft-cli/create-app/`.

Skills ship inside the npm package. Symlink the agent skills directory at that copy. Do not copy the files. After `sensecraft-cli update` replaces the package, the link still reads the new docs.

```bash
npm install -g @seeed-studio/sensecraft-cli
SRC="$(npm root -g)/@seeed-studio/sensecraft-cli/skills/sensecraft-cli"
mkdir -p ~/.cursor/skills
ln -sfn "$SRC" ~/.cursor/skills/sensecraft-cli
```

Claude Code, same directory link:

```bash
mkdir -p ~/.claude/skills
ln -sfn "$SRC" ~/.claude/skills/sensecraft-cli
```

Link the directory, not `SKILL.md` alone. `references/` and `assets/` are reached by relative paths, and splitting them breaks those paths. Run `ln` again after Node or the npm global directory changes. If Windows cannot create a symlink, copy the directory, and copy it again after every CLI upgrade.

While developing this repository, link the repo's `skills/sensecraft-cli` into the agent skills directory. Do not link an old npm package.

Usage:

- Read `sensecraft-cli` first. It opens the matching file under `references/`.
- `sensecraft-cli` must be on PATH, and the user must already be signed in.
- The agent must be able to run commands on this machine. A chat-only host can read the notes and cannot claim a command ran.
- `auth init`, local USB or IP deploy, and flashing stay with the user.

---

## 10. Checks

```bash
sensecraft-cli --version
sensecraft-cli auth info

# Sandbox smoke that does not spend a real quota (needs a profile or a temporary config)
sensecraft-cli model constants --sandbox
sensecraft-cli app list --mine --sandbox
sensecraft-cli device list --sandbox
sensecraft-cli group list --sandbox
sensecraft-cli data latest --eui SANDBOXEUI000001 --sandbox
```

---

## 11. Common problems

### `sensecraft-cli: command not found`

- Confirm `npx @seeed-studio/sensecraft-cli install` or `npm install -g …` finished
- Add `$(npm prefix -g)/bin` to PATH (on Windows, `npm prefix -g`)
- Open a new terminal and try again

### Binary download failed

The installer suggests:

1. Confirm the release artifacts are at `{RELEASE_BASE}/v{version}/`
2. Point `SENSECRAFT_CLI_LOCAL_DIST` at a local `dist/`
3. Point `SENSECRAFT_CLI_RELEASE_BASE` at a mirror that works

When a company network blocks OSS, use a local `dist/` or an internal mirror.

### `auth info` says it is not initialized, or install asks for init again

- An npm install expects an Access Key (`app_config_list`). SenseCraft Data, SenseCraft AI, and CodeCraft Hub share that key.
- Confirm both commands use the same config file: `echo $SENSECRAFT_CLI_PATH` / `%SENSECRAFT_CLI_PATH%`.
- Write one with `sensecraft-cli auth init`, or with `auth add` when you already have an Access Key.

### The npm install and a `go build` binary see different configs

See "Config file location" above. The first defaults to `~/.config/…`, the second to the working directory. Point both at one place with `SENSECRAFT_CLI_PATH` or `SENSECRAFT_CONFIG_FILE`.

### Windows antivirus quarantines the binary

The first download of `sensecraft-cli.exe` may be quarantined. Allow the install directory or `vendor`, then install again.

### The version number does not change after an upgrade

```bash
npm uninstall -g @seeed-studio/sensecraft-cli
npm cache clean --force
npx @seeed-studio/sensecraft-cli@<target version> install
sensecraft-cli --version
```

---

## 12. Related docs

| Doc | Contents |
|-----|----------|
| [`README.md`](README.md) | English overview and command reference |
| [`README-zh.md`](README-zh.md) | Chinese overview and command reference |
| [`docs/md-to-app.md`](docs/md-to-app.md) | Write an app in Markdown |
| [`skills/README.md`](skills/README.md) | Skills layout and agent conventions |
| [`LICENSE`](LICENSE) | MIT |
