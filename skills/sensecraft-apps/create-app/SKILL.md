---
name: create-app
description: Create a SenseCraft Application Plaza app from a finished local project using a versioned local manifest (sensecraft-app.json) as the single source of truth — mine the project for app info, create the app in one go, then keep later edits local-first and push updates safely with drift detection. Use whenever the user wants to turn a finished or current project into a SenseCraft app, "publish my project" to SenseCraft, make an app from this repo/codebase, or update an app that has a sensecraft-app.json manifest — even if they only say "make this project an app" or "push my app changes". Publishing, unpublishing, deleting, and cloning stay with the parent sensecraft-apps skill.
---

# create-app

Turn a finished local project into a SenseCraft Application Plaza app through a local manifest: gather → write manifest → one-go create → local-first updates.

Parent skill: [`../SKILL.md`](../SKILL.md). Routing: [`../../sensecraft-ai/SKILL.md`](../../sensecraft-ai/SKILL.md).

> Everything declared upstream applies in full here — lifecycle ops (publish / unpublish / delete / clone) belong to the parent skill, L2 writes follow [`../../confirmation.md`](../../confirmation.md), and the parent's agent rules (auth, visibility values, output contract, re-read after update) hold. Below is only what create-app adds on top.

## Why the manifest exists

The manifest makes the workflow resumable and safe. It remembers the `app_id` so a retry never spawns a duplicate (the server allows duplicate names). It records what was last pushed so a later Console edit is detected as drift instead of silently overwritten. And in a project without git, its timestamped archive snapshots are the only version history — treat them as precious.

Manifest schema, field-by-field: [reference-manifest.md](reference-manifest.md). Starting template: [assets/manifest-template.json](assets/manifest-template.json).

## Manifest files on disk

- **Canonical**: `sensecraft-app.json` in the project root. Several apps in one repo: `sensecraft-app.<slug>.json` per app (`<slug>` = lowercase-hyphen app short name).
- **Versioning**: run `git rev-parse --is-inside-work-tree` **once at gather time** and reuse the decision for the whole session.
  - Inside git → canonical file only; suggest the user `git add` it after every successful push.
  - No git → after every successful create/push and at the end of gather, also write `.sensecraft/archive/sensecraft-app[.<slug>].YYYYMMDD-HHmmss.json`. The canonical file is always the latest; archives are immutable history.
- **Discovery** (update flow): prefer `sensecraft-app.json`; else a single `sensecraft-app.*.json` — excluding the reserved transient names `*.payload.json` / `*.patch.json` / `*.materials.json`; else list candidates and ask which app. A candidate must have manifest shape (`manifest_version` + `identity` + `sync`) — reject lookalikes. No manifest → this is a create flow. A discovered manifest whose `identity.app_id` is null and which fails validation or cannot be trusted: tell the user what was wrong, archive it under `.sensecraft/archive/`, rebuild from the template, and run `app list --mine --name "<app_name>"` as a duplicate check before creating.
- **Transient payload/patch files** (the projected `--from-json` bodies) live under `.sensecraft/` — create the dir if needed — and are deleted once the push completes. Never leave them as untracked clutter in the project root. Only delete transients this session created — pre-existing leftovers from an earlier session are reported to the user and removed only with their agreement.
- Never store tokens, access keys, or device keys in the manifest. If a mined value looks secret-like, warn and keep it out.

## Workflow A — gather

Mine first, ask approval once at the end.

