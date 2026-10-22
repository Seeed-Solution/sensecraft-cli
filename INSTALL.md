# SenseCraft CLI 安装指南

本文说明如何安装、升级、卸载 `sensecraft-cli`，以及如何把 Agent Skills 接到 Cursor 等工具。命令参考与业务用法见 [`README.md`](README.md)。

## 1. 环境要求

| 项 | 要求 |
|----|------|
| Node.js | **≥ 16**（推荐 18 / 20 LTS） |
| 网络 | 能访问 npm registry，以及二进制下载源（默认阿里云 OSS） |
| 可选 | Go（仅从源码构建时需要，版本见 `go.mod`） |

### 支持的平台

| OS | 架构 | 二进制名 |
|----|------|----------|
| macOS | Intel (`x64`) | `sensecraft-cli-darwin-amd64` |
| macOS | Apple Silicon (`arm64`) | `sensecraft-cli-darwin-arm64` |
| Linux | `x64` / `arm64` | `sensecraft-cli-linux-amd64` / `…-arm64` |
| Windows | `x64` / `arm64` | `sensecraft-cli-windows-amd64.exe` / `…-arm64.exe` |

其它组合会在安装时报 `Unsupported platform`。

---

## 2. 推荐安装（npm / npx）

适用于大多数用户。安装器会：

1. 按当前平台下载**与本 npm 包版本一致**的二进制  
2. 全局注册 `sensecraft-cli` 命令  
3. 若本地尚无配置，在交互终端下引导 `auth init`

### macOS / Linux

```bash
# 确认 Node
node -v    # >= v16

# 安装（首次会下载二进制）
npx @seeed-studio/sensecraft-cli install

# 验证
sensecraft-cli --version
sensecraft-cli --help
```

若提示找不到命令，把 npm 全局 bin 加入 PATH：

```bash
npm prefix -g
# 常见路径：
#   macOS (Homebrew Node): /opt/homebrew/bin 或 /usr/local/bin
#   Linux: ~/.npm-global/bin 或 /usr/bin
echo 'export PATH="$(npm prefix -g)/bin:$PATH"' >> ~/.zshrc   # 或 ~/.bashrc
source ~/.zshrc
```

### Windows（PowerShell 或 cmd）

```powershell
node -v
npx @seeed-studio/sensecraft-cli install
sensecraft-cli --version
```

- 全局命令一般为 `sensecraft-cli.cmd`，装在 `npm prefix -g` 目录下。  
- 若新开终端仍找不到命令：把该目录加入系统 **PATH**，或注销后重开终端。  
- 建议使用 **PowerShell 5+** 或 Windows Terminal；Git Bash 一般也可用。

### 不全局安装、临时调用

```bash
npx @seeed-studio/sensecraft-cli --version
npx @seeed-studio/sensecraft-cli auth info
npx @seeed-studio/sensecraft-cli device list
```

`npx` 每次仍可能触发 postinstall 下载/复用本地缓存的二进制。

### 指定版本安装

```bash
npx @seeed-studio/sensecraft-cli@0.1.3 install
# 或
npm install -g @seeed-studio/sensecraft-cli@0.1.3
```

二进制 URL 形如：

`{RELEASE_BASE}/v{版本}/sensecraft-cli-{os}-{arch}[.exe]`

默认 `RELEASE_BASE`：

`https://sensecraft-statics.seeed.cc/static/bin/sensecraft-cli`

---

## 3. 首次认证

安装完成后初始化：

```bash
# SenseCAP Access Key（设备 / 分组 / 数据，以及 AI 应用 / 模型 / 云部署）
sensecraft-cli auth init --kind sensecap
```

AI 初始化时选择站点：

| 选项 | 用途 |
|------|------|
| Global | 国际生产 |
| China | 国内账号（AI API host 与 Global 相同） |

查看配置：

```bash
sensecraft-cli auth info
```

无交互环境（CI）不会自动跑 `auth init`，需事先准备好配置文件或自行执行初始化。

---

## 4. 配置文件位置

