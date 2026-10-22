# SenseCraft CLI 测试用例

> 面向 QA / 发版验收，覆盖 `sensecraft-cloud-terminal`（二进制 `sensecraft-cli`）全部已实现功能细节。  
> 权威命令参考：`skills/sensecraft-data/commands.md`、`skills/*/commands.md`。  
> 单元/集成测试：`go test ./...`；沙箱冒烟：`script/smoke-sensecap.sh`、`script/smoke-ai.sh`。  
> **发版只跑冒烟时用 → [`sensecraft-cli-smoke.md`](sensecraft-cli-smoke.md)**。

---

## 环境与常量

### 测试前提

| 项                 | 说明                                                                                           |
| ------------------ | ---------------------------------------------------------------------------------------------- |
| 二进制             | `go build -o sensecraft-cli .` 或 `npx @seeed-studio/sensecraft-cli`                           |
| SenseCAP / SenseCraft AI 账号 | 已有 Access Key，或可走 `auth init --kind sensecap` |
| 交互环境           | `auth init` 需要 TTY + 浏览器；CI 用 `--sandbox` 或不测 init                                   |
| 配置隔离           | 建议用独立 `SENSECRAFT_CONFIG_FILE`，避免污染本机生产配置                                      |
| 写操作账号         | 具备可丢弃的测试设备 / App / Model；勿在生产组织盲删                                           |

### 能力分级（验收口径）

| 级别 | 含义           | CLI 约定                                                       |
| ---- | -------------- | -------------------------------------------------------------- |
| L1   | Agent 可直接跑 | 成功 = stdout 缩进 JSON；失败 = stderr + 非 0                  |
| L2   | 须先人工确认   | 缺 `--yes` → 非 0 + `confirmation required`（或等价文案）      |
| L3   | 人机交互       | `auth init`、USB/Web Serial 烧录 — **不在本 CLI 自动化范围内** |
| L4   | 运营专用       | `privacy=1` / Admin 审核 — **禁止 CLI/Agent 写入**             |

### 站点 / Host 矩阵

| `remote_type` | SenseCAP OpenAPI            | AI `/openapi`                 |
| ------------- | --------------------------- | ----------------------------- |
| Global        | `sensecap.seeed.cc/openapi` | `api-gateway.seeed.cc/openapi` |
| China         | `sensecap.seeed.cn/openapi` | 同 Global AI host             |

同一把 Access Key 打 SenseCAP OpenAPI 和 SenseCraft AI `/openapi`（HTTP Basic）。China 与 Global 的 AI host 相同，账号 AuthAPI 走 `.cn`。

### 认证模型

| Kind       | 持久化                  | 日常鉴权                      | 适用命令                                                                |
| ---------- | ----------------------- | ----------------------------- | ----------------------------------------------------------------------- |
| `sensecap` | `key_id` + `access_key` | HTTP Basic → `{host}/openapi` | `device` / `group` / `data` **以及** `app` / `model` / `file` / `cloud` |

### 配置路径优先级

1. `SENSECRAFT_CONFIG_FILE`（绝对或相对路径）
2. `{SENSECRAFT_CLI_PATH}/sensecraft.json`（npm wrapper 常注入 `~/.config/sensecraft-cli`）
3. `{cwd}/sensecraft.json`（纯 `go build` 默认常落这里）

### 关键设备 / 任务常量

| 常量                   | 值                                                                 | 备注                                       |
| ---------------------- | ------------------------------------------------------------------ | ------------------------------------------ |
| SenseCAP `device_type` | `1` gateway / `2` node（默认） / `3` devkit                        |                                            |
| Cloud `uniform_type`   | 常用 `26`（Jetson）                                                | 非 SenseCAP `device`                       |
| reCamera classic       | `40`                                                               |                                            |
| reCamera Pro           | `53`                                                               | 部署 = Console L3，**不是** `cloud deploy` |
| App publish privacy    | `2` 提交审核 / `0` 私有                                            | **禁止写 `1`**                             |
| Model publish status   | `1` 发布 / `0` 下架                                                | 勿与 app privacy 混用                      |
| File `type`            | `1` 图 / `2` 文件模型 / `3` app / `4` 固件 / `5` 文档 / `6` 压缩包 |                                            |

### 「设备」三套消歧（必测认知）

| 用户说「设备」          | 命令前缀                        | Auth                             |
| ----------------------- | ------------------------------- | -------------------------------- |
| SenseCAP 节点/网关/遥测 | `device` / `group` / `data`     | sensecap                         |
| Jetson / 云套件         | `cloud device` / `cloud deploy` | sensecap Access Key → `/openapi` |

---

## 冒烟清单（每次发版必跑）

精简勾选页（含命令与清理）：[`sensecraft-cli-smoke.md`](sensecraft-cli-smoke.md)。

| 序号 | 场景                                                                                  | 通过标准                                            |
| ---- | ------------------------------------------------------------------------------------- | --------------------------------------------------- |
| S1   | `go test ./...`                                                                       | 全绿                                                |
| S2   | `script/smoke-sensecap.sh`                                                            | 退出 0                                              |
| S3   | `script/smoke-ai.sh`                                                                  | 退出 0                                              |
| S4   | `auth info`                                                                           | 展示 Access Key 的 Site / Org / Key ID，不打印密钥 |
| S5   | `device list --sandbox` + `app list --sandbox` + `model list --sandbox`               | 均为合法 JSON                                       |
| S6   | L2 缺 `--yes`：`app delete` / `model publish` / `cloud deploy` / `device unbind` / `group delete` | 均非 0，提示需确认                                  |
| S7   | `app create --name …`（真实或 sandbox）                                               | stdout 含可用 `app_id`（即使后端 update 返回 `{}`） |
| S8   | 稀疏 `app update` 仅改名                                                              | `view --sketch` 新名称生效，其它 info 字段未丢      |

