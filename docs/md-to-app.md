# 把用户 Markdown 写成应用

给人和 Agent 用。网页编辑器是自研块编辑器（`sensecraft-app/src/views/app/edit`），不是「把整篇 Markdown 存进去再渲染」。有 Markdown 时直接：

```bash
sensecraft-cli app create --from-md ./README.md
sensecraft-cli app update --id <app_id> --from-md ./README.md --yes
```

命令内部会先编成编辑器 HTML 再提交。文首 `# 标题` 在 `--name` 和 JSON 都没写 `app_name` 时填上；下一段在 JSON 没写 `excerpt` 时填上，超过 300 字会截断并写进 `warnings`。stdout 带 `pending_uploads` 和 `warnings`。本地图片、视频、附件不会写进正文。

只要看编译结果、先不创建应用，仍用 `sensecraft-cli app doc compile --from-md 文件.md`（不联网、不上传）。

工具块按第 7 节写进 `--from-json` 的 `deployment.blocks`。和 `--from-md` 一起用时，这些块接在编译出的正文后面，不要把同一段正文再写一遍。场景、封面、物料不会自动填。没有一条会直接发布的 `app import-md`。

`--from-json` 不会把 Markdown 重新编译。正文里还留着 `**粗体**`、`` `代码` ``、行首 `#`、行首列表符时会拒绝。`code` 块是源码原文，不在此列。JSON 里已经是 HTML 的文本块会补齐编辑器缺的属性：整段只有一个远程视频地址或裸 `<video>` 时收成播放器；文件链接补上下载标记；普通链接补上 `target` 和样式；图片补上 `draggable="false"` 和默认宽度。句子中间的视频地址保持原样。已经带齐这些属性的标签不再改写。

块类型和工具块字段以编辑器为准。工具块每个字段怎么填，见 `skills/sensecraft-cli/references/tool-blocks.md`。资料字段和物料见 `skills/sensecraft-cli/references/app-fields.md`。

---

## 1. 编辑器实际存什么

一篇应用文档只有一个数组：`deployment.blocks`。顺序就是阅读顺序。没有嵌套 children。

```text
[ 介绍区文本块... ] [ step ][ 这段里的文本或工具块... ] [ step ][ ... ]
```

| 区域 | 规则 |
| --- | --- |
| 介绍 | 第一个 `type === "step"` **之前**的所有块 |
| 一个部署步骤 | 一个 `step`，加上它后面、下一个 `step` 之前的块 |
| 没有 `step` | 整篇都算介绍，详情页不会出现「第 N 步」 |

预览按 `type` 自己包标签（段落包 `<p>`，标题包 `<h1>`…）。`content` 里再写 `# 标题`、`**粗体**`、`- 列表`，会原样显示成字符，不会变成标题或列表。行内格式必须已经是 HTML。

旧字段 `introduction`（一大段纯文本）不要再写。编辑器可能把它再插进介绍区，和 blocks 重复。

`deployment.myModels` 是编辑器从 `deploy-ai-model` 里抽出来的，发布前网页会自己填。CLI 不要手写这个数组。

### 编译命令

```bash
sensecraft-cli app doc compile --from-md ./README.md
sensecraft-cli app create --from-md ./README.md --from-json ./scene.json
```

`app doc compile` 不需要 Access Key。stdout 里的 `deployment` 也可以放进 `app create` / `app update` 的 JSON。直接 `--from-md` 时不用再抄一遍。另外三个字段：

| 字段 | 含义 |
| --- | --- |
| `suggested_app_name` | 文首 `# 标题`。不做成 `h1` 块 |
| `suggested_excerpt` | 紧跟这个标题的第一段。不进 blocks。超过 300 字要自己截 |
| `pending_uploads` | 本地图片、视频、附件。先 `file upload`，再把返回的 https 地址写进 `src` / `href`。相对路径不会写进 `content` |

和编辑器工具栏的对应关系：

