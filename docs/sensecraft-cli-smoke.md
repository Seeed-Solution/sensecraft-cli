# SenseCraft CLI 发版冒烟清单

> 每次发版 / 合并 `main` 前必跑。完整功能用例见 [`sensecraft-cli-test-cases.md`](sensecraft-cli-test-cases.md)。

---

## 0. 自动化门禁（无真实账号）

```bash
go test ./...
./script/smoke-sensecap.sh
./script/smoke-ai.sh
# 可选：先 ./script/build.sh
cd packages/cli && npm test && npm run test:install
```

| 序号 | 场景 | 通过标准 |
|------|------|----------|
| S1 | `go test ./...` | 全绿 |
| S2 | `script/smoke-sensecap.sh` | 退出 0 |
| S3 | `script/smoke-ai.sh` | 退出 0（含 L2 缺 `--yes` 断言） |
| S3b | `packages/cli` npm test / test:install | 退出 0 |

---

## 1. 鉴权与沙箱抽查

建议使用隔离配置：

```bash
export SENSECRAFT_CONFIG_FILE=/tmp/sensecraft-smoke.json
# 或复用已有双 profile 的本机配置（勿把密钥写进日志）
```

| 序号 | 场景 | 命令 / 步骤 | 通过标准 |
|------|------|-------------|----------|
| S4 | `auth info` | `sensecraft-cli auth info` | 展示 Site / Org / Key ID，不打印密钥 |
| S5 | sandbox 只读 | `device list --sandbox`<br>`app list --sandbox`<br>`model list --sandbox` | 均为合法缩进 JSON，退出 0 |
| S6 | L2 门禁 | 下列命令**不传** `--yes`：<br>`app delete --id 1`<br>`model publish --id 1`<br>`cloud deploy --eui x --model-id 1` | 均非 0，提示需确认（sandbox 配置亦可） |
| S9 | 未登录 | 空配置时 `app list` | 非 0，提示 `auth init` |

---

## 2. 真实联调抽查（已有 Access Key 时）

在可丢弃组织执行；跑完清理 QA 数据。

| 序号 | 场景 | 步骤 | 通过标准 |
|------|------|------|----------|
| S7 | `app create` 保 id | `app create --name "smoke-$(date +%s)"` | stdout 含可用 `app_id`（即使后端 update 为 `{}`） |
| S8 | 稀疏改名 | `echo '{"app_name":"smoke-renamed"}' \| app update --id <id> --from-json - --yes`<br>再 `app view --id <id> --sketch` | 新名称生效；其它 info 未丢 |
| 清理 | 删测试 app | `app delete --id <id> --yes` | 删除成功 |

可选加跑（有对应资源时）：

```bash
sensecraft-cli model constants
sensecraft-cli cloud device list --uniform-type 26
```

---

## 3. 发版勾选

- [ ] S1–S3（+ S3b）通过  
- [ ] S4–S6、S9 抽查通过  
- [ ] （若发 AI 相关版本）S7–S8、S10 + 清理  
- [ ] `packages/cli/package.json` version 与即将打的 tag 一致
- [ ] `sensecraft-cli update --check` 能查到 npm 上的版本（不要求当场升级）  
- [ ] Skills 已 `script/sync-skills-to-npm.sh`（若改了 skills）  
- [ ] CHANGELOG / PR 描述已写清行为变更  

---

## 相关文档

| 文档 | 用途 |
|------|------|
| [`sensecraft-cli-test-cases.md`](sensecraft-cli-test-cases.md) | 全量功能测试用例 |
| [`../script/smoke-ai.sh`](../script/smoke-ai.sh) | AI sandbox 冒烟脚本 |
| [`../script/smoke-sensecap.sh`](../script/smoke-sensecap.sh) | SenseCAP sandbox 冒烟脚本 |
| [`../ARCHITECTURE.md`](../ARCHITECTURE.md) §10 | CI 与冒烟说明 |