---

## 一、安装与配置

**优先级**：P0（发版） / P1（回归）

| 编号   | 用例名称           | 前置条件          | 步骤                                                      | 预期结果                                                                    |
| ------ | ------------------ | ----------------- | --------------------------------------------------------- | --------------------------------------------------------------------------- |
| CFG-01 | 显示帮助           | 有二进制          | `sensecraft-cli` / `--help` / `--version`                 | 打印命令树与版本，退出 0                                                    |
| CFG-02 | CONFIG_FILE 优先   | 准备两份不同 json | 设 `SENSECRAFT_CONFIG_FILE` 指向 A，CLI_PATH 下有 B       | `auth info` 读到 A                                                          |
| CFG-03 | CLI_PATH 次优先    | 未设 CONFIG_FILE  | 设 `SENSECRAFT_CLI_PATH`，其下有 sensecraft.json          | 读该文件                                                                    |
| CFG-04 | cwd 兜底           | 两 env 皆空       | 在含 sensecraft.json 的目录运行                           | 读 cwd 文件                                                                 |
| CFG-05 | npm 注入路径       | 全局 npm 包       | `npx … auth info`                                         | 使用 `~/.config/sensecraft-cli`（或 Windows APPDATA），与纯 go 默认可能不同 |
| CFG-06 | 本地 dist 跳过下载 | 有本地制品        | `SENSECRAFT_CLI_LOCAL_DIST=…` 再 install                  | 不拉远程或使用本地                                                          |
| CFG-07 | 跳过下载           | CI                | `SENSECRAFT_CLI_SKIP_DOWNLOAD=1`                          | 不联网下载                                                                  |
| CFG-08 | Skills 随包        | npm 安装后        | 检查 `$(npm root -g)/@seeed-studio/sensecraft-cli/skills` | 存在 apps/models/data/files 等 skill                                        |

---

## 二、认证 Auth

### 2.1 `auth info`（L1）

| 编号     | 用例名称    | 前置条件                   | 步骤        | 预期结果                                                                      |
| -------- | ----------- | -------------------------- | ----------- | ----------------------------------------------------------------------------- |
| AUTH-I01 | 展示配置    | 已有 Access Key            | `auth info` | Site / Org / Key ID；不打印 access_key                                    |
| AUTH-I02 | 未初始化    | 空配置                     | `auth info` | 提示 `auth init --kind sensecap`；可提示 `auth add`                       |
| AUTH-I03 | 多把 Key    | 多于一条 Access Key        | `auth info` | 当前 `use_api` 标 `[active]`                                              |
| AUTH-I05 | 不调云端    | 断网                       | `auth info` | 仍成功（纯本地）                                                          |

### 2.2 `auth init`（L3）

| 编号     | 用例名称                  | 前置条件               | 步骤                                                    | 预期结果                                                                              |
| -------- | ------------------------- | ---------------------- | ------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| AUTH-N01 | kind sensecap 全流程      | TTY + 浏览器           | `auth init --kind sensecap` → 选站点 → 授权 → 选/建 Key | 写入 Access profile 并 active；可 `device list` 与 `app list`                         |
| AUTH-N03 | kind 别名                 | TTY                    | `--kind sc` / `openapi`                                 | 落到 sensecap                                                                         |
| AUTH-N04 | 默认 kind                 | 不传 `--kind`          | `auth init`                                             | 与 `--kind sensecap` 相同                                                             |
| AUTH-N05 | 站点 Global/China         | 各跑一次               | 选对应站点完成 init                                     | `remote_type` 为 Global 或 China                                                      |
| AUTH-N06 | 非 TTY 拒绝               | 管道/无 TTY            | `auth init`（非 sandbox）                               | 非 0，提示需交互终端                                                                  |
| AUTH-N07 | sandbox 不真发网          | 任意                   | `auth init --sandbox`                                   | 模拟成功；配置可写 sandbox 凭证                                                       |
| AUTH-N09 | Agent 不得代跑            | —                      | 文档/Skill 验收                                         | Skill 明确 Never run auth init for user                                               |
| AUTH-N10 | SenseCAP 无 Key 时创建    | 账号无 Key             | init sensecap                                           | 可创建 Key（label + 读写权限）后保存                                                  |

### 2.3 `auth add`（SenseCAP only）

