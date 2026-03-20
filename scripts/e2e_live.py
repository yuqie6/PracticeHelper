#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any


DEFAULT_SCENARIO_PATH = Path(__file__).with_name("e2e_live.sample.json")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run a live PracticeHelper end-to-end smoke test.")
    parser.add_argument("--base-url", default="http://127.0.0.1:8090", help="Go API base URL")
    parser.add_argument(
        "--repo-url",
        default="",
        help="Repository URL used for project import; defaults to scenario.project.repo_url",
    )
    parser.add_argument("--import-timeout-seconds", type=float, default=180.0)
    parser.add_argument("--poll-interval-seconds", type=float, default=2.0)
    parser.add_argument(
        "--output",
        default="",
        help="Optional output path for the final JSON summary",
    )
    parser.add_argument(
        "--scenario",
        default=str(DEFAULT_SCENARIO_PATH),
        help="Replayable scenario JSON path",
    )
    return parser.parse_args()


def load_scenario(path: str) -> dict[str, Any]:
    with open(path, "r", encoding="utf-8") as handle:
        payload = json.load(handle)

    if not isinstance(payload, dict):
        raise RuntimeError(f"scenario must be a JSON object: {path}")

    for key in ("profile", "basics", "project"):
        if key not in payload or not isinstance(payload[key], dict):
            raise RuntimeError(f"scenario missing object field: {key}")

    return payload


def request_json(
    base_url: str,
    method: str,
    path: str,
    payload: dict[str, Any] | None = None,
    timeout: float = 60.0,
) -> tuple[int, dict[str, Any]]:
    data = None if payload is None else json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(
        urllib.parse.urljoin(base_url, path),
        data=data,
        headers={"Content-Type": "application/json"},
        method=method,
    )

    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            return response.status, json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {exc.code} {path}: {body}") from exc


def get_data(base_url: str, path: str, timeout: float = 60.0) -> Any:
    _, payload = request_json(base_url, "GET", path, timeout=timeout)
    return payload["data"]


def post_data(base_url: str, path: str, payload: dict[str, Any], timeout: float = 60.0) -> tuple[int, Any]:
    status, body = request_json(base_url, "POST", path, payload=payload, timeout=timeout)
    return status, body["data"]


