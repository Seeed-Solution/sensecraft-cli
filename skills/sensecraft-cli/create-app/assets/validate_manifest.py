#!/usr/bin/env python3
"""Validate a CodeCraft Hub app manifest (create-app skill).

Mechanical checks only — structure, field limits, image constraints, BOM and
block shape. This validates FORMAT, never content truth: image content,
licensing, and factual accuracy remain the agent's job (see SKILL.md and
references/content-review.md). A pass with pending inputs is not deployability.

Usage: python3 validate_manifest.py [manifest-path] [--scene-ids 1,5,...]
       (both "--scene-ids 1,5" and "--scene-ids=1,5" work; a bare or
       unparsed --scene-ids flag is a hard error, never a silent skip)

Exit 0 = pass (warnings allowed); exit 1 = one or more violations.
"""
import hashlib
import json
import os
import re
import sys
from datetime import datetime
from urllib.parse import urlsplit

IMAGE_EXTS = {".jpg", ".jpeg", ".png", ".gif", ".webp"}
IMAGE_MAX_BYTES = 5 * 1024 * 1024
IMAGES_MAX = 16
NAME_MAX = 60      # UTF-16 code units, Console-aligned
EXCERPT_MAX = 300  # UTF-16 code units
SCENE_RE = re.compile(r"^[0-9]+$")
TEXT_BLOCK_TYPES = {"step", "paragraph", "h1", "h2", "h3", "h4", "h5", "h6",
                    "bullet", "number", "todo", "quote", "code", "divider"}
# Modern deployment.blocks uses one config envelope for every tool. Keep these
# config-relative fields aligned with internal/service/app_tool_blocks.go.
TOOL_CONFIG_FIELDS = {
    "flash-firmware": {"device_id", "device_type", "flash_mode", "firmware_url", "firmware_name", "firmware_size", "flash_address", "project_zip_url", "project_zip_name", "project_zip_size", "precheck_fields", "muse_gadget", "full_flash"},
    "deploy-ai-model": {"model", "device_type", "powerMode", "conf", "iou", "threshold", "triggerAction", "deploy_tool_entry", "firmware_url"},
    "ai-vibration": {"detectionType", "deviceType", "windowSize", "isPower", "gpio", "dStatus"},
    "deploy-container-app": {"app_id", "app_key", "app_display_name", "app_icon", "app_description", "architecture", "architectures", "version", "version_options", "install_form", "merged_service", "user_override_flags", "form_field_values", "raw_by_version", "detail_form_fields"},
    "node-red": {"flow_data_url", "flow_data", "model_id", "model_data"},
    "aily-blockly": {"project_id", "device_id"},
}
FLASH_ADDRESSES = {"0x0", "0x4000", "0x10000", "0x90000"}
# Keep aligned with museDeviceTable (muse: true) and tool-blocks.md.
MUSE_GADGET_SKUS = {
    "114993068",  # SenseCAP Indicator
    "113991315", "113991275",  # SenseCAP Watcher
    "100073581",  # reTerminal E1001
    "100017057",  # reTerminal E1002
    "100069905",  # XIAO 1.47" Touch Display
    "100056818",  # reTerminal Sticky
}
BOM_QUANTITY_MAX = 999
SHA256_RE = re.compile(r"^[0-9a-f]{64}$")
QUANTITY_RE = re.compile(r"^[1-9][0-9]*$")
# Shapes that must never appear in fields that get published. Matches
# credential VALUES (known prefixes, key=value assignments), never the mere
# words "token"/"password" — instructions about creating a token are legit.
SECRET_VALUE_RE = re.compile(
    r"(mgst_[A-Za-z0-9_-]{6,}|sk-[A-Za-z0-9_-]{8,}|bearer\s+[A-Za-z0-9._-]{8,}"
    r"|(?:access[_-]?key|password|passwd|secret|api[_-]?key)\s*[:=]\s*\S{6,})",
    re.IGNORECASE,
)
LONG_HEX_RE = re.compile(r"\b[0-9a-f]{40,}\b")
SECRET_FIELDS = {"access_key", "api_key", "password", "passwd", "secret",
                 "device_key", "sdk_token", "access_token", "refresh_token"}
