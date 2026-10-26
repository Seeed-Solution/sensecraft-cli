# 应用广场能力同步到 CLI

给 `sensecraft-cli` 开发。网页端是 `sensecraft-app`（应用广场 / 我的应用 / 详情 / 编辑）。本文只列 **CLI 还没对齐、需要改代码** 的功能。

用户拿 Markdown 来生成应用：`app create --from-md` / `app update --from-md` 会先编成编辑器块再写入。`app doc compile --from-md` 只预览、不创建。不要再做一条会直接发布的 `import-md`。转换规则见 [md-to-app.md](md-to-app.md)。

对照日期：2026-10-22。网页列表热门排序已上线（`develop` `36ed158`）。同日复核：点赞后列表不再立刻重排（`5985961`）；应用站已独立到 `app.seeed.cc` / `test-app.seeed.cc`，详情不再走 `/ai/application/detail`。

## 已经对齐、不要重做

这些命令保持现状，本文不改它们的主路径：

| 能力 | CLI | 网页接口 |
| --- | --- | --- |
| 公开 / 我的应用列表的名称、场景、设备、分页 | `app list --public\|--mine --name --scene --uniform-type --page --length` | `GET /flow/app/v2/list_app` |
| 看已发布 / 草稿 | `app view --id --sketch` | `GET /flow/app/v3/view_app` |
| 按名称创建、整包 JSON 创建 | `app create` | `POST /flow/app/v3/create_app` |
| 改资料、文档块、封面、参加活动 | `app update --activity-id` | `POST /flow/app/v3/update_app` |
| 物料搜索 / 读取 / 覆盖 | `app bom search\|list\|set` | `get_production_options` + `update_app` |
| 提交审核 / 已公开后私有化 / 删除私有应用 | `app publish` / `app unpublish` / `app delete` | `POST /flow/app/v3/edit_privacy_app`（`privacy` 仅 `2` 或 `0`）、`POST /flow/app/v2/delete_app` |

约定不变：

- 成功只往 stdout 打业务 `data` 的缩进 JSON；失败走 stderr，退出码非 0。
- 写操作沿用现有 L2：先让调用方确认，再带 `--yes`。`app create` 仍然不要 `--yes`。
- **永远不要传 `privacy=1`**。`1` 是运营审核通过后的公开态。
- 改完命令后，同步 `skills/sensecraft-apps/commands.md` 和 `skills/sensecraft-apps/SKILL.md`。Agent 只看技能文档，不看本文。

## 明确不做

| 项 | 原因 |
| --- | --- |
| 烧录固件、USB 部署、容器安装 | 网页详情里的执行动作。CLI 只写 `deployment.blocks`，不在设备上跑。 |
| 分享卡片 / 复制图片 | 纯前端。创建成功已有 `detail_url`。不要做 canvas 分享。 |
| 硬件 BOM 筛选（`bom` 查询参数） | 网页开关 `APP_STORE_HARDWARE_FILTER_ENABLED` 仍是 `false`，后端未接。两边都先不做。 |
| 退出已参加的活动 | 网页取消勾选也不会提交。不传 `activity_id` 时服务端保留原标签。不要发明「清空活动」参数。 |
| 落地页热度分 | 网页首页「热门应用」仍是前端给一页结果打分，不是广场热门。CLI 对齐的是广场列表，不是首页打分。 |
| 列表 / 详情机翻 | `stampListI18nSource` 只改展示文案。CLI 打印接口原文。 |
| 搜索框里的「设备、作者」 | 请求仍只有 `app_name`。`--name` 已对齐，不要加作者或设备搜索参数。 |
| 点赞后冻结卡片顺序 | 见 1.2。纯列表 UI，CLI 每次都是新请求。 |
| 列表页发布声明的 vercel 链接 | 编辑页声明是 `{控制台}/legal/{lang}/declaration`。CLI 已有的 `--agree-declaration` 用这条，不要改成 `sensecraft-ai-app-creation.vercel.app`。列表页那条写死链接是网页残留。 |

---

## 1. 列表：热门优先活动、排序、竞赛、点赞/收藏

网页：`sensecraft-app/src/views/app/components/list.vue` 的 `buildAppListParams`，广场页 `src/views/app/index.vue`。

接口不变：`GET /flow/app/v2/list_app`。

