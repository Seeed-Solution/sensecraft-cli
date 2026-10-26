# SenseCraft CLI 架构与实现详解

本文是 `sensecraft-cloud-terminal`（产物名 `sensecraft-cli`）的内部架构说明：目录分层、鉴权、网络层、业务规则、L1–L4 安全模型、Agent Skills、npm 分发与 CI。

| 文档 | 用途 |
|------|------|
| [`README.md`](README.md) | 功能概览、SenseCAP 命令参考、快速开始 |
| [`INSTALL.md`](INSTALL.md) | 安装 / 升级 / 卸载 / 离线 / Skills 接入 |
| **本文** | 架构、实现细节、已知问题 |
| [`docs/sensecraft-cli-smoke.md`](docs/sensecraft-cli-smoke.md) | 发版冒烟清单 |
| [`docs/sensecraft-cli-test-cases.md`](docs/sensecraft-cli-test-cases.md) | 全量功能测试用例 |
| [`skills/README.md`](skills/README.md) | Agent Skills 发现与版本同步约定 |

---

## 1. 项目定位

`sensecraft-cli` 是 Seeed 两条云产品线的统一命令行入口：

| 能力面 | 后端 | 鉴权 | 子命令 |
|--------|------|------|--------|
| SenseCAP IoT（设备 / 分组 / 遥测） | `{host}/openapi` | Access Key（HTTP Basic） | `device` · `group` · `data` |
| SenseCraft AI（应用 / 模型 / 云部署） | `{host}/openapi` | Access Key（HTTP Basic） | `app` · `model` · `file` · `cloud` · `recamera` |

目标使用者：脚本自动化、CI/CD、以及 **AI Agent**（仓库内 `skills/` 供 Cursor / Claude Code 等驱动）。

技术栈：Go（`cobra` + `oauth2` + `term`，可 `CGO_ENABLED=0` 静态交叉编译）+ 零依赖 npm 壳（`@seeed-studio/sensecraft-cli`）。

---

## 2. 目录结构与分层

```
main.go                     → cmd.Execute()；错误则 os.Exit(1)
cmd/                        → Cobra 命令树（薄层：解析 flag → 调 service）
  root.go  auth/ device/ group/ data/
  app/ model/ file/ recamera/ cloud/
internal/
  app/          CmdContext / AICmdContext（凭证、sandbox、stdout/stderr）
  client/       Base + OpenAPI + AI + Auth HTTP 客户端
  config/       sensecraft.json 读写、profile、活跃指针、路径
  enum/         域名、endpoint、业务码、设备类型
  service/      业务逻辑主体（校验 / 编排 / sandbox 假数据 / L2 门禁）
  task/         auth_init_flow.go（五步 Device Flow 状态机）
  definition/ output/ ui/ utils/ version/
packages/cli/               → npm 包（JS 壳 + postinstall 下二进制）
script/                     → build / smoke / sync-skills
skills/                   → Agent Skills（SenseCAP 与 SenseCraft AI）
```

约定：`cmd` 只做 flag 绑定与调用；**业务规则、校验、`--yes` 门禁、sandbox 假响应均在 `internal/service`**。单测可构造 `AICmdContext{SandBox: true}` 直接测 service，不必经 Cobra。

---

## 3. 命令树

二进制：`sensecraft-cli`。约 10 个一级命令、80+ 叶子子命令。

```
sensecraft-cli
├── auth          init | info | add
├── device        list | view | channels | status | bind | unbind | sim
│                 └── devkit create | key
├── group         list | create | update | delete | move-devices
├── data          latest | list | chart
├── app           list | view | fields | blocks | create | update
│                 publish | unpublish | delete
│                 └── bom search | list | set
├── model         list | view | download | upload | constants
│                 create | update | publish | unpublish | delete
│                 └── recamera-pro resolve
├── file          upload
├── cloud         device list | view · deploy · order
└── update        检查 npm 最新版并升级 CLI
```