| 场景 | 路径 |
|------|------|
| **npm / npx 安装（默认）** | macOS/Linux：`~/.config/sensecraft-cli/sensecraft.json`；Windows：`%APPDATA%\sensecraft-cli\sensecraft.json` |
| **源码 `go build` 直接运行** | 当前工作目录下的 `sensecraft.json`（除非设置了环境变量） |

环境变量（优先级从高到低）：

| 变量 | 含义 |
|------|------|
| `SENSECRAFT_CONFIG_FILE` | 配置文件完整路径 |
| `SENSECRAFT_CLI_PATH` | 配置**目录**（文件名固定为 `sensecraft.json`） |

npm 包装器会自动注入 `SENSECRAFT_CLI_PATH` 指向上述默认目录，因此 **npm 装的 CLI 与纯 `go build` 二进制默认读的不是同一个文件**。

### 迁移配置

```bash
# 示例：把 cwd 下的配置迁到 npm 默认位置
mkdir -p ~/.config/sensecraft-cli
cp ./sensecraft.json ~/.config/sensecraft-cli/sensecraft.json
```

Windows：

```powershell
New-Item -ItemType Directory -Force "$env:APPDATA\sensecraft-cli"
Copy-Item .\sensecraft.json "$env:APPDATA\sensecraft-cli\sensecraft.json"
```

---

## 5. 升级

已通过 npm 安装时，直接运行：

```bash
sensecraft-cli update
```

命令会查询 npm 上的最新版本，并执行 `npm install -g @seeed-studio/sensecraft-cli@<版本>`。本地版本号已经相同也会重新安装。安装器接着比较 `vendor/.version` 与新包版本，不一致就重新下载二进制。配置目录里的 Access Key 会保留。`--version` 只打印本地版本号，不访问 npm。

其它用法：

```bash
sensecraft-cli update --check   # 只查询，不安装
npm install -g @seeed-studio/sensecraft-cli@latest
```

平时使用其它命令时，如果缓存里发现了更新，结束时会在 stderr 提示运行 `sensecraft-cli update`。同一版本 24 小时内只提示一次，出现更新的版本会再提示。24 小时内不会重复访问 npm。本地 `dev` 构建、CI、以及设置了 `SENSECRAFT_CLI_NO_UPDATE_NOTIFIER` 时不提示。

进行中的活动也在 stderr 以 `Notice:` 提醒，名称、日期和详情链接来自 `https://api-gateway.seeed.cc/openapi/activity/list`（不带参数，即尚未结束的活动）。24 小时内只提醒一次。交互终端会问 `Ignore this activity? [y/N]`，回答 `y` 之后按 `activity_id` 不再提醒；回答 `n` 时 24 小时后还会再出现。非交互只打印，不提问，也不记成忽略。CI 以及 `SENSECRAFT_CLI_NO_ACTIVITY_NOTIFIER` 时不提醒。

当前程序如果不是 npm 安装的（例如直接运行编译出的二进制），`update` 只打印上面的手动安装命令，不会覆盖正在运行的文件。

验证：

```bash
sensecraft-cli --version
# 应与 npm 包版本一致
```

---

## 6. 卸载

```bash
npm uninstall -g @seeed-studio/sensecraft-cli
```

可选清理：

```bash
# 配置（会删除本地 Token / Access Key，请先备份）
rm -rf ~/.config/sensecraft-cli          # macOS / Linux
# Windows: Remove-Item -Recurse "$env:APPDATA\sensecraft-cli"

# npx 缓存（可选）
npm cache clean --force
```

包目录下的 `vendor/` 二进制会随全局包卸载一起消失。

---

## 7. 离线 / 内网 / 自定义二进制源

### 使用本地已编译的 `dist/`

```bash
./script/build.sh
export SENSECRAFT_CLI_LOCAL_DIST="$PWD/dist"
npx @seeed-studio/sensecraft-cli install
# 或在已 clone 的 packages/cli 下：
# SENSECRAFT_CLI_LOCAL_DIST=$PWD/../../dist npm run postinstall
```

### 覆盖下载基址

默认从 OSS 拉 `{base}/v{version}/{artifact}`。内网镜像时：