### 1.1 热门：`priority_activity_id`

广场默认排序是「热门」。请求会带：

```
GET /flow/app/v2/list_app?be_public=1&sort_by=popular&priority_activity_id=<活动 id>&page=1&length=12
```

`priority_activity_id` 是进行中活动列表的 **第一场** `activity_id`（`GET /activity/list` 不传 `time`，服务端按开始时间从早到晚）。没有进行中的活动时，网页不传这个参数。

只在热门时带。切到「最新」「点赞量」必须重拉，且 **不能** 再带 `priority_activity_id`，否则最新/点赞会排在「活动优先」的那一页上。

CLI 现状：`buildAppListQuery`（`internal/service/open_api_app.go`）没有该字段。`app list --public` 的顺序和广场热门不一致。

要做：

- 新增 `app list --priority-activity-id <正整数>`，原样放到 query `priority_activity_id`。
- 新增 `app list --sort popular`。当 `--sort popular` 且用户没写 `--priority-activity-id` 时，CLI 自己请求 `GET /activity/list`，取第一场的 `activity_id` 填上。列表为空或请求失败：不传该参数，列表仍成功（与网页一致）。
- `--sort newest`、`--sort likes` 只传 `sort_by`，禁止自动补 `priority_activity_id`。
- 不传 `--sort` 时保持今天的行为（不传 `sort_by`，也不要偷偷加活动优先），避免把「我的应用」默认列表改掉。广场热门必须显式 `--public --sort popular`。

活动 id 规则与现有 `app update --activity-id` 相同：正整数。`0`、负数、非整数在发请求前拒绝。

### 1.2 `sort_by`

网页三个排序键，原样作为 `sort_by`：

| 网页 | `sort_by` | 说明 |
| --- | --- | --- |
| 热门 | `popular` | 顺序由服务端按 `priority_activity_id` 决定，前端不再重排 |
| 最新 | `newest` | 网页在重新拉列表后，对已加载页按 `created` 降序排。服务端是否已实现 `sort_by=newest` 未确认，CLI 先把参数送出去 |
| 点赞量 | `likes` | 同上，重新拉列表后按 `like_num` 降序排已加载页 |

「最新」「点赞量」下，网页会记住当前卡片顺序：原地改 `like_num` 不挪位，只有重新请求（换排序、筛选、刷新、往下加载的新尾巴）才重排。这是 `list.vue` 的展示层。CLI 每次调用都是新请求，送出 `sort_by` 即可，不要做冻结。

建议 flag：`--sort popular|newest|likes`。不要用帮助文案里的 `hot`。

### 1.3 竞赛筛选：`tag_id`

网页侧栏「竞赛」选中后：

```
GET /flow/app/v2/list_app?...&tag_id=1
```

`tag_id` 取该活动 `tag_ids` 的第一个。Muse Gadget 目前是 `"1"`。这是筛选，不是排序；可以和热门同时存在。

要做：`app list --tag-id <id>` → query `tag_id`。空字符串不要传。

### 1.4 我的点赞 / 我的收藏：`mark_type` 取值写错了

网页 `src/views/common/likeComponent.vue`：

| 页签 | query `mark_type` |
| --- | --- |
| 我的点赞 | `1` |
| 我的收藏 | `0` |

同时仍是公开列表：`be_public=1`（组件把 `currentTab` 设为 `public`）。

CLI 已有 `--mark-type`，但帮助和 `skills/sensecraft-apps/commands.md` 写成 `like` / `star`。网页从不传这两个字符串。

要做：

- 保留 `--mark-type`，帮助改为：`1` = 我的点赞，`0` = 我的收藏。不要再写 `like`/`star`。
- 可选别名，二选一实现即可，最终发出去的值必须是 `1` 或 `0`：
  - `--likes` → `mark_type=1` 且默认加上 `be_public=1`
  - `--collections` → `mark_type=0` 且默认加上 `be_public=1`
- `--likes` / `--collections` 与 `--mine` 互斥。

示例：

```bash
sensecraft-cli app list --public --sort popular
sensecraft-cli app list --public --sort popular --tag-id 1
sensecraft-cli app list --public --sort newest
sensecraft-cli app list --public --likes
sensecraft-cli app list --public --collections
sensecraft-cli app list --public --priority-activity-id 1 --sort popular
```