| 编号     | 用例名称     | 前置条件   | 步骤                                               | 预期结果                                                                |
| -------- | ------------ | ---------- | -------------------------------------------------- | ----------------------------------------------------------------------- |
| AUTH-A01 | 非交互成功   | 有效 Key   | `auth add --key-id … --access-key … --site Global` | `view_organization` 校验通过；profile active；打印 OrgID（数字）与 Site |
| AUTH-A02 | 默认 Global  | 有效 Key   | 省略 `--site`                                      | remote_type=Global                                                      |
| AUTH-A03 | site 别名    | 有效 Key   | `--site China` / `cn`                              | 映射到 China                                                            |
| AUTH-A04 | 错误密钥     | 假 Key     | 同上                                               | 校验失败，非 0，**不**写入坏配置（或写入前失败）                        |
| AUTH-A05 | 缺参非交互   | 无 TTY     | 不传 key-id/access-key                             | 非 0                                                                    |
| AUTH-A06 | 交互补全     | TTY        | 只传部分参数                                       | 提示输入缺失项                                                          |
| AUTH-A07 | 密钥不落日志 | 任意       | 观察 stdout/stderr                                 | 不回显完整 access-key                                                   |
| AUTH-A08 | OrgID 格式   | 合并后回归 | 成功路径打印 Organization ID                       | 使用 `%d`（int64），进程不 panic；`go test` 编译通过                    |

---

## 三、SenseCAP — device

**Auth**：sensecap。成功 stdout = OpenAPI `data` JSON。

| 编号   | 用例名称                 | 前置条件      | 步骤                                                                              | 预期结果                                    |
| ------ | ------------------------ | ------------- | --------------------------------------------------------------------------------- | ------------------------------------------- |
| DEV-01 | list 默认 node           | Access Key    | `device list`                                                                     | JSON 列表；默认 device_type=2               |
| DEV-02 | list gateway             | 同上          | `--device-type 1`                                                                 | 仅网关类                                    |
| DEV-03 | list 按组过滤            | 有 group      | `--group-uuid …`                                                                  | 仅该组设备                                  |
| DEV-04 | list sandbox             | —             | `--sandbox`                                                                       | 样例 JSON，无真实网络                       |
| DEV-05 | view 单 EUI              | 已知 EUI      | `device view --eui <eui>`                                                         | 详情 JSON                                   |
| DEV-06 | view 批量 EUI            | 多个          | `--eui a --eui b`（≤约 50）                                                       | 批量结果                                    |
| DEV-07 | view 缺 eui              | —             | 不传 `--eui`                                                                      | 参数错误，非 0                              |
| DEV-08 | channels                 | 在线节点      | `device channels --eui …`                                                         | 通道信息                                    |
| DEV-09 | status                   | 同上          | `device status --eui …`                                                           | 在线/状态字段                               |
| DEV-10 | bind 成功（L2 确认口径） | 合法 eui+code | 用户确认后 `bind --eui --code [--name] [--group-uuid] [--longitude] [--latitude]` | 绑定成功                                    |
| DEV-11 | bind 缺 code             | —             | 仅 `--eui`                                                                        | 参数错误                                    |
| DEV-12 | unbind（L2）             | 已绑定        | 无 `--yes` → 非 0；`unbind --eui … --yes`                                             | 解绑成功                                    |
| DEV-13 | sim                      | 有 ICCID      | `device sim --iccid …`                                                            | SIM 信息                                    |
| DEV-14 | devkit create            | 测试 org      | `device devkit create --sku <Blank\|xiao\|Grove Vision AI V2> --name …`           | 创建成功；**凭证当密钥**，不全文回显        |
| DEV-15 | devkit key               | 16 位 EUI     | `device devkit key --eui …`                                                       | EUI 转大写；返回 key/token 类字段，注意保密 |
| DEV-16 | EUI 非法长度             | —             | 超限批量或非法 EUI                                                                | 明确错误                                    |
| DEV-17 | 未授权                   | 无 Access Key | `device list`                                                                     | 提示先 auth                                 |
| DEV-18 | 只读 Key 写操作          | Access Key 为 only-read | `device unbind --eui … --yes`（或其它写操作）                            | 非 0；提示缺写权限 + `auth init --kind sensecap`，非原始 `You have no options permission` |

---

## 四、SenseCAP — group

| 编号   | 用例名称       | 前置条件    | 步骤                                        | 预期结果                                           |
| ------ | -------------- | ----------- | ------------------------------------------- | -------------------------------------------------- |
| GRP-01 | list           | Access Key  | `group list`                                | 组列表 JSON                                        |
| GRP-02 | list sandbox   | —           | `--sandbox`                                 | 样例                                               |
| GRP-03 | create         | —           | `group create --name "QA-…"`                | 返回含 group uuid；组数上限约 50                   |
| GRP-04 | create 缺 name | —           | 无 `--name`                                 | 非 0                                               |
| GRP-05 | update         | 已有组      | `group update --group-uuid … --name …`      | 名称更新                                           |
| GRP-06 | move-devices（L2） | 设备+目标组 | 无 `--yes` → 非 0；`group move-devices --group-uuid … --eui … --yes` | 设备出现在目标组                                   |
| GRP-07 | delete（L2）       | 可删空组    | 无 `--yes` → 非 0；`group delete --group-uuid … --yes`               | 删除成功                                           |
| GRP-08 | 写操作人工确认     | Agent 场景  | Skill/流程                                                          | bind/unbind/create/update/delete/move 前需用户确认；unbind/delete/move 须带 `--yes` |

---

## 五、SenseCAP — data