```bash
export SENSECRAFT_CLI_RELEASE_BASE="https://your-mirror.example.com/sensecraft-cli"
npx @seeed-studio/sensecraft-cli install
```

镜像目录结构须与发版一致，例如：

```text
…/sensecraft-cli/v0.1.3/sensecraft-cli-darwin-arm64
…/sensecraft-cli/v0.1.3/sensecraft-cli-linux-amd64
…
```

### 跳过下载（仅复用已有 vendor）

```bash
export SENSECRAFT_CLI_SKIP_DOWNLOAD=1
# 若 vendor 中没有匹配当前版本的二进制，安装会失败
```

### 可选校验和

若发版方提供了 SHA256，可设置：

```bash
export SENSECRAFT_CLI_SHA256="<hex>"
```

与下载内容不一致时安装失败。

### 代理

遵循 Node / 系统代理环境变量，例如：

```bash
export HTTPS_PROXY=http://proxy.example.com:8080
export HTTP_PROXY=http://proxy.example.com:8080
```

---

## 8. 从源码构建

```bash
git clone <本仓库 URL>
cd sensecraft-cloud-terminal

# 本机调试
go build -o sensecraft-cli .
./sensecraft-cli --version

# 全平台发布产物 → dist/
./script/build.sh
```

本地用源码二进制时，配置默认落在**当前目录** `sensecraft.json`。若希望与 npm 安装共用配置：

```bash
export SENSECRAFT_CLI_PATH="$HOME/.config/sensecraft-cli"
./sensecraft-cli auth info
```

开发自检：

```bash
./script/smoke-ai.sh
./script/smoke-sensecap.sh
cd packages/cli && SENSECRAFT_CLI_LOCAL_DIST=$PWD/../../dist npm run test:install
```

发版流程见 [`README.md`](README.md)。

---

## 9. Agent Skills（Cursor 等）

Skills 是给 Agent 读的 Markdown 能力说明，**不是** CLI 子命令。

### 安装来源

| 来源 | 路径 |
|------|------|
| 本仓库 | `skills/` |
| npm 包安装后 | 全局包目录下的 `skills/`（与 `bin`、`scripts` 同级） |

查找全局包目录：

```bash
npm root -g
# 例如 …/node_modules/@seeed-studio/sensecraft-cli/skills/sensecraft-ai/SKILL.md
```

### 推荐布局

| Skill | 路径 | 鉴权 |
|-------|------|------|
| Meta 路由（优先加载） | `skills/sensecraft-ai/SKILL.md` | — |
| SenseCAP PaaS | `skills/sensecraft-data/` | SenseCAP |
| 应用广场 | `skills/sensecraft-apps/` | Access Key |
| 文件上传 | `skills/sensecraft-files/` | Access Key |
| 模型库 | `skills/sensecraft-models/` | Access Key |
| 经典 reCamera 40 | `skills/sensecraft-recamera/` | Access Key |
| 云设备 | `skills/sensecraft-devices-cloud/` | Access Key |

更细的发现说明：[`skills/README.md`](skills/README.md)。

### 接到 Cursor

任选其一：

1. **工作区即本仓库**  
   打开 `sensecraft-cloud-terminal` 作为 Cursor 项目，Agent 可直接读 `skills/`。

2. **拷贝 / 软链到 Cursor skills 目录**（路径以你本机 Cursor 版本为准，常见为用户目录下的 skills 配置）：

   ```bash
   # 示例：软链 meta + 各域 skill（按需调整目标目录）
   ln -s "$PWD/skills/sensecraft-ai"   ~/.cursor/skills/sensecraft-ai
   ln -s "$PWD/skills/sensecraft-apps" ~/.cursor/skills/sensecraft-apps
   # …
   ```

3. **只用 npm 包里的 skills**  
   把 `$(npm root -g)/@seeed-studio/sensecraft-cli/skills` 链到 Agent 可发现的路径。

### 接到 Claude Code / 其他 Agent

这些 skill 只依赖标准 `SKILL.md`（YAML `name`/`description` + Markdown 正文），不包含 Cursor 私有指令。支持 Agent Skills 的宿主可以直接复制或软链整个 skill 目录：