### 1.5 验收

- `--sandbox` 的样例 JSON 里能看出 query 意图（至少不要在 sandbox 里忽略新 flag）。
- `go test` 覆盖：热门自动填活动 id、没有活动时不填、newest/likes 不带 `priority_activity_id`、`--tag-id`、`--likes`/`--collections` 发出 `1`/`0`。
- 用真实环境对一下广场：`app list --public --sort popular --length 12` 的第一页 `app_id` 顺序与网页热门第一页一致（同一活动进行中时）。

---

## 2. 进行中的活动要能列出来

网页编辑页、广场竞赛筛选项都调用：

```
GET /activity/list
```

不传 `time`，不要求登录。响应 `data.list[]` 里网页实际使用的字段：

| 字段 | 说明 |
| --- | --- |
| `activity_id` | 正整数，常以字符串返回。参加活动、热门优先都用它 |
| `name` | 展示名 |
| `tag_ids` | 字符串 id 数组。广场竞赛筛选只用 **第一个** |
| `start_time` / `end_time` | unix 秒 |

CLI 今天只在命令结束后的 stderr 打一条 Notice（名称、日期、链接），**不打印 `activity_id`**，也没有子命令。调用方无法合法地得到 `--activity-id` 和 `--priority-activity-id`。

要做：`app activity list`（或 `app activities`，实现时定一个，写进技能文档）。

- stdout 为 JSON 数组（或 `{ "list": [...] }`，与其它 list 命令的信封保持一致）。
- 每项至少包含 `activity_id`（数字）、`name`、`tag_ids`。
- 过滤规则与网页 `normalizeOngoingActivity` 一致：缺 `activity_id` 或缺 `name` 的行丢掉；`activity_id` 必须是正整数。
- 这个命令是只读，不要 `--yes`。可加 `--sandbox`。
- 不要用 Notice 代替这个命令。Notice 可以继续存在。

活动列表失败时，`app list --sort popular` 的自动填充按 1.1 降级，不要让整个列表失败。

---

## 3. 点赞 / 收藏

网页：`sensecraft-app/src/components/AppStoreCard.vue` 的 `toggleHoard`。

```
POST /flow/app/v2/mark_app
{ "app_id": "<id>", "op_type": <number> }
```

| `op_type` | 含义 |
| --- | --- |
| `1` | 点赞 |
| `0` | 取消点赞 |
| `3` | 收藏 |
| `2` | 取消收藏 |
| `4` | 记录一次部署 / 下载。见 3.1，不要和点赞、收藏混成一个 flag |

未登录网页会弹登录；CLI 沿用现有 AI Access Key，没有 key 就和今天一样失败。

详情里是否已点赞 / 已收藏，看 `view_app` 的 `like_time`、`follow_time`（非 0 即已标记）。计数是 `like_num`、`follow_num`。

要做：

```bash
sensecraft-cli app like --id <app_id> --yes
sensecraft-cli app unlike --id <app_id> --yes
sensecraft-cli app collect --id <app_id> --yes
sensecraft-cli app uncollect --id <app_id> --yes
```

- 四条都是 L2（会改用户自己的标记）。
- 重复点赞 / 重复收藏：与网页一样把对应 `op_type` 再送一次即可，不要在 CLI 里先 view 再决定，除非接口对重复操作返回错误。若实测重复会报错，再改成「已是该状态则退出 0 并打印当前状态」。
- 成功后 stdout 至少包含 `app_id` 和执行的 `op_type`。接口若返回空对象，CLI 自己补上，避免调用方以为没生效。

### 3.1 部署计数：`op_type=4`

网页：`src/views/app/edit/utils/recordAppDeployMark.js`，由公开详情确认开始部署 / 烧录时调用（`recordSensecraftAppDeploy`）。编辑页、`disableDeploy` 不记。

```
POST /flow/app/v2/mark_app
{ "app_id": "<id>", "op_type": 4 }
```

失败被网页吞掉，不打断烧录。同一 `app_id` 在 1500ms 内只记一次，这是前端去重。

CLI 仍然不烧录、不连设备。下载次数是广场数据，要单独记：

```bash
sensecraft-cli app mark-deploy --id <app_id> --yes
```