| 编号 | 用例名称 | 前置条件 | 步骤 | 预期结果 |
|------|----------|----------|------|----------|
| DATA-01 | latest | 有遥测 | `data latest --eui … [--measurement-id] [--channel-index]` | 最新点 |
| DATA-02 | latest 缺 eui | — | 无 `--eui` | 非 0 |
| DATA-03 | list | 有历史 | `data list --eui … [--limit] [--time-start/end ms]` | 历史点列 |
| DATA-04 | chart | 同上 | `data chart --eui … --interval 60` | 按分钟聚合；默认 interval=60 |
| DATA-05 | sandbox | — | `data latest --eui … --sandbox` | 样例 JSON |
| DATA-06 | LoRaWAN 站点 | 国际设备 | Global profile | Global OpenAPI；勿误用 China |

---

## 六、SenseCraft AI — app（应用广场）

**Auth**：SenseCAP Access Key → `{ai-host}/openapi` Basic。`app list` **默认 mine**；公开库需 `--public`。

### 6.1 只读

| 编号    | 用例名称          | 前置条件   | 步骤                                      | 预期结果                                          |
| ------- | ----------------- | ---------- | ----------------------------------------- | ------------------------------------------------- |
| APP-R01 | list 默认 mine    | Access Key | `app list`                                | 仅我的应用                                        |
| APP-R02 | list public       | 同上       | `app list --public [--name] [--scene]`    | 公开广场；`--name` → API `app_name`（同 Console） |
| APP-R02b | list 设备类型     | 同上       | `app list --public --uniform-type 40`     | 只返回 reCamera（`uniform_type=40`）应用          |
| APP-R03 | public/mine 互斥  | —          | 同时传两者                                | 错误或按文档互斥处理                              |
| APP-R04 | 分页              | —          | `--page` / `--length`                     | 分页字段正确                                      |
| APP-R05 | mark-type 过滤    | 若支持     | `--mark-type …`                           | 过滤生效                                          |
| APP-R06 | view              | 已知 id    | `app view --id …`                         | 详情（文档/部署等）                               |
| APP-R07 | view sketch       | 同上       | `--sketch`                                | 精简结构，供 update merge 对照                    |
| APP-R08 | fields            | Access Key | `app fields`                              | 静态 schema JSON                                  |
| APP-R09 | blocks            | 同上       | `app blocks`                              | tool-block catalog                                |
| APP-R10 | sandbox list/view | —          | `--sandbox`                               | 样例，无网络                                      |
| APP-R11 | scene 常量        | AI         | `model constants` → `app_scenarios_array` | 与 `--scene` 取值一致                             |

### 6.2 写入与 L2

| 编号     | 用例名称                         | 前置条件                         | 步骤                                                                 | 预期结果                                                                |
| -------- | -------------------------------- | -------------------------------- | -------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| APP-W01  | create 仅 name（非 L2）          | AI                               | `app create --name "QA-…"`                                           | **不需要** `--yes`；stdout 含 `app_id`/`app_name`；可有 console_url     |
| APP-W02  | create from-json 全量            | 准备 JSON                        | `app create --from-json …`                                           | 非 L2；一次创建成功                                                     |
| APP-W03  | create 后端 update 返回 `{}`     | 可复现环境                       | create 后内部 follow-up update                                       | CLI **仍保留** app_id，不丢身份                                         |
| APP-W04  | 禁止 blind-retry                 | create 输出异常                  | **先** `app list --mine --name …`                                    | 不二次盲目 create；避免重复应用                                         |
| APP-W07  | update 需 --yes                  | 已有 app                         | 无 `--yes`                                                           | 非 0                                                                    |
| APP-W08  | 稀疏改名 merge                   | 已有完整 info                    | `echo '{"app_name":"New"}' \| app update --id … --from-json - --yes` | 新名称生效；excerpt/scene/images 等未清空                               |
| APP-W08b | 空 app_name                      | —                                | update/create JSON `app_name:""`                                     | 非 0；`app_name is required`                                            |
| APP-W08c | excerpt 空/超 300                | —                                | update JSON                                                          | 非 0；明确字段错误                                                      |
| APP-W08d | scene 空/非法 id                 | —                                | update JSON                                                          | 非 0；非 Database analysis error                                        |
| APP-W08e | allow_clone/commerce 非法        | null / -1 / 2                    | update JSON                                                          | 非 0；须为 0 或 1                                                       |
| APP-W08f | images 空/非法 URL               | null / [] / bad URL              | update JSON                                                          | 非 0；至少 1 张合法 http(s) URL                                         |
| APP-W09  | materials-only 不强制 info merge | 仅 materials JSON                | update                                                               | 不因缺 info 字段而误覆盖                                                |
| APP-W10  | 禁止 be_public=1 / privacy=1     | —                                | create/update JSON 含禁止值                                          | CLI/校验拒绝                                                            |
| APP-W11  | publish                          | 私有 app                         | `app publish --id … --agree-declaration --yes`                       | privacy 流程为 **2**（待审）                                            |
| APP-W11b | publish 缺声明同意               | 私有 app                         | 仅 `--yes`、无 `--agree-declaration`                                 | 非 0；提示发布声明 URL（同 Console warrantyAccepted）                   |
| APP-W12  | unpublish                        | 已发布/待审                      | `app unpublish --id … --yes`                                         | privacy=**0**；不需 `--agree-declaration`                               |
| APP-W13  | delete                           | **私有**测试 app (`be_public=0`) | `app delete --id … --yes`                                            | 删除；缺 yes 失败；已发布/审核中拒绝                                    |
| APP-W16  | 已发布可编辑                     | `be_public=1` 自己的应用         | `app update --yes`                                                   | 成功；stdout `sketch_saved`；CLI 发 `be_public=1`；广场未改直到 publish |
| APP-W17  | 审核中锁定                       | `be_public=2`                    | update / bom set / cover                                             | 拒绝（同 Console `isAppLockedForEditing`）                              |
| APP-W18  | 发布草稿                         | 已发布 + 草稿                    | `app publish --agree-declaration --yes`                              | 同 Console publishDraft（`privacy=2`）；`draft_publish`                 |
| APP-W19  | 他人应用不可编辑                 | 广场别人的 app                   | update / publish / delete / bom                                      | 拒绝                                                                    |
| APP-W14  | update 成功体可能 `{}`           | —                                | update 后                                                            | **必须** `app view --id --sketch` 复核                                  |
| APP-W15  | name 非唯一                      | 同名已存在                       | 再 create 同名                                                       | 允许成功（新 id）；Agent 不得当失败重试                                 |

