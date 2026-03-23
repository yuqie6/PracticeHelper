from __future__ import annotations

import logging
import secrets
import time
from pathlib import Path

from fastapi import FastAPI, Request, Response
from fastapi.responses import JSONResponse, StreamingResponse

from app.adapters.llm_client import ModelClientError, OpenAICompatibleModelClient
from app.config import load_settings
from app.flows.langgraph import build_flows
from app.prompts.bundles import (
    evaluate_prompt_meta,
    question_prompt_meta,
    review_prompt_meta,
)
from app.prompts.loader import list_prompt_sets
from app.runtime import AgentRuntime
from app.schemas import (
    AnalyzeJobTargetEnvelope,
    AnalyzeJobTargetRequest,
    AnalyzeRepoEnvelope,
    AnalyzeRepoRequest,
    EmbeddedMemoryVector,
    EmbedMemoryRequest,
    EmbedMemoryResponse,
    EvaluateAnswerEnvelope,
    EvaluateAnswerRequest,
    GenerateQuestionEnvelope,
    GenerateQuestionRequest,
    GenerateReviewEnvelope,
    GenerateReviewRequest,
    PromptSetSummary,
    RerankMemoryRequest,
    RerankMemoryResponse,
    RerankMemoryResult,
)

settings = load_settings()
logger = logging.getLogger(__name__)
# 这些对象跟着进程生命周期一起初始化：
# 配置有问题时可以在启动期尽早暴露，而不是等到某个请求打进来才失败。
model_client = OpenAICompatibleModelClient(settings)
runtime = AgentRuntime(settings, model_client=model_client if settings.llm_enabled else None)


def configure_logging() -> None:
    handlers: list[logging.Handler] = [logging.StreamHandler()]

    if settings.log_path:
        # sidecar 经常被 uvicorn、make dev 或 supervisor 接管 stdout，
        # 但排查长任务或线上问题时仍需要可持久化的本地日志文件。
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


# 和 Go 服务共用 `X-Request-ID`，这样一条训练请求跨服务的日志能串起来看。
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
    return JSONResponse(
        status_code=status_code,
        content={"error": {"code": exc.code or classify_error_code(exc), "message": message}},
    )


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


@app.post("/internal/analyze_repo", response_model=AnalyzeRepoEnvelope)
def analyze_repo_endpoint(request: AnalyzeRepoRequest) -> AnalyzeRepoEnvelope:
    return flows["analyze_repo"].invoke({"request": request})["result"]


@app.post("/internal/analyze_job_target", response_model=AnalyzeJobTargetEnvelope)
def analyze_job_target_endpoint(request: AnalyzeJobTargetRequest) -> AnalyzeJobTargetEnvelope:
    return flows["analyze_job_target"].invoke({"request": request})["result"]


@app.post("/internal/generate_question", response_model=GenerateQuestionEnvelope)
def generate_question_endpoint(
    request: GenerateQuestionRequest, response: Response
) -> GenerateQuestionEnvelope:
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


@app.post("/internal/evaluate_answer", response_model=EvaluateAnswerEnvelope)
def evaluate_answer_endpoint(
    request: EvaluateAnswerRequest, response: Response
) -> EvaluateAnswerEnvelope:
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


@app.post("/internal/generate_review", response_model=GenerateReviewEnvelope)
def generate_review_endpoint(
    request: GenerateReviewRequest, response: Response
) -> GenerateReviewEnvelope:
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


@app.post("/internal/embed_memory", response_model=EmbedMemoryResponse)
def embed_memory_endpoint(request: EmbedMemoryRequest) -> EmbedMemoryResponse:
    vectors, model_name = model_client.create_embeddings([item.text for item in request.items])
    return EmbedMemoryResponse(
        items=[
            EmbeddedMemoryVector(
                id=request.items[index].id,
                vector=vector,
                model_name=model_name,
            )
            for index, vector in enumerate(vectors)
        ]
    )


@app.post("/internal/rerank_memory", response_model=RerankMemoryResponse)
def rerank_memory_endpoint(request: RerankMemoryRequest) -> RerankMemoryResponse:
    ranked = model_client.rerank_documents(
        query=request.query,
        documents=[item.text for item in request.candidates],
        top_k=request.top_k,
    )
    return RerankMemoryResponse(
        items=[
            RerankMemoryResult(
                id=request.candidates[int(item["index"])].id,
                score=float(item["score"]),
                rank=rank + 1,
            )
            for rank, item in enumerate(ranked)
            if 0 <= int(item["index"]) < len(request.candidates)
        ]
    )


def _ndjson_stream(events):
    # 流式响应一旦开始写 body，就不能再切回 FastAPI 的常规 JSON 错误响应；
    # 这里统一把异常包装成最后一条 NDJSON 事件，前端才能稳定收尾。
    try:
        for event in events:
            yield JSONResponse(content=event).body + b"\n"
    except ModelClientError as exc:
        yield (
            JSONResponse(
                content={
                    "type": "error",
                    "code": exc.code or classify_error_code(exc),
                    "message": str(exc),
                }
            ).body
            + b"\n"
        )
    except Exception as exc:
        yield (
            JSONResponse(
                content={
                    "type": "error",
                    "code": classify_error_code(exc),
                    "message": str(exc),
                }
            ).body
            + b"\n"
        )


def apply_prompt_headers(response: Response, prompt_meta) -> None:
    # Go 侧会把这些头落到训练记录里，便于回看某次生成到底命中了哪套 prompt。
    response.headers[PROMPT_SET_HEADER] = prompt_meta.prompt_set_id
    response.headers[PROMPT_HASH_HEADER] = prompt_meta.prompt_hash
    response.headers[MODEL_NAME_HEADER] = settings.model


def classify_error_code(exc: Exception) -> str:
    # 流式链路经常只剩异常文本可用，这里做一层粗粒度映射，保证前后端还能按 code 分支。
    if isinstance(exc, ModelClientError) and exc.code:
        return exc.code

    message = str(exc).lower()
    if "required" in message and "llm" in message:
        return "llm_required"
    if "required tool context" in message:
        return "tool_context_missing"
    if "validation" in message:
        return "semantic_validation_failed"
    if "json" in message:
        return "json_parse_failed"
    if "timeout" in message or "timed out" in message:
        return "timeout"
    if "go backend" in message:
        return "backend_callback_failed"
    if "tool loop" in message:
        return "tool_loop_exhausted"
    if "single-shot" in message:
        return "single_shot_failed"
    return "unknown_error"
