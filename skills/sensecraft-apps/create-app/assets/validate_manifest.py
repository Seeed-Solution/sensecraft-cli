#!/usr/bin/env python3
"""Validate a sensecraft-app manifest (create-app skill).

Mechanical checks only — structure, field limits, image constraints, BOM and
block shape. This validates FORMAT, never content truth: image content,
licensing, and factual accuracy remain the agent's job (see SKILL.md,
Workflow A).

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

IMAGE_EXTS = {".jpg", ".jpeg", ".png", ".gif", ".webp"}
IMAGE_MAX_BYTES = 5 * 1024 * 1024
IMAGES_MAX = 16
NAME_MAX = 60      # UTF-16 code units, Console-aligned
EXCERPT_MAX = 300  # UTF-16 code units
SCENE_RE = re.compile(r"^\d+$")
# Shapes that must never appear in fields that get published. Matches
# credential VALUES (known prefixes, key=value assignments), never the mere
# words "token"/"password" — instructions about creating a token are legit.
SECRET_VALUE_RE = re.compile(
    r"(mgst_[A-Za-z0-9_-]{6,}|sk-[A-Za-z0-9_-]{8,}|bearer\s+[A-Za-z0-9._-]{8,}"
    r"|(?i:access[_-]?key|password|passwd|secret|api[_-]?key)\s*[:=]\s*\S{6,})"
)
LONG_HEX_RE = re.compile(r"\b[0-9a-f]{40,}\b")


def utf16_len(s):
    return len(s.encode("utf-16-le")) // 2


def main():
    argv = sys.argv[1:]
    path, scene_ids = "sensecraft-app.json", set()
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

    violations, warnings = [], []

    def vio(msg):
        violations.append(msg)

    def warn(msg):
        warnings.append(msg)

    try:
        with open(path, encoding="utf-8") as f:
            m = json.load(f)
    except (OSError, json.JSONDecodeError) as e:
        print(f"VIOLATION cannot read/parse {path}: {e}")
        return 1
    base = os.path.dirname(os.path.abspath(path))

    for key in ("manifest_version", "identity", "info", "images", "deployment", "bom", "sync"):
        if key not in m:
            vio(f"missing top-level section: {key}")

    ident = m.get("identity", {})
    rt = ident.get("remote_type")
    if rt is not None and rt not in ("Global", "China", "Private"):
        vio(f"identity.remote_type must be Global|China|Private, got {rt!r}")
    if ident.get("app_id") is not None and not str(ident["app_id"]).strip():
        vio("identity.app_id is set but empty")

    info = m.get("info", {})
    name = info.get("app_name", "")
    if not name.strip():
        vio("info.app_name is empty")
    elif utf16_len(name) > NAME_MAX:
        vio(f"info.app_name is {utf16_len(name)} UTF-16 code units (max {NAME_MAX})")
    excerpt = info.get("excerpt", "")
    if not excerpt.strip():
        vio("info.excerpt is empty")
    elif utf16_len(excerpt) > EXCERPT_MAX:
        vio(f"info.excerpt is {utf16_len(excerpt)} UTF-16 code units (max {EXCERPT_MAX})")
    scene = str(info.get("scene", ""))
    if not SCENE_RE.match(scene):
        vio(f"info.scene must be a string id from app_scenarios_array, got {scene!r}")
    elif scene_ids and scene not in scene_ids:
        vio(f"info.scene {scene} not in known ids {{{', '.join(sorted(scene_ids))}}}")
    for flag in ("allow_clone", "allow_commerce"):
        if info.get(flag) not in (0, 1):
            vio(f"info.{flag} must be 0 or 1, got {info.get(flag)!r}")

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
            p, role = img.get("path", ""), img.get("role", "")
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
                if img.get("sha256"):
                    digest = hashlib.sha256(open(full, "rb").read()).hexdigest()
                    if digest != img["sha256"]:
                        vio(f"images[{i}] {p!r}: sha256 mismatch (file changed since hash)")
                else:
                    warn(f"images[{i}] {p!r}: sha256 not computed yet")
            url = img.get("file_url")
            if url is not None and not re.match(r"^https?://", str(url)):
                vio(f"images[{i}].file_url must be http(s), got {url!r}")
            elif url is None:
                warn(f"images[{i}] {p!r}: not uploaded yet (file_url null)")

    blocks = m.get("deployment", {}).get("blocks", [])
    if not isinstance(blocks, list):
        vio("deployment.blocks must be an array")
    else:
        seen = set()
        for b in blocks:
            bid = b.get("id")
            if bid in seen:
                vio(f"deployment.blocks: duplicate id {bid!r}")
            seen.add(bid)
            if b.get("type") == "step" and not b.get("title"):
                vio(f"deployment.blocks[{bid}]: step block missing title")
            if b.get("type") not in ("step", "paragraph", "h1", "h2", "h3", "h4", "h5", "h6",
                                     "bullet", "number", "todo", "quote", "code", "divider") \
                    and "config" not in b:
                vio(f"deployment.blocks[{bid}]: tool block {b.get('type')!r} missing config")
            cfg = b.get("config")
            if isinstance(cfg, dict):
                for k, val in cfg.items():
                    if k.endswith("_url") and isinstance(val, str) and not val.strip():
                        warn(f"deployment.blocks[{bid}]: config.{k} is empty — pending value, fill before publishing")

    materials = m.get("bom", {}).get("materials", [])
    if not isinstance(materials, list):
        vio("bom.materials must be an array")
    else:
        for i, row in enumerate(materials):
            for forbidden in ("price", "pic_url", "image"):
                if forbidden in row:
                    vio(f"bom.materials[{i}]: {forbidden!r} is not a persistable field (dropped server-side)")
            if "id" in row:  # Seeed catalog row
                for req in ("name", "quantity", "note", "buy_url"):
                    if req not in row:
                        vio(f"bom.materials[{i}] (catalog row): missing {req!r}")
            else:  # custom row
                if not row.get("name") or not row.get("quantity"):
                    vio(f"bom.materials[{i}] (custom row): needs at least name + quantity")

    sync = m.get("sync", {})
    lr, lpr = sync.get("local_revision"), sync.get("last_pushed_revision")
    if not isinstance(lr, int) or lr < 1:
        vio(f"sync.local_revision must be an integer >= 1, got {lr!r}")
    if not isinstance(lpr, int) or lpr < 0 or (isinstance(lr, int) and lpr > lr):
        vio(f"sync.last_pushed_revision must be 0..local_revision, got {lpr!r}")

    prose_fields = [info.get("app_name", ""), info.get("excerpt", "")]
    prose_fields += [str(b.get("content", "")) + str(b.get("title", "")) for b in blocks if isinstance(b, dict)]
    prose_fields += [str(r.get("note", "")) for r in materials if isinstance(r, dict)]
    for text in prose_fields:
        if text and SECRET_VALUE_RE.search(text):
            vio(f"secret-shaped value in publishable text: {text[:60]!r}…")
        elif text and LONG_HEX_RE.search(text):
            warn(f"long hex literal in publishable text (key/hash?): {text[:60]!r}…")

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
