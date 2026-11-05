# sensecraft-cli

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25-blue.svg)](https://go.dev/)
[![npm version](https://img.shields.io/npm/v/@seeed-studio/sensecraft-cli.svg)](https://www.npmjs.com/package/@seeed-studio/sensecraft-cli)

[中文](./README.zh.md) | [English](./README.md)

[SenseCAP](https://sensecap.seeed.cc) 设备与 [SenseCraft AI](https://sensecraft.seeed.cc) 的命令行工具。一把 OpenAPI Access Key 即可访问设备、遥测、应用、模型、文件和云设备。给人用，也给 AI Agent 用。Skills 在 [`skills/`](./skills/)。

[安装](#安装) · [AI Agent](#给-ai-agent) · [Skills](#agent-skills) · [登录](#登录) · [命令](#命令) · [进阶](#进阶) · [安全](#安全)

## 能做什么

| 类别 | 能力 |
|------|------|
| 设备 | 列出、查看、绑定、解绑网关、节点和开发套件 |
| 分组 | 创建分组、移动设备 |
| 遥测 | 最新值、历史记录、折线聚合点 |
| 应用 | CodeCraft Hub：浏览应用，管理自己的应用、物料和活动 |
| 模型 | 搜索模型库，上传和发布模型 |
| 文件 | 上传本地文件，得到 `file_url` |
| 云设备 | 列出 Jetson 等云设备，部署模型 |
| reCamera | 检查经典 reCamera 的应用或模型，并打开控制台部署页 |

网页控制台是 [https://sensecraft.seeed.cc](https://sensecraft.seeed.cc)。

## 安装

需要 Node.js 16 及以上，以及能访问 SenseCAP 和 SenseCraft AI 的网络。从源码构建另需 Go 1.25 及以上（见 `go.mod`）。

```bash
npx @seeed-studio/sensecraft-cli install
sensecraft-cli --version
```

Windows 上的命令名是 `sensecraft-cli.cmd`。PATH、升级和卸载见 [`INSTALL.md`](INSTALL.md)。

### 三步上手

在 SenseCAP 门户创建 OpenAPI Access Key，然后：

```bash
# 1. 写入钥匙。--site 为 Global 或 China，省略时是 Global
sensecraft-cli auth add --key-id <Key ID> --access-key <Access Key> --site Global

# 2. 确认。不会打印 Access Key
sensecraft-cli auth info

# 3. 调用接口
sensecraft-cli device list
```

交互终端里可以不带参数运行 `auth add`，按提示填写。

## 给 AI Agent

和用户一起完成。不要让用户把 Access Key 贴进对话，也不要在会被记录的命令行里带上这串密钥。

1. 如果没有命令，先安装：

```bash
npx @seeed-studio/sensecraft-cli install
```

2. 请用户在自己的电脑上运行 `auth add`。
3. 确认钥匙已经生效：

```bash
sensecraft-cli auth info
```

4. 再跑只读命令，例如 `sensecraft-cli device list` 或 `sensecraft-cli app list --public`。
5. 先读 [`skills/sensecraft-cli/SKILL.md`](skills/sensecraft-cli/SKILL.md)，它会转到对应的产品说明。

解绑、删除、发布、部署这类写操作，要等用户单独确认之后才能加 `--yes`。

## Agent Skills

| Skill | 说明 |
|-------|------|
| [`sensecraft-cli`](skills/sensecraft-cli/SKILL.md) | SenseCraft Data、SenseCraft AI 和 CodeCraft Hub |
| [`create-app`](skills/sensecraft-cli/create-app/SKILL.md) | 把本地项目做成 CodeCraft Hub 应用 |

全局 npm 安装之后：

```bash
ls "$(npm root -g)/@seeed-studio/sensecraft-cli/skills"
```

## 登录

| 命令 | 说明 |
|------|------|
| `auth add` | 校验 Access Key，写入本机，并设为当前钥匙 |
| `auth info` | 显示本地认证摘要；加 `--json` 输出结构化数据 |

```bash
sensecraft-cli auth add --key-id <Key ID> --access-key <Access Key> --site Global
sensecraft-cli auth info
sensecraft-cli auth info --json
```

`auth info --json` 返回 `{ "profiles": [...] }`，每项仅含 `site`、`org_id`、`key_id`、`kind`、`created_at` 和 `active`。

| `--site` | SenseCAP OpenAPI | SenseCraft AI OpenAPI |
|----------|------------------|------------------------|
| `Global` | `https://sensecap.seeed.cc/openapi` | `https://api-gateway.seeed.cc/openapi` |
| `China` | `https://sensecap.seeed.cn/openapi` | `https://api-gateway.seeed.cc/openapi` |

中国站仅支持 SenseCraft Data。SenseCraft AI 和 CodeCraft Hub 请用 Global。同一把 Access Key 用 HTTP Basic 调用接口：用户名是 Key ID，密码是 Access Key。

## 命令

参数用 `sensecraft-cli <command> --help` 查看。每一类一条示例：

```bash
sensecraft-cli device list --device-type 1
sensecraft-cli data latest --eui <EUI>

sensecraft-cli app list --public --sort popular
sensecraft-cli model list --public
sensecraft-cli file upload --type 1 --path ./cover.png --for-app
sensecraft-cli cloud device list
sensecraft-cli recamera check --model-id <id>
```

| 命令 | 说明 |
|------|------|
| `device` | 网关（`--device-type 1`）、节点（`2`，默认）、开发套件（`3`） |
| `group` | 设备分组 |
| `data` | 最新遥测、历史、折线点 |
| `app` | CodeCraft Hub 和我的应用。不传 `--public` 时列出自己的应用 |
| `model` | 模型库。不传 `--mine` 时列出公开模型 |
| `file` | 上传。`--type`：`1` 图片，`2` 模型，`3` 应用，`4` 固件，`5` 静态资源（需要 `--flow-id`），`6` 压缩包 |
| `cloud` | 云设备，以及部署模型 |
| `recamera` | 检查，或打开控制台页面。烧录在网页上完成 |
| `update` | 安装 npm 上的最新版本 |

`app list --sort` 取 `popular`、`newest` 或 `likes`。常用 `--uniform-type`：`40` reCamera，`32` XIAO ESP32S3，`53` reCamera Pro。

`app create` 会先记录自己的精确同名应用。创建请求超时后，只有恰好新增一个同名应用时才继续保存，并返回 `recovered_after_timeout: true`、应用 ID 和常规链接；没有新增则返回原错误，多个候选则停止并列出 ID。结果不确定时，用 `app list --mine --name "<名称>"` 核对。客户端超时保持为 15 秒。

从未发布的应用，用 `app view --id <id>` 只能读到名称；读取和核对已保存草稿请用 `app view --id <id> --sketch`。执行 CLI 更新前，先关闭该应用的 CodeCraft Hub 编辑页，验证后再重新打开：打开的编辑页会保存它加载时的副本，可能覆盖期间通过 CLI 做的修改。

`app bom search --name "XIAO ESP32-S3 1.47"` 会逐词匹配产品名称、SKU、分类、备注或描述，忽略大小写和标点；`1.47 Touch`、`Touch Display ESP32-S3` 和完整 SKU 也能匹配。`ESP32S3` 与 `ESP32-S3` 等价，结果保持目录原顺序。目录保留原始 `block_support`，同时增加布尔值 `block_supported`（`"是"` 对应 `true`）。

`file upload --name <文件名>` 可指定存储名称；省略时沿用本地文件名去空格的规则。类型 `4` 会在 stderr 提醒：同一组织内，同名上传会替换原文件内容，URL 保持相同。Muse Gadget 应上传构建产物 `muse-gadget.bin`，它是链接在 `0x20000` 的**应用镜像**，并为每个应用指定独立的存储名称：

```bash
sensecraft-cli file upload --path ./build/muse-gadget.bin --type 4 --name clock-muse-gadget.bin
```

重命名后的应用镜像已在硬件上验证可从 Hub 草稿页烧录并启动。将各自返回的 URL 写入对应应用的 `firmware_url`；连续创建多个应用后，用 `--sketch` 读取草稿，确认这些 URL 互不相同。

## 进阶

成功的数据命令把缩进 JSON 打到 stdout。含 `list` 的对象会在缺少 `count` 时补上当前列表长度；`page` 等原字段及服务器已有的 `count` 保持原值。`auth add` 和默认的 `auth info` 打可读文本，`auth info --json` 输出认证摘要。失败在 stderr，退出码非 0。

`sensecraft-cli --version` 打印 npm 包版本。直接 `go build` 的二进制打印 `dev`。

配置文件名是 `sensecraft.json`。

| 运行方式 | 文件位置 |
|----------|----------|
| npm 安装 | macOS / Linux：`~/.config/sensecraft-cli/sensecraft.json`。Windows：`%APPDATA%\sensecraft-cli\sensecraft.json` |
| `go build` 的二进制 | 当前目录下的 `sensecraft.json` |

命令成功后，stderr 上可能各出现一条英文提示，同一条 24 小时内只出现一次：`Warning:` 表示有新版本，运行 `sensecraft-cli update`；`Notice:` 写出正在进行的活动名称、日期和链接。交互终端里会再问 `Ignore this activity? [y/N]`。

## 从源码构建

```bash
go build -o sensecraft-cli .
./sensecraft-cli --version

./script/build.sh
```

`./script/build.sh` 按 `packages/cli/package.json` 的版本号交叉编译到 `dist/`。

## 安全

Access Key 是密钥。不要提交进仓库，也不要贴进 Agent 对话。`auth info` 只打印 Key ID。

解绑、删除、发布、部署在带上 `--yes` 之前不会执行。Agent 必须先得到用户对这一次操作的确认，再加这个参数。

本工具不通过 USB 或串口烧录固件。`recamera open` 只打开控制台部署页。

## 许可证

[MIT](LICENSE)