注册入口：`cmd/root.go`。

### 3.1 鉴权域划分

| 命令族 | Context | 凭证来源 |
|--------|---------|----------|
| `auth`、`device`、`group`、`data` | `GetCmdContext()` | SenseCAP Access Key（`auth init --kind sensecap`） |
| `app`、`model`、`file`、`cloud` | `GetAICmdContext()` | SenseCAP Access Key → `{ai-host}/openapi` Basic |
| `model recamera-pro resolve` | 无 | 纯本地校验，无网络 / 无 auth |

### 3.2 全局 flag 注意点

- **root 上无 PersistentFlags**：`--sandbox` / `--yes` 均为叶子本地 flag。正确写法：`sensecraft-cli device list --sandbox`，不是 `sensecraft-cli --sandbox device list`。
- **无 `--profile`**：活跃凭证由配置里的 `use_api` 决定。
- 输出：业务成功 → stdout 2 空格缩进 JSON；失败 → stderr + exit 1；`auth init|info` 为人类可读文本。
- Cobra：`SilenceUsage` + `SilenceErrors`；`Execute` 统一把错误打到 stderr。

命令级完整 flag 表以 `README.md`（SenseCAP）与各 `skills/*/commands.md`（AI）为准。

---

## 4. 配置与认证

### 4.1 `sensecraft.json` 结构

```json
{
  "app_config_list": [
    {
      "remote_type": "Global|China|Private",
      "host": "https://...（Private 必需；其它由 remote_type 解析）",
      "org_id": "...",
      "key_id": "...",
      "access_key": "...",
      "create_time": "RFC3339"
    }
  ],
  "use_api": "<活跃 Access Key 的 key_id>"
}
```

- 文件权限 **0600**；写入为 temp → chmod → rename 原子替换。
- `use_api` **必须显式设置**，无「取列表第一项」兜底。

### 4.2 配置路径优先级

1. `SENSECRAFT_CONFIG_FILE` — 配置文件完整路径  
2. `SENSECRAFT_CLI_PATH` — 配置**目录**（文件名固定 `sensecraft.json`）  
3. 都无 → **当前工作目录**下的 `sensecraft.json`

npm 壳会注入 `SENSECRAFT_CLI_PATH` → macOS/Linux：`~/.config/sensecraft-cli`；Windows：`%APPDATA%\sensecraft-cli`。  
因此：**`go build` 直接运行默认读 cwd；npm 安装默认读用户配置目录** — 二者不是同一文件。

### 4.3 OAuth Device Flow（`auth init`）

- `client_id = sensecraft-cli`（`enum.OAuthClientID`）
- 标准 Device Authorization Grant（`golang.org/x/oauth2`，`AuthStyleInParams`）
- 实现：`internal/task/auth_init_flow.go` 的 step 状态机（共 5 步）

步骤：

1. 选站点：Global / China
2. `StartDeviceFlow` → `user_code` + `verification_uri(_complete)`
3. 用户浏览器授权；CLI spinner 轮询 `PollDeviceToken`
4. org info → 列出或创建 Access Key → `UpsertAccessProfile`

这把 Access Key 同时用于 SenseCAP OpenAPI 和 SenseCraft AI `/openapi`。

账号中心默认按站点走 `account.seeed.cc`（Global）或 `account.seeed.cn`（China）。设置 `SENSECRAFT_AUTH_HOST` 后，设备码、换 token、以及用 token 换 Access Key 的用户接口都打到该站点根（末尾 `/` 或 `/authapi` 会去掉）。浏览器打开的是该服务返回的 `verification_uri`。

约束：非 sandbox 时必须 TTY（`IsInteractive()`）。**L3：Agent 不得代跑 `auth init`。**

Device Flow 过程中的 `access_token` 只用来查询组织和创建 Access Key，不会写入配置文件。

---

## 5. HTTP 客户端架构

### 5.1 BaseClient

