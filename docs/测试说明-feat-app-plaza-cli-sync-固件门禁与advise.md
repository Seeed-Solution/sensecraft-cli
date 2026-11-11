# CLI 固件上传引导 / 门禁【feat/app-plaza-cli-sync】

**仓库 / 分支**：`sensecraft-cli` · `feat/app-plaza-cli-sync`  
**一句话**：告诉 AI「这块板默认烧录地址是什么、该上传哪种固件」，并在传错时直接拦住；**不负责帮你编译固件**。

---

## 本轮改了什么（通俗版）

| 改动 | 人话 |
|------|------|
| 新增 `firmware advise` | 问 CLI「Sticky/E1001/… 该传什么包？」会给出默认地址、该传合并包还是应用包、Muse 时怎么传 |
| `advise --list` | 一张表列出所有支持的板：默认地址 → 该上传什么 |
| `upload_guide` | 明确写「要传什么 / 别传什么 / 推荐文件名 / Muse·全量怎么填」 |
| 上传 / 创建 门禁 | 地址不对、体积离谱、Muse 开关和文件名对不上、缺 SKU 或 size → **报错并点出字段**，避免静默传错 |
| Muse 目录上传 | 只上传 `muse-gadget.bin`，旁边的 bootloader 等不传 OSS；配置里地址仍用板默认，**不要写 `0x20000`** |
| `app blocks` 免登录 | 没 Access Key 也能看设备默认表（和 help 一致） |
| Grove 芯片标注 | 改为 ATmega328P（以前标错） |
| skills 文案 | 统一成：先 advise，再按地址选对文件上传；CLI 不生成固件 |

**不改 / 不做**：真机串口烧录；替 AI 编译出 bin；自动 merge-bin。

---

## 怎么理解「该传什么」

板子在 Hub 里有一个**默认烧录地址**。地址决定文件形状：

| 默认地址 | 该上传 | 常见板 |
|----------|--------|--------|
| `0x0` | **合并固件**（引导 + 分区表 + 应用打成一个 bin） | Sticky、Watcher、Indicator、1.47"、IPS |
| `0x90000` | **应用区固件**（只有应用，不要整片合并包） | E1001、E1002、电子纸 |
| `0x4000` | **应用区固件** | Wio Terminal |
| `0x10000` | **application 应用包**（文件名里常有 application） | 普通 XIAO（C3/C6/S3…） |
| 开了 Muse | 只传 **`muse-gadget.bin`** | 上面 Muse 白名单板；配置里地址仍写板默认，站点自己写到 `0x20000` |

口诀：**地址对形状，形状对文件；Muse 另走一条，配置地址别改成 `0x20000`。**

### Sticky 举例

| 场景 | Muse 开关 | 全量烧录 | 配置地址 | 上传文件 |
|------|-----------|----------|----------|----------|
| 普通固件 | 关 | 默认开 | `0x0` | 合并 bin |
| Muse 固件 | 开 | 必须关 | 仍是 `0x0` | 只有 `muse-gadget.bin` |

---

## 准备

```bash
cd sensecraft-cli && go build -o sensecraft-cli .
./sensecraft-cli firmware advise --list                 # 看全部板，免登录
./sensecraft-cli firmware advise --device-id 100056818  # Sticky
./sensecraft-cli app blocks                             # 设备默认表，免登录
```

云端上传 / 创建 应用需要先登录：`sensecraft-cli auth info`（Global）。

---

## 自动化（发测前）

```bash
go test ./internal/service/ ./cmd/... -count=1
python3 test/test_create_app_manifest.py
```

应全部通过。

---

## 手工抽查清单

- [ ] `--list` 能列出各板默认地址和「该上传什么」
- [ ] Sticky / Watcher → `0x0` 合并；E100x → `0x90000` 应用区；Wio → `0x4000`
- [ ] Muse：只提 `muse-gadget.bin`，并提醒配置地址不要写 `0x20000`
- [ ] 传错类型时（太小的「合并包」、Muse 名对不上、缺 device_id 等）报错能看懂、点出字段
- [ ] 未登录也能跑 `advise` / `app blocks`

---

## 给测试同学

- 本轮验收重点是 **引导对不对、拦错对不对**，不是烧录能不能亮屏。
- 真机烧录仍以 Hub 编辑器 / `sensecraft-app` 测试说明为准。
