# SenseCraft AI · cloud devices

Auth: **Access Key** (`auth init`). The China site supports SenseCraft Data only; use Global for SenseCraft AI. Parent: [`../SKILL.md`](../SKILL.md).

## Agent rules

1. `sensecraft-cli auth info` first — need SenseCAP Access Key for `{ai-host}/openapi`.
2. **Never** use SenseCAP `device` commands here (`device list` = SenseCAP OpenAPI).
3. **L2 confirm is a separate chat turn** ([confirmation.md](confirmation.md)) before `cloud deploy --yes` (and device output / order writes). 「部署到这台设备」is **not** confirmation — ask (部署 + 环境 + EUI + model_id), wait for `确认`, then `--yes`. Never pass `--yes` on the same turn as the first request.
4. Local USB / Web Serial = **L3** → guide Console `/ai/device/local/...`.
5. **reCamera Pro** (`uniform_type=53`) local IP/USB RC2/ALPKG deploy is **not** `cloud deploy` / `update_stream` — guide users to Console device workspace (L3).

## CLI status

| Command | Status |
|---------|--------|
| `cloud device list` / `view` | **Implemented** |
| `cloud deploy` / `cloud order` | **Implemented** (L2 deploy) |

## Workflows

```bash
sensecraft-cli cloud device list --uniform-type 26 --page 1 --length 20
sensecraft-cli cloud device list --uniform-type 26 --adapteds <adapted_id> --name "Jetson"
sensecraft-cli cloud device view --eui <EUI>
# confirm with user:
sensecraft-cli cloud deploy --eui <EUI> --model-id <id> --from-json ./stream.json --yes
sensecraft-cli cloud order --request-id <request_id> --device-uuid <uuid>
```

`uniform_type` 26 is common for the Jetson cloud device UI; model deploy pages may pass other types / `adapteds`.

Full flag reference and copyable examples: [cloud-commands.md](cloud-commands.md).

## Disambiguation

| Want | Use |
|------|-----|
| SenseCAP node/gateway telemetry | `sensecraft-cli device …` (Access Key) |
| Jetson / AI cloud kit | `cloud device …` (this skill) |
| USB flash | Console local device (L3) |