- L2。只 POST `op_type=4`。
- 不要塞进 `like` / `collect`，也不要在 CLI 里做 1500ms 去重。每次命令记一次。
- 成功 stdout 带 `app_id` 和 `op_type: 4`。接口返回空对象时 CLI 自己补上。
- 计数本身仍从 `view_app` 读（`view_num` 等已有字段）。不要为这个再造查询接口。

---

## 4. 克隆

网页：`list.vue` 的 `cloneAppFn`，编辑页、详情页同一接口。

```
POST /flow/app/v3/clone_app
{ "app_id": "<源 id>", "app_name": "Clone_<源名称前 54 个字符>" }
```

成功时 `data.app_id` 或 `data.id` 是 **新应用** id。网页随即打开新应用的编辑页。源应用 `allow_clone == 0` 时，菜单项禁用。

技能文档把 clone 写成已有能力，但 `cmd/app` 没有这个子命令，`internal/enum/ai_api.go` 也没有该 path。

要做：

```bash
sensecraft-cli app clone --id <源 app_id> --yes
sensecraft-cli app clone --id <源 app_id> --name "新名称" --yes
```

- L2。这是给当前用户新建一个应用。
- 不传 `--name` 时，名称规则与网页一致：`Clone_` + 源 `app_name` 的前 54 个字符（`Clone_` 占 6，总长不超过网页的 60）。源名称用 `app view` 读取。
- 克隆前看源应用 `allow_clone`。值为 `0` 时拒绝，stderr 说明不允许克隆，不要发 POST。
- stdout 必须带新 `app_id`，以及和 `app create` 相同的 `console_url`、`detail_url`。链接格式见第 8 节：`detail_url` 带 `from=user-center&tab=custom`，否则封面看起来是空的，返回也会进旧的「我的应用」。不要再拼 `/ai/application/detail`。
- 不要把源应用的 `be_public` 抄到新应用上。克隆结果按接口返回为准，CLI 不要再调 `publish`。

---

## 5. 撤回审核（不是私有化）

网页详情在 `be_public == 2`（审核中）时，菜单是「撤回审核」：

```
POST /flow/app/v3/callback_sketch
{ "app_id": "<id>" }
```

不传 `privacy`。成功后网页把本地状态改成 `be_public = 0`。

已公开（`be_public == 1`）时，网页走的是另一条：「私有化」= `edit_privacy_app` 且 `privacy=0`。这就是今天的 `app unpublish`，不要改它的请求体。

CLI 现状：`app unpublish` 一律 `privacy=0`。审核中的应用没有 `callback_sketch`。两条路不能合并成一个 flag 默默切换，否则调用方分不清撤回的是审核还是已上架版本。

要做：

```bash
sensecraft-cli app withdraw --id <app_id> --yes
```

- L2。仅所有者。
- 先 `app view --sketch`。`be_public != 2` 时拒绝，并提示：已公开请用 `app unpublish`，私有应用无需撤回。
- `be_public == 2` 时只 POST `callback_sketch`，body 只有 `app_id`。
- `app unpublish` 的帮助里写明：用于已公开应用的私有化，不用于撤回审核。
- 广场列表菜单在 `be_public != 0` 时（含审核中）仍走 **v2** `edit_privacy_app` 且 `privacy=0`。这和详情页不一致。CLI 以详情页为准，继续用 v3，审核中只走 `withdraw`。不要为了「跟列表菜单一样」把两条路合并。

---

## 6. 从竞赛入口创建后自动报名

网页：编辑页 `applyContestEntryIfNeeded`（`edit.vue`）。从竞赛进来时路由带 `activity=<activity_id>`。同时满足下面三条才自动保存一次：

1. 该活动仍在 `GET /activity/list` 的结果里。
2. 应用详情的 `tags` 显示还没参加任何进行中的活动。
3. 把 `activity_id` 和 `allow_clone=1` 一并写入 `update_app`。

CLI 今天要调用方自己先 `app activity list`，再 `app update --activity-id`。可以接受继续分两步，但创建技能容易漏掉 `allow_clone`。

要做（放在已有 `app update` 上，不要新造报名协议）：

