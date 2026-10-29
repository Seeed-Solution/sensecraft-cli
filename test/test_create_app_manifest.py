"""Regression checks for the create-app skill's offline manifest validator.

Run: python3 -B -m unittest discover -s test -p 'test_create_app_manifest.py'
"""
import copy
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[1]
SKILL = ROOT / "skills/sensecraft-cli/create-app"
VALIDATOR = SKILL / "assets/validate_manifest.py"


class ManifestValidationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="codecraft-manifest-test-")
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        self.manifest = json.loads((SKILL / "assets/manifest-template.json").read_text())
        self.manifest["info"].update(
            app_name="Timer", excerpt="Time laps on the display.", scene="5",
            allow_clone=1, allow_commerce=0,
        )
        self.manifest["deployment"]["blocks"] = [{
            "id": "flash", "type": "flash-firmware", "config": {
                "device_id": "102991299", "device_type": "", "flash_mode": "bin", "flash_address": "0x4000",
                "firmware_url": "https://files.example.invalid/timer.bin",
                "firmware_name": "timer.bin", "firmware_size": "1024",
                "muse_gadget": False, "full_flash": False,
            },
        }]

    def validate(self, manifest=None, extra=None):
        path = self.directory / "codecraft-app.json"
        path.write_text(json.dumps(self.manifest if manifest is None else manifest))
        return subprocess.run(
            [sys.executable, "-B", str(VALIDATOR), *(extra if extra is not None else ["--scene-ids", "5,7"])],
            cwd=self.directory, capture_output=True, text=True,
        )

    def assert_invalid(self, manifest):
        result = self.validate(manifest)
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        self.assertNotIn("Traceback", result.stderr)

    def test_default_name_and_renderer_readable_firmware(self):
        result = self.validate()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_blank_template_requires_authoring_decisions(self):
        template = json.loads((SKILL / "assets/manifest-template.json").read_text())
        self.assert_invalid(template)

    def test_root_and_sections_fail_cleanly(self):
        for value in ([], "manifest", 7, None):
            with self.subTest(root=value):
                # None has special meaning in validate(), so write it directly.
                if value is None:
                    (self.directory / "codecraft-app.json").write_text("null")
                    result = subprocess.run([sys.executable, "-B", str(VALIDATOR)], cwd=self.directory, capture_output=True, text=True)
                    self.assertEqual(result.returncode, 1)
                    self.assertNotIn("Traceback", result.stderr)
                else:
                    self.assert_invalid(value)
        for key in ("identity", "info", "images", "deployment", "bom", "sync"):
            for value in (None, 42, "wrong", [] if key != "images" else {}):
                with self.subTest(section=key, value=value):
                    manifest = copy.deepcopy(self.manifest)
                    manifest[key] = value
                    self.assert_invalid(manifest)

    def test_canonical_types_and_version(self):
        cases = [("manifest_version", None, 2), ("manifest_version", None, True),
                 ("info", "scene", 5), ("info", "app_name", None),
                 ("info", "excerpt", []), ("info", "allow_clone", True),
                 ("info", "allow_commerce", 1.0), ("sync", "local_revision", True)]
        for section, key, value in cases:
            with self.subTest(section=section, key=key, value=value):
                manifest = copy.deepcopy(self.manifest)
                if key is None:
                    manifest[section] = value
                else:
                    manifest[section][key] = value
                self.assert_invalid(manifest)

    def test_scene_option_argument_cannot_silently_disable_check(self):
        for args in (["--scene-ids="], ["--scene-ids"], ["--scene-ids", "nope"]):
            with self.subTest(args=args):
                self.assertEqual(self.validate(extra=args).returncode, 2)
        self.manifest["info"]["scene"] = "99"
        self.assert_invalid(self.manifest)

    def test_optional_activity_selection_uses_positive_integer_or_none(self):
        for value in (None, 1, 2**63 - 1):
            with self.subTest(valid_activity=value):
                self.manifest["info"]["activity_id"] = value
                result = self.validate()
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.manifest["info"].pop("activity_id")
        self.assertEqual(self.validate().returncode, 0)
        for value in (0, -1, True, False, 1.0, "1", "", [], {}, 2**63):
            with self.subTest(invalid_activity=value):
                self.manifest["info"]["activity_id"] = value
                self.assert_invalid(self.manifest)

    def test_nonfinite_numbers_are_not_json_payloads(self):
        self.manifest["deployment"]["blocks"] = [{"id":"model", "type":"deploy-ai-model", "config":{"threshold":float("nan")}}]
        self.assert_invalid(self.manifest)

    def test_bom_types_and_custom_rows(self):
        self.manifest["bom"]["materials"] = [{"name": "Printed case", "quantity": "1"}]
        self.assertEqual(self.validate().returncode, 0)
        for row in (None, "part", {"id": None, "name": "Board", "quantity": "1", "note": "", "buy_url": ""},
                    {"name": "Case", "quantity": 1}, {"name": "Case", "quantity": "0"},
                    {"name": "Case", "quantity": "1", "price": "10"},
                    {"name": "Board", "quantity": "1", "sku": "102991299"}):
            with self.subTest(row=row):
                self.manifest["bom"]["materials"] = [row]
                self.assert_invalid(self.manifest)

    def test_malformed_blocks_fail_cleanly(self):
        for block in (None, [], {"id": "a", "type": {}},
                      {"id": [], "type": "paragraph", "content": "text"},
                      {"id": "a", "type": "unknown", "config": {}},
                      {"id": "a", "type": "deploy-ai-model", "config": []}):
            with self.subTest(block=block):
                self.manifest["deployment"]["blocks"] = [block]
                self.assert_invalid(self.manifest)

    def test_nested_firmware_allowed_flat_or_mixed_firmware_rejected(self):
        block = self.manifest["deployment"]["blocks"][0]
        block["config"]["firmware_url"] = ""
        self.assertEqual(self.validate().returncode, 0)
        block["firmware_url"] = "https://files.example.invalid/flat.bin"
        self.assert_invalid(self.manifest)
        block.pop("config")
        self.assert_invalid(self.manifest)

    def test_all_tools_use_one_config_object(self):
        for kind in ("flash-firmware", "deploy-ai-model", "ai-vibration", "deploy-container-app", "node-red", "aily-blockly"):
            with self.subTest(kind=kind):
                block = {"id":"tool", "type":kind, "config":{}}
                self.manifest["deployment"]["blocks"] = [block]
                self.assertEqual(self.validate().returncode, 0)
                block["config"] = {"config": {"firmware_url":"https://files.example.invalid/image.bin"}}
                self.assert_invalid(self.manifest)

    def test_node_red_flow_arrays_and_aily_extension_string(self):
        block = {"id":"flow", "type":"node-red", "config":{"flow_data":[{"id":"node-1"}], "model_data":None}}
        self.manifest["deployment"]["blocks"] = [block]
        self.assertEqual(self.validate().returncode, 0)
        block["config"]["flow_data"] = "[]"
        self.assert_invalid(self.manifest)
        block.update(type="aily-blockly", config={"project_id":"123", "config":""})
        self.assertEqual(self.validate().returncode, 0)

    def test_firmware_flags_are_booleans_and_muse_disables_full_flash(self):
        cfg = self.manifest["deployment"]["blocks"][0]["config"]
        cfg.update(device_id="113991315",device_type="37",flash_address="0x0")
        for key in ("muse_gadget", "full_flash"):
            for value in ("false", "true", 0, 1):
                with self.subTest(key=key,value=value):
                    cfg[key] = value
                    self.assert_invalid(self.manifest)
            cfg[key] = False
        cfg.update(muse_gadget=True,full_flash=True)
        self.assert_invalid(self.manifest)
        cfg["full_flash"] = False
        self.assertEqual(self.validate().returncode, 0)

    def test_firmware_modes_sizes_and_addresses(self):
        block = self.manifest["deployment"]["blocks"][0]["config"]
        block.update(device_id="113991115",device_type="32",flash_address="0x0")
        block.update(flash_mode="zip", project_zip_url="https://files.example.invalid/project.zip", project_zip_size="2048")
        self.assertEqual(self.validate().returncode, 0)
        for key, value in (("flash_mode", 0), ("flash_address", []), ("flash_address", "0x20000"),
                           ("device_id", 0), ("device_id", False), ("device_id", None),
                           ("firmware_size", 1024), ("project_zip_size", "large"),
                           ("project_zip_url", "file:///tmp/project.zip"), ("project_zip_url", "https://")):
            with self.subTest(key=key, value=value):
                manifest = copy.deepcopy(self.manifest)
                manifest["deployment"]["blocks"][0]["config"][key] = value
                self.assert_invalid(manifest)

    def test_markdown_is_compiled_but_source_code_is_preserved(self):
        block = {"id": "p", "type": "paragraph", "content": "**bold**"}
        self.manifest["deployment"]["blocks"] = [block]
        self.assert_invalid(self.manifest)
        block.update(type="code", content="**pointer;\n# include <stdio.h>")
        self.assertEqual(self.validate().returncode, 0)

    def test_credentials_in_prose_and_config_are_redacted(self):
        sentinel = "mgst_SYNTHETIC_CREDENTIAL_FOR_TEST"
        for key in ("excerpt", "config"):
            with self.subTest(location=key):
                manifest = copy.deepcopy(self.manifest)
                if key == "excerpt":
                    manifest["info"]["excerpt"] = sentinel
                else:
                    manifest["deployment"]["blocks"] = [{"id":"n", "type":"node-red", "config":{"flow_data":{"api_key":sentinel}}}]
                result = self.validate(manifest)
                self.assertEqual(result.returncode, 1)
                self.assertNotIn(sentinel, result.stdout + result.stderr)

    def test_image_hashes_urls_and_entry_shapes(self):
        image = self.directory / "cover.png"
        image.write_bytes(b"image fixture bytes")
        digest = hashlib.sha256(image.read_bytes()).hexdigest()
        self.manifest["images"] = [{"path":"cover.png", "role":"cover", "sha256":digest, "file_url":None}]
        self.assertEqual(self.validate().returncode, 0)
        for value in (None, 7, {"path":None}, {"path":"cover.png", "role":"cover", "sha256":"short"},
                      {"path":"cover.png", "role":"cover", "sha256":digest, "file_url":"https://"}):
            with self.subTest(image=value):
                self.manifest["images"] = [value]
                self.assert_invalid(self.manifest)

    def test_pushed_state_requires_complete_sync_metadata(self):
        self.manifest["sync"]["last_pushed_revision"] = 1
        self.assert_invalid(self.manifest)
        self.manifest["identity"]["app_id"] = "12345"
        self.manifest["sync"].update(last_pushed_hash="sha256:"+"0"*64, last_pushed_at="2026-10-28T08:00:00Z")
        self.assertEqual(self.validate().returncode, 0)
        self.manifest["sync"]["last_pushed_at"] = "2026-10-28T08:00:00"
        self.assert_invalid(self.manifest)

    def test_reference_example_has_valid_shape_and_reproducible_hash(self):
        reference = (SKILL / "references/manifest.md").read_text()
        example = json.loads(re.findall(r"```json\n(.*?)\n```", reference, re.S)[-1])
        self.assertEqual(self.validate(example).returncode, 0)
        projection = {"info":{key:value for key,value in example["info"].items() if key != "activity_id"}, "images":[i["file_url"] for i in example["images"]],
                      "deployment":example["deployment"], "materials":example["bom"]["materials"]}
        canonical = json.dumps(projection,sort_keys=True,separators=(",",":"),ensure_ascii=False).encode()
        self.assertEqual(example["sync"]["last_pushed_hash"], "sha256:"+hashlib.sha256(canonical).hexdigest())


if __name__ == "__main__":
    unittest.main()
