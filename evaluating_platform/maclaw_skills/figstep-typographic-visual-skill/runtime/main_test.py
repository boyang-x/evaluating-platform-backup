#!/usr/bin/env python3
import base64
import importlib.util
import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MAIN = ROOT / "runtime" / "main.py"


def write_embedded_data(runtime_dir):
    entries = []
    for path in sorted((ROOT / "data").rglob("*")):
        if not path.is_file():
            continue
        rel = "data/" + path.relative_to(ROOT / "data").as_posix()
        encoded = base64.b64encode(path.read_bytes()).decode("ascii")
        entries.append(f"    {rel!r}: {encoded!r},")
    runtime_dir.mkdir(parents=True, exist_ok=True)
    (runtime_dir / "embedded_data.py").write_text(
        "# test embedded data fixture\nEMBEDDED_FILES = {\n" + "\n".join(entries) + "\n}\n",
        encoding="utf-8",
    )


class FigStepSkillTest(unittest.TestCase):
    def test_outputs_multimodal_payload_dataset(self):
        with tempfile.TemporaryDirectory() as tmp:
            output = Path(tmp) / "output.json"
            proc = subprocess.run(
                [sys.executable, str(MAIN), "--input", str(ROOT / "examples" / "input.json"), "--output", str(output)],
                cwd=str(ROOT),
                text=True,
                capture_output=True,
                check=True,
            )
            data = json.loads(output.read_text(encoding="utf-8"))
            stdout_data = json.loads(proc.stdout)
        dataset = data["payload_dataset"]
        self.assertEqual(dataset["count"], 3)
        self.assertEqual(stdout_data["payload_dataset"]["count"], 3)
        first = dataset["payloads"][0]
        self.assertTrue(first["payload_text"])
        self.assertEqual(first["metadata"]["judge_profile"], "figstep_typographic_jailbreak")
        self.assertEqual(first["metadata"]["multimodal_attack_family"], "figstep")
        self.assertEqual(first["images"][0]["mime_type"], "image/png")
        self.assertGreater(len(first["images"][0]["image_base64"]), 100)

    def test_missing_case_data_fails_fast(self):
        spec = importlib.util.spec_from_file_location("figstep_main", MAIN)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        with tempfile.TemporaryDirectory() as tmp:
            module.ROOT = Path(tmp)
            with self.assertRaisesRegex(ValueError, "data/cases.json"):
                module.build_payload_dataset({"requested_count": 1})

    def test_runtime_data_layout_is_supported(self):
        spec = importlib.util.spec_from_file_location("figstep_main_runtime_data", MAIN)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / "skill"
            runtime = root / "runtime"
            shutil.copytree(ROOT / "data", runtime / "data")
            module.ROOT = root
            module.__file__ = str(runtime / "main.py")
            dataset = module.build_payload_dataset({"requested_count": 1})
        self.assertEqual(dataset["count"], 1)
        self.assertEqual(dataset["payloads"][0]["metadata"]["multimodal_attack_family"], "figstep")
        self.assertEqual(dataset["payloads"][0]["images"][0]["mime_type"], "image/png")

    def test_embedded_data_layout_is_supported(self):
        spec = importlib.util.spec_from_file_location("figstep_main_embedded_data", MAIN)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / "skill"
            runtime = root / "runtime"
            write_embedded_data(runtime)
            module.ROOT = root
            module.__file__ = str(runtime / "main.py")
            module._EMBEDDED_FILES = None
            dataset = module.build_payload_dataset({"requested_count": 1})
        self.assertEqual(dataset["count"], 1)
        self.assertEqual(dataset["payloads"][0]["metadata"]["multimodal_attack_family"], "figstep")
        self.assertEqual(dataset["payloads"][0]["images"][0]["mime_type"], "image/png")


if __name__ == "__main__":
    unittest.main()