### 6.3 BOM

| 编号    | 用例名称     | 前置条件 | 步骤                                                  | 预期结果                               |
| ------- | ------------ | -------- | ----------------------------------------------------- | -------------------------------------- |
| APP-B01 | bom search   | AI       | `app bom search --name "XIAO"`                        | 含 SKU；Seeed 行可读 price/pic_url     |
| APP-B02 | bom list     | 有 app   | `app bom list --id … [--sketch]`                      | materials 列表                         |
| APP-B03 | bom set L2   | —        | 无 `--yes`                                            | 非 0                                   |
| APP-B04 | bom set 成功 | 用户确认 | `app bom set --id … --from-json materials.json --yes` | Seeed 行带 `id`(=SKU)；自定义行可无 id |
| APP-B05 | 自定义图价   | —        | 试图写入自定义 price/image                            | 平台不支持；文档/行为一致              |

---

## 七、SenseCraft AI — model（模型库）

**Auth**：SenseCAP Access Key → `{ai-host}/openapi` Basic。`model list` **默认 public**（与 app 相反）。

| 编号    | 用例名称               | 前置条件                                                     | 步骤                                                      | 预期结果                                                                            |
| ------- | ---------------------- | ------------------------------------------------------------ | --------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| MDL-01  | list 默认 public       | Access Key                                                   | `model list`                                              | 公开模型                                                                            |
| MDL-02  | list mine              | —                                                            | `model list --mine`                                       | 我的模型                                                                            |
| MDL-03  | public/mine 互斥       | —                                                            | 同时传                                                    | 互斥错误或文档行为                                                                  |
| MDL-04  | 名称/任务过滤          | —                                                            | `--name face` / `--task` / 分页                           | `--name` 映射 API `search`（同 Console）；count 为过滤后总量，非全库                |
| MDL-04b | 设备类型过滤           | —                                                            | `--uniform-type 40` / `--adapteds <id>`                   | 映射 API `uniform_type` / `adapteds`（同 Console cardList）                         |
| MDL-05  | view                   | 已知 id                                                      | `model view --id …`                                       | 详情；公开模型无 `file_url`                                                         |
| MDL-06  | constants              | —                                                            | `model constants`                                         | 场景/设备/任务等枚举                                                                |
| MDL-07  | download 文件          | **私有**模型                                                 | `model download --id … --out ./m.bin`                     | 文件落地；JSON **不含** url                                                         |
| MDL-08  | download url-only      | **私有**模型                                                 | `--url-only`                                              | 仅 URL，不落盘                                                                      |
| MDL-09  | download apply         | **私有**且支持                                               | `--apply`                                                 | 按实现应用；公开模型不可用                                                          |
| MDL-09b | 公开模型禁止下载       | 公开库 id                                                    | `model download --id …`（含 `--apply` / `--url-only`）    | 非 0；提示走部署拉固件，不给下载链接                                                |
| MDL-10  | upload 无关联拒绝      | 本地文件、无 `--id`/`--unlinked`                             | `model upload --path ./x.rknn`                            | 非 0；**不上传**；`ErrAttachRequired`（引导问用户）                                 |
| MDL-10a | upload 显式未关联      | 新建模型流程                                                 | `model upload --path ./x.rknn --unlinked`                 | 返回 file_url（type=2+checksum）；`attached: false` + hint                          |
| MDL-10b | upload 关联并自增版本  | 私有模型、确认                                               | `model upload --path ./x.tflite --id <id> --yes`          | 写入 file_url/checksum/model_size；版本 patch+1（如 1.0.0→1.0.1）；`attached: true` |
| MDL-10c | upload 无 --yes        | 私有模型                                                     | `--id` 无 `--yes`                                         | 非 0；**不上传**（避免孤儿 OSS）                                                    |
| MDL-10d | upload 公开模型        | 公开库 id                                                    | `--id` 公开模型 `--yes`                                   | 非 0；拒绝挂到公开库                                                                |
| MDL-11  | create 需 --yes        | 完整 JSON 含 file_url                                        | 无 yes                                                    | 非 0（**与 app create 不对称**）                                                    |
| MDL-12  | create 成功            | 用户确认                                                     | `model create --from-json … --yes`                        | 返回 model_id                                                                       |
| MDL-12b | create/update 字段闸门 | 空 name / 超长 description / 非法 precision·scenario·pic_url | create 或 update --from-json                              | 非 0；字段级错误（对齐 Console add.vue）                                            |
| MDL-12b | 公开模型禁止克隆       | 他人公开库 id                                                | `model create --from-json` 带该 `id`/`model_id`/`from_id` | 非 0；提示无 model clone，走部署                                                    |
| MDL-13  | create Pro 断言        | Pro JSON                                                     | `--target recamera-pro`                                   | 校验 format/precision 等符合 Pro；失败则明确错误                                    |
| MDL-14  | update 增量 JSON       | 已有私有模型                                                 | `model update --id … --from-json … --yes`                 | 增量合并成功；仅改 description 不升版本                                             |
| MDL-14b | update 换文件          | 私有模型 + 本地文件                                          | `model update --id … --path ./x.tflite --yes`             | 上传并挂载；版本 patch+1；返回 previous_version/version                             |
| MDL-14e | 更新已发布自己的模型   | 自己的 be_public=1                                           | `model update --id … --path … --yes`                      | 成功；`re_review=true`；请求带 be_public=1（重新审核）                              |
| MDL-14f | 更新他人公开库         | 他人公开 id                                                  | `model update --id … --path/--from-json --yes`            | 非 0；不可编辑他人公开模型                                                          |
| MDL-14c | update 改 labels       | 私有模型                                                     | `--from-json` 改 labels                                   | 与 Console 相同，版本 patch+1                                                       |
| MDL-14d | update 无 payload      | —                                                            | 无 `--from-json` 且无 `--path`                            | 非 0                                                                                |
| MDL-15  | publish/unpublish      | —                                                            | `publish`/`unpublish --id … --yes`                        | status 1 / 0；勿用 privacy                                                          |
| MDL-16  | delete                 | 私有测试模型                                                 | `delete --id … --yes`                                     | 删除                                                                                |
| MDL-16b | delete 已发布          | be_public≠0                                                  | `delete --id … --yes`                                     | 非 0；需先 unpublish                                                                |
| MDL-17  | recamera-pro resolve   | **无 auth**                                                  | `model recamera-pro resolve --task … [--ai-framework …]`  | 本地解析算法/格式映射；不联网                                                       |
| MDL-18  | Pro ≠ cloud deploy     | Pro 模型                                                     | 部署意图                                                  | 引导 Console USB/IP（L3），不走 `cloud deploy`                                      |
| MDL-19  | sandbox                | —                                                            | 各命令 `--sandbox`                                        | 样例 JSON                                                                           |

