#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run a live PracticeHelper end-to-end smoke test.")
    parser.add_argument("--base-url", default="http://127.0.0.1:8090", help="Go API base URL")
    parser.add_argument(
        "--repo-url",
        default="https://github.com/octocat/Hello-World",
        help="Repository URL used for project import",
    )
    parser.add_argument("--import-timeout-seconds", type=float, default=180.0)
    parser.add_argument("--poll-interval-seconds", type=float, default=2.0)
    parser.add_argument(
        "--output",
        default="",
        help="Optional output path for the final JSON summary",
    )
    return parser.parse_args()


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
    base_url = args.base_url.rstrip("/")

    ensure_import_job_support(base_url)

    _, profile = post_data(
        base_url,
        "/api/profile",
        {
            "target_role": "Go 后端工程师",
            "target_company_type": "互联网公司",
            "current_stage": "在职看机会",
            "application_deadline": "2026-04-10T00:00:00Z",
            "tech_stacks": ["Go", "Redis", "Kafka"],
            "primary_projects": ["PracticeHelper"],
            "self_reported_weaknesses": ["项目表达"],
        },
        timeout=30.0,
    )

    dashboard_before = get_data(base_url, "/api/dashboard", timeout=30.0)
    imported = import_project(
        base_url,
        args.repo_url,
        timeout_seconds=args.import_timeout_seconds,
        poll_interval_seconds=args.poll_interval_seconds,
    )
    project = imported["project"]

    basics_session, basics_create_events = stream_json(
        base_url,
        "/api/sessions/stream",
        {"mode": "basics", "topic": "redis", "intensity": "standard"},
        timeout=240.0,
    )
    basics_main, basics_main_events = stream_json(
        base_url,
        f"/api/sessions/{urllib.parse.quote(basics_session['id'])}/answer/stream",
        {
            "answer": "Redis 快主要因为数据在内存、单线程事件循环减少切换和锁成本，并结合高效数据结构与 I/O 多路复用提升吞吐。线上还要注意大 key、慢命令和持久化带来的抖动。"
        },
        timeout=240.0,
    )
    basics_final, basics_followup_events = stream_json(
        base_url,
        f"/api/sessions/{urllib.parse.quote(basics_session['id'])}/answer/stream",
        {
            "answer": "I/O 多路复用让 Redis 用单线程就能同时监听大量连接，把就绪事件交给事件循环处理。这样不用为每个连接开线程，但也要求单次命令足够快，否则整个事件循环都会被阻塞。"
        },
        timeout=240.0,
    )
    basics_review = get_data(base_url, f"/api/reviews/{urllib.parse.quote(basics_final['review_id'])}", timeout=30.0)

    project_session, project_create_events = stream_json(
        base_url,
        "/api/sessions/stream",
        {"mode": "project", "project_id": project["id"], "intensity": "standard"},
        timeout=240.0,
    )
    project_main, project_main_events = stream_json(
        base_url,
        f"/api/sessions/{urllib.parse.quote(project_session['id'])}/answer/stream",
        {
            "answer": "这个仓库的核心价值不在业务复杂度，而在如何把配置加载这件小事做成可复用库。面试时我会从 API 设计、默认行为、环境变量覆盖关系和侵入性控制来讲它的取舍。"
        },
        timeout=240.0,
    )
    project_final, project_followup_events = stream_json(
        base_url,
        f"/api/sessions/{urllib.parse.quote(project_session['id'])}/answer/stream",
        {
            "answer": "如果继续扩展，我会优先补边界测试、错误语义和多配置源优先级说明，再决定是否增加更强的 schema 校验。这样能先稳住库的可信度，而不是过早堆功能。"
        },
        timeout=240.0,
    )
    project_review = get_data(base_url, f"/api/reviews/{urllib.parse.quote(project_final['review_id'])}", timeout=30.0)

    dashboard_after = get_data(base_url, "/api/dashboard", timeout=30.0)

    summary = {
        "base_url": base_url,
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
