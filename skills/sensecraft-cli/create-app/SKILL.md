---
name: create-app
description: Turn a local project into a deployable CodeCraft Hub app, or update an app managed by codecraft-app.json. Discover project facts, resolve user decisions, author and review app copy, configure deployment tool blocks, verify BOM products through the API, then validate and upload the local manifest. Use for requests to upload a project or repo as a CodeCraft Hub or SenseCraft app. App publishing, unpublishing, deleting, and cloning use the parent skill.
---

# Create a CodeCraft Hub app

Deliver a reviewed `codecraft-app.json` and a private CodeCraft Hub app whose deployment instructions work through the available tool blocks. Follow discovery → synthesis → decisions → manifest → review → upload → verification. A saved draft with missing deployment inputs is an incomplete result; report those inputs explicitly.

The manifest is the source of truth for app values. Project evidence and review notes explain those values; they are not a second payload. An existing `identity.app_id` always means update/resume, never another create.

## Read references before authoring

Read each applicable reference **in full before drafting or changing the corresponding information**, including on an update. A link, search snippet, template, static schema, or remembered config does not replace the read. Reuse a reference already read in this session unless it changed. If a required reference is unavailable, continue independent discovery and report the gap; do not guess the affected fields.

| Before this work | Required reading / inspection |
|---|---|
| Any CLI work | [Parent skill](../SKILL.md), [app commands](../references/app-commands.md); `auth info` before cloud calls |
| Project discovery | [Project discovery](references/project-discovery.md), then the relevant project files themselves |
| App fields, scene, images, permissions, BOM rows | [App fields and BOM](../references/app-fields.md); `app fields`; `model constants` for actual scene options |
| Activity participation | [Activity selection](../references/app-fields.md#activity-selection-during-app-authoring), [activity command](../references/app-commands.md#app-activity-list) |
| Any human-facing copy | [Writing and review](references/content-review.md), plus project evidence for the claims |
| Deployment plan or tool-block config | [Tool blocks](../references/tool-blocks.md), including device-specific rules; `app blocks` |
| Model-backed tool needing model selection/metadata | [Models](../references/models.md), [model commands](../references/model-commands.md), [model fields](../references/model-fields.md); verify the selected model through the supported API reads |
| Manifest creation, projection, sync bookkeeping | [Manifest contract](references/manifest.md), [template](assets/manifest-template.json) |
| Asset upload | [File upload](../references/files.md), [file command flags](../references/file-commands.md) |
| Update of an existing remote app | [Confirmation protocol](../references/confirmation.md); review the concrete batch before requesting any outstanding approval |

`app fields` and `app blocks` are static, zero-network commands: run them bare, without `--sandbox`. They describe structure, not live catalog matches. `model constants` and `app bom search` are cloud reads; sandbox samples are not API confirmation.

## 1. Discover the project and current app state

Read [project-discovery.md](references/project-discovery.md). Mine this conversation first, then inspect the actual project: purpose and audience, implemented behavior, supported hardware, build/deployment configuration, artifacts, runtime setup, limitations, assets, and license. README claims are candidates; cross-check them with implementation and configuration before using them in app copy.

Keep concise evidence and open decisions in `.sensecraft/<manifest-stem>.review.md` (for example `.sensecraft/codecraft-app.review.md`). Record file locations or conversation decisions, not credentials. Use it to carry unresolved facts, the deployment plan, BOM lookup evidence, artifact paths/hashes, and the final review through a resumed session. Final app values live in the manifest.

- Run `git rev-parse --is-inside-work-tree` once and reuse the result for versioning.
- Prefer `codecraft-app.json`; otherwise use a single `codecraft-app.<slug>.json` with `manifest_version`, `identity`, and `sync`. Exclude `*.payload.json`, `*.patch.json`, and `*.materials.json`. Several candidates → ask which app. An explicit request for another app → use a separate slugged manifest, preserve the existing one.
- Also discover legacy `sensecraft-app.json` / `sensecraft-app.<slug>.json` with the same shape checks and transient exclusions. Migrate the chosen manifest to the corresponding `codecraft-app` filename, preserving its contents, especially `identity.app_id` and `sync`. Never overwrite an existing destination: resolve competing manifests first. Preserve old archive snapshots; carry forward any existing review/artifact notes and sync baseline. A filename change is not a new app.
- Preserve an invalid pre-create manifest in `.sensecraft/archive/` before rebuilding it from the template. Explain the fault and run `app list --mine --name "<app_name>"` before creating. If an app ID exists, preserve it and repair/resume that app.
- Run `sensecraft-cli auth info` before cloud reads. CodeCraft Hub requires Global and an active SenseCAP Access Key. If auth is missing, use the split `auth init` flow in [`../SKILL.md`](../SKILL.md): `--site Global --no-wait`, show `verification_url` and end the turn, then `--device-code` after the user authorizes. Do not run bare `auth init`. Do not invent account, scene, or catalog values before authentication succeeds.

## 2. Synthesize fields, deployment, and BOM

Read the applicable references in the table **before** doing each part. Build evidence-backed candidates first; serialize after material decisions are resolved.

**App information.** Read [app-fields.md](../references/app-fields.md) in full before synthesizing these fields, including its activity selection guidance. Keep the project's name unless the user chooses a change. Write an excerpt for the intended reader: what it does, who uses it, and the useful outcome. Propose a scene label and its real ID from `app_scenarios_array`. Preserve known choices for `allow_clone` and `allow_commerce`; if unknown, ask rather than inherit template defaults or infer permissions from the license. Resolve author from the authenticated account when established; identify the project license separately. Inspect cover/gallery candidates and keep their order explicit. No cover is allowed for a private draft; note the publishing requirement.

**Activity participation.** Use the project understanding from discovery and follow [app-fields.md § Activity selection](../references/app-fields.md#activity-selection-during-app-authoring) to decide whether to join an activity. Record the resolved selection in `info.activity_id` and the rationale in the review.

**Deployment.** Read all of [tool-blocks.md](../references/tool-blocks.md), then map the complete first-time user journey to its supported blocks. For each action, record prerequisites, the chosen tool, verified config/artifact source, and the observable result. Inspect real artifact existence, target/architecture, build configuration, and required runtime inputs. A firmware file does not establish a model-deploy step; source code alone does not establish a flashable image. Place each tool directly after its instructions in a flat `deployment.blocks` array.

Use available artifacts or a documented reproducible build when feasible within the user's scope. If an artifact, board variant, model, container version, or necessary action is unresolved, explain the specific gap and propose choices. Keep needed blocks in the local draft with pending values; never replace them with prose that pretends the deployment works. If no tool supports a required action, say so and agree on a supported path or an explicitly incomplete draft. Do not claim deployability until every required action has a usable path.

**BOM.** Follow [app-fields.md § Catalog verification](../references/app-fields.md#catalog-verification-before-authoring-a-bom). Identify devices and app-specific accessories from project evidence; exclude ordinary cables, computers, and generic tools by default. Query `app bom search --name "<product or SKU>"` for each catalog candidate, broadening the name if necessary. Match the exact variant. Copy returned `sku` to manifest row `id`, along with the returned name and purchase URL; derive quantity and a useful role note from the project. Do not use a catalog's internal `id` as the SKU.

Record the query and selected result in review notes. Multiple plausible variants → ask. A successful search with no matching product permits a clearly labeled custom row without `id`; API/auth failure means unverified, not no match. Never present custom rows or sandbox output as API-confirmed. No-hardware projects may have an empty BOM with that reason stated. After upload, checking saved rows is a separate verification from catalog matching.

## 3. Resolve uncertain decisions

Present a concise project summary, proposed fields, activity selection or reason for none, deployment actions/tools, BOM matches, and actual gaps. Batch questions about consequential choices: name/language/audience when unclear, scene, clone/commercial permissions, uncertain activity eligibility/selection, hardware variant and BOM scope/quantity, assets, or an unsupported deployment path. Include a recommendation and why; reuse answers already in the conversation.

Ask only for uncertainty that affects correctness or the user's choice. Do not require an arbitrary one-round limit: a new material conflict can need a follow-up. Continue independent preparation while waiting, but do not commit guessed decisions or perform dependent uploads. Review is not an extra approval gate when the requested creation and all necessary decisions are already authorized.

## 4. Build the local manifest

Read [manifest.md](references/manifest.md) and the template. Populate the canonical manifest with resolved fields, optional activity selection, ordered images, the authored intro and deployment blocks, and confirmed BOM rows. Fresh create: `identity.app_id: null`, `sync.local_revision: 1`; fill environment/account from auth. Do not send the manifest itself to the API.

- Hash the exact local image and artifact bytes to be uploaded. Keep firmware/model/archive paths, sizes, hashes, and destination block IDs in review notes; those paths do not belong in public tool configs.
- Local images have `file_url: null` until uploaded. A pending tool URL is not a fake `https://example.com/...` link. Explain every pending input in review notes and the user summary.
- Use the field locations from the references: every tool, including `flash-firmware`, stores its settings inside one `block.config` object. Do not repeat them on the block. The ordered blocks array is flat; tool settings are nested. SKU and `uniform_type` are distinct. Sizes are strings.
- Store no access keys, device keys, SDK tokens, or personal credentials in the manifest, notes, or distributed artifacts. Check candidate archives/binaries for embedded credentials; if found, identify the affected artifact without echoing the secret and resolve a clean distribution path with the user before upload.
- After gather and each successful push: in git, keep the canonical manifest reviewable in the working tree and follow the user's established Git workflow; outside git, also write an immutable `.sensecraft/archive/<manifest-stem>.YYYYMMDD-HHmmss.json` snapshot. Avoid overwriting a same-second snapshot. Retain the verified sync baseline per the manifest contract; do not hand versioning commands to the user.

### Markdown sources

Read [tool-blocks.md § Content format](../references/tool-blocks.md#content-format) and [app-commands.md § app doc compile](../references/app-commands.md#app-doc-compile) before compiling. Author only the reader-facing content; do not upload a developer README wholesale. `app doc compile --from-md <file>` is local and does not upload or insert tools. Inspect its `deployment`, `warnings`, and `pending_uploads`, then merge the compiled text and tool blocks **at their intended positions in the manifest**. Step headings such as `## Step: Flash` become steps; a plain `h2` does not.

Push the final manifest projection with `--from-json`. Combining `--from-md` again would append JSON blocks after the compiled document and can duplicate/reorder the reviewed content. JSON text `content` uses HTML/plain text, never raw Markdown markers; `code` content is plain source. Local media paths must be uploaded and replaced with real URLs before submission.

## 5. Review before upload

Read [content-review.md](references/content-review.md) before drafting copy and apply its full review to the assembled manifest before every push. Review the actual name, excerpt, introduction, step titles/instructions, and BOM notes as a first-time reader. Fix unsupported claims, ambiguity, repetition, missing prerequisites, and instructions that do not match the adjacent tool. Run this editorial pass separately from format validation.

```bash
python3 <this-skill-dir>/assets/validate_manifest.py <manifest> --scene-ids <comma-separated-ids-from-model-constants>
```

Fix format violations before any upload or app write. This validator does **not** prove factual accuracy, catalog identity, tool compatibility, or deployability. Review its warnings: planned uploads may remain pending until the next stage; missing deployment artifacts/configs remain blockers to a deployable result.

Update the review notes with corrections, evidence, remaining gaps, and readiness. Show the user the final name/excerpt, permission choices, activity name/ID and rationale or reason for no activity, ordered deployment actions and tools, BOM including custom/unverified status, selected assets, manifest path, and whether required deployment inputs are complete. Link the local manifest and review for full inspection. If all decisions are settled and creation is authorized, proceed without demanding another confirmation. An upload of a knowingly incomplete draft needs the user's choice of that outcome.

## 6. Upload and create once

1. Read the upload references. Run `sensecraft-cli auth info`. If no Access Key is active, use the split `auth init` flow in [`../SKILL.md`](../SKILL.md). Do not run bare `auth init`. Recheck the environment and that selected files still match their reviewed hashes. Upload only reviewed assets; reuse unchanged URLs. Images: `file upload --path <image> --type 1 --for-app`. Firmware: `--type 4`. Other artifacts: select the actual type/flags from the reference. App images use at most 16 JPG/JPEG/PNG/GIF/WebP files, each ≤5 MB; other artifacts are ≤100 MB.

   For Muse Gadget, upload the application image built as `muse-gadget.bin` and linked at `0x20000`. Give each app its own requested name:

   ```bash
   sensecraft-cli file upload --path <build-dir>/muse-gadget.bin --type 4 --name <project>-muse-gadget.bin
   ```

   The server may rewrite part of that name. `firmware_url` must be the returned `file_url`.

   For Muse Gadget firmware, scan the exact file selected for upload before `file upload --type 4`. A complete SDK token is `mgst_` followed by 43 characters from `[A-Za-z0-9_-]`. The short `mgst_` prefix used by firmware format validation is not a credential match. Keep the scan quiet so matching values stay local:

   ```bash
   grep -aqE 'mgst_[A-Za-z0-9_-]{43}' <firmware-file>
   ```

   Exit code **1** means no complete token was found and this check passes. **0** means a token pattern was found: rebuild without the token and scan again. **2 or higher** is a path or read error to fix before retrying. Report only pass, match, or check error. Do not print matching values.
2. Write returned `file_url` values into the canonical manifest as each upload succeeds, keeping image order. Update the intended tool/media fields and the artifact ledger. For firmware fill `config.firmware_url`, `config.firmware_name`, and string `config.firmware_size` on the target block. Optional `file upload --name` only sets the name sent to the server. The server may rewrite part of that name for security, so `firmware_url` and every other stored URL must be the returned `file_url`. Do not build a URL from `--name` or from stdout `file_name`. Never expose temporary signed upload URLs in review prose.
3. Validate again, inspect the final projection, and reassess readiness. Required deployment inputs must now be filled unless the user chose an incomplete draft. Apply the [activity submission rules](../references/app-fields.md#activity-selection-during-app-authoring) when relevant and follow [manifest → CLI payload](references/manifest.md#manifest--cli-payload). Omit local bookkeeping; images become URLs with `pic_url = images[0]`; no images → omit both keys, never `images: []`. Never send `be_public` or `privacy`.
4. Write `.sensecraft/<manifest-stem>.payload.json`, then create **without `--yes`**:
   ```bash
   sensecraft-cli app create --from-json .sensecraft/codecraft-app.payload.json
   ```
   Parse successful stdout for `app_id`, `console_url`, and `detail_url`. Record the ID and local UTC `created_at` immediately, before verification. On any ambiguous failure or missing ID/URLs, run `app list --mine --name "<app_name>"` before deciding whether creation happened; do not blindly retry. An internal follow-up save may fail after creating the app: recover that ID and resume by update.
5. Re-read `app view --id <app_id> --sketch` and `app bom list --id <app_id> --sketch`. Compare intended fields, image order, blocks/configs, and BOM rows, accounting for documented server normalization. For an app never published, the default `app view` returns only its name; `--sketch` reads the saved draft. Verify any requested activity association per the activity reference. A create response alone is not proof all sections saved. On a mismatch preserve the ID and report/repair through the update path.
6. Only after verification, advance `sync.last_pushed_*` using the manifest contract and snapshot. Delete only transient payloads created by this session after success; preserve prior files and failure evidence.
7. Report app name, ID, returned editor/detail links, counts, manifest revision/path, activity choice/result, and readiness. State **private draft (`be_public=0`)**; private and deployment-ready are separate facts. Missing cover affects publishing; missing tool inputs affect deployment. Never claim the hardware was flashed or the app is public. Publish/unpublish/delete/clone use [apps.md](../references/apps.md).

## Existing app: update or resume

Read the references for the affected sections before editing them. Preserve the same discovery, synthesis, and review standards, scaled to the requested change.

Handle activity participation on updates per [app-fields.md](../references/app-fields.md#activity-selection-during-app-authoring) and the manifest projection rules.

Have the user close this app's CodeCraft Hub editor page before CLI updates, then reopen it after verification. An open editor saves the copy it loaded and can replace changes made through the CLI meanwhile.

1. Edit the manifest first, increment `sync.local_revision`, and validate. Recheck auth: `identity.remote_type` and `org_id` must match. A mismatch requires switching auth or an explicit decision to adopt the environment; do not write across environments silently.
2. Read `app view --id <app_id> --sketch`. Under review (`be_public=2`) → stop and explain the lock. Compare with the last verified baseline per [manifest sync](references/manifest.md#sync--local-vs-remote-bookkeeping). Detect outside edits, merge non-conflicting changes, and present conflicts as concrete choices. No verified baseline after a partial create → inspect current remote state and reconcile; never assume it is empty.
3. Complete local preparation and review before any outstanding L2 confirmation. Describe environment, app name/ID, and the entire pending batch. Follow the [confirmation protocol](../references/confirmation.md), honoring valid approval already given for that batch; do not ask per field/upload/BOM row.
4. Upload changed assets and send only changed sections, using session-owned transients under `.sensecraft/`:

   | Change | Command |
   |---|---|
   | Info, images, deployment | `app update --id <app_id> --from-json <patch-path> --yes` |
   | BOM | `app bom set --id <app_id> --from-json <materials-path> --yes` |

   Images require the full ordered URL array; upload with `--for-app`, not `--app-id` (attach prepends a cover). Deployment updates carry the full intended `deployment`; do not accidentally drop unchanged tools.
5. Re-read sketch and, when changed, `app bom list --id <app_id> --sketch` even if the write returns `{}`. Advance sync baseline and snapshot only for verified success. On partial failure keep the known ID/upload URLs and reconcile the saved sections before retrying. A published app's update saves a draft; the public copy changes only through the parent publish workflow.
