# sensecraft-files — CLI commands

Binary: `sensecraft-cli`. Auth: Access Key (`auth init --kind sensecap`).

## `file upload`

Acquires an upload URL, PUTs the local bytes, and prints the permanent `file_url`.

| Flag | Required | Default | Notes |
|------|----------|---------|-------|
| `--path` | yes | — | Local file |
| `--type` | no | `2` | `1=image`, `2=file/model`, `3=app`, `4=firmware`, `5=doc`, `6=archive/zip` |
| `--checksum` | no | false | Compute local MD5 and include it in output |
| `--model-id` | no | — | Attach to this model (implies type=2 + checksum; **L2 `--yes`**; bumps version) |
| `--app-id` | no | — | Attach as app cover (implies type=1, **5 MB**; **L2 `--yes`**; app `images[]` max **16**) |
| `--for-app` | no | false | Unlinked type-1 for app `images[]` (**5 MB**); without it type-1 is **2 MB** (model `pic_url`) |
| `--unlinked` | types 1–2 | — | Explicit orphan opt-in for later create/update JSON; required when no `--model-id`/`--app-id`/`--for-app` |
| `--yes` | with attach | — | Required when `--model-id` or `--app-id` is set |
| `--sandbox` | no | false | No network; return sample JSON |

**Size / format (Console `upload.vue` / `infoForm` / `model/add.vue`):**

| type | Max size | Formats / notes |
|------|----------|-----------------|
| `1` image (default / model `pic_url`) | **2 MB** | JPG/JPEG/PNG/GIF/WebP (SVG rejected); Console model example image |
| `1` + `--for-app` or `--app-id` | **5 MB** | same formats; app `images[]` max **16** |
| `2`–`6` | **100 MB** | per Console accept rules |

`app create/update --from-json` with `images[]` also enforces **max 16** URLs (Console `maxImageCount`).

`--model-id` and `--app-id` are mutually exclusive. Types `1`/`2` without attach/`--for-app`/`--unlinked` exit non-zero **before** PUT (`ErrAttachRequired`). Types `3`–`6` upload directly.

```bash
sensecraft-cli file upload --path ./cover.png --type 1 --for-app
sensecraft-cli file upload --path ./cover.png --type 1 --app-id <app_id> --yes
sensecraft-cli file upload --path ./model-demo.png --type 1 --unlinked
sensecraft-cli file upload --path ./model.rknn --type 2 --checksum --unlinked
sensecraft-cli file upload --path ./model.rknn --model-id <model_id> --yes
sensecraft-cli file upload --path ./application.zip --type 3 --checksum
sensecraft-cli file upload --path ./firmware.bin --type 4 --checksum
sensecraft-cli file upload --path ./manual.pdf --type 5
sensecraft-cli file upload --path ./dataset.zip --type 6 --checksum
```

Output:

Unlinked (`--unlinked` / `--for-app`) output includes `"attached": false` and `hint`. Attached model output:

```json
{
  "file_name": "model.rknn",
  "file_type": 2,
  "file_url": "https://…",
  "size": 123456,
  "checksum": "…",
  "attached": true,
  "model_id": "63754",
  "previous_version": "1.0.0",
  "version": "1.0.1"
}
```
