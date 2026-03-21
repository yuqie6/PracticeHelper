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
    server_base_url: str = ""
    internal_token: str = ""
    backend_timeout_seconds: float = 10.0
    log_path: str = ""

    @property
    def llm_enabled(self) -> bool:
        # 三个配置缺一不可；半配状态统一按关闭处理，避免请求跑到不确定的降级分支。
        return bool(self.model and self.openai_base_url and self.openai_api_key)

    @property
    def backend_enabled(self) -> bool:
        return bool(self.server_base_url and self.internal_token)


def load_settings() -> Settings:
    return Settings(
        github_token=os.getenv("PRACTICEHELPER_SIDECAR_GITHUB_TOKEN", "").strip(),
        model=os.getenv("PRACTICEHELPER_SIDECAR_MODEL", "").strip(),
        openai_base_url=os.getenv("PRACTICEHELPER_SIDECAR_OPENAI_BASE_URL", "").strip(),
        openai_api_key=os.getenv("PRACTICEHELPER_SIDECAR_OPENAI_API_KEY", "").strip(),
        llm_timeout_seconds=float(
            os.getenv("PRACTICEHELPER_SIDECAR_LLM_TIMEOUT_SECONDS", "45").strip()
        ),
        server_base_url=os.getenv("PRACTICEHELPER_SIDECAR_SERVER_BASE_URL", "").strip(),
        internal_token=os.getenv("PRACTICEHELPER_INTERNAL_TOKEN", "").strip(),
        backend_timeout_seconds=float(
            os.getenv("PRACTICEHELPER_SIDECAR_BACKEND_TIMEOUT_SECONDS", "10").strip()
        ),
        log_path=os.getenv("PRACTICEHELPER_SIDECAR_LOG_PATH", "../data/logs/sidecar.log").strip(),
    )