| 编辑器 | Markdown |
| --- | --- |
| 粗体 / 斜体 / 删除线 / 行内代码 | `**`、`*`、`~~`、反引号。写成 `<strong>` `<em>` `<s>` `<code>` |
| 下划线 | `<u>…</u>`（Markdown 没有下划线） |
| 字色 / 底色 | `<span style="color:#ef4444">`、`background-color`。`--from-md` 只保留编辑器色板里的颜色，其它颜色会去掉。编辑器保存时不按色板删：合法的 hex、rgb、hsl、颜色名都会留下 |
| H1–H6 | `#` 到 `######`。文首那个 `#` 是应用名，不是块 |
| 无序 / 有序 / 待办 | `-`、`1.`、`- [ ]` / `- [x]`。每行一块 |
| 引用 / 代码 / 分割线 | `>`、围栏、`---` |
| 对齐 | 标题行末 `{align=center}`，或上一行 `<!-- align: center -->`。取值 `left` / `center` / `right`，只作用在段落和标题上 |
| 图片 | `![说明](https://…)`，写成带默认宽度样式的 `<img draggable="false">`。本地路径进 `pending_uploads` |
| 视频 | 单独一段远程地址，或 `<video src="https://…">`。直链 mp4/webm/mov 等写成 `<video controls>`；YouTube、TikTok、抖音、Bilibili、Vimeo、优酷、爱奇艺写成 `video-embed-wrapper` 里的 iframe。本地视频进 `pending_uploads` |
| 链接 / 文件 | `[文字](https://…)` 带 `target="_blank"`。pdf、zip、bin、hex、json 等后缀再加 `doc-file-download-link` 和 `download`。本地文件进 `pending_uploads` |
| 步骤 | `## 步骤：烧录固件`、`## Step: Flash`、`## Step 2. Connect`、`### 1. 部署模型`、`## 第 1 步：烧录`。`title` 只留动作。`## 介绍`、`## Introduction` 这个标题丢掉，下面的段落保留。`## 简介`、`## Overview` 仍是普通标题 |
| 显示预览 | 编辑器界面，CLI 不实现 |

编译器不会插入 `flash-firmware` 等工具块。烧录、部署模型仍按第 7 节手写。

---

## 2. 一篇应用在 CLI 里分成四块

| 块 | JSON | 从用户 Markdown 的哪里来 | 命令 |
| --- | --- | --- | --- |
| 资料 | 顶层 `app_name` `excerpt` `scene` `allow_clone` `allow_commerce` `license` `author` `images` `pic_url` | 标题、第一段、文首元数据 | `app create` / `app update` |
| 文档 | `deployment.blocks` | 正文 | 同上，和资料可以一次 `--from-json` |
| 物料 | `materials` | 「硬件 / BOM / 物料」那一节 | 可放进同一次 JSON，或之后 `app bom set` |
| 上架 | 不在 JSON 里 | 用户明确说要发布 | `app publish` |

创建成功只是私有草稿（`be_public = 0`）。广场上看不到，直到 `app publish`（提交审核，`privacy = 2`）。不要在 JSON 里写 `be_public` 或 `privacy`。不要传 `privacy = 1`。

---

## 3. 资料字段：从 Markdown 里抽，不要塞进正文块

