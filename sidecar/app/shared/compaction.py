from __future__ import annotations

import json
from typing import Any

from app.repo_analysis.context import RepoAnalysisBundle
from app.schemas import RepoChunk


def compact_chunks(
    chunks: list[RepoChunk], *, limit: int = 6, max_chars: int = 420
) -> list[dict[str, Any]]:
    compact: list[dict[str, Any]] = []
    for chunk in chunks[:limit]:
        compact.append(
            {
                "file_path": chunk.file_path,
                "file_type": chunk.file_type,
                "importance": chunk.importance,
                "fts_key": chunk.fts_key,
                "content": chunk.content[:max_chars],
            }
        )
    return compact


def trim_text(value: str, max_chars: int) -> str:
    if max_chars <= 0:
        return ""
    stripped = value.strip()
    if len(stripped) <= max_chars:
        return stripped
    return stripped[: max_chars - 1].rstrip() + "…"


def compact_string_list(items: list[str], *, limit: int, max_chars: int | None = None) -> list[str]:
    compacted: list[str] = []
    for item in items[:limit]:
        value = item.strip()
        if not value:
            continue
        if max_chars is not None:
            value = trim_text(value, max_chars)
        compacted.append(value)
    return compacted


def estimate_payload_chars(value: Any) -> int:
    try:
        return len(json.dumps(value, ensure_ascii=False, separators=(",", ":")))
    except TypeError:
        return len(str(value))


def build_compaction_details(
    *,
    section: str,
    before_count: int,
    after_count: int,
    before_value: Any,
    after_value: Any,
    budget: str,
) -> dict[str, Any]:
    return {
        "section": section,
        "before_count": before_count,
        "after_count": after_count,
        "before_chars": estimate_payload_chars(before_value),
        "after_chars": estimate_payload_chars(after_value),
        "budget": budget,
    }


def repo_overview_payload(bundle: RepoAnalysisBundle) -> dict[str, Any]:
    return {
        "repo_url": bundle.repo_url,
        "name": bundle.name,
        "default_branch": bundle.default_branch,
        "import_commit": bundle.import_commit,
        "tech_stack": bundle.tech_stack,
        "top_paths": bundle.top_paths[:8],
        "chunk_count": len(bundle.chunks),
    }
