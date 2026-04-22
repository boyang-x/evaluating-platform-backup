from __future__ import annotations

import json
import os
import uuid
from typing import Any, Dict, List, Tuple


SKILL_NAME = "example-generator-skill"
SKILL_VERSION = "1.0.0"


def load_json(path: str) -> Dict[str, Any]:
    with open(path, "r", encoding="utf-8") as handle:
        return json.load(handle)


def write_json(path: str, payload: Dict[str, Any]) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as handle:
        json.dump(payload, handle, ensure_ascii=False, indent=2)


def normalize_source_samples(input_payload: Dict[str, Any]) -> List[Dict[str, Any]]:
    normalized: List[Dict[str, Any]] = []
    for raw in input_payload.get("source_samples") or []:
        text = str((raw or {}).get("text") or "").strip()
        if not text:
            continue
        normalized.append(
            {
                "index": int((raw or {}).get("index") or len(normalized) + 1),
                "text": text,
            }
        )
    return normalized


def generate_payload_dataset(input_payload: Dict[str, Any], run_type: str) -> Tuple[Dict[str, Any], Dict[str, Any]]:
    source_samples = normalize_source_samples(input_payload)
    goal = str(input_payload.get("goal") or "").strip()
    requested_count = int(input_payload.get("requested_count") or 0)
    source_sample_id = str(input_payload.get("source_sample_id") or "").strip()

    if not source_samples:
        fallback_text = goal or "Generate a controlled red-team prompt."
        source_samples = [{"index": 1, "text": fallback_text}]

    if requested_count > 0:
        source_samples = source_samples[:requested_count]

    payloads: List[Dict[str, Any]] = []
    for item in source_samples:
        payload_text = f"[template] Controlled test payload derived from: {item['text']}"
        payloads.append(
            {
                "id": f"example-{item['index']}",
                "original_question": item["text"],
                "payload_text": payload_text,
                "question_summary": f"Sample #{item['index']} input.",
                "payload_summary": "Template-generated payload.",
                "language": "en",
                "sensitive": True,
            }
        )

    result = {
        "payload_dataset": {
            "dataset_id": str(uuid.uuid4()),
            "summary": f"Generated {len(payloads)} example payloads.",
            "count": len(payloads),
            "payloads": payloads,
        },
        "metadata": {
            "skill_name": SKILL_NAME,
            "skill_version": SKILL_VERSION,
            "run_type": run_type,
            "input_source_mode": "platform_resource_only",
            "source_sample_id": source_sample_id,
            "source_sample_count": len(source_samples),
        },
    }
    validation_report = {
        "skill_name": SKILL_NAME,
        "skill_version": SKILL_VERSION,
        "run_type": run_type,
        "all_passed": len(payloads) > 0,
        "sample_count": len(source_samples),
        "generated_count": len(payloads),
        "used_llm_count": 0,
        "notes": [
            "Validated payload_dataset shape.",
            "Replace the template runtime with your real generation logic.",
        ],
    }
    return result, validation_report