- URL：`baseURL + path` + query；body JSON；默认超时 **15s**
- 选项：`WithBasicAuth` / `WithHeader` / `WithHTTPClient` / `WithSandbox`
- sandbox：stderr 打日志后返回 `ErrSandboxMode`，不发真实请求
- **无 HMAC / 签名**

### 5.2 各客户端对照

| Client | Base URL | 鉴权 | appid | 信封 | 超时 |
|--------|----------|------|-------|------|------|
| OpenApiClient | `{host}/openapi` | HTTP Basic(`key_id`:`access_key`) | 无 | `code` **字符串** `"0"` | 15s |
| AIApiClient | `{host}/openapi` | HTTP Basic(`key_id`:`access_key`) | 无 | `code` 数字 `0` | 15s |
| AuthAPIClient | `{authapi}` | 调用方原样传入 | 无 | `code == 0` | 30s |

`portal_api_client.go` 为空文件；portal 用户/Access Key 走 `AuthAPIClient` + `service/portal_access_service.go`。

### 5.3 环境域名

| API 族 | Global | China |
|--------|--------|-------|
| SenseCAP OpenAPI | `https://sensecap.seeed.cc` | `https://www.sensecap.seeed.cn` |
| AuthAPI | `https://account.seeed.cc/authapi` | `https://account.seeed.cn/authapi` |
| SenseCraft AI API | `https://api-gateway.seeed.cc` | **同 Global** |
| SenseCraft AI Console | `https://sensecraft.seeed.cc` | **同 Global** |

China 与 Global 的 AI host 相同；差异在账号 AuthAPI（`.cn` vs `.cc`）。`SENSECRAFT_DATA_HOST`、`SENSECRAFT_AI_HOST`、`SENSECRAFT_AUTH_HOST` 分别盖住 OpenAPI、AI 网关和账号中心；变量一旦设置，Global / China 都打到变量里的地址。OpenAPI 另支持 `Private` + 配置内自定义 `host`。

Endpoint 常量集中在 `internal/enum/`：`open_url.go`、`ai_api.go`、`auth.go`、`auth_biz.go`、`config.go`。

---

## 6. 业务服务层要点

### 6.1 模型校验（`model_validation.go`）

入口：`ValidateAndNormalizeModelPayload`（要求**完整模型**）。`model update` 先 `view_model` → merge → 再校验，故增量 `--from-json` 亦可；`--path` 会先上传再写入 `file_url`。

- 必填：`name` `description` `content` `pic_url` `file_url` `checksum` `precision`
- 长度按 **UTF-16 码元**（对齐 JS `String.length`）：`name ≤ 64`，`description ≤ 300`
- 版本正则 `^[1-9]\d?(\.([1-9]?\d)){2}$`；不可低于旧版本，**相等允许**
- 同版本且 `labels`/`file_url` 变化 → 自动 patch+1（`.99` 溢出则要求用户手填）
- labels 类名：首字符非数字、至少一个 ASCII 字母、仅字母数字空格
- 设备族互斥：`32,52` xiao-respeaker；`36,37,41` vision；`26` jetson；其余 single（只许 1 个）
- 格式×设备族、格式×扩展名映射见 Skills `reference-model-fields.md`
- 写回白名单约 18 字段；用户 JSON 禁止 `be_public=1`。**自己的已发布模型**更新时 CLI 会按 Console 补上 `be_public=1`（重新进审核），`re_review` 写入结果。他人公开库拒绝编辑。删除仅 `be_public=0`。

### 6.2 reCamera Pro（`uniform_type=53`）

`adapteds` 含 53 时自动走 Pro 规则（`--target recamera-pro` 为断言）。

| task | ai_framework | 格式 |
|------|--------------|------|
| 1 Detection | 必填，映射 YOLO 系 | rknn(6) |
| 2 Classification | **删除** | rknn(6) |
| 4 Pose | **仅 YOLOV8** | rknn(6) |
| 6 Audio | 删除 | alpkg(7) / zip(8) |
| 7 Tracking | 删除 | rknn(6) |
| 3 / 5 | 不支持（隐藏） | — |

