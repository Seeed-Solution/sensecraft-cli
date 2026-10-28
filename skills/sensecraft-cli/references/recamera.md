# SenseCraft AI · classic reCamera (uniform_type 40)

Orchestration skill: **list → pick → check → open Console → user clicks Deploy**. The China site supports SenseCraft Data only; use Global for SenseCraft AI.

Parent: [`../SKILL.md`](../SKILL.md).  
List/view: [`apps.md`](apps.md), [`models.md`](models.md).  
Commands: [recamera-commands.md](recamera-commands.md).

**This skill does not flash.** Flash is L3 in the browser (IP or USB). There is no `recamera deploy` and no device HTTP in this CLI.

## Agent rules

1. Auth first:
   ```bash
   sensecraft-cli auth info
   ```
   Need **SenseCAP Access Key** (`auth init`) for `app` / `model` / `recamera`. If missing, the agent runs `auth init` in a PTY/TTY session and handles prompts per the [parent auth workflow](../SKILL.md#auth); the user completes browser authorization. Recheck auth, then continue without asking the user to run CLI commands.
2. **Classic reCamera only (`40`)**. Do **not** treat Pro `53`, XIAO `32`, or Grove `36` as the same device. `/ai/recamera` redirects to Grove `36` — do not send users there.
3. Filter **only** with API `--uniform-type 40`. Do not guess from local materials, SKU strings, or adapteds-only. Public **and** mine:
   ```bash
   sensecraft-cli app list --public --uniform-type 40
   sensecraft-cli app list --mine --uniform-type 40
   sensecraft-cli model list --public --uniform-type 40
   sensecraft-cli model list --mine --uniform-type 40
   ```
4. **Empty list: say so and stop.** Do not drop `--uniform-type` or widen to 53/32.
5. After the user picks an app or model, `recamera check`. If `ok` is false, say it is not classic reCamera and stop.
6. **Multiple firmwares / models** (`ask_pick: true` or several candidates): **ask again** which one to flash. Do not pick for them.
7. Both allowed:
   - Flash the model already on the app `deploy-ai-model` block
   - Flash a **separately picked** model from the 40 list
8. **Public models are deploy-only.** Never `model download`, never print `file_url`, never copy a public model into My Models. Private My Models may be used as-is. Flashing does not require copying the app.
9. **Do not `cloud deploy`.** That path is Jetson / cloud kit (`26`), not reCamera.
10. **`flash-firmware` blocks are out of scope** this pass. Do not invent a firmware-flash CLI.
11. After a successful check, if the user wants to flash, run `recamera open` (or `--print-only` if they will paste the URL). Success = they say the Console page worked and they clicked Deploy. Opening a page is **not** an L2 `--yes` write.
12. If there is **no device / no IP**: still open the Console deploy page and stop. Do not invent a device protocol. Do not require an IP to confirm.
13. Retry list/check/open at most **5 times per run**. Then stop and report the last error.
14. CodeCraft App Square is unrelated — do not route plaza-upload work here.

## Typical workflow

```bash
sensecraft-cli auth info
sensecraft-cli model list --public --uniform-type 40
sensecraft-cli app list --public --uniform-type 40
# optional: mine
sensecraft-cli model list --mine --uniform-type 40
sensecraft-cli app list --mine --uniform-type 40

sensecraft-cli recamera check --model-id <id>
# or
sensecraft-cli recamera check --app-id <id>

# After the user chose the item (and, if ask_pick, which firmware):
sensecraft-cli recamera open --model-id <id>
# or --print-only if a browser cannot be opened
```

Then tell the user: on the opened **model/app detail** page, click **Deploy**, connect classic reCamera over **IP or USB**, and finish in Console. Do not download the public weights.

## CLI status

| Area | Status |
|------|--------|
| `app/model list --uniform-type 40` | **Implemented** |
| `recamera check` / `recamera open` | **Implemented** (P2) |
| Device flash / `recamera deploy` | **Not implemented** (L3 Console) |
| `model recamera-pro resolve` | Pro `53` only — not this skill |
