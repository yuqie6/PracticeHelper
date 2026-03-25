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
    embedding_model: str = ""
    embedding_base_url: str = ""
    embedding_api_key: str = ""
    embedding_timeout_seconds: float = 10.0
    rerank_model: str = ""
    rerank_base_url: str = ""
    rerank_api_key: str = ""
    rerank_timeout_seconds: float = 10.0
    server_base_url: str = ""
    internal_token: str = ""
    backend_timeout_seconds: float = 10.0
    log_path: str = ""

    @property
    def llm_enabled(self) -> bool:
        # 本地 OpenAI-compatible 服务通常不要求鉴权，因此这里只强制 model/base_url。
        return bool(self.model and self.openai_base_url)

    @property
    def embedding_enabled(self) -> bool:
        return bool(self.embedding_model and self.embedding_base_url)

    @property
    def rerank_enabled(self) -> bool:
        return bool(self.rerank_model and self.rerank_base_url)

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
        embedding_model=os.getenv("PRACTICEHELPER_SIDECAR_EMBEDDING_MODEL", "").strip(),
        embedding_base_url=os.getenv("PRACTICEHELPER_SIDECAR_EMBEDDING_BASE_URL", "").strip(),
        embedding_api_key=os.getenv("PRACTICEHELPER_SIDECAR_EMBEDDING_API_KEY", "").strip(),
        embedding_timeout_seconds=float(
            os.getenv("PRACTICEHELPER_SIDECAR_EMBEDDING_TIMEOUT_SECONDS", "10").strip()
        ),
        rerank_model=os.getenv("PRACTICEHELPER_SIDECAR_RERANK_MODEL", "").strip(),
        rerank_base_url=os.getenv("PRACTICEHELPER_SIDECAR_RERANK_BASE_URL", "").strip(),
        rerank_api_key=os.getenv("PRACTICEHELPER_SIDECAR_RERANK_API_KEY", "").strip(),
        rerank_timeout_seconds=float(
            os.getenv("PRACTICEHELPER_SIDECAR_RERANK_TIMEOUT_SECONDS", "10").strip()
        ),
        server_base_url=os.getenv("PRACTICEHELPER_SIDECAR_SERVER_BASE_URL", "").strip(),
        internal_token=os.getenv("PRACTICEHELPER_INTERNAL_TOKEN", "").strip(),
        backend_timeout_seconds=float(
            os.getenv("PRACTICEHELPER_SIDECAR_BACKEND_TIMEOUT_SECONDS", "10").strip()
        ),
        log_path=os.getenv("PRACTICEHELPER_SIDECAR_LOG_PATH", "../data/logs/sidecar.log").strip(),
    )
