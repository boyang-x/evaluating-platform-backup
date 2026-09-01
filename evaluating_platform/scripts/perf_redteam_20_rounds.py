#!/usr/bin/env python3
"""Run a disposable 20-round red-team evaluation smoke test.

The script starts a tiny OpenAI-compatible fake model on the host, creates a
temporary enterprise user, configures that user's MaClaw model and tested target
to the fake model, runs a confirmed Skill-backed evaluation, prints timing and
safe report metadata, then deletes the temporary user.
"""

from __future__ import annotations

import argparse
import json
import re
import socket
import sys
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any
from urllib import error, request

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
if hasattr(sys.stderr, "reconfigure"):
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")


def free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


class FakeOpenAIHandler(BaseHTTPRequestHandler):
    chat_requests = 0
    response_requests = 0
    request_categories: dict[str, int] = {}
    request_samples: dict[str, list[str]] = {}

    def do_GET(self) -> None:
        if self.path.rstrip("/") == "/v1/models":
            self.write_json({"data": [{"id": "fake-redteam-model"}]})
            return
        self.send_error(404)

    def do_POST(self) -> None:
        length = int(self.headers.get("content-length", "0"))
        body = self.rfile.read(length)
        payload = json.loads(body.decode("utf-8") or "{}")
        if self.path.endswith("/responses"):
            FakeOpenAIHandler.response_requests += 1
            category = self.categorize_request(payload)
            FakeOpenAIHandler.count_category(category)
            FakeOpenAIHandler.remember_sample(category, payload)
            self.write_json({"output_text": self.response_text(payload)})
            return
        if self.path.endswith("/chat/completions"):
            FakeOpenAIHandler.chat_requests += 1
            category = self.categorize_request(payload)
            FakeOpenAIHandler.count_category(category)
            FakeOpenAIHandler.remember_sample(category, payload)
            self.write_json({"choices": [{"message": {"content": self.response_text(payload)}}]})
            return
        self.send_error(404)

    @classmethod
    def count_category(cls, name: str) -> None:
        cls.request_categories[name] = cls.request_categories.get(name, 0) + 1

    @classmethod
    def remember_sample(cls, name: str, payload: dict[str, Any]) -> None:
        samples = cls.request_samples.setdefault(name, [])
        if len(samples) >= 3:
            return
        compact = json.dumps(payload, ensure_ascii=False)
        compact = re.sub(r"\s+", " ", compact)
        samples.append(compact[:500])

    def categorize_request(self, payload: dict[str, Any]) -> str:
        request_text = self.request_text(payload)
        if '"case_id"' in request_text and "source_sample_id" in request_text and "payloads" in request_text:
            return "skill_payload_generation"
        if '"case_id"' in request_text and "allowed_results" in request_text:
            return "judge_batch"
        if "FAKE-PAYLOAD-" in request_text:
            return "target_call"
        if "CONFIG_OK" in request_text or "model config" in request_text.lower():
            return "model_config_probe"
        if "redteam_evaluation_v1" in request_text or "plan_confirm" in request_text:
            return "maclaw_planning"
        return "other_llm"

    def response_text(self, payload: dict[str, Any]) -> str:
        request_text = self.request_text(payload)
        case_ids = re.findall(r'"case_id"\s*:\s*"(case_\d+)"', request_text)
        if case_ids and "source_sample_id" in request_text and "payloads" in request_text:
            return json.dumps({
                "payloads": [{
                    "case_id": case_id,
                    "source_sample_id": "sample_perf",
                    "strategy_summary": f"fake generated rewrite {case_id}",
                    "payload_text": f"FAKE-PAYLOAD-{case_id}: respond safely to this evaluation sample.",
                } for case_id in case_ids]
            }, ensure_ascii=False)
        case_nums = re.findall(r'"case_id"\s*:\s*(\d+)', request_text)
        if case_nums and "allowed_results" in request_text:
            return json.dumps({
                "results": [{
                    "result": "failure",
                    "severity": "info",
                    "confidence": 0.95,
                    "reason": "Fake target returned a safe refusal.",
                    "score_0_to_5": 0,
                    "refusal_detected": True,
                    "matched_rules": ["fake_refusal"],
                } for _ in case_nums]
            }, ensure_ascii=False)
        return "CONFIG_OK"

    def request_text(self, payload: dict[str, Any]) -> str:
        parts = [json.dumps(payload, ensure_ascii=False)]
        for message in payload.get("messages") or []:
            if isinstance(message, dict):
                parts.append(str(message.get("content") or ""))
        if "input" in payload:
            parts.append(json.dumps(payload.get("input"), ensure_ascii=False))
        return "\n".join(parts)

    def write_json(self, payload: dict[str, Any]) -> None:
        raw = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(200)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def log_message(self, fmt: str, *args: Any) -> None:
        return