```bash
# Claude Code：项目级
mkdir -p .claude/skills
for d in "$PWD"/skills/*; do
  ln -s "$d" ".claude/skills/$(basename "$d")"
done

# Claude Code：用户级（可选）
mkdir -p ~/.claude/skills
ln -s "$PWD/skills/sensecraft-ai" ~/.claude/skills/sensecraft-ai
```

其他 Agent：

1. 若支持 `SKILL.md` 自动发现，把 `skills/*/` 原样放入其项目级或用户级 skills 目录；具体目录以宿主文档为准。
2. 若不支持 Skills，把 `skills/sensecraft-ai/SKILL.md` 和命中的领域 skill 作为系统/项目上下文加载。
3. Agent 必须具有本机命令执行能力，才能调用 `sensecraft-cli`；纯聊天宿主只能读取说明，不能声称命令已执行。
4. `auth init`、本地 USB/IP 部署与烧录仍需用户交互，不能因为换了 Agent 就降级安全级别。

使用约定：

- SenseCraft AI 相关问题先走 **`sensecraft-ai`** 路由 skill。  
- 确保机器上 `sensecraft-cli` 在 PATH 中，且已 `auth init`。

---

## 10. 验证清单

```bash
sensecraft-cli --version
sensecraft-cli auth info

# 不消耗真实配额的冒烟（需已有对应 profile 或临时 config）
sensecraft-cli model constants --sandbox
sensecraft-cli app list --mine --sandbox
sensecraft-cli device list --sandbox
sensecraft-cli group list --sandbox
sensecraft-cli data latest --eui SANDBOXEUI000001 --sandbox
```

---

## 11. 常见问题

### `sensecraft-cli: command not found`

- 确认已执行 `npx @seeed-studio/sensecraft-cli install` 或 `npm install -g …`  
- 把 `$(npm prefix -g)/bin`（Windows 为 `npm prefix -g`）加入 PATH  
- 新开一个终端再试  

### 二进制下载失败

安装器会提示可尝试：

1. 确认发版产物已上传到 `{RELEASE_BASE}/v{version}/`  
2. `SENSECRAFT_CLI_LOCAL_DIST` 指向本地 `dist/`  
3. `SENSECRAFT_CLI_RELEASE_BASE` 指向可用镜像  

公司网络拦截 OSS 时，优先用本地 `dist/` 或内网镜像。

### `auth info` 显示未初始化 / 装完又提示 init

- npm 安装认 **SenseCAP Access Key**（`app_config_list`）。  
- 确认读写的是同一配置文件：`echo $SENSECRAFT_CLI_PATH` / `%SENSECRAFT_CLI_PATH%`。  
- 应用 / 模型 / 云设备：`sensecraft-cli auth init --kind sensecap`。  

### npm 装的和 `go build` 的配置不一致

见上文「配置文件位置」：前者默认 `~/.config/…`，后者默认 cwd。用 `SENSECRAFT_CLI_PATH` 或 `SENSECRAFT_CONFIG_FILE` 统一。

### Windows 杀毒软件拦截

首次下载的 `sensecraft-cli.exe` 可能被隔离。将安装目录或 `vendor` 加入白名单后重装。

### 升级后版本号不变

```bash
npm uninstall -g @seeed-studio/sensecraft-cli
npm cache clean --force
npx @seeed-studio/sensecraft-cli@<目标版本> install
sensecraft-cli --version
```

---

## 12. 相关文档

| 文档 | 内容 |
|------|------|
| [`README.md`](README.md) | 功能概览、命令参考、发版说明 |
| [`ARCHITECTURE.md`](ARCHITECTURE.md) | 架构分层、鉴权、业务规则、安全模型、CI |
| [`docs/sensecraft-cli-smoke.md`](docs/sensecraft-cli-smoke.md) | 发版冒烟清单 |
| [`docs/sensecraft-cli-test-cases.md`](docs/sensecraft-cli-test-cases.md) | 全量功能测试用例 |
| [`skills/README.md`](skills/README.md) | Skills 目录与 Agent 约定 |
| [`LICENSE`](LICENSE) | MIT |