---

## 八、SenseCraft AI — file

| 编号     | 用例名称               | 前置条件             | 步骤                                                          | 预期结果                                     |
| -------- | ---------------------- | -------------------- | ------------------------------------------------------------- | -------------------------------------------- |
| FIL-01   | 默认 type=2 无关联拒绝 | Access Key           | `file upload --path ./a.bin`                                  | 非 0；**不上传**；`ErrAttachRequired`        |
| FIL-01a  | 显式未关联             | 新建流程             | `file upload --path ./a.bin --unlinked`                       | 上传成功；`attached: false` + hint           |
| FIL-01b  | 关联模型               | 私有模型             | `file upload --path ./a.tflite --model-id <id> --yes`         | 同 MDL-10b                                   |
| FIL-01c  | 关联应用封面           | 已有 app             | `file upload --path ./cover.png --type 1 --app-id <id> --yes` | `images[0]` 为新封面；`attached: true`       |
| FIL-01d  | model-id/app-id 互斥   | —                    | 同时传                                                        | 非 0                                         |
| FIL-01e  | 图片 >5MB              | 本地 >5MB 图         | `file upload --type 1 --unlinked` 或 `--for-app`              | 非 0；所有图片上限 5MB                       |
| FIL-01e2 | type 5 缺少 flow_id    | 本地静态文件         | `file upload --type 5`                                        | 非 0；不申请凭证；`--flow-id` 后才上传      |
| FIL-01f  | 非图 >100MB            | 本地大文件           | `file upload --type 2 --unlinked`                             | 非 0；对齐 Console 100MB                     |
| FIL-01f2 | 空文件 0B              | 本地 0 字节          | `file upload --unlinked` / `model upload --unlinked`          | 非 0；拒绝空文件                             |
| FIL-01g  | 应用图 >16 张          | from-json 含 17+ URL | `app update --from-json` / attach                             | 非 0；对齐 Console maxImageCount=16          |
| FIL-02   | type 1 图片            | —                    | `--type 1`                                                    | 可供 app images[]；扩展名限 JPG/PNG/GIF/WebP |
| FIL-03   | type 3–6               | —                    | 分别上传                                                      | 类型与用途匹配                               |
| FIL-04   | checksum               | —                    | `--checksum <md5>`                                            | 服务端校验通过                               |
| FIL-05   | 缺 path                | —                    | 无 `--path`                                                   | 非 0                                         |
| FIL-06   | 大文件/失败            | 坏路径               | 不存在文件                                                    | 明确错误                                     |
| FIL-07   | sandbox                | —                    | `--sandbox`                                                   | 样例                                         |

---


## 十、SenseCraft AI — cloud（Jetson / 云套件）

**Auth**：SenseCAP Access Key → `{ai-host}/openapi` Basic。