| 字段 | 规则 | 从 Markdown 怎么取 |
| --- | --- | --- |
| `app_name` | 必填，去空格后 1–60 字 | 文首唯一的一级标题。没有标题就用文件名，先跟用户确认 |
| `excerpt` | 必填，1–300 字 | 「简介 / Overview」的第一段，或标题下、第一个空行前的那段。不要把步骤、参数表、更新日志写进来 |
| `scene` | 必填，字符串 id | 先跑 `sensecraft-cli model constants`，在 `app_scenarios_array` 里对场景中文名或英文名。对不上就问，不要编 id |
| `allow_clone` | 必填，`0` 或 `1` | 文里写了「允许克隆 / allow clone」用 `1`，写了禁止用 `0`。没写：普通教程用 `1`，含私有固件或密钥的用 `0`，并在提问里说一声 |
| `allow_commerce` | 必填，`0` 或 `1` | 没写就用 `0` |
| `license` | 可空字符串 | 文里的 License 行。没有就 `""` |
| `author` | 建议填 | `sensecraft-cli auth info` 的账号。不要从 Markdown 署名乱填成别人 |
| `images` | 发布前至少 1 张，最多 16，JPG/GIF/PNG/WebP，每张 ≤ 5 MB | 文中第一张适合当封面的图（优先 4:3）。本地路径先上传。第一张同时写进 `pic_url` |
| `activity_id` | 可选正整数 | 创建时先按 [app-fields.md 的活动选择规则](../skills/sensecraft-cli/references/app-fields.md#activity-selection-during-app-authoring)运行 `app activity list`，阅读候选活动的 `guide_url`，结合项目事实、时间和参赛要求选择合适活动或不参加；不确定就给用户候选项和「不参加」选项。尊重已有选择。提交前复查活动仍开放，并明确写入已决定的 `allow_clone`；省略它时 CLI 可能补为 `1`。已有应用的无关更新不要重复提交历史活动 ID |
| `cancel_activity_id` | 可选正整数 | 只有用户说退出某场已参加的活动才加。CLI 原样把正整数交给 `update_app`，不改 `allow_clone`。不要和同一个 `activity_id` 一起写 |

封面和文中插图是两套：

- 封面进 `images[]`，广场卡片用。
- 文中的图留在某个块的 `content` 里，用上传后的 `https` 地址。

本地图上传：

```bash
sensecraft-cli file upload --path ./cover.png --type 1 --for-app
sensecraft-cli file upload --path ./wiring.png --type 1 --for-app
```

stdout 里的 `file_url` 才是能写进 JSON 的地址。Markdown 里的 `./images/a.png` 不能直接当 `src`。

---

## 4. 正文：Markdown 语法 → 块

下面这张表以编辑器粘贴解析（`pasteMarkdownUtils.js`）为准。粘贴不会产生 `step`，也不会把围栏语言写进 `config.lang`，也不认行首 `+`。这三件是 `--from-md` 编译才做的，表里单独标出。

空行结束当前段落。连续的非空普通行合并成**一个** `paragraph`，行与行之间变成空格，不是 `<br>`。

| 用户 Markdown | `type` | `content` 里写什么 |
| --- | --- | --- |
| 普通段落 | `paragraph` | 行内 HTML，见第 5 节。不要带外层 `<p>` |
| `#` … `######` 且**不是**步骤 | `h1` … `h6` | 只放标题文字的行内 HTML。`#` 本身不要写入 |
| 用来分步骤的标题 | `step` | 见第 6 节。不要做成 `h2` |
| `-` / `*` 开头的一行 | `bullet` | 去掉列表符之后的行内 HTML。每一行一个块。粘贴不认 `+`；`--from-md` 认 |
| `1.` `2.` 这种有序列表 | `number` | 去掉序号之后的行内 HTML。每一行一个块。序号编辑器会自己画，不要写进 content |
| `- [ ]` / `- [x]` | `todo` | 未勾选：行内 HTML，**不要**写 `[ ]`。已勾选：`[x] ` 加行内 HTML（方括号后有一个空格） |
| `>` 连续几行 | `quote` | 这几行用空格拼成一段行内 HTML |
| ` ``` ` 围栏 | `code` | **纯文本**，保留换行。不要包 `<pre>`。粘贴不写 `config.lang`。`--from-md` 把围栏语言写进 `config.lang`（可省略） |
| `---` / `***` / `___` 单独成行 | `divider` | `""` |
| `![说明](url)` | 不单独成块 | 放进所在段落：`<img src="url" alt="说明" />` |
| `[文字](url)` | 不单独成块 | `<a href="url">文字</a>` |
| 表格 | **没有表格块** | `--from-md` 不会改写，只警告并把竖线开头的行留成段落。要进编辑器就改成一组 `bullet`，或整段放进一个 `code` |
| HTML 表格、Word 粘贴 | 不要原样抄 | 抽成段落和列表 |

每个块都要有字符串 `id`，同一篇里不重复。推荐 `p1`、`h1`、`s1`、`b1`、`c1`、`ff1`。编辑器自己生成的是 `tool_<时间>_<随机>`，两种都可以。

文本块建议带 `"indent": 0`。编辑器缩进按钮最多到 `2`，手写 JSON 用 `0`、`1`、`2`。介绍区用 `0`。`--from-md` 也最多写到 `2`，更深的列表缩进会收成 `2`。

`align` 只对 `paragraph` 和 `h1`–`h6` 有意义，取值 `left` / `center` / `right`。不写就是默认左对齐。

---

## 5. 行内 HTML：只留这些标签

编辑器保存时会洗掉其它标签（`sanitizeInlineHtml`）。转换时就按洗完的结果写。

| Markdown | 写入 `content` |
| --- | --- |
| `**粗体**` 或 `__粗体__` | `<strong>粗体</strong>` |
| `*斜体*` 或 `_斜体_` | `<em>斜体</em>` |
| `` `代码` `` | `<code>代码</code>` |
| `~~删除~~` | `<s>删除</s>` |
| `[看这里](https://example.com)` | `<a href="https://example.com" target="_blank" rel="noopener noreferrer">看这里</a>`，并带编辑器的链接颜色 |
| `[固件](https://…/a.bin)` | 同上，再加 `download`、`class="doc-file-download-link"`、`data-file-download="1"`。pdf、zip、hex、json 等同样处理 |
| `![接线](https://…/a.png)` | `<img src="…" alt="接线" draggable="false" style="max-width: 100%; width: 100%; …" />` |
| 下划线（Markdown 没有） | `<u>…</u>` |
| 换行（很少需要） | `<br>` |

允许的标签只有：`b` `strong` `i` `em` `u` `s` `strike` `code` `br` `img` `video` `a` `iframe`，以及包 YouTube 等嵌入用的 `<div class="video-embed-wrapper">`。`<span>` 只保留 `color` / `background-color`。编辑器不按色板过滤，合法颜色都会留下；`--from-md` 编译时只留色板里的值。`<script>`、`<table>`、`<div>`（除视频容器）、样式表都会被丢掉。

单独成段的视频地址，以及整段只有一个 `<video src="https://…">`，会收成编辑器「插入视频」保存的 HTML。直链文件：

```html
<video src="https://…/demo.mp4" controls style="max-width: 100%; height: auto; display: block; border-radius: 4px;"></video>
```

YouTube、TikTok、抖音、Bilibili、Vimeo、优酷、爱奇艺收成 `<div class="video-embed-wrapper">` 包住的 iframe，播放器地址与编辑器 `videoPlatforms.js` 一致。不要手写其它域名的 iframe。写在句子中间的平台地址仍是普通链接，不会变成播放器。

指向 `.pdf` `.zip` `.bin` `.hex` `.json` 等文件的链接会带下载块标记。固件若要给用户烧录，优先用 `flash-firmware` 工具块，不要只丢一个下载链接。

`code` 块的 `content` 是源码原文。`<` `>` 不要转成实体，也不要写成 HTML。可选：

```json
{ "id": "c1", "type": "code", "content": "void setup() {}\n", "indent": 0, "config": { "lang": "cpp" } }
```

---

## 6. 什么时候用 `step`

详情页的「第 1 步 / Step 1」只来自 `type === "step"`。`h2` 只是大字，不会变成步骤。

把下面这些标题收成 `step`，标题文字放进 `title`，`content` 写成和 `title` 相同的字符串，`expanded: true`，`indent: 0`：

- `## 步骤：烧录固件`、`## Step 2. Connect`、`### 1. 部署模型`
- 文中明确的「使用步骤」列表，每一项一个 `step`

`title` 只写动作，不要自己加 `Step 1:` 或 `第 1 步`。网页会加序号。上面例子的 `title` 是 `烧录固件`、`Connect`、`部署模型`。

这个 `step` **后面**、下一个 `step` 之前，放该步的段落、列表、代码、工具块。不要把整步说明写进 `step.content`。

介绍区（第一个 `step` 之前）控制在一两段。不要把目录、故障排除大表、和 excerpt 重复的句子再铺一遍。一个应用通常 3–5 个步骤。某一步用户在网页上做不到（没有对应工具块），就不要写成步骤，改成说明或在提问里告诉用户。

`## 介绍` 和 `## Introduction`（不区分大小写）这个标题丢掉，下面的段落保留。`## 简介`、`## Overview` 会变成普通标题，不会丢掉。

文首那个当应用名的 `# 标题` 不要再做成 `h1` 块。

---

## 7. 工具块：正文里出现「让用户在页面上做事」时插入

工具块和文本块一样放在数组里，一般放在某一步的说明段落后面。CLI 只写入配置。真正烧录、部署、装容器是网页上的按钮，CLI 不要声称已经烧进去。

| 正文在说 | `type` | 配置放哪 |
| --- | --- | --- |
| 把固件烧进板子 | `flash-firmware` | 只放在 **`config`** 里。详情页只读 `config`。SKU、地址见 `skills/sensecraft-cli/references/tool-blocks.md` |
| 部署视觉 / 声音模型 | `deploy-ai-model` | 放在 `config` 里。两个入口是同一个 `type`，用 `config.deploy_tool_entry`：`vision` 或 `sound`。声音会排除 reCamera Pro（`uniform_type` `53`） |
| 异常振动 | `ai-vibration` | 放在 `config` 里 |
| 边缘容器应用 | `deploy-container-app` | 放在 `config` 里 |
| Node-RED 流程 | `node-red` | 放在 `config` 里。`config.flow_data`、`config.model_data` 是对象，不要写成字符串。保存时网页把 `flow_data` `JSON.stringify` 成应用顶层的 `flow_data` 字符串，并把 `model_data` 抄到应用顶层 |
| Aily Blockly 公开项目 | `aily-blockly` | 放在 `config` 里 |

没有对应文件时，块仍然留着，缺的 URL 先空着并在回复里标明「待用户提供」，不要删掉这一步。

固件先上传再写入 URL：

```bash
sensecraft-cli file upload --path ./firmware.bin --type 4 --checksum
```

`firmware_size` 用字符串。`device_id` 是物料 SKU（如 `"100017057"`），不是 `uniform_type`。

物料清单（Markdown 表格或「硬件」小节）不要写进 blocks，写进 `materials`：

- 能在 `sensecraft-cli app bom search --name "XIAO ESP32S3"` 里对上的：带 `id`（SKU）、`name`、`quantity`（字符串）、`note`、`buy_url`。不要写 `price` / `pic_url`，网页按 SKU 自己查。
- 对不上的：只有 `name`、`quantity`，可选 `note`、`buy_url`，**不要** `id`。

普通杜邦线、电脑、螺丝刀不要进物料。设备本身和文档点名的配件要进。

---

## 8. 对照例子

用户给出的 Markdown：

```markdown
# 罐盖计数

传送带上用 XIAO ESP32S3 Sense 数瓶盖。适合产线巡检。

## 介绍

把摄像头对准瓶盖，打开应用后即可看到计数。

## 步骤：烧录固件

用 USB 连接 XIAO ESP32S3 Sense，然后烧录。

- 固件：`firmware.bin`
- 烧录时不要断开 USB

## 步骤：看计数

烧录完成后重新上电，屏幕上会显示当前数量。

## 硬件

- Seeed Studio XIAO ESP32S3 Sense × 1
```

转成的 `--from-json`（封面 URL、固件 URL、场景 id、SKU 按实际上传和检索结果替换）：

```json
{
  "app_name": "罐盖计数",
  "excerpt": "传送带上用 XIAO ESP32S3 Sense 数瓶盖。适合产线巡检。",
  "scene": "7",
  "allow_clone": 1,
  "allow_commerce": 0,
  "license": "",
  "author": "maker@example.com",
  "images": [
    "https://files.example.com/cover.png"
  ],
  "pic_url": "https://files.example.com/cover.png",
  "deployment": {
    "blocks": [
      {
        "id": "p1",
        "type": "paragraph",
        "content": "把摄像头对准瓶盖，打开应用后即可看到计数。",
        "indent": 0
      },
      {
        "id": "s1",
        "type": "step",
        "title": "烧录固件",
        "content": "烧录固件",
        "expanded": true,
        "indent": 0
      },
      {
        "id": "p2",
        "type": "paragraph",
        "content": "用 USB 连接 XIAO ESP32S3 Sense，然后烧录。",
        "indent": 0
      },
      {
        "id": "b1",
        "type": "bullet",
        "content": "固件：<code>firmware.bin</code>",
        "indent": 0
      },
      {
        "id": "b2",
        "type": "bullet",
        "content": "烧录时不要断开 USB",
        "indent": 0
      },
      {
        "id": "ff1",
        "type": "flash-firmware",
        "config": {
          "device_id": "113991115",
          "device_type": "32",
          "flash_mode": "bin",
          "firmware_url": "https://files.example.com/firmware.bin",
          "firmware_name": "firmware.bin",
          "firmware_size": "412316",
          "flash_address": "0x0",
          "muse_gadget": false,
          "full_flash": false
        }
      },
      {
        "id": "s2",
        "type": "step",
        "title": "看计数",
        "content": "看计数",
        "expanded": true,
        "indent": 0
      },
      {
        "id": "p3",
        "type": "paragraph",
        "content": "烧录完成后重新上电，屏幕上会显示当前数量。",
        "indent": 0
      }
    ]
  },
  "materials": [
    {
      "id": "113991115",
      "name": "Seeed Studio XIAO ESP32S3 Sense",
      "quantity": "1",
      "note": "",
      "buy_url": "https://www.seeedstudio.com/"
    }
  ]
}
```

纯文本、没有标签的 `content` 可以不包 HTML。一旦有强调、链接、图片，就按第 5 节写成标签，不要混用 `**`。

`flash-firmware` 的 `device_type` / `flash_address` 以 `skills/sensecraft-cli/references/tool-blocks.md` 或 `sensecraft-cli app blocks` 里的 `device_defaults` 为准。上面的 `32` 和 `0x0` 只说明字段位置。例子里的 `113991115` 是 XIAO ESP32S3 Sense，不要写成 `113991254`（那是另一块板）。`muse_gadget` 为 `true` 时 `full_flash` 必须是 `false`（CLI/编辑器都会强制），且固件名需为 `muse-gadget.bin`。Muse 白名单设备在 `muse_gadget: false` 且未写 `full_flash` 时，CLI 会默认 `full_flash: true`（盖掉盘内 Muse 分区），**Watcher 除外**（恒为 `false`，不擦全片以保 nvsfactory）；Sticky/Indicator/1.47 显式 `false` 会被强制改回 `true`；E100x 显式 `false` 会保留。非 Muse 设备这两个字段都不要写成 `true`。空的 `flash_address` 会按设备补默认值；固定地址板写错会被纠正：E100x→`0x90000`、Wio→`0x4000`、Watcher/Sticky/Indicator/1.47→`0x0`（禁止 `0x10000`）。BOM `materials[].name` 可识别电子纸（→`0x90000`）与 IPS（→`0x0`）。普通 XIAO（C3/C6/Sense）仍可用 `0x0` / `0x10000`；未知 SKU 不会因 URL 含 `application` 自动填 `0x10000`。Indicator 别名（D1S `114993069` 等）与 Watcher 外壳 SKU 也支持 Muse。

---

## 9. 用 CLI 写上去

先读文档，列出将要创建的名称、步骤标题、物料、还缺的封面或固件。缺场景或封面时先问。用户确认后再创建。`app create` 本身不要 `--yes`。之后的 `app update` / `app bom set` / `app publish` 要另一次确认再加 `--yes`。

```bash
sensecraft-cli auth info
sensecraft-cli model constants
sensecraft-cli app bom search --name "XIAO ESP32S3 Sense"
sensecraft-cli file upload --path ./cover.png --type 1 --for-app
sensecraft-cli file upload --path ./firmware.bin --type 4 --checksum

sensecraft-cli app create --from-json ./payload.json
sensecraft-cli app view --id <app_id> --sketch
```

`app create` 的 stdout 必须有 `app_id`。没有就先 `app list --mine --name "罐盖计数"`，不要再 create 一次（重名会再造一个）。

更新已有应用（会改草稿，广场上的已发布版要等再次 publish 才变）：

```bash
sensecraft-cli app update --id <app_id> --from-json ./payload.json --yes
sensecraft-cli app view --id <app_id> --sketch
```

只改物料：

```bash
sensecraft-cli app bom set --id <app_id> --from-json ./materials.json --yes
```

`materials.json` 是 `{ "materials": [ ... ] }`。

发布（审核中会锁编辑）。先跟用户读免责声明，用户同意后才加 `--agree-declaration`：

```bash
sensecraft-cli app publish --id <app_id> --agree-declaration --yes
```

发布前网页和 CLI 都会卡这四项：名称、简介、场景、至少一张封面。缺一项就先补，不要发。

成功后告诉用户：应用仍是「审核中」，不是已经出现在广场上，也不是还能接着改的草稿。把 stdout 里的 `detail_url` 原样给用户，形式是 `/view-application/{id}-{名称}`，不要再加 `from`、`section`、`state`。作者登录后打开的是详情页。不要改成 `/application/detail/<id>`。不要给编辑链接：审核中没有 `console_url`，也不要自己拼 `/application/edit?id=`。打开编辑页会报 `The application must not allow editing`。要恢复编辑用 `app withdraw`，不要用 `app unpublish`（下架只用于已经公开的应用）。有 `web_link` 的应用卡片会跳外链，那种不是这个编辑器文档。

结构是否写对，可以在无网络时对照：

```bash
sensecraft-cli app fields
sensecraft-cli app blocks
```

这两个命令不加 `--sandbox`。
