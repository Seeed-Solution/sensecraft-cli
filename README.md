# SenseCraft CLI

SenseCraft CLI（`sensecraft-cli`）是 [SenseCAP](https://sensecap.seeed.cc) 与 [SenseCraft AI](https://sensecraft.seeed.cc) 的命令行工具：

| 能力面 | 鉴权 | 子命令 |
|--------|------|--------|
| SenseCAP 设备 / 分组 / 传感数据 | Access Key（`--kind sensecap`） | `device` · `group` · `data` |
| SenseCraft AI 应用 / 模型 / 云设备 | Access Key → `{ai-host}/openapi` | `app` · `model` · `file` · `cloud` · `recamera` |

适用于脚本自动化、CI/CD 与 AI Agent（Skills 见 `skills/`）。

**安装与排障请先读 → [`INSTALL.md`](INSTALL.md)**（平台要求、PATH、升级卸载、离线镜像、Cursor Skills）。  
**架构与实现细节 → [`ARCHITECTURE.md`](ARCHITECTURE.md)**（分层、鉴权、网络、业务规则、L1–L4、Skills、CI）。  
**发版冒烟 → [`docs/sensecraft-cli-smoke.md`](docs/sensecraft-cli-smoke.md)** · **全量测试用例 → [`docs/sensecraft-cli-test-cases.md`](docs/sensecraft-cli-test-cases.md)**。

## 环境要求

- Node.js >= 16（用于 npm 安装与分发）
- 构建源码：Go（见 `go.mod`）
- 可访问对应站点（SenseCAP OpenAPI / SenseCraft `{ai-host}/openapi`）

## 安装

```bash
npx @seeed-studio/sensecraft-cli install
sensecraft-cli --version
sensecraft-cli update
```

完整步骤、Windows PATH、离线安装与 Skills 接入见 [`INSTALL.md`](INSTALL.md)。

```bash
sensecraft-cli --help
# 或
npx @seeed-studio/sensecraft-cli device list
```

## 快速开始

### 1. 初始化认证

```bash
# SenseCAP OpenAPI Access Key（设备/数据，以及 AI 应用/模型/云部署）
sensecraft-cli auth init --kind sensecap
# 交互选择站点：Global / China
```

Access Key 用于 SenseCAP `/openapi` 与 SenseCraft `{ai-host}/openapi`（HTTP Basic）。

| `remote_type` | SenseCraft AI `/openapi` | 用途 |
|---------------|--------------------------|------|
| `Global` | `api-gateway.seeed.cc/openapi` | 国际生产 |
| `China` | `api-gateway.seeed.cc/openapi`（与 Global 同 AI host；账号走 CN） | 国内生产 |

网页控制台仍是 `https://sensecraft.seeed.cc`。命令成功后，stderr 上可能各出现一条英文提示，同一条 24 小时内只出现一次：`Warning:` 提示新版本并建议运行 `sensecraft-cli update`；`Notice:` 写出活动名称、日期和详情链接。交互终端里会再问 `Ignore this activity? [y/N]`，回答 `y` 之后这条活动不再提醒。`SENSECRAFT_CLI_NO_UPDATE_NOTIFIER` 和 `SENSECRAFT_CLI_NO_ACTIVITY_NOTIFIER` 分别关掉这两条。

### 2. 查看配置

```bash
sensecraft-cli auth info
```

### 3. 示例

```bash
# SenseCAP
sensecraft-cli device list
sensecraft-cli data latest --eui <EUI>

# SenseCraft AI
sensecraft-cli app list --public --name "vision"
sensecraft-cli app list --public --uniform-type 40
sensecraft-cli model list --public --task 1
sensecraft-cli model list --public --uniform-type 40
sensecraft-cli recamera check --model-id <id>
sensecraft-cli recamera open --model-id <id> --print-only
sensecraft-cli cloud device list --uniform-type 26
```

L2 写操作（publish / delete / deploy 等）须**另开一轮**向用户确认后再加 `--yes`。用户第一句（如「删这个63760」）不是确认。协议见 [`skills/confirmation.md`](skills/confirmation.md)。

## 配置

| 项 | 说明 |
|----|------|
| npm 安装默认 | macOS/Linux：`~/.config/sensecraft-cli/sensecraft.json`；Windows：`%APPDATA%\sensecraft-cli\sensecraft.json`（wrapper 注入 `SENSECRAFT_CLI_PATH`） |
| 源码直接运行默认 | 当前工作目录下的 `sensecraft.json` |
| `SENSECRAFT_CLI_PATH` | 配置目录（文件名为 `sensecraft.json`） |
| `SENSECRAFT_CONFIG_FILE` | 配置文件完整路径（优先于上面两项） |
| `SENSECRAFT_DATA_HOST` | SenseCAP OpenAPI 站点根。空则按 Global / China 走生产 |
| `SENSECRAFT_AI_HOST` | SenseCraft AI 站点根。空则为 `https://api-gateway.seeed.cc`。控制台页面不跟着改 |
| `SENSECRAFT_AUTH_HOST` | `auth init` 的账号中心站点根。空则 Global 为 `account.seeed.cc`、China 为 `account.seeed.cn`。末尾 `/authapi` 会去掉 |

## 输出格式

- 业务命令成功：stdout JSON（缩进 2 空格）
- `auth info` / `auth init`：人类可读文本（非 JSON）
- 失败：stderr + 非 0 退出码
- 版本：`sensecraft-cli --version`（与 npm 包版本一致）

## AI Agent Skills

| Skill | 路径 |
|-------|------|
| Meta 路由 | `skills/sensecraft-ai/SKILL.md` |
| SenseCAP PaaS | `skills/sensecraft-data/` |
| 应用广场 | `skills/sensecraft-apps/` |
| 模型库 | `skills/sensecraft-models/` |
| 经典 reCamera 40 | `skills/sensecraft-recamera/` |
| 云设备 | `skills/sensecraft-devices-cloud/` |

发现说明：[`skills/README.md`](skills/README.md)。

## 与 Console 的关系

CLI 调用 SenseCraft AI 的 **`{ai-host}/openapi`**（Access Key HTTP Basic，无 appid）。  
**不做**：Cookie 登录、本地 Web Serial 伪远程烧录、Admin 审核。

## 从源码构建

完整说明见 [`INSTALL.md` §8](INSTALL.md#8-从源码构建)。摘要：

```bash
# 本机调试
go build -o sensecraft-cli .
./sensecraft-cli --version

# 多平台发布二进制 → dist/（注入 -ldflags 版本信息）
./script/build.sh

# AI / SenseCAP 命令 sandbox 冒烟（无需真实 Token）
./script/smoke-ai.sh
./script/smoke-sensecap.sh

# 隔离安装冒烟（临时 HOME，不碰真实 ~/.config / 全局 npm）
# 需要先 ./script/build.sh
cd packages/cli && npm run test:install
```

发版勾选与联调抽查见 [`docs/sensecraft-cli-smoke.md`](docs/sensecraft-cli-smoke.md)；功能细节用例见 [`docs/sensecraft-cli-test-cases.md`](docs/sensecraft-cli-test-cases.md)。

发布：在 GitHub 上为 `sensecraft-cloud-terminal` 打与 `packages/cli/package.json` 一致的 tag（如 `v0.1.4`），交叉编译后上传 Release 资产，再发布 npm。npm 发布需要 `NPM_TOKEN`。

## 许可证

[MIT](LICENSE)

---

## 命令参考（SenseCAP）

以下为 SenseCAP OpenAPI 命令摘要；完整细节见 `skills/sensecraft-data/commands.md`。
SenseCraft AI 命令详见各 `skills/*/commands.md`。

### auth — 认证与配置

#### `auth init`

| 参数 | 说明 |
|------|------|
| `--kind sensecap` | 鉴权种类，可省略 |
| `--sandbox` | 沙盒：不发真实网络请求 |

#### `auth info`

`info` 展示当前 Access Key 的站点、组织 ID 和 Key ID，不打印密钥。

---

### device — 设备管理（SenseCAP）

#### `device list`

获取当前账号下的设备列表。

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `--device-type` | 否 | `2` | 设备类型：`1`=网关，`2`=节点，`3`=开发套件 |
| `--group-uuid` | 否 | — | 按分组 UUID 筛选；不传则返回所有分组下的设备 |

```bash
# 列出所有节点（默认）
sensecraft-cli device list

# 列出网关
sensecraft-cli device list --device-type 1

# 列出开发套件
sensecraft-cli device list --device-type 3

# 列出指定分组下的设备
sensecraft-cli device list --group-uuid <group-uuid>
```

#### `device view`

获取设备详情（最多 50 个 EUI）。

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `--eui` | 是 | — | 设备 EUI，可多次指定 |
| `--device-type` | 否 | `2` | 设备类型：`1`=网关，`2`=节点，`3`=开发套件 |

```bash
sensecraft-cli device view --eui 0123456789ABCDEF
```

#### `device channels`

查询节点通道列表（最多 50 个 EUI）。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--eui` | 是 | 设备 EUI，可多次指定 |

```bash
sensecraft-cli device channels --eui 0123456789ABCDEF
```

#### `device status`

获取设备运行状态（最多 50 个 EUI）。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--eui` | 是 | 设备 EUI，可多次指定 |

```bash
sensecraft-cli device status --eui 0123456789ABCDEF
```

#### `device bind`

将设备绑定到当前账号。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--eui` | 是 | 设备 EUI |
| `--code` | 是 | 设备绑定码 |
| `--name` | 否 | 设备名称 |
| `--group-uuid` | 否 | 目标分组 UUID |
| `--longitude` | 否 | 经度 |
| `--latitude` | 否 | 纬度 |

```bash
sensecraft-cli device bind --eui 0123456789ABCDEF --code ABCD1234 --name "我的传感器"
```

#### `device unbind`

解绑设备（最多 50 个 EUI）。**L2**：须用户确认后传 `--yes`。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--eui` | 是 | 设备 EUI，可多次指定 |
| `--yes` | 是（确认后） | L2 确认 |

```bash
sensecraft-cli device unbind --eui 0123456789ABCDEF --yes
```

#### `device sim`

查看 SIM 卡状态。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--iccid` | 是 | SIM 卡 ICCID |

```bash
sensecraft-cli device sim --iccid 89860000000000000000
```

---

### device devkit — 开发套件

#### `device devkit create`

创建开发套件设备。同一组织下每个 SKU 最多创建 5 个。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--sku` | 是 | 开发套件 SKU |
| `--name` | 否 | 设备名称 |

支持的 SKU：

| SKU | 说明 |
|-----|------|
| `blank_device` | Blank Device |
| `xiao` | XIAO |
| `grove_vision_ai` | Grove Vision AI V2 |

```bash
sensecraft-cli device devkit create --sku xiao --name "我的 XIAO 套件"
```

#### `device devkit key`

查看开发套件设备的 `device_key` 与 `token`（仅支持开发套件设备，EUI 为 16 位字符）。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--eui` | 是 | 开发套件设备 EUI（16 位） |

```bash
sensecraft-cli device devkit key --eui 0123456789ABCDEF
```

---

### group — 设备分组

#### `group list`

列出当前账号下的所有设备分组。

#### `group create`

创建设备分组（最多 50 个分组）。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--name` | 是 | 分组名称 |

```bash
sensecraft-cli group create --name "温室传感器"
```

#### `group update`

更新设备分组名称。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--group-uuid` | 是 | 分组 UUID |
| `--name` | 是 | 新分组名称 |

```bash
sensecraft-cli group update --group-uuid <uuid> --name "新名称"
```

#### `group delete`

删除设备分组（组内设备将移至默认分组）。**L2**：须用户确认后传 `--yes`。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--group-uuid` | 是 | 分组 UUID |
| `--yes` | 是（确认后） | L2 确认 |

```bash
sensecraft-cli group delete --group-uuid <uuid> --yes
```

#### `group move-devices`

将设备移动到指定分组。**L2**：须用户确认后传 `--yes`。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--group-uuid` | 是 | 目标分组 UUID |
| `--eui` | 是 | 设备 EUI，可多次指定 |
| `--yes` | 是（确认后） | L2 确认 |

```bash
sensecraft-cli group move-devices --group-uuid <uuid> --eui 0123456789ABCDEF --yes
```

---

### data — 传感数据

#### `data latest`

获取设备最近一年内的最新遥测数据；未指定通道时返回每个通道下最新数据点。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--eui` | 是 | 设备 EUI |
| `--measurement-id` | 否 | 测量值 ID |
| `--channel-index` | 否 | 通道编号 |

```bash
sensecraft-cli data latest --eui 0123456789ABCDEF --measurement-id 4097 --channel-index 1
```

#### `data list`

获取设备历史遥测数据（单次最长约一个月，且仅可查近三个月内数据）。

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `--eui` | 是 | — | 设备 EUI |
| `--measurement-id` | 否 | — | 测量值 ID |
| `--channel-index` | 否 | — | 通道编号 |
| `--limit` | 否 | — | 返回记录数 |
| `--time-start` | 否 | — | 起始时间戳（毫秒） |
| `--time-end` | 否 | — | 结束时间戳（毫秒） |

```bash
sensecraft-cli data list --eui 0123456789ABCDEF --limit 100 --time-start 1700000000000 --time-end 1700086400000
```

#### `data chart`

获取设备遥测折线聚合点（最长一年；每个测量最多约 250 个点）。

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `--eui` | 是 | — | 设备 EUI |
| `--measurement-id` | 否 | — | 测量值 ID |
| `--channel-index` | 否 | — | 通道编号 |
| `--interval` | 否 | `60` | 时间段长度（分钟） |
| `--time-start` | 否 | — | 起始时间戳（毫秒） |
| `--time-end` | 否 | — | 结束时间戳（毫秒） |

```bash
sensecraft-cli data chart --eui 0123456789ABCDEF --interval 60 --time-start 1700000000000 --time-end 1702592000000
```

---

## 典型工作流

```bash
# 安装并初始化
npx @seeed-studio/sensecraft-cli install

# 查看分组
sensecraft-cli group list

# 列出节点设备
sensecraft-cli device list

# 查看设备详情与通道
sensecraft-cli device view --eui 0123456789ABCDEF
sensecraft-cli device channels --eui 0123456789ABCDEF

# 查询最新数据
sensecraft-cli data latest --eui 0123456789ABCDEF
```