| 编号   | 用例名称                | 前置条件      | 步骤                                      | 预期结果                          |
| ------ | ----------------------- | ------------- | ----------------------------------------- | --------------------------------- |
| CLD-01 | device list 默认 26     | Access Key    | `cloud device list`                       | uniform_type=26 列表              |
| CLD-02 | 名称/adapteds/分页      | —             | `--name` / `--adapteds` / page            | 过滤正确                          |
| CLD-03 | device view             | 已知 eui      | `cloud device view --eui …`               | 详情                              |
| CLD-04 | view 缺 eui             | —             | 无 eui                                    | 非 0                              |
| CLD-05 | deploy 缺 yes           | —             | 有 eui+model-id 无 yes                    | 非 0                              |
| CLD-06 | deploy 成功             | 在线套件      | `cloud deploy --eui … --model-id … --yes` | 返回 request_id                   |
| CLD-07 | deploy + stream JSON    | —             | `--from-json stream.json --yes`           | 带流配置部署                      |
| CLD-08 | order 轮询              | 有 request_id | `cloud order --request-id …`              | 状态推进；**非 L2**               |
| CLD-09 | 与 SenseCAP device 隔离 | —             | 用 SenseCAP EUI 调 cloud                  | 失败或找不到，不串域              |
| CLD-10 | 与 Pro 本地部署隔离     | Pro           | —                                         | 不引导 cloud deploy 作为 Pro 方案 |
| CLD-11 | sandbox                 | —             | `--sandbox`                               | 样例                              |

---

## 十（附）、SenseCraft AI — recamera（经典 40）

| 编号    | 用例名称           | 前置条件     | 步骤                                                         | 预期结果                                                         |
| ------- | ------------------ | ------------ | ------------------------------------------------------------ | ---------------------------------------------------------------- |
| REC-01  | 列表必须带 40      | Access Key   | `app list --public --uniform-type 40` / `model list --public --uniform-type 40` | 仅 API `uniform_type=40`；空列表则停止，不放宽                   |
| REC-02  | 我的应用/模型      | Access Key   | `--mine --uniform-type 40`                                   | 可列出私有项                                                     |
| REC-03  | check 模型通过     | 40 模型      | `recamera check --model-id …`                                | `ok=true`；`flash=local`；`console_url` 为 `/ai/model/detail`    |
| REC-04  | check 拒绝非 40    | 32/36/53     | `recamera check --model-id …`                                | 非 0；不打开浏览器                                               |
| REC-05  | check 应用多固件   | 多 block     | `recamera check --app-id …`                                  | `ask_pick=true`；Agent 须再问用户                                |
| REC-06  | open 打开部署页    | check 通过   | `recamera open --model-id …`                                 | 打开模型详情；用户点部署；IP/USB；**不** `cloud deploy`          |
| REC-07  | open --print-only  | —            | `--print-only`                                               | 只打印 URL，不打开浏览器                                         |
| REC-08  | 公开模型不下载     | 公开 40 模型 | `model download` / 索要 file_url                             | 拒绝；引导 `recamera open`                                       |
| REC-09  | 公开模型不克隆     | 公开 40 模型 | `model create` 抄 view                                       | 拒绝；公开模型只部署                                             |
| REC-10  | 无设备/IP          | 无板子       | 用户要烧录                                                   | 仍给 Console 链接并停止；不编造设备协议                          |
| REC-11  | 与 Pro/Grove 隔离  | —            | `/ai/recamera` 或 `uniform_type 53`                          | 不把 40 与 53/36 当成同一设备                                    |
| REC-12  | sandbox            | —            | `recamera check/open --sandbox`                              | 样例 JSON；open 不打开浏览器                                     |

---


## 十二、跨域与 Agent / Skill 规则

| 编号 | 用例名称                    | 前置条件     | 步骤                                                                                      | 预期结果                                                                            |
| ---- | --------------------------- | ------------ | ----------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| X-01 | 先 auth info                | Agent        | 任何云调用前                                                                              | 必须先检查 auth                                                                     |
| X-02 | 不代跑 auth init            | Agent        | 缺凭证                                                                                    | 只指导用户本地执行                                                                  |
| X-03 | L2 不擅自 --yes / 协议 flag | Agent        | 用户第一句「删这个63760」/ publish/delete/deploy                                          | **先问**（操作类型+环境+ID），等「确认」再 `--yes`；同轮不带 yes；publish 不传 `--agree-declaration` |
| X-04 | 密钥不回显                  | 全程         | 日志/回复                                                                                 | 无完整 access_token / access_key                                                    |
| X-05 | 成功只解析 stdout JSON      | —            | 混合 stderr 进度                                                                          | 仅成功时 parse stdout                                                               |
| X-06 | 术语：应用                  | —            | 「应用」无额外上下文                                                                      | 默认应用广场                                                                        |
| X-07 | 术语：部署                  | —            | 「部署模型」                                                                              | 先澄清 Jetson cloud / 本地 L3 / app tool-block                                      |
| X-08 | 交叉流：遥测→模型→Jetson    | 双 profile   | data latest → model list → cloud deploy --yes → order                                     | 全链路成功且 auth 不混用                                                            |
| X-09 | L3 烧录                     | USB/IP 板    | Agent                                                                                     | 经典 40：`recamera open` 后 Console 点部署；其它板 `/ai/device/local/32\|36\|37\|41\|52`；不假装远程烧录 |
| X-10 | L4 禁止                     | —            | 尝试 privacy=1                                                                            | 拒绝                                                                                |
| X-11 | Skills 路由                 | 装包后       | 按意图加载 sensecraft-ai / apps / models / recamera / devices-cloud / data / files | 不串 skill                                                                          |
| X-12 | China SenseCAP + Global AI  | 混合站点需求 | 分别 profile                                                                              | SenseCAP 可用 China；AI 仍打文档规定 host                                           |

