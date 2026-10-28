# sensecraft-recamera — CLI commands

Target binary: `sensecraft-cli` (this repo). Auth: active **SenseCAP Access Key** (`auth init`) → `{ai-host}/openapi`.

> Status: **shipped (P1 list + P2 check/open)**. Flash stays in Console. There is no `recamera deploy`.

## Auth prerequisite

```bash
sensecraft-cli auth info
sensecraft-cli auth init   # user only, interactive
```

## List (existing commands)

Always pass `--uniform-type 40`. Empty list → stop.

```bash
sensecraft-cli app list --public --uniform-type 40
sensecraft-cli app list --mine --uniform-type 40
sensecraft-cli model list --public --uniform-type 40
sensecraft-cli model list --mine --uniform-type 40
```

## `recamera check`

**Status: implemented.**

Verify the id is classic reCamera (`40`). Does not open a browser.

| Flag         | Required | Notes                                      |
| ------------ | -------- | ------------------------------------------ |
| `--model-id` | one of   | Model id                                   |
| `--app-id`   | one of   | App id                                     |
| `--sandbox`  | no       | Sample JSON, no network                    |

Exactly one of `--model-id` / `--app-id`.

Stdout JSON includes `ok`, `device` (40), `flash` (`local`), `console_url`, `candidates`, `ask_pick`.  
`ok=false` → non-zero exit. Do not loosen the filter.

```bash
sensecraft-cli recamera check --model-id <model_id>
sensecraft-cli recamera check --app-id <app_id>
```

If `ask_pick` is true, ask the user which candidate to flash.

## `recamera open`

**Status: implemented.**

Runs the same check, then opens `console_url` (or prints it). Sandbox never opens a browser.

| Flag          | Required | Notes                                         |
| ------------- | -------- | --------------------------------------------- |
| `--model-id`  | one of   | Model detail = AI deploy page (click 部署)    |
| `--app-id`    | one of   | App detail; plaza omits `from=user-center`    |
| `--print-only`| no       | Print URL only                                |
| `--sandbox`   | no       | Sample JSON; never open a browser             |

```bash
sensecraft-cli recamera open --model-id <model_id>
sensecraft-cli recamera open --app-id <app_id> --print-only
```

`console_url` is `/ai/model/detail?id=…` for a model, or `https://codecraft.seeed.cc/hub/view-application/<id>-<slug>` for an app.  
**Not** `/ai/device/local/40` (that family is 32/36/37/41/52). **Not** `/ai/recamera` (301 → Grove 36).

## Out of scope

| Action | What to do instead |
|--------|--------------------|
| `cloud deploy` | Jetson / cloud kit only |
| Public model download / clone | Refuse; deploy-only |
| `recamera deploy` / device HTTP | Do not invent; user flashes in Console |
| `flash-firmware` block | Later |
| reCamera Pro `53` | `sensecraft-models` + `model recamera-pro resolve` |
| CodeCraft App Square | Unrelated |
