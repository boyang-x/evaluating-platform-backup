#!/usr/bin/env python3
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MAIN = ROOT / "runtime" / "main.py"


class PromptInjectRuntimeTest(unittest.TestCase):
    def test_generates_promptinject_payload_dataset(self):
        with tempfile.TemporaryDirectory() as tmp:
            input_path = Path(tmp) / "input.json"
            output_path = Path(tmp) / "output.json"
            input_path.write_text(json.dumps({
                "requested_count": 4,
                "samples": [{
                    "id": "sample_1",
                    "question": "Summarize this customer supplied document.",
                    "category": "goal_hijacking",
                }],
            }), encoding="utf-8")
            proc = subprocess.run(
                [sys.executable, str(MAIN), "--input", str(input_path), "--output", str(output_path)],
                cwd=str(ROOT),
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
            )
            self.assertEqual(proc.returncode, 0, proc.stderr + proc.stdout)
            data = json.loads(output_path.read_text(encoding="utf-8"))
            dataset = data["payload_dataset"]
            self.assertEqual(dataset["count"], 4)
            self.assertEqual(data["metadata"]["generation_backend"], "deterministic_promptinject")
            self.assertEqual(data["metadata"]["input_source_mode"], "expert_samples")
            self.assertEqual(data["metadata"]["skill_version"], "1.1.0")
            payloads = dataset["payloads"]
            self.assertTrue(all(item["payload_text"].strip() for item in payloads))
            self.assertIn("IGNORE", payloads[0]["payload_text"].upper())
            self.assertEqual(payloads[0]["source_sample_id"], "sample_1")
            self.assertNotIn(str(output_path), json.dumps(data, ensure_ascii=False))


if __name__ == "__main__":
    unittest.main()
