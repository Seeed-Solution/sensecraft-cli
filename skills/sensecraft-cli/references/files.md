# SenseCraft File Upload

Auth: **Access Key**. The China site supports SenseCraft Data only; file upload follows SenseCraft AI and CodeCraft Hub, so use Global. Parent: [`../SKILL.md`](../SKILL.md).

## Agent rules

1. Run `sensecraft-cli auth info` first.
2. Use the correct `--type`; do not invent a file type:
   - `1` image — **≤5 MB** (model `pic_url` and app `images[]`); JPG/JPEG/PNG/GIF/WebP; no SVG
   - `2` generic file / model (default; **≤100 MB**)
   - `3` app (**≤100 MB**)
   - `4` firmware (**≤100 MB**)
   - `5` static resource (**≤100 MB**; **`--flow-id` required**)
   - `6` archive / ZIP (**≤100 MB**)
3. Use `--checksum` when the downstream payload needs an MD5 (model uploads usually do).
4. **Attach before upload.** Types `1`/`2` refuse the OSS POST without association. If the user provides a file without a model/app id, **ask first** (do not call upload):
   - Existing model file: `model update --id <model_id> --path <file> --yes` or `model upload --path <file> --id <model_id> --yes` (auto-increments patch version, same as Console).
   - App cover: `file upload --path <img> --type 1 --app-id <app_id> --yes` (or unlinked `--type 1 --for-app`; max **16** images).
   - Model example image (`pic_url`): `file upload --path <img> --type 1 --unlinked` (**5 MB**), then put `file_url` into model JSON as `pic_url`.
   - Brand-new model binary only: `model upload --path <file> --unlinked` (or `file upload --type 2 --unlinked`) then put `file_url` into `model create --from-json`. Never bare `model upload --path` / `file upload --path` — CLI exits non-zero (`ErrAttachRequired`).
5. Treat returned signed/upload URLs and private file URLs as secrets.
6. Upload alone is not L2; attaching to a model/app (`--id` / `--model-id` / `--app-id`) **is L2**. Follow [confirmation.md](confirmation.md): ask first, wait for `确认`, then `--yes`. Never pass `--yes` on the same turn as the first request. `--unlinked` is an explicit orphan opt-in (not L2).
7. Do not pass oversized files or more than 16 app image URLs; CLI rejects them before upload/update.
8. Types `3`–`6` have no model/app attach path and upload directly (no `--unlinked` needed).
9. **Security gates**: empty (0 B) files rejected; type 2–6 ≤ **100 MB**; every image ≤ **5 MB**. The CLI does not count uploads. A rate limit comes back as `11209` or `11210`.
10. **The access name is the server's.** The filename used to open or download the file is the object name in the returned `file_url`. A name specified for the upload is not necessarily that download name. Do not build a URL from the local file name or from stdout `file_name`.

## Examples

```bash
# Existing private model: attach + bump version (preferred)
sensecraft-cli model upload --path ./model.rknn --id <model_id> --yes
# equivalent: sensecraft-cli file upload --path ./model.rknn --model-id <model_id> --yes

# App cover image (5 MB)
sensecraft-cli file upload --path ./cover.png --type 1 --app-id <app_id> --yes
# Unlinked app image for later images[] JSON:
sensecraft-cli file upload --path ./cover.png --type 1 --for-app

# Model inference example image → pic_url (5 MB)
sensecraft-cli file upload --path ./demo.png --type 1 --unlinked

# Unlinked only when creating a brand-new model (then put file_url into model create)
sensecraft-cli model upload --path ./model.rknn --unlinked

# Firmware, static resource, and archive (no model/app attach)
sensecraft-cli file upload --path ./firmware.bin --type 4 --checksum
sensecraft-cli file upload --path ./guide.pdf --type 5 --flow-id <flow_id>
sensecraft-cli file upload --path ./dataset.zip --type 6 --checksum

# Offline exercise: no network
sensecraft-cli file upload --path ./fixture.bin --type 2 --checksum --unlinked --sandbox
```

Success JSON contains `file_name`, `file_type`, `file_url`, `size`, optional `checksum`, and `attached`. `file_url` is the access link, and the object name in it is the name the server stored. `file_name` is the name sent with the upload; it is not necessarily the downloaded file name.
When attached to a model it also includes `model_id`, `previous_version`, `version`. Explicit `--unlinked` / `--for-app` sets `attached: false` and a `hint`.

## Model shortcut

For model artifacts, this wrapper fixes `type=2` and enables checksum. Pass `--id` to attach and bump version:

```bash
sensecraft-cli model upload --path ./model.rknn --id <model_id> --yes
```

See [file-commands.md](file-commands.md) for the flag reference.