precision：音频 `"3"`，视觉 `"1"`。非 Pro 强制删除 `ai_framework`。  
部署走**本地 IP / USB（L3）**，不是 `cloud deploy`。  
`model recamera-pro resolve`：本地预检，无 auth / 无网络。

### 6.3 应用创建两阶段

`create_app` 只收 `app_name`，故 CLI：

1. 从 `--from-json` 抽 `app_name`，删掉 `app_name`/`app_id`，余字段保留  
2. `POST create_app` → `app_id`  
3. 有余字段则 `POST update_app`（失败报错保留已创建 id）  
4. 注入 `console_url` + `detail_url`  

`app create`（含完整 from-json）**不是 L2**，不需 `--yes`。

应用编辑对齐 Console `edit.vue` / `list.vue` / `detail.vue`：仅 **作者**（`user_id`）可 update/封面/BOM/publish/delete；他人广场应用不能改。`be_public=2` 锁定编辑；已发布（`1`）可改并写入 sketch（info 补 `be_public=1`）；删除仅 `0`。`app publish` 一律 `privacy=2`（含已上架后的「发布草稿」），并额外要求 `--agree-declaration`（对齐 Console 发布确认弹窗 `warrantyAccepted`；同意状态不进 API）。

**Info 字段闸门**（`validateAppInfoPatch`，create `--from-json` 额外字段与 update 补丁在发请求前校验）：`app_name` 非空 ≤60；`excerpt` 非空 ≤300；`scene` 非空且须为 `app_scenarios_array` id；`allow_clone`/`allow_commerce` 为 0|1（拒 null/-1/2）；`images` 至少 1 个 http(s) URL、最多 16。仅校验补丁中**出现的**键（稀疏改名仍可只传 `app_name`）。

`update_app` 可选 `activity_id`（正整数，JSON number）。不传或空字符串表示不改活动标签。非法值在发请求前拒绝。活动不存在、未发布、已删除或不在有效期内时，服务端整次编辑失败（`25000` `Activity does not exist or has expired`），同一次请求里的其它修改也不会写入。`tags` 不再被服务端读取，CLI 会从请求体里去掉。

### 6.4 可见性闸门

`rejectForbiddenPublicFlags`：拒绝**用户载荷**中的 `privacy=1` / `be_public=1`。已发布应用的 info 保存由 CLI **自行注入** `be_public=1`。  
应用：`privacy=2` 提审（含草稿发布）、`privacy=0` 下架。模型：`status=1/0`（与 app privacy **字段不同**）。

### 6.5 文件上传（OSS POST 策略）

1. 本地 stat；文件名**去掉所有空格**  
2. **体积/格式闸门**（OpenAPI 上传规范，CLI service 层强制，发生在申请凭证之前）：  
   - type `1` 一律 ≤ **5 MB**（含模型 `pic_url` 与应用图）  
   - 扩展名 JPG/JPEG/PNG/GIF/WebP（拒 SVG）  
   - type `2`–`6` ≤ **100 MB**  
   - **拒绝 0 字节**空文件  
   - type `5` 静态资源必须带 `--flow-id`，否则不发请求  
3. 可选 MD5（`--checksum`）  
4. `GET /user/acquire_file_url` → `host`、`policy`、`signature`、`OSSAccessKeyId`、`security_token`、`key`、`file_url`、`max_bytes`；图片另有 `content_type`。`max_bytes` 接受 JSON 数字或字符串。本地大小超过返回的 `max_bytes` 时不再 POST  
5. 向 `host` POST `multipart/form-data`（`file` 最后）。图片同时带表单字段 `Content-Type`，且文件部分的 MIME 与 `content_type` 相同。成功是 HTTP **204**。这个请求不带 Access Key，也不带 `sensecraft-cli-*` 头。超时 30 分钟。业务侧只保存 `file_url`  

