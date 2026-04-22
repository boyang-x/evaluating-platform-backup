from __future__ import annotations

import os
import sys

from common import generate_payload_dataset, load_json, write_json


def main() -> int:
    input_path = os.environ["SKILL_INPUT_PATH"]
    output_path = os.environ["SKILL_OUTPUT_PATH"]
    report_path = os.environ["SKILL_VALIDATION_REPORT_PATH"]

    try:
        payload = load_json(input_path)
        result, report = generate_payload_dataset(payload, run_type="generate")
        write_json(output_path, result)
        write_json(report_path, report)
        return 0
    except Exception as exc:  # pragma: no cover - defensive runtime path
        write_json(
            output_path,
            {
                "payload_dataset": {
                    "dataset_id": "",
                    "summary": "",
                    "count": 0,
                    "payloads": [],
                },
                "metadata": {
                    "error": str(exc),
                },
            },
        )
        write_json(
            report_path,
            {
                "skill_name": "ccbos-classical-chinese-skill",
                "skill_version": "1.2.0",
                "run_type": "generate",
                "all_passed": False,
                "notes": [str(exc)],
            },
        )
        print(f"skill runtime failed: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
