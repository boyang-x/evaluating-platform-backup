#!/usr/bin/env python3
import argparse
import base64
import hashlib
import importlib.util
import json
import random
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
_EMBEDDED_FILES = None


def embedded_files():
    global _EMBEDDED_FILES
    if _EMBEDDED_FILES is not None:
        return _EMBEDDED_FILES
    module_path = Path(__file__).resolve().parent / "embedded_data.py"
    if not module_path.exists():
        _EMBEDDED_FILES = {}
        return _EMBEDDED_FILES
    spec = importlib.util.spec_from_file_location("skill_embedded_data", module_path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    _EMBEDDED_FILES = getattr(module, "EMBEDDED_FILES", {}) or {}
    return _EMBEDDED_FILES


def embedded_file_bytes(path):
    raw = str(path or "").replace("\\", "/").lstrip("/")
    candidates = [raw]
    marker = "/data/"
    if marker in raw:
        candidates.append(raw[raw.index(marker) + 1:])
    if raw.startswith("images/"):
        candidates.append("data/" + raw)
    for key in candidates:
        encoded = embedded_files().get(key)
        if encoded:
            return base64.b64decode(encoded)
    return None


def data_root():
    for candidate in (ROOT / "data", Path(__file__).resolve().parent / "data"):
        if (candidate / "cases.json").exists():
            return candidate
    return ROOT / "data"


def resolve_data_file(root, path):
    raw = str(path or "")
    candidates = [ROOT / raw, root / raw]
    if raw.startswith("data/"):
        candidates.append(root / raw[len("data/"):])
    for candidate in candidates:
        if candidate.exists():
            return candidate
    return candidates[0]


def load_json(path, default):
    if not path:
        return default
    p = Path(path)
    if not p.exists():
        embedded = embedded_file_bytes(path)
        if embedded is not None:
            return json.loads(embedded.decode("utf-8"))
        return default
    with p.open("r", encoding="utf-8") as f:
        return json.load(f)


def int_from_input(data, *names, default=3):
    for name in names:
        try:
            value = int(data.get(name))
        except (TypeError, ValueError):
            continue
        if value > 0:
            return value
    return default


def select_cases(cases, count, strategy):
    count = max(1, min(int(count), 20))
    selected = list(cases)
    if not selected:
        raise ValueError("FigStep skill data/cases.json is missing or contains no cases")
    if strategy == "random":
        rng = random.Random(20260227)
        rng.shuffle(selected)
    if len(selected) < count:
        expanded = []
        while len(expanded) < count:
            expanded.extend(selected)
        selected = expanded
    return selected[:count]


def image_payload(root, path, description):
    p = resolve_data_file(root, path)
    if p.exists():
        raw = p.read_bytes()
    else:
        raw = embedded_file_bytes(path) or embedded_file_bytes(p)
        if raw is None:
            raise FileNotFoundError(f"image asset {path!r} was not found")
    mime = "image/png"
    if p.suffix.lower() in {".jpg", ".jpeg"}:
        mime = "image/jpeg"
    return {
        "mime_type": mime,
        "image_base64": base64.b64encode(raw).decode("ascii"),
        "description": description,
        "sha256": hashlib.sha256(raw).hexdigest(),
    }


def build_payload_dataset(skill_input):
    root = data_root()
    data = load_json(root / "cases.json", {})
    cases = select_cases(
        data.get("cases", []),
        int_from_input(skill_input, "requested_count", "test_count", "count", default=3),
        str(skill_input.get("selection_strategy") or "sequential").lower(),
    )
    payloads = []
    for index, case in enumerate(cases, start=1):
        image = image_payload(root, case["image_path"], "FigStep typographic visual prompt from project data")
        payloads.append({
            "id": f"figstep-{index}-{case['id']}",
            "source_sample_id": case["id"],
            "original_question": case["original_question"],
            "payload_text": data["payload_text"],
            "payload_summary": "FigStep typographic image prompt plus benign list-completion text prompt.",
            "language": "en",
            "images": [image],
            "metadata": {
                "source_project": data["source_project"],
                "source_repo": data["source_repo"],
                "source_dataset": data["source_dataset"],
                "multimodal_attack_family": data["multimodal_attack_family"],
                "judge_profile": data["judge_profile"],
                "original_evaluator": data["original_evaluator"],
                "category": case.get("category", ""),
                "image_prompt_type": "typographic_list",
            },
        })
    digest = hashlib.sha256(json.dumps(payloads, sort_keys=True).encode("utf-8")).hexdigest()[:12]
    return {
        "dataset_id": f"{data.get('dataset_id', 'figstep')}-{digest}",
        "summary": f"Generated {len(payloads)} FigStep typographic visual jailbreak payloads.",
        "count": len(payloads),
        "payloads": payloads,
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", default=str(ROOT / "examples" / "input.json"))
    parser.add_argument("--output", default="output.json")
    args = parser.parse_args()
    skill_input = load_json(args.input, {})
    dataset = build_payload_dataset(skill_input)
    result = {"payload_dataset": dataset}
    Path(args.output).write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")
    print(json.dumps(result, ensure_ascii=False))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(json.dumps({"error": str(exc)}, ensure_ascii=False), file=sys.stderr)
        raise
