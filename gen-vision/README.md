# gen-vision 选型记录

检索日期：2026-10-18。

约束：云端模型可以写图形代码；图片由本机渲染器生成，不调用云端生图模型。目标成品是图标、海报和精确文字。

下面每个项目都按仓库、介绍、用法记录。数字（star、版本、下载量）来自当天的检索结果，之后会变。

## 1. Takumi

- 仓库：https://github.com/kane50613/takumi
- 文档：https://takumi.kane.tw/docs
- 包：`takumi-js`（npm 2.14.0，2026-09-15）、crate `takumi`（2.16.3，2026-10-16）
- 协议：MIT OR Apache-2.0（以 npm / crates.io 的 SPDX 为准）
- 规模：约 3115 star

不用无头浏览器，把 JSX、HTML、CSS 或纯节点树渲染成 PNG、JPEG、WebP、ICO、SVG，也能出动画帧和分页 PDF（`takumi-pdf`）。布局用 Taffy，文字用 Parley 和 Skrifa，绘制走到 resvg。支持 Flexbox、Grid、block、inline、float，以及 `text-wrap: balance`、按行盒缩放标题的 `text-fit`、`lang` 选择汉字字形。

节点树是普通 JSON，字体和图片是渲染参数，不是写在树上的。云端模型可以交 JSX，也可以交结构化节点。

用法：

```bash
npm install takumi-js
```

```tsx
import { render, renderSvg } from "takumi-js";
import { readFile, writeFile } from "node:fs/promises";

const sourceHan = await readFile("./fonts/SourceHanSans-Regular.otf");

const png = await render(
  <div tw="w-full h-full flex items-center justify-center">
    <h1 tw="text-6xl font-bold">SenseCraft</h1>
  </div>,
  {
    width: 1200,
    height: 630,
    fonts: [{ name: "Source Han Sans", data: sourceHan }],
  },
);

const svg = await renderSvg(<div>SenseCraft</div>, {
  width: 1200,
  height: 630,
  fonts: [{ name: "Source Han Sans", data: sourceHan }],
});

await writeFile("./output.png", png);
await writeFile("./output.svg", svg);
```

它不读系统字体。内置 Geist 只覆盖拉丁字母、字重 300–800。中文必须把字体字节传进去。`googleFonts()` 会访问网络；离线渲染用本地文件。脚本不执行，CSS 少于 Chrome。

## 2. Satori

- 仓库：https://github.com/vercel/satori
- 协议：MPL-2.0
- 规模：约 14095 star
- 演示：https://og-playground.vercel.app

把无状态 JSX 转成 SVG。当前 README 写明布局用 Taffy，覆盖 Flexbox、Grid 和 block，文字用自己的 inline 布局。它不是完整 CSS，也不保证和浏览器像素一致。官方示例里的文字变成 SVG `<path>`，字形在生成时就画死。

用法：

```bash
npm install satori
```

```jsx
import satori from "satori";

const svg = await satori(
  {
    type: "div",
    props: {
      children: "hello, world",
      style: { color: "black", display: "flex" },
    },
  },
  {
    width: 600,
    height: 400,
    fonts: [{ name: "Roboto", data: robotoArrayBuffer, weight: 400, style: "normal" }],
  },
);
```

不需要 JSX 编译器，直接传 `{ type, props }` 即可。`useState`、`useEffect` 不可用。`<link>` 和外部样式不会加载。PNG 要再接一个 SVG 光栅器；官方示例用 Sharp（底层是 librsvg）。

这是「HTML 子集 → SVG」这条路线的起点，也是 `@vercel/og` 的核心。Takumi 的文档把 Satori 列为迁移来源。

## 3. resvg

- 仓库：https://github.com/linebender/resvg
- 协议：Apache-2.0 OR MIT（当前 0.48.1；0.44 及更早是 MPL-2.0）
- 规模：约 4139 star
- crate：0.48.1（2026-08-02）

纯 Rust 的静态 SVG 渲染库，同时提供 C 接口和命令行。解析（`usvg`）和绘制分开，不依赖系统图形库，同一份 SVG 在不同操作系统上的像素一致。不做动画和脚本。文字不走系统排版栈，需要把字体交给它。命令行二进制小于 5MB。

用法：

```bash
# macOS
brew install resvg

resvg input.svg output.png
```

Node 绑定是另一个仓库，协议不同：

- 仓库：https://github.com/thx/resvg-js
- 包：`@resvg/resvg-js`
- 协议：MPL-2.0
- 规模：约 2001 star