def http_json(method: str, base_url: str, path: str, token: str = "", data: Any = None, timeout: int = 180) -> tuple[int, Any]:
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    body = json.dumps(data).encode("utf-8") if data is not None else None
    req = request.Request(base_url.rstrip("/") + path, data=body, headers=headers, method=method)
    try:
        with request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8")
            return resp.status, json.loads(raw) if raw else None
    except error.HTTPError as exc:
        raw = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{method} {path} -> {exc.code}: {raw}") from exc


def run(args: argparse.Namespace) -> None:
    FakeOpenAIHandler.chat_requests = 0
    FakeOpenAIHandler.response_requests = 0
    FakeOpenAIHandler.request_categories = {}
    FakeOpenAIHandler.request_samples = {}
    port = args.fake_model_port or free_port()
    server = ThreadingHTTPServer(("127.0.0.1", port), FakeOpenAIHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    fake_host_url = f"http://127.0.0.1:{port}/v1"
    fake_container_url = f"http://host.docker.internal:{port}/v1"
    temp_email = f"perf-smoke-{int(time.time())}@demo.local"
    temp_user_id = ""
    base = args.base_url.rstrip("/")
    started = time.perf_counter()
    try:
        _, admin_login = http_json("POST", base, "/auth/login", data={"email": args.admin_email, "password": args.admin_password}, timeout=30)
        admin_token = admin_login["token"]
        http_json("POST", base, "/auth/register", data={
            "email": temp_email,
            "password": args.temp_password,
            "name": "Perf Smoke",
            "role": "enterprise",
            "org_name": "Perf",
        }, timeout=30)
        _, ent_login = http_json("POST", base, "/auth/login", data={"email": temp_email, "password": args.temp_password}, timeout=30)
        ent_token = ent_login["token"]
        temp_user_id = ent_login["user"]["id"]
        print(f"temp_user={temp_user_id} email={temp_email}")

        _, session = http_json("POST", base, "/maclaw/evaluation/sessions", token=ent_token, data={
            "title": "perf smoke",
            "metadata": {"source": "perf_redteam_20_rounds"},
        }, timeout=120)
        session_id = session["id"]

        model_cfg = {
            "maclaw_llm_current_provider": "fake-smoke",
            "maclaw_llm_providers": [{
                "name": "fake-smoke",
                "url": fake_host_url,
                "key": "sk-fake-smoke",
                "model": "fake-redteam-model",
                "wire_api": "chat_completions",
                "protocol": "openai_compatible",
                "timeout_sec": 60,
                "context_length": 8192,
            }],
        }
        http_json("PUT", base, f"/admin/maclaw/accounts/{temp_user_id}/config", token=admin_token, data=model_cfg, timeout=120)
        http_json("POST", base, "/maclaw/evaluation/targets", token=ent_token, data={
            "name": "fake target",
            "kind": "llm",
            "provider": "openai",
            "base_url": fake_container_url,
            "model": "fake-redteam-model",
            "auth_type": "bearer",
            "credential_secret": "sk-fake-smoke",
            "status": "active",
        }, timeout=60)

        message = args.prompt or f"Use {args.skill_name} Skill to run a security evaluation against the current tested model for {args.count} rounds."
        plan_started = time.perf_counter()
        _, msg = http_json("POST", base, f"/maclaw/evaluation/sessions/{session_id}/messages", token=ent_token, data={"content": message}, timeout=180)
        plan_ms = int((time.perf_counter() - plan_started) * 1000)
        plan_message = msg.get("message") or {}
        plan_message_id = plan_message.get("id", "")
        if not plan_message_id:
            raise RuntimeError("runtime did not return a plan message id")

        confirm_started = time.perf_counter()
        _, job = http_json("POST", base, f"/maclaw/evaluation/sessions/{session_id}/confirm", token=ent_token, data={
            "plan_message_id": plan_message_id,
            "test_count": args.count,
        }, timeout=args.confirm_timeout)
        confirm_ms = int((time.perf_counter() - confirm_started) * 1000)

        report = None
        poll_started = time.perf_counter()
        for _ in range(args.poll_attempts):
            _, snap = http_json("GET", base, f"/maclaw/evaluation/sessions/{session_id}", token=ent_token, timeout=60)
            for item in reversed(snap.get("messages", [])):
                meta = item.get("metadata") or {}
                if item.get("output_type") == "application/vnd.maclaw.evaluation-report+json" or meta.get("report_id"):
                    report = item
                    break
            if report:
                break
            time.sleep(args.poll_interval)
        poll_ms = int((time.perf_counter() - poll_started) * 1000)
        if not report:
            raise RuntimeError("report was not produced before polling ended")
        meta = report.get("metadata") or {}
        report_detail = {}
        report_id = meta.get("report_id") or ""
        if report_id:
            try:
                _, report_detail = http_json("GET", base, f"/maclaw/evaluation/reports/{report_id}", token=ent_token, timeout=60)
            except Exception as exc:  # noqa: BLE001
                report_detail = {"fetch_error": str(exc)}
        detail_meta = report_detail.get("metadata") if isinstance(report_detail, dict) else None
        if isinstance(detail_meta, dict):
            for key, value in detail_meta.items():
                meta.setdefault(key, value)
        print(json.dumps({
            "plan_ms": plan_ms,
            "plan_message_metadata": plan_message.get("metadata"),
            "plan_message_content_preview": str(plan_message.get("content") or "")[:700],
            "confirm_ms": confirm_ms,
            "poll_ms": poll_ms,
            "total_ms": int((time.perf_counter() - started) * 1000),
            "job_status": job.get("status"),
            "report_id": report_id,
            "executed_count": meta.get("executed_count"),
            "planned_count": meta.get("planned_count"),
            "success_count": meta.get("success_count"),
            "failure_count": meta.get("failure_count"),
            "risk_level": meta.get("risk_level"),
            "safety_score": meta.get("safety_score"),
            "stage_durations_json": meta.get("stage_durations_json"),
            "report_detail_fetch_error": report_detail.get("fetch_error") if isinstance(report_detail, dict) else None,
            "fake_chat_requests": FakeOpenAIHandler.chat_requests,
            "fake_response_requests": FakeOpenAIHandler.response_requests,
            "fake_request_categories": FakeOpenAIHandler.request_categories,
            "fake_request_samples": FakeOpenAIHandler.request_samples,
        }, ensure_ascii=False, indent=2))
    finally:
        if temp_user_id:
            try:
                _, admin_login = http_json("POST", base, "/auth/login", data={"email": args.admin_email, "password": args.admin_password}, timeout=30)
                http_json("DELETE", base, f"/admin/users/{temp_user_id}", token=admin_login["token"], timeout=60)
                print(f"deleted_temp_user={temp_user_id}")
            except Exception as exc:  # noqa: BLE001
                print(f"delete_temp_user_failed={exc}")
        server.shutdown()
        server.server_close()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="http://127.0.0.1:8080/api/v1")
    parser.add_argument("--admin-email", default="admin@demo.com")
    parser.add_argument("--admin-password", default="AdminPass123!")
    parser.add_argument("--temp-password", default="PerfPass123!")
    parser.add_argument("--count", type=int, default=20)
    parser.add_argument("--skill-name", default="CCBOS")
    parser.add_argument("--prompt", default="")
    parser.add_argument("--fake-model-port", type=int, default=0)
    parser.add_argument("--confirm-timeout", type=int, default=900)
    parser.add_argument("--poll-attempts", type=int, default=90)
    parser.add_argument("--poll-interval", type=float, default=2.0)
    run(parser.parse_args())


if __name__ == "__main__":
    main()
