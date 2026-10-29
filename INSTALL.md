# SenseCraft CLI install guide

How to install, upgrade, and uninstall `sensecraft-cli`, and how to connect Agent Skills to Cursor and other tools. Commands and sign-in: [`README.md`](README.md) (Chinese: [`README.zh.md`](README.zh.md)).

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
3. In an interactive terminal, asks for an OpenAPI Access Key when no local config exists yet

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

`npx` may still run postinstall and download or reuse a cached binary.

### Install a specific version

```bash
npx @seeed-studio/sensecraft-cli@<version> install
# or
npm install -g @seeed-studio/sensecraft-cli@<version>
```

The binary URL looks like:

`{RELEASE_BASE}/v{version}/sensecraft-cli-{os}-{arch}[.exe]`

Default `RELEASE_BASE`:

`https://sensecraft-statics.seeed.cc/static/bin/sensecraft-cli`

---

## 3. First sign-in

Sign in with the OpenAPI Access Key from the SenseCAP portal. After the Key ID and Access Key are saved on this machine, devices, groups, sensor data, apps, models, files, and cloud devices all use that one key.

```bash
sensecraft-cli auth add --key-id <Key ID> --access-key <Access Key> --site Global
sensecraft-cli auth info
```

`--site` is `Global` or `China`. The default is `Global`. A non-interactive environment must pass `--key-id` and `--access-key`. `auth info` does not print the Access Key.

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

While you run other commands, a newer version found in the cache is reported on stderr at the end, with a hint to run `sensecraft-cli update`. The same version is mentioned at most once every 24 hours. A newly published version is mentioned again. The CLI does not contact npm again inside that 24-hour window. A local `dev` build, CI, and `SENSECRAFT_CLI_NO_UPDATE_NOTIFIER` skip the hint.

An ongoing activity is also printed on stderr as `Notice:`. The name, dates, and detail link come from `https://api-gateway.seeed.cc/openapi/activity/list` (no query, so activities that have not ended). It appears at most once every 24 hours. An interactive terminal asks `Ignore this activity? [y/N]`. Answering `y` suppresses that `activity_id`. Answering `n` shows it again after 24 hours. A non-interactive run prints the notice, does not ask, and does not record an ignore. CI and `SENSECRAFT_CLI_NO_ACTIVITY_NOTIFIER` skip it.

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
# Config (this deletes the local token / Access Key; back it up first)
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
…/sensecraft-cli/v0.1.3/sensecraft-cli-darwin-arm64
…/sensecraft-cli/v0.1.3/sensecraft-cli-linux-amd64
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

Checks while developing this repo:

```bash
./script/smoke-ai.sh
./script/smoke-sensecap.sh
cd packages/cli && SENSECRAFT_CLI_LOCAL_DIST=$PWD/../../dist npm run test:install
```

The public overview is [`README.md`](README.md).

---

## 9. Agent Skills (Cursor and others)

Skills are Markdown capability notes for an agent. They are not CLI subcommands.

### Where they come from

| Source | Path |
|--------|------|
| This repository | `skills/` |
| After an npm install | `skills/` inside the global package, next to `bin` and `scripts` |

Find the global package:

```bash
npm root -g
# for example …/node_modules/@seeed-studio/sensecraft-cli/skills/sensecraft-cli/SKILL.md
```

### Layout

| Skill | Path | Auth |
|-------|------|------|
| Entry | `skills/sensecraft-cli/SKILL.md` | Access Key |
| Create an app from a local project | `skills/sensecraft-cli/create-app/` | Access Key |

Discovery notes: [`skills/README.md`](skills/README.md).

### Connect Cursor

Pick one:

1. **This repository is the workspace.** Open `sensecraft-cloud-terminal` as the Cursor project. The agent can read `skills/` directly.

2. **Copy or symlink into the Cursor skills directory** (the path depends on the Cursor version; a common place is the skills folder in the user directory):

   ```bash
   ln -s "$PWD/skills/sensecraft-cli" ~/.cursor/skills/sensecraft-cli
   ```

3. **Use the skills from the npm package.** Link `$(npm root -g)/@seeed-studio/sensecraft-cli/skills` to a path the agent discovers.

### Connect Claude Code and other agents

These skills use a standard `SKILL.md` (YAML `name` / `description` plus Markdown). They contain no Cursor-only instructions. A host that supports Agent Skills can copy or symlink the skill directory:

```bash
# Claude Code, project scope
mkdir -p .claude/skills
for d in "$PWD"/skills/*; do
  ln -s "$d" ".claude/skills/$(basename "$d")"
done

# Claude Code, user scope (optional)
mkdir -p ~/.claude/skills
ln -s "$PWD/skills/sensecraft-cli" ~/.claude/skills/sensecraft-cli
```

Other agents:

1. If the host discovers `SKILL.md` on its own, put `skills/*/` into its project or user skills directory. Use that host's documented path.
2. If it does not support Skills, load `skills/sensecraft-cli/SKILL.md` as system or project context.
3. The agent must be able to run commands on this machine before it can call `sensecraft-cli`. A chat-only host can read the notes and cannot claim a command ran.
4. With no Access Key, use the split `auth init` (`--no-wait`, then hand the verification URL to the user). Local USB or IP deploy and flashing stay with the user.

Usage:

- Read **`sensecraft-cli`** first. It opens the matching file under `references/` and covers SenseCraft Data, SenseCraft AI, and CodeCraft Hub.
- `sensecraft-cli` must be on PATH. The agent runs `auth info` first. With no Access Key, it runs `sensecraft-cli auth init --site Global --no-wait`, shows `verification_url` unchanged, and ends the turn. After the user authorizes, it continues with `--device-code`. Do not run bare `auth init` in a one-shot shell. An existing key uses `auth add`.
- The agent must be able to run commands on this machine. A chat-only host can read the notes and cannot claim a command ran.
- Local USB or IP deploy and flashing stay in the browser or on the device.

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

### `auth info` says there is no Access Key yet

- Write one with `sensecraft-cli auth add --key-id <Key ID> --access-key <Access Key> --site Global`. Use `--site China` for the China site.
- Confirm both commands use the same config file: `echo $SENSECRAFT_CLI_PATH` / `%SENSECRAFT_CLI_PATH%`.
- Devices, apps, models, and cloud devices share that Access Key.

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
| [`README.md`](README.md) | English overview |
| [`README.zh.md`](README.zh.md) | Chinese overview |
| [`docs/sensecraft-cli-smoke.md`](docs/sensecraft-cli-smoke.md) | Release smoke checklist |
| [`docs/sensecraft-cli-test-cases.md`](docs/sensecraft-cli-test-cases.md) | Full functional test cases |
| [`skills/README.md`](skills/README.md) | Skills layout and agent conventions |
| [`LICENSE`](LICENSE) | MIT |