1. **Mine this conversation, then the project directory** (read-only). You are usually inside the development session — what was built, configured, and decided here is the primary source; project files are the cross-check. Organize what is already known directly.
   
   - `README*` → `app_name` / `excerpt` candidates (excerpt ≤ 300 UTF-16 code units).
   - `package.json` / firmware README → name, `license`, `author`.
   - `docs/`, `images/`, `assets/`, `screenshots/` → image candidates: JPG/JPEG/PNG/GIF/WebP, ≤ 5 MB each, up to 16, first = cover, prefer 4:3. No usable cover yet is fine — a private draft doesn't need one; publishing does. The user decides when to choose, upload, or generate one.
   - **Deployment plan**: the steps serve non-developers who can only use the tool blocks provided — plan what they do on the app page (connect, run a block, configure, use), never how the project is built. Read [reference-tool-blocks.md](../reference-tool-blocks.md) before drafting to know what's available. An artifact you can't upload yet never removes a block the flow needs; resolve the config per the reference and mark what's pending. Note candidate artifacts (firmware bins, model files, flow JSON) without deciding yet.
   - Hardware mentions (XIAO, Grove, reCamera…) → BOM candidates; resolve SKUs with `sensecraft-cli app bom search --name "<kw>"`.
2. **Fill gaps in ONE consolidated question round**: present a brief overview of the app fields and the deployment steps, then ask for what is missing — `scene` pick (list ids from `sensecraft-cli model constants` → `app_scenarios_array`; propose the best match), `allow_clone`, `allow_commerce`, and anything you flagged for the user to decide. If a step can't be completed by that audience, surface it here with options instead of writing a flow that contradicts itself. `app_name` defaults to the project's own name — propose any embellishment in the round rather than renaming silently. BOM default scope: **the device plus app-specific accessories named in the project docs** (ordinary cables, computers, and generic tools are excluded). When no Seeed SKU resolves via `app bom search`, custom BOM rows (name/quantity/note/buy_url, no `id`) are the correct fallback.
3. **Write the manifest** from the template with `identity.app_id: null`, `sync.local_revision: 1`.
4. **Validate locally before any network write** — fail fast here, not after create. Run the shipped validator so every session applies the same checks:
   
   ```bash
   python3 <this-skill-dir>/assets/validate_manifest.py <manifest> --scene-ids <ids from model constants>
   ```
   It only validates **format, never content truth**. Cross-check structure with the static schema commands `sensecraft-cli app fields` and `sensecraft-cli app blocks` — they are zero-network, run them bare (they take **no** `--sandbox` flag).
5. Snapshot the manifest (archive copy when there is no git).

## Writing the prose

Published text is what plaza users actually read — mine it from the project, but author it deliberately.

