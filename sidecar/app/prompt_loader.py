from __future__ import annotations

import hashlib
import json
from collections.abc import Mapping
from dataclasses import dataclass
from functools import cache
from pathlib import Path

PROMPTS_DIR = Path(__file__).with_name("prompts")
PROMPT_SETS_DIR = PROMPTS_DIR / "sets"
PROMPT_REGISTRY_PATH = PROMPTS_DIR / "registry.json"


@dataclass(frozen=True)
class PromptSet:
    id: str
    label: str
    description: str
    status: str
    is_default: bool = False


@dataclass(frozen=True)
class PromptLoadResult:
    prompt_set_id: str
    content: str
    prompt_hash: str


@cache
def list_prompt_sets() -> tuple[PromptSet, ...]:
    payload = json.loads(PROMPT_REGISTRY_PATH.read_text(encoding="utf-8"))
    items = payload.get("prompt_sets", [])
    prompt_sets = tuple(
        PromptSet(
            id=item["id"],
            label=item["label"],
            description=item.get("description", ""),
            status=item["status"],
            is_default=bool(item.get("is_default")),
        )
        for item in items
    )

    if not prompt_sets:
        raise ValueError("prompt registry is empty")

    default_count = sum(1 for item in prompt_sets if item.is_default)
    if default_count != 1:
        raise ValueError("prompt registry must contain exactly one default prompt set")

    for prompt_set in prompt_sets:
        set_dir = PROMPT_SETS_DIR / prompt_set.id
        if not set_dir.is_dir():
            raise ValueError(f"prompt set directory does not exist: {prompt_set.id}")

    return prompt_sets


def get_default_prompt_set() -> PromptSet:
    return next(item for item in list_prompt_sets() if item.is_default)


def get_prompt_set(prompt_set_id: str = "") -> PromptSet:
    if not prompt_set_id:
        return get_default_prompt_set()

    for item in list_prompt_sets():
        if item.id == prompt_set_id:
            return item

    raise ValueError(f"unknown prompt set: {prompt_set_id}")


@cache
def load_prompt(name: str, prompt_set_id: str = "") -> str:
    return load_prompt_with_meta(name, prompt_set_id).content


@cache
def load_prompt_with_meta(name: str, prompt_set_id: str = "") -> PromptLoadResult:
    prompt_set = get_prompt_set(prompt_set_id)
    content = (PROMPT_SETS_DIR / prompt_set.id / name).read_text(encoding="utf-8").strip()
    return PromptLoadResult(
        prompt_set_id=prompt_set.id,
        content=content,
        prompt_hash=hashlib.sha256(content.encode("utf-8")).hexdigest(),
    )


def render_prompt(
    name: str,
    replacements: Mapping[str, str] | None = None,
    prompt_set_id: str = "",
) -> str:
    return render_prompt_with_meta(name, replacements, prompt_set_id).content


def render_prompt_with_meta(
    name: str,
    replacements: Mapping[str, str] | None = None,
    prompt_set_id: str = "",
) -> PromptLoadResult:
    loaded = load_prompt_with_meta(name, prompt_set_id)
    content = loaded.content
    for key, value in (replacements or {}).items():
        content = content.replace(f"__{key}__", value)
    return PromptLoadResult(
        prompt_set_id=loaded.prompt_set_id,
        content=content,
        prompt_hash=hashlib.sha256(content.encode("utf-8")).hexdigest(),
    )