- `app update --id <id> --activity-id <id> --yes` 保持现在的语义。
- 文档和 `create-app` 技能补一句：竞赛报名除了 `--activity-id`，还要在同一次 JSON 里带 `"allow_clone": 1`，与网页自动保存一致。若这次 JSON 没写 `allow_clone`，且这次带了 `--activity-id`，CLI 在合并资料字段时把 `allow_clone` 补成 `1`（仅当视图里当前不是 `1`）。已经是 `1` 则不要改。
- 活动不在进行中列表里时，不要发 `update_app`（网页会直接跳过）。错误信息沿用服务端 `25000` 也可以，但最好在 POST 前用活动列表先拦住，避免一次无效写入把同请求里的其它字段一起丢掉（服务端会整单拒绝）。

`app create` 不必增加 `--activity-id`。创建是空壳或整包 JSON，报名仍走之后的 `update`。

---

## 7. 列表默认值（只改帮助，不改默认行为）

`app list` 两个 flag 都不传时，不带 `be_public`，结果是「我的应用」。网页打开广场默认是公开列表。

不要把 CLI 默认改成 `--public`，那会破坏现有脚本。只把帮助写成：

- 广场：必须 `--public`
- 我的应用：`--mine`，或不传可见性 flag
- 广场热门：`--public --sort popular`

---

## 8. 详情 / 编辑链接改到独立应用站

应用站已从主控制台 `/ai/` 拆出。生产 `https://app.seeed.cc`，测试 `https://test-app.seeed.cc`。路由在站点根，没有 `/ai` 前缀。API 仍走 `sensecraft.seeed.cc` / `test-sensecraft.seeed.cc`，只改页面链接。

CLI 今天的 `console_url` / `detail_url` 仍是：

- `https://sensecraft.seeed.cc/ai/application/edit?id=…`
- `https://sensecraft.seeed.cc/ai/application/detail/<id>?tab=custom&from=my-apps`

网页卡片（`list.vue` 的 `cardTarget`）和分享（`buildAppShareUrl`）实际打开：

| 场景 | URL |
| --- | --- |
| 我的应用（草稿 / 私有，封面在 sketch） | `{origin}/view-application/{app_id}-{slug}?from=user-center&tab=custom` |
| 广场公开详情 | `{origin}/view-application/{app_id}-{slug}?tab=public&from=application-store` |
| 编辑 | `{origin}/application/edit?id={app_id}` |

`origin`：生产 `https://app.seeed.cc`；测试环境 `https://test-app.seeed.cc`。`--sandbox` 里打印同样的路径形状即可，host 跟现有沙箱站点一致。

`slug` 与网页 `formatToTitleCase` 一致：名称先转小写；英文字母、数字、下划线和中日韩统一表意文字保留，其余字符换成 `-`；连续 `-` 合并；去掉首尾 `-`。例：`reTerminal Sticky` → `reterminal-sticky`，路径 `/view-application/11633-reterminal-sticky`。名称为空时路径只到 `{app_id}`，不要补假 slug。

`from=user-center` 留在「我的」链接上，详情才会读 sketch 封面，返回也回到创作中心。不要再用 `from=my-apps`，那个会回到广场里的旧「我的应用」。公开链接不要带 `from=user-center`。

列表项若有非空 `web_link`，卡片不进站内详情，直接打开该外链。这种条目不要再拼 `view-application`。技能文档写明：有 `web_link` 时把外链给用户。

`app create` / `app clone` 成功输出里的 `console_url`、`detail_url` 按上表改。测试里写死 `/ai/application/` 的断言一起改。法律声明、Wiki、AI 控制台首页仍在 `sensecraft.seeed.cc`（测试 `test-sensecraft.seeed.cc`）的 `/legal`、`/wiki`、`/ai/home`，那些不要改到 `app.seeed.cc`。

---

## 建议落地顺序

1. `app activity list`，以及列表的 `--sort` / `--priority-activity-id` / `--tag-id` / `mark_type` 修正。广场热门靠这组才能对齐。
2. `app withdraw`。避免把审核中的应用用 `unpublish` 走错接口。
3. `app like|unlike|collect|uncollect`、`app mark-deploy` 和 `app clone`。
4. 竞赛报名时补 `allow_clone=1`，并改技能文档。
5. `console_url` / `detail_url` 改到应用站（第 8 节）。`create` 和 `clone` 一起改，避免克隆先发出旧链接。

每项都要补 `cmd/app` 测试和技能文档。沙箱模式要能在无网络时跑通新 flag 的参数校验。
