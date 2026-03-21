from __future__ import annotations

from collections.abc import Mapping
from functools import cache
from pathlib import Path

PROMPTS_DIR = Path(__file__).with_name("prompts")


@cache
def load_prompt(name: str) -> str:
    return (PROMPTS_DIR / name).read_text(encoding="utf-8").strip()


def render_prompt(name: str, replacements: Mapping[str, str] | None = None) -> str:
    content = load_prompt(name)
    for key, value in (replacements or {}).items():
        content = content.replace(f"__{key}__", value)
    return content
