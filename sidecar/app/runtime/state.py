from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from pydantic import BaseModel

from app.schemas import RuntimeTrace


@dataclass(frozen=True)
class TaskExecutionResult[ResultModelT: BaseModel]:
    result: ResultModelT
    raw_output: str = ""
    side_effects: dict[str, Any] = field(default_factory=dict)
    command_results: list[dict[str, Any]] = field(default_factory=list)
    trace: RuntimeTrace = field(default_factory=RuntimeTrace)


@dataclass
class ToolRuntimeState:
    side_effects: dict[str, Any] = field(default_factory=dict)
    command_results: list[dict[str, Any]] = field(default_factory=list)
    command_cache: dict[str, dict[str, Any]] = field(default_factory=dict)
    command_counts: dict[str, int] = field(default_factory=dict)
    command_budget: dict[str, int] = field(
        default_factory=lambda: {
            "transition_session": 1,
            "upsert_review_path": 1,
        }
    )
