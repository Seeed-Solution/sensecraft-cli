# SenseCraft CLI

[English](README.md) | 中文

SenseCraft CLI（`sensecraft-cli`）是 [SenseCraft](https://sensecraft.seeed.cc) 的官方命令行工具，供开发者和 AI Agent 使用。可以在终端里读取传感数据、创建模型、发布应用，并管理设备和 reCamera。

它覆盖这三块产品：

| 产品 | 管什么 | 子命令 |
|------|--------|--------|
| [SenseCraft Data](https://sensecap.seeed.cc) | 设备、分组、传感数据 | `device` · `group` · `data` |
| [SenseCraft AI](https://sensecraft.seeed.cc) | 模型库，以及 reCamera | `model` · `recamera` · `cloud` |
| [CodeCraft Hub](https://codecraft.seeed.cc/hub) | 应用 | `app` |

三个平台均以开放 Access Key 访问。在 [SenseCraft 用户中心](https://account.seeed.cc/account/access-keys) 创建和管理钥匙，再用 `auth add` 写入本机。运行 `auth init`，使用 OAuth2 Device Flow 授权 CLI 访问你的账号。

也可以放进脚本和 CI。Agent Skills 见 `skills/`。

**安装与排障请先读 → [`INSTALL.md`](INSTALL.md)**（平台要求、PATH、升级卸载、离线安装、Cursor Skills）。

## 环境要求

- Node.js >= 16（用于 npm 安装与分发）
- 构建源码：Go（见 `go.mod`）

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
# 使用 OAuth2 Device Flow 授权 CLI 访问你的账号。交互选择站点：Global / China
sensecraft-cli auth init

# 已在账号页创建好钥匙时，直接写入本机
sensecraft-cli auth add --key-id <Key ID> --access-key <Access Key>
```

创建和管理 Access Key：<https://account.seeed.cc/account/access-keys>。

中国站仅支持 SenseCraft Data。如果需要使用 SenseCraft CLI 访问 SenseCraft AI 和 CodeCraft Hub，请选择国际站（Global）。

### 2. 查看配置

```bash
sensecraft-cli auth info
```

### 3. 示例

```bash
# SenseCraft Data
sensecraft-cli device list
sensecraft-cli data latest --eui <EUI>

# SenseCraft AI。任务编号用 model constants 查，不要猜一个数字
sensecraft-cli model constants
sensecraft-cli model list --public --uniform-type 40
sensecraft-cli recamera check --model-id <id>
sensecraft-cli recamera open --model-id <id> --print-only
sensecraft-cli cloud device list --uniform-type 26

# CodeCraft Hub
sensecraft-cli app list --public --name "vision"
sensecraft-cli app list --public --uniform-type 40
```

`--uniform-type`：`40` 是 reCamera，`32` 是 XIAO ESP32S3，`53` 是 reCamera Pro。`cloud device list` 里的 `26` 是 Jetson 云设备界面常用的设备类型。

删除、解绑、发布、部署这类写操作必须加 `--yes`。Agent 的确认步骤见 [skills/sensecraft-cli/references/confirmation.md](skills/sensecraft-cli/references/confirmation.md)。

## 配置

| 项 | 说明 |
|----|------|
| npm 安装默认 | macOS/Linux：`~/.config/sensecraft-cli/sensecraft.json`；Windows：`%APPDATA%\sensecraft-cli\sensecraft.json`（wrapper 注入 `SENSECRAFT_CLI_PATH`） |
| 源码直接运行默认 | 当前工作目录下的 `sensecraft.json` |
| `SENSECRAFT_CLI_PATH` | 配置目录（文件名为 `sensecraft.json`） |
| `SENSECRAFT_CONFIG_FILE` | 配置文件完整路径（优先于上面两项） |

站点不会根据主机名里是否含有 `test-` 切换。

## 输出格式

- 业务命令成功：stdout JSON（缩进 2 空格）
- `auth info` / `auth init`：人类可读文本（非 JSON）
- 失败：stderr + 非 0 退出码
- 版本：`sensecraft-cli --version`（与 npm 包版本一致）

## 用在脚本和 CI

无交互环境不会打开浏览器。用 `auth add` 写入钥匙，并用 `SENSECRAFT_CONFIG_FILE` 指到项目自己的配置文件，避免写进本机的 `~/.config/sensecraft-cli/`。

```bash
export SENSECRAFT_CONFIG_FILE="$PWD/.sensecraft/sensecraft.json"
sensecraft-cli auth add \
  --key-id "$SENSECRAFT_KEY_ID" \
  --access-key "$SENSECRAFT_ACCESS_KEY" \
  --site Global
sensecraft-cli device list
```

这个文件含有 Access Key，不要提交到仓库。`--site` 省略时是 `Global`。业务命令成功时解析 stdout 的 JSON；失败看退出码，错误在 stderr。`auth info` 的输出是文本。

## AI Agent Skills

入口是 [`skills/sensecraft-cli/SKILL.md`](skills/sensecraft-cli/SKILL.md)。设备、模型、应用等说明在 `skills/sensecraft-cli/references/`。从本地工程生成应用用嵌套技能 `skills/sensecraft-cli/create-app/`。

安装后把这个目录软链到 Agent 的技能目录，不要复制。升级 CLI 时技能会一起更新。步骤见 [`INSTALL.md` §9](INSTALL.md#9-agent-skills-cursor-and-others)。

## 从源码构建

完整说明见 [`INSTALL.md` §8](INSTALL.md#8-build-from-source)。摘要：

```bash
# 本机调试
go build -o sensecraft-cli .
./sensecraft-cli --version

# 多平台发布二进制 → dist/（注入 -ldflags 版本信息）
./script/build.sh
```

贡献本仓库时的自检：`./script/smoke-ai.sh`、`./script/smoke-sensecap.sh`。隔离安装检查要先 `./script/build.sh`，再在 `packages/cli` 里运行 `npm run test:install`。使用 CLI 的项目不需要这些脚本。

## 许可证

[MIT](LICENSE)

---

## 命令文档

本文展开 SenseCraft Data。其它命令的参数、限制和示例：

| 范围 | 文档 |
|------|------|
| SenseCraft Data | [data-commands.md](skills/sensecraft-cli/references/data-commands.md) |
| 模型库 | [model-commands.md](skills/sensecraft-cli/references/model-commands.md) |
| reCamera | [recamera-commands.md](skills/sensecraft-cli/references/recamera-commands.md) |
| 云设备部署与订单 | [cloud-commands.md](skills/sensecraft-cli/references/cloud-commands.md) |
| CodeCraft Hub 应用 | [app-commands.md](skills/sensecraft-cli/references/app-commands.md) |
| 文件上传 | [file-commands.md](skills/sensecraft-cli/references/file-commands.md) |

## 命令参考（SenseCraft Data）

### auth — 认证与配置

#### `auth init`

使用 OAuth2 Device Flow 授权 CLI 访问你的账号。交互选择站点：Global / China。

| 参数 | 说明 |
|------|------|
| `--sandbox` | 沙盒：不发真实网络请求 |

#### `auth add`

把已有的 Access Key 写入本机，并设为当前钥匙。无交互环境必须带上 `--key-id` 和 `--access-key`。

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `--key-id` | 交互终端可省略 | — | Key ID |
| `--access-key` | 交互终端可省略 | — | Access Key |
| `--site` | 否 | `Global` | `Global` 或 `China` |

```bash
sensecraft-cli auth add --key-id <Key ID> --access-key <Access Key> --site Global
```

#### `auth info`

`info` 展示当前 Access Key 的站点、组织 ID 和 Key ID，不打印密钥。

---

### device — 设备管理（SenseCraft Data）

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

解绑设备（最多 50 个 EUI）。须加 `--yes`。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--eui` | 是 | 设备 EUI，可多次指定 |
| `--yes` | 是 | 确认执行 |

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

删除设备分组（组内设备将移至默认分组）。须加 `--yes`。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--group-uuid` | 是 | 分组 UUID |
| `--yes` | 是 | 确认执行 |

```bash
sensecraft-cli group delete --group-uuid <uuid> --yes
```

#### `group move-devices`

将设备移动到指定分组。须加 `--yes`。

| 参数 | 必填 | 说明 |
|------|------|------|
| `--group-uuid` | 是 | 目标分组 UUID |
| `--eui` | 是 | 设备 EUI，可多次指定 |
| `--yes` | 是 | 确认执行 |

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