> CLI 不在本地计次。触发限流时，服务端返回 `11209` 或 `11210`。

`file_type`：1 图 / 2 文件·模型 / 3 app / 4 固件 / 5 静态资源（要 `flow_id`）/ 6 归档。`model upload` = type=2 + checksum。

关联（避免未挂库的 OSS 对象）：

- `model upload --path --id --yes` 或 `model update --id --path --yes`：先 `view_model`（公开库拒绝），再上传，再 `edit_model` 写入 `file_url`/`checksum`/`model_size`。`file_url` 或 `labels` 变化且版本字符串未改时，与 Console `incrementVersion` 相同：patch +1（`1.0.0` → `1.0.1`）。
- `model update --id --from-json --path --yes`：JSON 增量 + 新文件一次提交。
- `file upload --type 1 --app-id --yes`：封面图插入 `images[]` 首位；应用 `images[]` 最多 **16** 张（create/update/attach 均校验）。
- **无 id 默认拒绝上传**（`ErrAttachRequired`，避免孤儿 OSS）。新建模型 / `pic_url` / 后续 JSON 必须显式 `--unlinked`（或 type1 `--for-app`）。交互 TTY 下无 flag 时引导选择关联模型 / 应用 / 未关联新建；Agent（非 TTY）须先问用户再带 flag 重跑。有 id 无 `--yes`：**不上传**。

### 6.6 其它实现细节

- 模型下载：仅 **私有**（`be_public=0`）允许；默认 `file_url`，`--apply` 走 `apply_model`。公开库拒绝下载（含 `--url-only`），`list`/`view` 抹掉 `file_url`；部署走 `cloud deploy` / Console，设备在 apply 时拉固件。落盘结果默认不含 `url`（除非 `--url-only`）。`model_snapshot` 可能是对象或 JSON 字符串；剥 `timestamp` query
- Access Key 创建后**不返回密钥本体**，须回查列表（按 label 倒序匹配）
- `edit_model` 须同时带 `id` 与 `model_id`（缺 `id` → 11202）
- `publish_model` / `delete_model` 的 `model_id` 须为 **JSON number**（CLI `--id` 为纯数字时发 `63758` 而非 `"63758"`；字符串会 11202）
- 深链：模型仍是 `sensecraft.seeed.cc/ai/model/detail?...`。应用编辑和详情在应用站 `app.seeed.cc`（测试 AI host 含 `test-` 时为 `test-app.seeed.cc`）：`/application/edit?id=`、`/view-application/<id>-<slug>?from=user-center&tab=custom`。create / clone 注入，update 不注入

---

## 7. 安全模型：L1–L4 与 `--yes`

| Level | 含义 | 处理 |
|-------|------|------|
| **L1** | 只读 / 轻量写 | Agent 可直接执行 |
| **L2** | 敏感写 | 先向用户确认（聊天问一句），再带 `--yes` |
| **L3** | 强交互 | `auth init`、Web Serial、支付、Console 数据集导出 — Agent 只引导 |
| **L4** | 运维专属 | Admin `privacy=1` 等 — 不对终端 Agent 开放 |

硬门禁（service 层）：

```go
// ErrConfirmRequired — 缺 --yes 时返回
func requireYes(yes bool) error
```

**L2 命令清单：**

`app update|publish|unpublish|delete|bom set`  
`model create|update|publish|unpublish|delete`  
`cloud deploy`  
`device unbind`  
`group delete|move-devices`

**刻意非 L2：** `app create`、各类 upload、`device bind`、`group create|update`、`devkit create`。

缺 `--yes` 时：TTY 下用 `ui.Console.Confirm`（默认 No）；非 TTY（Agent）返回 `ErrConfirmRequired`，文案要求先在对话里询问。  
**用户第一句（「删这个63760」）不是确认。** Agent 必须先问：操作类型 + 环境 + 对象 ID，等用户回复「确认」后再加 `--yes`。协议见 [`skills/confirmation.md`](skills/confirmation.md)。

