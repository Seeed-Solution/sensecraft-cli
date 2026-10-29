# sensecraft-devices-cloud — CLI commands

Binary: `sensecraft-cli`. Auth: Access Key (`auth init`).

## `cloud device list`

| Flag | Default | Notes |
|------|---------|-------|
| `--uniform-type` | `26` | Device type; Jetson cloud UI commonly uses `26` |
| `--adapteds` | — | Optional adapted-device/model filter |
| `--name` | — | Filter by `device_name` |
| `--page` / `--length` | `1` / `20` | Pagination |
| `--sandbox` | false | No network; sample JSON |

```bash
sensecraft-cli cloud device list --uniform-type 26 --page 1 --length 20
sensecraft-cli cloud device list --uniform-type 26 --adapteds <adapted_id> --name "Jetson"
```

## `cloud device view`

```bash
sensecraft-cli cloud device view --eui <device_eui>
```

`--eui` is required. Add `--sandbox` for an offline sample.

## `cloud deploy` — L2

Requires explicit user confirmation and `--yes`.

```bash
# Minimal model selection
sensecraft-cli cloud deploy \
  --eui <device_eui> --model-id <model_id> --yes

# Stream settings supplied as a JSON object
sensecraft-cli cloud deploy \
  --eui <device_eui> --model-id <model_id> \
  --from-json ./stream.json --yes
```

The response contains a `request_id`; poll it with `cloud order`.

## `cloud order`

```bash
sensecraft-cli cloud order --request-id <request_id>
```

This command only polls deployment status and is not L2.

## Disambiguation

- `cloud device …`: SenseCraft AI Jetson / cloud kit.
- `device …`: SenseCAP node/gateway OpenAPI.
- Local USB / Web Serial flashing: Console L3, not this CLI.