INPUT_PLACEHOLDER_RE = re.compile(r"^(?:<[^>]+>|\$\{[^}]+\}|YOUR_[A-Z0-9_]+)$")
# Keep these aligned with rawMarkdownPatterns in internal/service/app_doc.go.
RAW_MARKDOWN_RE = re.compile(
    r"\*\*[^*\n]+\*\*|__[^_\n]+__|~~[^~\n]+~~|`[^`\n]+`"
    r"|^#{1,6}[ \t]|^[ \t]*[-*+][ \t]|^[ \t]*[0-9]+\.[ \t]",
    re.MULTILINE,
)


def utf16_len(s):
    return len(s.encode("utf-16-le")) // 2


def http_url(value):
    if not isinstance(value, str) or any(character.isspace() for character in value):
        return False
    try:
        parsed = urlsplit(value)
        return parsed.scheme in ("http", "https") and bool(parsed.hostname)
    except ValueError:
        return False


def timestamp(value):
    if not isinstance(value, str) or "T" not in value:
        return False
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00")).utcoffset() is not None
    except ValueError:
        return False


def reject_nonfinite(_value):
    raise ValueError("non-finite numbers are not valid JSON")


def main():
    argv = sys.argv[1:]
    path, scene_ids = "codecraft-app.json", set()
    i = 0
    while i < len(argv):
        a = argv[i]
        if a == "--scene-ids":
            if i + 1 >= len(argv) or argv[i + 1].startswith("--"):
                print("ERROR: --scene-ids needs a comma-separated value (or use --scene-ids=...)")
                return 2
            scene_ids = set(argv[i + 1].split(",")) - {""}
            i += 2
        elif a.startswith("--scene-ids="):
            scene_ids = set(a.split("=", 1)[1].split(",")) - {""}
            i += 1
        elif a.startswith("--"):
            print(f"ERROR: unknown flag {a!r}")
            return 2
        else:
            path = a
            i += 1

    if any(a == "--scene-ids" or a.startswith("--scene-ids=") for a in argv):
        if not scene_ids or any(not SCENE_RE.fullmatch(value) for value in scene_ids):
            print("ERROR: --scene-ids needs a non-empty comma-separated list of numeric ids")
            return 2

    violations, warnings = [], []

    def vio(msg):
        violations.append(msg)

    def warn(msg):
        warnings.append(msg)

    try:
        with open(path, encoding="utf-8") as f:
            m = json.load(f, parse_constant=reject_nonfinite)
    except (OSError, ValueError) as e:
        print(f"VIOLATION cannot read/parse {path}: {e}")
        return 1
    if not isinstance(m, dict):
        print("VIOLATION manifest must be a JSON object")
        return 1
    base = os.path.dirname(os.path.abspath(path))

    for key in ("manifest_version", "identity", "info", "images", "deployment", "bom", "sync"):
        if key not in m:
            vio(f"missing top-level section: {key}")

    if type(m.get("manifest_version")) is not int or m["manifest_version"] != 1:
        vio("manifest_version must be integer 1")
    if set(m) - {"manifest_version", "identity", "info", "images", "deployment", "bom", "sync"}:
        vio("unsupported top-level fields for manifest version 1")
    for key in ("identity", "info", "deployment", "bom", "sync"):
        if not isinstance(m.get(key), dict):
            vio(f"{key} must be an object")
            m[key] = {}
    for key in ("be_public", "privacy"):
        if key in m or key in m["info"]:
            vio(f"{key} must not appear in the manifest or app info; use the parent lifecycle workflow")

    ident = m.get("identity", {})
    rt = ident.get("remote_type")
    if rt is not None and rt not in ("Global", "China", "Private"):
        vio("identity.remote_type must be Global|China|Private or null")
    app_id = ident.get("app_id")
    if app_id is not None and (not isinstance(app_id, str) or not app_id.strip()):
        vio("identity.app_id must be a non-empty string or null")
    for key in ("org_id", "account"):
        if ident.get(key) is not None and not isinstance(ident[key], str):
            vio(f"identity.{key} must be a string or null")
    if ident.get("created_at") is not None and not timestamp(ident["created_at"]):
        vio("identity.created_at must be an RFC3339 timestamp with timezone or null")

    info = m.get("info", {})
    if set(info) - {"app_name", "excerpt", "scene", "allow_clone", "allow_commerce", "license", "author", "activity_id"}:
        vio("info contains unsupported fields; use the documented app info keys")
    name = info.get("app_name", "")
    if not isinstance(name, str) or not name.strip():
        vio("info.app_name must be a non-empty string")
    elif utf16_len(name) > NAME_MAX:
        vio(f"info.app_name is {utf16_len(name)} UTF-16 code units (max {NAME_MAX})")
    excerpt = info.get("excerpt", "")
    if not isinstance(excerpt, str) or not excerpt.strip():
        vio("info.excerpt must be a non-empty string")
    elif utf16_len(excerpt) > EXCERPT_MAX:
        vio(f"info.excerpt is {utf16_len(excerpt)} UTF-16 code units (max {EXCERPT_MAX})")
    scene = info.get("scene", "")
    if not isinstance(scene, str) or not SCENE_RE.fullmatch(scene):
        vio("info.scene must be a numeric string id from app_scenarios_array")
    elif scene_ids and scene not in scene_ids:
        vio("info.scene is not in the supplied app_scenarios_array ids")
    for flag in ("allow_clone", "allow_commerce"):
        if type(info.get(flag)) is not int or info[flag] not in (0, 1):
            vio(f"info.{flag} must be integer 0 or 1")
    for key in ("license", "author"):
        if not isinstance(info.get(key), str):
            vio(f"info.{key} must be a string (empty is allowed)")
    activity_id = info.get("activity_id")
    if activity_id is not None and (type(activity_id) is not int or not 0 < activity_id <= 2**63 - 1):
        vio("info.activity_id must be a positive integer within the CLI's int64 range, or null/omitted for no activity")

    images = m.get("images", [])
    if not isinstance(images, list):
        vio(f"images must be an array, got {type(images).__name__}")
        images = []
    elif len(images) > IMAGES_MAX:
        vio(f"images must hold at most {IMAGES_MAX} entries, got {len(images)}")
    elif len(images) == 0:
        warn("no images — fine for a private draft (omit the `images` key in the payload; the CLI rejects an empty array); a cover is required before publishing")
    if isinstance(images, list):
        for i, img in enumerate(images):
            if not isinstance(img, dict):
                vio(f"images[{i}] must be an object")
                continue
            p, role = img.get("path", ""), img.get("role", "")
            if not isinstance(p, str) or not p.strip():
                vio(f"images[{i}].path must be a non-empty local path string")
                continue
            if i == 0 and role != "cover":
                vio("images[0].role must be 'cover'")
            if i > 0 and role != "gallery":
                vio(f"images[{i}].role must be 'gallery'")
            ext = os.path.splitext(p)[1].lower()
            if ext not in IMAGE_EXTS:
                vio(f"images[{i}] {p!r}: extension {ext or '(none)'} not in {sorted(IMAGE_EXTS)} (SVG rejected)")
            full = os.path.join(base, p)
            if not os.path.isfile(full):
                vio(f"images[{i}] {p!r}: file not found relative to manifest")
            else:
                size = os.path.getsize(full)
                if size == 0:
                    vio(f"images[{i}] {p!r}: zero-byte file")
                elif size > IMAGE_MAX_BYTES:
                    vio(f"images[{i}] {p!r}: {size} bytes exceeds 5 MB")
                sha = img.get("sha256")
                if sha is not None:
                    if not isinstance(sha, str) or not SHA256_RE.fullmatch(sha):
                        vio(f"images[{i}].sha256 must be a complete lowercase SHA-256 hex digest or null")
                    with open(full, "rb") as image_file:
                        digest = hashlib.sha256(image_file.read()).hexdigest()
                    if digest != sha:
                        vio(f"images[{i}] {p!r}: sha256 mismatch (file changed since hash)")
                else:
                    warn(f"images[{i}] {p!r}: sha256 not computed yet")
            url = img.get("file_url")
            if url is not None and not http_url(url):
                vio(f"images[{i}].file_url must be a valid http(s) URL or null")
            elif url is None:
                warn(f"images[{i}] {p!r}: not uploaded yet (file_url null)")

    blocks = m.get("deployment", {}).get("blocks", [])
    if not isinstance(blocks, list):
        vio("deployment.blocks must be an array")
        blocks = []
    else:
        if not blocks:
            warn("no deployment blocks — author and review the deployment journey before claiming deployability")
        seen = set()
        for index, b in enumerate(blocks):
            label = f"deployment.blocks[{index}]"
            if not isinstance(b, dict):
                vio(f"deployment.blocks[{index}]: must be an object")
                continue
            bid = b.get("id")
            if not isinstance(bid, str) or not bid.strip():
                vio(f"deployment.blocks[{index}]: needs a non-empty string id")
            elif bid in seen:
                vio(f"{label}: duplicate block id")
            else:
                seen.add(bid)
            block_type = b.get("type")
            if not isinstance(block_type, str):
                vio(f"{label}: type must be a supported block type string")
                continue
            if block_type in TEXT_BLOCK_TYPES:
                if not isinstance(b.get("content"), str):
                    vio(f"{label}: text/code content must be a string")
                elif block_type not in ("code", "divider") and RAW_MARKDOWN_RE.search(b["content"]):
                    vio(f"{label}: content contains raw Markdown; compile it with app doc compile before inserting tool blocks")
            if block_type == "step" and (not isinstance(b.get("title"), str) or not b["title"].strip()):
                vio(f"{label}: step block needs a non-empty title string")
            cfg = b.get("config")
            if block_type in TOOL_CONFIG_FIELDS:
                if not isinstance(cfg, dict):
                    vio(f"{label}: tool block needs a config object; flat legacy fields are not rendered")
                    continue
                if TOOL_CONFIG_FIELDS[block_type].intersection(b):
                    vio(f"{label}: tool settings belong inside config; remove flat or duplicated settings")
                if isinstance(cfg.get("config"), (dict, list)):
                    vio(f"{label}: remove the extra config.config wrapper")
            elif block_type not in TEXT_BLOCK_TYPES:
                vio(f"{label}: unsupported block type")
            if block_type == "flash-firmware":
                if "device_id" in cfg and not isinstance(cfg["device_id"], str):
                    vio(f"{label}: config.device_id must be a SKU string")
                elif not cfg.get("device_id"):
                    warn(f"{label}: config.device_id is pending — resolve the exact device SKU")
                for key in ("device_type", "firmware_name", "project_zip_name", "flash_mode", "flash_address", "firmware_url", "project_zip_url"):
                    if key in cfg and not isinstance(cfg[key], str):
                        vio(f"{label}: config.{key} must be a string (empty is allowed for a pending value)")
                mode = cfg.get("flash_mode")
                if mode is None or mode == "":
                    warn(f"{label}: config.flash_mode is pending")
                elif mode not in ("bin", "zip"):
                    vio(f"{label}: config.flash_mode must be bin or zip")
                else:
                    prefix = "firmware" if mode == "bin" else "project_zip"
                    if not cfg.get(prefix + "_url"):
                        warn(f"{label}: config.{prefix}_url is pending — deployment input incomplete")
                for size_key in ("firmware_size", "project_zip_size"):
                    if size_key in cfg and not isinstance(cfg[size_key], str):
                        vio(f"{label}: config.{size_key} must be a string")
                    elif cfg.get(size_key) and not QUANTITY_RE.fullmatch(cfg[size_key]):
                        vio(f"{label}: config.{size_key} must be a positive byte count string or pending empty string")
                address = cfg.get("flash_address")
                if address is not None and address != "" and (not isinstance(address, str) or address not in FLASH_ADDRESSES):
                    vio(f"{label}: unsupported config.flash_address; read tool-blocks.md for device defaults")
                for key in ("muse_gadget", "full_flash"):
                    if key in cfg and type(cfg[key]) is not bool:
                        vio(f"{label}: config.{key} must be a JSON boolean")
                if cfg.get("muse_gadget") is True and cfg.get("full_flash") is True:
                    vio(f"{label}: config.full_flash must be false when config.muse_gadget is true")
                elif cfg.get("muse_gadget") is True or cfg.get("full_flash") is True:
                    device_id = cfg.get("device_id")
                    if not isinstance(device_id, str) or device_id.strip() not in MUSE_GADGET_SKUS:
                        vio(f"{label}: config.muse_gadget and config.full_flash can be true only for a Muse device SKU")
            if block_type == "node-red":
                if cfg.get("flow_data") is not None and not isinstance(cfg["flow_data"], (dict, list)):
                    vio(f"{label}: config.flow_data must be an object or array, not a JSON string")
                if cfg.get("model_data") is not None and not isinstance(cfg["model_data"], dict):
                    vio(f"{label}: config.model_data must be an object or null")
            if isinstance(cfg, dict):
                for key, val in cfg.items():
                    if not key.endswith("_url"):
                        continue
                    if val is None or val == "":
                        warn(f"{label}: config.{key} is empty — pending deployment input")
                    elif not http_url(val):
                        vio(f"{label}: config.{key} must be a valid http(s) URL or a pending empty value")

    materials = m.get("bom", {}).get("materials", [])
    if not isinstance(materials, list):
        vio("bom.materials must be an array")
        materials = []
    else:
        for i, row in enumerate(materials):
            if not isinstance(row, dict):
                vio(f"bom.materials[{i}] must be an object")
                continue
            if set(row) - {"id", "name", "quantity", "note", "buy_url", "price", "pic_url", "image"}:
                vio(f"bom.materials[{i}] contains unsupported fields; project catalog sku into id rather than copying a raw catalog result")
            for forbidden in ("price", "pic_url", "image"):
                if forbidden in row:
                    vio(f"bom.materials[{i}]: {forbidden!r} is not a persistable field (dropped server-side)")
            if "id" in row:  # Seeed catalog row
                if not isinstance(row["id"], str) or not row["id"].strip():
                    vio(f"bom.materials[{i}].id must be a non-empty SKU string; omit id for custom rows")
                for req in ("name", "quantity", "note", "buy_url"):
                    if req not in row:
                        vio(f"bom.materials[{i}] (catalog row): missing {req!r}")
            if not isinstance(row.get("name"), str) or not row["name"].strip():
                vio(f"bom.materials[{i}].name must be a non-empty string")
            quantity = row.get("quantity")
            if not isinstance(quantity, str) or not QUANTITY_RE.fullmatch(quantity) or int(quantity) > BOM_QUANTITY_MAX:
                vio(f"bom.materials[{i}].quantity must be an integer string from 1 to {BOM_QUANTITY_MAX}")
            if "note" in row and not isinstance(row["note"], str):
                vio(f"bom.materials[{i}].note must be a string")
            if "buy_url" in row and row["buy_url"] != "" and not http_url(row["buy_url"]):
                vio(f"bom.materials[{i}].buy_url must be an http(s) URL or an empty string")

    sync = m.get("sync", {})
    lr, lpr = sync.get("local_revision"), sync.get("last_pushed_revision")
    if type(lr) is not int or lr < 1:
        vio("sync.local_revision must be an integer >= 1")
    if type(lpr) is not int or lpr < 0 or (type(lr) is int and lpr > lr):
        vio("sync.last_pushed_revision must be 0..local_revision")
    pushed_hash = sync.get("last_pushed_hash")
    if pushed_hash is not None and (not isinstance(pushed_hash, str) or not re.fullmatch(r"sha256:[0-9a-f]{64}", pushed_hash)):
        vio("sync.last_pushed_hash must be a full sha256:<64 lowercase hex digits> digest or null")
    if sync.get("last_pushed_at") is not None and not timestamp(sync["last_pushed_at"]):
        vio("sync.last_pushed_at must be an RFC3339 timestamp with timezone or null")
    if type(lpr) is int and lpr > 0 and (not app_id or pushed_hash is None or sync.get("last_pushed_at") is None):
        vio("a pushed revision needs app_id, last_pushed_hash, and last_pushed_at")

    def check_public_values(value, location):
        if isinstance(value, dict):
            for key, child in value.items():
                if key.lower() in SECRET_FIELDS and isinstance(child, str) and child.strip() and not INPUT_PLACEHOLDER_RE.fullmatch(child):
                    vio(f"{location}.{key}: credential value must not be distributed (value redacted)")
                check_public_values(child, f"{location}.{key}")
        elif isinstance(value, list):
            for index, child in enumerate(value):
                check_public_values(child, f"{location}[{index}]")
        elif isinstance(value, str):
            if SECRET_VALUE_RE.search(value):
                vio(f"{location}: secret-shaped value in publishable data (value redacted)")
            elif LONG_HEX_RE.search(value):
                warn(f"{location}: long hex literal in publishable data; review whether it is a public hash")

    for section in ("info", "deployment"):
        check_public_values(m[section], section)
    check_public_values(materials, "bom.materials")
    check_public_values([img.get("file_url") for img in images if isinstance(img, dict)], "images.file_url")

    for w in warnings:
        print(f"WARNING   {w}")
    for v in violations:
        print(f"VIOLATION {v}")
    if violations:
        print(f"\nFAIL: {len(violations)} violation(s), {len(warnings)} warning(s)")
        return 1
    print(f"\nPASS: {path} ({len(warnings)} warning(s))")
    return 0


if __name__ == "__main__":
    sys.exit(main())
