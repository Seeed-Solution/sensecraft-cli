import { readFile, writeFile, mkdir } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = dirname(fileURLToPath(import.meta.url));
const outDir = join(root, "out");
const fontPath = "/System/Library/Fonts/Supplemental/Arial Unicode.ttf";
const fontFamily = "PosterCJK";

await mkdir(outDir, { recursive: true });
const font = await readFile(fontPath);
const results = [];

async function step(name, fn) {
  const started = Date.now();
  try {
    const detail = await fn();
    const row = { name, ok: true, ms: Date.now() - started, detail };
    results.push(row);
    console.log(`OK  ${name}  ${row.ms}ms  ${detail}`);
  } catch (error) {
    const message = error instanceof Error ? error.stack || error.message : String(error);
    results.push({ name, ok: false, ms: Date.now() - started, detail: message });
    console.error(`FAIL ${name}\n${message}\n`);
  }
}

await step("takumi", async () => {
  const { render, renderSvg } = await import("takumi-js");
  const html = `
    <div style="width:1200px;height:630px;display:flex;background:#0f172a;color:#f8fafc;font-family:${fontFamily};padding:72px;flex-direction:column;justify-content:center;">
      <div style="font-size:22px;letter-spacing:6px;color:#38bdf8;">SENSECRAFT · LOCAL RENDER</div>
      <div style="font-size:68px;margin-top:18px;">本地渲染海报</div>
      <div style="font-size:28px;margin-top:16px;color:#cbd5e1;">温度 12.5°C · 湿度 48% · EUI 2CF7F1200000001</div>
    </div>`;
  const options = {
    width: 1200,
    height: 630,
    fonts: [{ name: fontFamily, data: font, weight: 400, style: "normal" }],
  };
  const png = await render(html, options);
  const svg = await renderSvg(html, options);
  await writeFile(join(outDir, "takumi.png"), png);
  await writeFile(join(outDir, "takumi.svg"), svg);
  return `png ${png.length} bytes, svg ${svg.length} bytes`;
});

await step("satori+resvg", async () => {
  const satori = (await import("satori")).default;
  const { Resvg } = await import("@resvg/resvg-js");
  const svg = await satori(
    {
      type: "div",
      props: {
        style: {
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          background: "#14532d",
          color: "white",
          padding: "64px",
          fontFamily,
        },
        children: [
          {
            type: "div",
            props: { style: { fontSize: 22, color: "#bbf7d0" }, children: "SATORI + RESVG" },
          },
          {
            type: "div",
            props: { style: { fontSize: 64, marginTop: 12 }, children: "精确文字 12.5°C" },
          },
        ],
      },
    },
    {
      width: 1200,
      height: 630,
      fonts: [{ name: fontFamily, data: font, weight: 400, style: "normal" }],
    },
  );
  await writeFile(join(outDir, "satori.svg"), svg);
  const png = new Resvg(svg, { fitTo: { mode: "width", value: 1200 } }).render().asPng();
  await writeFile(join(outDir, "satori.png"), png);
  return `svg ${svg.length} bytes, png ${png.length} bytes`;
});

await step("resvg-text", async () => {
  const { Resvg } = await import("@resvg/resvg-js");
  const svg = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="420">
  <rect width="1200" height="420" fill="#1e293b"/>
  <text x="64" y="180" fill="#f8fafc" font-family="${fontFamily}" font-size="64">resvg 直接绘制文字</text>
  <text x="64" y="270" fill="#94a3b8" font-family="${fontFamily}" font-size="32">湿度 48% · 不是云端采样</text>
</svg>`;
  await writeFile(join(outDir, "resvg.svg"), svg);
  const png = new Resvg(svg, {
    font: {
      fontFiles: [fontPath],
      loadSystemFonts: false,
      defaultFontFamily: fontFamily,
    },
  }).render().asPng();
  await writeFile(join(outDir, "resvg.png"), png);
  return `png ${png.length} bytes`;
});

await step("lucide", async () => {
  const { readFile } = await import("node:fs/promises");
  const { Resvg } = await import("@resvg/resvg-js");
  const raw = await readFile(join(root, "node_modules/lucide-static/icons/cpu.svg"), "utf8");
  const svg = raw
    .replace(/\swidth="[^"]*"/, ' width="256"')
    .replace(/\sheight="[^"]*"/, ' height="256"')
    .replaceAll("currentColor", "#0f766e");
  await writeFile(join(outDir, "lucide-cpu.svg"), svg);
  const png = new Resvg(svg, {
    fitTo: { mode: "width", value: 256 },
    background: "#ecfdf5",
  }).render().asPng();
  await writeFile(join(outDir, "lucide-cpu.png"), png);
  return `png ${png.length} bytes`;
});

await step("iconify", async () => {
  const { readFile, writeFile } = await import("node:fs/promises");
  const { getIconData, iconToSVG, iconToHTML, replaceIDs } = await import("@iconify/utils");
  const { Resvg } = await import("@resvg/resvg-js");
  const iconSet = JSON.parse(
    await readFile(join(root, "node_modules/@iconify-json/lucide/icons.json"), "utf8"),
  );
  const data = getIconData(iconSet, "thermometer");
  if (!data) throw new Error("thermometer icon missing");
  const rendered = iconToSVG(data, { height: "256", color: "#b45309" });
  const svg = iconToHTML(replaceIDs(rendered.body), rendered.attributes);
  await writeFile(join(outDir, "iconify-thermometer.svg"), svg);
  const png = new Resvg(svg, {
    fitTo: { mode: "width", value: 256 },
    background: "#fff7ed",
  }).render().asPng();
  await writeFile(join(outDir, "iconify-thermometer.png"), png);
  return `png ${png.length} bytes`;
});

await step("antv-infographic", async () => {
  const { renderToString } = await import("@antv/infographic/ssr");
  const { Resvg } = await import("@resvg/resvg-js");
  const svg = await renderToString(`
infographic list-row-simple-horizontal-arrow
data
  title SenseCraft 信息图
  desc 本机 SSR，无浏览器
  items
    - label 温度
      value 12.5
      desc 本地渲染
    - label 湿度
      value 48
      desc 精确文字
    - label 电量
      value 86
      desc 无云端生图
`);
  await writeFile(join(outDir, "antv.svg"), svg);
  const png = new Resvg(svg, {
    fitTo: { mode: "width", value: 1200 },
    font: {
      fontFiles: [fontPath],
      loadSystemFonts: false,
      defaultFontFamily: fontFamily,
    },
    background: "white",
  }).render().asPng();
  await writeFile(join(outDir, "antv.png"), png);
  return `svg ${svg.length} bytes, png ${png.length} bytes`;
});

await step("poster-ai", async () => {
  const { Poster } = await import("poster-ai");
  const poster = new Poster({ engine: "takumi" });
  const png = await poster.render(
    {
      tsx: `export default function PosterCard() {
        return (
          <div className="w-[1200px] h-[630px] bg-zinc-950 text-white flex flex-col justify-center px-20">
            <div className="text-sm tracking-[0.3em] text-amber-300">POSTER-AI</div>
            <h1 className="text-7xl font-black mt-4">一个 TSX 文件</h1>
          </div>
        );
      }`,
    },
    { format: "png", width: 1200, height: 630 },
  );
  await writeFile(join(outDir, "poster-ai.png"), png);
  return `png ${png.length} bytes`;
});

await writeFile(join(outDir, "results.json"), JSON.stringify(results, null, 2));
const failed = results.filter((row) => !row.ok).length;
console.log(`\n${results.length - failed}/${results.length} demos wrote files in ${outDir}`);
process.exitCode = failed ? 1 : 0;
