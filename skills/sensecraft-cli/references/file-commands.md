# sensecraft-files — CLI commands

Binary: `sensecraft-cli`. Auth: Access Key (`auth init`).

## `file upload`

Acquires an OSS POST policy, uploads the local file with multipart/form-data, and prints the permanent `file_url`.

| Flag | Required | Default | Notes |
|------|----------|---------|-------|
| `--path` | yes | — | Local file |
| `--type` | no | `2` | `1=image`, `2=file/model`, `3=app`, `4=firmware`, `5=static resource`, `6=archive/zip` |
| `--flow-id` | type 5 | — | Required for static resources; omitted for other types |
| `--checksum` | no | false | Compute local MD5 and include it in output |
| `--model-id` | no | — | Attach to this model (implies type=2 + checksum; **L2 `--yes`**; bumps version) |
| `--app-id` | no | — | Attach as app cover (implies type=1, **5 MB**; **L2 `--yes`**; app `images[]` max **16**) |
| `--for-app` | no | false | Unlinked type-1 for later app `images[]` JSON (same **5 MB** cap as every image) |
| `--unlinked` | types 1–2 | — | Explicit orphan opt-in for later create/update JSON; required when no `--model-id`/`--app-id`/`--for-app` |
| `--yes` | with attach | — | Required when `--model-id` or `--app-id` is set |
| `--sandbox` | no | false | No network; return sample JSON |

**Size / format:**

| type | Max size | Formats / notes |
|------|----------|-----------------|
| `1` image (model `pic_url` and app `images[]`) | **5 MB** | JPG/JPEG/PNG/GIF/WebP (SVG rejected); app `images[]` max **16** |
| `2`–`4`, `6` | **100 MB** | suffix not checked locally |
| `5` static resource | **100 MB** | requires `--flow-id` |

`app create/update --from-json` with `images[]` also enforces **max 16** URLs (Console `maxImageCount`).

`--model-id` and `--app-id` are mutually exclusive. Types `1`/`2` without attach/`--for-app`/`--unlinked` exit non-zero **before** the OSS POST (`ErrAttachRequired`). Types `3`–`6` upload directly. Type `5` without `--flow-id` exits before acquire.

```bash
sensecraft-cli file upload --path ./cover.png --type 1 --for-app
sensecraft-cli file upload --path ./cover.png --type 1 --app-id <app_id> --yes
sensecraft-cli file upload --path ./model-demo.png --type 1 --unlinked
sensecraft-cli file upload --path ./model.rknn --type 2 --checksum --unlinked
sensecraft-cli file upload --path ./model.rknn --model-id <model_id> --yes
sensecraft-cli file upload --path ./application.zip --type 3 --checksum
sensecraft-cli file upload --path ./firmware.bin --type 4 --checksum
sensecraft-cli file upload --path ./manual.pdf --type 5 --flow-id <flow_id>
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