---

## 十三、负面 / 异常 / 安全

| 编号   | 用例名称               | 前置条件      | 步骤                         | 预期结果                        |
| ------ | ---------------------- | ------------- | ---------------------------- | ------------------------------- |
| NEG-01 | 无网真实调用           | 断网          | `device list`（非 sandbox）  | 网络错误，非 0                  |
| NEG-02 | 错误 site 别名         | —             | `auth add --site foo`        | 明确非法 site                   |
| NEG-03 | JSON 语法错误          | —             | `--from-json` 坏文件         | 非 0，可读错误                  |
| NEG-04 | 无权限资源             | 他人私有 id   | view/update/delete           | API 错误透出                    |
| NEG-04b | 只读 Access Key 写操作 | Key level=read | `device unbind` / `group delete` / app 写 | 友好提示缺写权限 + 引导 `auth init`；保留 code=11201 |
| NEG-05 | 并发写同一 app         | 双进程 update | 同时改                       | 不崩溃；最终一致性可接受        |
| NEG-06 | 配置文件权限           | Unix          | 新建 sensecraft.json         | 建议 0600（若实现有保证则断言） |
| NEG-07 | sandbox 不污染生产 API | 生产凭证在旁  | 全程 `--sandbox`             | 无外网请求（抓包/代理验证）     |
| NEG-08 | 命令拼写               | —             | `sensecraft-cli deivce list` | cobra 未知命令提示              |

---

## 十四、自动化映射（已有 vs 缺口）

| 区域                   | 已有覆盖                         | 本用例优先补测                                   |
| ---------------------- | -------------------------------- | ------------------------------------------------ | ----------------- | ------------------------ |
| SenseCAP cmd           | `cmd/device                      | group                                            | data/\*\_test.go` | 真实站点抽测 bind/unbind |
| App write 边界         | `open_api_app_write_test.go`     | 真实 create 保 id、稀疏 rename、blind-retry 流程 |
| Model                  | validation + write tests         | Pro `--target`、publish status                   |
| Auth / profile     | `internal/config` 单测           | **交互** init/add 人工；AUTH-A08 编译回归        |
| Cloud              | cloud sandbox                    | 有设备再跑 deploy/order                          |
| npm                    | `packages/cli/scripts/*.test.js` | 全局 install 路径与 Skills 存在性                |
| 冒烟脚本               | smoke-\*.sh                      | 发版门禁 S1–S3                                   |

---

## 十五、建议执行顺序

1. **门禁**：S1–S3（`go test` + 双 smoke）
2. **鉴权**：AUTH-I\* → AUTH-A\*；（有条件再 AUTH-N\*）
3. **SenseCAP 只读** → 可控写（测试组/测试设备）
4. **AI 只读** app/model/constants
5. **AI 写**：create 保 id → 稀疏 update → publish/unpublish → delete 清理
6. **file/model upload 关联**（`--id`/`--model-id`/`--app-id --yes`）→ 新建才走 `--unlinked` + `model create`；裸 upload 必须失败
7. **cloud**（有设备再跑 L2）
8. **跨域 X-08 + 消歧抽查**

清理：删除本轮创建的 QA App / Model / Group；勿留隐私=2 垃圾数据在生产组织。

---

## 附录 A — 常用命令速查

```bash
# 配置隔离
export SENSECRAFT_CONFIG_FILE=/tmp/sensecraft-qa.json

# 鉴权
sensecraft-cli auth info
sensecraft-cli auth add --key-id … --access-key … --site Global
sensecraft-cli auth init --kind sensecap  # L3 交互

# SenseCAP
sensecraft-cli device list
sensecraft-cli group list
sensecraft-cli data latest --eui <eui>

# AI
sensecraft-cli app list --public
sensecraft-cli app create --name "QA-$(date +%s)"
sensecraft-cli model list --mine
sensecraft-cli recamera check --model-id <id> --sandbox
sensecraft-cli recamera open --model-id <id> --print-only --sandbox
sensecraft-cli model constants
sensecraft-cli file upload --path ./f.bin --type 2

# Cloud
sensecraft-cli cloud device list --uniform-type 26
```

## 附录 B — 文档索引

| 路径 | 用途 |
|------|------|
| `skills/sensecraft-data/SKILL.md` / `commands.md` | SenseCAP PaaS |
| `skills/sensecraft-ai/SKILL.md` | 路由与分级 |
| `skills/sensecraft-apps/` | 应用广场 |
| `skills/sensecraft-models/` | 模型库 |
| `skills/sensecraft-recamera/` | 经典 reCamera 40 |
| `skills/sensecraft-devices-cloud/` | Jetson 云部署 |
| `skills/sensecraft-files/` | 文件上传 |
| `ARCHITECTURE.md` | 架构 |
| `script/smoke-*.sh` | 沙箱冒烟 |
