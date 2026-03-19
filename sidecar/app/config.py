from __future__ import annotations

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    github_token: str
    model: str
    openai_base_url: str
    openai_api_key: str
    llm_timeout_seconds: float

    @property
    def llm_enabled(self) -> bool:
        return bool(self.model and self.openai_base_url and self.openai_api_key)


def load_settings() -> Settings:
    return Settings(
        github_token=os.getenv("PRACTICEHELPER_SIDECAR_GITHUB_TOKEN", "").strip(),
        model=os.getenv("PRACTICEHELPER_SIDECAR_MODEL", "").strip(),
        openai_base_url=os.getenv("PRACTICEHELPER_SIDECAR_OPENAI_BASE_URL", "").strip(),
        openai_api_key=os.getenv("PRACTICEHELPER_SIDECAR_OPENAI_API_KEY", "").strip(),
        llm_timeout_seconds=float(
            os.getenv("PRACTICEHELPER_SIDECAR_LLM_TIMEOUT_SECONDS", "45").strip()
        ),
    )
