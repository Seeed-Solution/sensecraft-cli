# L2 confirmation protocol (mandatory)

`--yes` / `--agree-declaration` mean **the user already answered a confirmation question in this conversation**. They are not shortcuts.

## First user request is INTENT, not confirmation

These are **not** confirmation. Do **not** pass `--yes` on the same turn:

- 「删这个63760」「删除应用 demo」「把模型下架」「发布这个模型」
- 「delete this」「unpublish」「deploy to the device」
- Any first mention of create / modify / publish / unpublish / delete / deploy / stop / bind / unbind

## Required chat question (before any L2 CLI)

Ask in chat. **Do not run** the write command yet. The question MUST include at least:

1. **操作类型** — 删除 / 发布 / 私有化 / 部署 / 停止 / 修改 / 绑定 / …
2. **环境** — `auth info` 的 Global / China（生产环境须点明）
3. **对象** — 名称 + ID（没有名称就先 `view` / `list` 再问）

Template:

```
将从{环境}环境{操作}{对象}（ID: {id}）。{后果}。是否确认执行？
```

Example (user said 「删除应用 demo」):

```
将从 Global 环境删除应用 demo（ID: 123）。删除后可能无法恢复。是否确认执行？
```

Example (user said 「删这个63760」):

```
将从 China 环境删除模型 63760。删除后可能无法恢复。是否确认执行？
```

Classic reCamera flash is **not** an L2 `--yes` write. After the user picked the app/model (and firmware if several), `recamera open` may run. The user then clicks Deploy in Console (L3). Do not invent `recamera deploy`.

Wait for a **new** user reply: `确认` / `confirm` / `yes`. Only then:

```bash
sensecraft-cli model delete --id 63760 --yes
```

## Never

- Never add `--yes` on the same turn as the first request
- Never treat `--yes` as “the user already said delete”
- Never skip the question because the user sounded sure
- Production / Global / China: say the environment explicitly (special confirm)

## After the user confirms

Then pass `--yes` (and `--agree-declaration` only if they also agreed to that legal text).