### Sandbox

与 `--yes` 正交。客户端返回 `ErrSandboxMode`，service 返回形状真实的假 JSON。  
注意：`auth init --sandbox` 仍可能写配置。  
部分 SenseCAP data view 方法未处理 sandbox 哨兵错误。

---

## 8. Agent Skills

布局见 [`skills/README.md`](skills/README.md)。

```
sensecraft-ai          Meta 路由（优先加载）
 ├─ sensecraft-apps
 ├─ sensecraft-files
 ├─ sensecraft-models
 └─ sensecraft-devices-cloud
sensecraft-data        SenseCAP（兄弟 skill，Access Key）
```

Agent 规则摘要：

1. 云调用前先 `auth info`
2. 永不代跑 `auth init`
3. L2 须**另开一轮**向用户确认后再加 `--yes`（第一句意图 ≠ 确认；模板见 `skills/confirmation.md`）
4. 不回显 access_key / device_key；公开模型不回显 `file_url` / 下载链接
5. 不伪造响应、不发明不存在的命令
6. L3 诚实引导 Console；L4 不碰  

零网络辅助：`app fields` / `app blocks` / `app doc compile`（静态 schema，或把 Markdown 编成编辑器块）。`app create --from-md` / `app update --from-md` 先走同一套编译再写入。  
版本同步：新增 CLI 命令须同变更更新对应 `SKILL.md` + `commands.md`。

---

## 9. npm 分发（`packages/cli`）

- 包名：`@seeed-studio/sensecraft-cli`（版本以 `package.json` 为准）
- 零 runtime 依赖；`bin/sensecraft-cli.js` → `spawnSync` Go 二进制，`stdio: inherit`
- 注入：`SENSECRAFT_CLI_PATH`（若未设）
- `sensecraft-cli install` 由 JS 处理：下二进制 → 全局注册 → TTY 下可选 `auth init`
- 二进制不进 npm tarball；`postinstall` / 首次调用从 CDN 下载
- 默认 base：`https://sensecraft-statics.seeed.cc/static/bin/sensecraft-cli`
- URL：`{RELEASE_BASE}/v{version}/sensecraft-cli-{os}-{arch}[.exe]`
- `vendor/.version`：版本戳；不一致则强制重下
- `sensecraft-cli update`：查询 `https://registry.npmjs.org/@seeed-studio/sensecraft-cli/latest`，npm 安装时执行 `npm install -g`。版本检查缓存在配置目录的 `update-state.json`，24 小时刷新一次；同一新版本的 stderr 警告也是 24 小时一次。活动列表缓存在 `activity-state.json`，同样 24 小时内只提醒一次。交互终端会问 `Ignore this activity? [y/N]`，回答 `y` 后按 `activity_id` 不再提醒；回答 `n` 时 24 小时后还会再出现。`SENSECRAFT_CLI_NO_ACTIVITY_NOTIFIER` 关闭活动提醒
- `prepack`：`script/sync-skills-to-npm.sh` 把 Skills 拷进包目录

| 环境变量 | 作用 |
|----------|------|
| `SENSECRAFT_CLI_RELEASE_BASE` | 下载基址 |
| `SENSECRAFT_CLI_LOCAL_DIST` | 本地 `dist/` |
| `SENSECRAFT_CLI_SKIP_DOWNLOAD=1` | 禁止联网 |
| `SENSECRAFT_CLI_SHA256` | 期望校验和 |
| `SENSECRAFT_CLI_STRICT_POSTINSTALL=1` | postinstall 失败则中断安装 |

支持平台：darwin/linux/win32 × x64/arm64。

---

## 10. 构建、测试与 CI

### 10.1 `script/build.sh`

