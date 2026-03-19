from __future__ import annotations

import logging
import secrets
import time
from pathlib import Path

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from app.config import load_settings
from app.langgraph_flows import build_flows
from app.llm_client import ModelClientError
from app.schemas import (
    AnalyzeRepoRequest,
    AnalyzeRepoResponse,
    EvaluateAnswerRequest,
    EvaluationResult,
    GenerateQuestionRequest,
    GenerateQuestionResponse,
    GenerateReviewRequest,
    ReviewCard,
)

settings = load_settings()
logger = logging.getLogger(__name__)


def configure_logging() -> None:
    handlers: list[logging.Handler] = [logging.StreamHandler()]

    if settings.log_path:
        log_path = Path(settings.log_path)
        log_path.parent.mkdir(parents=True, exist_ok=True)
        handlers.append(logging.FileHandler(log_path, encoding="utf-8"))

    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
        handlers=handlers,
        force=True,
    )


configure_logging()
flows = build_flows(settings)

app = FastAPI(title="PracticeHelper Sidecar", version="0.1.0")


@app.middleware("http")
async def log_requests(request: Request, call_next):
    request_id = request.headers.get("X-Request-ID", f"req_{secrets.token_hex(8)}")
    started_at = time.perf_counter()

    try:
        response = await call_next(request)
    except Exception:
        logger.exception(
            "sidecar request failed",
            extra={
                "request_id": request_id,
                "method": request.method,
                "path": request.url.path,
            },
        )
        raise

    duration_ms = round((time.perf_counter() - started_at) * 1000, 2)
    response.headers["X-Request-ID"] = request_id
    logger.info(
        "sidecar request completed",
        extra={
            "request_id": request_id,
            "method": request.method,
            "path": request.url.path,
            "status_code": response.status_code,
            "duration_ms": duration_ms,
        },
    )
    return response


@app.exception_handler(ModelClientError)
def handle_model_client_error(_: Request, exc: ModelClientError) -> JSONResponse:
    message = str(exc)
    status_code = 503 if "LLM is required" in message else 502
    return JSONResponse(status_code=status_code, content={"error": {"message": message}})


@app.get("/healthz")
def healthcheck() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/internal/analyze_repo", response_model=AnalyzeRepoResponse)
def analyze_repo_endpoint(request: AnalyzeRepoRequest) -> AnalyzeRepoResponse:
    return flows["analyze_repo"].invoke({"request": request})["result"]


@app.post("/internal/generate_question", response_model=GenerateQuestionResponse)
def generate_question_endpoint(request: GenerateQuestionRequest) -> GenerateQuestionResponse:
    return flows["generate_question"].invoke({"request": request})["result"]


@app.post("/internal/evaluate_answer", response_model=EvaluationResult)
def evaluate_answer_endpoint(request: EvaluateAnswerRequest) -> EvaluationResult:
    return flows["evaluate_answer"].invoke({"request": request})["result"]


@app.post("/internal/generate_review", response_model=ReviewCard)
def generate_review_endpoint(request: GenerateReviewRequest) -> ReviewCard:
    return flows["generate_review"].invoke({"request": request})["result"]
