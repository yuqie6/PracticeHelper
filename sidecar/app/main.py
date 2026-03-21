from __future__ import annotations

import logging
import secrets
import time
from pathlib import Path

from fastapi import FastAPI, Request, Response
from fastapi.responses import JSONResponse, StreamingResponse

from app.agent_runtime import AgentRuntime
from app.config import load_settings
from app.langgraph_flows import build_flows
from app.llm_client import ModelClientError
from app.prompt_loader import list_prompt_sets
from app.runtime_prompts import (
    evaluate_prompt_meta,
    question_prompt_meta,
    review_prompt_meta,
)
from app.schemas import (
    AnalyzeJobTargetRequest,
    AnalyzeJobTargetResponse,
    AnalyzeRepoRequest,
    AnalyzeRepoResponse,
    EvaluateAnswerRequest,
    EvaluationResult,
    GenerateQuestionRequest,
    GenerateQuestionResponse,
    GenerateReviewRequest,
    PromptSetSummary,
    ReviewCard,
)

settings = load_settings()
logger = logging.getLogger(__name__)
runtime = AgentRuntime(settings)


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

PROMPT_SET_HEADER = "X-PracticeHelper-Prompt-Set"
PROMPT_HASH_HEADER = "X-PracticeHelper-Prompt-Hash"
MODEL_NAME_HEADER = "X-PracticeHelper-Model-Name"


@app.middleware("http")
async def log_requests(request: Request, call_next):
    request_id = request.headers.get("X-Request-ID", f"req_{secrets.token_hex(8)}")
    started_at = time.perf_counter()

    try:
        response = await call_next(request)
    except Exception:
        logger.exception(
            "sidecar request failed request_id=%s method=%s path=%s",
            request_id,
            request.method,
            request.url.path,
        )
        raise

    duration_ms = round((time.perf_counter() - started_at) * 1000, 2)
    response.headers["X-Request-ID"] = request_id
    logger.info(
        "sidecar request completed request_id=%s method=%s path=%s status=%s duration_ms=%.2f",
        request_id,
        request.method,
        request.url.path,
        response.status_code,
        duration_ms,
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


@app.get("/internal/prompt-sets", response_model=list[PromptSetSummary])
def list_prompt_sets_endpoint() -> list[PromptSetSummary]:
    return [
        PromptSetSummary(
            id=item.id,
            label=item.label,
            description=item.description,
            status=item.status,
            is_default=item.is_default,
        )
        for item in list_prompt_sets()
    ]


@app.post("/internal/analyze_repo", response_model=AnalyzeRepoResponse)
def analyze_repo_endpoint(request: AnalyzeRepoRequest) -> AnalyzeRepoResponse:
    return flows["analyze_repo"].invoke({"request": request})["result"]


@app.post("/internal/analyze_job_target", response_model=AnalyzeJobTargetResponse)
def analyze_job_target_endpoint(request: AnalyzeJobTargetRequest) -> AnalyzeJobTargetResponse:
    return flows["analyze_job_target"].invoke({"request": request})["result"]


@app.post("/internal/generate_question", response_model=GenerateQuestionResponse)
def generate_question_endpoint(
    request: GenerateQuestionRequest, response: Response
) -> GenerateQuestionResponse:
    apply_prompt_headers(response, question_prompt_meta(request))
    return flows["generate_question"].invoke({"request": request})["result"]


@app.post("/internal/generate_question/stream")
def generate_question_stream_endpoint(request: GenerateQuestionRequest) -> StreamingResponse:
    response = StreamingResponse(
        _ndjson_stream(runtime.stream_generate_question(request)),
        media_type="application/x-ndjson",
    )
    apply_prompt_headers(response, question_prompt_meta(request))
    return response


@app.post("/internal/evaluate_answer", response_model=EvaluationResult)
def evaluate_answer_endpoint(
    request: EvaluateAnswerRequest, response: Response
) -> EvaluationResult:
    apply_prompt_headers(response, evaluate_prompt_meta(request))
    return flows["evaluate_answer"].invoke({"request": request})["result"]


@app.post("/internal/evaluate_answer/stream")
def evaluate_answer_stream_endpoint(request: EvaluateAnswerRequest) -> StreamingResponse:
    response = StreamingResponse(
        _ndjson_stream(runtime.stream_evaluate_answer(request)),
        media_type="application/x-ndjson",
    )
    apply_prompt_headers(response, evaluate_prompt_meta(request))
    return response


@app.post("/internal/generate_review", response_model=ReviewCard)
def generate_review_endpoint(
    request: GenerateReviewRequest, response: Response
) -> ReviewCard:
    apply_prompt_headers(response, review_prompt_meta(request))
    return flows["generate_review"].invoke({"request": request})["result"]


@app.post("/internal/generate_review/stream")
def generate_review_stream_endpoint(request: GenerateReviewRequest) -> StreamingResponse:
    response = StreamingResponse(
        _ndjson_stream(runtime.stream_generate_review(request)),
        media_type="application/x-ndjson",
    )
    apply_prompt_headers(response, review_prompt_meta(request))
    return response


def _ndjson_stream(events):
    try:
        for event in events:
            yield JSONResponse(content=event).body + b"\n"
    except Exception as exc:
        yield JSONResponse(content={"type": "error", "message": str(exc)}).body + b"\n"


def apply_prompt_headers(response: Response, prompt_meta) -> None:
    response.headers[PROMPT_SET_HEADER] = prompt_meta.prompt_set_id
    response.headers[PROMPT_HASH_HEADER] = prompt_meta.prompt_hash
    response.headers[MODEL_NAME_HEADER] = settings.model