- 版本：`$VERSION` 环境变量，否则读取 `packages/cli/package.json`。本地 `go build` 不注入时为 `dev`
- ldflags：`Version` / `Commit` / `Date` → `internal/version`
- `CGO_ENABLED=0` 交叉编译 6 平台 → `dist/`

### 10.2 冒烟

- `script/smoke-ai.sh`：Access Key 假配置 + `--sandbox`；含 L2「缺 `--yes` 必须失败」断言  
- `script/smoke-sensecap.sh`：device / group / data list  
- `packages/cli`：`npm test`、`npm run test:install`（隔离 HOME）  
- 发版人工勾选：[`docs/sensecraft-cli-smoke.md`](docs/sensecraft-cli-smoke.md)；全量用例：[`docs/sensecraft-cli-test-cases.md`](docs/sensecraft-cli-test-cases.md)

### 10.3 发版

仓库里没有 CI 配置。发版步骤：

1. tag 版本必须等于 `packages/cli/package.json` 的 version
2. `script/build.sh` 交叉编译，并生成 `SHA256SUMS`
3. 在 GitHub 仓库 `sensecraft-cli` 创建同名 Release，上传 `dist/` 里的二进制
4. `npm publish`（需要 `NPM_TOKEN`）

npm 安装器默认从 `https://sensecraft-statics.seeed.cc/static/bin/sensecraft-cli` 下载对应版本的二进制。

---

## 11. 与 Console 的关系

公开 CLI 调用 SenseCraft AI 的 **`{ai-host}/openapi`**（Access Key HTTP Basic，无 appid）。

**AI 鉴权**：`app` / `model` / `file` / `cloud` 固定 Access Key → `{ai-host}/openapi` Basic。

**不做：** Cookie 登录、本地 Web Serial 伪远程烧录、Admin 审核、发明不存在的数据集 export API。

业务校验尽量对齐 Console（UTF-16 长度、版本号、Pro 算法表）。

---

## 12. 已知问题与风险

1. **安装包下载**：npm 安装器默认从 CDN `sensecraft-statics.seeed.cc` 拉二进制。Release 附件和 CDN 上的文件名、版本必须一致，否则 postinstall 会 404。  
2. **校验和**：`SHA256SUMS` 无消费方；仅 `SENSECRAFT_CLI_SHA256` 时校验；注释中的旁车 `.sha256` 未实现。  
3. **tag 正则不接受 pre-release**（如 `v0.1.3-rc1`）。  
4. **`NPM_TOKEN` 缺失**：文案像 skip，实际 `exit 1`。  
5. **Go 版本**：`go.mod` 与 CI 镜像可能依赖 `GOTOOLCHAIN=auto`。  
6. **SenseCAP 部分写操作仍无 L2 `--yes`**（`device bind`、`group create|update`、`devkit create`）。  
7. **部分 data view 未处理 sandbox**。  
8. **测试缺口**：`auth_init_flow`、`device_flow`、portal Access Key 回查、devkit/group 写路径等。  
9. **其它**：Skills 同步目录可能带入 `.DS_Store`；Ctrl-C 时 JS 壳 exit 1 而非 130。

---

## 13. 相关源码索引

| 主题 | 路径 |
|------|------|
| 命令注册 | `cmd/root.go` |
| Context / 输出 | `internal/app/cmd_app.go` |
| 配置路径 / schema | `internal/config/` |
| HTTP 客户端 | `internal/client/` |
| L2 `requireYes` | `internal/service/open_api_app_write.go` |
| 模型校验 / Pro | `internal/service/model_validation.go`、`recamera_pro.go` |
| Device Flow / init | `internal/service/device_flow.go`、`internal/task/auth_init_flow.go` |
| npm 壳 / 下载 | `packages/cli/bin/`、`packages/cli/scripts/` |
| Skills 路由 | `skills/sensecraft-ai/SKILL.md` |
| 发版 | `script/build.sh`、`script/release.sh` |