```js
const { Resvg } = require("@resvg/resvg-js");

const resvg = new Resvg(svg, {
  fitTo: { mode: "width", value: 1200 },
  font: {
    fontFiles: ["./fonts/SourceHanSerifCN-Light-subset.ttf"],
    loadSystemFonts: false,
  },
});
const png = resvg.render().asPng();
```

海报如果已经由 Takumi 直接出 PNG，就不需要再包一层 resvg。它适合「输入已经是 SVG」：Lucide 图标、手写 SVG、Typst 导出的 SVG。

## 4. Typst

- 仓库：https://github.com/typst/typst
- 协议：Apache-2.0（编译器；网页应用不开源）
- 版本：编译器 0.15.0（`Cargo.toml`）
- 说明：https://typst.app/open-source/

本地编译器，把 `.typ` 标记编译成 PDF、PNG、SVG、HTML。换行、断行、字体回退、多栏由编译器完成。云端模型写的是标记，不是像素。

用法：

```bash
typst compile --font-path ./fonts notes.typ notes.pdf
typst compile --font-path ./fonts --format svg notes.typ notes.svg
typst compile --font-path ./fonts --format png --ppi 144 notes.typ notes.png
typst fonts --font-path ./fonts
```

多页 SVG / PNG 的输出名要带 `{p}` 或 `{0p}`。SVG 为了跨设备一致，把字形嵌成轮廓，文字不能再被选中或复制；要保留真文字用 PDF。PNG 默认 144 PPI，印刷常用 300 或 600。