def stream_json(base_url: str, path: str, payload: dict[str, Any], timeout: float = 300.0) -> tuple[Any, list[dict[str, Any]]]:
    request = urllib.request.Request(
        urllib.parse.urljoin(base_url, path),
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    events: list[dict[str, Any]] = []
    result = None
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            for raw in response:
                line = raw.decode("utf-8", errors="replace").strip()
                if not line:
                    continue
                event = json.loads(line)
                events.append(event)
                if event.get("type") == "error":
                    raise RuntimeError(f"stream error {path}: {event.get('message')}")
                if event.get("type") == "result":
                    result = event.get("data")
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {exc.code} {path}: {body}") from exc

    if result is None:
        raise RuntimeError(f"stream {path} did not return a result event")
    return result, events


def summarize_stream(events: list[dict[str, Any]]) -> dict[str, Any]:
    counts: dict[str, int] = {}
    reasoning: list[str] = []
    for event in events:
        event_type = str(event.get("type", "unknown"))
        counts[event_type] = counts.get(event_type, 0) + 1
        if event_type == "reasoning" and event.get("text"):
            reasoning.append(str(event["text"]))
    return {
        "counts": counts,
        "reasoning_samples": reasoning[:3],
    }


def find_project_by_repo_url(projects: list[dict[str, Any]], repo_url: str) -> dict[str, Any] | None:
    for project in projects:
        if project.get("repo_url") == repo_url:
            return project
    return None


def ensure_import_job_support(base_url: str) -> None:
    try:
        get_data(base_url, "/api/import-jobs", timeout=10.0)
    except RuntimeError as exc:
        if "HTTP 404" in str(exc):
            raise RuntimeError(
                "当前 API 还没有 /api/import-jobs，通常说明你还没重启到最新 server 进程。"
            ) from exc
        raise


def import_project(base_url: str, repo_url: str, timeout_seconds: float, poll_interval_seconds: float) -> dict[str, Any]:
    projects = get_data(base_url, "/api/projects", timeout=30.0)
    existing = find_project_by_repo_url(projects, repo_url)
    if existing is not None:
        return {
            "job": None,
            "project": existing,
            "mode": "reused_existing_project",
        }

    status, job = post_data(
        base_url,
        "/api/projects/import",
        {"repo_url": repo_url},
        timeout=30.0,
    )
    if status != 202:
        raise RuntimeError(f"expected 202 from /api/projects/import, got {status}")

    started = time.time()
    while True:
        current = get_data(base_url, f"/api/import-jobs/{job['id']}", timeout=30.0)
        if current["status"] == "completed":
            if not current.get("project_id"):
                raise RuntimeError("import job completed without project_id")
            project = get_data(base_url, f"/api/projects/{current['project_id']}", timeout=30.0)
            return {
                "job": current,
                "project": project,
                "mode": "imported_via_job",
                "duration_seconds": round(time.time() - started, 2),
            }
        if current["status"] == "failed":
            raise RuntimeError(
                f"import job failed stage={current.get('stage')} message={current.get('message')} error={current.get('error_message')}"
            )
        if time.time() - started > timeout_seconds:
            raise RuntimeError(
                f"import job timed out after {timeout_seconds}s stage={current.get('stage')} message={current.get('message')}"
            )
        time.sleep(poll_interval_seconds)


def main() -> int:
    args = parse_args()
    scenario = load_scenario(args.scenario)
    base_url = str(scenario.get("base_url") or args.base_url).rstrip("/")
    repo_url = args.repo_url if args.repo_url else str(scenario["project"].get("repo_url", "")).strip()
    if not repo_url:
        raise RuntimeError("repo_url is required either in --repo-url or scenario.project.repo_url")

    ensure_import_job_support(base_url)

    profile_payload = scenario["profile"]
    _, profile = post_data(
        base_url,
        "/api/profile",
        profile_payload,
        timeout=30.0,
    )

    dashboard_before = get_data(base_url, "/api/dashboard", timeout=30.0)
    imported = import_project(
        base_url,
        repo_url,
        timeout_seconds=args.import_timeout_seconds,
        poll_interval_seconds=args.poll_interval_seconds,
    )
    project = imported["project"]

    basics_payload = scenario["basics"]
    basics_session, basics_create_events = stream_json(
        base_url,
        "/api/sessions/stream",
        {
            "mode": "basics",
            "topic": basics_payload["topic"],
            "intensity": basics_payload["intensity"],
        },
        timeout=240.0,
    )
    basics_main, basics_main_events = stream_json(
        base_url,
        f"/api/sessions/{urllib.parse.quote(basics_session['id'])}/answer/stream",
        {"answer": basics_payload["main_answer"]},
        timeout=240.0,
    )
    basics_final, basics_followup_events = stream_json(
        base_url,
        f"/api/sessions/{urllib.parse.quote(basics_session['id'])}/answer/stream",
        {"answer": basics_payload["followup_answer"]},
        timeout=240.0,
    )
    basics_review = get_data(base_url, f"/api/reviews/{urllib.parse.quote(basics_final['review_id'])}", timeout=30.0)

    project_payload = scenario["project"]
    project_session, project_create_events = stream_json(
        base_url,
        "/api/sessions/stream",
        {
            "mode": "project",
            "project_id": project["id"],
            "intensity": project_payload["intensity"],
        },
        timeout=240.0,
    )
    project_main, project_main_events = stream_json(
        base_url,
        f"/api/sessions/{urllib.parse.quote(project_session['id'])}/answer/stream",
        {"answer": project_payload["main_answer"]},
        timeout=240.0,
    )
    project_final, project_followup_events = stream_json(
        base_url,
        f"/api/sessions/{urllib.parse.quote(project_session['id'])}/answer/stream",
        {"answer": project_payload["followup_answer"]},
        timeout=240.0,
    )
    project_review = get_data(base_url, f"/api/reviews/{urllib.parse.quote(project_final['review_id'])}", timeout=30.0)

    dashboard_after = get_data(base_url, "/api/dashboard", timeout=30.0)

    summary = {
        "base_url": base_url,
        "scenario_path": str(Path(args.scenario).resolve()),
        "profile_id": profile["id"],
        "dashboard_days_until_deadline": dashboard_before.get("days_until_deadline"),
        "import": imported,
        "current_session_after_runs": dashboard_after.get("current_session"),
        "basics": {
            "session_id": basics_session["id"],
            "final_status": basics_final["status"],
            "review_id": basics_final["review_id"],
            "review_overall": basics_review["overall"],
            "events": {
                "create": summarize_stream(basics_create_events),
                "main": summarize_stream(basics_main_events),
                "followup": summarize_stream(basics_followup_events),
            },
        },
        "project": {
            "project_id": project["id"],
            "project_name": project["name"],
            "session_id": project_session["id"],
            "final_status": project_final["status"],
            "review_id": project_final["review_id"],
            "review_overall": project_review["overall"],
            "events": {
                "create": summarize_stream(project_create_events),
                "main": summarize_stream(project_main_events),
                "followup": summarize_stream(project_followup_events),
            },
        },
    }

    output = json.dumps(summary, ensure_ascii=False, indent=2)
    print(output)
    if args.output:
        with open(args.output, "w", encoding="utf-8") as handle:
            handle.write(output)
            handle.write("\n")

    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # pragma: no cover - smoke script failure path
        print(f"[e2e-live] {exc}", file=sys.stderr)
        raise SystemExit(1)
