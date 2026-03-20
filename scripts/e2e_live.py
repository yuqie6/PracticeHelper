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
    if "job_target" in payload and not isinstance(payload["job_target"], dict):
        raise RuntimeError("scenario.job_target must be a JSON object when provided")

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


def patch_data(base_url: str, path: str, payload: dict[str, Any], timeout: float = 60.0) -> tuple[int, Any]:
    status, body = request_json(base_url, "PATCH", path, payload=payload, timeout=timeout)
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
    status_names: list[str] = []
    phases: list[str] = []
    for event in events:
        event_type = str(event.get("type", "unknown"))
        counts[event_type] = counts.get(event_type, 0) + 1
        if event_type == "reasoning" and event.get("text"):
            reasoning.append(str(event["text"]))
        if event_type == "status" and event.get("name"):
            status_names.append(str(event["name"]))
        if event_type == "phase" and event.get("phase"):
            phases.append(str(event["phase"]))
    return {
        "counts": counts,
        "status_names": status_names,
        "phases": phases,
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


def expect(condition: bool, message: str) -> None:
    if not condition:
        raise RuntimeError(message)


def expect_context_name(events: list[dict[str, Any]], name: str, label: str) -> None:
    names = [str(event.get("name", "")) for event in events if event.get("type") == "context"]
    expect(name in names, f"{label} missing context event {name}; got {names}")


def as_string_list(value: Any) -> list[str]:
    if not isinstance(value, list):
        return []
    return [str(item) for item in value]


def project_to_input(project: dict[str, Any]) -> dict[str, Any]:
    return {
        "name": str(project.get("name", "")).strip(),
        "summary": str(project.get("summary", "")).strip(),
        "tech_stack": as_string_list(project.get("tech_stack")),
        "highlights": as_string_list(project.get("highlights")),
        "challenges": as_string_list(project.get("challenges")),
        "tradeoffs": as_string_list(project.get("tradeoffs")),
        "ownership_points": as_string_list(project.get("ownership_points")),
        "followup_points": as_string_list(project.get("followup_points")),
    }


def build_project_edit_payload(project: dict[str, Any]) -> tuple[dict[str, Any], str]:
    payload = project_to_input(project)
    marker = "phase6 smoke: 项目画像已验证可编辑并保存"
    payload["name"] = f"{payload['name']} [phase6-smoke]"
    payload["summary"] = f"{payload['summary']}（phase6 smoke 已验证）".strip()
    followup_points = list(payload["followup_points"])
    if marker not in followup_points:
        followup_points.append(marker)
    payload["followup_points"] = followup_points
    return payload, marker


def verify_project_edit_cycle(base_url: str, project: dict[str, Any]) -> tuple[dict[str, Any], dict[str, Any]]:
    original_payload = project_to_input(project)
    edited_payload, marker = build_project_edit_payload(project)
    project_id = str(project["id"])
    edited_applied = False

    try:
        _, updated = patch_data(
            base_url,
            f"/api/projects/{urllib.parse.quote(project_id)}",
            edited_payload,
            timeout=30.0,
        )
        edited_applied = True
        refreshed = get_data(base_url, f"/api/projects/{urllib.parse.quote(project_id)}", timeout=30.0)

        expect(updated["name"] == edited_payload["name"], "project edit response did not persist edited name")
        expect(
            refreshed["summary"] == edited_payload["summary"],
            "project edit verification failed: summary was not persisted",
        )
        expect(
            marker in as_string_list(refreshed.get("followup_points")),
            "project edit verification failed: followup point marker missing after save",
        )
    finally:
        if edited_applied:
            patch_data(
                base_url,
                f"/api/projects/{urllib.parse.quote(project_id)}",
                original_payload,
                timeout=30.0,
            )

    restored = get_data(base_url, f"/api/projects/{urllib.parse.quote(project_id)}", timeout=30.0)
    expect(restored["name"] == original_payload["name"], "project restore failed: name mismatch")
    expect(restored["summary"] == original_payload["summary"], "project restore failed: summary mismatch")
    expect(
        as_string_list(restored.get("followup_points")) == original_payload["followup_points"],
        "project restore failed: followup points mismatch",
    )

    return restored, {
        "project_id": project_id,
        "edited_name": edited_payload["name"],
        "edited_summary": edited_payload["summary"],
        "marker_followup_point": marker,
        "restored": True,
    }


def create_and_prepare_job_target(base_url: str, payload: dict[str, Any]) -> dict[str, Any]:
    status, created = post_data(
        base_url,
        "/api/job-targets",
        {
            "title": str(payload.get("title", "")).strip(),
            "company_name": str(payload.get("company_name", "")).strip(),
            "source_text": str(payload.get("source_text", "")).strip(),
        },
        timeout=30.0,
    )
    expect(status == 201, f"expected 201 from /api/job-targets, got {status}")
    target_id = str(created.get("id", "")).strip()
    expect(target_id != "", "job target creation returned empty id")

    analyze_status, analysis = post_data(
        base_url,
        f"/api/job-targets/{urllib.parse.quote(target_id)}/analyze",
        {},
        timeout=240.0,
    )
    expect(analyze_status == 201, f"expected 201 from job target analyze, got {analyze_status}")
    expect(analysis.get("status") == "succeeded", f"job target analysis did not succeed: {analysis}")

    ready_target = get_data(base_url, f"/api/job-targets/{urllib.parse.quote(target_id)}", timeout=30.0)
    expect(
        ready_target.get("latest_analysis_status") == "succeeded",
        f"job target did not reach succeeded state: {ready_target.get('latest_analysis_status')}",
    )
    latest_successful = ready_target.get("latest_successful_analysis")
    expect(isinstance(latest_successful, dict), "job target missing latest_successful_analysis after success")
    expect(
        str(latest_successful.get("id", "")).strip() == str(analysis.get("id", "")).strip(),
        "job target latest_successful_analysis id does not match analyze response",
    )

    activate_status, profile = post_data(
        base_url,
        f"/api/job-targets/{urllib.parse.quote(target_id)}/activate",
        {},
        timeout=30.0,
    )
    expect(activate_status == 200, f"expected 200 from activate job target, got {activate_status}")
    expect(
        str(profile.get("active_job_target_id", "")).strip() == target_id,
        "profile active_job_target_id did not update after activation",
    )

    return {
        "target": ready_target,
        "analysis": analysis,
        "profile": profile,
    }


def verify_bound_job_target(
    session: dict[str, Any],
    review: dict[str, Any],
    prepared_job_target: dict[str, Any],
    label: str,
) -> None:
    target = prepared_job_target["target"]
    analysis = prepared_job_target["analysis"]
    target_id = str(target["id"])
    analysis_id = str(analysis["id"])

    expect(str(session.get("job_target_id", "")).strip() == target_id, f"{label} session lost job_target_id")
    expect(
        str(session.get("job_target_analysis_id", "")).strip() == analysis_id,
        f"{label} session lost job_target_analysis_id",
    )
    expect(str(review.get("job_target_id", "")).strip() == target_id, f"{label} review lost job_target_id")
    expect(
        str(review.get("job_target_analysis_id", "")).strip() == analysis_id,
        f"{label} review lost job_target_analysis_id",
    )

    session_job_target = session.get("job_target")
    expect(isinstance(session_job_target, dict), f"{label} session missing job_target ref")
    expect(
        str(session_job_target.get("title", "")).strip() == str(target.get("title", "")).strip(),
        f"{label} session job_target title mismatch",
    )
    review_job_target = review.get("job_target")
    expect(isinstance(review_job_target, dict), f"{label} review missing job_target ref")
    expect(
        str(review_job_target.get("title", "")).strip() == str(target.get("title", "")).strip(),
        f"{label} review job_target title mismatch",
    )


def mark_job_target_stale(base_url: str, target: dict[str, Any], stale_source_text: str) -> dict[str, Any]:
    _, updated = patch_data(
        base_url,
        f"/api/job-targets/{urllib.parse.quote(str(target['id']))}",
        {
            "title": str(target.get("title", "")).strip(),
            "company_name": str(target.get("company_name", "")).strip(),
            "source_text": stale_source_text.strip(),
        },
        timeout=30.0,
    )
    expect(updated.get("latest_analysis_status") == "stale", "job target did not become stale after source update")
    latest_successful = updated.get("latest_successful_analysis")
    expect(
        isinstance(latest_successful, dict) and str(latest_successful.get("id", "")).strip() != "",
        "stale job target should still expose latest successful analysis",
    )
    return updated


def expect_create_session_conflict(base_url: str, payload: dict[str, Any], expected_code: str) -> dict[str, Any]:
    request = urllib.request.Request(
        urllib.parse.urljoin(base_url, "/api/sessions"),
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(request, timeout=60.0) as response:
            raise RuntimeError(
                f"expected /api/sessions to fail with {expected_code}, got HTTP {response.status}"
            )
    except urllib.error.HTTPError as exc:
        body = json.loads(exc.read().decode("utf-8"))
        error = body.get("error") or {}
        expect(exc.code == 409, f"expected HTTP 409, got {exc.code}")
        expect(
            str(error.get("code", "")).strip() == expected_code,
            f"expected error code {expected_code}, got {error.get('code')}",
        )
        return body


def extract_status_names(events: list[dict[str, Any]]) -> list[str]:
    return [str(event["name"]) for event in events if event.get("type") == "status" and event.get("name")]


def expect_status_subsequence(events: list[dict[str, Any]], expected: list[str], label: str) -> list[str]:
    got = extract_status_names(events)
    cursor = 0
    for name in got:
        if cursor < len(expected) and name == expected[cursor]:
            cursor += 1

    expect(
        cursor == len(expected),
        f"{label} status sequence mismatch: got {got}, expected subsequence {expected}",
    )
    return got


def verify_dashboard_binding(
    dashboard: dict[str, Any],
    expected_session_ids: list[str],
) -> dict[str, Any]:
    weaknesses = dashboard.get("weaknesses")
    expect(isinstance(weaknesses, list) and len(weaknesses) > 0, "dashboard weaknesses is empty after live run")

    top_weakness = weaknesses[0]
    label = str(top_weakness.get("label", "")).strip()
    recommended_track = str(dashboard.get("recommended_track", "")).strip()
    today_focus = str(dashboard.get("today_focus", "")).strip()
    recent_sessions = dashboard.get("recent_sessions")
    expect(label != "", "dashboard top weakness label is empty")
    expect(recommended_track != "", "dashboard recommended_track is empty")
    expect(today_focus != "", "dashboard today_focus is empty")
    expect(label.lower() in recommended_track.lower(), "recommended_track is not bound to top weakness label")
    expect(label.lower() in today_focus.lower(), "today_focus is not bound to top weakness label")
    expect(isinstance(recent_sessions, list), "dashboard recent_sessions is not a list")

    recent_session_ids = {str(item.get("id", "")) for item in recent_sessions}
    for session_id in expected_session_ids:
        expect(session_id in recent_session_ids, f"dashboard recent_sessions missing session {session_id}")

    return {
        "top_weakness": top_weakness,
        "recommended_track": recommended_track,
        "today_focus": today_focus,
        "recent_session_ids": sorted(recent_session_ids),
    }


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
    prepared_job_target = None
    dashboard_with_ready_job_target = None
    job_target_payload = scenario.get("job_target")
    if isinstance(job_target_payload, dict):
        prepared_job_target = create_and_prepare_job_target(base_url, job_target_payload)
        dashboard_with_ready_job_target = get_data(base_url, "/api/dashboard", timeout=30.0)
        active_job_target = dashboard_with_ready_job_target.get("active_job_target")
        expect(isinstance(active_job_target, dict), "dashboard missing active_job_target after job target activation")
        expect(
            str(active_job_target.get("id", "")).strip()
            == str(prepared_job_target["target"].get("id", "")).strip(),
            "dashboard active_job_target id mismatch after job target activation",
        )
        expect(
            dashboard_with_ready_job_target.get("recommendation_scope") == "job_target",
            "dashboard recommendation_scope did not switch to job_target after successful JD activation",
        )

    imported = import_project(
        base_url,
        repo_url,
        timeout_seconds=args.import_timeout_seconds,
        poll_interval_seconds=args.poll_interval_seconds,
    )
    project, project_edit_verification = verify_project_edit_cycle(base_url, imported["project"])

    basics_payload = scenario["basics"]
    basics_create_payload = {
        "mode": "basics",
        "topic": basics_payload["topic"],
        "intensity": basics_payload["intensity"],
    }
    if prepared_job_target is not None:
        basics_create_payload["job_target_id"] = prepared_job_target["target"]["id"]
    basics_session, basics_create_events = stream_json(
        base_url,
        "/api/sessions/stream",
        basics_create_payload,
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
    basics_main_statuses = expect_status_subsequence(
        basics_main_events,
        ["answer_received", "evaluation_started", "feedback_ready", "answer_saved", "followup_ready"],
        "basics main answer",
    )
    basics_followup_statuses = expect_status_subsequence(
        basics_followup_events,
        ["answer_received", "evaluation_started", "feedback_ready", "answer_saved", "review_started", "review_saved"],
        "basics followup answer",
    )
    expect(basics_final["status"] == "completed", f"basics session did not complete: {basics_final['status']}")
    expect(str(basics_final.get("review_id", "")).strip() != "", "basics session missing review_id")
    if prepared_job_target is not None:
        expect_context_name(basics_create_events, "read_job_target_analysis", "basics create")
        verify_bound_job_target(basics_final, basics_review, prepared_job_target, "basics")

    project_payload = scenario["project"]
    project_create_payload = {
        "mode": "project",
        "project_id": project["id"],
        "intensity": project_payload["intensity"],
    }
    if prepared_job_target is not None:
        project_create_payload["job_target_id"] = prepared_job_target["target"]["id"]
    project_session, project_create_events = stream_json(
        base_url,
        "/api/sessions/stream",
        project_create_payload,
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
    project_main_statuses = expect_status_subsequence(
        project_main_events,
        ["answer_received", "evaluation_started", "feedback_ready", "answer_saved", "followup_ready"],
        "project main answer",
    )
    project_followup_statuses = expect_status_subsequence(
        project_followup_events,
        ["answer_received", "evaluation_started", "feedback_ready", "answer_saved", "review_started", "review_saved"],
        "project followup answer",
    )
    expect(project_final["status"] == "completed", f"project session did not complete: {project_final['status']}")
    expect(str(project_final.get("review_id", "")).strip() != "", "project session missing review_id")
    if prepared_job_target is not None:
        expect_context_name(project_create_events, "read_job_target_analysis", "project create")
        verify_bound_job_target(project_final, project_review, prepared_job_target, "project")

    dashboard_after = get_data(base_url, "/api/dashboard", timeout=30.0)
    dashboard_binding = verify_dashboard_binding(
        dashboard_after,
        [str(basics_final["id"]), str(project_final["id"])],
    )

    stale_job_target = None
    stale_block_error = None
    dashboard_after_stale = None
    if prepared_job_target is not None:
        stale_source_text = str(job_target_payload.get("stale_source_text", "")).strip()
        if stale_source_text:
            stale_job_target = mark_job_target_stale(
                base_url,
                prepared_job_target["target"],
                stale_source_text,
            )
            stale_block_error = expect_create_session_conflict(
                base_url,
                {
                    "mode": "basics",
                    "topic": basics_payload["topic"],
                    "job_target_id": prepared_job_target["target"]["id"],
                    "intensity": basics_payload["intensity"],
                },
                "job_target_not_ready",
            )
            dashboard_after_stale = get_data(base_url, "/api/dashboard", timeout=30.0)
            expect(
                dashboard_after_stale.get("recommendation_scope") == "generic",
                "dashboard recommendation_scope should fall back to generic after JD becomes stale",
            )

    summary = {
        "base_url": base_url,
        "scenario_path": str(Path(args.scenario).resolve()),
        "profile_id": profile["id"],
        "dashboard_days_until_deadline": dashboard_before.get("days_until_deadline"),
        "import": imported,
        "project_edit_verification": project_edit_verification,
        "dashboard_with_ready_job_target": dashboard_with_ready_job_target,
        "current_session_after_runs": dashboard_after.get("current_session"),
        "dashboard_binding": dashboard_binding,
        "basics": {
            "session_id": basics_session["id"],
            "final_status": basics_final["status"],
            "review_id": basics_final["review_id"],
            "review_overall": basics_review["overall"],
            "job_target_id": basics_final.get("job_target_id", ""),
            "job_target_analysis_id": basics_final.get("job_target_analysis_id", ""),
            "status_sequence": {
                "main": basics_main_statuses,
                "followup": basics_followup_statuses,
            },
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
            "job_target_id": project_final.get("job_target_id", ""),
            "job_target_analysis_id": project_final.get("job_target_analysis_id", ""),
            "status_sequence": {
                "main": project_main_statuses,
                "followup": project_followup_statuses,
            },
            "events": {
                "create": summarize_stream(project_create_events),
                "main": summarize_stream(project_main_events),
                "followup": summarize_stream(project_followup_events),
            },
        },
    }
    if prepared_job_target is not None:
        summary["job_target"] = {
            "target_id": prepared_job_target["target"]["id"],
            "analysis_id": prepared_job_target["analysis"]["id"],
            "title": prepared_job_target["target"]["title"],
            "latest_status_after_prepare": prepared_job_target["target"]["latest_analysis_status"],
            "stale_status_after_update": stale_job_target.get("latest_analysis_status") if stale_job_target else "",
            "stale_block_error": stale_block_error.get("error") if stale_block_error else None,
            "dashboard_after_stale": dashboard_after_stale,
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