- `excerpt`: what the app is, who it is for, why it is interesting. Nothing else — no hardware/framework spec dump, no deploy steps, no version history or changelog.
- `deployment.blocks` prose: short onboarding steps — necessary prep, main actions, how to start and stop. Include a step only if the feature exists. Target structure: an intro of at most two short paragraphs, then one step per distinct user action (typically 3–5 steps). No full menu enumeration, no exhaustive troubleshooting, no spec tables, no restating the excerpt inside steps. Text blocks render Markdown — images and external video links are available; see [reference-tool-blocks.md](../reference-tool-blocks.md#content-format).
- Review once before every push: would a first-time user find every sentence clear, with nothing repeated?

## Workflow B — create (one-go)

1. **Preflight**: Run `sensecraft-cli auth info`. Need an active SenseCAP Access Key. If missing, ask the user to run `sensecraft-cli auth init --kind sensecap` locally. Never run `auth init` for them.
2. **Upload assets**, writing each returned `file_url` back into the manifest entry:
   
   - Images: `sensecraft-cli file upload --path <img> --type 1 --for-app` (unlinked, 5 MB cap). Keep manifest order; index 0 = cover.
   - Firmware/binaries for blocks: check binaries for embedded credentials before uploading — build outputs can compile tokens in; if anything shows up, flag it and let the user decide. Whatever the upload decision, a block the flow needs stays in the doc: fill its config per [reference-tool-blocks.md](../reference-tool-blocks.md) and mark unfilled values as pending (e.g. firmware built locally by the end user). Upload itself: `sensecraft-cli file upload --path <fw> --type 4` → put the URL into the block's `config.firmware_url` (+ `firmware_size`).
3. **Build the payload** from the manifest per the projection in [reference-manifest.md](reference-manifest.md#manifest--cli-payload). Omit keys that have no values — the CLI rejects an empty `images: []` but accepts the key absent. Write the projection to `.sensecraft/sensecraft-app.payload.json` (transient — delete after the push).
4. **Create** — deliberately not L2, no `--yes`:
   ```bash
   sensecraft-cli app create --from-json .sensecraft/sensecraft-app.payload.json
   ```
   Expect `app_id` + `console_url` + `detail_url` on stdout. If the **`app_id` or URLs** are missing or empty, run `sensecraft-cli app list --mine --name "<app_name>"` BEFORE any retry — never blind re-create. Echoed fields differing from what you sent are not by themselves a failure signal.
5. **Record**: write `app_id`, `created_at` into `identity`; set `sync.last_pushed_*` (revision, RFC3339 time, payload hash); archive snapshot.
6. **Report**: app name, `app_id`, `console_url`, `detail_url`, image/BOM/block counts, `sync.local_revision` as the version — and the lifecycle state: created **private** (`be_public=0`), a draft only the user can see, already test-deployable to their own devices, not on the plaza (publish = parent skill). No cover yet → note one is required before publishing. Never describe a private or pending app as 已上线 / published.

**Partial failure resume**: once `identity.app_id` is set, creation happened. Any later failure (upload, follow-up write) → continue with Workflow C against that `app_id`. Never start Workflow B over for the same manifest.

## Workflow C — update (local-first)

The manifest leads, the remote follows. Every remote write below is L2: **one confirmation per batch, on a separate chat turn** ([`../../confirmation.md`](../../confirmation.md)) — the user's first request is intent, not confirmation.

1. **Edit the manifest first** (user or agent), bump `sync.local_revision`, validate as in Workflow A step 4.
2. **Preflight + env guard**: `auth info` — `remote_type`/`org_id` must match `identity`. Mismatch → stop; the user either switches auth or explicitly adopts the new env into the manifest.
3. **Drift check**: `sensecraft-cli app view --id <app_id> --sketch`, project the remote into the same sections the manifest covers, compare with `sync.last_pushed_hash` (algorithm pinned in [reference-manifest.md](reference-manifest.md#sync--local-vs-remote-bookkeeping)). Same → push. Different → the app was changed outside this manifest: show the field-level diff and use your judgment — merge what doesn't conflict, bring real conflicts to the user as a concrete proposal. Never silently overwrite someone else's edits.
4. **Confirm once for the whole batch**: state environment, app name + id, and every pending change (fields, images, BOM rows, blocks). Wait for the user's `确认` / `yes` reply.
5. **Push only what changed**:
   | Changed section | Command |
   |---|---|
   | info fields and/or image set | `sensecraft-cli app update --id <app_id> --from-json .sensecraft/sensecraft-app.patch.json --yes` (sparse OK — CLI merges missing info keys from the sketch) |
   | BOM | `sensecraft-cli app bom set --id <app_id> --from-json .sensecraft/sensecraft-app.materials.json --yes` |
   | deployment blocks | include `deployment` in the `app update` patch |
   - **Images**: upload new/changed files with `--type 1 --for-app` (compare `sha256` in the manifest), then send the **full** `images[]` URL array in the update patch. Never use `--app-id` attach here — it prepends one image and scrambles the manifest order (see [../commands.md](../commands.md)).
6. **Verify** per the parent rule: success may return `{}`, so re-read with `app view --id <app_id> --sketch` (and `app bom list --id <app_id>` when BOM changed).
7. **Record**: `last_pushed_revision = local_revision`, new hash + time; archive snapshot; suggest `git add` when in git.
8. **Published app**: the update saves a **draft** — the plaza copy is unchanged until publish (parent skill). Under review (`be_public=2`) the CLI refuses — stop and say so.

## Field and block rules

- App info + BOM: [../reference-app-fields.md](../reference-app-fields.md)
- Deployment layout + tool-block configs: [../reference-tool-blocks.md](../reference-tool-blocks.md)
- CLI flags for every command above: [../commands.md](../commands.md)