自由海报要自己写坐标或用模板，例如 [poster-syndrome](https://typst.app/universe/package/poster-syndrome/) 的命名框，或 [typst-pagemaker](https://github.com/maucworks/typst-pagemaker) 的网格。标记语言比 JSX 少见，模型对版式的稳定性取决于模板把坐标固定到什么程度。

## 5. AntV Infographic

- 仓库：https://github.com/antvis/Infographic
- 包：`@antv/infographic` 0.2.20（2026-09-21）
- 协议：MIT
- 文档：https://infographic.antv.vision/

声明式信息图引擎。模型写一套比 SVG / CSS 更小的语法，引擎用大约 200 个模板渲染成 SVG。支持流式渲染：语法还没写完也可以先画。另有一组 JSX 图元（`Rect`、`Text`、`Group`）和 `renderSVG`，可以直接得到 SVG 字符串。

用法：

```bash
npm install @antv/infographic
```

```ts
import { Infographic } from "@antv/infographic";

const infographic = new Infographic({
  container: "#container",
  width: 900,
  height: 540,
  padding: 24,
});

infographic.render(`
infographic list-row-simple-horizontal-arrow
data
  title 客户增长引擎
  items
    - label 线索获取
      value 18.6
      desc 渠道投放与内容获客
`);
```

`toDataURL({ type: "svg" | "png" })` 的类型声明写明：要在 `render()` 之后、并且在浏览器环境里调用。语法规则是两空格缩进，`infographic <模板名>` 开头，`data` / `theme` 为顶层块。

图标名（如 `mdi/rocket-launch`）不是字体。官方自定义加载器示例向 `https://api.iconify.design` 取 SVG。离线时用 `registerResourceLoader`，从本地文件返回 `loadSVGResource(svgString)`。

适合条目、时间线、层级、对比。不适合一张自由构图的品牌海报。

## 6. Lucide

- 仓库：https://github.com/lucide-icons/lucide
- 协议：ISC；从 Feather 派生的那一部分图标是 MIT
- 站点：https://lucide.dev
- 静态包：`lucide-static` 1.59.0

约 1600 个风格一致的 SVG 图标，商用和个人使用都可以。这不是生成器。云端模型只选名字，路径来自仓库。

用法：

```bash
npm install lucide-static
```

图标文件在 `lucide-static/icons/<name>.svg`，例如 `arrow-right.svg`。本地读取这个文件，再交给 Takumi 的 `<img>` 或 resvg。不要走 unpkg / jsDelivr 上的同一个文件。

## 7. Iconify 离线数据

- 框架仓库：https://github.com/iconify/iconify
- 协议：MIT（框架和 `@iconify/utils`）。图标本身各自有协议，要看具体图标集
- 规模：约 6367 star
- 工具包：`@iconify/utils` 3.1.7
- 数据包：`@iconify-json/<集合名>`，例如 `@iconify-json/lucide`

200 多套、约 20 万枚开源图标的统一 JSON。框架自带的 Web Component 会向 Iconify API 要数据，离线时不能用那条路。离线路径是：安装数据包，用 utils 在本地拼 SVG。

用法：

```bash
npm install @iconify/utils @iconify-json/lucide
```

```js
import { readFileSync } from "node:fs";
import { getIconData, iconToSVG, iconToHTML, replaceIDs } from "@iconify/utils";

const iconSet = JSON.parse(readFileSync(
  "node_modules/@iconify-json/lucide/icons.json",
  "utf8",
));
const data = getIconData(iconSet, "cpu");
const rendered = iconToSVG(data, { height: "24" });
const svg = iconToHTML(replaceIDs(rendered.body), rendered.attributes);
```

Lucide 不够用时再加集合。每个集合单独核对许可证。

## 8. poster-ai

- 仓库：https://github.com/Michaelliv/poster
- 包：`poster-ai`
- 协议：MIT
- 规模：约 73 star，创建于 2026-04-16

一个 `.tsx` 进，PNG / PDF / SVG / JPG / WebP 出。CLI 和库都可以。默认 PNG 引擎是 Takumi；`--engine chrome` 用本机 Chrome 或 Puppeteer 的 headless shell，换完整 CSS、PDF 和 SVG。

用法：

```bash
npm install poster-ai
poster export app.tsx -o out.png
poster export app.tsx -o out.pdf --engine chrome
```

```ts
import { Poster } from "poster-ai";

const poster = new Poster();
const png = await poster.render(
  { tsx: `export default () => <h1 className="text-5xl p-10">Hi</h1>` },
  { format: "png", width: 1600, height: 900 },
);
```

stdin 也可以：`poster export - -o out.png`。命令支持 `--json`。源码会留在 `.poster/`，下一轮改文件而不是重写整张图。

默认 Takumi 路径会把 Google Fonts 缓存到 `~/.cache/poster/fonts/`，裸 import 会通过 esm.sh 解析。渲染计算在本地，资源获取不在本地。要空气隔离就得改掉这两处，或直接用 Takumi 并传入本地字体。

## 9. pi-poster

- 仓库：https://github.com/Michaelliv/pi-poster
- 包：`pi-poster` 0.1.3（2026-04-23）
- 协议：MIT
- 规模：约 36 star

[pi](https://github.com/badlogic/pi-mono) 的扩展，不是通用渲染器。它注册一个 `poster_render` 工具和一份 `poster` skill：规定根节点用 `w-[Npx]` 声明画布、先写一份 TSX、再按 `tsxPath` 改已归档的文件。预检会拒绝 `min-h-screen`、根上的 `w-full`、百分比高度。`examples/` 里每张图有成对的提示词和 TSX。

用法绑在 pi 上：

```bash
pi install npm:pi-poster
```

```js
poster_render({
  tsx: `export default () => (
    <div className="w-[1200px] h-[630px] p-12 bg-black text-white">
      <h1 className="text-7xl font-black">Shipped.</h1>
    </div>
  )`,
  out: "./release.png",
});
```

它的 README 仍写 poster-ai 会带上 Chromium。poster 仓库后来把默认 PNG 改成了 Takumi，Chrome 变为可选。这里要学的是约束和迭代方式，不是把 pi 装进 SenseCraft。

## 综合对比

| 项目 | 角色 | 模型写出的东西 | 本机出图 | 精确文字 | 离线缺口 | 协议 |
|---|---|---|---|---|---|---|
| Takumi | 海报 / 卡片渲染器 | JSX、HTML 或 JSON 节点 | PNG、SVG，无浏览器 | 字体字节传入后由 Parley 整形 | 默认不读系统字体；`googleFonts()` 会联网 | MIT OR Apache-2.0 |
| Satori | 同上，上一代 | 无状态 JSX 或 `{type,props}` | 只出 SVG，PNG 另接光栅器 | 字形变成 path | 字体必须自带 | MPL-2.0 |
| resvg | SVG 光栅器 | 不写，只渲染已有 SVG | PNG | SVG `<text>` 加字体文件 | 无 | Apache-2.0 OR MIT；Node 绑定 MPL-2.0 |
| Typst | 长文排版编译器 | `.typ` | PDF、PNG、SVG | 编译器排版；PDF 保留真文字，SVG 转轮廓 | `--font-path` 即可离线 | Apache-2.0 |
| AntV Infographic | 条目型信息图 | 模板语法 | 浏览器里出 SVG / PNG | 模板内的 label / desc | 图标示例走 Iconify API；`toDataURL` 要浏览器 | MIT |
| Lucide | 图标零件 | 图标名 | 读本地 SVG | 不负责正文 | 用 `lucide-static` 的文件 | ISC（部分 MIT） |
| Iconify JSON | 更大的图标零件库 | `集合/名字` | utils 在本地拼 SVG | 不负责正文 | 不能用会请求 API 的组件 | 工具 MIT；图标集各自不同 |
| poster-ai | agent 循环参考 | 单个 TSX | 默认 Takumi，可选本机 Chrome | 同 Takumi | 默认拉 Google Fonts 和 esm.sh | MIT |
| pi-poster | 提示词和工具协议参考 | 同上，且必须改已存文件 | 调用 poster-ai | 同 poster-ai | 绑死 pi | MIT |

能力上可以叠成一条流水线，而不是九选一：

```text
云端模型
  ├─ 图标名          → Lucide / Iconify JSON → 本地 SVG
  ├─ 海报 JSX        → Takumi → PNG / SVG
  ├─ 长文 .typ       → Typst  → PDF / PNG
  └─ 信息图语法      → AntV   → SVG（图标加载器指向本地）
```

Satori 和 Takumi 做同一件事。Takumi 多了直接出 PNG、Grid 之外的排版特性、JSON 节点，以及更宽松的协议。poster-ai 的默认引擎就是 Takumi，额外价值是「源码归档后再改」；pi-poster 把这件事写成了给模型的合同。

## 结论

海报和精确标题用 Takumi。云端模型写固定画布的 JSX，或者写它的 JSON 节点树；本机 `takumi-js` 渲染。思源黑体或 Noto Sans CJK 的字体文件放在仓库里，作为 `fonts` 传入。不调用 `googleFonts()`。

图标用 Lucide 的本地 SVG。模型只输出名字。需要 Lucide 没有的符号时，再加指定的 `@iconify-json/*`，用 `@iconify/utils` 在本地拼 SVG。

长正文、多栏、需要可选中文字的 PDF，用 Typst，并带上 `--font-path`。这是第二条渲染器，不替代海报。

条目型信息图用 AntV 的语法。图标通过 `registerResourceLoader` 读本地 SVG。`Infographic.toDataURL` 依赖浏览器，无浏览器出图时用它的 `renderSVG`，或把 SVG 交给 resvg。

resvg 只处理已经是 SVG 的输入。海报路径上 Takumi 已经包含光栅化，不必再套 `@resvg/resvg-js`。若单独依赖，用 linebender 的库或 CLI（Apache-2.0 OR MIT），而不是 MPL 的 Node 绑定。

Satori 留作对照和示例库，不作为实现。poster-ai 和 pi-poster 留作模型合同：一次一个组件、画布写在根节点上、下一轮改已保存的源码。不把它们的运行时和 pi 装进来，避免 Google Fonts、esm.sh 和另一套 agent。

## 本机试跑（2026-10-18）

机器是 macOS 12.0.1（Intel x86_64），Node.js 24.16.0。Homebrew 的 `rustc` 1.93.0 因 LLVM 符号不匹配无法启动，所以没有从源码编译。`pi` 未安装。

Demo 在 `gen-vision/demo`，图片在 `gen-vision/demo/out`。重跑：`node gen-vision/demo/run.mjs`。中文用的是系统里的 `Arial Unicode.ttf`，字体名在调用时指定为 `PosterCJK`。

| 项目 | 这台机器 | 看到的效果 |
|---|---|---|
| Takumi 2.16.3 | 能跑，约 130ms | `out/takumi.png`：中文标题和 12.5°C 都正确 |
| Satori + `@resvg/resvg-js` | 能跑，Satori 约 8s（首次解析 22MB 字体） | `out/satori.png`：中文正确 |
| resvg CLI | 未安装；Node 绑定能跑 | `out/resvg.png`：SVG `<text>` 中文正确 |
| Typst 0.15.1 | 官方 x86_64 二进制可执行 | `out/typst.png`、`out/typst.pdf`：中文正确。二进制在 `demo/bin/`，已被 gitignore |
| AntV SSR | `renderToString` 能跑 | SVG 里的中文写在 `foreignObject` 中，并引用蚂蚁字体 CDN。resvg 只画出箭头和 01/02/03（`out/antv.png`）。本机 Chrome 无头截图能画出中文（`out/antv-chrome.png`） |
| Lucide | 能跑 | `out/lucide-cpu.png` |
| Iconify JSON | 能跑，无 API | `out/iconify-thermometer.png` |
| poster-ai 0.5.0 | Takumi 引擎能跑，约 600ms | `out/poster-ai.png`：英文在，中文是方框。默认字体不含汉字，调用参数里也不能传入本地字体文件 |
| pi-poster | 不能跑 | 没有 `pi` 命令。它底下的 poster-ai 已经单独跑过 |
